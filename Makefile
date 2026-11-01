SHELL := /bin/bash
.DEFAULT_GOAL := help

-include .env
export

API_PORT ?= 8080
WEB_PORT ?= 3000

.PHONY: help
help: ## List targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.env:
	cp .env.example .env

.PHONY: up
up: .env ## Start Postgres + Caddy (http://binarysust.localhost)
	mkdir -p .data/media
	docker compose up -d --wait

.PHONY: down
down: ## Stop Postgres + Caddy (data kept)
	docker compose down

.PHONY: reset
reset: ## Stop everything and delete the local database
	docker compose down -v

.PHONY: migrate
migrate: up ## Apply database migrations
	cd api && go run ./cmd/server migrate

.PHONY: seed
seed: migrate ## Load demo data (terms, people, events, posts)
	cd api && go run ./cmd/seed

.PHONY: api
api: ## Run the Go API on :$(API_PORT)
	cd api && PORT=$(API_PORT) go run ./cmd/server

.PHONY: web
web: ## Run the TanStack Start dev server on :$(WEB_PORT)
	cd web && pnpm dev --port $(WEB_PORT) --strictPort

.PHONY: snapshot
snapshot: ## Build web/.snapshot from the local database
	cd api && go run ./cmd/snapshot -out ../web

.PHONY: gen
gen: ## Regenerate openapi.json and the web API types
	cd api && go run ./cmd/server openapi > openapi.json
	cd web && pnpm gen:api

.PHONY: dev
dev: up ## Everything: db + caddy + api + web (Ctrl-C stops api/web)
	@echo "→ http://binarysust.localhost"
	@trap 'kill 0' EXIT; \
	  $(MAKE) --no-print-directory api 2>&1 | sed -u 's/^/[api] /' & \
	  $(MAKE) --no-print-directory web 2>&1 | sed -u 's/^/[web] /' & \
	  wait

.PHONY: test
test: up ## Run API + web checks
	cd api && EKMAN_TEST_DATABASE_URL="$(DATABASE_URL)" go test ./...
	cd web && pnpm typecheck && pnpm lint
