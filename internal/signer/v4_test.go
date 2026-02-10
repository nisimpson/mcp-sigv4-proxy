package signer

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestV4Signer_SignRequest(t *testing.T) {
	tests := []struct {
		name        string
		signer      *V4Signer
		request     *http.Request
		payloadHash string
		wantErr     bool
		errContains string
		checkFunc   func(t *testing.T, req *http.Request)
	}{
		{
			name: "successfully signs request with valid credentials",
			signer: &V4Signer{
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
			wantErr:     false,
			checkFunc: func(t *testing.T, req *http.Request) {
				// Check that Authorization header is present
				authHeader := req.Header.Get("Authorization")
				assert.NotEmpty(t, authHeader, "Authorization header should be present")
				assert.Contains(t, authHeader, "AWS4-HMAC-SHA256", "Authorization header should contain AWS4-HMAC-SHA256")

				// Check that X-Amz-Date header is present
				dateHeader := req.Header.Get("X-Amz-Date")
				assert.NotEmpty(t, dateHeader, "X-Amz-Date header should be present")

				// Check that credential scope contains service and region
				assert.Contains(t, authHeader, "us-east-1/execute-api", "Authorization header should contain region and service")
			},
		},
		{
			name: "successfully signs request with session token",
			signer: &V4Signer{
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
			wantErr:     false,
			checkFunc: func(t *testing.T, req *http.Request) {
				// Check that X-Amz-Security-Token header is present
				tokenHeader := req.Header.Get("X-Amz-Security-Token")
				assert.NotEmpty(t, tokenHeader, "X-Amz-Security-Token header should be present")
				assert.Equal(t, "AQoDYXdzEJr...<remainder of session token>", tokenHeader)
			},
		},
		{
			name: "fails when region is missing",
			signer: &V4Signer{
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
			name: "fails when service name is missing",
			signer: &V4Signer{
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
			name: "fails when credentials are missing",
			signer: &V4Signer{
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
			name: "includes service and region in credential scope",
			signer: &V4Signer{
				Credentials: aws.Credentials{
					AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
				Region:  "eu-west-1",
				Service: "lambda",
			},
			request: func() *http.Request {
				req, _ := http.NewRequest("POST", "https://example.com/api", nil)
				return req
			}(),
			payloadHash: "UNSIGNED-PAYLOAD",
			wantErr:     false,
			checkFunc: func(t *testing.T, req *http.Request) {
				authHeader := req.Header.Get("Authorization")
				// Credential scope format: YYYYMMDD/region/service/aws4_request
				assert.Contains(t, authHeader, "eu-west-1/lambda", "Authorization header should contain configured region and service")
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
			} else {
				require.NoError(t, err)
				if tt.checkFunc != nil {
					tt.checkFunc(t, tt.request)
				}
			}
		})
	}
}
