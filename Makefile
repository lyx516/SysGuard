.PHONY: build run test clean deps fmt vet lint demo help

# Binary names
BINARY_NAME=sysguard
UI_BINARY_NAME=sysguard-ui

# Main package paths
MAIN_PATH=./cmd/sysguard
UI_PATH=./cmd/sysguard-ui

# Build directory
BUILD_DIR=./build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download dependencies
	$(GOMOD) download
	$(GOMOD) tidy

fmt: ## Format code
	$(GOFMT) -s -w .
	$(GOCMD) fmt ./...

vet: ## Run go vet
	$(GOVET) ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/" && exit 1)
	golangci-lint run

build: deps ## Build daemon and UI binaries
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	$(GOBUILD) -o $(BUILD_DIR)/$(UI_BINARY_NAME) $(UI_PATH)

run: build ## Build and run the binary
	$(BUILD_DIR)/$(BINARY_NAME)

run-ui: build ## Build and run the UI
	$(BUILD_DIR)/$(UI_BINARY_NAME)

demo: ## Start a local demo dashboard with a synthetic missing service
	./scripts/demo-local.sh

test: deps ## Run tests
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

test-coverage: test ## Display test coverage
	@echo "Coverage Report:"
	$(GOCMD) tool cover -func=coverage.out | grep total

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

install: build ## Install the binary to GOPATH/bin
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)
	$(GOBUILD) -o $(GOPATH)/bin/$(UI_BINARY_NAME) $(UI_PATH)

all: fmt vet lint build ## Run all checks and build
