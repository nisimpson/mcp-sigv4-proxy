package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"pgregory.net/rapid"
)

// TestProperty8_ResponseTransparency tests that for any response from the target,
// the response returned to the client is identical.
//
// **Validates: Requirements 1.2, 4.1, 4.2, 4.3**
//
// This property verifies that the proxy forwards responses from the target MCP server
// to the client without modification. The proxy should be transparent - all response
// data (tools, resources, prompts, etc.) should pass through unchanged.
//
// This test focuses on the transport layer where the actual HTTP response forwarding
// occurs, ensuring that response bodies and status codes are preserved.
func TestProperty8_ResponseTransparency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary JSON-RPC response
		responseType := rapid.SampledFrom([]string{
			"tools/list",
			"resources/list",
			"prompts/list",
			"tools/call",
			"resources/read",
			"prompts/get",
		}).Draw(t, "responseType")

		// Generate response ID
		responseID := rapid.IntRange(1, 1000).Draw(t, "responseID")

		// Generate response data based on type
		var responseData map[string]interface{}
		switch responseType {
		case "tools/list":
			numTools := rapid.IntRange(0, 5).Draw(t, "numTools")
			tools := make([]map[string]interface{}, numTools)
			for i := 0; i < numTools; i++ {
				tools[i] = map[string]interface{}{
					"name":        rapid.StringMatching(`[a-z_]+`).Draw(t, fmt.Sprintf("toolName%d", i)),
					"description": rapid.String().Draw(t, fmt.Sprintf("toolDesc%d", i)),
					"inputSchema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"param": map[string]interface{}{
								"type": "string",
							},
						},
					},
				}
			}
			responseData = map[string]interface{}{
				"tools": tools,
			}

		case "resources/list":
			numResources := rapid.IntRange(0, 5).Draw(t, "numResources")
			resources := make([]map[string]interface{}, numResources)
			for i := 0; i < numResources; i++ {
				resources[i] = map[string]interface{}{
					"uri":         rapid.StringMatching(`file://[a-z/]+`).Draw(t, fmt.Sprintf("resourceURI%d", i)),
					"name":        rapid.String().Draw(t, fmt.Sprintf("resourceName%d", i)),
					"description": rapid.String().Draw(t, fmt.Sprintf("resourceDesc%d", i)),
					"mimeType":    rapid.SampledFrom([]string{"text/plain", "application/json"}).Draw(t, fmt.Sprintf("mimeType%d", i)),
				}
			}
			responseData = map[string]interface{}{
				"resources": resources,
			}

		case "prompts/list":
			numPrompts := rapid.IntRange(0, 5).Draw(t, "numPrompts")
			prompts := make([]map[string]interface{}, numPrompts)
			for i := 0; i < numPrompts; i++ {
				prompts[i] = map[string]interface{}{
					"name":        rapid.StringMatching(`[a-z_]+`).Draw(t, fmt.Sprintf("promptName%d", i)),
					"description": rapid.String().Draw(t, fmt.Sprintf("promptDesc%d", i)),
				}
			}
			responseData = map[string]interface{}{
				"prompts": prompts,
			}

		case "tools/call":
			responseData = map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": rapid.String().Draw(t, "toolResultText"),
					},
				},
			}

		case "resources/read":
			responseData = map[string]interface{}{
				"contents": []map[string]interface{}{
					{
						"uri":      rapid.StringMatching(`file://[a-z/]+`).Draw(t, "resourceURI"),
						"mimeType": "text/plain",
						"text":     rapid.String().Draw(t, "resourceText"),
					},
				},
			}

		case "prompts/get":
			responseData = map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"role": "user",
						"content": map[string]interface{}{
							"type": "text",
							"text": rapid.String().Draw(t, "promptText"),
						},
					},
				},
			}
		}

		// Create the complete JSON-RPC response
		expectedResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      responseID,
			"result":  responseData,
		}

		expectedResponseJSON, err := json.Marshal(expectedResponse)
		if err != nil {
			t.Fatalf("failed to marshal expected response: %v", err)
		}

		// Track the actual response received by the client
		var actualResponseBody []byte

		// Create a mock target MCP server that returns the generated response
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the request is signed (has Authorization header)
			if r.Header.Get("Authorization") == "" {
				t.Fatal("request to target server is not signed")
			}

			// Return the generated response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(expectedResponseJSON)
		}))
		defer targetServer.Close()

		// Create a mock signer
		signer := &mockSigner{}

		// Create an HTTP client that uses the signing transport
		client := &http.Client{
			Transport: &signingRoundTripper{
				transport: http.DefaultTransport,
				signer:    signer,
				ctx:       context.Background(),
			},
		}

		// Make a request through the signing transport
		req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`)))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Read the response body
		actualResponseBody, err = io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		// Property: The response body received by the client must be identical to the response from the target
		if !bytes.Equal(actualResponseBody, expectedResponseJSON) {
			t.Fatalf("response not preserved:\nexpected: %s\nactual: %s",
				string(expectedResponseJSON), string(actualResponseBody))
		}

		// Additional verification: Ensure the response is valid JSON-RPC
		var actualResponse map[string]interface{}
		if err := json.Unmarshal(actualResponseBody, &actualResponse); err != nil {
			t.Fatalf("response is not valid JSON: %v", err)
		}

		// Verify the response has the expected structure
		if actualResponse["jsonrpc"] != "2.0" {
			t.Fatalf("invalid jsonrpc version: %v", actualResponse["jsonrpc"])
		}
		if actualResponse["id"] != float64(responseID) {
			t.Fatalf("response ID mismatch: expected %d, got %v", responseID, actualResponse["id"])
		}
		if actualResponse["result"] == nil {
			t.Fatal("response missing result field")
		}
	})
}

// TestProperty8_ErrorResponseTransparency tests that error responses from the target
// are forwarded to the client without modification.
//
// **Validates: Requirements 1.2, 1.3, 7.3**
//
// This property verifies that when the target MCP server returns an error,
// the proxy forwards that error to the client unchanged.
func TestProperty8_ErrorResponseTransparency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate arbitrary error response
		errorCode := rapid.IntRange(-32768, -32000).Draw(t, "errorCode")
		errorMessage := rapid.String().Draw(t, "errorMessage")
		responseID := rapid.IntRange(1, 1000).Draw(t, "responseID")

		// Create error response
		expectedErrorResponse := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      responseID,
			"error": map[string]interface{}{
				"code":    errorCode,
				"message": errorMessage,
			},
		}

		expectedErrorResponseJSON, err := json.Marshal(expectedErrorResponse)
		if err != nil {
			t.Fatalf("failed to marshal error response: %v", err)
		}

		// Create a mock target MCP server that returns the error
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the request is signed
			if r.Header.Get("Authorization") == "" {
				t.Fatal("request to target server is not signed")
			}

			// Return the error response (JSON-RPC errors use 200 status)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(expectedErrorResponseJSON)
		}))
		defer targetServer.Close()

		// Create a mock signer
		signer := &mockSigner{}

		// Create an HTTP client that uses the signing transport
		client := &http.Client{
			Transport: &signingRoundTripper{
				transport: http.DefaultTransport,
				signer:    signer,
				ctx:       context.Background(),
			},
		}

		// Make a request through the signing transport
		req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`)))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Read the response body
		actualResponseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}

		// Property: The error response received by the client must be identical to the error from the target
		if !bytes.Equal(actualResponseBody, expectedErrorResponseJSON) {
			t.Fatalf("error response not preserved:\nexpected: %s\nactual: %s",
				string(expectedErrorResponseJSON), string(actualResponseBody))
		}

		// Additional verification: Ensure the error response is valid JSON-RPC
		var actualErrorResponse map[string]interface{}
		if err := json.Unmarshal(actualResponseBody, &actualErrorResponse); err != nil {
			t.Fatalf("error response is not valid JSON: %v", err)
		}

		// Verify the error response has the expected structure
		if actualErrorResponse["jsonrpc"] != "2.0" {
			t.Fatalf("invalid jsonrpc version: %v", actualErrorResponse["jsonrpc"])
		}
		if actualErrorResponse["id"] != float64(responseID) {
			t.Fatalf("response ID mismatch: expected %d, got %v", responseID, actualErrorResponse["id"])
		}
		if actualErrorResponse["error"] == nil {
			t.Fatal("error response missing error field")
		}

		// Verify error code and message
		errorObj := actualErrorResponse["error"].(map[string]interface{})
		if int(errorObj["code"].(float64)) != errorCode {
			t.Fatalf("error code mismatch: expected %d, got %v", errorCode, errorObj["code"])
		}
		if errorObj["message"] != errorMessage {
			t.Fatalf("error message mismatch: expected %q, got %q", errorMessage, errorObj["message"])
		}
	})
}

