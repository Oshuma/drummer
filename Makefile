
# Makefile for the Drummer project

.PHONY: test dev prod build-dev build-prod clean

ENV ?= development

test: test-backend test-frontend

## Backend Tests
.PHONY: test-backend
test-backend:
	@echo "Running backend tests..."
	go test -v ./...

## Frontend Tests
.PHONY: test-frontend
test-frontend:
	@echo "Running frontend tests..."
	(cd web && npm test -- --watchAll=false)

## Development Environment
.PHONY: dev
dev: build-dev
	@echo "Starting development environment..."
	ENV=development docker compose up

## Production Environment
.PHONY: prod
prod: build-prod
	@echo "Starting production environment..."
	ENV=production GIN_MODE=release docker compose up -d

## Build for Development
.PHONY: build-dev
build-dev:
	@echo "Building for development..."
	ENV=development docker compose build

## Build for Production
.PHONY: build-prod
build-prod:
	@echo "Building for production..."
	ENV=production docker compose build

## Clean up containers and images
.PHONY: clean
clean:
	@echo "Cleaning up containers and images..."
	docker compose down --volumes --remove-orphans
	docker system prune -f
