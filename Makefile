SHELL := /usr/bin/env bash
.RECIPEPREFIX := >

.PHONY: bootstrap bootstrap-server bootstrap-web bootstrap-sync
.PHONY: check check-server check-web check-sync
.PHONY: infra-up infra-down infra-logs infra-ps infra-check
.PHONY: migrate-status migrate-up migrate-down migrate-reset
.PHONY: fmt-server build-server vet-server test-server test-race-server lint-server
.PHONY: format-web format-check-web lint-web typecheck-web test-web e2e-web build-web

bootstrap: bootstrap-server bootstrap-web bootstrap-sync

bootstrap-server:
>source scripts/dev-env.sh && cd server && go mod tidy

bootstrap-web:
>bash scripts/bootstrap-web.sh

bootstrap-sync:
>bash scripts/bootstrap-sync.sh

check: check-server check-web check-sync

check-server:
>bash scripts/server-check.sh

check-web:
>bash scripts/web-check.sh

check-sync:
>bash scripts/sync-check.sh

infra-up:
>docker compose up -d

infra-down:
>docker compose down

infra-logs:
>docker compose logs -f

infra-ps:
>docker compose ps

infra-check:
>bash scripts/infra-check.sh

migrate-status:
>bash scripts/server-migrate.sh status

migrate-up:
>bash scripts/server-migrate.sh up

migrate-down:
>bash scripts/server-migrate.sh down

migrate-reset:
>bash scripts/server-migrate.sh reset

fmt-server:
>source scripts/dev-env.sh && cd server && go fmt ./...

build-server:
>source scripts/dev-env.sh && cd server && go build ./...

vet-server:
>source scripts/dev-env.sh && cd server && go vet ./...

test-server:
>source scripts/dev-env.sh && cd server && go test ./...

test-race-server:
>source scripts/dev-env.sh && cd server && CGO_ENABLED=1 go test -race ./...

lint-server:
>source scripts/dev-env.sh && cd server && golangci-lint run ./...

format-web:
>source scripts/dev-env.sh && cd web && pnpm format

format-check-web:
>source scripts/dev-env.sh && cd web && pnpm format:check

lint-web:
>source scripts/dev-env.sh && cd web && pnpm lint

typecheck-web:
>source scripts/dev-env.sh && cd web && pnpm typecheck

test-web:
>source scripts/dev-env.sh && cd web && pnpm test:run

e2e-web:
>source scripts/dev-env.sh && cd web && pnpm test:e2e

build-web:
>source scripts/dev-env.sh && cd web && pnpm build
