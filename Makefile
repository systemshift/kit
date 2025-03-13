# Kit Makefile

.PHONY: all build cli demo clean test install setup help

# Default target
all: cli demo

# Build the CLI
cli:
	@echo "Building Kit CLI..."
	@mkdir -p bin
	@go build -o bin/kit ./cmd/kit

# Build and run the CLI
cli-run: cli
	@echo "Running Kit CLI..."
	@./bin/kit help

# Build the demonstration example
demo:
	@echo "Building Kit demonstration..."
	@mkdir -p bin
	@go build -o bin/kitdemo example.go

# Run the demonstration
demo-run:
	@echo "Running Kit demonstration..."
	@go run example.go

# Run with verbose output
demo-verbose:
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
	@go test ./pkg/...

# Install CLI (for development)
install:
	@echo "Installing Kit CLI..."
	@go install ./cmd/kit

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@go mod tidy

# Help
help:
	@echo "Kit - A Kernel-Oriented Version Control System"
	@echo ""
	@echo "Available commands:"
	@echo "  make cli          - Build the Kit CLI executable"
	@echo "  make cli-run      - Build and run the Kit CLI"
	@echo "  make demo         - Build the Kit demonstration"
	@echo "  make demo-run     - Run the Kit demonstration"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make install      - Install Kit CLI"
	@echo "  make setup        - Setup development environment"
	@echo "  make help         - Show this help message"
