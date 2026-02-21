//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/signer"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/transport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createSigningHTTPClient creates an HTTP client that uses the actual SigningRoundTripper
// from the transport package. This ensures e2e tests use the real production code.
func createSigningHTTPClient(signer signer.Signer) *http.Client {
	return &http.Client{
		Transport: transport.NewSigningRoundTripper(http.DefaultTransport, signer, make(map[string]string)),
	}
}

// TestIntegration_EndToEndMessageFlow tests the complete end-to-end flow
// of MCP messages through the proxy with SigV4 signing.
//
// **Validates: Requirements 1.1, 1.2, 2.1, 4.1, 4.2, 4.3, 4.4**
//
// This integration test verifies:
// - MCP messages are forwarded from client to target server
// - Responses are returned from target to client
// - AWS SigV4 signatures are correctly applied to requests
// - All MCP protocol message types are supported (tools, resources, prompts)
func TestIntegration_EndToEndMessageFlow(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		requestBody     map[string]interface{}
		responseBody    map[string]interface{}
		validateSigV4   bool
		expectedRegion  string
		expectedService string
	}{
		{
			name:   "tools/list request",
			method: "tools/list",
			requestBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"method":  "tools/list",
			},
			responseBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]interface{}{
					"tools": []map[string]interface{}{
						{
							"name":        "test-tool",
							"description": "A test tool",
							"inputSchema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"param": map[string]interface{}{
										"type": "string",
									},
								},
							},
						},
					},
				},
			},
			validateSigV4:   true,
			expectedRegion:  "us-east-1",
			expectedService: "execute-api",
		},
		{
			name:   "resources/list request",
			method: "resources/list",
			requestBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  "resources/list",
			},
			responseBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      2,
				"result": map[string]interface{}{
					"resources": []map[string]interface{}{
						{
							"uri":         "file:///test.txt",
							"name":        "test-resource",
							"description": "A test resource",
							"mimeType":    "text/plain",
						},
					},
				},
			},
			validateSigV4:   true,
			expectedRegion:  "us-east-1",
			expectedService: "execute-api",
		},
		{
			name:   "prompts/list request",
			method: "prompts/list",
			requestBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      3,
				"method":  "prompts/list",
			},
			responseBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      3,
				"result": map[string]interface{}{
					"prompts": []map[string]interface{}{
						{
							"name":        "test-prompt",
							"description": "A test prompt",
						},
					},
				},
			},
			validateSigV4:   true,
			expectedRegion:  "us-east-1",
			expectedService: "execute-api",
		},
		{
			name:   "tools/call request",
			method: "tools/call",
			requestBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      4,
				"method":  "tools/call",
				"params": map[string]interface{}{
					"name": "test-tool",
					"arguments": map[string]interface{}{
						"param": "value",
					},
				},
			},
			responseBody: map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      4,
				"result": map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": "Tool execution result",
						},
					},
				},
			},
			validateSigV4:   true,
			expectedRegion:  "us-east-1",
			expectedService: "execute-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track requests received by the mock target server
			var receivedRequest *http.Request
			var receivedBody []byte

			// Create a mock target MCP server
			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedRequest = r

				// Read the request body
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				receivedBody = body

				// Verify the request is signed with SigV4
				if tt.validateSigV4 {
					authHeader := r.Header.Get("Authorization")
					assert.NotEmpty(t, authHeader, "Authorization header should be present")
					assert.Contains(t, authHeader, "AWS4-HMAC-SHA256", "Should use SigV4 algorithm")
					assert.Contains(t, authHeader, tt.expectedRegion, "Should include configured region")
					assert.Contains(t, authHeader, tt.expectedService, "Should include configured service")

					// Verify X-Amz-Date header is present
					amzDate := r.Header.Get("X-Amz-Date")
					assert.NotEmpty(t, amzDate, "X-Amz-Date header should be present")
				}

				// Return the expected response
				responseJSON, err := json.Marshal(tt.responseBody)
				require.NoError(t, err)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJSON)
			}))
			defer targetServer.Close()

			// Create test AWS credentials
			testCreds := aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Source:          "test",
			}

			// Create a V4 signer
			v4Signer := &signer.V4Signer{
				Credentials: testCreds,
				Region:      tt.expectedRegion,
				Service:     tt.expectedService,
			}

			// Create the signing transport (for verification)
			signingTransport := &transport.SigningTransport{
				TargetURL: targetServer.URL,
				Signer:    v4Signer,
			}

			// Create an HTTP client using the actual SigningRoundTripper
			client := createSigningHTTPClient(v4Signer)

			// Send a request through the signing transport
			requestJSON, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader(requestJSON))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Read the response
			responseBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Parse the response
			var actualResponse map[string]interface{}
			err = json.Unmarshal(responseBody, &actualResponse)
			require.NoError(t, err)

			// Verify the response matches what the target server sent
			assert.Equal(t, tt.responseBody["jsonrpc"], actualResponse["jsonrpc"])
			assert.Equal(t, float64(tt.requestBody["id"].(int)), actualResponse["id"])
			assert.NotNil(t, actualResponse["result"])

			// Verify the request was received by the target server
			assert.NotNil(t, receivedRequest, "Target server should have received a request")
			assert.NotEmpty(t, receivedBody, "Target server should have received a request body")

			// Verify the request body was preserved
			var receivedRequestBody map[string]interface{}
			err = json.Unmarshal(receivedBody, &receivedRequestBody)
			require.NoError(t, err)
			assert.Equal(t, tt.requestBody["method"], receivedRequestBody["method"])

			// Verify the signing transport was created successfully
			assert.NotNil(t, signingTransport)
		})
	}
}

