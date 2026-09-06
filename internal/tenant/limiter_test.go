package tenant

import (
	"testing"
	"time"
)

// TestMessageRateLimiterAllow verifies message rate limiting (M3.4.2)
func TestMessageRateLimiterAllow(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "acme-corp",
		MessageRateLimit: 10,
	})

	limiter := NewMessageRateLimiter(manager)

	// Allow up to 10 messages
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.AllowMessage("acme-corp")
		if !allowed {
			t.Errorf("Message %d should be allowed (within limit)", i+1)
		}
	}

	// 11th message should be rejected
	allowed, _ := limiter.AllowMessage("acme-corp")
	if allowed {
		t.Error("11th message should be rejected (exceeds limit)")
	}
}

// TestMessageRateLimiterUnlimited verifies unlimited domains (M3.4.2)
func TestMessageRateLimiterUnlimited(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "unlimited-domain",
		MessageRateLimit: 0, // Unlimited
	})

	limiter := NewMessageRateLimiter(manager)

	// All messages should be allowed
	for i := 0; i < 1000; i++ {
		allowed, _ := limiter.AllowMessage("unlimited-domain")
		if !allowed {
			t.Errorf("Message %d should be allowed (unlimited)", i+1)
		}
	}
}

// TestMessageRateLimiterWindowReset verifies window resets after 1 second (M3.4.2)
func TestMessageRateLimiterWindowReset(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "acme-corp",
		MessageRateLimit: 5,
	})

	limiter := NewMessageRateLimiter(manager)

	// Use up 5 messages
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.AllowMessage("acme-corp")
		if !allowed {
			t.Fatalf("Message %d should be allowed", i+1)
		}
	}

	// 6th message rejected
	allowed, _ := limiter.AllowMessage("acme-corp")
	if allowed {
		t.Error("6th message should be rejected")
	}

	// Wait for window reset
	time.Sleep(1100 * time.Millisecond)

	// Now message should be allowed
	allowed, _ = limiter.AllowMessage("acme-corp")
	if !allowed {
		t.Error("Message after window reset should be allowed")
	}
}

// TestMessageRateLimiterMultipleDomains verifies independent limits (M3.4.2)
func TestMessageRateLimiterMultipleDomains(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "domain-a",
		MessageRateLimit: 10,
	})
	manager.RegisterTenant(&Config{
		Name:             "domain-b",
		MessageRateLimit: 5,
	})

	limiter := NewMessageRateLimiter(manager)

	// Domain A allows 10
	for i := 0; i < 10; i++ {
		allowed, _ := limiter.AllowMessage("domain-a")
		if !allowed {
			t.Fatalf("Domain A message %d should be allowed", i+1)
		}
	}

	// Domain A blocks 11th
	allowed, _ := limiter.AllowMessage("domain-a")
	if allowed {
		t.Error("Domain A 11th message should be rejected")
	}

	// Domain B allows 5 (independent)
	for i := 0; i < 5; i++ {
		allowed, _ := limiter.AllowMessage("domain-b")
		if !allowed {
			t.Fatalf("Domain B message %d should be allowed", i+1)
		}
	}

	// Domain B blocks 6th
	allowed, _ = limiter.AllowMessage("domain-b")
	if allowed {
		t.Error("Domain B 6th message should be rejected")
	}
}

// TestWorkerSlotManagerAcquireRelease verifies slot acquisition (M3.4.2)
func TestWorkerSlotManagerAcquireRelease(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:        "acme-corp",
		WorkerSlots: 3,
	})

	slots := NewWorkerSlotManager(manager)

	// Acquire 3 slots
	for i := 0; i < 3; i++ {
		acquired := slots.AcquireSlot("acme-corp")
		if !acquired {
			t.Fatalf("Slot %d should be acquired", i+1)
		}
	}

	// 4th slot should fail
	acquired := slots.AcquireSlot("acme-corp")
	if acquired {
		t.Error("4th slot should not be acquired (quota exceeded)")
	}

	// Release one slot
	slots.ReleaseSlot("acme-corp")

	// Now 4th slot should succeed
	acquired = slots.AcquireSlot("acme-corp")
	if !acquired {
		t.Error("Slot should be acquired after release")
	}
}

// TestWorkerSlotManagerUnlimited verifies unlimited slots (M3.4.2)
func TestWorkerSlotManagerUnlimited(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:        "unlimited",
		WorkerSlots: 0, // Unlimited
	})

	slots := NewWorkerSlotManager(manager)

	// All slots should be acquirable
	for i := 0; i < 1000; i++ {
		acquired := slots.AcquireSlot("unlimited")
		if !acquired {
			t.Fatalf("Slot %d should be acquirable (unlimited)", i+1)
		}
	}
}

// TestWorkerSlotManagerQueueTracking verifies queue count (M3.4.2)
func TestWorkerSlotManagerQueueTracking(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:        "acme-corp",
		WorkerSlots: 2,
	})

	slots := NewWorkerSlotManager(manager)

	// Acquire 2 slots
	slots.AcquireSlot("acme-corp")
	slots.AcquireSlot("acme-corp")

	// Try to acquire 3 more (should queue)
	for i := 0; i < 3; i++ {
		slots.AcquireSlot("acme-corp") // Queue them
	}

	used, queued := slots.GetSlotUsage("acme-corp")
	if used != 2 {
		t.Errorf("Expected 2 used slots, got %d", used)
	}

	if queued != 3 {
		t.Errorf("Expected 3 queued, got %d", queued)
	}
}

// TestWorkerSlotManagerMultipleDomains verifies independent slot allocation (M3.4.2)
func TestWorkerSlotManagerMultipleDomains(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:        "domain-a",
		WorkerSlots: 10,
	})
	manager.RegisterTenant(&Config{
		Name:        "domain-b",
		WorkerSlots: 5,
	})

	slots := NewWorkerSlotManager(manager)

	// Domain A can acquire 10
	for i := 0; i < 10; i++ {
		if !slots.AcquireSlot("domain-a") {
			t.Fatalf("Domain A slot %d should be acquired", i+1)
		}
	}

	// Domain A 11th fails
	if slots.AcquireSlot("domain-a") {
		t.Error("Domain A 11th slot should fail")
	}

	// Domain B independent: can acquire 5
	for i := 0; i < 5; i++ {
		if !slots.AcquireSlot("domain-b") {
			t.Fatalf("Domain B slot %d should be acquired", i+1)
		}
	}

	// Domain B 6th fails
	if slots.AcquireSlot("domain-b") {
		t.Error("Domain B 6th slot should fail")
	}

	// Release from A doesn't affect B
	slots.ReleaseSlot("domain-a")

	usedA, _ := slots.GetSlotUsage("domain-a")
	if usedA != 9 {
		t.Errorf("Expected 9 slots used in A after release, got %d", usedA)
	}

	usedB, _ := slots.GetSlotUsage("domain-b")
	if usedB != 5 {
		t.Errorf("Expected 5 slots used in B (unchanged), got %d", usedB)
	}
}
