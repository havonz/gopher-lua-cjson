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

func TestAgentzhExecutesAllCases(t *testing.T) {
	t.Parallel()

	cases, err := upstreamtests.ParseAgentzhCases()
	if err != nil {
		t.Fatalf("ParseAgentzhCases returned error: %v", err)
	}
	if len(cases) != 24 {
		t.Fatalf("unexpected agentzh case count: got %d want 24", len(cases))
	}

	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.Name, func(t *testing.T) {
			if err := upstreamtests.RunAgentzhCase(testCase); err != nil {
				t.Fatalf("RunAgentzhCase failed: %v", err)
			}
		})
	}
}
