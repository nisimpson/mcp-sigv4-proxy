package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/nisimpson/mcp-sigv4-proxy/internal/config"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/credentials"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/proxy"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/signer"
	"github.com/nisimpson/mcp-sigv4-proxy/internal/transport"
)

const (
	serverName    = "sigv4-proxy"
	serverVersion = "v1.0.0"
)

func main() {
	// Set up structured logging
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Run the proxy and handle errors
	if err := run(logger); err != nil {
		logger.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

// run contains the main application logic
func run(logger *log.Logger) error {
	logger.Printf("AWS SigV4 Signing Proxy MCP Server v%s\n", serverVersion)

	// Load configuration from environment variables and command-line flags
	logger.Println("Loading configuration...")
	cfg, err := config.Load(logger)
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	logger.Printf("Configuration loaded successfully:")
	logger.Printf("  Target URL: %s", cfg.TargetURL)
	logger.Printf("  Region: %s", cfg.Region)
	logger.Printf("  Service: %s", cfg.ServiceName)
	logger.Printf("  Signature Version: %s", cfg.SignatureVersion)
	logger.Printf("  Profile: %s", cfg.Profile)
	logger.Printf("  EnableSSE: %v", cfg.EnableSSE)

	// Create context that can be cancelled on shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown on SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
	}()

	// Initialize AWS credentials
	logger.Println("Loading AWS credentials...")
	credProvider := &credentials.Provider{
		Profile: cfg.Profile,
		Region:  cfg.Region,
	}

	creds, err := credProvider.LoadCredentials(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS credentials: %w (ensure AWS credentials are configured via environment variables, ~/.aws/credentials, or IAM role)", err)
	}

	// Mask the secret key in logs for security
	logger.Printf("AWS credentials loaded successfully (Access Key: %s...)", maskAccessKey(creds.AccessKeyID))
	if creds.SessionToken != "" {
		logger.Println("  Session token present")
	}

	// Create the appropriate signer based on signature version
	var sig signer.Signer
	switch cfg.SignatureVersion {
	case "v4":
		logger.Println("Using AWS Signature Version 4 (SigV4)")
		sig = &signer.V4Signer{
			Credentials: creds,
			Region:      cfg.Region,
			Service:     cfg.ServiceName,
		}
	case "v4a":
		logger.Println("Using AWS Signature Version 4A (SigV4a)")
		sig = &signer.V4aSigner{
			Credentials: creds,
			Region:      cfg.Region,
			Service:     cfg.ServiceName,
		}
	default:
		return fmt.Errorf("unsupported signature version: %s (must be 'v4' or 'v4a')", cfg.SignatureVersion)
	}

	// Create the signing transport
	signingTransport := &transport.SigningTransport{
		TargetURL:  cfg.TargetURL,
		Signer:     sig,
		EnableSSE:  cfg.EnableSSE,
		HTTPClient: &http.Client{Timeout: cfg.Timeout},
		Headers:    make(map[string]string),
	}

	if cfg.Headers != "" {
		tokens := strings.Split(cfg.Headers, ",")
		for _, token := range tokens {
			pair := strings.Split(token, "=")
			signingTransport.Headers[pair[0]] = pair[1]
		}
	}
	
	// Create the proxy server
	logger.Println("Creating proxy server...")
	proxyServer, err := proxy.New(proxy.Config{
		Transport:     signingTransport,
		ServerName:    serverName,
		ServerVersion: serverVersion,
	})
	if err != nil {
		return fmt.Errorf("failed to create proxy server: %w", err)
	}

	// Start the proxy server
	logger.Println("Starting proxy server on stdio...")
	logger.Println("Proxy is ready to accept MCP protocol messages")

	if err := proxyServer.Run(ctx); err != nil {
		// Check if this is a graceful shutdown
		if ctx.Err() == context.Canceled {
			logger.Println("Proxy server stopped gracefully")
			return nil
		}
		return fmt.Errorf("proxy server error: %w", err)
	}

	logger.Println("Proxy server stopped")
	return nil
}

// maskAccessKey masks most of the access key for security logging
func maskAccessKey(accessKey string) string {
	if len(accessKey) <= 8 {
		return "****"
	}
	return accessKey[:4] + "****" + accessKey[len(accessKey)-4:]
}
