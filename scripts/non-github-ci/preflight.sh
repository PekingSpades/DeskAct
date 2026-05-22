#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

include_vms=0
check_linux_deps=1

usage() {
  cat <<'EOF' >&2
Usage: preflight.sh [--include-vms] [--no-linux-deps]

Checks the host tools needed by the non-GitHub DeskAct build scripts.
EOF
  exit 64
}

for arg in "$@"; do
  case "${arg}" in
    --include-vms)
      include_vms=1
      ;;
    --no-linux-deps)
      check_linux_deps=0
      ;;
    -h | --help)
      usage
      ;;
    *)
      usage
      ;;
  esac
done

missing=0

check_cmd() {
  local command="$1"
  if optional_cmd "${command}"; then
    log "found ${command}: $(command -v "${command}")"
  else
    warn "missing command: ${command}"
    missing=1
  fi
}

check_pkg_config() {
  local module="$1"
  if pkg-config --exists "${module}" 2>/dev/null; then
    log "found pkg-config module ${module}"
  else
    warn "missing pkg-config module: ${module}"
    missing=1
  fi
}

log "repo root: ${REPO_ROOT}"

check_cmd git
check_cmd go
check_cmd node
check_cmd npm
check_cmd pkg-config

if optional_cmd zip; then
  log "found zip: $(command -v zip)"
elif optional_cmd 7z; then
  log "found 7z: $(command -v 7z)"
else
  warn "missing zip or 7z"
  missing=1
fi

if [[ "${check_linux_deps}" == "1" && "$(uname -s)" == "Linux" ]]; then
  check_pkg_config x11
  check_pkg_config xtst
  check_pkg_config xinerama
  check_pkg_config xrandr
  check_pkg_config libpng
fi

if [[ "${include_vms}" == "1" ]]; then
  if [[ -e /dev/kvm ]]; then
    log "found /dev/kvm"
  else
    warn "missing /dev/kvm"
    missing=1
  fi

  if [[ -e /dev/net/tun ]]; then
    log "found /dev/net/tun"
  else
    warn "missing /dev/net/tun"
    missing=1
  fi

  check_cmd docker
  if docker compose version >/dev/null 2>&1; then
    log "found docker compose"
  else
    warn "docker compose plugin is unavailable"
    missing=1
  fi
fi

if [[ "${missing}" != "0" ]]; then
  die "preflight checks failed"
fi

log "preflight checks passed"
