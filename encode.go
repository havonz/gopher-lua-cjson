package luacjson

import (
	"fmt"
	"math"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type luaCJSONEncoder struct {
	L      *lua.LState
	config *luaCJSONConfig
	assets *luaCJSONAssets
}

type luaCJSONTableKey struct {
	kind    string
	text    string
	numeric float64
	value   lua.LValue
}

func newLuaCJSONEncoder(L *lua.LState, config *luaCJSONConfig, assets *luaCJSONAssets) *luaCJSONEncoder {
	return &luaCJSONEncoder{
		L:      L,
		config: config.clone(),
		assets: assets,
	}
}

func (e *luaCJSONEncoder) encode(value lua.LValue) (string, bool, error) {
	var builder strings.Builder
	if err := e.appendValue(&builder, value, 0, true); err != nil {
		if err == errLuaCJSONSkipValue {
			return "", true, nil
		}
		return "", false, err
	}
	return builder.String(), false, nil
}

func (e *luaCJSONEncoder) appendValue(builder *strings.Builder, value lua.LValue, depth int, skipUnsupported bool) error {
	if depth > e.config.encodeMaxDepth {
		return fmt.Errorf("Cannot serialise, excessive nesting (%d)", depth)
	}

	switch typed := value.(type) {
	case *lua.LNilType:
		builder.WriteString("null")
		return nil
	case lua.LBool:
		if bool(typed) {
			builder.WriteString("true")
		} else {
			builder.WriteString("false")
		}
		return nil
	case lua.LString:
		e.appendString(builder, string(typed))
		return nil
	case lua.LNumber:
		return e.appendNumber(builder, float64(typed))
	case *lua.LTable:
		return e.appendTable(builder, typed, depth+1)
	case *lua.LUserData:
		if luaCJSONIsNull(e.assets, value) {
			builder.WriteString("null")
			return nil
		}
		if luaCJSONIsEmptyArray(e.assets, value) {
			builder.WriteString("[]")
			return nil
		}
	}

	if skipUnsupported && e.config.encodeSkipUnsupportedValueType {
		return errLuaCJSONSkipValue
	}
	return fmt.Errorf("Cannot serialise %s: type not supported", value.Type().String())
}

func (e *luaCJSONEncoder) appendNumber(builder *strings.Builder, value float64) error {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		switch e.config.encodeInvalidNumbers {
		case luaCJSONInvalidOff:
			return fmt.Errorf("Cannot serialise number: must not be NaN or Infinity")
		case luaCJSONInvalidAsNull:
			builder.WriteString("null")
			return nil
		default:
			builder.WriteString(luaCJSONNumberString(value, e.config.encodeNumberPrecision))
			return nil
		}
	}
	builder.WriteString(luaCJSONNumberString(value, e.config.encodeNumberPrecision))
	return nil
}

func (e *luaCJSONEncoder) appendTable(builder *strings.Builder, table *lua.LTable, depth int) error {
	if luaCJSONMetatableEquals(table, e.assets.arrayMT) {
		return e.appendArray(builder, table, depth, e.arrayMTLength(table), true)
	}

	if length, asArray, raw, err := e.detectArray(table); err != nil {
		return err
	} else if asArray {
		return e.appendArray(builder, table, depth, length, raw)
	}

	if luaCJSONMetatableEquals(table, e.assets.emptyArrayMT) && table.Len() == 0 && e.tableEntryCount(table) == 0 {
		return e.appendArray(builder, table, depth, 0, true)
	}
	return e.appendObject(builder, table, depth)
}

func (e *luaCJSONEncoder) arrayMTLength(table *lua.LTable) int {
	if table == nil {
		return 0
	}

	keys := make(map[int]struct{})
	maxIndex := 0
	table.ForEach(func(key lua.LValue, _ lua.LValue) {
		number, ok := key.(lua.LNumber)
		if !ok {
			return
		}

		numeric := float64(number)
		if numeric < 1 || math.Trunc(numeric) != numeric {
			return
		}

		index := int(numeric)
		keys[index] = struct{}{}
		if index > maxIndex {
			maxIndex = index
		}
	})

	if maxIndex == 0 {
		return 0
	}

	bestSize := 0
	count := 0
	sortedKeys := make([]int, 0, len(keys))
	for index := range keys {
		sortedKeys = append(sortedKeys, index)
	}
	sort.Ints(sortedKeys)

	for size, i := 1, 0; ; {
		for i < len(sortedKeys) && sortedKeys[i] <= size {
			count++
			i++
		}
		if count > size/2 {
			bestSize = size
		}
		if size >= maxIndex || size > math.MaxInt/2 {
			break
		}
		size *= 2
	}

	for index := bestSize; index >= 1; index-- {
		if _, ok := keys[index]; ok {
			return index
		}
	}

	return 0
}

