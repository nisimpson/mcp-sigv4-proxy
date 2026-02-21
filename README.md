# AWS SigV4 Signing Proxy MCP Server

[![Test](https://github.com/nisimpson/mcp-sigv4-proxy/actions/workflows/test.yml/badge.svg)](https://github.com/nisimpson/mcp-sigv4-proxy/actions/workflows/test.yml)
[![Release](https://github.com/nisimpson/mcp-sigv4-proxy/actions/workflows/release.yml/badge.svg)](https://github.com/nisimpson/mcp-sigv4-proxy/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nisimpson/mcp-sigv4-proxy)](https://goreportcard.com/report/github.com/nisimpson/mcp-sigv4-proxy)
[![GoDoc](https://godoc.org/github.com/nisimpson/mcp-sigv4-proxy?status.svg)](https://godoc.org/github.com/nisimpson/mcp-sigv4-proxy)

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
- **Server-Sent Events**: Optional SSE support for streaming responses
- **Request Timeout**: Configurable timeout for HTTP requests to target server
- **Custom Headers**: Add custom headers to proxied requests
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals for clean shutdown
- **Structured Logging**: Provides detailed logging for debugging and monitoring

## Quick Start

### Installation

#### From Source

```bash
git clone https://github.com/nisimpson/mcp-sigv4-proxy.git
cd mcp-sigv4-proxy
go build -o sigv4-proxy ./sigv4-proxy
```

#### Using Go Install

```bash
go install github.com/nisimpson/mcp-sigv4-proxy@latest
```

### Basic Usage

1. **Set up AWS credentials** (see [AWS Credentials](#aws-credentials) section)

2. **Configure the proxy** using environment variables:

```bash
export MCP_TARGET_URL=https://your-mcp-server.execute-api.us-east-1.amazonaws.com
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api
```

3. **Run the proxy**:

```bash
sigv4-proxy
```

The proxy will start and listen for MCP protocol messages on stdin/stdout.

## Configuration

### Configuration Options

The proxy can be configured via command-line flags or environment variables. Command-line flags take precedence over environment variables.

| Parameter | Flag | Environment Variable | Required | Default | Description |
|-----------|------|---------------------|----------|---------|-------------|
| Target URL | `--target-url` | `MCP_TARGET_URL` | Yes | - | The HTTPS endpoint of the target MCP server |
| Region | `--region` | `AWS_REGION` | Yes | - | AWS region for signing (e.g., us-east-1) |
| Service Name | `--service-name` | `AWS_SERVICE_NAME` | Yes | - | AWS service name for signing (e.g., execute-api) |
| Signature Version | `--sig-version` | `AWS_SIG_VERSION` | No | `v4` | Signature version: `v4` or `v4a` |
| Profile | `--profile` | `AWS_PROFILE` | No | `default` | AWS credential profile name |
| Enable SSE | `--sse` | `MCP_ENABLE_SSE` | No | `false` | Enable Server-Sent Events for streaming responses |
| Timeout | `--timeout` | `MCP_TIMEOUT` | No | No timeout | Request timeout duration (e.g., 30s, 1m) |
| Headers | `--headers` | `MCP_HEADERS` | No | - | Comma-delimited custom headers (format: key=value,key2=value2) |

### Configuration Examples

#### Example 1: API Gateway MCP Server

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

#### Example 2: Using Environment Variables

```bash
export MCP_TARGET_URL=https://abc123.execute-api.us-west-2.amazonaws.com
export AWS_REGION=us-west-2
export AWS_SERVICE_NAME=execute-api
export AWS_SIG_VERSION=v4

sigv4-proxy
```

#### Example 3: Multi-Region with SigV4a

```bash
sigv4-proxy \
  --target-url https://global-mcp-server.example.com \
  --region us-east-1 \
  --service-name execute-api \
  --sig-version v4a
```

#### Example 4: Using a Named AWS Profile

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api \
  --profile production
```

#### Example 5: With Server-Sent Events and Timeout

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api \
  --sse \
  --timeout 30s
```

#### Example 6: With Custom Headers

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api \
  --headers "X-Custom-Header=value,X-API-Version=v2"
```

See [docs/examples.md](docs/examples.md) for more detailed configuration examples.

## AWS Credentials

The proxy uses the standard AWS SDK credential chain to load credentials. Credentials are loaded in the following order:

1. **Environment Variables**
   - `AWS_ACCESS_KEY_ID`
   - `AWS_SECRET_ACCESS_KEY`
   - `AWS_SESSION_TOKEN` (optional, for temporary credentials)

2. **Shared Credentials File** (`~/.aws/credentials`)
   - Default profile or named profile specified via `--profile` or `AWS_PROFILE`

3. **IAM Role** (when running on AWS infrastructure)
   - EC2 instance profile
   - ECS task role
   - Lambda execution role

### Setting Up Credentials

#### Option 1: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_SESSION_TOKEN=AQoDYXdzEJr...  # Optional, for temporary credentials
```

#### Option 2: AWS Credentials File

Create or edit `~/.aws/credentials`:

```ini
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
```

Then use the profile:

```bash
sigv4-proxy --profile production ...
```

#### Option 3: IAM Role (Recommended for AWS Infrastructure)

When running on AWS infrastructure (EC2, ECS, Lambda), the proxy will automatically use the IAM role attached to the instance/task/function. No additional configuration is needed.

For more details, see [docs/aws-credentials.md](docs/aws-credentials.md).

## Usage with MCP Clients

### Claude Desktop

Add the proxy to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "iam-protected-server": {
      "command": "/path/to/sigv4-proxy",
      "args": [
        "--target-url", "https://abc123.execute-api.us-east-1.amazonaws.com",
        "--region", "us-east-1",
        "--service-name", "execute-api"
      ]
    }
  }
}
```

### Other MCP Clients

The proxy communicates via stdio, so it can be used with any MCP client that supports stdio transport. Configure your client to launch the proxy as a subprocess and communicate via stdin/stdout.

## Troubleshooting

### Common Issues

#### "configuration error: target URL is required"

**Cause**: The target URL was not provided.

**Solution**: Set `MCP_TARGET_URL` environment variable or use `--target-url` flag.

#### "failed to load AWS credentials"

**Cause**: AWS credentials are not configured or not accessible.

**Solution**: 
- Verify credentials are set via environment variables, `~/.aws/credentials`, or IAM role
- Check that the profile name is correct if using `--profile`
- Ensure credentials have not expired (for temporary credentials)

#### "target URL must use http or https scheme"

**Cause**: The target URL is malformed or uses an unsupported protocol.

**Solution**: Ensure the target URL starts with `https://` (or `http://` for local testing).

#### "signature version must be 'v4' or 'v4a'"

**Cause**: An invalid signature version was specified.

**Solution**: Use either `v4` or `v4a` for the `--sig-version` flag or `AWS_SIG_VERSION` environment variable.

For more troubleshooting tips, see [docs/troubleshooting.md](docs/troubleshooting.md).

## Security Considerations

- **Credential Protection**: The proxy never logs or exposes AWS credentials. Access keys are masked in logs.
- **HTTPS Required**: Target server connections should use HTTPS to protect data in transit.
- **Signature Security**: AWS SDK handles signature generation securely according to AWS specifications.
- **No Credential Forwarding**: The proxy does not forward credentials to the client.

## Performance

- **Connection Pooling**: HTTP connections to the target server are reused for efficiency.
- **Minimal Latency**: Signing adds approximately 1-2ms overhead per request.
- **Streaming**: Large responses are streamed without buffering the entire payload in memory.

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing guidelines, and how to submit pull requests.

## Support

For issues, questions, or contributions, please visit the [GitHub repository](https://github.com/nisimpson/mcp-sigv4-proxy).

## Additional Documentation

- [AWS Credentials Setup Guide](docs/aws-credentials.md)
- [Configuration Examples](docs/examples.md)
- [Troubleshooting Guide](docs/troubleshooting.md)
