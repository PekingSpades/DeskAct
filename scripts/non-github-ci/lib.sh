#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

DEFAULT_ARTIFACT_ROOT="${REPO_ROOT}/artifacts/non-github-ci"
ALL_X86_PLATFORMS=("linux-amd64" "windows-amd64" "darwin-amd64")
GO_TARGETS=(
  "deskact-tester:./cmd/deskact-tester"
  "capture:./examples/capture"
  "display:./examples/display"
  "keyboard:./examples/keyboard"
  "mouse:./examples/mouse"
  "window:./examples/window"
  "apps:./examples/apps"
  "windowops:./examples/windowops"
  "windowops-selftest:./examples/windowops/selftest"
)

log() {
  printf '[deskact-non-github-ci] %s\n' "$*"
}

warn() {
  printf '[deskact-non-github-ci] WARNING: %s\n' "$*" >&2
}

die() {
  printf '[deskact-non-github-ci] ERROR: %s\n' "$*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

optional_cmd() {
  command -v "$1" >/dev/null 2>&1
}

resolve_version() {
  local requested="${1:-}"
  if [[ -n "${requested}" ]]; then
    printf '%s\n' "${requested}"
    return 0
  fi

  local sha
  sha="$(git -C "${REPO_ROOT}" rev-parse --short=8 HEAD 2>/dev/null || true)"
  if [[ -n "${sha}" ]]; then
    printf 'dev-%s\n' "${sha}"
  else
    printf 'dev\n'
  fi
}

platform_goos() {
  case "$1" in
    linux-amd64) printf 'linux\n' ;;
    windows-amd64) printf 'windows\n' ;;
    darwin-amd64) printf 'darwin\n' ;;
    *) die "unsupported platform: $1" ;;
  esac
}

platform_goarch() {
  case "$1" in
    linux-amd64 | windows-amd64 | darwin-amd64) printf 'amd64\n' ;;
    *) die "unsupported platform: $1" ;;
  esac
}

platform_ext() {
  case "$1" in
    windows-amd64) printf '.exe\n' ;;
    linux-amd64 | darwin-amd64) printf '\n' ;;
    *) die "unsupported platform: $1" ;;
  esac
}

normalize_platforms() {
  local raw="${1:-all}"
  raw="$(printf '%s' "${raw}" | tr '[:upper:]' '[:lower:]')"
  if [[ -z "${raw//[[:space:],]/}" || "${raw}" == "all" ]]; then
    printf '%s\n' "${ALL_X86_PLATFORMS[@]}"
    return 0
  fi

  local token
  local seen=" "
  tr ',[:space:]' '\n' <<<"${raw}" | while IFS= read -r token; do
    [[ -n "${token}" ]] || continue
    case "${token}" in
      linux-amd64 | windows-amd64 | darwin-amd64) ;;
      *) die "unsupported platform '${token}'. Use all, linux-amd64, windows-amd64, darwin-amd64." ;;
    esac
    if [[ "${seen}" != *" ${token} "* ]]; then
      printf '%s\n' "${token}"
      seen="${seen}${token} "
    fi
  done
}

artifact_dir_for_version() {
  local root="${1:-${DEFAULT_ARTIFACT_ROOT}}"
  local version="$2"
  printf '%s\n' "${root%/}/${version}"
}

ensure_artifact_dirs() {
  local output_dir="$1"
  mkdir -p \
    "${output_dir}/go/linux-amd64/examples" \
    "${output_dir}/go/windows-amd64/examples" \
    "${output_dir}/go/darwin-amd64/examples" \
    "${output_dir}/electron/linux-amd64" \
    "${output_dir}/electron/windows-amd64" \
    "${output_dir}/electron/darwin-amd64" \
    "${output_dir}/logs" \
    "${output_dir}/vm"
}

prepare_vm_shared_output_dir() {
  local output_dir="$1"
  chmod -R a+rwX "${output_dir}" 2>/dev/null || {
    warn "could not make ${output_dir} writable by VM guests; adjust shared-folder permissions before running VM workers"
  }
}

