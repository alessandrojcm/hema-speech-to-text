# HEMA Tournament Replay System Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=hema-replay-system
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_PATH=./cmd/replay-system

# Build flags
LDFLAGS=-ldflags "-s -w"
BUILD_FLAGS=-trimpath $(LDFLAGS)

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test coverage deps fmt vet lint run install help

all: deps fmt vet test build

# Build the application
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Build complete: $(BINARY_PATH)$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning...$(NC)"
	$(GOCLEAN)
	@rm -rf bin/
	@rm -rf coverage/
	@echo "$(GREEN)Clean complete$(NC)"

# Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) -v -race ./...

# Run integration tests (requires OBS Studio)
test-integration:
	@echo "$(GREEN)Running OBS integration tests...$(NC)"
	@./scripts/test-obs-integration.sh

# Run tests with coverage
coverage:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@mkdir -p coverage
	$(GOTEST) -v -race -coverprofile=coverage/coverage.out ./...
	$(GOCMD) tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "$(GREEN)Coverage report generated: coverage/coverage.html$(NC)"

# Install dependencies
deps:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	$(GOMOD) download
	$(GOMOD) tidy

# Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOCMD) fmt ./...

# Vet code
vet:
	@echo "$(GREEN)Vetting code...$(NC)"
	$(GOCMD) vet ./...

# Run golint (install with: go install golang.org/x/lint/golint@latest)
lint:
	@echo "$(GREEN)Running golint...$(NC)"
	@if command -v golint > /dev/null; then \
		golint ./...; \
	else \
		echo "$(YELLOW)golint not installed, skipping...$(NC)"; \
	fi

# Run the application
run: build
	@echo "$(GREEN)Running $(BINARY_NAME)...$(NC)"
	$(BINARY_PATH)

# Run with config file
run-config: build
	@echo "$(GREEN)Running $(BINARY_NAME) with config...$(NC)"
	$(BINARY_PATH) -config config/settings.yaml

# Install the application
install: build
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)

# Development tools
dev-setup:
	@echo "$(GREEN)Setting up development environment...$(NC)"
	$(GOGET) golang.org/x/lint/golint
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run golangci-lint
golangci-lint:
	@echo "$(GREEN)Running golangci-lint...$(NC)"
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "$(YELLOW)golangci-lint not installed, run 'make dev-setup' first$(NC)"; \
	fi

# Watch for changes and rebuild (requires entr: brew install entr)
watch:
	@echo "$(GREEN)Watching for changes...$(NC)"
	@if command -v entr > /dev/null; then \
		find . -name "*.go" | entr -r make build; \
	else \
		echo "$(RED)entr not installed. Install with: brew install entr$(NC)"; \
	fi

# Quick development cycle
dev: fmt vet test build

# Release build with optimizations
release:
	@echo "$(GREEN)Building release version...$(NC)"
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PATH)
	@echo "$(GREEN)Release build complete: $(BINARY_PATH)$(NC)"

# Cross-compile for different platforms
build-all:
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "$(GREEN)Cross-compilation complete$(NC)"

# Help
help:
	@echo "$(GREEN)Available targets:$(NC)"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-integration - Run OBS integration tests (requires OBS Studio)"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  deps         - Install dependencies"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  lint         - Run golint"
	@echo "  run          - Build and run the application"
	@echo "  run-config   - Build and run with config file"
	@echo "  install      - Install the application"
	@echo "  dev-setup    - Set up development tools"
	@echo "  golangci-lint- Run golangci-lint"
	@echo "  watch        - Watch for changes and rebuild"
	@echo "  dev          - Quick development cycle (fmt, vet, test, build)"
	@echo "  release      - Build optimized release version"
	@echo "  build-all    - Cross-compile for multiple platforms"
	@echo "  help         - Show this help message"