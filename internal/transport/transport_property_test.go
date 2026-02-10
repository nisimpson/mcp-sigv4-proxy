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

	"pgregory.net/rapid"
)

// TestProperty7_MessageBodyPreservation tests that for any MCP message,
// the message body sent to the target equals the message body received from the client.
//
// **Validates: Requirements 1.1, 4.4**
//
// This property verifies that the SigningTransport preserves message bodies
// while adding AWS signatures. The transport should not modify the request body
// during the signing process.
func TestProperty7_MessageBodyPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an arbitrary JSON-RPC message body
		messageBody := rapid.StringMatching(`\{.*\}`).Draw(t, "messageBody")

		// Track what body the server receives
		var receivedBody string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
			receivedBody = string(body)

			// Verify the request is signed
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				t.Fatal("Authorization header is missing")
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
		}))
		defer server.Close()

		// Create the signing round tripper
		signer := &mockSigner{}
		rt := NewSigningRoundTripper(http.DefaultTransport, signer)

		// Create and execute a request with the generated message body
		req, err := http.NewRequest("POST", server.URL, strings.NewReader(messageBody))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Property: The message body received by the server must equal the original message body
		if receivedBody != messageBody {
			t.Fatalf("message body not preserved: sent %q, received %q", messageBody, receivedBody)
		}

		// Additional verification: Ensure signing occurred
		if len(signer.signedRequests) != 1 {
			t.Fatalf("expected 1 signed request, got %d", len(signer.signedRequests))
		}
	})
}

// TestProperty_SignatureIncludesPayloadHash tests that for any request body,
// the signature is calculated using the correct payload hash.
//
// This property verifies that the signing process correctly calculates
// the SHA256 hash of the request body and uses it for signing.
func TestProperty_SignatureIncludesPayloadHash(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an arbitrary request body
		requestBody := rapid.String().Draw(t, "requestBody")

		// Calculate the expected payload hash
		hash := sha256.Sum256([]byte(requestBody))
		expectedHash := hex.EncodeToString(hash[:])

		// Track the payload hash used for signing
		var actualHash string
		testSigner := &testHashSigner{
			onSign: func(ctx context.Context, req *http.Request, payloadHash string) error {
				actualHash = payloadHash
				req.Header.Set("Authorization", "AWS4-HMAC-SHA256 test")
				return nil
			},
		}

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create the signing round tripper
		rt := NewSigningRoundTripper(http.DefaultTransport, testSigner)

		// Create and execute a request
		var body io.Reader
		if requestBody != "" {
			body = strings.NewReader(requestBody)
		}
		req, err := http.NewRequest("POST", server.URL, body)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Property: The payload hash used for signing must match the expected hash
		if actualHash != expectedHash {
			t.Fatalf("payload hash mismatch: expected %q, got %q", expectedHash, actualHash)
		}
	})
}

// TestProperty_SigningPreservesHeaders tests that signing adds headers
// without removing existing headers.
func TestProperty_SigningPreservesHeaders(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid header values (printable ASCII characters only)
		headerName := rapid.StringMatching(`X-[A-Z][a-z]+-[A-Z][a-z]+`).Draw(t, "headerName")
		headerValue := rapid.StringMatching(`[a-zA-Z0-9\-_\.]+`).Draw(t, "headerValue")

		// Track headers received by the server
		var receivedHeaders http.Header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHeaders = r.Header.Clone()
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Create the signing round tripper
		signer := &mockSigner{}
		rt := NewSigningRoundTripper(http.DefaultTransport, signer)

		// Create a request with a custom header
		req, err := http.NewRequest("POST", server.URL, nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set(headerName, headerValue)

		resp, err := rt.RoundTrip(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Property: The original header must be preserved
		if receivedHeaders.Get(headerName) != headerValue {
			t.Fatalf("header not preserved: expected %q, got %q",
				headerValue, receivedHeaders.Get(headerName))
		}

		// Property: Signing headers must be added
		if receivedHeaders.Get("Authorization") == "" {
			t.Fatal("Authorization header not added")
		}
	})
}

// testHashSigner is a test signer that captures the payload hash
type testHashSigner struct {
	onSign func(ctx context.Context, req *http.Request, payloadHash string) error
}

func (s *testHashSigner) SignRequest(ctx context.Context, req *http.Request, payloadHash string) error {
	return s.onSign(ctx, req, payloadHash)
}
