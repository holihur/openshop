SHELL := /bin/bash
BACKEND := backend
FRONTEND := frontend

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## Resolve Go dependencies
	cd $(BACKEND) && go mod tidy

.PHONY: build
build: ## Build the backend binaries
	cd $(BACKEND) && go build ./...

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
fe-install: ## Install frontend dependencies
	cd $(FRONTEND) && pnpm install

.PHONY: fe-dev
fe-dev: ## Run the frontend dev server
	cd $(FRONTEND) && pnpm run dev

.PHONY: fe-build
fe-build: ## Build the frontend
	cd $(FRONTEND) && pnpm run build
