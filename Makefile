# Makefile for diagnose-mcp
# Constitution requirement: Build, test, lint targets

.PHONY: build test lint clean run help

# Binary name
BINARY_NAME=diagnose-mcp
BUILD_DIR=build

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Display this help message
help:
	@echo "Available targets:"
	@echo "  build      - Build the diagnose-mcp binary"
	@echo "  test       - Run all tests with coverage"
	@echo "  test-unit  - Run unit tests only"
	@echo "  test-integration - Run integration tests only"
	@echo "  bench      - Run benchmark tests"
	@echo "  lint       - Run golangci-lint"
	@echo "  fmt        - Format code with gofmt"
	@echo "  clean      - Remove build artifacts"
	@echo "  run        - Build and run (pass ARGS for arguments)"
	@echo "  deps       - Download dependencies"
	@echo "  coverage   - Generate test coverage report"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/diagnose-mcp

## test: Run all tests with coverage
test:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "Coverage report:"
	$(GOCMD) tool cover -func=coverage.out

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -race ./tests/unit/...

## test-integration: Run integration tests only
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race ./tests/integration/...

## bench: Run benchmark tests (constitution requirement: performance validation)
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./tests/benchmark/...

## coverage: Generate HTML coverage report
coverage: test
	@echo "Generating HTML coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint (constitution requirement)
lint:
	@echo "Running golangci-lint..."
	$(GOLINT) run ./...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## run: Build and run the binary (use ARGS="..." for arguments)
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install ./cmd/diagnose-mcp

.DEFAULT_GOAL := help
