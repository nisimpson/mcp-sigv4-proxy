package credentials

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// Provider loads AWS credentials using the SDK's default credential chain.
// It supports environment variables, shared config files, IAM roles, and profiles.
type Provider struct {
	// Profile is the AWS credential profile name to use (optional)
	Profile string

	// Region is the AWS region (optional, can be loaded from config)
	Region string
}

// LoadCredentials loads AWS credentials using the default credential chain.
// The credential chain includes (in order):
// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN)
// 2. Shared credentials file (~/.aws/credentials)
// 3. Shared config file (~/.aws/config)
// 4. IAM role for EC2 instances
// 5. IAM role for ECS tasks
//
// If a profile is specified, credentials are loaded from that profile.
// Session tokens are automatically included if present in the credentials.
func (p *Provider) LoadCredentials(ctx context.Context) (aws.Credentials, error) {
	// Build config options
	var opts []func(*config.LoadOptions) error

	// Add profile if specified
	if p.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(p.Profile))
	}

	// Add region if specified
	if p.Region != "" {
		opts = append(opts, config.WithRegion(p.Region))
	}

	// Load AWS config using the default credential chain
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Retrieve credentials
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return aws.Credentials{}, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	// Validate that we have credentials
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return aws.Credentials{}, fmt.Errorf("AWS credentials are incomplete: missing access key or secret key")
	}

	return creds, nil
}

// LoadConfig loads the full AWS config including credentials.
// This is useful when you need both credentials and other AWS configuration.
func (p *Provider) LoadConfig(ctx context.Context) (aws.Config, error) {
	// Build config options
	var opts []func(*config.LoadOptions) error

	// Add profile if specified
	if p.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(p.Profile))
	}

	// Add region if specified
	if p.Region != "" {
		opts = append(opts, config.WithRegion(p.Region))
	}

	// Load AWS config using the default credential chain
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Validate credentials
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to retrieve AWS credentials: %w", err)
	}

	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return aws.Config{}, fmt.Errorf("AWS credentials are incomplete: missing access key or secret key")
	}

	return cfg, nil
}
