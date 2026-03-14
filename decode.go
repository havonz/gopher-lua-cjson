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
	value, err := d.parseValue(0)
	if err != nil {
		return nil, err
	}
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos != len(d.text) {
		return nil, fmt.Errorf("unexpected trailing content at character %d", d.pos+1)
	}
	return value, nil
}

func (d *luaCJSONDecoder) parseValue(depth int) (lua.LValue, error) {
	if depth > d.config.decodeMaxDepth {
		return nil, fmt.Errorf("Found too many nested data structures (%d)", depth)
	}
	if err := d.skipIgnored(); err != nil {
		return nil, err
	}
	if d.pos >= len(d.text) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	switch d.text[d.pos] {
	case '"':
		text, err := d.parseString()
		if err != nil {
			return nil, err
		}
		return lua.LString(text), nil
	case '{':
		return d.parseObject(depth + 1)
	case '[':
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
			return nil, fmt.Errorf("expected object key string at character %d", d.pos+1)
		}
		key, err := d.parseString()
		if err != nil {
			return nil, err
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
			return nil, fmt.Errorf("expected comma or object end at character %d", d.pos+1)
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
			return nil, fmt.Errorf("expected comma or array end at character %d", d.pos+1)
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
				r, err := d.parseUnicodeEscape()
				if err != nil {
					return "", err
				}
				builder.WriteRune(r)
			default:
				return "", fmt.Errorf("invalid escape code at character %d", d.pos)
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

func (d *luaCJSONDecoder) parseUnicodeEscape() (rune, error) {
	if d.pos+4 > len(d.text) {
		return 0, fmt.Errorf("invalid unicode escape at character %d", d.pos+1)
	}
	value, err := strconv.ParseUint(d.text[d.pos:d.pos+4], 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode escape at character %d", d.pos+1)
	}
	d.pos += 4
	r := rune(value)
	if utf16.IsSurrogate(r) {
		if d.pos+6 > len(d.text) || d.text[d.pos] != '\\' || d.text[d.pos+1] != 'u' {
			return 0, fmt.Errorf("invalid unicode surrogate pair at character %d", d.pos+1)
		}
		d.pos += 2
		lowValue, err := strconv.ParseUint(d.text[d.pos:d.pos+4], 16, 16)
		if err != nil {
			return 0, fmt.Errorf("invalid unicode escape at character %d", d.pos+1)
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
		return nil, fmt.Errorf("expected value at character %d", start+1)
	}
	if d.config.decodeInvalidNumbers {
		switch strings.ToLower(token) {
		case "nan", "+nan", "-nan":
			return lua.LNumber(math.NaN()), nil
		case "inf", "+inf", "infinity", "+infinity":
			return lua.LNumber(math.Inf(1)), nil
		case "-inf", "-infinity":
			return lua.LNumber(math.Inf(-1)), nil
		}
	}
	if !d.config.decodeInvalidNumbers && !luaCJSONIsStandardJSONNumber(token) {
		return nil, fmt.Errorf("invalid number at character %d", start+1)
	}
	number, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number at character %d", start+1)
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
				for d.pos+1 < len(d.text) {
					if d.text[d.pos] == '*' && d.text[d.pos+1] == '/' {
						d.pos += 2
						break
					}
					d.pos++
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
