package signer

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV4aSigner_SignRequest(t *testing.T) {
	tests := []struct {
		name        string
		signer      *V4aSigner
		request     *http.Request
		payloadHash string
		wantErr     bool
		errContains string
		checkFunc   func(t *testing.T, err error)
	}{
		{
			name: "returns not available error with valid credentials",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Region:  "us-east-1",
				Service: "execute-api",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", strings.NewReader("test body"))
				req.Header.Set("Content-Type", "application/json")
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "SigV4a signing is not available",
			checkFunc: func(t *testing.T, err error) {
				// Verify it returns the specific ErrV4aNotAvailable error
				assert.True(t, errors.Is(err, ErrV4aNotAvailable), "Error should be ErrV4aNotAvailable")
			},
		},
		{
			name: "validates region is required",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Region:  "", // Missing region
				Service: "execute-api",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "region is required",
		},
		{
			name: "validates service name is required",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Region:  "us-east-1",
				Service: "", // Missing service
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "service name is required",
		},
		{
			name: "validates credentials are required",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					// Missing credentials
				},
				Region:  "us-east-1",
				Service: "execute-api",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "AWS credentials are required",
		},
		{
			name: "struct supports session token field",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
					SessionToken:    "AQoDYXdzEJr...<remainder of session token>",
				},
				Region:  "us-west-2",
				Service: "execute-api",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/api", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "SigV4a signing is not available",
			checkFunc: func(t *testing.T, err error) {
				// Verify the struct accepts session tokens (even though signing isn't available yet)
				assert.True(t, errors.Is(err, ErrV4aNotAvailable), "Error should be ErrV4aNotAvailable")
			},
		},
		{
			name: "struct supports multi-region configuration",
			signer: &V4aSigner{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Region:  "us-east-1",
				Service: "s3",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("GET", "https://example.com/bucket/key", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     true,
			errContains: "SigV4a signing is not available",
			checkFunc: func(t *testing.T, err error) {
				// Verify the struct is ready for multi-region signing once available
				assert.True(t, errors.Is(err, ErrV4aNotAvailable), "Error should be ErrV4aNotAvailable")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.signer.SignRequest(ctx, tt.request, tt.payloadHash)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				if tt.checkFunc != nil {
					tt.checkFunc(t, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestV4aSigner_Interface verifies that V4aSigner implements the Signer interface
func TestV4aSigner_Interface(t *testing.T) {
	var _ Signer = (*V4aSigner)(nil)
}

// TestV4aSigner_StructureForFutureImplementation verifies the struct has all
// necessary fields for when v4a becomes publicly available
func TestV4aSigner_StructureForFutureImplementation(t *testing.T) {
	signer := &V4aSigner{
		Credentials: aws.Credentials{
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			SessionToken:    "session-token",
		},
		Region:  "us-east-1",
		Service: "execute-api",
	}

	// Verify all fields are accessible and properly typed
	assert.Equal(t, "AKIAIOSFODNN7EXAMPLE", signer.Credentials.AccessKeyID)
	assert.Equal(t, "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", signer.Credentials.SecretAccessKey)
	assert.Equal(t, "session-token", signer.Credentials.SessionToken)
	assert.Equal(t, "us-east-1", signer.Region)
	assert.Equal(t, "execute-api", signer.Service)
}
