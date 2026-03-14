package luacjson

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

const (
	luaCJSONModuleName    = "cjson"
	luaCJSONSafeModule    = "cjson.safe"
	luaCJSONVersion       = "2.1.0.11"
	luaCJSONInvalidOff    = 0
	luaCJSONInvalidOn     = 1
	luaCJSONInvalidAsNull = 2
	luaCJSONAssetsKey     = "__gopher_lua_cjson_assets__"
)

type luaCJSONConfig struct {
	encodeSparseConvert            bool
	encodeSparseRatio              int
	encodeSparseSafe               int
	encodeMaxDepth                 int
	decodeMaxDepth                 int
	encodeInvalidNumbers           int
	decodeInvalidNumbers           bool
	encodeKeepBuffer               bool
	encodeNumberPrecision          int
	encodeEmptyTableAsObject       bool
	decodeArrayWithArrayMT         bool
	decodeAllowComment             bool
	encodeEscapeForwardSlash       bool
	encodeSkipUnsupportedValueType bool
	encodeIndent                   string
}

type luaCJSONAssets struct {
	nullValue       *lua.LUserData
	emptyArrayValue *lua.LUserData
	arrayMT         *lua.LTable
	emptyArrayMT    *lua.LTable
}

type luaCJSONModule struct {
	config *luaCJSONConfig
	assets *luaCJSONAssets
	safe   bool
}

func defaultLuaCJSONConfig() *luaCJSONConfig {
	return &luaCJSONConfig{
		encodeSparseConvert:            false,
		encodeSparseRatio:              2,
		encodeSparseSafe:               10,
		encodeMaxDepth:                 1000,
		decodeMaxDepth:                 1000,
		encodeInvalidNumbers:           luaCJSONInvalidOff,
		decodeInvalidNumbers:           false,
		encodeKeepBuffer:               true,
		encodeNumberPrecision:          14,
		encodeEmptyTableAsObject:       true,
		decodeArrayWithArrayMT:         false,
		decodeAllowComment:             false,
		encodeEscapeForwardSlash:       true,
		encodeSkipUnsupportedValueType: false,
		encodeIndent:                   "",
	}
}

func (cfg *luaCJSONConfig) clone() *luaCJSONConfig {
	if cfg == nil {
		return defaultLuaCJSONConfig()
	}
	next := *cfg
	return &next
}

// Loader returns a gopher-lua module loader for require("cjson").
func Loader() lua.LGFunction {
	return func(L *lua.LState) int {
		module := newLuaCJSONModule(defaultLuaCJSONConfig(), luaCJSONAssetsForState(L), false)
		L.Push(module.luaTable(L))
		return 1
	}
}

// SafeLoader returns a gopher-lua module loader for require("cjson.safe").
func SafeLoader() lua.LGFunction {
	return func(L *lua.LState) int {
		module := newLuaCJSONModule(defaultLuaCJSONConfig(), luaCJSONAssetsForState(L), true)
		L.Push(module.luaTable(L))
		return 1
	}
}

func luaCJSONAssetsForState(L *lua.LState) *luaCJSONAssets {
	if L == nil {
		return nil
	}
	if table, ok := L.GetField(L.Get(lua.RegistryIndex), luaCJSONAssetsKey).(*lua.LTable); ok {
		nullValue, _ := L.GetField(table, "null").(*lua.LUserData)
		emptyArrayValue, _ := L.GetField(table, "empty_array").(*lua.LUserData)
		arrayMT, _ := L.GetField(table, "array_mt").(*lua.LTable)
		emptyArrayMT, _ := L.GetField(table, "empty_array_mt").(*lua.LTable)
		if nullValue != nil && emptyArrayValue != nil && arrayMT != nil && emptyArrayMT != nil {
			return &luaCJSONAssets{
				nullValue:       nullValue,
				emptyArrayValue: emptyArrayValue,
				arrayMT:         arrayMT,
				emptyArrayMT:    emptyArrayMT,
			}
		}
	}
	assets := newLuaCJSONAssets(L)
	record := L.NewTable()
	L.SetField(record, "null", assets.nullValue)
	L.SetField(record, "empty_array", assets.emptyArrayValue)
	L.SetField(record, "array_mt", assets.arrayMT)
	L.SetField(record, "empty_array_mt", assets.emptyArrayMT)
	L.SetField(L.Get(lua.RegistryIndex), luaCJSONAssetsKey, record)
	return assets
}

