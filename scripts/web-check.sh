#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/dev-env.sh"

cd "${ROOT_DIR}/web"

if [[ ! -d node_modules ]]; then
  echo "error: web dependencies are not installed; run pnpm install in web/" >&2
  exit 1
fi

pnpm format:check
pnpm lint
pnpm typecheck
pnpm test:run
pnpm build
pnpm test:e2e
