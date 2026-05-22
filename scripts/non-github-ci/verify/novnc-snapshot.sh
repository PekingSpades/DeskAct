#!/usr/bin/env bash
#
# novnc-snapshot.sh — grab a single frame from a dockur VM's noVNC web
# viewer to produce a PNG. Useful as cross-check evidence that pairs
# with the selftest report.
#
# Strategy: the dockur images expose a "control" / RFB endpoint on
# localhost:<port>. We connect with `vncsnapshot` if available, else
# spawn `xvfb-run` + `vncviewer -nojpeg -shared -snapshot=file.jpg`
# which works on most Linux distros.
#
# Usage:
#   bash scripts/non-github-ci/verify/novnc-snapshot.sh \
#        --target windows \
#        --out artifacts/verify/novnc-windows.png

set -euo pipefail

target=""
out_file=""
host="localhost"
port=""
password="admin"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --target)
      target="$2"; shift 2;;
    --out)
      out_file="$2"; shift 2;;
    --host)
      host="$2"; shift 2;;
    --port)
      port="$2"; shift 2;;
    --password)
      password="$2"; shift 2;;
    -h|--help)
      grep -E '^#( |$)' "$0" | sed 's/^#//' >&2
      exit 0;;
    *)
      echo "unknown arg: $1" >&2; exit 64;;
  esac
done

if [[ -z "${target}" || -z "${out_file}" ]]; then
  echo "--target and --out are required" >&2
  exit 64
fi

if [[ -z "${port}" ]]; then
  case "${target}" in
    windows) port="${DESKACT_WINDOWS_RDP_PORT:-3400}";;
    macos)   port="${DESKACT_MACOS_VNC_PORT:-5921}";;
    *) echo "unknown target ${target}" >&2; exit 64;;
  esac
fi

# Prefer vncsnapshot (most reliable).
if command -v vncsnapshot >/dev/null 2>&1; then
  echo "[novnc-snapshot] using vncsnapshot host=${host} port=${port}"
  echo -n "${password}" > /tmp/.deskact-vncpass-$$
  trap 'rm -f /tmp/.deskact-vncpass-$$' EXIT
  vncsnapshot -passwd /tmp/.deskact-vncpass-$$ \
    "${host}::${port}" "${out_file}"
  exit $?
fi

# Fallback: ffmpeg's vncconnect-style demuxer.
if command -v ffmpeg >/dev/null 2>&1; then
  echo "[novnc-snapshot] using ffmpeg vnc demuxer"
  ffmpeg -hide_banner -loglevel error \
    -f rawvnc -i "vnc://:${password}@${host}:${port}" \
    -frames:v 1 -y "${out_file}" || {
    echo "ffmpeg vnc capture failed — install vncsnapshot or run noVNC manually" >&2
    exit 3
  }
  exit 0
fi

echo "neither vncsnapshot nor ffmpeg-with-vnc is available." >&2
echo "manual fallback: open http://${host}:$(($port - 394)) (the dockur noVNC port)" >&2
echo "                 and use your browser's screenshot facility." >&2
exit 4
