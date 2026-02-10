.PHONY: help build test test-e2e test-all lint clean install version changelog version-dry-run changelog-dry-run

# Default target
help:
	@echo "Available targets:"
	@echo "  build             - Build the binary"
	@echo "  test              - Run unit tests"
	@echo "  test-e2e          - Run e2e integration tests"
	@echo "  test-all          - Run all tests (unit + e2e)"
	@echo "  lint              - Run golangci-lint"
	@echo "  clean             - Remove build artifacts"
	@echo "  install           - Install the binary to GOPATH/bin"
	@echo "  version           - Calculate next version based on commits"
	@echo "  version-dry-run   - Show version calculation details (dry run)"
	@echo "  changelog         - Generate changelog since last tag"
	@echo "  changelog-dry-run - Show changelog without writing file (dry run)"

# Build the binary
build:
	@echo "Building sigv4-proxy..."
	@go build -o sigv4-proxy -ldflags="-s -w" .

# Run unit tests
test:
	@echo "Running unit tests..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

# Run e2e tests
test-e2e:
	@echo "Running e2e tests..."
	@go test -tags=e2e -v ./e2e/...

# Run all tests
test-all: test test-e2e

# Run linter
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f sigv4-proxy coverage.txt
	@go clean

# Install binary
install:
	@echo "Installing sigv4-proxy..."
	@go install .

# Calculate next version based on conventional commits
version:
	@./scripts/version.sh

# Dry run: Show version calculation details
version-dry-run:
	@./scripts/version.sh --dry-run

# Generate changelog since last tag
changelog:
	@./scripts/changelog.sh

# Dry run: Show changelog without writing file
changelog-dry-run:
	@./scripts/changelog.sh --dry-run
