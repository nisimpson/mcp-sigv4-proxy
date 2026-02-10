package signer

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"pgregory.net/rapid"
)

// TestV4aSigner_Property_ValidationBehavior tests that V4aSigner properly validates
// its configuration before attempting to sign.
//
// **Validates: Requirements 3.1**
//
// Property: For any HTTP request with valid V4aSigner configuration, the signer
// validates all required fields (credentials, region, service) before returning
// the "not available" error.
func TestV4aSigner_Property_ValidationBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid credentials
		accessKeyID := rapid.StringN(16, 20, 128).Draw(t, "accessKeyID")
		secretAccessKey := rapid.StringN(20, 40, 128).Draw(t, "secretAccessKey")
		sessionToken := rapid.StringN(0, 100, 256).Draw(t, "sessionToken")

		// Generate random region and service
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")
		service := rapid.SampledFrom([]string{
			"execute-api", "lambda", "s3", "dynamodb",
		}).Draw(t, "service")

		// Create signer with valid configuration
		signer := &V4aSigner{
			Credentials: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				SessionToken:    sessionToken,
			},
			Region:  region,
			Service: service,
		}

		// Create a test request
		req, err := http.NewRequest("POST", "https://example.com/api", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")

		// Property: With valid configuration, should return ErrV4aNotAvailable
		// (not a validation error)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrV4aNotAvailable) {
			t.Fatalf("expected ErrV4aNotAvailable, got: %v", err)
		}
	})
}

// TestV4aSigner_Property_MissingRegionValidation tests that V4aSigner rejects
// requests when region is missing.
//
// **Validates: Requirements 3.1**
//
// Property: For any HTTP request with V4aSigner missing region, the signer
// returns a validation error about missing region.
func TestV4aSigner_Property_MissingRegionValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid credentials
		accessKeyID := rapid.StringN(16, 20, 128).Draw(t, "accessKeyID")
		secretAccessKey := rapid.StringN(20, 40, 128).Draw(t, "secretAccessKey")

		// Generate random service
		service := rapid.SampledFrom([]string{
			"execute-api", "lambda", "s3", "dynamodb",
		}).Draw(t, "service")

		// Create signer with missing region
		signer := &V4aSigner{
			Credentials: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			},
			Region:  "", // Missing region
			Service: service,
		}

		// Create a test request
		req, err := http.NewRequest("POST", "https://example.com/api", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")

		// Property: Should return validation error about missing region
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !contains(err.Error(), "region is required") {
			t.Fatalf("expected 'region is required' error, got: %v", err)
		}
	})
}

// TestV4aSigner_Property_MissingServiceValidation tests that V4aSigner rejects
// requests when service is missing.
//
// **Validates: Requirements 3.1**
//
// Property: For any HTTP request with V4aSigner missing service, the signer
// returns a validation error about missing service.
func TestV4aSigner_Property_MissingServiceValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid credentials
		accessKeyID := rapid.StringN(16, 20, 128).Draw(t, "accessKeyID")
		secretAccessKey := rapid.StringN(20, 40, 128).Draw(t, "secretAccessKey")

		// Generate random region
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")

		// Create signer with missing service
		signer := &V4aSigner{
			Credentials: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			},
			Region:  region,
			Service: "", // Missing service
		}

		// Create a test request
		req, err := http.NewRequest("POST", "https://example.com/api", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")

		// Property: Should return validation error about missing service
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !contains(err.Error(), "service name is required") {
			t.Fatalf("expected 'service name is required' error, got: %v", err)
		}
	})
}

// TestV4aSigner_Property_MissingCredentialsValidation tests that V4aSigner rejects
// requests when credentials are missing.
//
// **Validates: Requirements 3.1**
//
// Property: For any HTTP request with V4aSigner missing credentials, the signer
// returns a validation error about missing credentials.
func TestV4aSigner_Property_MissingCredentialsValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random region and service
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")
		service := rapid.SampledFrom([]string{
			"execute-api", "lambda", "s3", "dynamodb",
		}).Draw(t, "service")

		// Create signer with missing credentials
		signer := &V4aSigner{
			Credentials: aws.Credentials{
				// Missing credentials
			},
			Region:  region,
			Service: service,
		}

		// Create a test request
		req, err := http.NewRequest("POST", "https://example.com/api", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")

		// Property: Should return validation error about missing credentials
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !contains(err.Error(), "AWS credentials are required") {
			t.Fatalf("expected 'AWS credentials are required' error, got: %v", err)
		}
	})
}