func newLuaCJSONAssets(L *lua.LState) *luaCJSONAssets {
	nullValue := L.NewUserData()
	nullValue.Value = luaCJSONModuleName + ".null"

	emptyArrayValue := L.NewUserData()
	emptyArrayValue.Value = luaCJSONModuleName + ".empty_array"

	return &luaCJSONAssets{
		nullValue:       nullValue,
		emptyArrayValue: emptyArrayValue,
		arrayMT:         L.NewTable(),
		emptyArrayMT:    L.NewTable(),
	}
}

func newLuaCJSONModule(config *luaCJSONConfig, assets *luaCJSONAssets, safe bool) *luaCJSONModule {
	return &luaCJSONModule{
		config: config.clone(),
		assets: assets,
		safe:   safe,
	}
}

func (m *luaCJSONModule) luaTable(L *lua.LState) *lua.LTable {
	mod := L.NewTable()
	L.SetField(mod, "_NAME", lua.LString(luaCJSONModuleName))
	L.SetField(mod, "_VERSION", lua.LString(luaCJSONVersion))
	L.SetField(mod, "null", m.assets.nullValue)
	L.SetField(mod, "empty_array", m.assets.emptyArrayValue)
	L.SetField(mod, "array_mt", m.assets.arrayMT)
	L.SetField(mod, "empty_array_mt", m.assets.emptyArrayMT)
	L.SetField(mod, "new", L.NewFunction(m.luaNew))
	L.SetField(mod, "encode", L.NewFunction(m.wrap(m.luaEncode)))
	L.SetField(mod, "decode", L.NewFunction(m.wrap(m.luaDecode)))
	L.SetField(mod, "encode_sparse_array", L.NewFunction(m.wrap(m.luaEncodeSparseArray)))
	L.SetField(mod, "encode_max_depth", L.NewFunction(m.wrap(m.luaEncodeMaxDepth)))
	L.SetField(mod, "decode_max_depth", L.NewFunction(m.wrap(m.luaDecodeMaxDepth)))
	L.SetField(mod, "encode_number_precision", L.NewFunction(m.wrap(m.luaEncodeNumberPrecision)))
	L.SetField(mod, "encode_keep_buffer", L.NewFunction(m.wrap(m.luaEncodeKeepBuffer)))
	L.SetField(mod, "encode_invalid_numbers", L.NewFunction(m.wrap(m.luaEncodeInvalidNumbers)))
	L.SetField(mod, "decode_invalid_numbers", L.NewFunction(m.wrap(m.luaDecodeInvalidNumbers)))
	L.SetField(mod, "encode_empty_table_as_object", L.NewFunction(m.wrap(m.luaEncodeEmptyTableAsObject)))
	L.SetField(mod, "decode_array_with_array_mt", L.NewFunction(m.wrap(m.luaDecodeArrayWithArrayMT)))
	L.SetField(mod, "decode_allow_comment", L.NewFunction(m.wrap(m.luaDecodeAllowComment)))
	L.SetField(mod, "encode_escape_forward_slash", L.NewFunction(m.wrap(m.luaEncodeEscapeForwardSlash)))
	L.SetField(mod, "encode_skip_unsupported_value_types", L.NewFunction(m.wrap(m.luaEncodeSkipUnsupportedValueTypes)))
	L.SetField(mod, "encode_indent", L.NewFunction(m.wrap(m.luaEncodeIndent)))
	return mod
}

