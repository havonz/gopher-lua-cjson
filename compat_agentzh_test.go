package luacjson_test

import (
	"testing"

	"github.com/havonz/gopher-lua-cjson/internal/upstreamtests"
)

func TestAgentzhRuntimeCapturesPrint(t *testing.T) {
	t.Parallel()

	result, err := upstreamtests.RunLua(`
		local cjson = require("cjson")
		print(cjson.encode({ greeting = "hello" }))
	`)
	if err != nil {
		t.Fatalf("RunLua returned error: %v", err)
	}

	expected := "{\"greeting\":\"hello\"}\n"
	if result.Output != expected {
		t.Fatalf("unexpected output: got %q want %q", result.Output, expected)
	}
}

func TestAgentzhParserReadsFirstCase(t *testing.T) {
	t.Parallel()

	cases, err := upstreamtests.ParseAgentzhCases()
	if err != nil {
		t.Fatalf("ParseAgentzhCases returned error: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("ParseAgentzhCases returned no cases")
	}

	first := cases[0]
	if first.Name != "TEST 1: empty tables as objects" {
		t.Fatalf("unexpected first case name: %q", first.Name)
	}
	if first.Lua == "" {
		t.Fatal("first case lua source is empty")
	}
	if first.Out != "{}\n{\"dogs\":{}}" {
		t.Fatalf("unexpected first case output: %q", first.Out)
	}
}

func TestAgentzhExecutesSelectedCases(t *testing.T) {
	t.Parallel()

	cases, err := upstreamtests.ParseAgentzhCases()
	if err != nil {
		t.Fatalf("ParseAgentzhCases returned error: %v", err)
	}

	selected := []string{
		"TEST 1: empty tables as objects",
		"TEST 2: empty tables as arrays",
		"TEST 3: empty tables as objects (explicit)",
		"TEST 4: empty_array userdata",
		"TEST 5: empty_array_mt",
		"TEST 6: empty_array_mt and empty tables as objects (explicit)",
		"TEST 7: empty_array_mt and empty tables as objects (explicit)",
		"TEST 8: empty_array_mt on non-empty tables",
		"TEST 9: array_mt on empty tables",
		"TEST 10: array_mt on non-empty tables",
		"TEST 11: array_mt on non-empty tables with holes",
		"TEST 12: decode() by default does not set array_mt on empty arrays",
		"TEST 13: decode() sets array_mt on non-empty arrays if enabled",
		"TEST 14: cfg can enable/disable setting array_mt",
		"TEST 15: array_mt on tables with hash part",
		"TEST 16: multiple calls to lua_cjson_new (1/3)",
		"TEST 17: multiple calls to lua_cjson_new (2/3)",
		"TEST 18: multiple calls to lua_cjson_new (3/3)",
	}

	for _, name := range selected {
		testCase, ok := upstreamtests.FindAgentzhCase(cases, name)
		if !ok {
			t.Fatalf("missing selected agentzh case %q", name)
		}

		t.Run(name, func(t *testing.T) {
			if err := upstreamtests.RunAgentzhCase(testCase); err != nil {
				t.Fatalf("RunAgentzhCase failed: %v", err)
			}
		})
	}
}
