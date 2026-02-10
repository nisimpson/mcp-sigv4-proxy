package proxy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nisimpson/mcp-sigv4-proxy/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorHandling_UnreachableTarget tests that network errors are handled
// with descriptive messages when the target server is unreachable.
//
// **Validates: Requirement 7.1**
func TestErrorHandling_UnreachableTarget(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		wantErr   string
	}{
		{
			name:      "invalid host",
			targetURL: "https://invalid-host-that-does-not-exist-12345.com",
			wantErr:   "failed to connect to target MCP server",
		},
		{
			name:      "connection refused",
			targetURL: "http://localhost:9999", // Assuming this port is not in use
			wantErr:   "failed to connect to target MCP server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock signer
			signer := &mockErrorSigner{}

			// Create the signing transport with unreachable target
			signingTransport := &transport.SigningTransport{
				TargetURL: tt.targetURL,
				Signer:    signer,
			}

			// Create the proxy
			proxy, err := New(Config{
				Transport:     signingTransport,
				ServerName:    "test-proxy",
				ServerVersion: "v1.0.0",
			})
			require.NoError(t, err)

			// Try to run the proxy - should fail with network error
			ctx := context.Background()
			err = proxy.Run(ctx)

			// Verify we get a descriptive error message
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), "check network connectivity")
		})
	}
}

// TestErrorHandling_SigningFailure tests that signing errors are handled
// with descriptive messages.
//
// **Validates: Requirement 7.2**
func TestErrorHandling_SigningFailure(t *testing.T) {
	tests := []struct {
		name      string
		signError error
		wantErr   string
	}{
		{
			name:      "missing credentials",
			signError: errors.New("AWS credentials are required for SigV4 signing"),
			wantErr:   "AWS signature generation failed",
		},
		{
			name:      "invalid region",
			signError: errors.New("region is required for SigV4 signing"),
			wantErr:   "AWS signature generation failed",
		},
		{
			name:      "invalid service",
			signError: errors.New("service name is required for SigV4 signing"),
			wantErr:   "AWS signature generation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock target server
			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer targetServer.Close()

			// Create a signer that returns an error
			signer := &mockErrorSigner{
				signError: tt.signError,
			}

			// Create the signing transport
			signingTransport := &transport.SigningTransport{
				TargetURL: targetServer.URL,
				Signer:    signer,
			}

			// Create the proxy
			proxy, err := New(Config{
				Transport:     signingTransport,
				ServerName:    "test-proxy",
				ServerVersion: "v1.0.0",
			})
			require.NoError(t, err)

			// Try to run the proxy - should fail with signing error
			ctx := context.Background()
			err = proxy.Run(ctx)

			// Verify we get a descriptive error message
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestErrorHandling_TargetServerError tests that errors from the target
// server are forwarded to the client unchanged.
//
// **Validates: Requirement 7.3**
func TestErrorHandling_TargetServerError(t *testing.T) {
	// This is already tested in proxy_property_test.go
	// (TestProperty8_ErrorResponseTransparency)
	// This test serves as documentation that the requirement is covered
	t.Skip("Covered by TestProperty8_ErrorResponseTransparency")
}

// TestErrorHandling_MissingTransport tests that creating a proxy without
// a transport returns a descriptive error.
func TestErrorHandling_MissingTransport(t *testing.T) {
	// Try to create a proxy without a transport
	proxy, err := New(Config{
		Transport:     nil,
		ServerName:    "test-proxy",
		ServerVersion: "v1.0.0",
	})

	// Verify we get a descriptive error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transport is required")
	assert.Nil(t, proxy)
}

// TestErrorHandling_NetworkErrorMessage tests that network errors include
// helpful information about the target URL.
func TestErrorHandling_NetworkErrorMessage(t *testing.T) {
	// Create a server that immediately closes connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Close the connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
		}
	}))
	server.Close() // Close the server to make it unreachable

	// Create a mock signer
	signer := &mockErrorSigner{}

	// Create the signing transport
	signingTransport := &transport.SigningTransport{
		TargetURL: server.URL,
		Signer:    signer,
	}

	// Create the proxy
	proxy, err := New(Config{
		Transport:     signingTransport,
		ServerName:    "test-proxy",
		ServerVersion: "v1.0.0",
	})
	require.NoError(t, err)

	// Try to run the proxy
	ctx := context.Background()
	err = proxy.Run(ctx)

	// Verify the error message includes helpful information
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to target MCP server")
	assert.Contains(t, err.Error(), server.URL)
	assert.Contains(t, err.Error(), "check network connectivity")
	assert.Contains(t, err.Error(), "AWS credentials")
	assert.Contains(t, err.Error(), "target server availability")
}

// mockErrorSigner is a test implementation of the Signer interface
// that can be configured to return errors
type mockErrorSigner struct {
	signError error
}

func (m *mockErrorSigner) SignRequest(ctx context.Context, req *http.Request, payloadHash string) error {
	if m.signError != nil {
		return m.signError
	}
	// Add a test signature header to verify signing occurred
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20240101/us-east-1/execute-api/aws4_request")
	req.Header.Set("X-Amz-Date", "20240101T000000Z")
	return nil
}
