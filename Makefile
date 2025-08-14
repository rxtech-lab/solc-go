.PHONY: build test fmt clean help

# Default target
all: fmt build test

# Build the package
build:
	go build ./...

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Clean build artifacts
clean:
	go clean ./...

# Tidy dependencies
tidy:
	go mod tidy

# Run linter (if available)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping lint"; \
	fi

# Help
help:
	@echo "Available targets:"
	@echo "  build  - Build the package"
	@echo "  test   - Run tests"
	@echo "  fmt    - Format code"
	@echo "  clean  - Clean build artifacts"
	@echo "  tidy   - Tidy dependencies"
	@echo "  lint   - Run linter (if available)"
	@echo "  all    - Run fmt, build, and test"
	@echo "  help   - Show this help message"