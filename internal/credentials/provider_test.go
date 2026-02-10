package credentials

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_LoadCredentials_FromEnvironment(t *testing.T) {
	// Set up environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	os.Setenv("AWS_SESSION_TOKEN", "test-session-token")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_SESSION_TOKEN")
	}()

	provider := &Provider{
		Region: "us-east-1",
	}

	creds, err := provider.LoadCredentials(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "test-access-key", creds.AccessKeyID)
	assert.Equal(t, "test-secret-key", creds.SecretAccessKey)
	assert.Equal(t, "test-session-token", creds.SessionToken)
}

func TestProvider_LoadCredentials_WithoutSessionToken(t *testing.T) {
	// Set up environment variables without session token
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	}()

	provider := &Provider{
		Region: "us-east-1",
	}

	creds, err := provider.LoadCredentials(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "test-access-key", creds.AccessKeyID)
	assert.Equal(t, "test-secret-key", creds.SecretAccessKey)
	assert.Empty(t, creds.SessionToken)
}

func TestProvider_LoadCredentials_MissingCredentials(t *testing.T) {
	// Clear all AWS environment variables
	os.Unsetenv("AWS_ACCESS_KEY_ID")
	os.Unsetenv("AWS_SECRET_ACCESS_KEY")
	os.Unsetenv("AWS_SESSION_TOKEN")
	os.Unsetenv("AWS_PROFILE")

	provider := &Provider{
		Region: "us-east-1",
	}

	_, err := provider.LoadCredentials(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to retrieve AWS credentials")
}

func TestProvider_LoadConfig_FromEnvironment(t *testing.T) {
	// Set up environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	os.Setenv("AWS_REGION", "us-west-2")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	}()

	provider := &Provider{}

	cfg, err := provider.LoadConfig(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "us-west-2", cfg.Region)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test-access-key", creds.AccessKeyID)
	assert.Equal(t, "test-secret-key", creds.SecretAccessKey)
}

func TestProvider_LoadConfig_WithRegionOverride(t *testing.T) {
	// Set up environment variables
	os.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	os.Setenv("AWS_REGION", "us-west-2")
	defer func() {
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_REGION")
	}()

	provider := &Provider{
		Region: "eu-west-1", // Override with provider region
	}

	cfg, err := provider.LoadConfig(context.Background())
	require.NoError(t, err)

	// Provider region should take precedence
	assert.Equal(t, "eu-west-1", cfg.Region)
}
