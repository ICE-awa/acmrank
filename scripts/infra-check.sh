#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/dev-env.sh"

cd "${ROOT_DIR}"

docker compose ps

if [[ "${CHECK_POSTGRES:-1}" == "1" ]]; then
  docker compose exec -T postgres pg_isready -U "${POSTGRES_USER:-acmrank}" -d "${POSTGRES_DB:-acmrank}"
fi

if [[ "${CHECK_REDIS:-1}" == "1" ]]; then
  docker compose exec -T redis redis-cli ping
fi

if [[ "${CHECK_NATS:-1}" == "1" ]]; then
  curl -fsS "http://127.0.0.1:${NATS_MONITOR_PORT:-8222}/healthz" >/dev/null
fi

if [[ "${CHECK_ELASTICSEARCH:-1}" == "1" ]]; then
  curl -fsS "http://127.0.0.1:${ELASTICSEARCH_PORT:-9200}/_cluster/health?wait_for_status=yellow&timeout=5s" >/dev/null
fi

echo "local infrastructure is reachable"
