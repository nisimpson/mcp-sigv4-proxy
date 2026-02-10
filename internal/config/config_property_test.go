package config

import (
	"fmt"
	"testing"

	"pgregory.net/rapid"
)

// genValidURL generates valid HTTP/HTTPS URLs
func genValidURL() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		scheme := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "scheme")
		// Use simple sampled hosts instead of regex - much faster
		host := rapid.SampledFrom([]string{
			"example.com", "api.example.com", "localhost",
			"test.example.org", "mcp-server.aws.com",
		}).Draw(t, "host")

		// Sometimes include port, sometimes not
		if rapid.Bool().Draw(t, "includePort") {
			port := rapid.IntRange(1, 65535).Draw(t, "port")
			return fmt.Sprintf("%s://%s:%d", scheme, host, port)
		}
		return scheme + "://" + host
	})
}

// genValidRegion generates valid AWS region strings
func genValidRegion() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-central-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	})
}

// genValidServiceName generates valid AWS service names
func genValidServiceName() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"execute-api", "lambda", "s3", "dynamodb", "ec2", "ecs",
	})
}

// genValidSignatureVersion generates valid signature versions
func genValidSignatureVersion() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"v4", "v4a"})
}

// genValidProfile generates valid AWS profile names
func genValidProfile() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{
		"default", "dev", "prod", "staging", "test-profile", "my-profile",
	})
}

// TestProperty1_ValidConfigurationValidates tests that any valid configuration passes validation.
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5**
func TestProperty1_ValidConfigurationValidates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid configuration
		cfg := Config{
			TargetURL:        genValidURL().Draw(t, "targetURL"),
			Region:           genValidRegion().Draw(t, "region"),
			ServiceName:      genValidServiceName().Draw(t, "serviceName"),
			SignatureVersion: genValidSignatureVersion().Draw(t, "signatureVersion"),
			Profile:          genValidProfile().Draw(t, "profile"),
		}

		// Validation should succeed
		err := cfg.Validate()
		if err != nil {
			t.Fatalf("valid configuration failed validation: %v\nConfig: %+v", err, cfg)
		}
	})
}

// TestProperty2_MissingRequiredFieldsFailValidation tests that any configuration
// missing required fields fails validation with a descriptive error.
// **Validates: Requirements 6.6**
func TestProperty2_MissingRequiredFieldsFailValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a configuration with at least one missing required field
		missingField := rapid.SampledFrom([]string{"targetURL", "region", "serviceName"}).Draw(t, "missingField")

		cfg := Config{
			TargetURL:        genValidURL().Draw(t, "targetURL"),
			Region:           genValidRegion().Draw(t, "region"),
			ServiceName:      genValidServiceName().Draw(t, "serviceName"),
			SignatureVersion: genValidSignatureVersion().Draw(t, "signatureVersion"),
			Profile:          genValidProfile().Draw(t, "profile"),
		}

		// Clear the selected field
		switch missingField {
		case "targetURL":
			cfg.TargetURL = ""
		case "region":
			cfg.Region = ""
		case "serviceName":
			cfg.ServiceName = ""
		}

		// Validation should fail
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("configuration with missing %s should fail validation\nConfig: %+v", missingField, cfg)
		}

		// Error message should be descriptive - it should mention the missing field
		errMsg := err.Error()
		switch missingField {
		case "targetURL":
			// Error should mention "target URL" or "MCP_TARGET_URL" or "--target-url"
			if !containsAny(errMsg, []string{"target URL", "target url", "MCP_TARGET_URL", "--target-url"}) {
				t.Fatalf("error message should describe missing target URL, got: %s", errMsg)
			}
		case "region":
			// Error should mention "region" or "AWS_REGION" or "--region"
			if !containsAny(errMsg, []string{"region", "AWS_REGION", "--region"}) {
				t.Fatalf("error message should describe missing region, got: %s", errMsg)
			}
		case "serviceName":
			// Error should mention "service name" or "AWS_SERVICE_NAME" or "--service-name"
			if !containsAny(errMsg, []string{"service name", "service", "AWS_SERVICE_NAME", "--service-name"}) {
				t.Fatalf("error message should describe missing service name, got: %s", errMsg)
			}
		}
	})
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

// TestProperty_InvalidSignatureVersionFailsValidation tests that invalid signature versions fail validation.
// **Validates: Requirements 6.6**
func TestProperty_InvalidSignatureVersionFailsValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use sampled invalid versions instead of regex - much faster
		invalidSigVersion := rapid.SampledFrom([]string{
			"v3", "v5", "v2", "sigv4", "aws4", "invalid",
		}).Draw(t, "invalidSigVersion")

		cfg := Config{
			TargetURL:        genValidURL().Draw(t, "targetURL"),
			Region:           genValidRegion().Draw(t, "region"),
			ServiceName:      genValidServiceName().Draw(t, "serviceName"),
			SignatureVersion: invalidSigVersion,
			Profile:          genValidProfile().Draw(t, "profile"),
		}

		// Validation should fail
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("configuration with invalid signature version %q should fail validation", invalidSigVersion)
		}
	})
}

// TestProperty_InvalidURLSchemeFailsValidation tests that URLs with invalid schemes fail validation.
// **Validates: Requirements 6.6**
func TestProperty_InvalidURLSchemeFailsValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a URL with an invalid scheme (not http or https)
		invalidScheme := rapid.SampledFrom([]string{"ftp", "file", "ws", "wss", "ssh"}).Draw(t, "invalidScheme")
		// Use sampled hosts instead of regex - much faster
		host := rapid.SampledFrom([]string{
			"example.com", "test.org", "localhost", "api.example.com",
		}).Draw(t, "host")
		invalidURL := invalidScheme + "://" + host

		cfg := Config{
			TargetURL:        invalidURL,
			Region:           genValidRegion().Draw(t, "region"),
			ServiceName:      genValidServiceName().Draw(t, "serviceName"),
			SignatureVersion: genValidSignatureVersion().Draw(t, "signatureVersion"),
			Profile:          genValidProfile().Draw(t, "profile"),
		}

		// Validation should fail
		err := cfg.Validate()
		if err == nil {
			t.Fatalf("configuration with invalid URL scheme %q should fail validation", invalidURL)
		}
	})
}
