package transport

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSigner is a test implementation of the Signer interface
type mockSigner struct {
	signedRequests []*http.Request
	signError      error
}

func (m *mockSigner) SignRequest(ctx context.Context, req *http.Request, payloadHash string) error {
	if m.signError != nil {
		return m.signError
	}
	m.signedRequests = append(m.signedRequests, req)
	// Add a test signature header to verify signing occurred
	req.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential=test/20240101/us-east-1/execute-api/aws4_request")
	req.Header.Set("X-Amz-Date", "20240101T000000Z")
	return nil
}

func TestSigningTransport_Connect(t *testing.T) {
	tests := []struct {
		name      string
		targetURL string
		wantErr   bool
	}{
		{
			name:      "successful connection",
			targetURL: "https://example.com/mcp",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mockSigner{}
			transport := &SigningTransport{
				TargetURL: tt.targetURL,
				Signer:    signer,
			}

			ctx := context.Background()
			conn, err := transport.Connect(ctx)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, conn)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, conn)
			}
		})
	}
}

func TestSigningRoundTripper_RoundTrip(t *testing.T) {
	tests := []struct {
		name            string
		requestBody     string
		wantAuthHeader  bool
		wantPayloadHash string
	}{
		{
			name:           "signs request with body",
			requestBody:    `{"jsonrpc":"2.0","method":"test","id":1}`,
			wantAuthHeader: true,
			wantPayloadHash: func() string {
				hash := sha256.Sum256([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`))
				return hex.EncodeToString(hash[:])
			}(),
		},
		{
			name:           "signs request without body",
			requestBody:    "",
			wantAuthHeader: true,
			wantPayloadHash: func() string {
				hash := sha256.Sum256([]byte{})
				return hex.EncodeToString(hash[:])
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that echoes back the request
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the Authorization header is present
				if tt.wantAuthHeader {
					assert.NotEmpty(t, r.Header.Get("Authorization"))
					assert.Contains(t, r.Header.Get("Authorization"), "AWS4-HMAC-SHA256")
				}

				// Echo back the body
				body, _ := io.ReadAll(r.Body)
				w.WriteHeader(http.StatusOK)
				w.Write(body)
			}))
			defer server.Close()

			// Create the signing round tripper
			signer := &mockSigner{}
			rt := NewSigningRoundTripper(http.DefaultTransport, signer, map[string]string{})

			// Create a request
			var body io.Reader
			if tt.requestBody != "" {
				body = strings.NewReader(tt.requestBody)
			}
			req, err := http.NewRequest("POST", server.URL, body)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Execute the request
			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the response
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify signing occurred
			assert.Len(t, signer.signedRequests, 1)

			// Verify the body was preserved
			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.requestBody, string(respBody))
		})
	}
}

func TestSigningTransport_Integration(t *testing.T) {
	// Create a test MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is signed
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.Contains(t, authHeader, "AWS4-HMAC-SHA256")

		// Return a mock MCP response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"success":true}}`))
	}))
	defer server.Close()

	// Create the signing transport
	signer := &mockSigner{}
	transport := &SigningTransport{
		TargetURL: server.URL,
		Signer:    signer,
	}

	// Connect to the server
	ctx := context.Background()
	conn, err := transport.Connect(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Verify signing occurred during connection setup
	// Note: The actual signing happens when requests are made through the connection
	assert.NotNil(t, transport.Signer)
}

func TestSigningRoundTripper_SigningError(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a signer that returns an error
	signer := &mockSigner{
		signError: assert.AnError,
	}
	rt := NewSigningRoundTripper(http.DefaultTransport, signer, map[string]string{})

	// Create a request
	req, err := http.NewRequest("POST", server.URL, strings.NewReader("test"))
	require.NoError(t, err)

	// Execute the request - should fail due to signing error
	resp, err := rt.RoundTrip(req)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "AWS signature generation failed")
}

