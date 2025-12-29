.PHONY: build clean install test run help

# Binary name
BINARY_NAME=telegraf-influxdb-input

# Build directory
BUILD_DIR=build

# Default target
all: build

## build: Build the plugin binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean
	@echo "Clean complete"

## install: Install the plugin to /usr/local/bin
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete"

## test: Run tests
test:
	@echo "Running tests..."
	go test -v ./...

## run: Build and run the plugin
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Formatting complete"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "Vet complete"

## check: Run all checks (fmt, vet, test)
check: fmt vet test
	@echo "All checks passed"

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' Makefile | column -t -s ':' | sed -e 's/^/ /'
