# OpenResty Lua CJSON Compatibility Design

**Date:** 2026-03-14

**Goal:** In `gopher-lua-cjson/`, preserve a synced snapshot of relevant upstream `openresty/lua-cjson` tests and docs, drive compatibility verification through `go test`, and keep this project's user-facing documentation minimal while pointing semantic reference to the upstream repository.

## Context

The current repository is a small Go library that exposes `cjson` and `cjson.safe` modules for `github.com/yuin/gopher-lua`. It already includes a small set of Go tests, but it does not yet preserve upstream OpenResty test assets, document how compatibility is tracked, or provide a stable mechanism for syncing against upstream.

The desired behavior is:

- semantic compatibility should follow OpenResty `lua-cjson` as closely as practical;
- tests should be runnable via `go test` only;
- upstream tests and documentation should be preserved as a local snapshot for traceability and future updates;
- this repository's documentation should stay focused on this implementation and explicitly point readers to the upstream project for behavior semantics.

## Design Summary

The repository will separate upstream source material from local implementation:

- store a curated upstream snapshot under a dedicated `upstream/` subtree;
- add a sync script that refreshes only whitelisted upstream files and records the source commit;
- implement a Go-native compatibility test harness that executes upstream-inspired scenarios through `gopher-lua`;
- keep local docs concise and point users to `https://github.com/openresty/lua-cjson` as the primary semantic reference.

This avoids mixing upstream source-of-truth material with local implementation code, while still letting `go test` act as the single validation entry point.

## Repository Structure

### Upstream Snapshot

Create `upstream/openresty-lua-cjson/` to hold a curated snapshot of the upstream project. This directory should include:

- `tests/`
- `README.md`
- `manual.adoc`
- `performance.adoc`
- `LICENSE`
- a small metadata file such as `UPSTREAM_VERSION`

The snapshot is intended to be readable and reviewable, not executable on its own. Its purpose is traceability and future syncing.

### Sync Script

Create `scripts/sync_openresty_lua_cjson.sh` to refresh the snapshot from `https://github.com/openresty/lua-cjson`. The script should:

- accept an optional ref argument;
- clone or fetch upstream into a temporary directory;
- copy only the allowed files into `upstream/openresty-lua-cjson/`;
- rewrite `UPSTREAM_VERSION` with source repository, resolved ref, commit hash, and sync date;
- avoid modifying local implementation or documentation files.

This keeps the boundary clear: upstream assets are synced deliberately, while local code remains under this repository's control.

### Test Adaptation Layer

Create a focused Go helper package under `internal/upstreamtests/` (or equivalent internal path) with these responsibilities:

- build a `gopher-lua` runtime with `cjson` and `cjson.safe` preloaded from the local implementation;
- execute Lua snippets and capture output, return values, and errors;
- load upstream test fixture files from `upstream/openresty-lua-cjson/tests/`;
- normalize output for stable assertions, especially where JSON object key order may vary;
- parse the subset of upstream `.t` test block syntax needed for compatibility checks.

The helper package should stay narrow. It is a compatibility harness, not a general TAP runner.

### Go Test Entry Points

Add or expand Go tests so `go test ./...` becomes the canonical way to verify compatibility. Organize tests into two groups:

- tests that directly adapt structured upstream cases, especially `tests/agentzh.t`;
- tests that assert key semantics from upstream `tests/test.lua` where direct execution is not worth the complexity.

This allows the repository to keep meaningful coupling to upstream behavior without trying to interpret the entire upstream Lua test harness.

## Testing Strategy

### Structured Upstream Cases

`tests/agentzh.t` is the most suitable starting point because it already expresses isolated cases as blocks with Lua input and expected output. The harness should support the subset needed by current upstream content, especially:

- `--- lua`
- `--- out`

Each block should become a Go subtest. The harness will execute the Lua chunk against the local implementation and compare captured output to the expected output.

### Semantic Adaptation Cases

`tests/test.lua` is more procedural and contains helper functions, generated data, and broader assertions. Instead of attempting full interpretation, the local Go tests should extract and re-express the most important semantics as direct compatibility tests, such as:

- empty table encoding behavior;
- `empty_array`, `array_mt`, and `empty_array_mt`;
- array metatable behavior on decode;
- invalid number handling;
- maximum nesting behavior;
- representative parse and encode errors;
- compatibility-sensitive configuration toggles.

This preserves the important behavior contract while keeping the test harness maintainable.

### Output Normalization

The harness should normalize only where necessary. In particular:

- if an assertion depends on JSON object key order, normalize before comparing;
- if the assertion is already order-insensitive in Lua output, do not add extra normalization;
- error strings should be compared intentionally, with any unavoidable runtime-specific differences documented.

The goal is semantic fidelity, not fragile byte-for-byte imitation where the runtime cannot realistically provide it.

## Documentation Strategy

### Local README

The repository `README.md` should remain concise and describe:

- what this Go module provides for `gopher-lua`;
- that behavior semantics and API expectations follow OpenResty `lua-cjson` as the reference target;
- where to find local compatibility notes;
- how to run `go test ./...`.

It should explicitly link to `https://github.com/openresty/lua-cjson` rather than attempting to duplicate the upstream manual.

### Local Compatibility Document

Add `docs/compatibility.md` to document only local concerns:

- what upstream repository and snapshot are being tracked;
- what upstream tests are already represented in `go test`;
- what known gaps or runtime differences remain;
- how to refresh the upstream snapshot and rerun verification.

This document should describe the state of compatibility, not restate the full API manual.

### Upstream Documentation Snapshot

Keep upstream documentation files in `upstream/openresty-lua-cjson/` without local rewriting. They serve as preserved reference material and must remain clearly attributable to upstream.

## Licensing and Attribution

The upstream license permits copying, but the repository must preserve upstream copyright and license text with copied materials.

Any locally written documentation that references the upstream snapshot should make the provenance clear and should avoid presenting the upstream manual as if it were newly authored for this repository.

## Acceptance Criteria

The work is complete when all of the following are true:

1. The repository contains a curated upstream snapshot and a repeatable sync script.
2. `README.md` clearly states that OpenResty `lua-cjson` is the semantic reference.
3. `go test ./...` runs compatibility tests without requiring Perl, TAP, or external Lua test runners.
4. A meaningful set of upstream compatibility scenarios is represented in Go tests, with priority on structured upstream cases and core semantic behaviors.
5. Local docs clearly distinguish between this repository's compatibility notes and upstream reference material.
6. Any known gaps are documented rather than left implicit.

## Risks and Tradeoffs

- A full interpretation of upstream `tests/test.lua` would be expensive and brittle, so the design intentionally uses semantic adaptation for that file.
- Some error text or ordering details may differ because the runtime is `gopher-lua`, not the original C module in Lua/LuaJIT. Those differences should be documented explicitly when they cannot be aligned.
- The sync script introduces process discipline but keeps future upstream updates cheaper and easier to audit.

## Recommended Implementation Order

1. Add the upstream snapshot directory layout and sync script.
2. Sync the initial upstream snapshot and record the exact source commit.
3. Build the Go test adaptation helpers for Lua execution and fixture loading.
4. Migrate structured upstream `.t` cases into `go test`-driven subtests.
5. Add targeted Go compatibility tests for high-value semantics from `tests/test.lua`.
6. Update `README.md` and add `docs/compatibility.md`.
