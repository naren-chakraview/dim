package authz

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTValidator validates JWT tokens and extracts claims
type JWTValidator struct {
	hmacSecret  string
	rsaPublicKey *rsa.PublicKey
	useHMAC     bool
}

// NewJWTValidator creates a JWT validator from environment configuration
// Prefers JWT_PUBLIC_KEY (RSA) over JWT_SECRET (HMAC)
func NewJWTValidator() (*JWTValidator, error) {
	// Try RSA public key first
	publicKeyPEM := os.Getenv("JWT_PUBLIC_KEY")
	if publicKeyPEM != "" {
		rsaPub, err := parseRSAPublicKey(publicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JWT_PUBLIC_KEY: %w", err)
		}
		return &JWTValidator{
			rsaPublicKey: rsaPub,
			useHMAC:      false,
		}, nil
	}

	// Fall back to HMAC secret
	hmacSecret := os.Getenv("JWT_SECRET")
	if hmacSecret == "" {
		return nil, fmt.Errorf("neither JWT_PUBLIC_KEY nor JWT_SECRET is set")
	}

	return &JWTValidator{
		hmacSecret: hmacSecret,
		useHMAC:    true,
	}, nil
}

// NewJWTValidatorWithSecret creates a JWT validator with the given HMAC secret (for testing)
func NewJWTValidatorWithSecret(secret string) *JWTValidator {
	return &JWTValidator{
		hmacSecret: secret,
		useHMAC:    true,
	}
}

// ValidateToken parses and validates a JWT token, returning a Principal
// or error. Token should not include "Bearer " prefix.
func (v *JWTValidator) ValidateToken(tokenString string) (*Principal, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	var claims jwt.MapClaims
	var token *jwt.Token
	var err error

	if v.useHMAC {
		token, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(v.hmacSecret), nil
		})
	} else {
		token, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return v.rsaPublicKey, nil
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract subject claim
	subject, ok := claims["sub"].(string)
	if !ok || subject == "" {
		return nil, fmt.Errorf("subject claim (sub) not found or invalid")
	}

	// Extract roles claim (optional, can be array or string)
	var roles []string
	if rolesRaw, ok := claims["roles"]; ok {
		switch v := rolesRaw.(type) {
		case []interface{}:
			for _, r := range v {
				if roleStr, ok := r.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		case string:
			roles = []string{v}
		}
	}

	// Extract custom attributes from all claims except standard ones
	attributes := make(map[string]string)
	standardClaims := map[string]bool{
		"sub": true, "roles": true, "exp": true, "iat": true, "nbf": true,
		"iss": true, "aud": true, "jti": true,
	}

	for key, val := range claims {
		if !standardClaims[key] {
			// Convert value to string
			if strVal, ok := val.(string); ok {
				attributes[key] = strVal
			} else {
				attributes[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	return &Principal{
		Subject:    subject,
		Roles:      roles,
		Attributes: attributes,
	}, nil
}

// ExtractBearerToken extracts the token from an Authorization header
// Returns the token (without "Bearer " prefix) or empty string if not found
func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// parseRSAPublicKey parses an RSA public key from PEM format
func parseRSAPublicKey(keyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(keyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

// ValidateTokenWithKey validates a JWT with a specific key (for testing)
// useHMAC determines whether the key is HMAC secret or RSA public key
func ValidateTokenWithKey(tokenString string, key interface{}, useHMAC bool) (*Principal, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	var claims jwt.MapClaims
	var token *jwt.Token
	var err error

	if useHMAC {
		token, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return key, nil
		})
	} else {
		token, err = jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return key, nil
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract subject claim
	subject, ok := claims["sub"].(string)
	if !ok || subject == "" {
		return nil, fmt.Errorf("subject claim (sub) not found or invalid")
	}

	// Extract roles claim (optional)
	var roles []string
	if rolesRaw, ok := claims["roles"]; ok {
		switch v := rolesRaw.(type) {
		case []interface{}:
			for _, r := range v {
				if roleStr, ok := r.(string); ok {
					roles = append(roles, roleStr)
				}
			}
		case string:
			roles = []string{v}
		}
	}

	// Extract custom attributes
	attributes := make(map[string]string)
	standardClaims := map[string]bool{
		"sub": true, "roles": true, "exp": true, "iat": true, "nbf": true,
		"iss": true, "aud": true, "jti": true,
	}

	for key, val := range claims {
		if !standardClaims[key] {
			if strVal, ok := val.(string); ok {
				attributes[key] = strVal
			} else {
				attributes[key] = fmt.Sprintf("%v", val)
			}
		}
	}

	return &Principal{
		Subject:    subject,
		Roles:      roles,
		Attributes: attributes,
	}, nil
}

// ExtractPrincipalFromHeader validates a bearer token from Authorization header
// Returns nil principal if no auth header present
// Returns error if auth header is malformed or token is invalid
func ExtractPrincipalFromHeader(authHeader string, validator *JWTValidator) (*Principal, error) {
	token := ExtractBearerToken(authHeader)
	if token == "" {
		// No token present is not an error; principal is just nil
		return nil, nil
	}

	principal, err := validator.ValidateToken(token)
	if err != nil {
		// Token present but invalid is an error
		log.Printf("JWT validation failed: %v", err)
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	return principal, nil
}

// CreateTestToken creates a JWT for testing purposes using HMAC
// This is a helper function not meant for production use
func CreateTestToken(subject string, roles []string, attributes map[string]string, secret string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(expiresIn).Unix(),
	}

	if len(roles) > 0 {
		claims["roles"] = roles
	}

	for key, val := range attributes {
		claims[key] = val
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
