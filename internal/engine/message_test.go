package engine

import (
	"testing"
	"time"
)

// TestMessageEnvelope verifies message structure and copy semantics
func TestMessageEnvelope(t *testing.T) {
	t.Run("string body", func(t *testing.T) {
		msg := NewMessage("hello", "test-route", "v1")
		if msg.Body != "hello" {
			t.Errorf("Body mismatch: got %v, want hello", msg.Body)
		}

		// Verify metadata
		if msg.Metadata.Route != "test-route" {
			t.Errorf("Route mismatch: got %s, want test-route", msg.Metadata.Route)
		}
		if msg.Metadata.RouteVersion != "v1" {
			t.Errorf("RouteVersion mismatch: got %s, want v1", msg.Metadata.RouteVersion)
		}
		if msg.Metadata.CorrelationID == "" {
			t.Error("CorrelationID should not be empty")
		}
		if msg.Metadata.IngestedAt.IsZero() {
			t.Error("IngestedAt should not be zero")
		}
	})

	t.Run("map body", func(t *testing.T) {
		body := map[string]interface{}{"key": "value"}
		msg := NewMessage(body, "test-route", "v1")
		// Maps must be compared by value, not ==
		bodyMap := msg.Body.(map[string]interface{})
		if bodyMap["key"] != "value" {
			t.Errorf("Map key value mismatch: got %v, want value", bodyMap["key"])
		}
	})

	t.Run("nil body", func(t *testing.T) {
		msg := NewMessage(nil, "test-route", "v1")
		if msg.Body != nil {
			t.Errorf("Body should be nil, got %v", msg.Body)
		}
	})
}

// TestMessageCopy verifies copy creates independent headers but shares metadata
func TestMessageCopy(t *testing.T) {
	original := NewMessage("test", "route", "v1")
	original.Headers["key"] = "value"
	original.Metadata.Stage = "filter"

	copy := original.Copy()

	// Headers should be independent
	copy.Headers["key"] = "modified"
	if original.Headers["key"] != "value" {
		t.Error("Original headers should not be affected by copy modifications")
	}

	// Metadata should be the same (shallow copy)
	if copy.Metadata.Route != original.Metadata.Route {
		t.Error("Copy should share metadata")
	}
	if copy.Metadata.CorrelationID != original.Metadata.CorrelationID {
		t.Error("Copy should have same CorrelationID")
	}
}

// TestMessageNilCopy verifies nil-safe copy
func TestMessageNilCopy(t *testing.T) {
	var msg *Message
	copy := msg.Copy()
	if copy != nil {
		t.Error("Copy of nil message should be nil")
	}
}

// TestMessagePrincipal verifies principal attachment
func TestMessagePrincipal(t *testing.T) {
	msg := NewMessage("test", "route", "v1")
	msg.Metadata.Principal = &Principal{
		Subject: "user@example.com",
		Roles:   []string{"admin", "user"},
		Claims: map[string]interface{}{
			"iat": float64(time.Now().Unix()),
		},
	}

	if msg.Metadata.Principal.Subject != "user@example.com" {
		t.Error("Principal subject not set correctly")
	}
	if len(msg.Metadata.Principal.Roles) != 2 {
		t.Error("Principal roles not set correctly")
	}
}
