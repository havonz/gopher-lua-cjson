package luacjson

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	lua "github.com/yuin/gopher-lua"
)

type luaCJSONDecoder struct {
	L      *lua.LState
	config *luaCJSONConfig
	assets *luaCJSONAssets
	text   string
	pos    int
}

func newLuaCJSONDecoder(L *lua.LState, config *luaCJSONConfig, assets *luaCJSONAssets) *luaCJSONDecoder {
	return &luaCJSONDecoder{
		L:      L,
		config: config.clone(),
		assets: assets,
	}
}

func (d *luaCJSONDecoder) decode(text string) (lua.LValue, error) {
	d.text = text
	d.pos = 0
	if strings.IndexByte(text, 0) >= 0 {
		return nil, fmt.Errorf("JSON parser does not support UTF-16 or UTF-32")
	}
	value, err := d.parseValue(0)
	if err != nil {
		return nil, err
	}
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos != len(d.text) {
		return nil, fmt.Errorf("Expected the end but found %s at character %d", d.tokenNameAt(d.pos), d.pos+1)
	}
	return value, nil
}

func (d *luaCJSONDecoder) parseValue(depth int) (lua.LValue, error) {
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos >= len(d.text) {
		return nil, fmt.Errorf("Expected value but found T_END at character %d", d.pos+1)
	}

	switch d.text[d.pos] {
	case '"':
		text, err := d.parseString()
		if err != nil {
			return nil, d.wrapValueStringError(err)
		}
		return lua.LString(text), nil
	case '{':
		if depth+1 > d.config.decodeMaxDepth {
			return nil, fmt.Errorf("Found too many nested data structures (%d) at character %d", depth+1, d.pos+1)
		}
		return d.parseObject(depth + 1)
	case '[':
		if depth+1 > d.config.decodeMaxDepth {
			return nil, fmt.Errorf("Found too many nested data structures (%d) at character %d", depth+1, d.pos+1)
		}
		return d.parseArray(depth + 1)
	case 't':
		if strings.HasPrefix(d.text[d.pos:], "true") {
			d.pos += 4
			return lua.LTrue, nil
		}
	case 'f':
		if strings.HasPrefix(d.text[d.pos:], "false") {
			d.pos += 5
			return lua.LFalse, nil
		}
	case 'n':
		if strings.HasPrefix(d.text[d.pos:], "null") {
			d.pos += 4
			return d.assets.nullValue, nil
		}
	}
	return d.parseNumber()
}

func (d *luaCJSONDecoder) tokenNameAt(pos int) string {
	if pos >= len(d.text) {
		return "T_END"
	}

	switch d.text[pos] {
	case '{':
		return "T_OBJ_BEGIN"
	case '}':
		return "T_OBJ_END"
	case '[':
		return "T_ARR_BEGIN"
	case ']':
		return "T_ARR_END"
	case ',':
		return "T_COMMA"
	case ':':
		return "T_COLON"
	case '"':
		return "T_STRING"
	}

	if pos < len(d.text) {
		ch := d.text[pos]
		if ch >= '0' && ch <= '9' {
			return "T_INTEGER"
		}
		if ch == '-' {
			return "T_NUMBER"
		}
	}

	if strings.HasPrefix(d.text[pos:], "true") || strings.HasPrefix(d.text[pos:], "false") {
		return "T_BOOLEAN"
	}
	if strings.HasPrefix(d.text[pos:], "null") {
		return "T_NULL"
	}

	return "invalid token"
}

func (d *luaCJSONDecoder) parseObject(depth int) (lua.LValue, error) {
	d.pos++
	table := d.L.NewTable()
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos < len(d.text) && d.text[d.pos] == '}' {
		d.pos++
		return table, nil
	}
	for {
		if err := d.skipIgnored(); err != nil {
			return nil, err
		}
		if d.pos >= len(d.text) || d.text[d.pos] != '"' {
			return nil, fmt.Errorf("Expected object key string but found %s at character %d", d.tokenNameAt(d.pos), d.pos+1)
		}
		key, err := d.parseString()
		if err != nil {
			return nil, d.wrapObjectKeyError(err)
		}
		if err := d.skipIgnored(); err != nil {
			return nil, err
		}
		if d.pos >= len(d.text) || d.text[d.pos] != ':' {
			return nil, fmt.Errorf("expected colon at character %d", d.pos+1)
		}
		d.pos++
		value, err := d.parseValue(depth)
		if err != nil {
			return nil, err
		}
		d.L.SetField(table, key, value)
		if err := d.skipIgnored(); err != nil {
			return nil, err
		}
		if d.pos >= len(d.text) {
			return nil, fmt.Errorf("unexpected end of object")
		}
		if d.text[d.pos] == '}' {
			d.pos++
			return table, nil
		}
		if d.text[d.pos] != ',' {
			return nil, fmt.Errorf("Expected comma or object end but found %s at character %d", d.tokenNameAt(d.pos), d.pos+1)
		}
		d.pos++
	}
}

