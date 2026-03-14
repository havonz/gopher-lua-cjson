# gopher-lua-cjson

`gopher-lua-cjson` 是一个给 [`github.com/yuin/gopher-lua`](https://github.com/yuin/gopher-lua) 使用的 `cjson` / `cjson.safe` 兼容实现。

它的目标是让 `gopher-lua` 运行时可以通过标准的 `require("cjson")` 和 `require("cjson.safe")` 方式加载 JSON 模块，并尽量对齐 OpenResty `lua-cjson`。

行为语义以上游 OpenResty README 为准：

- 上游仓库：[`openresty/lua-cjson`](https://github.com/openresty/lua-cjson)
- 仓库内快照：[`upstream/openresty-lua-cjson/README.md`](upstream/openresty-lua-cjson/README.md)

本仓库不重新定义 `cjson` API 语义；这里只提供 `gopher-lua` 适配实现、兼容性测试、同步脚本和差异记录。与覆盖范围、验证方式和已知边界相关的说明见 [docs/compatibility.md](docs/compatibility.md)。

## 安装

```bash
go get github.com/havonz/gopher-lua-cjson
```

## 用法

```go
package main

import (
	luacjson "github.com/havonz/gopher-lua-cjson"
	lua "github.com/yuin/gopher-lua"
)

func main() {
	L := lua.NewState()
	defer L.Close()

	L.PreloadModule("cjson", luacjson.Loader())
	L.PreloadModule("cjson.safe", luacjson.SafeLoader())

	if err := L.DoString(`
		local cjson = require("cjson")
		local value = cjson.decode('{"ok":true}')
		print(cjson.encode(value))
	`); err != nil {
		panic(err)
	}
}
```

## 验证

```bash
go test ./...
```

兼容性测试会读取仓库内的上游快照，并通过 Go 测试执行已接入的 OpenResty `lua-cjson` 场景。
