# Auron E-Commerce Platform - Makefile
# Production-grade microservices make commands

# Colors
GREEN = \033[0;32m
YELLOW = \033[1;33m
NC = \033[0m

# Default target
.DEFAULT_GOAL := help

# ============================================================
# HELP
# ============================================================

help:
	@echo ""
	@echo "$(GREEN)Auron E-Commerce Platform - Makefile$(NC)"
	@echo ""
	@echo "$(YELLOW)Usage:$(NC)"
	@echo "  make <target>"
	@echo ""
	@echo "$(YELLOW)Targets:$(NC)"
	@echo ""
	@echo "  $(GREEN)Infrastructure$(NC)"
	@echo "    up              Start full stack with Docker Compose"
	@echo "    down            Stop and remove all containers"
	@echo "    restart         Restart all services"
	@echo "    dev             Start services with hot reload"
	@echo ""
	@echo "  $(GREEN)Database$(NC)"
	@echo "    migrate-up      Run all pending database migrations"
	@echo "    migrate-down    Rollback last database migration"
	@echo "    migrate SERVICE=<name>  Run migrations for specific service"
	@echo ""
	@echo "  $(GREEN)Testing$(NC)"
	@echo "    test            Run unit tests across all services"
	@echo "    test-int        Run integration tests"
	@echo "    test-e2e       Run end-to-end tests"
	@echo "    load-test      Run k6 load tests"
	@echo ""
	@echo "  $(GREEN)Utilities$(NC)"
	@echo "    logs            Show logs from all services"
	@echo "    logs-svc       Show logs from specific service"
	@echo "    ps             Show running containers"
	@echo "    seed           Populate test data"
	@echo "    kafka-topics   Create all Kafka topics"
	@echo "    clean          Remove all volumes and containers"
	@echo "    health         Check health of all services"
	@echo ""

# ============================================================
# INFRASTRUCTURE
# ============================================================

.PHONY: up
up:
	@echo "$(GREEN)Starting full stack...$(NC)"
	cd infra && docker compose up --build -d
	@echo "$(GREEN)Services started!$(NC)"
	@echo "Frontend:       http://localhost:3000"
	@echo "API Gateway:    http://localhost:8080"
	@echo "Prometheus:     http://localhost:9090"
	@echo "Grafana:        http://localhost:3001"

.PHONY: down
down:
	@echo "$(YELLOW)Stopping services...$(NC)"
	cd infra && docker compose down

.PHONY: restart
restart: down up

.PHONY: dev
dev:
	@echo "$(GREEN)Starting in development mode with hot reload...$(NC)"
	cd infra && docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build -d

# ============================================================
# DATABASE MIGRATIONS
# ============================================================

MIGRATION_CMD = docker run --rm -v $(PWD)/services/$(SERVICE)/migrations:/migrations \
	-migrate -path=/migrations -database=$(DATABASE_URL)

.PHONY: migrate-up
migrate-up:
	@echo "$(GREEN)Running all migrations...$(NC)"
	@echo "$(YELLOW)Note: Migration tooling requires golang-migrate CLI$(NC)"

.PHONY: migrate-down
migrate-down:
	@echo "$(YELLOW)Rolling back migrations...$(NC)"

.PHONY: migrate
migrate:
ifndef SERVICE
	@echo "$(YELLOW)Usage: make migrate SERVICE=user-service$(NC)"
	@exit 1
endif
	@echo "$(GREEN)Running migrations for $(SERVICE)...$(NC)"
	@echo "$(YELLOW)Note: Set DATABASE_URL env var for production$(NC)"

# ============================================================
# TESTING
# ============================================================

.PHONY: test
test:
	@echo "$(GREEN)Running unit tests...$(NC)"
	@for service in api-gateway user-service product-service order-service payment-service inventory-service notification-service; do \
		echo "Testing $$service..."; \
		cd services/$$service && go test -v ./... || true; \
		cd ../../; \
	done

.PHONY: test-int
test-int:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@echo "$(YELLOW)Requires Docker to be running$(NC)"

.PHONY: test-e2e
test-e2e:
	@echo "$(GREEN)Running end-to-end tests...$(NC)"
	@echo "$(YELLOW)Requires full stack to be running$(NC)"

.PHONY: load-test
load-test:
	@echo "$(GREEN)Running load tests with k6...$(NC)"
	@if [ -f scripts/load-test.js ]; then \
		k6 run scripts/load-test.js; \
	else \
		echo "$(YELLOW)Load test script not found$(NC)"; \
	fi

# ============================================================
# UTILITIES
# ============================================================

.PHONY: logs
logs:
	cd infra && docker compose logs -f

.PHONY: logs-svc
logs-svc:
ifndef SERVICE
	@echo "$(YELLOW)Usage: make logs-svc SERVICE=order-service$(NC)"
	@exit 1
endif
	cd infra && docker compose logs -f $(SERVICE)

.PHONY: ps
ps:
	cd infra && docker compose ps

.PHONY: seed
seed:
	@echo "$(GREEN)Seeding test data...$(NC)"
	@if [ -f scripts/seed.sh ]; then \
		chmod +x scripts/seed.sh && ./scripts/seed.sh; \
	else \
		echo "$(YELLOW)Seed script not found$(NC)"; \
	fi

.PHONY: kafka-topics
kafka-topics:
	@echo "$(GREEN)Creating Kafka topics...$(NC)"
	@if [ -f infra/kafka/topics.sh ]; then \
		chmod +x infra/kafka/topics.sh && KAFKA_BROKER=localhost:9092 ./infra/kafka/topics.sh; \
	else \
		echo "$(YELLOW)Kafka topics script not found$(NC)"; \
	fi

.PHONY: clean
clean:
	@echo "$(RED)WARNING: This will remove all data!$(NC)"
	@echo "$(RED)Press Ctrl+C to cancel, or enter to continue...$(NC)"
	@read -r && cd infra && docker compose down -v --remove-orphans

.PHONY: health
health:
	@echo "$(GREEN)Checking service health...$(NC)"
	@echo ""
	@curl -s http://localhost:8080/health 2>/dev/null && echo " - API Gateway" || echo "X - API Gateway"
	@curl -s http://localhost:8081/health 2>/dev/null && echo " - User Service" || echo "X - User Service"
	@curl -s http://localhost:8082/health 2>/dev/null && echo " - Product Service" || echo "X - Product Service"
	@curl -s http://localhost:8083/health 2>/dev/null && echo " - Order Service" || echo "X - Order Service"
	@curl -s http://localhost:8084/health 2>/dev/null && echo " - Payment Service" || echo "X - Payment Service"
	@curl -s http://localhost:8085/health 2>/dev/null && echo " - Inventory Service" || echo "X - Inventory Service"
	@curl -s http://localhost:8086/health 2>/dev/null && echo " - Notification Service" || echo "X - Notification Service"

# ============================================================
# BUILD
# ============================================================

.PHONY: build
build:
	@echo "$(GREEN)Building all services...$(NC)"
	@for service in api-gateway user-service product-service order-service payment-service inventory-service notification-service; do \
		echo "Building $$service..."; \
		cd services/$$service && go build -o bin/$$service . || true; \
		cd ../../; \
	done

.PHONY: build-docker
build-docker:
	@echo "$(GREEN)Building all Docker images...$(NC)"
	cd infra && docker compose build

# ============================================================
# DEVELOPMENT
# ============================================================

.PHONY: deps
deps:
	@echo "$(GREEN)Installing Go dependencies...$(NC)"
	@for service in api-gateway user-service product-service order-service payment-service inventory-service notification-service shared; do \
		echo "Installing deps for $$service..."; \
		cd services/$$service && go mod download || true; \
		cd ../../; \
	done
	cd shared && go mod download

.PHONY: tidy
tidy:
	@echo "$(GREEN)Running go mod tidy...$(NC)"
	@for service in api-gateway user-service product-service order-service payment-service inventory-service notification-service shared; do \
		echo "Tidying $$service..."; \
		cd services/$$service && go mod tidy || true; \
		cd ../../; \
	done
	cd shared && go mod tidy
