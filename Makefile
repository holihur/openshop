SHELL := /bin/bash
BACKEND := backend

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## Resolve Go dependencies
	cd $(BACKEND) && go mod tidy

.PHONY: build
build: ## Build the backend binaries (serves the embedded SPAs if present)
	cd $(BACKEND) && go build ./...

.PHONY: embed
embed: ## Build front + ops and embed them into the backend
	./scripts/embed-frontend.sh

.PHONY: test
test: ## Run backend tests
	cd $(BACKEND) && go test ./... -race -count=1

.PHONY: test-integration
test-integration: ## Run PostgreSQL integration tests
	cd $(BACKEND) && TEST_DATABASE_URL="$${TEST_DATABASE_URL:-host=localhost port=5432 user=openshop password=openshop dbname=openshop sslmode=disable TimeZone=UTC}" \
		go test ./internal/adapter/postgres/ -run TestOutboxClaimSemantics -v

.PHONY: smoke
smoke: ## Run the end-to-end smoke test against a running API
	API_BASE="$${API_BASE:-http://localhost:8080/api/v1}" ./scripts/smoke.sh

.PHONY: smoke-web
smoke-web: ## Verify the embedded SPAs are served by a running server
	WEB_BASE="$${WEB_BASE:-http://localhost:8080}" ./scripts/smoke-web.sh

.PHONY: lint
lint: ## Run gofmt check and go vet
	cd $(BACKEND) && test -z "$$(gofmt -l .)" && go vet ./...

.PHONY: vet
vet: ## Run go vet
	cd $(BACKEND) && go vet ./...

.PHONY: fmt
fmt: ## Format Go code
	cd $(BACKEND) && gofmt -w .

.PHONY: run
run: ## Run the API locally (expects postgres/redis/nats)
	cd $(BACKEND) && go run ./cmd/server

.PHONY: migrate
migrate: ## Apply database migrations
	cd $(BACKEND) && go run ./cmd/migrate -dir migrations

.PHONY: migrate-down
migrate-down: ## Revert the last N migrations (N=1 by default)
	cd $(BACKEND) && go run ./cmd/migrate -dir migrations -down $(or $(N),1)

.PHONY: seed
seed: ## Insert demo data
	cd $(BACKEND) && go run ./cmd/seed

.PHONY: infra
infra: ## Start postgres, redis and nats
	docker compose up -d postgres redis nats

.PHONY: up
up: ## Start the whole stack
	docker compose up --build -d

.PHONY: down
down: ## Stop the stack
	docker compose down

.PHONY: fe-install
fe-install: ## Install frontend dependencies (pnpm workspace)
	pnpm install

.PHONY: fe-dev
fe-dev: ## Run the storefront dev server (http://localhost:5173)
	pnpm --filter @openshop/front dev

.PHONY: ops-dev
ops-dev: ## Run the admin console dev server (http://localhost:5174)
	pnpm --filter @openshop/ops dev

.PHONY: fe-build
fe-build: ## Build both SPAs and copy them into the backend
	./scripts/embed-frontend.sh

.PHONY: release
release: ## Build release artifacts locally with GoReleaser (snapshot)
	goreleaser release --snapshot --clean
