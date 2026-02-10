package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/signer"
)

// SigningTransport implements mcp.Transport with AWS signature support.
// It wraps HTTP requests to the target MCP server with AWS SigV4/SigV4a signatures.
type SigningTransport struct {
	// HTTPClient makes the actual HTTP requests
	HTTPClient *http.Client

	// Signer signs HTTP requests
	Signer signer.Signer

	// TargetURL is the endpoint of the target MCP server
	TargetURL string
}

// Connect implements mcp.Transport by creating a connection to the target MCP server
// using the streamable HTTP transport with request signing.
func (t *SigningTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	if t.HTTPClient == nil {
		t.HTTPClient = http.DefaultClient
	}

	// Create a signing HTTP client that wraps the original client's transport
	signingClient := &http.Client{
		Transport: NewSigningRoundTripper(t.HTTPClient.Transport, t.Signer),
		Timeout:   t.HTTPClient.Timeout,
	}

	// Use the MCP SDK's StreamableClientTransport with our signing client
	streamTransport := &mcp.StreamableClientTransport{
		Endpoint:   t.TargetURL,
		HTTPClient: signingClient,
	}

	return streamTransport.Connect(ctx)
}

// SigningRoundTripper wraps an http.RoundTripper and signs all requests.
// This is exported for use in testing and custom HTTP client configurations.
type SigningRoundTripper struct {
	Transport http.RoundTripper
	Signer    signer.Signer
}

// NewSigningRoundTripper creates a new SigningRoundTripper with the given transport and signer.
func NewSigningRoundTripper(transport http.RoundTripper, signer signer.Signer) *SigningRoundTripper {
	return &SigningRoundTripper{
		Transport: transport,
		Signer:    signer,
	}
}

// RoundTrip implements the http.RoundTripper interface with request signing
func (rt *SigningRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Use the default transport if none is specified
	transport := rt.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// Read the request body to calculate the payload hash
	var payloadHash string
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body for signing: %w", err)
		}
		req.Body.Close() // Close the original body

		// Calculate SHA256 hash of the payload
		hash := sha256.Sum256(body)
		payloadHash = hex.EncodeToString(hash[:])

		// Create a new reader with the body content for the actual request
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
	} else {
		// Empty payload hash for requests without a body
		hash := sha256.Sum256([]byte{})
		payloadHash = hex.EncodeToString(hash[:])
	}

	// Sign the request using the context from the request
	if err := rt.Signer.SignRequest(req.Context(), req, payloadHash); err != nil {
		return nil, fmt.Errorf("AWS signature generation failed: %w", err)
	}

	// Execute the signed request
	resp, err := transport.RoundTrip(req)
	if err != nil {
		// Enhance network error messages
		return nil, fmt.Errorf("failed to connect to target MCP server at %s: %w", req.URL.Host, err)
	}

	return resp, nil
}
