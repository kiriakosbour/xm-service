.PHONY: all build run test lint clean docker-build docker-up docker-down help

# Variables
APP_NAME := xm-company-service
BINARY := server
GO := go
DOCKER_COMPOSE := docker-compose

# Default target
all: lint test build

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	$(GO) build -o bin/$(BINARY) ./cmd/server

# Run the application locally
run:
	@echo "Running $(APP_NAME)..."
	$(GO) run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -race -tags=integration ./tests/...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	goimports -w .

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

docker-up:
	@echo "Starting services..."
	$(DOCKER_COMPOSE) up -d

docker-up-build:
	@echo "Building and starting services..."
	$(DOCKER_COMPOSE) up -d --build

docker-down:
	@echo "Stopping services..."
	$(DOCKER_COMPOSE) down

docker-logs:
	@echo "Showing logs..."
	$(DOCKER_COMPOSE) logs -f

docker-clean:
	@echo "Cleaning Docker resources..."
	$(DOCKER_COMPOSE) down -v --rmi local

# Database commands
db-migrate:
	@echo "Running migrations..."
	psql -h localhost -U xm_user -d xm_db -f migrations/001_init.sql

# Help
help:
	@echo "Available targets:"
	@echo "  all              - Run lint, test, and build"
	@echo "  build            - Build the application binary"
	@echo "  run              - Run the application locally"
	@echo "  test             - Run unit tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-integration - Run integration tests"
	@echo "  lint             - Run golangci-lint"
	@echo "  fmt              - Format code"
	@echo "  tidy             - Tidy go modules"
	@echo "  clean            - Remove build artifacts"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start all services"
	@echo "  docker-up-build  - Build and start all services"
	@echo "  docker-down      - Stop all services"
	@echo "  docker-logs      - Show service logs"
	@echo "  docker-clean     - Remove Docker resources"
	@echo "  help             - Show this help"
