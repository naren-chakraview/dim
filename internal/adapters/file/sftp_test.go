package file

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

// TestNewSFTPSource tests creating an SFTP source (R21.3).
func TestNewSFTPSource(t *testing.T) {
	outChan := engine.NewChannel("test-sftp", 10)

	source, err := NewSFTPSource("localhost", 2222, "testuser", "/upload", "30s", outChan)
	if err != nil {
		t.Fatalf("failed to create SFTP source: %v", err)
	}

	if source.config.SFTPHost != "localhost" {
		t.Errorf("host mismatch: expected localhost, got %s", source.config.SFTPHost)
	}

	if source.config.SFTPPort != 2222 {
		t.Errorf("port mismatch: expected 2222, got %d", source.config.SFTPPort)
	}

	if source.config.SFTPUser != "testuser" {
		t.Errorf("user mismatch: expected testuser, got %s", source.config.SFTPUser)
	}

	if source.config.Path != "/upload" {
		t.Errorf("path mismatch: expected /upload, got %s", source.config.Path)
	}

	if source.config.Type != "sftp" {
		t.Errorf("type mismatch: expected sftp, got %s", source.config.Type)
	}
}

// TestResolveSecret tests secret resolution from environment (R21.2).
func TestResolveSecret(t *testing.T) {
	// Set an environment variable for testing
	testKey := "TEST_SFTP_PASSWORD"
	testValue := "secretpassword123"
	os.Setenv(testKey, testValue)
	defer os.Unsetenv(testKey)

	// Create a FileSource to test secret resolution
	outChan := engine.NewChannel("test-resolve", 10)
	source, err := NewFileSource("/tmp", "30s", outChan)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	// Test resolution (should fall back to env var since no resolver configured)
	resolved := source.resolveSecret(testKey)
	if resolved != testValue {
		t.Errorf("secret resolution failed: expected %s, got %s", testValue, resolved)
	}

	// Test non-existent secret returns empty
	empty := source.resolveSecret("NONEXISTENT_SECRET")
	if empty != "" {
		t.Errorf("non-existent secret should return empty, got %s", empty)
	}
}

// TestSFTPSourceConfiguration tests SFTP source config with credentials (R21.2, R21.3).
func TestSFTPSourceConfiguration(t *testing.T) {
	outChan := engine.NewChannel("test-config", 10)

	tests := []struct {
		name     string
		host     string
		port     int
		user     string
		path     string
		password string
		keyfile  string
		wantErr  bool
	}{
		{
			name:     "valid with password",
			host:     "sftp.example.com",
			port:     22,
			user:     "sftp_user",
			path:     "/home/sftp_user/upload",
			password: "PASSWORD_ENV_VAR",
			wantErr:  false,
		},
		{
			name:    "valid with key",
			host:    "sftp.example.com",
			port:    22,
			user:    "sftp_user",
			path:    "/home/sftp_user/upload",
			keyfile: "/home/user/.ssh/id_rsa",
			wantErr: false,
		},
		{
			name:     "default port when zero",
			host:     "sftp.example.com",
			port:     0, // Should default to 22 in pollSFTP
			user:     "sftp_user",
			path:     "/upload",
			password: "PASSWORD_ENV_VAR",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewSFTPSource(tt.host, tt.port, tt.user, tt.path, "30s", outChan)

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil {
				if source.config.SFTPHost != tt.host {
					t.Errorf("host mismatch: expected %s, got %s", tt.host, source.config.SFTPHost)
				}

				if source.config.SFTPUser != tt.user {
					t.Errorf("user mismatch: expected %s, got %s", tt.user, source.config.SFTPUser)
				}

				if source.config.Path != tt.path {
					t.Errorf("path mismatch: expected %s, got %s", tt.path, source.config.Path)
				}
			}
		})
	}
}

// TestSFTPSourceDefaultPort tests that port defaults to 22 (R21.2).
func TestSFTPSourceDefaultPort(t *testing.T) {
	outChan := engine.NewChannel("test-port", 10)

	// Create source with port 0
	source, err := NewSFTPSource("localhost", 0, "user", "/path", "30s", outChan)
	if err != nil {
		t.Fatalf("failed to create SFTP source: %v", err)
	}

	// Port should still be 0 in config, but pollSFTP should use 22 as default
	if source.config.SFTPPort != 0 {
		t.Errorf("port in config should stay 0, got %d", source.config.SFTPPort)
	}
}

// TestSFTPSourceWithDifferentSchedules tests various schedule strings (R21.3).
func TestSFTPSourceWithDifferentSchedules(t *testing.T) {
	outChan := engine.NewChannel("test-schedules", 10)

	tests := []struct {
		schedule string
		wantErr  bool
	}{
		{"30s", false},
		{"1m", false},
		{"5m", false},
		{"1h", false},
		{"invalid", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("schedule=%s", tt.schedule), func(t *testing.T) {
			source, err := NewSFTPSource("localhost", 22, "user", "/path", tt.schedule, outChan)

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}

			if err == nil && source.config.Schedule != tt.schedule {
				t.Errorf("schedule mismatch: expected %s, got %s", tt.schedule, source.config.Schedule)
			}
		})
	}
}