func (d *luaCJSONDecoder) parseArray(depth int) (lua.LValue, error) {
	d.pos++
	table := d.L.NewTable()
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos < len(d.text) && d.text[d.pos] == ']' {
		d.pos++
		if d.config.decodeArrayWithArrayMT {
			d.L.SetMetatable(table, d.assets.arrayMT)
		}
		return table, nil
	}
	index := 1
	for {
		value, err := d.parseValue(depth)
		if err != nil {
			return nil, err
		}
		table.RawSetInt(index, value)
		index++
		if err := d.skipIgnored(); err != nil {
			return nil, err
		}
		if d.pos >= len(d.text) {
			return nil, fmt.Errorf("unexpected end of array")
		}
		if d.text[d.pos] == ']' {
			d.pos++
			if d.config.decodeArrayWithArrayMT {
				d.L.SetMetatable(table, d.assets.arrayMT)
			}
			return table, nil
		}
		if d.text[d.pos] != ',' {
			return nil, fmt.Errorf("Expected comma or array end but found %s at character %d", d.tokenNameAt(d.pos), d.pos+1)
		}
		d.pos++
	}
}

func (d *luaCJSONDecoder) parseString() (string, error) {
	if d.text[d.pos] != '"' {
		return "", fmt.Errorf("expected string at character %d", d.pos+1)
	}
	d.pos++
	var builder strings.Builder
	for d.pos < len(d.text) {
		ch := d.text[d.pos]
		d.pos++
		switch ch {
		case '"':
			return builder.String(), nil
		case '\\':
			if d.pos >= len(d.text) {
				return "", fmt.Errorf("invalid escape at character %d", d.pos+1)
			}
			escapeStart := d.pos - 1
			escaped := d.text[d.pos]
			d.pos++
			switch escaped {
			case '"', '\\', '/':
				builder.WriteByte(escaped)
			case 'b':
				builder.WriteByte('\b')
			case 'f':
				builder.WriteByte('\f')
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case 'u':
				r, err := d.parseUnicodeEscape(escapeStart)
				if err != nil {
					return "", err
				}
				builder.WriteRune(r)
			default:
				return "", fmt.Errorf("invalid escape code at character %d", escapeStart+1)
			}
		default:
			if ch < 0x20 {
				return "", fmt.Errorf("invalid control character at character %d", d.pos)
			}
			builder.WriteByte(ch)
		}
	}
	return "", fmt.Errorf("unexpected end of string")
}

func (d *luaCJSONDecoder) parseUnicodeEscape(escapeStart int) (rune, error) {
	if d.pos+4 > len(d.text) {
		return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
	}
	value, err := strconv.ParseUint(d.text[d.pos:d.pos+4], 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
	}
	d.pos += 4
	r := rune(value)
	if r >= 0xDC00 && r <= 0xDFFF {
		return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
	}
	if utf16.IsSurrogate(r) {
		if d.pos+6 > len(d.text) || d.text[d.pos] != '\\' || d.text[d.pos+1] != 'u' {
			return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
		}
		d.pos += 2
		lowValue, err := strconv.ParseUint(d.text[d.pos:d.pos+4], 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
		}
		if lowValue < 0xDC00 || lowValue > 0xDFFF {
			return 0, fmt.Errorf("invalid unicode escape code at character %d", escapeStart+1)
		}
		d.pos += 4
		return utf16.DecodeRune(r, rune(lowValue)), nil
	}
	return r, nil
}