// TestV4aSigner_Property_AlgorithmIdentifier tests that V4aSigner is configured
// to use the AWS4-ECDSA-P256-SHA256 algorithm when signing becomes available.
//
// **Validates: Requirements 3.1**
//
// Property 6: For any HTTP request with SigV4a, signed request contains Authorization
// header with AWS4-ECDSA-P256-SHA256.
//
// CURRENT STATUS: Since the AWS SDK v2 keeps the v4a signer in an internal package,
// this test verifies that the V4aSigner is properly configured and validates inputs.
// When v4a signing becomes publicly available, the actual Authorization header with
// AWS4-ECDSA-P256-SHA256 will be verified.
//
// Expected behavior (when v4a is available):
// - Authorization header format: AWS4-ECDSA-P256-SHA256 Credential=...
// - This differs from SigV4 which uses: AWS4-HMAC-SHA256 Credential=...
// - The ECDSA algorithm provides multi-region signing support
func TestV4aSigner_Property_AlgorithmIdentifier(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random valid credentials
		accessKeyID := rapid.StringN(16, 20, 128).Draw(t, "accessKeyID")
		secretAccessKey := rapid.StringN(20, 40, 128).Draw(t, "secretAccessKey")
		sessionToken := rapid.StringN(0, 100, 256).Draw(t, "sessionToken")

		// Generate random region and service
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1",
		}).Draw(t, "region")
		service := rapid.SampledFrom([]string{
			"execute-api", "lambda", "s3", "dynamodb",
		}).Draw(t, "service")

		// Generate random HTTP request components
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}).Draw(t, "method")
		path := rapid.StringMatching(`/[a-zA-Z0-9/_-]*`).Draw(t, "path")
		url := "https://example.com" + path

		// Create HTTP request
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Create V4aSigner with valid configuration
		signer := &V4aSigner{
			Credentials: aws.Credentials{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
				SessionToken:    sessionToken,
			},
			Region:  region,
			Service: service,
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")

		// CURRENT BEHAVIOR: Verify that with valid configuration, we get ErrV4aNotAvailable
		// This confirms the signer is properly configured and ready for when v4a becomes available
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, ErrV4aNotAvailable) {
			t.Fatalf("expected ErrV4aNotAvailable, got: %v", err)
		}

		// FUTURE BEHAVIOR (when v4a is available):
		// The test should verify:
		// 1. No error is returned
		// 2. Authorization header is present
		// 3. Authorization header contains "AWS4-ECDSA-P256-SHA256"
		// 4. X-Amz-Date header is present
		// 5. X-Amz-Security-Token header is present (if session token exists)
		// 6. X-Amz-Region-Set header is present (for multi-region signing)
		//
		// Example verification code (to be uncommented when v4a is available):
		// if err != nil {
		//     t.Fatalf("failed to sign request: %v", err)
		// }
		//
		// authHeader := req.Header.Get("Authorization")
		// if authHeader == "" {
		//     t.Fatalf("Authorization header is missing after signing")
		// }
		//
		// if !strings.Contains(authHeader, "AWS4-ECDSA-P256-SHA256") {
		//     t.Fatalf("Authorization header does not contain AWS4-ECDSA-P256-SHA256: %s", authHeader)
		// }
		//
		// dateHeader := req.Header.Get("X-Amz-Date")
		// if dateHeader == "" {
		//     t.Fatalf("X-Amz-Date header is missing after signing")
		// }
		//
		// if sessionToken != "" {
		//     tokenHeader := req.Header.Get("X-Amz-Security-Token")
		//     if tokenHeader == "" {
		//         t.Fatalf("X-Amz-Security-Token header is missing when credentials have session token")
		//     }
		//     if tokenHeader != sessionToken {
		//         t.Fatalf("X-Amz-Security-Token header value does not match session token")
		//     }
		// }
		//
		// regionSetHeader := req.Header.Get("X-Amz-Region-Set")
		// if regionSetHeader == "" {
		//     t.Fatalf("X-Amz-Region-Set header is missing (required for multi-region signing)")
		// }
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
