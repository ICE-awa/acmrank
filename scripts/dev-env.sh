#!/usr/bin/env bash

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "error: source scripts/dev-env.sh instead of executing it directly" >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CACHE_DIR="${ROOT_DIR}/.cache"

export XDG_CACHE_HOME="${XDG_CACHE_HOME:-${CACHE_DIR}/xdg}"
export COREPACK_HOME="${COREPACK_HOME:-${CACHE_DIR}/corepack}"
export NPM_CONFIG_CACHE="${NPM_CONFIG_CACHE:-${CACHE_DIR}/npm}"
export PNPM_HOME="${PNPM_HOME:-${CACHE_DIR}/pnpm-home}"
export PNPM_STORE_DIR="${PNPM_STORE_DIR:-${CACHE_DIR}/pnpm-store}"
export PLAYWRIGHT_BROWSERS_PATH="${PLAYWRIGHT_BROWSERS_PATH:-${CACHE_DIR}/playwright}"
export PLAYWRIGHT_DOWNLOAD_CONNECTION_TIMEOUT="${PLAYWRIGHT_DOWNLOAD_CONNECTION_TIMEOUT:-120000}"
export PIP_CACHE_DIR="${PIP_CACHE_DIR:-${CACHE_DIR}/pip}"
export GOCACHE="${GOCACHE:-${CACHE_DIR}/go-build}"
export GOTMPDIR="${GOTMPDIR:-${CACHE_DIR}/go-tmp}"
export GOPATH="${GOPATH:-${CACHE_DIR}/gopath}"

mkdir -p \
  "${XDG_CACHE_HOME}" \
  "${COREPACK_HOME}" \
  "${NPM_CONFIG_CACHE}" \
  "${PNPM_HOME}" \
  "${PNPM_STORE_DIR}" \
  "${PLAYWRIGHT_BROWSERS_PATH}" \
  "${PIP_CACHE_DIR}" \
  "${GOCACHE}" \
  "${GOTMPDIR}" \
  "${GOPATH}"

find_python_bin() {
  local candidate=""

  for candidate in python3.13 python3 python; do
    if command -v "${candidate}" >/dev/null 2>&1; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done

  return 1
}

require_python_313() {
  local python_bin="${1}"
  local detected_version=""

  detected_version="$("${python_bin}" -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')"
  if ! "${python_bin}" -c 'import sys; raise SystemExit(0 if sys.version_info >= (3, 13) else 1)'; then
    echo "error: ${python_bin} reports Python ${detected_version}, but sync/ requires >= 3.13" >&2
    return 1
  fi
}
