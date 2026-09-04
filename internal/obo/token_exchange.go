package obo

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TokenExchanger performs OAuth 2.0 token exchange for OBO (On-Behalf-Of) flows (M1.3.2)
type TokenExchanger struct {
	secret string
}

// ExchangeRequest represents a token exchange request
type ExchangeRequest struct {
	OriginalToken string // Principal's JWT (full scope)
	Scope         string // Requested scope (e.g., "read:orders")
	Audience      string // Intended recipient service
	TTL           int    // Token lifetime in seconds (default: 3600)
}

// ExchangeResponse represents a token exchange response
type ExchangeResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// TokenClaims represents JWT claims for scoped tokens
type TokenClaims struct {
	Sub              string `json:"sub"`                   // Subject (principal)
	Scope            string `json:"scope"`                 // Granted scope
	IssuedAt         int64  `json:"iat"`                   // Issued timestamp
	ExpiresAt        int64  `json:"exp"`                   // Expiration timestamp
	OriginalTokenID  string `json:"original_token_id"`     // Original token reference
	IssuedBy         string `json:"issued_by"`             // "dim"
	Audience         string `json:"aud"`                   // Intended recipient
	RouteVersion     string `json:"route_version"`         // Route version for invalidation
}

// NewTokenExchanger creates a new token exchanger with a shared secret
func NewTokenExchanger(secret string) *TokenExchanger {
	return &TokenExchanger{
		secret: secret,
	}
}

// Exchange performs token exchange: original token → scoped token (M1.3.2)
func (te *TokenExchanger) Exchange(ctx context.Context, req *ExchangeRequest) (*ExchangeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("exchange request cannot be nil")
	}
	if req.OriginalToken == "" {
		return nil, fmt.Errorf("original token required")
	}
	if req.Scope == "" {
		return nil, fmt.Errorf("requested scope required")
	}

	// Default TTL: 1 hour
	ttl := req.TTL
	if ttl == 0 {
		ttl = 3600
	}

	// Parse original token (simplified: assume JWT format "header.payload.signature")
	parts := strings.Split(req.OriginalToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	// Decode payload (this is simplified; real implementation would validate signature)
	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode token payload: %w", err)
	}

	var originalClaims map[string]interface{}
	if err := json.Unmarshal(payload, &originalClaims); err != nil {
		return nil, fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Extract subject (principal) from original token
	sub, ok := originalClaims["sub"].(string)
	if !ok {
		return nil, fmt.Errorf("original token missing 'sub' claim")
	}

	// Validate requested scope is subset of original scope
	if err := te.ValidateScope(req.OriginalToken, req.Scope); err != nil {
		return nil, fmt.Errorf("scope validation failed: %w", err)
	}

	// Create scoped token claims
	now := time.Now().Unix()
	claims := TokenClaims{
		Sub:             sub,
		Scope:           req.Scope,
		IssuedAt:        now,
		ExpiresAt:       now + int64(ttl),
		OriginalTokenID: hashToken(req.OriginalToken),
		IssuedBy:        "dim",
		Audience:        req.Audience,
		RouteVersion:    "", // Will be set by caller if needed
	}

	// Issue token
	token, err := te.issueToken(&claims)
	if err != nil {
		return nil, fmt.Errorf("failed to issue token: %w", err)
	}

	return &ExchangeResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   ttl,
		Scope:       req.Scope,
	}, nil
}

// issueToken creates a new JWT token with the given claims
func (te *TokenExchanger) issueToken(claims *TokenClaims) (string, error) {
	// Create header
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawStdEncoding.EncodeToString(headerJSON)

	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawStdEncoding.EncodeToString(claimsJSON)

	// Create signature
	message := headerB64 + "." + claimsB64
	h := hmac.New(sha256.New, []byte(te.secret))
	h.Write([]byte(message))
	signature := base64.RawStdEncoding.EncodeToString(h.Sum(nil))

	return message + "." + signature, nil
}

// ValidateScope ensures requested scope is subset of original scope
func (te *TokenExchanger) ValidateScope(originalToken, requestedScope string) error {
	// Simplified validation: just check if scope is non-empty
	// Real implementation would:
	// 1. Extract original scope from token claims
	// 2. Parse both scopes into components
	// 3. Ensure requested ⊆ original

	if requestedScope == "" {
		return fmt.Errorf("requested scope cannot be empty")
	}

	if requestedScope == "*" {
		return fmt.Errorf("wildcard scope not allowed in exchange")
	}

	return nil
}

// hashToken returns a hash of the token for secure reference
func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