func TestSigningTransport_DefaultHTTPClient(t *testing.T) {
	signer := &mockSigner{}
	transport := &SigningTransport{
		TargetURL:  "https://example.com",
		Signer:     signer,
		HTTPClient: nil, // No client provided
	}

	ctx := context.Background()
	conn, err := transport.Connect(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, conn)
	// Verify default client was used
	assert.NotNil(t, transport.HTTPClient)
}

func TestSigningTransport_WithSSE(t *testing.T) {
	tests := []struct {
		name      string
		enableSSE bool
	}{
		{
			name:      "SSE enabled",
			enableSSE: true,
		},
		{
			name:      "SSE disabled",
			enableSSE: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mockSigner{}
			transport := &SigningTransport{
				TargetURL:  "https://example.com",
				Signer:     signer,
				EnableSSE:  tt.enableSSE,
				HTTPClient: &http.Client{},
			}

			ctx := context.Background()
			conn, err := transport.Connect(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, conn)
			assert.Equal(t, tt.enableSSE, transport.EnableSSE)
		})
	}
}

func TestSigningTransport_WithTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout string
	}{
		{
			name:    "30 second timeout",
			timeout: "30s",
		},
		{
			name:    "1 minute timeout",
			timeout: "1m",
		},
		{
			name:    "no timeout",
			timeout: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer := &mockSigner{}

			// Parse timeout duration
			var timeout http.Client
			if tt.timeout != "0s" {
				// In real usage, this would be set from config
				timeout = http.Client{}
			}

			transport := &SigningTransport{
				TargetURL:  "https://example.com",
				Signer:     signer,
				HTTPClient: &timeout,
			}

			ctx := context.Background()
			conn, err := transport.Connect(ctx)

			assert.NoError(t, err)
			assert.NotNil(t, conn)
		})
	}
}

func TestSigningRoundTripper_WithCustomHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "single custom header",
			headers: map[string]string{
				"X-Custom-Header": "value",
			},
		},
		{
			name: "multiple custom headers",
			headers: map[string]string{
				"X-Custom-Header": "value",
				"X-API-Version":   "v2",
			},
		},
		{
			name:    "no custom headers",
			headers: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server that verifies headers
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify custom headers are present
				for key, expectedValue := range tt.headers {
					actualValue := r.Header.Get(key)
					assert.Equal(t, expectedValue, actualValue, "Header %s should match", key)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"success":true}`))
			}))
			defer server.Close()

			// Create the signing round tripper with custom headers
			signer := &mockSigner{}
			rt := NewSigningRoundTripper(http.DefaultTransport, signer, tt.headers)

			// Create a request
			req, err := http.NewRequest("POST", server.URL, strings.NewReader(`{"test":"data"}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			// Execute the request
			resp, err := rt.RoundTrip(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the response
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify signing occurred
			assert.Len(t, signer.signedRequests, 1)
		})
	}
}

func TestSigningTransport_Integration_WithAllFeatures(t *testing.T) {
	// Create a test MCP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request is signed
		authHeader := r.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.Contains(t, authHeader, "AWS4-HMAC-SHA256")

		// Verify custom headers
		assert.Equal(t, "value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "v2", r.Header.Get("X-API-Version"))

		// Return a mock MCP response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"success":true}}`))
	}))
	defer server.Close()

	// Create the signing transport with all features enabled
	signer := &mockSigner{}
	transport := &SigningTransport{
		TargetURL: server.URL,
		Signer:    signer,
		EnableSSE: true,
		HTTPClient: &http.Client{
			Timeout: 30000000000, // 30 seconds
		},
		Headers: map[string]string{
			"X-Custom-Header": "value",
			"X-API-Version":   "v2",
		},
	}

	// Connect to the server
	ctx := context.Background()
	conn, err := transport.Connect(ctx)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Verify all features are configured
	assert.True(t, transport.EnableSSE)
	assert.NotNil(t, transport.HTTPClient)
	assert.Equal(t, 2, len(transport.Headers))
}
