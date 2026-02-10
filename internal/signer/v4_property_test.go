package signer

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"pgregory.net/rapid"
)

// TestV4Signer_Property_AuthorizationHeaderPresence tests that for any HTTP request,
// the signed request contains an Authorization header with AWS4-HMAC-SHA256.
//
// **Validates: Requirements 2.1**
func TestV4Signer_Property_AuthorizationHeaderPresence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random HTTP request components
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}).Draw(t, "method")
		path := rapid.StringMatching(`/[a-zA-Z0-9/_-]*`).Draw(t, "path")
		url := "https://example.com" + path

		// Generate random body (can be empty)
		bodyContent := rapid.StringN(0, 1000, -1).Draw(t, "body")
		var body *strings.Reader
		if bodyContent != "" {
			body = strings.NewReader(bodyContent)
		} else {
			body = strings.NewReader("")
		}

		// Create HTTP request
		req, err := http.NewRequest(method, url, body)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Add some random headers
		headerCount := rapid.IntRange(0, 5).Draw(t, "headerCount")
		for i := 0; i < headerCount; i++ {
			headerName := rapid.StringMatching(`[A-Z][a-z-]+`).Draw(t, "headerName")
			headerValue := rapid.String().Draw(t, "headerValue")
			req.Header.Set(headerName, headerValue)
		}

		// Create V4Signer with valid credentials
		signer := &V4Signer{
			Credentials: aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			Region:  "us-east-1",
			Service: "execute-api",
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")
		if err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		// Property: Authorization header must be present
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Authorization header is missing after signing")
		}

		// Property: Authorization header must contain AWS4-HMAC-SHA256
		if !strings.Contains(authHeader, "AWS4-HMAC-SHA256") {
			t.Fatalf("Authorization header does not contain AWS4-HMAC-SHA256: %s", authHeader)
		}

		// Property: X-Amz-Date header must be present
		dateHeader := req.Header.Get("X-Amz-Date")
		if dateHeader == "" {
			t.Fatalf("X-Amz-Date header is missing after signing")
		}
	})
}

// TestV4Signer_Property_SessionTokenInclusion tests that for any credentials with
// a session token, the signed request includes the X-Amz-Security-Token header.
//
// **Validates: Requirements 5.3**
func TestV4Signer_Property_SessionTokenInclusion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random HTTP request
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
		path := rapid.StringMatching(`/[a-zA-Z0-9/_-]*`).Draw(t, "path")
		url := "https://example.com" + path

		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Generate random session token (non-empty)
		sessionToken := rapid.StringN(20, 200, -1).Draw(t, "sessionToken")

		// Create V4Signer with credentials that include a session token
		signer := &V4Signer{
			Credentials: aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    sessionToken,
			},
			Region:  "us-east-1",
			Service: "execute-api",
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")
		if err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		// Property: X-Amz-Security-Token header must be present
		tokenHeader := req.Header.Get("X-Amz-Security-Token")
		if tokenHeader == "" {
			t.Fatalf("X-Amz-Security-Token header is missing when credentials have session token")
		}

		// Property: The token value must match the credentials session token
		if tokenHeader != sessionToken {
			t.Fatalf("X-Amz-Security-Token header value does not match session token: got %s, want %s", tokenHeader, sessionToken)
		}
	})
}

// TestV4Signer_Property_CredentialScopeCorrectness tests that for any signed request,
// the credential scope in the Authorization header contains the configured service name and region.
//
// **Validates: Requirements 2.2**
func TestV4Signer_Property_CredentialScopeCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random region and service name
		region := rapid.SampledFrom([]string{
			"us-east-1", "us-west-2", "eu-west-1", "eu-central-1",
			"ap-southeast-1", "ap-northeast-1", "sa-east-1",
		}).Draw(t, "region")

		service := rapid.SampledFrom([]string{
			"execute-api", "lambda", "s3", "dynamodb", "ec2", "iam",
		}).Draw(t, "service")

		// Generate random HTTP request
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
		path := rapid.StringMatching(`/[a-zA-Z0-9/_-]*`).Draw(t, "path")
		url := "https://example.com" + path

		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		// Create V4Signer with the generated region and service
		signer := &V4Signer{
			Credentials: aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			Region:  region,
			Service: service,
		}

		// Sign the request
		ctx := context.Background()
		err = signer.SignRequest(ctx, req, "UNSIGNED-PAYLOAD")
		if err != nil {
			t.Fatalf("failed to sign request: %v", err)
		}

		// Property: Authorization header must contain the credential scope
		authHeader := req.Header.Get("Authorization")
		if authHeader == "" {
			t.Fatalf("Authorization header is missing")
		}

		// The credential scope format is: YYYYMMDD/region/service/aws4_request
		// We check that both region and service appear in the expected format
		expectedScope := region + "/" + service
		if !strings.Contains(authHeader, expectedScope) {
			t.Fatalf("Authorization header does not contain expected credential scope '%s': %s", expectedScope, authHeader)
		}

		// Property: The credential scope must also contain "aws4_request"
		if !strings.Contains(authHeader, "aws4_request") {
			t.Fatalf("Authorization header does not contain 'aws4_request': %s", authHeader)
		}
	})
}
