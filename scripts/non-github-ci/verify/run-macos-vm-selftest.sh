#!/usr/bin/env bash
#
# run-macos-vm-selftest.sh — analogue of run-windows-vm-selftest.sh
# for the dockur macOS VM. Driven over VNC (5921) or the noVNC viewer
# (http://localhost:8027). Inside the guest the shared mount surfaces
# via `sudo mount_9p shared`.
#
# Usage:
#   bash scripts/non-github-ci/verify/run-macos-vm-selftest.sh \
#        --scenario scenarios/textedit.json \
#        --out artifacts/verify/macos-$(date +%s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

scenario=""
out_dir="${REPO_ROOT}/artifacts/verify/macos-$(date +%s)"
selftest_bin=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scenario)
      scenario="$2"; shift 2;;
    --out)
      out_dir="$2"; shift 2;;
    --selftest-bin)
      selftest_bin="$2"; shift 2;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed 's/^#//' >&2
      exit 0;;
    *)
      echo "unknown arg: $1" >&2; exit 64;;
  esac
done

if [[ -z "${scenario}" ]]; then
  scenario="${SCRIPT_DIR}/scenarios/textedit.json"
fi
if [[ ! -f "${scenario}" ]]; then
  echo "scenario file not found: ${scenario}" >&2
  exit 2
fi

if [[ -z "${selftest_bin}" ]]; then
  selftest_bin="${REPO_ROOT}/artifacts/non-github-ci/v0.dev-windowops/go/darwin-amd64/examples/windowops-selftest"
fi
if [[ ! -f "${selftest_bin}" ]]; then
  echo "macos selftest binary not found: ${selftest_bin}" >&2
  echo "build it first on the macos worker, or pass --selftest-bin" >&2
  exit 2
fi

mkdir -p "${out_dir}"
stage_dir="${REPO_ROOT}/.deskact-vms/shared-verify-macos"
mkdir -p "${stage_dir}"
cp "${selftest_bin}" "${stage_dir}/windowops-selftest"
chmod +x "${stage_dir}/windowops-selftest"
sed "s#%OUTDIR%#/Volumes/shared/.deskact-vms/shared-verify-macos#g" "${scenario}" \
  > "${stage_dir}/scenario.json"

cat <<EOF
[run-macos-vm-selftest] Files staged on host:
  binary   = ${stage_dir}/windowops-selftest
  scenario = ${stage_dir}/scenario.json

Inside the macOS VM (VNC localhost:5921, noVNC http://localhost:8027):
1. If the shared mount is not yet attached:
     sudo mount_9p shared
2. Grant Accessibility + Screen Recording permission to Terminal in
   System Settings -> Privacy & Security.
3. Run:
     /Volumes/shared/.deskact-vms/shared-verify-macos/windowops-selftest \\
       < /Volumes/shared/.deskact-vms/shared-verify-macos/scenario.json \\
       > /Volumes/shared/.deskact-vms/shared-verify-macos/report.json

Report + PNGs come back on the host at:
  ${stage_dir}/report.json
  ${stage_dir}/step-*.png

Note: first run usually fails with ErrPermissionDenied because the OS
prompts for Accessibility access. Grant and re-run.
EOF