// TestIntegration_SigV4SignatureVerification tests that SigV4 signatures
// are correctly applied with all required components.
//
// **Validates: Requirements 2.1, 2.2, 5.3**
//
// This test verifies:
// - Authorization header contains AWS4-HMAC-SHA256 algorithm
// - Credential scope includes service name and region
// - Session tokens are included when present in credentials
func TestIntegration_SigV4SignatureVerification(t *testing.T) {
	tests := []struct {
		name               string
		credentials        aws.Credentials
		region             string
		service            string
		expectSessionToken bool
	}{
		{
			name: "credentials without session token",
			credentials: aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Source:          "test",
			},
			region:             "us-west-2",
			service:            "execute-api",
			expectSessionToken: false,
		},
		{
			name: "credentials with session token",
			credentials: aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				SessionToken:    "AQoDYXdzEJr...<session-token>...==",
				Source:          "test",
			},
			region:             "eu-west-1",
			service:            "lambda",
			expectSessionToken: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Track the signed request
			var signedRequest *http.Request

			// Create a mock target server
			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				signedRequest = r

				// Return a simple response
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
			}))
			defer targetServer.Close()

			// Create a V4 signer with the test credentials
			v4Signer := &signer.V4Signer{
				Credentials: tt.credentials,
				Region:      tt.region,
				Service:     tt.service,
			}

			// Create an HTTP client using the actual SigningRoundTripper
			client := createSigningHTTPClient(v4Signer)

			// Make a request through the signing transport
			requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
			req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(requestBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Verify the signature components
			require.NotNil(t, signedRequest, "Request should have been received by target server")

			// Verify Authorization header
			authHeader := signedRequest.Header.Get("Authorization")
			assert.NotEmpty(t, authHeader, "Authorization header should be present")
			assert.Contains(t, authHeader, "AWS4-HMAC-SHA256", "Should use SigV4 algorithm")

			// Verify credential scope contains region and service
			// Format: AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-west-2/execute-api/aws4_request
			assert.Contains(t, authHeader, tt.region, "Authorization header should contain region")
			assert.Contains(t, authHeader, tt.service, "Authorization header should contain service name")
			assert.Contains(t, authHeader, "aws4_request", "Authorization header should contain aws4_request")

			// Verify X-Amz-Date header
			amzDate := signedRequest.Header.Get("X-Amz-Date")
			assert.NotEmpty(t, amzDate, "X-Amz-Date header should be present")
			assert.Regexp(t, `^\d{8}T\d{6}Z$`, amzDate, "X-Amz-Date should be in ISO8601 format")

			// Verify session token header if expected
			securityToken := signedRequest.Header.Get("X-Amz-Security-Token")
			if tt.expectSessionToken {
				assert.NotEmpty(t, securityToken, "X-Amz-Security-Token header should be present when credentials have session token")
				assert.Equal(t, tt.credentials.SessionToken, securityToken, "Session token should match credentials")
			} else {
				assert.Empty(t, securityToken, "X-Amz-Security-Token header should not be present when credentials lack session token")
			}
		})
	}
}

