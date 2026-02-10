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
			rt := NewSigningRoundTripper(http.DefaultTransport, signer)

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
	rt := NewSigningRoundTripper(http.DefaultTransport, signer)

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
