.PHONY: dev build build-windows build-all clean lint test help

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)

# Binary names
BINARY_NAME := WinDash-Agent
WINDOWS_BINARY := $(BINARY_NAME).exe
DIST_DIR := dist

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev: ## Run the agent in development mode
	@echo "🚀 Running agent in development mode..."
	@go run ./cmd/agent

build: ## Build for current platform
	@echo "🔨 Building $(BINARY_NAME) for current platform..."
	@mkdir -p $(DIST_DIR)
	@CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/agent
	@echo "✅ Built: $(DIST_DIR)/$(BINARY_NAME)"

build-windows: ## Build for Windows (amd64)
	@echo "🔨 Building for Windows..."
	@mkdir -p $(DIST_DIR)
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(WINDOWS_BINARY) ./cmd/agent
	@echo "✅ Built: $(DIST_DIR)/$(WINDOWS_BINARY)"
	@ls -lh $(DIST_DIR)/$(WINDOWS_BINARY)

build-all: ## Build for all platforms using goreleaser
	@echo "🔨 Building for all platforms..."
	@goreleaser release --snapshot --clean
	@echo "✅ Built all platforms in dist/"

clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	@rm -rf $(DIST_DIR)
	@echo "✅ Cleaned"

lint: ## Run linters
	@echo "🔍 Running linters..."
	@go vet ./...
	@echo "✅ Lint passed"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test -v ./...
	@echo "✅ Tests passed"

deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies ready"

install-tools: ## Install development tools
	@echo "🔧 Installing tools..."
	@go install github.com/goreleaser/goreleaser@latest
	@echo "✅ Tools installed"
