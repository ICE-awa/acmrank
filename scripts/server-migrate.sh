#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/dev-env.sh"
source "${ROOT_DIR}/scripts/load-local-env.sh"

GOOSE_VERSION="${GOOSE_VERSION:-v3.25.0}"
MIGRATIONS_DIR="${GOOSE_MIGRATIONS_DIR:-${ROOT_DIR}/server/db/migrations}"
POSTGRES_HOST="${POSTGRES_HOST:-127.0.0.1}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-acmrank}"
POSTGRES_USER="${POSTGRES_USER:-acmrank}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-acmrank_dev}"
POSTGRES_SSLMODE="${POSTGRES_SSLMODE:-disable}"

export PGPASSWORD="${POSTGRES_PASSWORD}"

exec go run "github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION}" \
  -dir "${MIGRATIONS_DIR}" \
  postgres \
  "host=${POSTGRES_HOST} port=${POSTGRES_PORT} user=${POSTGRES_USER} dbname=${POSTGRES_DB} sslmode=${POSTGRES_SSLMODE}" \
  "$@"
