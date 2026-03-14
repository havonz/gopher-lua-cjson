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
	readmeText := string(readme)
	if !strings.Contains(readmeText, "https://github.com/openresty/lua-cjson") {
		t.Fatal("README.md must point to the OpenResty lua-cjson repository")
	}
	if !strings.Contains(readmeText, "upstream/openresty-lua-cjson/README.md") {
		t.Fatal("README.md must point to the vendored upstream README snapshot")
	}
	if strings.Contains(readmeText, "## 当前实现范围") {
		t.Fatal("README.md must not restate upstream API semantics locally")
	}

	compatDoc, err := os.ReadFile("docs/compatibility.md")
	if err != nil {
		t.Fatalf("read docs/compatibility.md: %v", err)
	}
	compatText := string(compatDoc)
	if !strings.Contains(compatText, "upstream/openresty-lua-cjson/UPSTREAM_VERSION") {
		t.Fatal("docs/compatibility.md must mention upstream snapshot metadata")
	}
	if !strings.Contains(compatText, "本文档不重新定义 API 语义") {
		t.Fatal("docs/compatibility.md must explicitly delegate API semantics to the upstream README")
	}
	if strings.Contains(compatText, "空表编码策略") {
		t.Fatal("docs/compatibility.md must not redefine upstream feature semantics")
	}
}
