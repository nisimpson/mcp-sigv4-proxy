package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransportError_NetworkFailure tests that network errors include
// descriptive messages about the target server.
//
// **Validates: Requirement 7.1**
func TestTransportError_NetworkFailure(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string // Returns target URL
		wantErr   string
	}{
		{
			name: "connection refused",
			setupFunc: func() string {
				// Use a port that's likely not in use
				return "http://localhost:59999"
			},
			wantErr: "failed to connect to target MCP server",
		},
		{
			name: "invalid host",
			setupFunc: func() string {
				return "https://invalid-host-12345.example.invalid"
			},
			wantErr: "failed to connect to target MCP server",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL := tt.setupFunc()

			// Create a mock signer
			signer := &mockSigner{}

			// Create the signing round tripper
			rt := NewSigningRoundTripper(http.DefaultTransport, signer)

			// Create a request to the unreachable target
			req, err := http.NewRequest("POST", targetURL, strings.NewReader("test"))
			require.NoError(t, err)

			// Execute the request - should fail with network error
			resp, err := rt.RoundTrip(req)

			// Verify we get a descriptive error
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestTransportError_SigningFailure tests that signing errors include
// descriptive messages.
//
// **Validates: Requirement 7.2**
func TestTransportError_SigningFailure(t *testing.T) {
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
			name:      "signing algorithm error",
			signError: errors.New("failed to calculate signature"),
			wantErr:   "AWS signature generation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server (won't be reached due to signing error)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("should not reach server due to signing error")
			}))
			defer server.Close()

			// Create a signer that returns an error
			signer := &mockSigner{
				signError: tt.signError,
			}

			// Create the signing round tripper
			rt := NewSigningRoundTripper(http.DefaultTransport, signer)

			// Create a request
			req, err := http.NewRequest("POST", server.URL, strings.NewReader("test"))
			require.NoError(t, err)

			// Execute the request - should fail with signing error
			resp, err := rt.RoundTrip(req)

			// Verify we get a descriptive error
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Contains(t, err.Error(), tt.wantErr)
			assert.Contains(t, err.Error(), tt.signError.Error())
		})
	}
}

// TestTransportError_BodyReadFailure tests that errors reading the request
// body are handled properly.
func TestTransportError_BodyReadFailure(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a mock signer
	signer := &mockSigner{}

	// Create the signing round tripper
	rt := NewSigningRoundTripper(http.DefaultTransport, signer)

	// Create a request with a body that fails to read
	req, err := http.NewRequest("POST", server.URL, &errorReader{})
	require.NoError(t, err)

	// Execute the request - should fail with body read error
	resp, err := rt.RoundTrip(req)

	// Verify we get a descriptive error
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to read request body for signing")
}

// TestTransportError_TargetServerHTTPError tests that HTTP errors from
// the target server are returned (not wrapped with additional context).
//
// **Validates: Requirement 7.3**
func TestTransportError_TargetServerHTTPError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
		},
		{
			name:       "internal server error",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "service unavailable",
			statusCode: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that returns the error status
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(`{"error":"test error"}`))
			}))
			defer server.Close()

			// Create a mock signer
			signer := &mockSigner{}

			// Create the signing round tripper
			rt := NewSigningRoundTripper(http.DefaultTransport, signer)

			// Create a request
			req, err := http.NewRequest("POST", server.URL, strings.NewReader("test"))
			require.NoError(t, err)

			// Execute the request
			resp, err := rt.RoundTrip(req)

			// HTTP errors are returned as responses, not errors
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.statusCode, resp.StatusCode)
		})
	}
}

// TestTransportError_NetworkErrorIncludesHost tests that network errors
// include the target host in the error message.
//
// **Validates: Requirement 7.1**
func TestTransportError_NetworkErrorIncludesHost(t *testing.T) {
	// Create a server and immediately close it
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	targetURL := server.URL
	server.Close()

	// Create a mock signer
	signer := &mockSigner{}

	// Create the signing round tripper
	rt := NewSigningRoundTripper(http.DefaultTransport, signer)

	// Create a request
	req, err := http.NewRequest("POST", targetURL, strings.NewReader("test"))
	require.NoError(t, err)

	// Execute the request - should fail with network error
	resp, err := rt.RoundTrip(req)

	// Verify the error includes the host
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to connect to target MCP server")
	assert.Contains(t, err.Error(), req.URL.Host)
}

// errorReader is a test reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}
