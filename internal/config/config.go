package config

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
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

// Load loads configuration from environment variables and command-line flags.
// Command-line flags take precedence over environment variables.
func Load() (*Config, error) {
	// First load from environment
	cfg, err := LoadFromEnv()
	if err != nil {
		// If validation fails, we still want to try flags
		// So create a new config with env values
		cfg = &Config{
			TargetURL:        os.Getenv("MCP_TARGET_URL"),
			Region:           os.Getenv("AWS_REGION"),
			ServiceName:      os.Getenv("AWS_SERVICE_NAME"),
			SignatureVersion: os.Getenv("AWS_SIG_VERSION"),
			Profile:          os.Getenv("AWS_PROFILE"),
		}
	}

	// Define and parse command-line flags
	targetURL := flag.String("target-url", "", "Target MCP server endpoint URL")
	region := flag.String("region", "", "AWS region for signing")
	serviceName := flag.String("service-name", "", "AWS service name for signing (e.g., execute-api)")
	sigVersion := flag.String("sig-version", "", "Signature version (v4 or v4a)")
	profile := flag.String("profile", "", "AWS credential profile name")

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
