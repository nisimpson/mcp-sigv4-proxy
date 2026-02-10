# AWS SigV4 Signing Proxy MCP Server

An MCP (Model Context Protocol) server that acts as a transparent proxy, forwarding MCP protocol messages to a target MCP server that requires IAM authentication. The proxy automatically signs requests with AWS Signature Version 4 (SigV4) or Signature Version 4A (SigV4a).

## Overview

This proxy enables MCP clients to connect to IAM-authenticated MCP servers without implementing signing logic themselves. It sits between the client and the target server, transparently forwarding messages while adding AWS authentication.

```
┌─────────────┐         ┌──────────────────────────┐         ┌─────────────────┐
│             │  stdio  │                          │  HTTPS  │                 │
│ MCP Client  │────────▶│  Signing Proxy           │────────▶│  Target MCP     │
│             │         │  (mcp.Server)            │  +SigV4 │  Server (IAM)   │
│             │◀────────│                          │◀────────│                 │
└─────────────┘         └──────────────────────────┘         └─────────────────┘
```

## Features

- **Transparent Proxying**: Forwards all MCP protocol messages without modification
- **AWS SigV4 Signing**: Automatically signs HTTP requests with AWS credentials
- **SigV4a Support**: Multi-region signing for global services
- **Standard Credential Chain**: Uses AWS SDK's default credential provider
- **Profile Support**: Can use named AWS credential profiles
- **Session Token Support**: Handles temporary credentials with session tokens

## Installation

```bash
go install github.com/nisimpson/mcp-sigv4-proxy/cmd/sigv4-proxy@latest
```

## Usage

### Configuration

The proxy can be configured via command-line flags or environment variables:

| Parameter | Flag | Environment Variable | Required | Default |
|-----------|------|---------------------|----------|---------|
| Target URL | `--target-url` | `MCP_TARGET_URL` | Yes | - |
| Region | `--region` | `AWS_REGION` | Yes | - |
| Service Name | `--service-name` | `AWS_SERVICE_NAME` | Yes | - |
| Signature Version | `--sig-version` | `AWS_SIG_VERSION` | No | `v4` |
| Profile | `--profile` | `AWS_PROFILE` | No | `default` |

### Example

```bash
sigv4-proxy \
  --target-url https://mcp-server.example.com \
  --region us-east-1 \
  --service-name execute-api \
  --sig-version v4
```

Or using environment variables:

```bash
export MCP_TARGET_URL=https://mcp-server.example.com
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api
export AWS_SIG_VERSION=v4

sigv4-proxy
```

## AWS Credentials

The proxy uses the standard AWS credential chain:

1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`)
2. Shared credentials file (`~/.aws/credentials`)
3. IAM role (when running on EC2, ECS, Lambda, etc.)

To use a specific profile:

```bash
sigv4-proxy --profile my-profile ...
```

## Development

### Prerequisites

- Go 1.25.7 or later
- AWS credentials configured

### Building

```bash
go build -o sigv4-proxy ./cmd/sigv4-proxy
```

### Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run property-based tests
go test -v ./... -rapid.checks=1000
```

## Project Structure

```
.
├── cmd/
│   └── sigv4-proxy/        # Main entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── proxy/              # Proxy server implementation
│   ├── signer/             # SigV4/SigV4a signing
│   └── transport/          # SigningTransport implementation
├── docs/                   # Additional documentation
└── .kiro/specs/sigv4-proxy/ # Specification documents
```

## License

[License information to be added]

## Contributing

[Contributing guidelines to be added]
