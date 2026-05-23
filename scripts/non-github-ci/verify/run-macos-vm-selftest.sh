#!/usr/bin/env bash
#
# run-macos-vm-selftest.sh — drive the windowops selftest against the
# dockur macOS VM. The macOS path is part-automated, part-manual today
# because of TCC: Accessibility + Screen Recording cannot be granted
# programmatically without a custom configuration-profile in the OS image.
#
# Prerequisites (honest list):
#   1. dockur/macos is already up (docker compose --profile macos up -d)
#   2. The macOS install has completed (Disk Utility erase + reinstall +
#      OOBE). dockur leaves a fresh image at the Recovery menu; the
#      install must be driven by hand or via a pre-baked image.
#   3. Inside the VM, an SSH server is running and reachable from the
#      host at the address printed by the dockur container's network
#      bridge (or the script falls back to VNC-only driving).
#   4. A macOS-built windowops-selftest binary exists on the host
#      (built on a real macOS host or in CI on a darwin-amd64 runner).
#      We cannot cross-compile from Linux without osxcross.
#   5. Inside macOS: Accessibility + Screen Recording have been granted
#      to Terminal (one-time, interactive).
#
# What this script automates:
#   - Stages the selftest binary + scenario on the 9p shared mount.
#   - With --ssh user@host: pushes the run via ssh (true unattended).
#   - With --vnc: types the run command via vncdotool (best-effort; the
#     macOS guest must be at a Terminal prompt with the shared mount
#     attached).
#   - Without either: prints the exact commands to paste manually and
#     waits for the report file to appear (poll-loop).
#
# Usage:
#   bash scripts/non-github-ci/verify/run-macos-vm-selftest.sh \
#        --scenario scenarios/textedit.json \
#        --out artifacts/verify/macos-$(date +%s) \
#        [--ssh user@host:port] [--vnc localhost::5921]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

scenario=""
out_dir="${REPO_ROOT}/artifacts/verify/macos-$(date +%s)"
selftest_bin=""
ssh_target=""
vnc_target=""
wait_secs=120

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scenario)
      scenario="$2"; shift 2;;
    --out)
      out_dir="$2"; shift 2;;
    --selftest-bin)
      selftest_bin="$2"; shift 2;;
    --ssh)
      ssh_target="$2"; shift 2;;
    --vnc)
      vnc_target="$2"; shift 2;;
    --wait)
      wait_secs="$2"; shift 2;;
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
  cat <<EOF >&2
macos selftest binary not found: ${selftest_bin}

A macOS-built binary is required. Options:
  (a) Run bootstrap-macos.sh inside the VM and have it build the
      selftest (it places the binary at the expected artifact path).
  (b) Build on a separate macOS host:
        GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build \\
          -o windowops-selftest ./examples/windowops/selftest
      then pass it via --selftest-bin.
  (c) Cross-compile from Linux with osxcross (not set up in this repo).
EOF
  exit 2
fi

mkdir -p "${out_dir}"
stage_dir="${REPO_ROOT}/.deskact-vms/shared-verify-macos"
mkdir -p "${stage_dir}"
cp "${selftest_bin}" "${stage_dir}/windowops-selftest"
chmod +x "${stage_dir}/windowops-selftest"
sed "s#%OUTDIR%#/Volumes/shared/.deskact-vms/shared-verify-macos#g" "${scenario}" \
  > "${stage_dir}/scenario.json"

report_path="${stage_dir}/report.json"
rm -f "${report_path}"

cat <<EOF
[run-macos-vm-selftest] Files staged on host:
  binary   = ${stage_dir}/windowops-selftest
  scenario = ${stage_dir}/scenario.json
  report   = ${report_path} (will appear after the guest runs the binary)
EOF

# Build the guest-side command once; reuse for ssh / vnc / manual paths.
guest_cmd='cd /Volumes/shared/.deskact-vms/shared-verify-macos && \
chmod +x ./windowops-selftest && \
./windowops-selftest < scenario.json > report.json 2>&1'

if [[ -n "${ssh_target}" ]]; then
  # Path: true unattended via ssh into the macOS guest.
  echo "[run-macos-vm-selftest] driving via ssh ${ssh_target}"
  # shellcheck disable=SC2029
  ssh -o StrictHostKeyChecking=accept-new "${ssh_target}" \
      "/bin/sh -c '${guest_cmd//\'/\'\\\'\'}'" || {
    echo "ssh exec returned non-zero; check ${report_path}" >&2
  }
elif [[ -n "${vnc_target}" ]]; then
  if ! command -v vncdo >/dev/null 2>&1; then
    echo "--vnc requires vncdotool: pip install vncdotool" >&2
    exit 3
  fi
  echo "[run-macos-vm-selftest] driving via vnc ${vnc_target}"
  echo "  (guest must be at a Terminal prompt, shared mount attached)"
  # vncdotool types the command + Enter. No way to verify a shell is
  # focused — operator must arrange that beforehand.
  vncdo -s "${vnc_target}" type "${guest_cmd}" key enter || {
    echo "vncdotool failed; the report file is still polled below" >&2
  }
else
  cat <<EOF
[run-macos-vm-selftest] No --ssh or --vnc supplied. Manual path:

Inside the macOS VM:
  1. (one-time) sudo mkdir -p /Volumes/shared && sudo mount_9p shared
  2. (one-time) Grant Accessibility + Screen Recording to Terminal in
     System Settings -> Privacy & Security.
  3. Run:
       ${guest_cmd}

The report and PNGs will appear on the host at ${stage_dir}/.
EOF
fi

# Poll for the report file. The guest may take a few seconds (mount,
# permission prompt) or longer if a TCC dialog is sitting on screen.
echo "[run-macos-vm-selftest] waiting up to ${wait_secs}s for ${report_path}"
elapsed=0
while [[ ${elapsed} -lt ${wait_secs} ]]; do
  if [[ -s "${report_path}" ]]; then
    echo "[run-macos-vm-selftest] report appeared after ${elapsed}s"
    cp "${report_path}" "${out_dir}/" || true
    # Copy any step-*.png too.
    cp "${stage_dir}"/*.png "${out_dir}/" 2>/dev/null || true
    cat "${report_path}"
    # Surface the report's overall ok flag as exit code.
    if grep -q '"ok": true' "${report_path}"; then
      exit 0
    fi
    exit 1
  fi
  sleep 5
  elapsed=$(( elapsed + 5 ))
done

echo "[run-macos-vm-selftest] timed out waiting for ${report_path}" >&2
echo "  Check the macOS guest for stuck TCC prompts or shell errors." >&2
exit 4
