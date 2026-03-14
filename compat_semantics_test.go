package luacjson_test

import (
	"testing"

	"github.com/havonz/gopher-lua-cjson/internal/upstreamtests"
)

func TestCompatibilitySemantics(t *testing.T) {
	t.Parallel()

	t.Run("decode partial json matches upstream error", func(t *testing.T) {
		result, err := upstreamtests.RunLua(`
			local cjson = require("cjson.safe")
			local value, decode_err = cjson.decode('{ "unexpected eof": ')
			print(value == nil)
			print(decode_err)
		`)
		if err != nil {
			t.Fatalf("RunLua returned error: %v", err)
		}

		expected := "true\nExpected value but found T_END at character 21\n"
		if result.Output != expected {
			t.Fatalf("unexpected output: got %q want %q", result.Output, expected)
		}
	})

	t.Run("decode extra comma matches upstream error", func(t *testing.T) {
		result, err := upstreamtests.RunLua(`
			local cjson = require("cjson.safe")
			local value, decode_err = cjson.decode('{ "extra data": true }, false')
			print(value == nil)
			print(decode_err)
		`)
		if err != nil {
			t.Fatalf("RunLua returned error: %v", err)
		}

		expected := "true\nExpected the end but found T_COMMA at character 23\n"
		if result.Output != expected {
			t.Fatalf("unexpected output: got %q want %q", result.Output, expected)
		}
	})

	t.Run("decode nested depth matches upstream error", func(t *testing.T) {
		result, err := upstreamtests.RunLua(`
			local cjson = require("cjson.safe")
			cjson.decode_max_depth(5)
			local value, decode_err = cjson.decode('[[[[[[ "nested" ]]]]]]')
			print(value == nil)
			print(decode_err)
		`)
		if err != nil {
			t.Fatalf("RunLua returned error: %v", err)
		}

		expected := "true\nFound too many nested data structures (6) at character 6\n"
		if result.Output != expected {
			t.Fatalf("unexpected output: got %q want %q", result.Output, expected)
		}
	})
}
