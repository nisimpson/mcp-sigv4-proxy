# Configuration Examples

This document provides detailed configuration examples for various use cases of the SigV4 Signing Proxy.

## Table of Contents

- [Basic Examples](#basic-examples)
- [AWS Service Examples](#aws-service-examples)
- [Deployment Examples](#deployment-examples)
- [Advanced Examples](#advanced-examples)

## Basic Examples

### Example 1: Minimal Configuration

The simplest configuration with required parameters only:

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

This uses:
- Default signature version (v4)
- Default AWS profile (default)
- Credentials from the standard credential chain

### Example 2: Using Environment Variables

Configure everything via environment variables:

```bash
export MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api

sigv4-proxy
```

### Example 3: Mixed Configuration

Combine environment variables and command-line flags (flags take precedence):

```bash
# Set some defaults via environment
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api

# Override target URL via flag
sigv4-proxy --target-url https://abc123.execute-api.us-east-1.amazonaws.com
```

## AWS Service Examples

### API Gateway

API Gateway is the most common use case for the proxy.

#### REST API

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com/prod \
  --region us-east-1 \
  --service-name execute-api
```

#### HTTP API

```bash
sigv4-proxy \
  --target-url https://xyz789.execute-api.us-west-2.amazonaws.com \
  --region us-west-2 \
  --service-name execute-api
```

#### Regional API

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.eu-west-1.amazonaws.com \
  --region eu-west-1 \
  --service-name execute-api
```

### Application Load Balancer (ALB)

For MCP servers behind an ALB with IAM authentication:

```bash
sigv4-proxy \
  --target-url https://my-alb-123456.us-east-1.elb.amazonaws.com \
  --region us-east-1 \
  --service-name elasticloadbalancing
```

### Lambda Function URL

For Lambda function URLs with IAM authentication:

```bash
sigv4-proxy \
  --target-url https://abc123.lambda-url.us-east-1.on.aws \
  --region us-east-1 \
  --service-name lambda
```

### Custom Service

For custom AWS services or internal services using SigV4:

```bash
sigv4-proxy \
  --target-url https://custom-service.example.com \
  --region us-east-1 \
  --service-name my-custom-service
```

## Deployment Examples

### Docker Container

#### Dockerfile

```dockerfile
FROM golang:1.21 AS builder

WORKDIR /app
COPY . .
RUN go build -o sigv4-proxy ./cmd/sigv4-proxy

FROM debian:bookworm-slim

# Install CA certificates for HTTPS
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/sigv4-proxy /usr/local/bin/

ENTRYPOINT ["sigv4-proxy"]
```

#### Running the Container

```bash
# Build
docker build -t sigv4-proxy .

# Run with environment variables
docker run \
  -e AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} \
  -e AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY} \
  -e MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com \
  -e AWS_REGION=us-east-1 \
  -e AWS_SERVICE_NAME=execute-api \
  -i \
  sigv4-proxy

# Run with credentials file mounted
docker run \
  -v ~/.aws:/root/.aws:ro \
  -e MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com \
  -e AWS_REGION=us-east-1 \
  -e AWS_SERVICE_NAME=execute-api \
  -i \
  sigv4-proxy
```

#### Docker Compose

```yaml
version: '3.8'

services:
  sigv4-proxy:
    build: .
    environment:
      MCP_TARGET_URL: https://abc123.execute-api.us-east-1.amazonaws.com
      AWS_REGION: us-east-1
      AWS_SERVICE_NAME: execute-api
      AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID}
      AWS_SECRET_ACCESS_KEY: ${AWS_SECRET_ACCESS_KEY}
    stdin_open: true
    tty: true
```

### Kubernetes

#### Deployment with Environment Variables

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sigv4-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sigv4-proxy
  template:
    metadata:
      labels:
        app: sigv4-proxy
    spec:
      containers:
      - name: sigv4-proxy
        image: sigv4-proxy:latest
        env:
        - name: MCP_TARGET_URL
          value: "https://abc123.execute-api.us-east-1.amazonaws.com"
        - name: AWS_REGION
          value: "us-east-1"
        - name: AWS_SERVICE_NAME
          value: "execute-api"
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-access-key
        stdin: true
        tty: true
```

#### Secret for Credentials

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
type: Opaque
stringData:
  access-key-id: AKIAIOSFODNN7EXAMPLE
  secret-access-key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### Using IAM Roles for Service Accounts (IRSA)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: sigv4-proxy
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/sigv4-proxy-role
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sigv4-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: sigv4-proxy
  template:
    metadata:
      labels:
        app: sigv4-proxy
    spec:
      serviceAccountName: sigv4-proxy
      containers:
      - name: sigv4-proxy
        image: sigv4-proxy:latest
        env:
        - name: MCP_TARGET_URL
          value: "https://abc123.execute-api.us-east-1.amazonaws.com"
        - name: AWS_REGION
          value: "us-east-1"
        - name: AWS_SERVICE_NAME
          value: "execute-api"
        stdin: true
        tty: true
```

### AWS ECS

#### Task Definition

```json
{
  "family": "sigv4-proxy",
  "taskRoleArn": "arn:aws:iam::123456789012:role/sigv4-proxy-task-role",
  "executionRoleArn": "arn:aws:iam::123456789012:role/ecsTaskExecutionRole",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "sigv4-proxy",
      "image": "123456789012.dkr.ecr.us-east-1.amazonaws.com/sigv4-proxy:latest",
      "environment": [
        {
          "name": "MCP_TARGET_URL",
          "value": "https://abc123.execute-api.us-east-1.amazonaws.com"
        },
        {
          "name": "AWS_REGION",
          "value": "us-east-1"
        },
        {
          "name": "AWS_SERVICE_NAME",
          "value": "execute-api"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/sigv4-proxy",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

### AWS Lambda

While the proxy is designed for stdio transport, you can adapt it for Lambda:

```go
package main

import (
    "context"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/nisimpson/mcp-sigv4-proxy/internal/config"
    "github.com/nisimpson/mcp-sigv4-proxy/internal/proxy"
    // ... other imports
)

func handler(ctx context.Context, event map[string]interface{}) (map[string]interface{}, error) {
    // Initialize proxy
    // Forward request
    // Return response
}

func main() {
    lambda.Start(handler)
}
```

### EC2 Instance

#### User Data Script

```bash
#!/bin/bash

# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Clone and build proxy
git clone https://github.com/nisimpson/mcp-sigv4-proxy.git
cd mcp-sigv4-proxy
go build -o /usr/local/bin/sigv4-proxy ./cmd/sigv4-proxy

# Create systemd service
cat > /etc/systemd/system/sigv4-proxy.service << EOF
[Unit]
Description=SigV4 Signing Proxy
After=network.target

[Service]
Type=simple
Environment="MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com"
Environment="AWS_REGION=us-east-1"
Environment="AWS_SERVICE_NAME=execute-api"
ExecStart=/usr/local/bin/sigv4-proxy
Restart=always

[Install]
WantedBy=multi-user.target
EOF

# Start service
systemctl daemon-reload
systemctl enable sigv4-proxy
systemctl start sigv4-proxy
```

## Advanced Examples

### Multi-Region with SigV4a

For services that span multiple regions:

```bash
sigv4-proxy \
  --target-url https://global-mcp-server.example.com \
  --region us-east-1 \
  --service-name execute-api \
  --sig-version v4a
```

### Using Named Profiles

#### Development Environment

```bash
sigv4-proxy \
  --profile dev \
  --target-url https://dev-api.example.com \
  --region us-east-1 \
  --service-name execute-api
```

#### Production Environment

```bash
sigv4-proxy \
  --profile prod \
  --target-url https://prod-api.example.com \
  --region us-east-1 \
  --service-name execute-api
```

### Shell Script Wrapper

Create a wrapper script for easier usage:

```bash
#!/bin/bash
# proxy.sh - Wrapper script for sigv4-proxy

set -e

# Default values
PROFILE="${AWS_PROFILE:-default}"
REGION="${AWS_REGION:-us-east-1}"
SERVICE="${AWS_SERVICE_NAME:-execute-api}"
SIG_VERSION="${AWS_SIG_VERSION:-v4}"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --target-url)
      TARGET_URL="$2"
      shift 2
      ;;
    --profile)
      PROFILE="$2"
      shift 2
      ;;
    --region)
      REGION="$2"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Validate required parameters
if [ -z "$TARGET_URL" ]; then
  echo "Error: --target-url is required"
  exit 1
fi

# Run proxy
exec sigv4-proxy \
  --target-url "$TARGET_URL" \
  --region "$REGION" \
  --service-name "$SERVICE" \
  --sig-version "$SIG_VERSION" \
  --profile "$PROFILE"
```

Usage:

```bash
chmod +x proxy.sh
./proxy.sh --target-url https://abc123.execute-api.us-east-1.amazonaws.com
```

### Claude Desktop Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):

```json
{
  "mcpServers": {
    "iam-protected-api": {
      "command": "/usr/local/bin/sigv4-proxy",
      "args": [
        "--target-url", "https://abc123.execute-api.us-east-1.amazonaws.com",
        "--region", "us-east-1",
        "--service-name", "execute-api"
      ]
    },
    "production-api": {
      "command": "/usr/local/bin/sigv4-proxy",
      "args": [
        "--target-url", "https://prod-api.example.com",
        "--region", "us-west-2",
        "--service-name", "execute-api",
        "--profile", "production"
      ]
    }
  }
}
```

### Testing with Local HTTP Server

For testing without AWS:

```bash
# Start a local HTTP server (in another terminal)
python3 -m http.server 8080

# Run proxy pointing to local server
sigv4-proxy \
  --target-url http://localhost:8080 \
  --region us-east-1 \
  --service-name execute-api
```

**Note**: This will sign requests, but the local server won't validate signatures.

### Cross-Account Access

Access an MCP server in a different AWS account:

```bash
# Assume role in target account
aws sts assume-role \
  --role-arn arn:aws:iam::987654321098:role/CrossAccountMCPAccess \
  --role-session-name proxy-session \
  --profile source-account

# Extract credentials from output and set environment variables
export AWS_ACCESS_KEY_ID=ASIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_SESSION_TOKEN=AQoDYXdzEJr...

# Run proxy
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

### Using AWS SSO

```bash
# Configure SSO
aws configure sso

# Login
aws sso login --profile my-sso-profile

# Run proxy with SSO profile
sigv4-proxy \
  --profile my-sso-profile \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

### Environment-Specific Configuration Files

Create configuration files for different environments:

**dev.env**:
```bash
export MCP_TARGET_URL=https://dev-api.example.com
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api
export AWS_PROFILE=dev
```

**prod.env**:
```bash
export MCP_TARGET_URL=https://prod-api.example.com
export AWS_REGION=us-west-2
export AWS_SERVICE_NAME=execute-api
export AWS_PROFILE=prod
```

Usage:

```bash
# Development
source dev.env
sigv4-proxy

# Production
source prod.env
sigv4-proxy
```

## Troubleshooting Examples

### Debug Mode

While the proxy doesn't have a built-in debug mode, you can capture logs:

```bash
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api \
  2> proxy.log
```

### Testing Connectivity

Test if the target server is reachable:

```bash
curl -v https://abc123.execute-api.us-east-1.amazonaws.com
```

### Verifying Credentials

Test if credentials are working:

```bash
aws sts get-caller-identity --profile your-profile
```

## Additional Resources

- [AWS Credentials Setup Guide](aws-credentials.md)
- [Troubleshooting Guide](troubleshooting.md)
- [Main README](../README.md)
