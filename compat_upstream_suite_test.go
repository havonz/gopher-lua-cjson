package luacjson_test

import (
	"testing"

	"github.com/havonz/gopher-lua-cjson/internal/upstreamtests"
)

func TestUpstreamLuaCJSONSuite(t *testing.T) {
	result, err := upstreamtests.RunUpstreamSuite()
	if err != nil {
		t.Fatalf("RunUpstreamSuite returned error: %v", err)
	}
	if !result.AllPassed {
		t.Fatalf("upstream suite reported failures:\n%s", result.Output)
	}
}
