.PHONY: build run test clean docker-build docker-push install deps

APP_NAME=kubeforge
VERSION?=0.1.0
BUILD_DIR=bin
DOCKER_REGISTRY?=localhost:5000
DOCKER_IMAGE=$(DOCKER_REGISTRY)/$(APP_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build flags
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

all: build

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Build the application
build: deps
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./cmd/kubeforge-server

# Build for Linux (useful for cross-compilation)
build-linux: deps
	@echo "Building $(APP_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 ./cmd/kubeforge-server

# Run the application
run:
	@echo "Running $(APP_NAME)..."
	$(GORUN) ./cmd/kubeforge-server

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -f *.db

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest -f Dockerfile .

# Push Docker image
docker-push: docker-build
	@echo "Pushing Docker image..."
	docker push $(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_IMAGE):latest

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run -it --rm -p 8080:8080 -v $(PWD)/kubeforge.db:/kubeforge.db $(DOCKER_IMAGE):latest

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(APP_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(APP_NAME) $(GOPATH)/bin/

# Development: run with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@which air > /dev/null || (echo "air not installed. Run: go install github.com/cosmtrek/air@latest" && exit 1)
	air

# Generate API documentation (requires swag: go install github.com/swaggo/swag/cmd/swag@latest)
docs:
	@which swag > /dev/null || (echo "swag not installed. Run: go install github.com/swaggo/swag/cmd/swag@latest" && exit 1)
	swag init -g cmd/kubeforge-server/main.go

help:
	@echo "KubeForge Makefile Commands:"
	@echo "  make build           - Build the application"
	@echo "  make build-linux     - Build for Linux"
	@echo "  make run             - Run the application"
	@echo "  make test            - Run tests"
	@echo "  make test-coverage   - Run tests with coverage report"
	@echo "  make fmt             - Format code"
	@echo "  make lint            - Lint code"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make docker-build    - Build Docker image"
	@echo "  make docker-push     - Push Docker image"
	@echo "  make docker-run      - Run Docker container"
	@echo "  make install         - Install binary to GOPATH/bin"
	@echo "  make dev             - Run with auto-reload (requires air)"
	@echo "  make deps            - Download dependencies"
	@echo "  make help            - Show this help"
