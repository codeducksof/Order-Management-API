.PHONY: all build run test test-coverage lint fmt vet clean docker-up docker-down docker-build tidy install-tools generate-mocks swagger help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=api
BINARY_PATH=./bin/$(BINARY_NAME)

# Docker
DOCKER_COMPOSE=docker-compose

# Default target
all: lint test build

## build: Build the application
build:
	@echo "Building..."
	@mkdir -p ./bin
	CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_PATH) ./cmd/api
	@echo "Build complete: $(BINARY_PATH)"

## run: Run the application
run:
	@echo "Running..."
	$(GORUN) ./cmd/api

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -short ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./...

## lint: Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local order-management-api .; \
	fi

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf ./bin
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	$(GOMOD) tidy

## install-tools: Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Tools installed"

## generate-mocks: Generate mock files
generate-mocks:
	@echo "Generating mocks..."
	@if command -v mockery >/dev/null 2>&1; then \
		mockery --all --dir=./internal/domain --output=./internal/mocks --outpkg=mocks; \
	else \
		echo "mockery not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

## swagger: Generate Swagger documentation
swagger:
	@echo "Generating Swagger documentation..."
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/api/main.go -o ./docs; \
	else \
		echo "swag not installed. Run 'make install-tools' first."; \
		exit 1; \
	fi

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t order-management-api:latest .

## docker-up: Start Docker services
docker-up:
	@echo "Starting Docker services..."
	$(DOCKER_COMPOSE) up -d

## docker-down: Stop Docker services
docker-down:
	@echo "Stopping Docker services..."
	$(DOCKER_COMPOSE) down

## docker-logs: View Docker logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f

## docker-restart: Restart Docker services
docker-restart: docker-down docker-up

## dev: Start development environment
dev: docker-up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@make run

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## help: Show this help message
help:
	@echo "Order Management API - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
