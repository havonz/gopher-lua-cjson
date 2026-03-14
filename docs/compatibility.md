# Compatibility Notes

本文档不重新定义 API 语义。`cjson` / `cjson.safe` 的行为语义以上游 [OpenResty `lua-cjson`](https://github.com/openresty/lua-cjson) README 和本仓库内的 [`upstream/openresty-lua-cjson/README.md`](../upstream/openresty-lua-cjson/README.md) 为准。

## 上游快照

- 上游仓库快照保存在 `upstream/openresty-lua-cjson/`
- 当前同步元数据记录在 `upstream/openresty-lua-cjson/UPSTREAM_VERSION`
- 快照包含本项目当前追踪的文档和测试素材，例如 `README.md`、`manual.adoc`、`performance.adoc` 和 `tests/`

## 当前兼容测试覆盖

当前 `go test ./...` 会覆盖两类内容：

- 结构化上游测试：解析并执行 `upstream/openresty-lua-cjson/tests/agentzh.t` 的全部 24 个 case
- 上游 suite 测试：在 Go 侧 runner 中执行 `upstream/openresty-lua-cjson/tests/test.lua`
- 关键语义测试：从上游 `tests/test.lua` 抽取并固定为 Go 子测试的兼容行为

当前文档只记录覆盖来源和验证方式，不单独复述这些 case 的语义解释；语义解释由上游 README 负责。

## 已知边界

- 当前并没有在 `go test` 中完整解释执行上游整份 `tests/test.lua`
- 当前 `tests/test.lua` runner 为了适配 `gopher-lua` 的 VM 限制，会跳过一个“生成全量 UTF-16 转义数据”的极限 case；除该极限 case 外，其余 suite 场景已在当前 runner 下通过
- 某些运行时细节和 Lua/LuaJIT 宿主差异，仍可能与 OpenResty C 实现存在剩余偏差
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
