package luacjson

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUpstreamSnapshotFilesExist(t *testing.T) {
	t.Parallel()

	required := []string{
		filepath.Join("upstream", "openresty-lua-cjson", "UPSTREAM_VERSION"),
		filepath.Join("upstream", "openresty-lua-cjson", "README.md"),
		filepath.Join("upstream", "openresty-lua-cjson", "manual.adoc"),
		filepath.Join("upstream", "openresty-lua-cjson", "performance.adoc"),
		filepath.Join("upstream", "openresty-lua-cjson", "LICENSE"),
		filepath.Join("upstream", "openresty-lua-cjson", "tests", "agentzh.t"),
	}

	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required upstream snapshot file %q is missing: %v", path, err)
		}
	}
}
