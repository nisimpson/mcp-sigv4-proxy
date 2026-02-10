# AWS Credentials Setup Guide

This guide provides detailed instructions for configuring AWS credentials for the SigV4 Signing Proxy.

## Overview

The proxy uses the AWS SDK for Go v2, which supports multiple credential sources through the standard credential chain. Credentials are loaded in the following order (first found wins):

1. Environment variables
2. Shared credentials file (`~/.aws/credentials`)
3. Shared configuration file (`~/.aws/config`)
4. IAM role (when running on AWS infrastructure)

## Credential Sources

### 1. Environment Variables

Environment variables are the simplest way to provide credentials, especially for testing or containerized environments.

#### Required Variables

```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### Optional Variables

```bash
# For temporary credentials (e.g., from AWS STS)
export AWS_SESSION_TOKEN=AQoDYXdzEJr...

# To specify a region (can also be set via --region flag)
export AWS_REGION=us-east-1

# To use a specific profile from credentials file
export AWS_PROFILE=production
```

#### Example: Running with Environment Variables

```bash
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com
export AWS_REGION=us-east-1
export AWS_SERVICE_NAME=execute-api

sigv4-proxy
```

### 2. Shared Credentials File

The shared credentials file is the recommended approach for local development and allows you to manage multiple sets of credentials.

#### File Location

- **Linux/macOS**: `~/.aws/credentials`
- **Windows**: `%USERPROFILE%\.aws\credentials`

#### File Format

```ini
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

[production]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY

[staging]
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
aws_session_token = AQoDYXdzEJr...  # For temporary credentials
```

#### Using a Specific Profile

```bash
# Via command-line flag
sigv4-proxy --profile production ...

# Via environment variable
export AWS_PROFILE=production
sigv4-proxy ...
```

#### Creating the Credentials File

You can create the credentials file manually or use the AWS CLI:

```bash
# Install AWS CLI
# macOS: brew install awscli
# Linux: pip install awscli
# Windows: Download from https://aws.amazon.com/cli/

# Configure credentials interactively
aws configure

# Configure a named profile
aws configure --profile production
```

### 3. Shared Configuration File

The shared configuration file can also contain credentials and additional settings.

#### File Location

- **Linux/macOS**: `~/.aws/config`
- **Windows**: `%USERPROFILE%\.aws\config`

#### File Format

```ini
[default]
region = us-east-1
output = json

[profile production]
region = us-west-2
output = json
aws_access_key_id = AKIAI44QH8DHBEXAMPLE
aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
```

**Note**: In the config file, profiles are prefixed with `profile` (except for `[default]`).

### 4. IAM Roles (Recommended for AWS Infrastructure)

When running on AWS infrastructure, the proxy can automatically use IAM roles without any credential configuration.

#### EC2 Instance Profile

1. Create an IAM role with the necessary permissions
2. Attach the role to your EC2 instance
3. The proxy will automatically use the instance profile credentials

```bash
# No credential configuration needed!
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

#### ECS Task Role

1. Create an IAM role with the necessary permissions
2. Assign the role to your ECS task definition
3. The proxy will automatically use the task role credentials

```json
{
  "family": "mcp-proxy-task",
  "taskRoleArn": "arn:aws:iam::123456789012:role/mcp-proxy-task-role",
  "containerDefinitions": [
    {
      "name": "sigv4-proxy",
      "image": "sigv4-proxy:latest",
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
      ]
    }
  ]
}
```

#### Lambda Execution Role

When running as a Lambda function, the proxy will automatically use the Lambda execution role.

## Temporary Credentials

Temporary credentials are issued by AWS Security Token Service (STS) and include a session token. They are commonly used for:

- Cross-account access
- Federated users
- IAM roles
- MFA-protected access

### Using Temporary Credentials

Temporary credentials include three components:

```bash
export AWS_ACCESS_KEY_ID=ASIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
export AWS_SESSION_TOKEN=AQoDYXdzEJr1K...
```

Or in the credentials file:

```ini
[temporary]
aws_access_key_id = ASIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
aws_session_token = AQoDYXdzEJr1K...
```

### Obtaining Temporary Credentials

#### Using AWS STS

```bash
# Assume a role
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/MyRole \
  --role-session-name my-session

# Get session token (for MFA)
aws sts get-session-token \
  --serial-number arn:aws:iam::123456789012:mfa/user \
  --token-code 123456
```

#### Using AWS SSO

```bash
# Configure SSO
aws configure sso

# Login
aws sso login --profile my-sso-profile

# Use the profile
sigv4-proxy --profile my-sso-profile ...
```

## Required IAM Permissions

