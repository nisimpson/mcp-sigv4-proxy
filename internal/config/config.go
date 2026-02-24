package config

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"
)

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

	// Comma delimited list of headers
	Headers string

	// Timeout is the request timeout duration for HTTP requests to the target server
	Timeout time.Duration

	// EnableSSE enables Server-Sent Events for streaming responses
	EnableSSE bool
}

// LoadFromEnv loads configuration from environment variables only.
// This is useful for testing and for environments where flags aren't used.
func LoadFromEnv() (*Config, error) {
	cfg := &Config{
		TargetURL:        os.Getenv("MCP_TARGET_URL"),
		Region:           os.Getenv("AWS_REGION"),
		ServiceName:      os.Getenv("AWS_SERVICE_NAME"),
		SignatureVersion: os.Getenv("AWS_SIG_VERSION"),
		Profile:          os.Getenv("AWS_PROFILE"),
		EnableSSE:        getBoolEnv("MCP_ENABLE_SSE"),
		Timeout:          getDurationEnv("MCP_TIMEOUT"),
		Headers:          os.Getenv("MCP_HEADERS"),
	}

	// Set default signature version if not specified
	if cfg.SignatureVersion == "" {
		cfg.SignatureVersion = "v4"
	}

	// Set default profile if not specified
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func getBoolEnv(key string) bool {
	value := os.Getenv(key)
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return false
	}
	return boolValue
}

func getDurationEnv(key string) time.Duration {
	value := os.Getenv(key)
	durationValue, err := time.ParseDuration(value)
	if err != nil {
		return 0
	}
	return durationValue
}

// Load loads configuration from environment variables and command-line flags.
// Command-line flags take precedence over environment variables.
func Load(logger *log.Logger) (*Config, error) {
	// First load from environment
	cfg, err := LoadFromEnv()
	if err != nil {
		logger.Printf("load environment variables failed")
	}

	// Define and parse command-line flags
	targetURL := flag.String("target-url", "", "Target MCP server endpoint URL")
	region := flag.String("region", "", "AWS region for signing")
	serviceName := flag.String("service-name", "", "AWS service name for signing (e.g., execute-api)")
	sigVersion := flag.String("sig-version", "", "Signature version (v4 or v4a)")
	profile := flag.String("profile", "", "AWS credential profile name")
	enableSSE := flag.Bool("sse", false, "enable server-side events")
	timeout := flag.Duration("timeout", 0, "mcp client timeout (default no timeout)")
	headers := flag.String("headers", "", "comma delimited list of headers (key=value)")

	flag.Parse()

	// Override with command-line flags if provided
	if *targetURL != "" {
		cfg.TargetURL = *targetURL
	}
	if *region != "" {
		cfg.Region = *region
	}
	if *serviceName != "" {
		cfg.ServiceName = *serviceName
	}
	if *sigVersion != "" {
		cfg.SignatureVersion = *sigVersion
	}
	if *profile != "" {
		cfg.Profile = *profile
	}
	if *enableSSE {
		cfg.EnableSSE = *enableSSE
	}
	if *timeout > 0 {
		cfg.Timeout = *timeout
	}
	if *headers != "" {
		cfg.Headers = *headers
	}

	// Set default signature version if not specified
	if cfg.SignatureVersion == "" {
		cfg.SignatureVersion = "v4"
	}

	// Set default profile if not specified
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration fields are present and valid.
func (c *Config) Validate() error {
	var errs []error

	// Check required fields
	if c.TargetURL == "" {
		errs = append(errs, errors.New("target URL is required (set MCP_TARGET_URL or --target-url)"))
	} else {
		// Validate URL format
		parsedURL, err := url.Parse(c.TargetURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid target URL: %w", err))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			errs = append(errs, fmt.Errorf("target URL must use http or https scheme, got: %s", parsedURL.Scheme))
		}
	}

	if c.Region == "" {
		errs = append(errs, errors.New("region is required (set AWS_REGION or --region)"))
	}

	if c.ServiceName == "" {
		errs = append(errs, errors.New("service name is required (set AWS_SERVICE_NAME or --service-name)"))
	}

	// Validate signature version
	if c.SignatureVersion != "v4" && c.SignatureVersion != "v4a" {
		errs = append(errs, fmt.Errorf("signature version must be 'v4' or 'v4a', got: %s", c.SignatureVersion))
	}

	// Combine all errors
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
