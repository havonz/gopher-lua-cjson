# gopher-lua-cjson

`gopher-lua-cjson` 是一个给 [`github.com/yuin/gopher-lua`](https://github.com/yuin/gopher-lua) 使用的 `cjson` / `cjson.safe` 模块包。

它的目标是让 `gopher-lua` 运行时可以通过标准的 `require("cjson")` 和 `require("cjson.safe")` 方式加载 JSON 模块。

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

## 当前支持

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

## 当前边界

- 目标是为 `gopher-lua` 提供高兼容度的 `cjson` / `cjson.safe` 行为。
- 当前版本并未声明已经逐项跑完 OpenResty `lua-cjson` 的全部官方测试数据。
- 错误文案和部分内部实现细节不保证与 OpenResty `lua-cjson` 的 C 实现逐字一致。
