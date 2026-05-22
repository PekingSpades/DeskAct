#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib.sh"

requested_version=""
output_dir=""

usage() {
  cat <<'EOF' >&2
Usage: finalize-artifacts.sh [options]

Options:
  --version=<version>     Version label used for the artifact directory.
  --output-dir=<dir>      Versioned artifact directory. Defaults to artifacts/non-github-ci/<version>.
EOF
  exit 64
}

for arg in "$@"; do
  case "${arg}" in
    --version=*)
      requested_version="${arg#*=}"
      ;;
    --output-dir=*)
      output_dir="${arg#*=}"
      ;;
    -h | --help)
      usage
      ;;
    *)
      usage
      ;;
  esac
done

version="$(resolve_version "${requested_version}")"
if [[ -z "${output_dir}" ]]; then
  output_dir="$(artifact_dir_for_version "${DEFAULT_ARTIFACT_ROOT}" "${version}")"
fi
ensure_artifact_dirs "${output_dir}"
write_checksums_and_manifest "${output_dir}" "${version}"
