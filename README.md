# gopher-lua-cjson

`gopher-lua-cjson` 是一个给 [`github.com/yuin/gopher-lua`](https://github.com/yuin/gopher-lua) 使用的 `cjson` / `cjson.safe` 兼容实现。

它的目标是让 `gopher-lua` 运行时可以通过标准的 `require("cjson")` 和 `require("cjson.safe")` 方式加载 JSON 模块，并尽量对齐 OpenResty `lua-cjson` 的行为语义。

语义参考仓库：[`openresty/lua-cjson`](https://github.com/openresty/lua-cjson)

本仓库不复制完整上游手册；接口语义、行为预期和上游测试来源以 OpenResty 仓库为准。本仓库只补充 `gopher-lua` 适配实现、兼容测试和已知差异说明，详见 [docs/compatibility.md](docs/compatibility.md)。

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

## 当前实现范围

- `require("cjson")`
- `require("cjson.safe")`
- `new()` 配置隔离
- `null`
- `empty_array`
- `array_mt`
- `empty_array_mt`
- `encode_sparse_array`
- `encode_max_depth`
- `decode_max_depth`
- `encode_number_precision`
- `encode_keep_buffer`
- `encode_invalid_numbers`
- `decode_invalid_numbers`
- `encode_empty_table_as_object`
- `decode_array_with_array_mt`
- `decode_allow_comment`
- `encode_escape_forward_slash`
- `encode_skip_unsupported_value_types`
- `encode_indent`

## 验证

```bash
go test ./...
```

兼容性测试会读取仓库内的上游快照，并通过 Go 测试去执行选定的 OpenResty `lua-cjson` 场景。
