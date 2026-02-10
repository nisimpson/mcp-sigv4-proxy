# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with the SigV4 Signing Proxy.

## Table of Contents

- [Configuration Issues](#configuration-issues)
- [Credential Issues](#credential-issues)
- [Connection Issues](#connection-issues)
- [Signing Issues](#signing-issues)
- [Runtime Issues](#runtime-issues)
- [Debugging Tips](#debugging-tips)

## Configuration Issues

### Error: "configuration error: target URL is required"

**Symptoms:**
```
ERROR: configuration error: target URL is required (set MCP_TARGET_URL or --target-url)
```

**Cause:** The target URL was not provided in environment variables or command-line flags.

**Solution:**
```bash
# Option 1: Use environment variable
export MCP_TARGET_URL=https://abc123.execute-api.us-east-1.amazonaws.com
sigv4-proxy --region us-east-1 --service-name execute-api

# Option 2: Use command-line flag
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api
```

### Error: "region is required"

**Symptoms:**
```
ERROR: configuration error: region is required (set AWS_REGION or --region)
```

**Cause:** The AWS region was not specified.

**Solution:**
```bash
# Option 1: Use environment variable
export AWS_REGION=us-east-1

# Option 2: Use command-line flag
sigv4-proxy --region us-east-1 ...
```

### Error: "service name is required"

**Symptoms:**
```
ERROR: configuration error: service name is required (set AWS_SERVICE_NAME or --service-name)
```

**Cause:** The AWS service name was not specified.

**Solution:**
```bash
# For API Gateway
export AWS_SERVICE_NAME=execute-api

# For Lambda Function URLs
export AWS_SERVICE_NAME=lambda

# For Application Load Balancer
export AWS_SERVICE_NAME=elasticloadbalancing
```

### Error: "target URL must use http or https scheme"

**Symptoms:**
```
ERROR: configuration error: target URL must use http or https scheme, got: ftp
```

**Cause:** The target URL uses an unsupported protocol.

**Solution:** Ensure the URL starts with `https://` (or `http://` for local testing):
```bash
# Correct
--target-url https://abc123.execute-api.us-east-1.amazonaws.com

# Incorrect
--target-url ftp://abc123.execute-api.us-east-1.amazonaws.com
```

### Error: "signature version must be 'v4' or 'v4a'"

**Symptoms:**
```
ERROR: unsupported signature version: v5 (must be 'v4' or 'v4a')
```

**Cause:** An invalid signature version was specified.

**Solution:**
```bash
# Use v4 (default, single-region)
--sig-version v4

# Use v4a (multi-region)
--sig-version v4a
```

## Credential Issues

### Error: "failed to load AWS credentials"

**Symptoms:**
```
ERROR: failed to load AWS credentials: ... (ensure AWS credentials are configured via environment variables, ~/.aws/credentials, or IAM role)
```

**Cause:** AWS credentials are not configured or not accessible.

**Solutions:**

#### 1. Check Environment Variables
```bash
# Verify credentials are set
echo $AWS_ACCESS_KEY_ID
echo $AWS_SECRET_ACCESS_KEY

# Set if missing
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

#### 2. Check Credentials File
```bash
# Verify file exists
ls -la ~/.aws/credentials

# Check file contents (should show profiles)
cat ~/.aws/credentials

# Create if missing
mkdir -p ~/.aws
cat > ~/.aws/credentials << EOF
[default]
aws_access_key_id = AKIAIOSFODNN7EXAMPLE
aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
EOF
```

#### 3. Verify Profile Name
```bash
# List available profiles
aws configure list-profiles

# Use correct profile
sigv4-proxy --profile your-profile-name ...
```

#### 4. Check IAM Role (EC2/ECS/Lambda)
```bash
# On EC2, verify instance profile
curl http://169.254.169.254/latest/meta-data/iam/security-credentials/

# Check if role is attached in AWS Console
# EC2: Instance → Security → IAM Role
# ECS: Task Definition → Task Role
```

### Error: "ExpiredToken" or "InvalidToken"

**Symptoms:**
```
ERROR: The security token included in the request is expired
```

**Cause:** Temporary credentials (session token) have expired.

**Solutions:**

#### For AWS SSO
```bash
# Re-login
aws sso login --profile your-profile

# Run proxy
sigv4-proxy --profile your-profile ...
```

#### For Assumed Roles
```bash
# Re-assume the role
aws sts assume-role \
  --role-arn arn:aws:iam::123456789012:role/MyRole \
  --role-session-name my-session

# Extract and set new credentials
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
export AWS_SESSION_TOKEN=...
```

#### For EC2 Instance Profiles
```bash
# Credentials should auto-refresh
# If not, check instance profile is attached
# Restart the proxy to pick up new credentials
```

### Error: "Access Denied" or "UnauthorizedException"

**Symptoms:**
```
ERROR: User: arn:aws:iam::123456789012:user/myuser is not authorized to perform: execute-api:Invoke
```

**Cause:** The IAM user/role doesn't have permission to invoke the target service.

**Solutions:**

#### 1. Verify IAM Permissions
Check that the IAM policy includes the necessary permissions:

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

#### 2. Check Resource ARN
Ensure the resource ARN in the policy matches your API:
```bash
# Get API ID from URL
# https://abc123.execute-api.us-east-1.amazonaws.com
# API ID: abc123

# Resource ARN format
arn:aws:execute-api:REGION:ACCOUNT_ID:API_ID/*
```

#### 3. Verify Caller Identity
```bash
# Check which credentials are being used
aws sts get-caller-identity

# Should match the expected user/role
```

#### 4. Check Resource Policy
If the target API has a resource policy, ensure it allows your IAM principal.

## Connection Issues

### Error: "dial tcp: lookup ... no such host"

**Symptoms:**
```
ERROR: proxy server error: dial tcp: lookup abc123.execute-api.us-east-1.amazonaws.com: no such host
```

**Cause:** DNS resolution failed for the target URL.

**Solutions:**

#### 1. Verify URL is Correct
```bash
# Test DNS resolution
nslookup abc123.execute-api.us-east-1.amazonaws.com

# Test with curl
curl -v https://abc123.execute-api.us-east-1.amazonaws.com
```

#### 2. Check Network Connectivity
```bash
# Test internet connectivity
ping 8.8.8.8

# Test DNS
ping google.com
```

#### 3. Check Firewall/Proxy Settings
```bash
# Check if corporate proxy is required
echo $HTTP_PROXY
echo $HTTPS_PROXY

# If proxy is required, configure it
export HTTPS_PROXY=http://proxy.example.com:8080
```

### Error: "connection refused"

**Symptoms:**
```
ERROR: proxy server error: dial tcp 127.0.0.1:8080: connect: connection refused
```

**Cause:** The target server is not running or not accessible.

**Solutions:**

#### 1. Verify Target Server is Running
```bash
# Test with curl
curl -v https://abc123.execute-api.us-east-1.amazonaws.com

# Check if port is open
telnet abc123.execute-api.us-east-1.amazonaws.com 443
```

#### 2. Check Security Groups (AWS)
- Ensure security groups allow outbound HTTPS (port 443)
- Ensure target server's security group allows inbound traffic

#### 3. Check Network ACLs (AWS)
- Verify NACLs allow traffic on port 443

### Error: "TLS handshake timeout"

**Symptoms:**
```
ERROR: proxy server error: net/http: TLS handshake timeout
```

**Cause:** Network latency or firewall blocking TLS connections.

**Solutions:**

#### 1. Check Network Latency
```bash
# Test latency
ping abc123.execute-api.us-east-1.amazonaws.com

# Test HTTPS connection
curl -v --max-time 10 https://abc123.execute-api.us-east-1.amazonaws.com
```

#### 2. Check Firewall Rules
- Ensure firewall allows outbound HTTPS (port 443)
- Check if TLS inspection is interfering

#### 3. Verify Certificates
```bash
# Check certificate
openssl s_client -connect abc123.execute-api.us-east-1.amazonaws.com:443
```

## Signing Issues

### Error: "SignatureDoesNotMatch"

**Symptoms:**
```
ERROR: The request signature we calculated does not match the signature you provided
```

**Cause:** The signature calculation is incorrect or credentials don't match.

**Solutions:**

#### 1. Verify Credentials
```bash
# Check which credentials are being used
aws sts get-caller-identity

# Ensure they match expected credentials
```

#### 2. Check Service Name
```bash
# For API Gateway, use execute-api
--service-name execute-api

# For Lambda Function URLs, use lambda
--service-name lambda
```

#### 3. Check Region
```bash
# Ensure region matches the target service
# Extract from URL: https://abc123.execute-api.us-east-1.amazonaws.com
--region us-east-1
```

#### 4. Check System Clock
```bash
# Signature includes timestamp - clock skew can cause failures
# Check system time
date

# Sync time (Linux)
sudo ntpdate -s time.nist.gov

# Sync time (macOS)
sudo sntp -sS time.apple.com
```

### Error: "InvalidSignatureException"

**Symptoms:**
```
ERROR: The request signature is invalid
```

**Cause:** Similar to SignatureDoesNotMatch - signature is malformed or incorrect.

**Solutions:**
- Follow the same steps as SignatureDoesNotMatch above
- Ensure you're using the correct signature version (v4 vs v4a)
- Verify the target service supports the signature version you're using

## Runtime Issues

### Proxy Hangs or Doesn't Respond

**Symptoms:** Proxy starts but doesn't process messages.

**Cause:** Waiting for input on stdin or target server not responding.

**Solutions:**

#### 1. Verify Proxy is Running
```bash
# Check logs for startup messages
# Should see: "Proxy is ready to accept MCP protocol messages"
```

#### 2. Test with Simple Message
```bash
# Send a test JSON-RPC message
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | sigv4-proxy ...
```

#### 3. Check Target Server
```bash
# Verify target server is responding
curl -v https://abc123.execute-api.us-east-1.amazonaws.com
```

### Proxy Exits Immediately

**Symptoms:** Proxy starts and exits without error.

**Cause:** No input on stdin or configuration error.

**Solutions:**

#### 1. Check for Error Messages
```bash
# Run with stderr redirected to see all messages
sigv4-proxy ... 2>&1 | tee proxy.log
```

#### 2. Verify Configuration
```bash
# Test configuration
sigv4-proxy \
  --target-url https://abc123.execute-api.us-east-1.amazonaws.com \
  --region us-east-1 \
  --service-name execute-api \
  2>&1
```

### High Memory Usage

**Symptoms:** Proxy consumes excessive memory.

**Cause:** Large messages or memory leak.

**Solutions:**

#### 1. Monitor Memory Usage
```bash
# Linux
top -p $(pgrep sigv4-proxy)

# macOS
top -pid $(pgrep sigv4-proxy)
```

#### 2. Check Message Sizes
- Large MCP messages may require more memory
- Consider if messages can be split or reduced

#### 3. Restart Proxy Periodically
- For long-running deployments, consider periodic restarts

## Debugging Tips

### Enable Verbose Logging

The proxy logs to stderr. Capture logs for debugging:

```bash
# Redirect stderr to file
sigv4-proxy ... 2> proxy.log

# View logs in real-time
sigv4-proxy ... 2>&1 | tee proxy.log
```

### Test AWS Credentials

Verify credentials work with AWS CLI:

```bash
# Test credentials
aws sts get-caller-identity

# Test with specific profile
aws sts get-caller-identity --profile your-profile

# Test API Gateway access
aws apigateway get-rest-apis
```

### Test Target Server

Test the target server directly:

```bash
# Test connectivity
curl -v https://abc123.execute-api.us-east-1.amazonaws.com

# Test with AWS signature (using awscurl)
pip install awscurl
awscurl --service execute-api \
  --region us-east-1 \
  https://abc123.execute-api.us-east-1.amazonaws.com
```

### Inspect Network Traffic

Use network tools to inspect traffic:

```bash
# Linux - tcpdump
sudo tcpdump -i any -n host abc123.execute-api.us-east-1.amazonaws.com

# macOS - tcpdump
sudo tcpdump -n host abc123.execute-api.us-east-1.amazonaws.com

# Wireshark
# Use Wireshark GUI to capture and analyze traffic
```

### Check Environment Variables

Verify all environment variables are set correctly:

```bash
# List all AWS-related variables
env | grep AWS

# List all MCP-related variables
env | grep MCP

# Check specific variables
echo "Target URL: $MCP_TARGET_URL"
echo "Region: $AWS_REGION"
echo "Service: $AWS_SERVICE_NAME"
echo "Profile: $AWS_PROFILE"
```

### Validate Configuration

Test configuration without running the proxy:

```bash
# Create a test script
cat > test-config.sh << 'EOF'
#!/bin/bash
echo "Configuration Test"
echo "=================="
echo "Target URL: ${MCP_TARGET_URL:-NOT SET}"
echo "Region: ${AWS_REGION:-NOT SET}"
echo "Service: ${AWS_SERVICE_NAME:-NOT SET}"
echo "Sig Version: ${AWS_SIG_VERSION:-v4 (default)}"
echo "Profile: ${AWS_PROFILE:-default}"
echo ""
echo "Testing AWS credentials..."
aws sts get-caller-identity
echo ""
echo "Testing target connectivity..."
curl -v --max-time 5 "${MCP_TARGET_URL}" 2>&1 | grep -E "(Connected|HTTP)"
EOF

chmod +x test-config.sh
./test-config.sh
```

### Common Diagnostic Commands

```bash
# Check Go version
go version

# Check proxy version
sigv4-proxy --help

# Test DNS resolution
nslookup abc123.execute-api.us-east-1.amazonaws.com

# Test network connectivity
ping -c 3 abc123.execute-api.us-east-1.amazonaws.com

# Test HTTPS connection
openssl s_client -connect abc123.execute-api.us-east-1.amazonaws.com:443

# Check system time (important for signatures)
date

# Check AWS CLI configuration
aws configure list

# List AWS profiles
aws configure list-profiles

# Test AWS credentials
aws sts get-caller-identity

# Check IAM permissions
aws iam get-user
aws iam list-attached-user-policies --user-name your-username
```

## Getting Help

If you're still experiencing issues:

1. **Check the logs**: Review stderr output for error messages
2. **Verify configuration**: Double-check all configuration parameters
3. **Test components**: Test credentials, network, and target server separately
4. **Review documentation**: Check [README](../README.md), [AWS Credentials Guide](aws-credentials.md), and [Examples](examples.md)
5. **Search issues**: Check the [GitHub repository](https://github.com/nisimpson/mcp-sigv4-proxy) for similar issues
6. **Create an issue**: If you've found a bug, create a detailed issue report with:
   - Proxy version
   - Configuration (redact credentials)
   - Error messages
   - Steps to reproduce
   - Environment details (OS, Go version, etc.)

## Additional Resources

- [AWS Credentials Setup Guide](aws-credentials.md)
- [Configuration Examples](examples.md)
- [Main README](../README.md)
- [AWS SigV4 Documentation](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html)
- [MCP Specification](https://modelcontextprotocol.io/)
