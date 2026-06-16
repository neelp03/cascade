.PHONY: help up down build lint test smoke seed

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

up: ## Start all services via docker compose
	docker compose -f infra/docker-compose.yml up -d --build

down: ## Stop and remove containers
	docker compose -f infra/docker-compose.yml down

logs: ## Tail all service logs
	docker compose -f infra/docker-compose.yml logs -f

build: ## Build all TS packages + Go services
	pnpm build
	$(MAKE) go-build

go-build: ## Build Go services
	cd services/ingest && go build ./...
	cd services/writer && go build ./...
	cd services/query  && go build ./...

lint: ## Lint all code
	pnpm lint
	$(MAKE) go-lint

go-lint: ## Run golangci-lint on Go services
	@for svc in ingest writer query; do \
		echo "==> lint services/$$svc"; \
		cd services/$$svc && golangci-lint run ./... && cd ../..; \
	done

test: ## Run all tests
	pnpm test
	$(MAKE) go-test

go-test: ## Run Go tests
	@for svc in ingest writer query; do \
		echo "==> test services/$$svc"; \
		cd services/$$svc && go test ./... && cd ../..; \
	done

smoke: ## Run M1 smoke test (requires services up)
	bash scripts/smoke-test.sh

seed: ## Seed ClickHouse + Postgres with dev data
	bash scripts/seed.sh

install: ## Install Node deps
	pnpm install
