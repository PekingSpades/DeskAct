#!/usr/bin/env bash
#
# run-linux-selftest.sh — run the windowops selftest against the Linux X
# session this host is running, with a real Notepad-like target window.
#
# By default the script spawns its OWN xterm titled "deskact-target" so
# the scenario windowMatch lines up. The xterm is started with
# allowSendEvents:true so the XSendEvent-based per-window keyboard path
# is actually delivered (xterm silently drops synthetic input
# otherwise). The script tears the target down on exit. Pass
# --no-spawn-target to skip spawning if you want to wire the selftest
# to an externally-managed window.
#
# Usage:
#   bash scripts/non-github-ci/verify/run-linux-selftest.sh \
#        --scenario scenarios/xterm-basic.json \
#        --out artifacts/verify/linux-$(date +%s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

scenario=""
out_dir="${REPO_ROOT}/artifacts/verify/linux-$(date +%s)"
spawn_target=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scenario)
      scenario="$2"; shift 2;;
    --out)
      out_dir="$2"; shift 2;;
    --no-spawn-target)
      spawn_target=0; shift;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed 's/^#//' >&2
      exit 0;;
    *)
      echo "unknown arg: $1" >&2; exit 64;;
  esac
done

if [[ -z "${DISPLAY:-}" ]]; then
  echo "DISPLAY is unset — start an X session first" >&2
  exit 2
fi

if [[ -z "${scenario}" ]]; then
  scenario="${SCRIPT_DIR}/scenarios/xterm-basic.json"
fi
if [[ ! -f "${scenario}" ]]; then
  echo "scenario file not found: ${scenario}" >&2
  exit 2
fi

mkdir -p "${out_dir}"

selftest_bin="${REPO_ROOT}/bin/windowops-selftest"
if [[ ! -x "${selftest_bin}" ]]; then
  echo "building selftest binary -> ${selftest_bin}"
  ( cd "${REPO_ROOT}" && CGO_ENABLED=1 go build -o "${selftest_bin}" ./examples/windowops/selftest )
fi

# Start a target window the scenario can match. We kill it before
# starting in case a previous run left one behind (the scenario uses
# titleContains, and a stale "deskact-target" would otherwise be
# targeted instead of a fresh one).
target_pid=""
cleanup_target() {
  if [[ -n "${target_pid}" ]] && kill -0 "${target_pid}" 2>/dev/null; then
    kill "${target_pid}" 2>/dev/null || true
    wait "${target_pid}" 2>/dev/null || true
  fi
  # Sweep any leftover deskact-target xterms (e.g. earlier abandoned
  # runs) so the next invocation has a clean state.
  pkill -f 'xterm.*deskact-target' 2>/dev/null || true
}
trap cleanup_target EXIT

if [[ "${spawn_target}" == "1" ]]; then
  if ! command -v xterm >/dev/null 2>&1; then
    echo "xterm not found and --no-spawn-target not passed; install xterm or provide your own target window" >&2
    exit 3
  fi
  pkill -f 'xterm.*deskact-target' 2>/dev/null || true
  sleep 0.5
  # allowSendEvents lets the XSendEvent-based per-window keyboard path
  # actually land; the OSC title sequence pins the window title so the
  # scenario's windowMatch sees "deskact-target" instead of whatever
  # PS1 the inner shell decides on.
  xterm -xrm 'XTerm*allowSendEvents:true' \
        -name deskact-target -title deskact-target \
        -geometry 60x20+50+50 \
        -e "bash -c 'echo -ne \"\\033]0;deskact-target\\007\"; cat'" &
  target_pid=$!
  # Wait briefly for the window to register with the X server before
  # the selftest's first windowMatch query.
  sleep 1
  echo "spawned target xterm pid=${target_pid}"
fi

# Substitute %OUTDIR% in the scenario so the report references the right
# directory. The scenario JSON's outDir field can also be edited directly.
tmp_scenario="$(mktemp)"
sed "s#%OUTDIR%#${out_dir}#g" "${scenario}" > "${tmp_scenario}"

report_path="${out_dir}/report.json"
echo "running selftest with scenario=${scenario}"
echo "report -> ${report_path}"
"${selftest_bin}" < "${tmp_scenario}" > "${report_path}" || rc=$?
rc="${rc:-0}"
rm -f "${tmp_scenario}"

echo "selftest exit code: ${rc}"
if [[ "${rc}" != "0" ]]; then
  echo "selftest reported errors; see ${report_path}" >&2
fi
exit "${rc}"