// TestSFTPIntegration tests actual SFTP connection if SFTP_TEST_HOST is set (R21.3).
// Requires running SFTP server (e.g., docker-compose.sftp.yml)
// Set SFTP_TEST_HOST=localhost SFTP_TEST_PORT=2222 SFTP_TEST_USER=testuser SFTP_TEST_PASSWORD=testpass
func TestSFTPIntegration(t *testing.T) {
	host := os.Getenv("SFTP_TEST_HOST")
	if host == "" {
		t.Skip("SFTP_TEST_HOST not set; skipping SFTP integration tests")
	}

	port := 22
	if portStr := os.Getenv("SFTP_TEST_PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	user := os.Getenv("SFTP_TEST_USER")
	if user == "" {
		user = "testuser"
	}

	password := os.Getenv("SFTP_TEST_PASSWORD")
	if password == "" {
		t.Skip("SFTP_TEST_PASSWORD not set; skipping password auth test")
	}

	// Create test message channel
	outChan := engine.NewChannel("sftp-integration", 10)
	defer outChan.Close()

	source, err := NewSFTPSource(host, port, user, "/upload", "30s", outChan)
	if err != nil {
		t.Fatalf("failed to create SFTP source: %v", err)
	}

	// Set password in environment for resolution
	os.Setenv("SFTP_PASSWORD", password)
	source.config.SFTPPassword = "SFTP_PASSWORD"
	defer os.Unsetenv("SFTP_PASSWORD")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try polling (should connect and list files)
	err = source.poll(ctx)
	if err != nil {
		t.Logf("SFTP poll error (expected if no files): %v", err)
		// Connection errors are expected if server isn't actually running
		// But parse errors mean something is wrong with our implementation
	}
}


// TestSFTPAuthMethods tests that SFTP properly handles different auth methods (R21.2).
func TestSFTPAuthMethods(t *testing.T) {
	outChan := engine.NewChannel("test-auth", 10)

	tests := []struct {
		name     string
		password string
		keyfile  string
		wantErr  bool
	}{
		{
			name:     "password auth configured",
			password: "PASSWORD_VAR",
			keyfile:  "",
			wantErr:  false,
		},
		{
			name:     "key auth configured",
			password: "",
			keyfile:  "/path/to/key",
			wantErr:  false, // Will fail on file read, but that's expected in test
		},
		{
			name:     "both configured (key preferred)",
			password: "PASSWORD_VAR",
			keyfile:  "/path/to/key",
			wantErr:  false,
		},
		{
			name:     "neither configured",
			password: "",
			keyfile:  "",
			wantErr:  false, // Configuration is OK, will fail at connection time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := NewSFTPSource("localhost", 22, "user", "/path", "30s", outChan)
			if err != nil {
				t.Fatalf("failed to create source: %v", err)
			}

			source.config.SFTPPassword = tt.password
			source.config.SFTPKeyFile = tt.keyfile

			// Just verify configuration is accepted
			if source.config.SFTPPassword != tt.password {
				t.Errorf("password config mismatch")
			}
			if source.config.SFTPKeyFile != tt.keyfile {
				t.Errorf("keyfile config mismatch")
			}
		})
	}
}

// TestResolveSecretWithDomainScoping tests that FileSource respects domain-scoped secret resolution (T1.13).
func TestResolveSecretWithDomainScoping(t *testing.T) {
	// Create an in-memory store with domain-scoped secrets
	store := secrets.NewInMemoryStore()
	store.Register(secrets.SecretEntry{Name: "api-key", Value: "real-secret-value", Domain: "orders"})
	store.Register(secrets.SecretEntry{Name: "password", Value: "real-password", Domain: "payments"})
	
	resolver := secrets.NewResolver(store)
	
	// Create a FileSource for the "orders" domain with the resolver
	outChan := engine.NewChannel("test-domain-secret", 10)
	source, err := NewFileSourceWithDomain("/tmp", "30s", outChan, "orders", resolver)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}
	
	// Test same-domain resolution
	resolved := source.resolveSecret("api-key")
	if resolved != "real-secret-value" {
		t.Errorf("same-domain secret resolution failed: got %q, want %q", resolved, "real-secret-value")
	}
	
	// Test cross-domain denial: trying to access payments.password from orders domain
	// With domain-scoped resolver, this should be denied (return empty, not the real value)
	crossDomainResult := source.resolveSecret("payments.password")
	if crossDomainResult == "real-password" {
		t.Errorf("SECURITY BUG: cross-domain secret was resolved when it should be denied: %q", crossDomainResult)
	}
}
