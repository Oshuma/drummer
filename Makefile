.DEFAULT_GOAL := help

# Makefile for the Drummer project

.PHONY: help test test-backend test-frontend dev prod build-dev build-prod clean

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

test: test-backend test-frontend ## Run all tests (backend and frontend)

test-backend: ## Run backend tests
	@echo "Running backend tests..."
	go test -v ./...

test-frontend: ## Run frontend tests
	@echo "Running frontend tests..."
	(cd web && npm test -- --watchAll=false)

dev: build-dev ## Start the development environment
	@echo "Starting development environment..."
	ENV=development docker compose up

prod: build-prod ## Start the production environment
	@echo "Starting production environment..."
	ENV=production GIN_MODE=release docker compose up -d

build-dev: ## Build for development
	@echo "Building for development..."
	ENV=development docker compose build

build-prod: ## Build for production
	@echo "Building for production..."
	ENV=production docker compose build

clean: ## Clean up containers and images
	@echo "Cleaning up containers and images..."
	docker compose down --volumes --remove-orphans
	docker system prune -f

