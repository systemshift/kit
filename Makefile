# Kit Makefile

.PHONY: build run clean test

# Default target
all: build

# Build the project
build:
	@echo "Building Kit..."
	@go build -o bin/kit example.go

# Run the demo
run:
	@echo "Running Kit demonstration..."
	@go run example.go

# Run with verbose output
run-verbose:
	@echo "Running Kit demonstration with verbose output..."
	@go run -v example.go

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@go clean

# Run tests
test:
	@echo "Running tests..."
	@go test ./kernel/...

# Install (for development)
install:
	@echo "Installing Kit..."
	@go install

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@go mod tidy

# Help
help:
	@echo "Kit - A Kernel-Oriented Version Control System"
	@echo ""
	@echo "Available commands:"
	@echo "  make build        - Build the Kit executable"
	@echo "  make run          - Run the demonstration"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make install      - Install Kit"
	@echo "  make setup        - Setup development environment"
	@echo "  make help         - Show this help message"
