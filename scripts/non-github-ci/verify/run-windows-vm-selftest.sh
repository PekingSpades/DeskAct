#!/usr/bin/env bash
#
# run-windows-vm-selftest.sh — run the windowops selftest inside the
# dockur Windows VM. Assumes the VM is up and the host shared mount has
# been bound to Z: inside Windows.
#
# Strategy:
#   1. Stage the selftest binary + scenario into the shared mount.
#   2. Pull a one-shot PowerShell command into the VM that executes the
#      binary, writing the JSON report next to the binary on Z:.
#   3. Wait for the report file to appear; surface contents to stdout.
#
# Connection to the VM is via the dockur container's "control" web
# endpoint or RDP. Two flavors:
#   --via=container : docker exec into the container, drive qemu monitor
#                     via the dockur scripts. Not all dockur images
#                     expose this; fallback below.
#   --via=manual    : print the exact command the operator needs to
#                     paste over RDP. Default — most reliable.
#
# Usage:
#   bash scripts/non-github-ci/verify/run-windows-vm-selftest.sh \
#        --scenario scenarios/notepad.json \
#        --out artifacts/verify/windows-$(date +%s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

scenario=""
out_dir="${REPO_ROOT}/artifacts/verify/windows-$(date +%s)"
via="manual"
selftest_exe=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scenario)
      scenario="$2"; shift 2;;
    --out)
      out_dir="$2"; shift 2;;
    --via)
      via="$2"; shift 2;;
    --selftest-exe)
      selftest_exe="$2"; shift 2;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed 's/^#//' >&2
      exit 0;;
    *)
      echo "unknown arg: $1" >&2; exit 64;;
  esac
done

if [[ -z "${scenario}" ]]; then
  scenario="${SCRIPT_DIR}/scenarios/notepad.json"
fi
if [[ ! -f "${scenario}" ]]; then
  echo "scenario file not found: ${scenario}" >&2
  exit 2
fi

if [[ -z "${selftest_exe}" ]]; then
  selftest_exe="${REPO_ROOT}/artifacts/non-github-ci/v0.dev-windowops/go/windows-amd64/examples/windowops-selftest.exe"
fi
if [[ ! -f "${selftest_exe}" ]]; then
  echo "windows selftest binary not found: ${selftest_exe}" >&2
  echo "build it first with the windows builder, or pass --selftest-exe path" >&2
  exit 2
fi

mkdir -p "${out_dir}"
stage_dir="${REPO_ROOT}/.deskact-vms/shared-verify-windows"
mkdir -p "${stage_dir}"
cp "${selftest_exe}" "${stage_dir}/windowops-selftest.exe"
sed "s#%OUTDIR%#Z:/.deskact-vms/shared-verify-windows#g" "${scenario}" \
  > "${stage_dir}/scenario.json"

cat <<EOF
[run-windows-vm-selftest] Files staged on host:
  binary   = ${stage_dir}/windowops-selftest.exe
  scenario = ${stage_dir}/scenario.json
  out_dir  = ${out_dir}

Inside the Windows VM (RDP localhost:3400, user Docker, password admin —
or via the noVNC viewer at http://localhost:8026), run:

  Z:\\.deskact-vms\\shared-verify-windows\\windowops-selftest.exe \\
    < Z:\\.deskact-vms\\shared-verify-windows\\scenario.json \\
    > Z:\\.deskact-vms\\shared-verify-windows\\report.json

Then on the host, the JSON report and PNGs will be at:
  ${stage_dir}/report.json
  ${stage_dir}/step-*.png
EOF

if [[ "${via}" == "container" ]]; then
  if ! docker compose -f "${REPO_ROOT}/scripts/non-github-ci/docker-compose.yml" \
        exec -T windows powershell -NoProfile -Command \
        "Z:\\.deskact-vms\\shared-verify-windows\\windowops-selftest.exe < Z:\\.deskact-vms\\shared-verify-windows\\scenario.json > Z:\\.deskact-vms\\shared-verify-windows\\report.json"; then
    echo "container exec failed; fall back to --via=manual" >&2
    exit 3
  fi
  cp "${stage_dir}/report.json" "${out_dir}/" || true
  echo "selftest exit + report copied to ${out_dir}/report.json"
fi