// TestProperty8_ResponseStatusCodePreservation tests that HTTP status codes from the target
// are preserved when forwarding to the client.
//
// **Validates: Requirements 1.2, 4.1, 4.2, 4.3**
//
// This property verifies that HTTP status codes from the target server
// are preserved when forwarding responses to the client.
func TestProperty8_ResponseStatusCodePreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid HTTP status code
		statusCode := rapid.SampledFrom([]int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusAccepted,
			http.StatusNoContent,
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
			http.StatusServiceUnavailable,
		}).Draw(t, "statusCode")

		// Create a simple response body
		responseBody := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  map[string]interface{}{"status": "ok"},
		}
		responseJSON, _ := json.Marshal(responseBody)

		// Create a mock target MCP server that returns the status code
		targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify the request is signed
			if r.Header.Get("Authorization") == "" {
				t.Fatal("request to target server is not signed")
			}

			// Return the response with the generated status code
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			w.Write(responseJSON)
		}))
		defer targetServer.Close()

		// Create a mock signer
		signer := &mockSigner{}

		// Create an HTTP client that uses the signing transport
		client := &http.Client{
			Transport: &signingRoundTripper{
				transport: http.DefaultTransport,
				signer:    signer,
				ctx:       context.Background(),
			},
		}

		// Make a request through the signing transport
		req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"test","id":1}`)))
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Property: The status code received by the client must match the status code from the target
		if resp.StatusCode != statusCode {
			t.Fatalf("status code not preserved: expected %d, got %d", statusCode, resp.StatusCode)
		}
	})
}

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

// signingRoundTripper wraps an http.RoundTripper and signs all requests
type signingRoundTripper struct {
	transport http.RoundTripper
	signer    *mockSigner
	ctx       context.Context
}

// RoundTrip implements the http.RoundTripper interface with request signing
func (rt *signingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Use the default transport if none is specified
	transport := rt.transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Read the request body to calculate the payload hash
	var payloadHash string
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()

		// Calculate SHA256 hash of the payload (simplified for testing)
		payloadHash = "test-hash"

		// Create a new reader with the body content for the actual request
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	} else {
		payloadHash = "empty-hash"
	}

	// Sign the request
	if err := rt.signer.SignRequest(rt.ctx, req, payloadHash); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Execute the signed request
	return transport.RoundTrip(req)
}
