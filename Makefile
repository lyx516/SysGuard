.PHONY: build run test clean deps fmt vet lint docker-build docker-run help

# Binary name
BINARY_NAME=sysguard

# Main package path
MAIN_PATH=./cmd/sysguard

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

build: deps ## Build the binary
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

run: build ## Build and run the binary
	$(BUILD_DIR)/$(BINARY_NAME)

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

docker-build: ## Build Docker image
	docker build -t sysguard:latest .

docker-run: ## Run Docker container
	docker run -it --rm sysguard:latest

install: build ## Install the binary to GOPATH/bin
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(MAIN_PATH)

all: fmt vet lint build ## Run all checks and build
