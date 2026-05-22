#!/usr/bin/env bash
#
# run-linux-selftest.sh — run the windowops selftest against the Linux X
# session this host is running, with a real Notepad-like target window.
#
# Requires: an X session ($DISPLAY set) and a target window (the script
# tries xterm, gnome-text-editor, xeyes — whichever launches).
#
# Usage:
#   bash scripts/non-github-ci/verify/run-linux-selftest.sh \
#        --scenario scenarios/notepad.json \
#        --out artifacts/verify/linux-$(date +%s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

scenario=""
out_dir="${REPO_ROOT}/artifacts/verify/linux-$(date +%s)"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --scenario)
      scenario="$2"; shift 2;;
    --out)
      out_dir="$2"; shift 2;;
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
