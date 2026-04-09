#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/dev-env.sh"

PYTHON_BIN="$(find_python_bin || true)"

if [[ -z "${PYTHON_BIN}" ]]; then
  echo "error: python interpreter not found; install Python 3.13 to bootstrap sync/" >&2
  exit 1
fi

require_python_313 "${PYTHON_BIN}"

cd "${ROOT_DIR}/sync"
"${PYTHON_BIN}" -m pip install -e ".[dev]"
