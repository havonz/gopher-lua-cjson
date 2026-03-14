package luacjson

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func TestLoaderProvidesCJSONModule(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())
	L.PreloadModule("cjson.safe", SafeLoader())

	if err := L.DoString(`
		local cjson = require("cjson")
		local encoded = cjson.encode({
			name = "demo",
			items = { "a", cjson.null, "b" },
		})
		assert(
			encoded == '{"name":"demo","items":["a",null,"b"]}'
			or encoded == '{"items":["a",null,"b"],"name":"demo"}'
		)
		local decoded = cjson.decode(encoded)
		assert(decoded.name == "demo")
		assert(decoded.items[2] == cjson.null)
	`); err != nil {
		t.Fatalf("expected cjson loader to register module: %v", err)
	}
}

func TestSafeLoaderProvidesSafeModule(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())
	L.PreloadModule("cjson.safe", SafeLoader())

	if err := L.DoString(`
		local cjson = require("cjson.safe")
		local value, err = cjson.decode("{]")
		assert(value == nil)
		assert(type(err) == "string")
		assert(#err > 0)
	`); err != nil {
		t.Fatalf("expected cjson.safe loader to register safe module: %v", err)
	}
}

func TestNewKeepsConfigIsolated(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())

	if err := L.DoString(`
		local cjson = require("cjson")
		local another = cjson.new()

		assert(cjson.encode({}) == "{}")
		another.encode_empty_table_as_object(false)
		assert(another.encode({}) == "[]")
		assert(cjson.encode({}) == "{}")
	`); err != nil {
		t.Fatalf("expected cjson.new() to keep config isolated: %v", err)
	}
}

func TestArrayMetatablesAndDecodeOptions(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())

	if err := L.DoString(`
		local cjson = require("cjson")

		local sparse = { [1] = "one", [2] = "two", [4] = "four", extra = "ignored" }
		setmetatable(sparse, cjson.array_mt)
		assert(cjson.encode(sparse) == '["one","two",null,"four"]')

		local empty = {}
		setmetatable(empty, cjson.empty_array_mt)
		assert(cjson.encode(empty) == "[]")

		cjson.decode_array_with_array_mt(true)
		local decoded = cjson.decode('[1,2,3]')
		assert(getmetatable(decoded) == cjson.array_mt)
	`); err != nil {
		t.Fatalf("expected array metatables and decode options to work: %v", err)
	}
}

func TestCommentsInvalidNumbersAndIndent(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())

	if err := L.DoString(`
		local cjson = require("cjson")

		cjson.decode_allow_comment(true)
		local commented = cjson.decode([[/*head*/{"ok":true}//tail]])
		assert(commented.ok == true)

		cjson.decode_invalid_numbers(true)
		local invalid = cjson.decode('[NaN,Infinity,-Infinity]')
		assert(invalid[1] ~= invalid[1])
		assert(invalid[2] > 1e308)
		assert(invalid[3] < -1e308)

		cjson.encode_invalid_numbers("null")
		assert(cjson.encode({ value = 0/0 }) == '{"value":null}')

		cjson.encode_indent("  ")
		local pretty = cjson.encode({ a = 1, b = { c = 2 } })
		assert(string.find(pretty, '\n  "a": 1,', 1, true) ~= nil)
		assert(string.find(pretty, '\n  "b": {\n    "c": 2\n  }\n', 1, true) ~= nil)
	`); err != nil {
		t.Fatalf("expected comments, invalid numbers, and indent to work: %v", err)
	}
}

func TestSparseArrayAndSkipUnsupportedTypes(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", Loader())

	if err := L.DoString(`
		local cjson = require("cjson")

		local sparse = { [1] = "one", [100] = "hundred" }
		local ok, err = pcall(function()
			return cjson.encode(sparse)
		end)
		assert(ok == false)
		assert(type(err) == "string")

		cjson.encode_sparse_array(true, 2, 10)
		local encoded_sparse = cjson.encode(sparse)
		assert(string.find(encoded_sparse, '"1":"one"', 1, true) ~= nil)
		assert(string.find(encoded_sparse, '"100":"hundred"', 1, true) ~= nil)

		cjson.encode_skip_unsupported_value_types(true)
		local mixed = cjson.encode({
			ok = "yes",
			skip = coroutine.create(function() end),
			seq = { "a", coroutine.create(function() end), "b" },
		})
		assert(string.find(mixed, '"ok":"yes"', 1, true) ~= nil)
		assert(string.find(mixed, '"seq":["a","b"]', 1, true) ~= nil)
	`); err != nil {
		t.Fatalf("expected sparse array and skip unsupported behaviors: %v", err)
	}
}
