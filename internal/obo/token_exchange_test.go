package obo

import (
	"context"
	"testing"
	"time"
)

// TestTokenExchangeBasic verifies basic token exchange (M1.3.2)
func TestTokenExchangeBasic(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	// Create a minimal JWT for testing
	originalToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhbGljZUBjb3JwLmNvbSIsInNjb3BlIjoicmVhZDpvcmRlcnMgd3JpdGU6Y3VzdG9tZXJzIiwiaWF0IjoxNzI1NDU2MDAwLCJleHAiOjE3MjU0NTk2MDB9.test-sig"

	req := &ExchangeRequest{
		OriginalToken: originalToken,
		Scope:         "read:orders",
		Audience:      "api.internal",
		TTL:           3600,
	}

	ctx := context.Background()
	resp, err := te.Exchange(ctx, req)

	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("TokenType: got %q, want Bearer", resp.TokenType)
	}
	if resp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn: got %d, want 3600", resp.ExpiresIn)
	}
	if resp.Scope != "read:orders" {
		t.Errorf("Scope: got %q, want read:orders", resp.Scope)
	}
}

// TestTokenExchangeDefaultTTL verifies default TTL (M1.3.2)
func TestTokenExchangeDefaultTTL(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	originalToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJib2JAY29ycC5jb20iLCJzY29wZSI6InJlYWQ6YWxsIiwiaWF0IjoxNzI1NDU2MDAwLCJleHAiOjE3MjU0NTk2MDB9.test-sig"

	req := &ExchangeRequest{
		OriginalToken: originalToken,
		Scope:         "read:customers",
		Audience:      "api.internal",
		TTL:           0, // Use default
	}

	resp, err := te.Exchange(context.Background(), req)
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}

	if resp.ExpiresIn != 3600 {
		t.Errorf("Default TTL: got %d, want 3600", resp.ExpiresIn)
	}
}

// TestTokenExchangeInvalidToken verifies error handling for malformed tokens (M1.3.2)
func TestTokenExchangeInvalidToken(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	tests := []struct {
		name      string
		token     string
		wantError bool
	}{
		{"empty token", "", true},
		{"not a JWT", "not-a-jwt", true},
		{"too many parts", "a.b.c.d", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ExchangeRequest{
				OriginalToken: tt.token,
				Scope:         "read:data",
			}

			_, err := te.Exchange(context.Background(), req)
			if (err != nil) != tt.wantError {
				t.Errorf("Exchange error: got %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestTokenExchangeEmptyScope verifies scope is required (M1.3.2)
func TestTokenExchangeEmptyScope(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	originalToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhbGljZUBjb3JwLmNvbSIsInNjb3BlIjoicmVhZDpvcmRlcnMiLCJpYXQiOjE3MjU0NTYwMDAsImV4cCI6MTcyNTQ1OTYwMH0.test-sig"

	req := &ExchangeRequest{
		OriginalToken: originalToken,
		Scope:         "", // Empty scope
	}

	_, err := te.Exchange(context.Background(), req)
	if err == nil {
		t.Error("Exchange should fail with empty scope")
	}
}

// TestTokenExchangeNilRequest verifies nil request handling (M1.3.2)
func TestTokenExchangeNilRequest(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	_, err := te.Exchange(context.Background(), nil)
	if err == nil {
		t.Error("Exchange should fail with nil request")
	}
}

// TestValidateScopeWildcard verifies wildcard scope rejection (M1.3.2)
func TestValidateScopeWildcard(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	originalToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhbGljZUBjb3JwLmNvbSIsInNjb3BlIjoiKiIsImlhdCI6MTcyNTQ1NjAwMCwiZXhwIjoxNzI1NDU5NjAwfQ.test-sig"

	err := te.ValidateScope(originalToken, "*")
	if err == nil {
		t.Error("ValidateScope should reject wildcard scope")
	}
}

// TestTokenStructure verifies issued token has correct structure (M1.3.2)
func TestTokenStructure(t *testing.T) {
	te := NewTokenExchanger("test-secret")

	claims := &TokenClaims{
		Sub:             "alice@corp.com",
		Scope:           "read:orders",
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour).Unix(),
		OriginalTokenID: "token-hash-abc123",
		IssuedBy:        "dim",
		Audience:        "api.internal",
	}

	token, err := te.issueToken(claims)
	if err != nil {
		t.Fatalf("issueToken failed: %v", err)
	}

	// Verify JWT structure (header.payload.signature)
	parts := len([]byte{})
	for _, ch := range token {
		if ch == '.' {
			parts++
		}
	}
	parts++ // Account for segments separated by dots

	if parts != 3 {
		t.Errorf("Token structure: got %d parts, want 3", parts)
	}
}
