# Contributing to AWS SigV4 Signing Proxy MCP Server

Thank you for your interest in contributing! This document provides guidelines and information for developers working on this project.

## Development Setup

### Prerequisites

- Go 1.25.7 or later
- AWS credentials configured
- Git
- golangci-lint (for linting)

### Getting Started

1. **Clone the repository**

```bash
git clone https://github.com/nisimpson/mcp-sigv4-proxy.git
cd mcp-sigv4-proxy
```

2. **Install dependencies**

```bash
go mod download
```

3. **Build the project**

```bash
make build
```

4. **Run tests**

```bash
make test
```

## Building from Source

### Using Make

```bash
# Build the binary
make build

# Install directly to GOPATH/bin
make install

# Run the binary
./sigv4-proxy --help
```

### Using Go Commands

```bash
# Build
go build -o sigv4-proxy .

# Install
go install .
```

## Running Tests

### Unit Tests

```bash
# Run all unit tests
make test

# Run with verbose output
go test -v ./...

# Run with coverage
go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
```

### End-to-End Tests

```bash
# Run e2e integration tests
make test-e2e

# Run all tests (unit + e2e)
make test-all
```

### Linting

```bash
# Run golangci-lint
make lint

# Run with auto-fix
golangci-lint run --fix
```

## Available Make Targets

- `make build` - Build the binary
- `make test` - Run unit tests with coverage
- `make test-e2e` - Run e2e integration tests
- `make test-all` - Run all tests (unit + e2e)
- `make lint` - Run golangci-lint
- `make clean` - Remove build artifacts
- `make install` - Install binary to GOPATH/bin
- `make version` - Calculate next version based on commits
- `make version-dry-run` - Show version calculation details (dry run)
- `make changelog` - Generate changelog since last tag
- `make changelog-dry-run` - Preview changelog without writing file (dry run)
- `make help` - Show all available targets

## Testing Release Scripts Locally

You can test the versioning and changelog generation without making any changes:

```bash
# Preview what version would be calculated
make version-dry-run

# Preview the changelog that would be generated
make changelog-dry-run
```

These dry-run targets show you exactly what would happen during a release without creating tags or modifying files.

## Project Structure

```
.
├── internal/
│   ├── config/             # Configuration management
│   ├── credentials/        # AWS credential loading
│   ├── proxy/              # Proxy server implementation
│   ├── signer/             # SigV4/SigV4a signing
│   └── transport/          # SigningTransport implementation
├── e2e/                    # End-to-end integration tests
├── docs/                   # Additional documentation
├── scripts/                # Build and release scripts
├── .github/workflows/      # GitHub Actions CI/CD
├── main.go                 # Main entry point
├── Makefile                # Build automation
└── go.mod                  # Go module definition
```

## Architecture

### Components

1. **Config Package** (`internal/config`)
   - Loads configuration from environment variables and command-line flags
   - Validates required parameters
   - Provides structured configuration to other components

2. **Credentials Package** (`internal/credentials`)
   - Wraps AWS SDK credential provider
   - Supports standard AWS credential chain
   - Handles profile-based credentials

3. **Signer Package** (`internal/signer`)
   - Implements SigV4 and SigV4a signing
   - Abstracts signing logic behind a common interface
   - Uses AWS SDK v2 signers

4. **Transport Package** (`internal/transport`)
   - Implements `http.RoundTripper` interface
   - Integrates signing into HTTP request flow
   - Preserves original request properties

5. **Proxy Package** (`internal/proxy`)
   - Implements MCP server using `mcp.Server`
   - Forwards messages to target server
   - Handles stdio communication

### Request Flow

```
Client (stdio) → Proxy Server → SigningTransport → Signer → Target Server (HTTPS)
                                                              ↓
Client (stdio) ← Proxy Server ← HTTP Response ←──────────────┘
```

## Testing Strategy

### Unit Tests

- Test individual components in isolation
- Mock external dependencies (AWS SDK, HTTP clients)
- Focus on business logic and error handling

### Property-Based Tests

- Use `pgregory.net/rapid` for property-based testing
- Test invariants and edge cases
- Validate behavior across wide input ranges

### Error Tests

- Dedicated error scenario tests
- Validate error messages and types
- Test error propagation

### End-to-End Tests

- Test complete request flow
- Use real HTTP servers (httptest)
- Validate integration between components

## Code Style

### General Guidelines

- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write clear, descriptive comments
- Keep functions small and focused

### Error Handling

```go
// Good: Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to load credentials: %w", err)
}

// Bad: Return raw errors
if err != nil {
    return err
}
```

### Logging

```go
// Use structured logging with slog
slog.Info("proxy started", "target", cfg.TargetURL)
slog.Error("request failed", "error", err, "url", url)
```

## Commit Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build process, tooling, dependencies
- `ci`: CI/CD changes

### Examples

```
feat(signer): add SigV4a support for multi-region signing

Implement SigV4a signer using AWS SDK v2. This enables
signing requests for global services that span multiple regions.

Refs #42
```

```
fix(transport): preserve request context during signing

Ensure the original request context is maintained through
the signing process to support cancellation and timeouts.

Fixes #156
```

## Pull Request Process

1. **Fork the repository** and create a feature branch
2. **Make your changes** following the code style guidelines
3. **Add tests** for new functionality
4. **Run tests and linter** to ensure everything passes
5. **Update documentation** if needed
6. **Commit your changes** using conventional commit format
7. **Push to your fork** and submit a pull request
8. **Respond to feedback** from maintainers

### PR Checklist

- [ ] Code compiles without errors
- [ ] All tests pass (`make test-all`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated (if applicable)
- [ ] Commit messages follow conventional commits
- [ ] No debug code or commented-out code
- [ ] No secrets or credentials in code

## Release Process

Releases are automated via GitHub Actions:

1. **Version Calculation**: Based on conventional commits since last tag
   - `feat:` → minor version bump
   - `fix:` → patch version bump
   - `BREAKING CHANGE:` → major version bump

2. **Changelog Generation**: Automatically generated from commit messages

3. **Release Creation**: GitHub release with binaries for multiple platforms

### Manual Release Testing

```bash
# Preview next version
make version-dry-run

# Preview changelog
make changelog-dry-run
```

## Getting Help

- **Issues**: Report bugs or request features via [GitHub Issues](https://github.com/nisimpson/mcp-sigv4-proxy/issues)
- **Discussions**: Ask questions in [GitHub Discussions](https://github.com/nisimpson/mcp-sigv4-proxy/discussions)
- **Pull Requests**: Submit contributions via pull requests

## Code of Conduct

- Be respectful and inclusive
- Provide constructive feedback
- Focus on the code, not the person
- Help others learn and grow

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