The AWS credentials used by the proxy need permissions to invoke the target MCP server. The exact permissions depend on your target server, but here's a typical example for API Gateway:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "execute-api:Invoke"
      ],
      "Resource": [
        "arn:aws:execute-api:us-east-1:123456789012:abc123/*"
      ]
    }
  ]
}
```

### Creating an IAM Policy

1. Go to the IAM console
2. Click "Policies" → "Create policy"
3. Use the JSON editor to paste the policy above
4. Adjust the resource ARN to match your API Gateway
5. Name the policy (e.g., "MCPProxyInvokePolicy")
6. Attach the policy to your IAM user or role

## Security Best Practices

### 1. Use IAM Roles When Possible

IAM roles are more secure than long-term credentials because:
- Credentials are automatically rotated
- No credentials need to be stored or managed
- Permissions can be scoped to specific resources

### 2. Use Temporary Credentials

When you can't use IAM roles, use temporary credentials:
- They expire automatically
- They can be scoped to specific permissions
- They support MFA

### 3. Rotate Long-Term Credentials Regularly

If you must use long-term credentials:
- Rotate them every 90 days
- Use AWS IAM credential reports to track age
- Delete unused credentials

### 4. Use Least Privilege

Grant only the permissions needed:
- Scope permissions to specific resources
- Use conditions to restrict access
- Regularly review and remove unnecessary permissions

### 5. Never Commit Credentials to Version Control

- Use `.gitignore` to exclude credential files
- Use environment variables or secret management services
- Scan repositories for accidentally committed secrets

### 6. Use AWS Secrets Manager or Parameter Store

For production deployments, consider using AWS Secrets Manager or Systems Manager Parameter Store:

```bash
# Store credentials in Secrets Manager
aws secretsmanager create-secret \
  --name mcp-proxy-credentials \
  --secret-string '{"access_key":"AKIA...","secret_key":"..."}'

# Retrieve and use in your application
# (requires additional code to fetch from Secrets Manager)
```

## Troubleshooting

### "failed to load AWS credentials"

**Possible Causes:**
1. No credentials configured in any source
2. Credentials file has incorrect format
3. Profile name doesn't exist
4. IAM role not attached to instance/task

**Solutions:**
1. Verify credentials are set via environment variables or credentials file
2. Check file format matches the examples above
3. List available profiles: `aws configure list-profiles`
4. Verify IAM role is attached in AWS console

### "ExpiredToken" or "InvalidToken"

**Cause:** Temporary credentials have expired.

**Solution:** Refresh your credentials:
```bash
# For AWS SSO
aws sso login --profile my-profile

# For assumed roles
# Re-run the assume-role command
```

### "Access Denied"

**Cause:** Credentials don't have permission to invoke the target server.

**Solution:** 
1. Verify the IAM policy includes `execute-api:Invoke` permission
2. Check the resource ARN matches your API Gateway
3. Verify the credentials are for the correct AWS account

### Credentials Not Found in Expected Location

**Check credential file location:**
```bash
# Linux/macOS
ls -la ~/.aws/credentials

# Windows
dir %USERPROFILE%\.aws\credentials
```

**Check environment variables:**
```bash
# Linux/macOS
env | grep AWS

# Windows
set | findstr AWS
```

## Examples

### Example 1: Local Development with Credentials File

```bash
# Create credentials file
mkdir -p ~/.aws
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
EOF

# Run proxy
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

### Example 2: Docker Container with Environment Variables

```dockerfile
FROM golang:1.21 AS builder
WORKDIR /app
COPY . .
RUN go build -o sigv4-proxy ./cmd/sigv4-proxy

FROM debian:bookworm-slim
COPY --from=builder /app/sigv4-proxy /usr/local/bin/
ENTRYPOINT ["sigv4-proxy"]
```

```bash
docker run \
  -e AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE \
  -e AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  -e MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com \
  -e AWS_REGION=us-east-1 \
  -e AWS_SERVICE_NAME=execute-api \
  sigv4-proxy
```

### Example 3: EC2 with Instance Profile

```bash
# No credentials needed - uses instance profile
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

### Example 4: Multiple Profiles for Different Environments

```bash
# Development
sigv4-proxy --profile dev \
  --target-url https://dev-api.example.com \
  --region us-east-1 \
  --service-name execute-api

# Production
sigv4-proxy --profile prod \
  --target-url https://prod-api.example.com \
  --region us-east-1 \
  --service-name execute-api
```

## Additional Resources

- [AWS SDK for Go v2 - Credentials](https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials)
- [AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
- [IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
- [AWS Security Token Service](https://docs.aws.amazon.com/STS/latest/APIReference/welcome.html)
