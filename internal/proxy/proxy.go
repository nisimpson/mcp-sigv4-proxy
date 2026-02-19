package proxy

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/transport"
)

// Proxy represents the main proxy server that forwards MCP messages
// from clients to an IAM-authenticated target MCP server.
//
// The proxy acts as a transparent intermediary:
// - It accepts MCP protocol messages from clients via stdio
// - It forwards messages to the target MCP server via HTTP with AWS SigV4/SigV4a signing
// - It returns responses from the target server back to the client
type Proxy struct {
	// server is the MCP server that accepts client connections via stdio
	server *mcp.Server

	// client is the MCP client that connects to the target server
	client *mcp.Client

	// transport is the signing transport used to connect to the target
	transport *transport.SigningTransport

	// clientSession is the active session with the target server
	clientSession *mcp.ClientSession
}

// Config holds the configuration for creating a new Proxy
type Config struct {
	// Transport is the signing transport for connecting to the target server
	Transport *transport.SigningTransport

	// ServerName is the name of the proxy server (for identification)
	ServerName string

	// ServerVersion is the version of the proxy server
	ServerVersion string
}

// New creates a new Proxy instance with the given configuration.
//
// The proxy will:
// - Create an MCP server for accepting client connections
// - Create an MCP client for connecting to the target server
// - Wire up message forwarding between client and target
func New(cfg Config) (*Proxy, error) {
	if cfg.Transport == nil {
		return nil, fmt.Errorf("transport is required")
	}

	// Set defaults
	if cfg.ServerName == "" {
		cfg.ServerName = "sigv4-proxy"
	}
	if cfg.ServerVersion == "" {
		cfg.ServerVersion = "v1.0.0"
	}

	// Create the MCP server for client-facing interface (stdio)
	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}, nil)

	// Create the MCP client for target connection with signing transport
	client := mcp.NewClient(&mcp.Implementation{
		Name:    cfg.ServerName,
		Version: cfg.ServerVersion,
	}, nil)

	proxy := &Proxy{
		server:    server,
		client:    client,
		transport: cfg.Transport,
	}

	return proxy, nil
}

// Run starts the proxy server and handles message forwarding.
//
// It performs the following steps:
// 1. Connects to the target MCP server using the signing transport
// 2. Discovers the target server's capabilities (tools, resources, prompts)
// 3. Registers forwarding handlers for all discovered capabilities
// 4. Accepts client connections via stdio and forwards messages
// 5. Runs until the context is cancelled or an error occurs
//
// The proxy is transparent - it forwards all MCP protocol messages
// (tools, resources, prompts, etc.) without modification.
//
// Error Handling:
// - Returns descriptive errors if connection to target fails (network errors)
// - Returns descriptive errors if signing fails (credential/configuration errors)
// - Forwards target server errors to clients unchanged
func (p *Proxy) Run(ctx context.Context) error {
	// Connect to the target MCP server using the signing transport
	clientSession, err := p.client.Connect(ctx, p.transport, nil)
	if err != nil {
		// Provide descriptive error message for connection failures
		// This could be due to network issues, signing errors, or target server problems
		return fmt.Errorf(
			"failed to connect to target MCP server at %s: %w "+
				"(check network connectivity, AWS credentials, and target server availability)",
			p.transport.TargetURL, err)
	}
	defer clientSession.Close()

	// Store the client session for use in forwarding handlers
	p.clientSession = clientSession

	// Discover and register the target server's capabilities
	if err := p.setupForwarding(ctx); err != nil {
		return fmt.Errorf("failed to setup message forwarding: %w", err)
	}

	// Run the server on stdio transport
	// This will accept client connections and forward messages to the target
	stdinTransport := &mcp.StdioTransport{}
	if err := p.server.Run(ctx, stdinTransport); err != nil {
		return fmt.Errorf("proxy server failed: %w", err)
	}

	return nil
}

// setupForwarding discovers the target server's capabilities and registers
// forwarding handlers for all tools, resources, and prompts.
//
// This makes the proxy transparent - all message types are forwarded
// without modification.
func (p *Proxy) setupForwarding(ctx context.Context) error {
	if p.clientSession == nil {
		return fmt.Errorf("not connected to target server")
	}

	// Discover and forward tools
	toolsResult, err := p.clientSession.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		// If listing tools fails, it might not be supported - continue anyway
		// The error will be returned to clients when they try to use tools
	} else {
		for _, tool := range toolsResult.Tools {
			// Create a handler that forwards to the target server
			p.server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				// Convert raw params to CallToolParams
				// The Arguments field is json.RawMessage, which we pass as-is
				var args any
				if len(req.Params.Arguments) > 0 {
					if unmarshalErr := json.Unmarshal(req.Params.Arguments, &args); unmarshalErr != nil {
						return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", unmarshalErr)
					}
				}

				params := &mcp.CallToolParams{
					Name:      req.Params.Name,
					Arguments: args,
				}

				progressToken := req.Params.GetProgressToken()
				if progressToken != nil {
					params.SetProgressToken(progressToken)
				}
	
				// Forward the tool call to the target server
				// Errors from the target server are forwarded unchanged to the client
				result, callErr := p.clientSession.CallTool(ctx, params)
				if callErr != nil {
					// Forward target server errors unchanged (Requirement 7.3)
					return nil, callErr
				}
				return result, nil
			})
		}
	}

	// Discover and forward resources
	resourcesResult, err := p.clientSession.ListResources(ctx, &mcp.ListResourcesParams{})
	if err != nil {
		// If listing resources fails, it might not be supported - continue anyway
	} else {
		for _, resource := range resourcesResult.Resources {
			// Create a handler that forwards to the target server
			p.server.AddResource(resource, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				// Forward the resource read to the target server
				// Errors from the target server are forwarded unchanged to the client
				result, readErr := p.clientSession.ReadResource(ctx, req.Params)
				if readErr != nil {
					// Forward target server errors unchanged (Requirement 7.3)
					return nil, readErr
				}
				return result, nil
			})
		}
	}

	// Discover and forward resource templates
	templatesResult, err := p.clientSession.ListResourceTemplates(ctx, &mcp.ListResourceTemplatesParams{})
	if err != nil {
		// If listing templates fails, it might not be supported - continue anyway
	} else {
		for _, template := range templatesResult.ResourceTemplates {
			// Create a handler that forwards to the target server
			p.server.AddResourceTemplate(template, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				// Forward the resource read to the target server
				// Errors from the target server are forwarded unchanged to the client
				result, readErr := p.clientSession.ReadResource(ctx, req.Params)
				if readErr != nil {
					// Forward target server errors unchanged (Requirement 7.3)
					return nil, readErr
				}
				return result, nil
			})
		}
	}

	// Discover and forward prompts
	promptsResult, err := p.clientSession.ListPrompts(ctx, &mcp.ListPromptsParams{})
	if err != nil {
		// If listing prompts fails, it might not be supported - continue anyway
	} else {
		for _, prompt := range promptsResult.Prompts {
			// Create a handler that forwards to the target server
			p.server.AddPrompt(prompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				// Forward the prompt request to the target server
				// Errors from the target server are forwarded unchanged to the client
				result, err := p.clientSession.GetPrompt(ctx, req.Params)
				if err != nil {
					// Forward target server errors unchanged (Requirement 7.3)
					return nil, err
				}
				return result, nil
			})
		}
	}

	return nil
}
