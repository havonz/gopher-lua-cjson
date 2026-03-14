#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
upstream_repo="https://github.com/openresty/lua-cjson"
upstream_ref="${1:-master}"
snapshot_dir="${repo_root}/upstream/openresty-lua-cjson"
tmpdir="$(mktemp -d)"

cleanup() {
	rm -rf "${tmpdir}"
}
trap cleanup EXIT

git clone --depth=1 --branch "${upstream_ref}" "${upstream_repo}" "${tmpdir}/src" >/dev/null 2>&1 || {
	git clone --depth=1 "${upstream_repo}" "${tmpdir}/src" >/dev/null 2>&1
	(
		cd "${tmpdir}/src"
		git fetch --depth=1 origin "${upstream_ref}" >/dev/null 2>&1 || true
		git checkout FETCH_HEAD >/dev/null 2>&1 || git checkout "${upstream_ref}" >/dev/null 2>&1
	)
}

resolved_commit="$(
	cd "${tmpdir}/src"
	git rev-parse HEAD
)"

mkdir -p "${snapshot_dir}"
rm -rf "${snapshot_dir}/tests"
rm -rf "${snapshot_dir}/lua"

cp -R "${tmpdir}/src/tests" "${snapshot_dir}/tests"
mkdir -p "${snapshot_dir}/lua/cjson"
cp "${tmpdir}/src/lua/cjson/util.lua" "${snapshot_dir}/lua/cjson/util.lua"
cp "${tmpdir}/src/README.md" "${snapshot_dir}/README.md"
cp "${tmpdir}/src/manual.adoc" "${snapshot_dir}/manual.adoc"
cp "${tmpdir}/src/performance.adoc" "${snapshot_dir}/performance.adoc"
cp "${tmpdir}/src/LICENSE" "${snapshot_dir}/LICENSE"

cat >"${snapshot_dir}/UPSTREAM_VERSION" <<EOF
source_repo=${upstream_repo}
requested_ref=${upstream_ref}
resolved_commit=${resolved_commit}
synced_at=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
EOF

printf 'Synced %s at %s\n' "${upstream_repo}" "${resolved_commit}"
