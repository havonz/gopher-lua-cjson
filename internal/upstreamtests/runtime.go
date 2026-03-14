package upstreamtests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	luacjson "github.com/havonz/gopher-lua-cjson"
	lua "github.com/yuin/gopher-lua"
)

type RunResult struct {
	Output string
}

func RunLua(source string) (RunResult, error) {
	L := lua.NewState()
	defer L.Close()

	var output strings.Builder

	L.PreloadModule("cjson", luacjson.Loader())
	L.PreloadModule("cjson.safe", luacjson.SafeLoader())
	L.PreloadModule("tests.sort_json", loadLuaModuleFromFile(filepath.Join("upstream", "openresty-lua-cjson", "tests", "sort_json.lua")))
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		values := make([]string, 0, L.GetTop())
		for i := 1; i <= L.GetTop(); i++ {
			values = append(values, L.Get(i).String())
		}
		output.WriteString(strings.Join(values, "\t"))
		output.WriteByte('\n')
		return 0
	}))

	if err := L.DoString(source); err != nil {
		return RunResult{}, err
	}

	return RunResult{Output: output.String()}, nil
}

func loadLuaModuleFromFile(path string) lua.LGFunction {
	return func(L *lua.LState) int {
		source, err := os.ReadFile(path)
		if err != nil {
			L.RaiseError("failed to read module %s: %v", path, err)
			return 0
		}

		chunk, err := L.LoadString(string(source))
		if err != nil {
			L.RaiseError("failed to load module %s: %v", path, err)
			return 0
		}

		L.Push(chunk)
		L.Call(0, 1)
		if L.GetTop() == 0 {
			L.Push(lua.LNil)
		}
		return 1
	}
}

func FixturePath(parts ...string) string {
	base := []string{"upstream", "openresty-lua-cjson"}
	return filepath.Join(append(base, parts...)...)
}

func ReadFixture(parts ...string) ([]byte, error) {
	path := FixturePath(parts...)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixture %s: %w", path, err)
	}
	return data, nil
}
