#!/usr/bin/env bash

set -euo pipefail

log() {
  printf '[deskact-non-github-ci] %s\n' "$*"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf '[deskact-non-github-ci] ERROR: missing command after bootstrap: %s\n' "$1" >&2
    exit 1
  }
}

install_go_user() {
  local version="${DESKACT_MACOS_GO_VERSION:-1.25.0}"
  local tools_dir="${DESKACT_MACOS_TOOLS_DIR:-${HOME}/.deskact-build-tools}"
  local cache_dir="${tools_dir}/cache"
  local install_dir="${tools_dir}/go-${version}"
  local active_dir="${tools_dir}/go"
  local archive="${cache_dir}/go${version}.darwin-amd64.tar.gz"

  if [[ ! -x "${install_dir}/bin/go" ]]; then
    log "installing Go ${version} under ${install_dir}"
    mkdir -p "${cache_dir}"
    curl -fL --retry 3 "https://go.dev/dl/go${version}.darwin-amd64.tar.gz" -o "${archive}"
    rm -rf "${install_dir}" "${tools_dir}/go.tmp"
    mkdir -p "${tools_dir}/go.tmp"
    tar -xzf "${archive}" -C "${tools_dir}/go.tmp"
    mv "${tools_dir}/go.tmp/go" "${install_dir}"
    rm -rf "${tools_dir}/go.tmp"
  fi

  ln -sfn "${install_dir}" "${active_dir}"
  export PATH="${active_dir}/bin:${PATH}"
}

install_node_user() {
  local version="${DESKACT_MACOS_NODE_VERSION:-20.18.1}"
  local tools_dir="${DESKACT_MACOS_TOOLS_DIR:-${HOME}/.deskact-build-tools}"
  local cache_dir="${tools_dir}/cache"
  local install_dir="${tools_dir}/node-v${version}-darwin-x64"
  local active_dir="${tools_dir}/node"
  local archive="${cache_dir}/node-v${version}-darwin-x64.tar.gz"

  if [[ ! -x "${install_dir}/bin/node" ]]; then
    log "installing Node.js ${version} under ${install_dir}"
    mkdir -p "${cache_dir}"
    curl -fL --retry 3 "https://nodejs.org/dist/v${version}/node-v${version}-darwin-x64.tar.gz" -o "${archive}"
    rm -rf "${install_dir}"
    tar -xzf "${archive}" -C "${tools_dir}"
  fi

  ln -sfn "${install_dir}" "${active_dir}"
  export PATH="${active_dir}/bin:${PATH}"
}

if [[ "$(uname -s)" != "Darwin" ]]; then
  printf '[deskact-non-github-ci] ERROR: bootstrap-macos.sh must run on macOS\n' >&2
  exit 1
fi

if ! xcode-select -p >/dev/null 2>&1; then
  log "Xcode Command Line Tools are not installed; starting installer"
  xcode-select --install || true
  printf '[deskact-non-github-ci] Re-run this script after Command Line Tools finish installing.\n' >&2
  exit 1
fi

tools_dir="${DESKACT_MACOS_TOOLS_DIR:-${HOME}/.deskact-build-tools}"
if [[ -d "${tools_dir}/go/bin" ]]; then
  export PATH="${tools_dir}/go/bin:${PATH}"
fi
if [[ -d "${tools_dir}/node/bin" ]]; then
  export PATH="${tools_dir}/node/bin:${PATH}"
fi

if command -v brew >/dev/null 2>&1; then
  log "installing macOS build tools with Homebrew"
  brew install go node || true
else
  log "Homebrew is unavailable; installing macOS build tools without sudo"
  command -v go >/dev/null 2>&1 || install_go_user
  command -v node >/dev/null 2>&1 || install_node_user
fi

require_cmd git
require_cmd go
require_cmd node
require_cmd npm
require_cmd zip

log "macOS builder bootstrap complete"
log "If using the Compose macOS worker, mount the shared folder before running the staged build script."