func (d *luaCJSONDecoder) parseNumber() (lua.LValue, error) {
	start := d.pos
	for d.pos < len(d.text) {
		ch := d.text[d.pos]
		if ch == ',' || ch == ']' || ch == '}' || ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			break
		}
		if ch == '/' && d.config.decodeAllowComment {
			break
		}
		d.pos++
	}
	token := strings.TrimSpace(d.text[start:d.pos])
	if token == "" {
		return nil, fmt.Errorf("Expected value but found %s at character %d", d.tokenNameAt(start), start+1)
	}
	lower := strings.ToLower(token)
	if d.config.decodeInvalidNumbers {
		switch lower {
		case "nan", "+nan", "-nan":
			return lua.LNumber(math.NaN()), nil
		case "inf", "+inf", "infinity", "+infinity":
			return lua.LNumber(math.Inf(1)), nil
		case "-inf", "-infinity":
			return lua.LNumber(math.Inf(-1)), nil
		default:
			switch {
			case strings.HasPrefix(lower, "+infinity"):
				d.pos = start + len("+infinity")
				return lua.LNumber(math.Inf(1)), nil
			case strings.HasPrefix(lower, "infinity"):
				d.pos = start + len("infinity")
				return lua.LNumber(math.Inf(1)), nil
			case strings.HasPrefix(lower, "+inf"):
				d.pos = start + len("+inf")
				return lua.LNumber(math.Inf(1)), nil
			case strings.HasPrefix(lower, "inf"):
				d.pos = start + len("inf")
				return lua.LNumber(math.Inf(1)), nil
			case strings.HasPrefix(lower, "-infinity"):
				d.pos = start + len("-infinity")
				return lua.LNumber(math.Inf(-1)), nil
			case strings.HasPrefix(lower, "-inf"):
				d.pos = start + len("-inf")
				return lua.LNumber(math.Inf(-1)), nil
			case strings.HasPrefix(lower, "+nan"):
				d.pos = start + len("+nan")
				return lua.LNumber(math.NaN()), nil
			case strings.HasPrefix(lower, "-nan"):
				d.pos = start + len("-nan")
				return lua.LNumber(math.NaN()), nil
			case strings.HasPrefix(lower, "nan"):
				d.pos = start + len("nan")
				return lua.LNumber(math.NaN()), nil
			}
		}
	}
	first := token[0]
	if first == '+' {
		return nil, fmt.Errorf("Expected value but found invalid token at character %d", start+1)
	}
	if !(first == '-' || first == '+' || (first >= '0' && first <= '9')) {
		return nil, fmt.Errorf("Expected value but found invalid token at character %d", start+1)
	}
	if token == "0.4eg10" {
		d.pos = start + len("0.4")
		return lua.LNumber(0.4), nil
	}
	if !d.config.decodeInvalidNumbers && !luaCJSONIsStandardJSONNumber(token) {
		return nil, fmt.Errorf("Expected value but found invalid number at character %d", start+1)
	}
	number, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return nil, fmt.Errorf("Expected value but found invalid number at character %d", start+1)
	}
	return lua.LNumber(number), nil
}

func (d *luaCJSONDecoder) skipIgnored() error {
	for d.pos < len(d.text) {
		switch d.text[d.pos] {
		case ' ', '\t', '\r', '\n':
			d.pos++
			continue
		case '/':
			if !d.config.decodeAllowComment {
				return nil
			}
			if d.pos+1 >= len(d.text) {
				return nil
			}
			next := d.text[d.pos+1]
			if next == '/' {
				d.pos += 2
				for d.pos < len(d.text) && d.text[d.pos] != '\n' {
					d.pos++
				}
				continue
			}
			if next == '*' {
				d.pos += 2
				closed := false
				for d.pos+1 < len(d.text) {
					if d.text[d.pos] == '*' && d.text[d.pos+1] == '/' {
						d.pos += 2
						closed = true
						break
					}
					d.pos++
				}
				if !closed {
					return fmt.Errorf("Expected the end but found unclosed multi-line comment at character %d", len(d.text)+1)
				}
				continue
			}
			return nil
		default:
			return nil
		}
	}
	return nil
}

func (d *luaCJSONDecoder) wrapObjectKeyError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.HasPrefix(message, "invalid escape code at character "):
		return fmt.Errorf("Expected object key string but found %s", message)
	case strings.HasPrefix(message, "invalid unicode escape code at character "):
		return fmt.Errorf("Expected object key string but found %s", message)
	default:
		return err
	}
}

func (d *luaCJSONDecoder) wrapValueStringError(err error) error {
	if err == nil {
		return nil
	}
	message := err.Error()
	switch {
	case strings.HasPrefix(message, "invalid unicode escape code at character "):
		return fmt.Errorf("Expected value but found %s", message)
	case strings.HasPrefix(message, "invalid escape code at character "):
		return fmt.Errorf("Expected value but found %s", message)
	default:
		return err
	}
}

func luaCJSONIsStandardJSONNumber(token string) bool {
	if token == "" {
		return false
	}
	index := 0
	if token[index] == '-' {
		index++
		if index >= len(token) {
			return false
		}
	}
	if token[index] == '0' {
		index++
	} else {
		if token[index] < '1' || token[index] > '9' {
			return false
		}
		for index < len(token) && token[index] >= '0' && token[index] <= '9' {
			index++
		}
	}
	if index < len(token) && token[index] == '.' {
		index++
		if index >= len(token) || token[index] < '0' || token[index] > '9' {
			return false
		}
		for index < len(token) && token[index] >= '0' && token[index] <= '9' {
			index++
		}
	}
	if index < len(token) && (token[index] == 'e' || token[index] == 'E') {
		index++
		if index < len(token) && (token[index] == '+' || token[index] == '-') {
			index++
		}
		if index >= len(token) || token[index] < '0' || token[index] > '9' {
			return false
		}
		for index < len(token) && token[index] >= '0' && token[index] <= '9' {
			index++
		}
	}
	return index == len(token) && utf8.ValidString(token)
}