func (m *luaCJSONModule) wrap(fn func(*lua.LState) (int, error)) lua.LGFunction {
	if !m.safe {
		return func(L *lua.LState) int {
			n, err := fn(L)
			if err != nil {
				L.RaiseError("%s", err.Error())
				return 0
			}
			return n
		}
	}
	return func(L *lua.LState) int {
		n, err := fn(L)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(err.Error()))
			return 2
		}
		return n
	}
}

func (m *luaCJSONModule) luaNew(L *lua.LState) int {
	module := newLuaCJSONModule(m.config, m.assets, m.safe)
	L.Push(module.luaTable(L))
	return 1
}

func (m *luaCJSONModule) luaEncode(L *lua.LState) (int, error) {
	value := L.Get(1)
	encoded, err := newLuaCJSONEncoder(L, m.config, m.assets).encode(value)
	if err != nil {
		return 0, err
	}
	L.Push(lua.LString(encoded))
	return 1, nil
}

func (m *luaCJSONModule) luaDecode(L *lua.LState) (int, error) {
	text, err := luaCJSONStringArg(L.Get(1))
	if err != nil {
		return 0, err
	}
	decoded, err := newLuaCJSONDecoder(L, m.config, m.assets).decode(text)
	if err != nil {
		return 0, err
	}
	L.Push(decoded)
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeSparseArray(L *lua.LState) (int, error) {
	if L.GetTop() >= 1 && L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeSparseConvert = value
	}
	if L.GetTop() >= 2 && L.Get(2) != lua.LNil {
		value, err := luaCJSONIntArg(L.Get(2), 0, math.MaxInt)
		if err != nil {
			return 0, err
		}
		m.config.encodeSparseRatio = value
	}
	if L.GetTop() >= 3 && L.Get(3) != lua.LNil {
		value, err := luaCJSONIntArg(L.Get(3), 0, math.MaxInt)
		if err != nil {
			return 0, err
		}
		m.config.encodeSparseSafe = value
	}
	L.Push(lua.LBool(m.config.encodeSparseConvert))
	L.Push(lua.LNumber(m.config.encodeSparseRatio))
	L.Push(lua.LNumber(m.config.encodeSparseSafe))
	return 3, nil
}

func (m *luaCJSONModule) luaEncodeMaxDepth(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONIntArg(L.Get(1), 1, math.MaxInt)
		if err != nil {
			return 0, err
		}
		m.config.encodeMaxDepth = value
	}
	L.Push(lua.LNumber(m.config.encodeMaxDepth))
	return 1, nil
}

func (m *luaCJSONModule) luaDecodeMaxDepth(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONIntArg(L.Get(1), 1, math.MaxInt)
		if err != nil {
			return 0, err
		}
		m.config.decodeMaxDepth = value
	}
	L.Push(lua.LNumber(m.config.decodeMaxDepth))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeNumberPrecision(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONIntArg(L.Get(1), 1, 16)
		if err != nil {
			return 0, err
		}
		m.config.encodeNumberPrecision = value
	}
	L.Push(lua.LNumber(m.config.encodeNumberPrecision))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeKeepBuffer(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeKeepBuffer = value
	}
	L.Push(lua.LBool(m.config.encodeKeepBuffer))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeInvalidNumbers(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONInvalidNumbersArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeInvalidNumbers = value
	}
	L.Push(luaCJSONInvalidNumbersValue(m.config.encodeInvalidNumbers))
	return 1, nil
}

func (m *luaCJSONModule) luaDecodeInvalidNumbers(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.decodeInvalidNumbers = value
	}
	L.Push(lua.LBool(m.config.decodeInvalidNumbers))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeEmptyTableAsObject(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeEmptyTableAsObject = value
	}
	L.Push(lua.LBool(m.config.encodeEmptyTableAsObject))
	return 1, nil
}

func (m *luaCJSONModule) luaDecodeArrayWithArrayMT(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.decodeArrayWithArrayMT = value
	}
	L.Push(lua.LBool(m.config.decodeArrayWithArrayMT))
	return 1, nil
}

