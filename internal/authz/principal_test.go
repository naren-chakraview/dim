package authz

import (
	"testing"
)

func TestPrincipalCreation(t *testing.T) {
	attrs := map[string]string{
		"org":  "acme",
		"site": "us-west",
	}

	principal := &Principal{
		Subject:    "user-123",
		Roles:      []string{"admin", "viewer"},
		Attributes: attrs,
	}

	if principal.Subject != "user-123" {
		t.Errorf("Subject = %s, want user-123", principal.Subject)
	}

	if len(principal.Roles) != 2 {
		t.Errorf("Roles length = %d, want 2", len(principal.Roles))
	}

	if principal.Attributes["org"] != "acme" {
		t.Errorf("org attribute = %s, want acme", principal.Attributes["org"])
	}
}

func TestPrincipalWithoutRoles(t *testing.T) {
	principal := &Principal{
		Subject: "service-account",
		Roles:   []string{},
	}

	if principal.Subject != "service-account" {
		t.Errorf("Subject = %s, want service-account", principal.Subject)
	}

	if len(principal.Roles) != 0 {
		t.Errorf("Roles length = %d, want 0", len(principal.Roles))
	}
}

func TestPrincipalWithoutAttributes(t *testing.T) {
	principal := &Principal{
		Subject: "user-456",
		Roles:   []string{"user"},
	}

	if principal.Attributes != nil && len(principal.Attributes) > 0 {
		t.Error("Attributes should be empty")
	}
}