func (e *luaCJSONEncoder) detectArray(table *lua.LTable) (int, bool, bool, error) {
	if table == nil {
		return 0, false, false, nil
	}

	if e.tableEntryCount(table) == 0 {
		length := e.L.ObjLen(table)
		if length > 0 {
			return length, true, false, nil
		}
	}
	if table.Len() == 0 && e.tableEntryCount(table) == 0 {
		return 0, !e.config.encodeEmptyTableAsObject, true, nil
	}

	maxIndex := 0
	items := 0
	isArray := true
	table.ForEach(func(key lua.LValue, _ lua.LValue) {
		if !isArray {
			return
		}
		number, ok := key.(lua.LNumber)
		if !ok {
			isArray = false
			return
		}
		numeric := float64(number)
		if numeric < 1 || math.Trunc(numeric) != numeric {
			isArray = false
			return
		}
		index := int(numeric)
		if index > maxIndex {
			maxIndex = index
		}
		items++
	})
	if !isArray {
		return 0, false, false, nil
	}
	if e.config.encodeSparseRatio > 0 && maxIndex > items*e.config.encodeSparseRatio && maxIndex > e.config.encodeSparseSafe {
		if !e.config.encodeSparseConvert {
			return 0, false, false, fmt.Errorf("Cannot serialise table: excessively sparse array")
		}
		return 0, false, false, nil
	}
	return maxIndex, true, true, nil
}

func (e *luaCJSONEncoder) appendArray(builder *strings.Builder, table *lua.LTable, depth, length int, raw bool) error {
	builder.WriteByte('[')
	written := 0
	for index := 1; index <= length; index++ {
		var value lua.LValue
		if raw {
			value = table.RawGetInt(index)
		} else {
			value = e.L.GetTable(table, lua.LNumber(index))
		}

		var child strings.Builder
		err := e.appendValue(&child, value, depth, true)
		if err != nil {
			if err == errLuaCJSONSkipValue {
				continue
			}
			return err
		}
		if written > 0 {
			builder.WriteByte(',')
		}
		if e.config.encodeIndent != "" {
			e.appendIndent(builder, depth)
		}
		builder.WriteString(child.String())
		written++
	}
	if written > 0 && e.config.encodeIndent != "" {
		e.appendIndent(builder, depth-1)
	}
	builder.WriteByte(']')
	return nil
}

func (e *luaCJSONEncoder) appendObject(builder *strings.Builder, table *lua.LTable, depth int) error {
	keys := make([]luaCJSONTableKey, 0, e.tableEntryCount(table))
	table.ForEach(func(key lua.LValue, _ lua.LValue) {
		switch typed := key.(type) {
		case lua.LString:
			keys = append(keys, luaCJSONTableKey{kind: "string", text: string(typed), value: key})
		case lua.LNumber:
			keys = append(keys, luaCJSONTableKey{
				kind:    "number",
				numeric: float64(typed),
				text:    luaCJSONNumberString(float64(typed), e.config.encodeNumberPrecision),
				value:   key,
			})
		default:
			keys = append(keys, luaCJSONTableKey{kind: "invalid", value: key})
		}
	})
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].kind == keys[j].kind {
			if keys[i].kind == "number" {
				return keys[i].numeric < keys[j].numeric
			}
			return keys[i].text < keys[j].text
		}
		return keys[i].kind < keys[j].kind
	})

	builder.WriteByte('{')
	written := 0
	for _, key := range keys {
		if key.kind == "invalid" {
			return fmt.Errorf("Cannot serialise %s: table key must be a number or string", key.value.Type().String())
		}

		var child strings.Builder
		err := e.appendValue(&child, table.RawGet(key.value), depth, true)
		if err != nil {
			if err == errLuaCJSONSkipValue {
				continue
			}
			return err
		}
		if written > 0 {
			builder.WriteByte(',')
		}
		if e.config.encodeIndent != "" {
			e.appendIndent(builder, depth)
		}
		e.appendString(builder, key.text)
		if e.config.encodeIndent != "" {
			builder.WriteString(": ")
		} else {
			builder.WriteByte(':')
		}
		builder.WriteString(child.String())
		written++
	}
	if written > 0 && e.config.encodeIndent != "" {
		e.appendIndent(builder, depth-1)
	}
	builder.WriteByte('}')
	return nil
}

func (e *luaCJSONEncoder) appendString(builder *strings.Builder, text string) {
	builder.WriteByte('"')
	for i := 0; i < len(text); i++ {
		b := text[i]
		switch b {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		case '/':
			if e.config.encodeEscapeForwardSlash {
				builder.WriteString(`\/`)
			} else {
				builder.WriteByte(b)
			}
		default:
			if b < 0x20 || b == 0x7f {
				builder.WriteString(`\u`)
				builder.WriteString(fmt.Sprintf("%04x", b))
			} else {
				builder.WriteByte(b)
			}
		}
	}
	builder.WriteByte('"')
}

func (e *luaCJSONEncoder) appendIndent(builder *strings.Builder, depth int) {
	builder.WriteByte('\n')
	if depth <= 0 {
		return
	}
	builder.WriteString(strings.Repeat(e.config.encodeIndent, depth))
}

func (e *luaCJSONEncoder) tableEntryCount(table *lua.LTable) int {
	count := 0
	table.ForEach(func(_ lua.LValue, _ lua.LValue) {
		count++
	})
	return count
}

var errLuaCJSONSkipValue = fmt.Errorf("skip unsupported value")
