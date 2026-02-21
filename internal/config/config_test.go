package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with v4",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
			},
			wantErr: false,
		},
		{
			name: "valid config with v4a",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-west-2",
				ServiceName:      "execute-api",
				SignatureVersion: "v4a",
				Profile:          "default",
			},
			wantErr: false,
		},
		{
			name: "valid config with http",
			config: Config{
				TargetURL:        "http://localhost:8080",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
			},
			wantErr: false,
		},
		{
			name: "missing target URL",
			config: Config{
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
			},
			wantErr: true,
			errMsg:  "target URL is required",
		},
		{
			name: "missing region",
			config: Config{
				TargetURL:        "https://example.com",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
			},
			wantErr: true,
			errMsg:  "region is required",
		},
		{
			name: "missing service name",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				SignatureVersion: "v4",
			},
			wantErr: true,
			errMsg:  "service name is required",
		},
		{
			name: "invalid signature version",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v5",
			},
			wantErr: true,
			errMsg:  "signature version must be 'v4' or 'v4a'",
		},
		{
			name: "invalid URL format",
			config: Config{
				TargetURL:        "not-a-url",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
			},
			wantErr: true,
			errMsg:  "target URL must use http or https scheme",
		},
		{
			name: "invalid URL scheme",
			config: Config{
				TargetURL:        "ftp://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
			},
			wantErr: true,
			errMsg:  "target URL must use http or https scheme",
		},
		{
			name: "multiple validation errors",
			config: Config{
				SignatureVersion: "invalid",
			},
			wantErr: true,
			errMsg:  "target URL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLoadFromEnv_WithAllValues(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")
	origSigVersion := os.Getenv("AWS_SIG_VERSION")
	origProfile := os.Getenv("AWS_PROFILE")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
		os.Setenv("AWS_SIG_VERSION", origSigVersion)
		os.Setenv("AWS_PROFILE", origProfile)
	}()

	// Set test environment variables
	os.Setenv("MCP_TARGET_URL", "https://test.example.com")
	os.Setenv("AWS_REGION", "us-west-2")
	os.Setenv("AWS_SERVICE_NAME", "execute-api")
	os.Setenv("AWS_SIG_VERSION", "v4a")
	os.Setenv("AWS_PROFILE", "test-profile")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "https://test.example.com", cfg.TargetURL)
	assert.Equal(t, "us-west-2", cfg.Region)
	assert.Equal(t, "execute-api", cfg.ServiceName)
	assert.Equal(t, "v4a", cfg.SignatureVersion)
	assert.Equal(t, "test-profile", cfg.Profile)
}

func TestLoadFromEnv_DefaultValues(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")
	origSigVersion := os.Getenv("AWS_SIG_VERSION")
	origProfile := os.Getenv("AWS_PROFILE")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
		os.Setenv("AWS_SIG_VERSION", origSigVersion)
		os.Setenv("AWS_PROFILE", origProfile)
	}()

	// Set only required environment variables
	os.Setenv("MCP_TARGET_URL", "https://test.example.com")
	os.Setenv("AWS_REGION", "us-east-1")
	os.Setenv("AWS_SERVICE_NAME", "execute-api")
	os.Unsetenv("AWS_SIG_VERSION")
	os.Unsetenv("AWS_PROFILE")

	cfg, err := LoadFromEnv()
	require.NoError(t, err)
	assert.Equal(t, "v4", cfg.SignatureVersion, "should default to v4")
	assert.Equal(t, "default", cfg.Profile, "should default to 'default'")
}

func TestLoadFromEnv_MissingRequired(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
	}()

	// Clear all environment variables
	os.Unsetenv("MCP_TARGET_URL")
	os.Unsetenv("AWS_REGION")
	os.Unsetenv("AWS_SERVICE_NAME")

	_, err := LoadFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target URL is required")
	assert.Contains(t, err.Error(), "region is required")
	assert.Contains(t, err.Error(), "service name is required")
}