host_platform() {
  local os arch
  os="$(uname -s)"
  arch="$(uname -m)"

  case "${os}:${arch}" in
    Linux:x86_64 | Linux:amd64) printf 'linux-amd64\n' ;;
    Darwin:x86_64 | Darwin:amd64) printf 'darwin-amd64\n' ;;
    MINGW*:x86_64 | MSYS*:x86_64 | CYGWIN*:x86_64) printf 'windows-amd64\n' ;;
    *) die "unsupported host platform: ${os}/${arch}" ;;
  esac
}

require_native_platform() {
  local platform="$1"
  local host
  host="$(host_platform)"
  if [[ "${host}" != "${platform}" ]]; then
    die "${platform} must be built on a native ${platform} worker; current host is ${host}"
  fi
}

go_output_path() {
  local output_dir="$1"
  local platform="$2"
  local name="$3"
  local ext
  ext="$(platform_ext "${platform}")"

  if [[ "${name}" == "deskact-tester" ]]; then
    printf '%s\n' "${output_dir}/go/${platform}/deskact-tester${ext}"
  else
    printf '%s\n' "${output_dir}/go/${platform}/examples/${name}${ext}"
  fi
}

electron_archive_path() {
  local output_dir="$1"
  local platform="$2"
  printf '%s\n' "${output_dir}/electron/${platform}/electron-display-${platform}.zip"
}

sha256_file() {
  local path="$1"
  if optional_cmd sha256sum; then
    sha256sum "${path}" | awk '{print $1}'
  elif optional_cmd shasum; then
    shasum -a 256 "${path}" | awk '{print $1}'
  else
    die "missing sha256sum or shasum"
  fi
}

copy_readable_file() {
  local source="$1"
  local destination="$2"

  chmod 0644 "${source}" 2>/dev/null || true
  rm -f "${destination}"
  if [[ "$(uname -s)" == "Darwin" ]]; then
    cp -X "${source}" "${destination}"
  elif optional_cmd install; then
    install -m 0644 "${source}" "${destination}"
  else
    cp "${source}" "${destination}"
    chmod a+r "${destination}" 2>/dev/null || true
  fi
}

json_escape() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/\\n}"
  printf '"%s"' "${value}"
}

write_checksums_and_manifest() {
  local output_dir="$1"
  local version="$2"
  local checksum_file="${output_dir}/SHA256SUMS"
  local manifest_file="${output_dir}/manifest.json"
  local checksum_tmp manifest_tmp commit generated_at

  commit="$(git -C "${REPO_ROOT}" rev-parse HEAD 2>/dev/null || true)"
  generated_at="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  checksum_tmp="$(mktemp -t deskact-checksums.XXXXXX)"
  manifest_tmp="$(mktemp -t deskact-manifest.XXXXXX)"

  : >"${checksum_tmp}"
  while IFS= read -r rel_path; do
    printf '%s  %s\n' "$(sha256_file "${output_dir}/${rel_path}")" "${rel_path}" >>"${checksum_tmp}"
  done < <(
    cd "${output_dir}"
    find go electron -type f 2>/dev/null | LC_ALL=C sort
  )

  {
    printf '{\n'
    printf '  "name": "deskact-non-github-ci",\n'
    printf '  "version": %s,\n' "$(json_escape "${version}")"
    printf '  "gitCommit": %s,\n' "$(json_escape "${commit}")"
    printf '  "generatedAt": %s,\n' "$(json_escape "${generated_at}")"
    printf '  "files": [\n'

    local first=1
    while IFS= read -r rel_path; do
      local checksum
      checksum="$(sha256_file "${output_dir}/${rel_path}")"
      if [[ "${first}" == "1" ]]; then
        first=0
      else
        printf ',\n'
      fi
      printf '    {"path": %s, "sha256": %s}' \
        "$(json_escape "${rel_path}")" \
        "$(json_escape "${checksum}")"
    done < <(
      cd "${output_dir}"
      find go electron -type f 2>/dev/null | LC_ALL=C sort
    )

    printf '\n  ]\n'
    printf '}\n'
  } >"${manifest_tmp}"

  copy_readable_file "${checksum_tmp}" "${checksum_file}"
  copy_readable_file "${manifest_tmp}" "${manifest_file}"
  rm -f "${checksum_tmp}" "${manifest_tmp}"

  log "wrote ${checksum_file}"
  log "wrote ${manifest_file}"
}
