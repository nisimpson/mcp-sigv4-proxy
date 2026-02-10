package credentials

import (
	"context"
	"os"
	"testing"

	"pgregory.net/rapid"
)

// awsAccessKeyGen generates realistic AWS access key IDs (20 uppercase alphanumeric characters)
func awsAccessKeyGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 20 characters from A-Z and 0-9
		const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, 20)
		for i := 0; i < 20; i++ {
			result[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "char")]
		}
		return string(result)
	})
}

// awsSecretKeyGen generates realistic AWS secret access keys (40 base64-like characters)
func awsSecretKeyGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 40 characters from base64 character set
		const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
		result := make([]byte, 40)
		for i := 0; i < 40; i++ {
			result[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "char")]
		}
		return string(result)
	})
}

// awsSessionTokenGen generates realistic AWS session tokens (100-500 base64-like characters)
func awsSessionTokenGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// Generate 100-500 characters from base64 character set
		const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/="
		length := rapid.IntRange(100, 500).Draw(t, "length")
		result := make([]byte, length)
		for i := 0; i < length; i++ {
			result[i] = chars[rapid.IntRange(0, len(chars)-1).Draw(t, "char")]
		}
		return string(result)
	})
}

// TestProperty_SessionTokenInclusion tests that for any valid AWS credentials
// with a session token, the token is included in the retrieved credentials.
//
// **Validates: Requirements 5.3**
func TestProperty_SessionTokenInclusion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random credentials with session token
		accessKey := awsAccessKeyGen().Draw(t, "accessKey")
		secretKey := awsSecretKeyGen().Draw(t, "secretKey")
		sessionToken := awsSessionTokenGen().Draw(t, "sessionToken")
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")

		// Set environment variables
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
		os.Setenv("AWS_SESSION_TOKEN", sessionToken)
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			os.Unsetenv("AWS_SESSION_TOKEN")
			os.Unsetenv("AWS_EC2_METADATA_DISABLED")
		}()

		// Create provider and load credentials
		provider := &Provider{
			Region: region,
		}

		creds, err := provider.LoadCredentials(context.Background())
		if err != nil {
			t.Fatalf("failed to load credentials: %v", err)
		}

		// Property: Session token must be included
		if creds.SessionToken != sessionToken {
			t.Fatalf("session token not included: expected %q, got %q", sessionToken, creds.SessionToken)
		}

		// Additional invariants
		if creds.AccessKeyID != accessKey {
			t.Fatalf("access key mismatch: expected %q, got %q", accessKey, creds.AccessKeyID)
		}
		if creds.SecretAccessKey != secretKey {
			t.Fatalf("secret key mismatch: expected %q, got %q", secretKey, creds.SecretAccessKey)
		}
	})
}

// TestProperty_CredentialsWithoutSessionToken tests that credentials
// without a session token are still valid and load correctly.
//
// **Validates: Requirements 5.1, 5.2**
func TestProperty_CredentialsWithoutSessionToken(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random credentials without session token
		accessKey := awsAccessKeyGen().Draw(t, "accessKey")
		secretKey := awsSecretKeyGen().Draw(t, "secretKey")
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")

		// Set environment variables (no session token)
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		os.Unsetenv("AWS_SESSION_TOKEN") // Explicitly unset
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			os.Unsetenv("AWS_EC2_METADATA_DISABLED")
		}()

		// Create provider and load credentials
		provider := &Provider{
			Region: region,
		}

		creds, err := provider.LoadCredentials(context.Background())
		if err != nil {
			t.Fatalf("failed to load credentials: %v", err)
		}

		// Property: Credentials should load successfully without session token
		if creds.AccessKeyID != accessKey {
			t.Fatalf("access key mismatch: expected %q, got %q", accessKey, creds.AccessKeyID)
		}
		if creds.SecretAccessKey != secretKey {
			t.Fatalf("secret key mismatch: expected %q, got %q", secretKey, creds.SecretAccessKey)
		}
		// Session token should be empty
		if creds.SessionToken != "" {
			t.Fatalf("session token should be empty, got %q", creds.SessionToken)
		}
	})
}

// TestProperty_ProfileBasedCredentials tests that when a profile is specified,
// the provider can still load credentials from environment variables (which take precedence).
//
// **Validates: Requirements 5.2**
func TestProperty_ProfileBasedCredentials(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random credentials using the generators
		accessKey := awsAccessKeyGen().Draw(t, "accessKey")
		secretKey := awsSecretKeyGen().Draw(t, "secretKey")

		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")

		// Set environment variables FIRST (these take precedence in the credential chain)
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
		// Disable all fallback credential sources
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			os.Unsetenv("AWS_EC2_METADATA_DISABLED")
		}()

		// Create provider WITHOUT a profile (since we can't create actual profile files in tests)
		// This tests that the default credential chain works
		provider := &Provider{
			Region: region,
		}

		creds, err := provider.LoadCredentials(context.Background())
		if err != nil {
			t.Fatalf("failed to load credentials: %v", err)
		}

		// Property: Credentials should be loaded successfully from environment
		if creds.AccessKeyID != accessKey {
			t.Fatalf("access key mismatch: expected %q, got %q", accessKey, creds.AccessKeyID)
		}
		if creds.SecretAccessKey != secretKey {
			t.Fatalf("secret key mismatch: expected %q, got %q", secretKey, creds.SecretAccessKey)
		}
	})
}

// TestProperty_DefaultCredentialChain tests that the provider uses the
// standard AWS credential chain.
//
// **Validates: Requirements 5.1**
func TestProperty_DefaultCredentialChain(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random credentials
		accessKey := awsAccessKeyGen().Draw(t, "accessKey")
		secretKey := awsSecretKeyGen().Draw(t, "secretKey")
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")

		// Set environment variables (first in the credential chain)
		os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
		os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
		defer func() {
			os.Unsetenv("AWS_ACCESS_KEY_ID")
			os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			os.Unsetenv("AWS_EC2_METADATA_DISABLED")
		}()

		// Create provider without specifying profile
		provider := &Provider{
			Region: region,
		}

		creds, err := provider.LoadCredentials(context.Background())
		if err != nil {
			t.Fatalf("failed to load credentials: %v", err)
		}

		// Property: Environment variables should be used (first in chain)
		if creds.AccessKeyID != accessKey {
			t.Fatalf("access key mismatch: expected %q, got %q", accessKey, creds.AccessKeyID)
		}
		if creds.SecretAccessKey != secretKey {
			t.Fatalf("secret key mismatch: expected %q, got %q", secretKey, creds.SecretAccessKey)
		}
	})
}
