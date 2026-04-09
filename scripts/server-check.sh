#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/dev-env.sh"

cd "${ROOT_DIR}/server"
go fmt ./...
go build ./...
go vet ./...
go test ./...

if ! command -v cc >/dev/null 2>&1 && ! command -v gcc >/dev/null 2>&1 && ! command -v clang >/dev/null 2>&1; then
  echo "error: a C toolchain is required for go test -race" >&2
  exit 1
fi

CGO_ENABLED=1 go test -race ./...

if ! command -v golangci-lint >/dev/null 2>&1; then
  echo "error: golangci-lint is not installed" >&2
  exit 1
fi

golangci-lint run ./...
