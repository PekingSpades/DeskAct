#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

requested_version=""
platform=""
output_dir=""

usage() {
  cat <<'EOF' >&2
Usage: build-go.sh --platform=<linux-amd64|darwin-amd64> [options]

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
require_cmd go

version="$(resolve_version "${requested_version}")"
if [[ -z "${output_dir}" ]]; then
  output_dir="$(artifact_dir_for_version "${DEFAULT_ARTIFACT_ROOT}" "${version}")"
fi
ensure_artifact_dirs "${output_dir}"

export GOOS
export GOARCH
export CGO_ENABLED=1
export CGO_LDFLAGS_ALLOW="-weak_framework|ScreenCaptureKit"
GOOS="$(platform_goos "${platform}")"
GOARCH="$(platform_goarch "${platform}")"

if [[ "${platform}" == "darwin-amd64" ]]; then
  export CGO_CFLAGS="-mmacosx-version-min=11.0 ${CGO_CFLAGS:-}"
  export CGO_LDFLAGS="-mmacosx-version-min=11.0 ${CGO_LDFLAGS:-}"
fi

log "building Go artifacts for ${platform}"
for target in "${GO_TARGETS[@]}"; do
  name="${target%%:*}"
  package="${target#*:}"
  output="$(go_output_path "${output_dir}" "${platform}" "${name}")"
  mkdir -p "$(dirname "${output}")"
  rm -f "${output}"
  log "go build ${package} -> ${output}"
  go build -a -v -o "${output}" "${package}"
  chmod a+rx "${output}" 2>/dev/null || warn "could not make ${output} executable by VM guests"
done

log "Go artifacts ready for ${platform}"
