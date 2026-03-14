package luacjson

import (
	"os"
	"strings"
	"testing"
)

func TestDocumentationPointsToOpenRestyReference(t *testing.T) {
	t.Parallel()

	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "https://github.com/openresty/lua-cjson") {
		t.Fatal("README.md must point to the OpenResty lua-cjson repository")
	}

	compatDoc, err := os.ReadFile("docs/compatibility.md")
	if err != nil {
		t.Fatalf("read docs/compatibility.md: %v", err)
	}
	if !strings.Contains(string(compatDoc), "upstream/openresty-lua-cjson/UPSTREAM_VERSION") {
		t.Fatal("docs/compatibility.md must mention upstream snapshot metadata")
	}
}
