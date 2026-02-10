package config

// Config holds proxy configuration
type Config struct {
	// TargetURL is the endpoint of the target MCP server
	TargetURL string

	// Region is the AWS region for signing
	Region string

	// ServiceName is the AWS service name for signing (e.g., "execute-api")
	ServiceName string

	// SignatureVersion is either "v4" or "v4a"
	SignatureVersion string

	// Profile is the AWS credential profile name (optional)
	Profile string
}
