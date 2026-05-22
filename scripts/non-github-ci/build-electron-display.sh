#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

requested_version=""
platform=""
output_dir=""

usage() {
  cat <<'EOF' >&2
Usage: build-electron-display.sh --platform=<linux-amd64|darwin-amd64> [options]

Options:
  --version=<version>     Version label used for the artifact directory.
  --output-dir=<dir>      Versioned artifact directory. Defaults to artifacts/non-github-ci/<version>.
EOF
  exit 64
}

for arg in "$@"; do
  case "${arg}" in
    --version=*)
      requested_version="${arg#*=}"
      ;;
    --platform=*)
      platform="${arg#*=}"
      ;;
    --output-dir=*)
      output_dir="${arg#*=}"
      ;;
    -h | --help)
      usage
      ;;
    *)
      usage
      ;;
  esac
done

[[ -n "${platform}" ]] || usage
case "${platform}" in
  linux-amd64 | darwin-amd64) ;;
  windows-amd64) die "use build-windows.ps1 for windows-amd64" ;;
  *) die "unsupported platform: ${platform}" ;;
esac

require_native_platform "${platform}"
require_cmd node
require_cmd npm

version="$(resolve_version "${requested_version}")"
if [[ -z "${output_dir}" ]]; then
  output_dir="$(artifact_dir_for_version "${DEFAULT_ARTIFACT_ROOT}" "${version}")"
fi
ensure_artifact_dirs "${output_dir}"

electron_dir="${REPO_ROOT}/examples/display/electron"
archive="$(electron_archive_path "${output_dir}" "${platform}")"
mkdir -p "$(dirname "${archive}")"
rm -f "${archive}"

case "${platform}" in
  linux-amd64)
    builder_args=(--linux --x64 --publish never)
    expected_dirs=("linux-unpacked")
    ;;
  darwin-amd64)
    builder_args=(--mac --x64 --publish never)
    expected_dirs=("mac" "mac-x64")
    ;;
esac

log "building Electron display for ${platform}"
(
  cd "${electron_dir}"
  rm -rf dist
  npm install
  npx electron-builder "${builder_args[@]}"
)

output_subdir=""
for candidate in "${expected_dirs[@]}"; do
  if [[ -d "${electron_dir}/dist/${candidate}" ]]; then
    output_subdir="${electron_dir}/dist/${candidate}"
    break
  fi
done

if [[ -z "${output_subdir}" ]]; then
  while IFS= read -r candidate; do
    output_subdir="${candidate}"
    break
  done < <(find "${electron_dir}/dist" -maxdepth 1 -type d \( -name '*-unpacked' -o -name 'mac' -o -name 'mac-*' \) | LC_ALL=C sort)
fi

[[ -n "${output_subdir}" && -d "${output_subdir}" ]] || die "Electron build output directory not found under ${electron_dir}/dist"

log "compressing ${output_subdir} -> ${archive}"
tmp_archive="$(mktemp -t deskact-electron-${platform}.XXXXXX.zip)"
rm -f "${tmp_archive}"
if optional_cmd zip; then
  (
    cd "${output_subdir}"
    zip -qry "${tmp_archive}" .
  )
elif optional_cmd 7z; then
  (
    cd "${output_subdir}"
    7z a -tzip "${tmp_archive}" . >/dev/null
  )
else
  die "missing zip or 7z"
fi
copy_readable_file "${tmp_archive}" "${archive}"
rm -f "${tmp_archive}"

log "Electron display artifact ready: ${archive}"
