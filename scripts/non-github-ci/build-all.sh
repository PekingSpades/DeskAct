#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

requested_version=""
platforms_raw="all"
output_dir=""
start_windows_vm=0
start_macos_vm=0
enable_dockur_macos=0
skip_electron=0

usage() {
  cat <<'EOF' >&2
Usage: build-all.sh [options]

Options:
  --version=<version>        Version label. Defaults to dev-<sha>.
  --platforms=<all|list>     Comma/space list: linux-amd64,windows-amd64,darwin-amd64.
  --output-dir=<dir>         Versioned artifact directory. Defaults to artifacts/non-github-ci/<version>.
  --skip-electron            Build only Go artifacts.
  --start-windows-vm         Start the Windows VM worker after staging the command.
  --enable-dockur-macos      Allow staging/starting the dockur macOS worker.
  --start-macos-vm           Start the macOS VM worker after staging the command. Requires --enable-dockur-macos.
EOF
  exit 64
}

for arg in "$@"; do
  case "${arg}" in
    --version=*)
      requested_version="${arg#*=}"
      ;;
    --platforms=*)
      platforms_raw="${arg#*=}"
      ;;
    --output-dir=*)
      output_dir="${arg#*=}"
      ;;
    --skip-electron)
      skip_electron=1
      ;;
    --start-windows-vm)
      start_windows_vm=1
      ;;
    --enable-dockur-macos)
      enable_dockur_macos=1
      ;;
    --start-macos-vm)
      start_macos_vm=1
      ;;
    -h | --help)
      usage
      ;;
    *)
      usage
      ;;
  esac
done

if [[ "${start_macos_vm}" == "1" && "${enable_dockur_macos}" != "1" ]]; then
  die "--start-macos-vm requires --enable-dockur-macos"
fi

version="$(resolve_version "${requested_version}")"
if [[ -z "${output_dir}" ]]; then
  output_dir="$(artifact_dir_for_version "${DEFAULT_ARTIFACT_ROOT}" "${version}")"
fi
ensure_artifact_dirs "${output_dir}"
prepare_vm_shared_output_dir "${output_dir}"

mapfile -t platforms < <(normalize_platforms "${platforms_raw}")
if [[ "${#platforms[@]}" -eq 0 ]]; then
  die "no platforms selected"
fi

log "version: ${version}"
log "output: ${output_dir}"
log "platforms: ${platforms[*]}"

host="$(host_platform)"
for platform in "${platforms[@]}"; do
  case "${platform}" in
    linux-amd64 | darwin-amd64)
      if [[ "${host}" == "${platform}" ]]; then
        "${SCRIPT_DIR}/build-go.sh" \
          --platform="${platform}" \
          --version="${version}" \
          --output-dir="${output_dir}"

        if [[ "${skip_electron}" != "1" ]]; then
          "${SCRIPT_DIR}/build-electron-display.sh" \
            --platform="${platform}" \
            --version="${version}" \
            --output-dir="${output_dir}"
        fi
      elif [[ "${platform}" == "darwin-amd64" ]]; then
        if [[ "${enable_dockur_macos}" == "1" ]]; then
          mkdir -p "${output_dir}/vm"
          cat >"${output_dir}/vm/run-macos-build.sh" <<EOF
#!/usr/bin/env bash
set -euo pipefail
REPO_ROOT="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")/../../../.." && pwd)"
source "\${REPO_ROOT}/scripts/non-github-ci/bootstrap-macos.sh"
"\${REPO_ROOT}/scripts/non-github-ci/build-go.sh" --platform=darwin-amd64 --version="${version}" --output-dir="\${REPO_ROOT}/artifacts/non-github-ci/${version}"
if [[ "${skip_electron}" != "1" ]]; then
  "\${REPO_ROOT}/scripts/non-github-ci/build-electron-display.sh" --platform=darwin-amd64 --version="${version}" --output-dir="\${REPO_ROOT}/artifacts/non-github-ci/${version}"
fi
"\${REPO_ROOT}/scripts/non-github-ci/finalize-artifacts.sh" --version="${version}" --output-dir="\${REPO_ROOT}/artifacts/non-github-ci/${version}"
EOF
          chmod +x "${output_dir}/vm/run-macos-build.sh"
          log "macOS worker command staged: ${output_dir}/vm/run-macos-build.sh"
        else
          warn "darwin-amd64 requires a native macOS worker; pass --enable-dockur-macos to stage the worker command"
        fi
      else
        warn "${platform} requires a native worker; current host is ${host}"
      fi
      ;;
    windows-amd64)
      mkdir -p "${output_dir}/vm"
      cat >"${output_dir}/vm/run-windows-build.ps1" <<EOF
\$ErrorActionPreference = 'Stop'
\$RepoRoot = (Resolve-Path (Join-Path \$PSScriptRoot '..\\..\\..\\..')).Path
\$OutputDir = Join-Path \$RepoRoot 'artifacts\\non-github-ci\\${version}'
\$Script = Join-Path \$RepoRoot 'scripts\\non-github-ci\\build-windows.ps1'
& \$Script -Version '${version}' -OutputDir \$OutputDir$(if [[ "${skip_electron}" == "1" ]]; then printf ' -SkipElectron'; fi)
EOF
      log "Windows worker command staged: ${output_dir}/vm/run-windows-build.ps1"
      ;;
  esac
done

prepare_vm_shared_output_dir "${output_dir}"

"${SCRIPT_DIR}/finalize-artifacts.sh" --version="${version}" --output-dir="${output_dir}"
prepare_vm_shared_output_dir "${output_dir}"

if [[ "${start_windows_vm}" == "1" || "${start_macos_vm}" == "1" ]]; then
  require_cmd docker
  docker compose version >/dev/null 2>&1 || die "docker compose plugin is unavailable"
  export DESKACT_VM_SHARED="${REPO_ROOT}"
  compose_file="${SCRIPT_DIR}/docker-compose.yml"
  if [[ "${start_windows_vm}" == "1" ]]; then
    log "starting Windows worker"
    docker compose -f "${compose_file}" up -d windows
  fi
  if [[ "${start_macos_vm}" == "1" ]]; then
    log "starting macOS worker"
    docker compose -f "${compose_file}" up -d macos
  fi
fi

log "build orchestration complete"
log "artifacts: ${output_dir}"
