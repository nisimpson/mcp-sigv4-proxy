package transport

import (
	"net/http"

	"github.com/nisimpson/mcp-sigv4-proxy/internal/signer"
)

// SigningTransport implements mcp.Transport with AWS signature support
type SigningTransport struct {
	// TargetURL is the endpoint of the target MCP server
	TargetURL string

	// Signer signs HTTP requests
	Signer signer.Signer

	// HTTPClient makes the actual HTTP requests
	HTTPClient *http.Client
}
