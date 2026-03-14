# Compatibility Notes

本项目的行为语义以 [OpenResty `lua-cjson`](https://github.com/openresty/lua-cjson) 为参考目标。

## 上游快照

- 上游仓库快照保存在 `upstream/openresty-lua-cjson/`
- 当前同步元数据记录在 `upstream/openresty-lua-cjson/UPSTREAM_VERSION`
- 快照包含本项目当前追踪的文档和测试素材，例如 `README.md`、`manual.adoc`、`performance.adoc` 和 `tests/`

## 当前兼容测试覆盖

当前 `go test ./...` 会覆盖两类内容：

- 结构化上游测试：解析并执行 `upstream/openresty-lua-cjson/tests/agentzh.t` 中已接入的 case
- 关键语义测试：从上游 `tests/test.lua` 抽取并固定为 Go 子测试的兼容行为

当前已经明确覆盖的方向包括：

- 空表编码策略
- `empty_array`
- `array_mt`
- `empty_array_mt`
- `decode_array_with_array_mt`
- 部分解码错误文案
- 部分嵌套深度错误文案

## 已知边界

- 当前并没有在 `go test` 中完整解释执行上游整份 `tests/test.lua`
- 某些错误文案、运行时细节和 Lua/LuaJIT 宿主差异，仍可能与 OpenResty C 实现存在剩余偏差
- 兼容性状态以当前仓库测试覆盖到的场景为准，未覆盖部分不应默认视为完全一致

## 同步上游快照

默认同步上游 `master`：

```bash
bash scripts/sync_openresty_lua_cjson.sh
```

同步指定 ref：

```bash
bash scripts/sync_openresty_lua_cjson.sh <ref>
```

同步后重新验证：

```bash
go test ./...
```