func TestLoadFromEnv_WithSSE(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")
	origEnableSSE := os.Getenv("MCP_ENABLE_SSE")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
		os.Setenv("MCP_ENABLE_SSE", origEnableSSE)
	}()

	tests := []struct {
		name     string
		sseValue string
		wantSSE  bool
	}{
		{
			name:     "SSE enabled with true",
			sseValue: "true",
			wantSSE:  true,
		},
		{
			name:     "SSE enabled with 1",
			sseValue: "1",
			wantSSE:  true,
		},
		{
			name:     "SSE disabled with false",
			sseValue: "false",
			wantSSE:  false,
		},
		{
			name:     "SSE disabled with 0",
			sseValue: "0",
			wantSSE:  false,
		},
		{
			name:     "SSE disabled with empty",
			sseValue: "",
			wantSSE:  false,
		},
		{
			name:     "SSE disabled with invalid value",
			sseValue: "invalid",
			wantSSE:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MCP_TARGET_URL", "https://test.example.com")
			os.Setenv("AWS_REGION", "us-east-1")
			os.Setenv("AWS_SERVICE_NAME", "execute-api")
			os.Setenv("MCP_ENABLE_SSE", tt.sseValue)

			cfg, err := LoadFromEnv()
			require.NoError(t, err)
			assert.Equal(t, tt.wantSSE, cfg.EnableSSE)
		})
	}
}

func TestLoadFromEnv_WithTimeout(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")
	origTimeout := os.Getenv("MCP_TIMEOUT")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
		os.Setenv("MCP_TIMEOUT", origTimeout)
	}()

	tests := []struct {
		name         string
		timeoutValue string
		wantTimeout  string
	}{
		{
			name:         "timeout with seconds",
			timeoutValue: "30s",
			wantTimeout:  "30s",
		},
		{
			name:         "timeout with minutes",
			timeoutValue: "5m",
			wantTimeout:  "5m0s",
		},
		{
			name:         "timeout with milliseconds",
			timeoutValue: "500ms",
			wantTimeout:  "500ms",
		},
		{
			name:         "no timeout set",
			timeoutValue: "",
			wantTimeout:  "0s",
		},
		{
			name:         "invalid timeout defaults to zero",
			timeoutValue: "invalid",
			wantTimeout:  "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MCP_TARGET_URL", "https://test.example.com")
			os.Setenv("AWS_REGION", "us-east-1")
			os.Setenv("AWS_SERVICE_NAME", "execute-api")
			os.Setenv("MCP_TIMEOUT", tt.timeoutValue)

			cfg, err := LoadFromEnv()
			require.NoError(t, err)
			assert.Equal(t, tt.wantTimeout, cfg.Timeout.String())
		})
	}
}

func TestLoadFromEnv_WithHeaders(t *testing.T) {
	// Save original environment
	origTargetURL := os.Getenv("MCP_TARGET_URL")
	origRegion := os.Getenv("AWS_REGION")
	origServiceName := os.Getenv("AWS_SERVICE_NAME")
	origHeaders := os.Getenv("MCP_HEADERS")

	// Restore environment after test
	defer func() {
		os.Setenv("MCP_TARGET_URL", origTargetURL)
		os.Setenv("AWS_REGION", origRegion)
		os.Setenv("AWS_SERVICE_NAME", origServiceName)
		os.Setenv("MCP_HEADERS", origHeaders)
	}()

	tests := []struct {
		name         string
		headersValue string
		wantHeaders  string
	}{
		{
			name:         "single header",
			headersValue: "X-Custom-Header=value",
			wantHeaders:  "X-Custom-Header=value",
		},
		{
			name:         "multiple headers",
			headersValue: "X-Custom-Header=value,X-API-Version=v2",
			wantHeaders:  "X-Custom-Header=value,X-API-Version=v2",
		},
		{
			name:         "no headers",
			headersValue: "",
			wantHeaders:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MCP_TARGET_URL", "https://test.example.com")
			os.Setenv("AWS_REGION", "us-east-1")
			os.Setenv("AWS_SERVICE_NAME", "execute-api")
			os.Setenv("MCP_HEADERS", tt.headersValue)

			cfg, err := LoadFromEnv()
			require.NoError(t, err)
			assert.Equal(t, tt.wantHeaders, cfg.Headers)
		})
	}
}

func TestConfig_Validate_WithNewFeatures(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config with SSE enabled",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
				EnableSSE:        true,
			},
			wantErr: false,
		},
		{
			name: "valid config with timeout",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
				Timeout:          30000000000, // 30 seconds in nanoseconds
			},
			wantErr: false,
		},
		{
			name: "valid config with headers",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
				Headers:          "X-Custom-Header=value,X-API-Version=v2",
			},
			wantErr: false,
		},
		{
			name: "valid config with all new features",
			config: Config{
				TargetURL:        "https://example.com",
				Region:           "us-east-1",
				ServiceName:      "execute-api",
				SignatureVersion: "v4",
				Profile:          "default",
				EnableSSE:        true,
				Timeout:          30000000000,
				Headers:          "X-Custom-Header=value",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
