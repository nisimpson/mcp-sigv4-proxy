package signer

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// V4aSigner implements SigV4a signing for HTTP requests.
// It uses the AWS SDK v4a signer to add authentication headers to requests
// with support for multi-region signing.
//
// IMPORTANT LIMITATION: The AWS SDK for Go v2 currently keeps the v4a signer
// in an internal package (github.com/aws/aws-sdk-go-v2/internal/v4a), which
// cannot be imported due to Go's internal package restrictions.
//
// This implementation provides the struct and interface but returns an error
// indicating that v4a signing is not yet available. Once AWS makes the v4a
// signer public, this implementation should be updated to use the public API.
//
// Tracking issue: https://github.com/aws/aws-sdk-go-v2/issues/1935
type V4aSigner struct {
	// Credentials are the AWS credentials used for signing
	Credentials aws.Credentials

	// Region is the AWS region for the signature (e.g., "us-east-1")
	// For multi-region signing, this is used as the primary region
	Region string

	// Service is the AWS service name for the signature (e.g., "execute-api")
	Service string
}

// ErrV4aNotAvailable is returned when attempting to use SigV4a signing,
// which is not yet publicly available in the AWS SDK for Go v2.
var ErrV4aNotAvailable = errors.New("SigV4a signing is not available: AWS SDK v2 keeps v4a signer in internal package")

// SignRequest adds AWS SigV4a signature headers to the HTTP request.
// It signs the request using the configured credentials, region, and service name
// with support for multi-region signing.
//
// The payloadHash parameter should be the SHA256 hash of the request body,
// or "UNSIGNED-PAYLOAD" if the payload should not be signed.
//
// After signing, the request will contain:
// - Authorization header with the AWS4-ECDSA-P256-SHA256 signature
// - X-Amz-Date header with the signing timestamp
// - X-Amz-Security-Token header (if credentials include a session token)
// - X-Amz-Region-Set header with the region set for multi-region signing
//
// CURRENT STATUS: This method currently returns ErrV4aNotAvailable because
// the AWS SDK v2 does not expose the v4a signer publicly. Use V4Signer for
// single-region signing instead.
func (s *V4aSigner) SignRequest(ctx context.Context, req *http.Request, payloadHash string) error {
	// Validate that we have the required configuration
	if s.Region == "" {
		return fmt.Errorf("region is required for SigV4a signing")
	}
	if s.Service == "" {
		return fmt.Errorf("service name is required for SigV4a signing")
	}
	if s.Credentials.AccessKeyID == "" || s.Credentials.SecretAccessKey == "" {
		return fmt.Errorf("AWS credentials are required for SigV4a signing")
	}

	// Return error indicating v4a is not available
	// Once AWS makes the v4a signer public, this should be replaced with actual signing logic
	return fmt.Errorf("%w: see https://github.com/aws/aws-sdk-go-v2/issues/1935 for status", ErrV4aNotAvailable)
}
