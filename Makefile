
# Makefile for the Drummer project

.PHONY: test

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
