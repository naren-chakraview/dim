package authz

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testSecret = "test-secret-key-for-hmac"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "valid bearer token",
			header:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "bearer with extra spaces",
			header:   "Bearer  eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9  ",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "lowercase bearer",
			header:   "bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "wrong auth type",
			header:   "Basic eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "",
		},
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "no space separator",
			header:   "BearereyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractBearerToken(tt.header)
			if got != tt.expected {
				t.Errorf("ExtractBearerToken() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidateTokenWithHMAC(t *testing.T) {
	// Create a valid token
	token, err := CreateTestToken("user-123", []string{"admin", "operator"}, map[string]string{
		"dept": "engineering",
		"team": "platform",
	}, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	principal, err := ValidateTokenWithKey(token, []byte(testSecret), true)
	if err != nil {
		t.Fatalf("ValidateTokenWithKey failed: %v", err)
	}

	if principal.Subject != "user-123" {
		t.Errorf("Subject = %s, want user-123", principal.Subject)
	}

	if len(principal.Roles) != 2 {
		t.Errorf("Roles length = %d, want 2", len(principal.Roles))
	}

	if principal.Roles[0] != "admin" || principal.Roles[1] != "operator" {
		t.Errorf("Roles = %v, want [admin operator]", principal.Roles)
	}

	if principal.Attributes["dept"] != "engineering" {
		t.Errorf("dept attribute = %s, want engineering", principal.Attributes["dept"])
	}
}

func TestValidateTokenExpired(t *testing.T) {
	// Create an expired token
	token, err := CreateTestToken("user-123", []string{}, map[string]string{}, testSecret, -1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	_, err = ValidateTokenWithKey(token, []byte(testSecret), true)
	if err == nil {
		t.Error("ValidateTokenWithKey should fail for expired token")
	}
}

func TestValidateTokenInvalidSignature(t *testing.T) {
	token, err := CreateTestToken("user-123", []string{}, map[string]string{}, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	// Try to validate with wrong secret
	_, err = ValidateTokenWithKey(token, []byte("wrong-secret"), true)
	if err == nil {
		t.Error("ValidateTokenWithKey should fail with wrong secret")
	}
}

func TestValidateTokenMissingSubject(t *testing.T) {
	// Create token without subject claim
	claims := jwt.MapClaims{
		"roles": []string{"admin"},
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("SignedString failed: %v", err)
	}

	_, err = ValidateTokenWithKey(tokenString, []byte(testSecret), true)
	if err == nil {
		t.Error("ValidateTokenWithKey should fail when subject is missing")
	}
}

func TestValidateTokenWithRoles(t *testing.T) {
	// Test with multiple roles
	token, err := CreateTestToken("user-456", []string{"viewer", "editor", "admin"}, map[string]string{}, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	principal, err := ValidateTokenWithKey(token, []byte(testSecret), true)
	if err != nil {
		t.Fatalf("ValidateTokenWithKey failed: %v", err)
	}

	if len(principal.Roles) != 3 {
		t.Errorf("Roles length = %d, want 3", len(principal.Roles))
	}
}

func TestValidateTokenWithoutRoles(t *testing.T) {
	// Test token without roles claim
	token, err := CreateTestToken("user-789", []string{}, map[string]string{}, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	principal, err := ValidateTokenWithKey(token, []byte(testSecret), true)
	if err != nil {
		t.Fatalf("ValidateTokenWithKey failed: %v", err)
	}

	if len(principal.Roles) != 0 {
		t.Errorf("Roles length = %d, want 0", len(principal.Roles))
	}
}

func TestValidateTokenWithCustomAttributes(t *testing.T) {
	attrs := map[string]string{
		"org_id":  "org-123",
		"dept":    "sales",
		"region":  "us-west",
	}
	token, err := CreateTestToken("user-555", []string{}, attrs, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	principal, err := ValidateTokenWithKey(token, []byte(testSecret), true)
	if err != nil {
		t.Fatalf("ValidateTokenWithKey failed: %v", err)
	}

	if principal.Attributes["org_id"] != "org-123" {
		t.Errorf("org_id = %s, want org-123", principal.Attributes["org_id"])
	}
	if principal.Attributes["dept"] != "sales" {
		t.Errorf("dept = %s, want sales", principal.Attributes["dept"])
	}
	if principal.Attributes["region"] != "us-west" {
		t.Errorf("region = %s, want us-west", principal.Attributes["region"])
	}
}

func TestExtractPrincipalFromHeaderNoAuth(t *testing.T) {
	// Set up HMAC validator with test secret
	t.Setenv("JWT_SECRET", testSecret)

	validator := &JWTValidator{
		hmacSecret: testSecret,
		useHMAC:    true,
	}

	// Test with no Authorization header
	principal, err := ExtractPrincipalFromHeader("", validator)
	if err != nil {
		t.Errorf("ExtractPrincipalFromHeader should not error on missing auth: %v", err)
	}
	if principal != nil {
		t.Error("principal should be nil when no auth header")
	}
}

func TestExtractPrincipalFromHeaderValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	validator := &JWTValidator{
		hmacSecret: testSecret,
		useHMAC:    true,
	}

	token, err := CreateTestToken("user-888", []string{"admin"}, map[string]string{}, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	authHeader := "Bearer " + token
	principal, err := ExtractPrincipalFromHeader(authHeader, validator)
	if err != nil {
		t.Fatalf("ExtractPrincipalFromHeader failed: %v", err)
	}

	if principal == nil {
		t.Fatal("principal should not be nil")
	}

	if principal.Subject != "user-888" {
		t.Errorf("Subject = %s, want user-888", principal.Subject)
	}
}

func TestExtractPrincipalFromHeaderInvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", testSecret)

	validator := &JWTValidator{
		hmacSecret: testSecret,
		useHMAC:    true,
	}

	authHeader := "Bearer invalid.token.here"
	principal, err := ExtractPrincipalFromHeader(authHeader, validator)
	if err == nil {
		t.Error("ExtractPrincipalFromHeader should fail with invalid token")
	}
	if principal != nil {
		t.Error("principal should be nil on error")
	}
}

func TestCreateTestToken(t *testing.T) {
	token, err := CreateTestToken(
		"test-user",
		[]string{"role1", "role2"},
		map[string]string{"attr": "value"},
		testSecret,
		1*time.Hour,
	)
	if err != nil {
		t.Fatalf("CreateTestToken failed: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	// Verify we can parse it back
	principal, err := ValidateTokenWithKey(token, []byte(testSecret), true)
	if err != nil {
		t.Fatalf("ValidateTokenWithKey failed: %v", err)
	}

	if principal.Subject != "test-user" {
		t.Errorf("Subject = %s, want test-user", principal.Subject)
	}
}


func TestJWTValidatorInitialization(t *testing.T) {
	t.Run("with HMAC secret", func(t *testing.T) {
		t.Setenv("JWT_SECRET", testSecret)
		t.Setenv("JWT_PUBLIC_KEY", "") // Clear RSA key

		validator, err := NewJWTValidator()
		if err != nil {
			t.Fatalf("NewJWTValidator failed: %v", err)
		}

		if !validator.useHMAC {
			t.Error("useHMAC should be true")
		}

		if validator.hmacSecret != testSecret {
			t.Errorf("hmacSecret = %s, want %s", validator.hmacSecret, testSecret)
		}
	})

	t.Run("without JWT credentials", func(t *testing.T) {
		t.Setenv("JWT_SECRET", "")
		t.Setenv("JWT_PUBLIC_KEY", "")

		_, err := NewJWTValidator()
		if err == nil {
			t.Error("NewJWTValidator should fail without JWT credentials")
		}
	})
}