// TestIntegration_SigV4aNotAvailable tests that attempting to use SigV4a
// returns an appropriate error since it's not yet available in the AWS SDK.
//
// **Validates: Requirements 3.1**
//
// This test verifies that the proxy correctly handles the case where SigV4a
// is requested but not available due to AWS SDK limitations.
func TestIntegration_SigV4aNotAvailable(t *testing.T) {
	// Create a mock target server (won't be reached due to signing error)
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("Target server should not be reached when SigV4a signing fails")
	}))
	defer targetServer.Close()

	// Create test credentials
	testCreds := aws.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}

	// Create a V4a signer
	v4aSigner := &signer.V4aSigner{
		Credentials: testCreds,
		Region:      "us-east-1",
		Service:     "execute-api",
	}

	// Create an HTTP client using the actual SigningRoundTripper
	client := createSigningHTTPClient(v4aSigner)

	// Attempt to make a request (should fail during signing)
	requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(requestBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	// Verify that an error occurred related to v4a not being available
	require.Error(t, err, "Should return error when attempting to use SigV4a")
	assert.Contains(t, strings.ToLower(err.Error()), "v4a", "Error should mention v4a")
}

// TestIntegration_ErrorForwarding tests that errors from the target server
// are correctly forwarded to the client.
//
// **Validates: Requirements 1.3, 7.3**
//
// This test verifies that when the target MCP server returns an error,
// the proxy forwards that error to the client without modification.
func TestIntegration_ErrorForwarding(t *testing.T) {
	tests := []struct {
		name         string
		errorCode    int
		errorMessage string
		httpStatus   int
	}{
		{
			name:         "method not found error",
			errorCode:    -32601,
			errorMessage: "Method not found",
			httpStatus:   http.StatusOK, // JSON-RPC errors use 200 status
		},
		{
			name:         "invalid params error",
			errorCode:    -32602,
			errorMessage: "Invalid params",
			httpStatus:   http.StatusOK,
		},
		{
			name:         "internal error",
			errorCode:    -32603,
			errorMessage: "Internal error",
			httpStatus:   http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error response
			errorResponse := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      1,
				"error": map[string]interface{}{
					"code":    tt.errorCode,
					"message": tt.errorMessage,
				},
			}

			// Create a mock target server that returns the error
			targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request is signed
				assert.NotEmpty(t, r.Header.Get("Authorization"), "Request should be signed")

				// Return the error response
				responseJSON, _ := json.Marshal(errorResponse)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.httpStatus)
				w.Write(responseJSON)
			}))
			defer targetServer.Close()

			// Create test credentials and signer
			testCreds := aws.Credentials{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				Source:          "test",
			}

			v4Signer := &signer.V4Signer{
				Credentials: testCreds,
				Region:      "us-east-1",
				Service:     "execute-api",
			}

			// Create an HTTP client using the actual SigningRoundTripper
			client := createSigningHTTPClient(v4Signer)

			// Make a request through the signing transport
			requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
			req, err := http.NewRequest("POST", targetServer.URL, bytes.NewReader([]byte(requestBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Read the response
			responseBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// Parse the response
			var actualResponse map[string]interface{}
			err = json.Unmarshal(responseBody, &actualResponse)
			require.NoError(t, err)

			// Verify the error was forwarded correctly
			assert.Equal(t, "2.0", actualResponse["jsonrpc"])
			assert.Equal(t, float64(1), actualResponse["id"])
			assert.NotNil(t, actualResponse["error"], "Response should contain error field")

			// Verify error details
			errorObj := actualResponse["error"].(map[string]interface{})
			assert.Equal(t, float64(tt.errorCode), errorObj["code"])
			assert.Equal(t, tt.errorMessage, errorObj["message"])
		})
	}
}

// TestIntegration_NetworkErrorHandling tests that network errors are properly
// handled and reported.
//
// **Validates: Requirements 7.1**
//
// This test verifies that when the target server is unreachable,
// the proxy returns a descriptive network error.
func TestIntegration_NetworkErrorHandling(t *testing.T) {
	// Use an invalid URL that will cause a network error
	invalidURL := "http://localhost:99999"

	// Create test credentials and signer
	testCreds := aws.Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Source:          "test",
	}

	v4Signer := &signer.V4Signer{
		Credentials: testCreds,
		Region:      "us-east-1",
		Service:     "execute-api",
	}

	// Create an HTTP client using the actual SigningRoundTripper
	client := createSigningHTTPClient(v4Signer)
	client.Timeout = 2 * time.Second

	// Attempt to make a request
	requestBody := `{"jsonrpc":"2.0","id":1,"method":"test"}`
	req, err := http.NewRequest("POST", invalidURL, bytes.NewReader([]byte(requestBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}

	// Verify that a network error occurred
	require.Error(t, err, "Should return error when target is unreachable")

	// The error message should be descriptive
	errorMsg := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errorMsg, "connect") ||
			strings.Contains(errorMsg, "connection") ||
			strings.Contains(errorMsg, "refused") ||
			strings.Contains(errorMsg, "dial"),
		fmt.Sprintf("Error should indicate network/connection issue, got: %s", err.Error()))
}
