package signer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

// V4Signer implements SigV4 signing for HTTP requests.
// It uses the AWS SDK v4 signer to add authentication headers to requests.
type V4Signer struct {
	// Credentials are the AWS credentials used for signing
	Credentials aws.Credentials

	// Region is the AWS region for the signature (e.g., "us-east-1")
	Region string

	// Service is the AWS service name for the signature (e.g., "execute-api")
	Service string
}

// SignRequest adds AWS SigV4 signature headers to the HTTP request.
// It signs the request using the configured credentials, region, and service name.
//
// The payloadHash parameter should be the SHA256 hash of the request body,
// or "UNSIGNED-PAYLOAD" if the payload should not be signed.
//
// After signing, the request will contain:
// - Authorization header with the AWS4-HMAC-SHA256 signature
// - X-Amz-Date header with the signing timestamp
// - X-Amz-Security-Token header (if credentials include a session token)
func (s *V4Signer) SignRequest(ctx context.Context, req *http.Request, payloadHash string) error {
	// Validate that we have the required configuration
	if s.Region == "" {
		return fmt.Errorf("region is required for SigV4 signing")
	}
	if s.Service == "" {
		return fmt.Errorf("service name is required for SigV4 signing")
	}
	if s.Credentials.AccessKeyID == "" || s.Credentials.SecretAccessKey == "" {
		return fmt.Errorf("AWS credentials are required for SigV4 signing")
	}

	// Create the v4 signer
	signer := v4.NewSigner()

	// Sign the request
	// The signer will add the Authorization, X-Amz-Date, and X-Amz-Security-Token headers
	err := signer.SignHTTP(ctx, s.Credentials, req, payloadHash, s.Service, s.Region, time.Now())
	if err != nil {
		return fmt.Errorf("failed to sign request with SigV4: %w", err)
	}

	return nil
}
