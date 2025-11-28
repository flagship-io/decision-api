.PHONY: help run setup start-infra stop-infra clean test

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Copy .env.example to .env (edit with your credentials)
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✓ Created .env file - please edit with your credentials"; \
	else \
		echo "✓ .env file already exists"; \
	fi

start-infra: ## Start Redis and Decision API with docker-compose
	@echo "Starting infrastructure..."
	docker compose -f docker-compose.demo.yml up -d redis decision-api
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@echo "✓ Infrastructure ready!"
	@echo ""
	@echo "Services:"
	@echo "  - Decision API: http://localhost:8080"
	@echo "  - Redis: localhost:6379"
	@echo ""
	@echo "Next: Update .env with your ENV_ID and API_KEY, then run 'make run'"

stop-infra: ## Stop all services
	docker compose -f docker-compose.demo.yml down

run: setup ## Run the demo application
	@echo "Running advanced features demo..."
	@echo ""
	go run main.go

run-docker: setup ## Run demo in Docker
	docker compose -f docker-compose.demo.yml --profile demo up demo-app

test: ## Test connection to Decision API
	@echo "Testing Decision API connection..."
	@curl -f http://localhost:8080/health || (echo "❌ Decision API not reachable" && exit 1)
	@echo "✓ Decision API is healthy"
	@echo ""
	@echo "Testing Redis connection..."
	@docker exec $$(docker ps -qf "name=redis") redis-cli ping || (echo "❌ Redis not reachable" && exit 1)
	@echo "✓ Redis is healthy"

logs: ## Show Decision API logs
	docker compose -f docker-compose.demo.yml logs -f decision-api

logs-redis: ## Show Redis logs
	docker compose -f docker-compose.demo.yml logs -f redis

clean: ## Remove all containers and volumes
	docker compose -f docker-compose.demo.yml down -v
	rm -f .env

inspect-cache: ## Inspect Redis cache contents
	@echo "Redis cache keys:"
	@docker exec $$(docker ps -qf "name=redis") redis-cli keys "*"
	@echo ""
	@echo "To view a specific key:"
	@echo "  docker exec -it $$(docker ps -qf 'name=redis') redis-cli get '<key>'"

clear-cache: ## Clear all Redis cache
	@echo "Clearing Redis cache..."
	@docker exec $$(docker ps -qf "name=redis") redis-cli flushall
	@echo "✓ Cache cleared"

build: ## Build the demo binary
	go build -o demo main.go
	@echo "✓ Built ./demo"

.DEFAULT_GOAL := help