func (m *luaCJSONModule) luaDecodeAllowComment(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.decodeAllowComment = value
	}
	L.Push(lua.LBool(m.config.decodeAllowComment))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeEscapeForwardSlash(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeEscapeForwardSlash = value
	}
	L.Push(lua.LBool(m.config.encodeEscapeForwardSlash))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeSkipUnsupportedValueTypes(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		value, err := luaCJSONBoolLikeArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeSkipUnsupportedValueType = value
	}
	L.Push(lua.LBool(m.config.encodeSkipUnsupportedValueType))
	return 1, nil
}

func (m *luaCJSONModule) luaEncodeIndent(L *lua.LState) (int, error) {
	if L.Get(1) != lua.LNil {
		text, err := luaCJSONStringArg(L.Get(1))
		if err != nil {
			return 0, err
		}
		m.config.encodeIndent = text
	}
	L.Push(lua.LString(m.config.encodeIndent))
	return 1, nil
}

func luaCJSONStringArg(value lua.LValue) (string, error) {
	text, ok := value.(lua.LString)
	if !ok {
		return "", fmt.Errorf("expected string")
	}
	return string(text), nil
}

func luaCJSONIntArg(value lua.LValue, minValue, maxValue int) (int, error) {
	number, ok := value.(lua.LNumber)
	if !ok {
		return 0, fmt.Errorf("expected integer")
	}
	numeric := float64(number)
	if math.Trunc(numeric) != numeric {
		return 0, fmt.Errorf("expected integer")
	}
	next := int(numeric)
	if next < minValue || next > maxValue {
		return 0, fmt.Errorf("integer out of range")
	}
	return next, nil
}

func luaCJSONBoolLikeArg(value lua.LValue) (bool, error) {
	switch typed := value.(type) {
	case lua.LBool:
		return bool(typed), nil
	case lua.LString:
		switch strings.ToLower(strings.TrimSpace(string(typed))) {
		case "on":
			return true, nil
		case "off":
			return false, nil
		}
	}
	return false, fmt.Errorf("expected boolean")
}

func luaCJSONInvalidNumbersArg(value lua.LValue) (int, error) {
	switch typed := value.(type) {
	case lua.LBool:
		if bool(typed) {
			return luaCJSONInvalidOn, nil
		}
		return luaCJSONInvalidOff, nil
	case lua.LString:
		switch strings.ToLower(strings.TrimSpace(string(typed))) {
		case "on":
			return luaCJSONInvalidOn, nil
		case "off":
			return luaCJSONInvalidOff, nil
		case "null":
			return luaCJSONInvalidAsNull, nil
		}
	}
	return 0, fmt.Errorf("expected boolean or one of on/off/null")
}

func luaCJSONInvalidNumbersValue(value int) lua.LValue {
	switch value {
	case luaCJSONInvalidOn:
		return lua.LTrue
	case luaCJSONInvalidAsNull:
		return lua.LString("null")
	default:
		return lua.LFalse
	}
}

func luaCJSONNumberString(value float64, precision int) string {
	if math.IsNaN(value) {
		return "NaN"
	}
	if math.IsInf(value, 1) {
		return "Infinity"
	}
	if math.IsInf(value, -1) {
		return "-Infinity"
	}
	return strconv.FormatFloat(value, 'g', precision, 64)
}

func luaCJSONIsNull(assets *luaCJSONAssets, value lua.LValue) bool {
	if assets == nil {
		return false
	}
	ud, ok := value.(*lua.LUserData)
	return ok && ud == assets.nullValue
}

func luaCJSONIsEmptyArray(assets *luaCJSONAssets, value lua.LValue) bool {
	if assets == nil {
		return false
	}
	ud, ok := value.(*lua.LUserData)
	return ok && ud == assets.emptyArrayValue
}

func luaCJSONMetatableEquals(value lua.LValue, target *lua.LTable) bool {
	if target == nil {
		return false
	}
	switch typed := value.(type) {
	case *lua.LTable:
		table, ok := typed.Metatable.(*lua.LTable)
		return ok && table == target
	case *lua.LUserData:
		table, ok := typed.Metatable.(*lua.LTable)
		return ok && table == target
	default:
		return false
	}
}
