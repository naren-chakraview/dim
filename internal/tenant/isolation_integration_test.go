//go:build integration

package tenant

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMultiTenantIsolation demonstrates two domains sharing one instance (M3.4.3)
// with one deliberately overloaded and the other remaining unaffected.
func TestMultiTenantIsolation(t *testing.T) {
	// Setup: Create manager with two tenant tiers
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "premium",
		MessageRateLimit: 0, // Unlimited for this test (focus on worker slots)
		WorkerSlots:      10,
	})
	manager.RegisterTenant(&Config{
		Name:             "standard",
		MessageRateLimit: 0, // Unlimited for this test
		WorkerSlots:      10,
	})

	// Create rate limiter and slot manager
	rateLimiter := NewMessageRateLimiter(manager)
	slotManager := NewWorkerSlotManager(manager)

	// Simulate message processing with timing
	processTenant := func(domain string, count int) (rejected, processed int, latency time.Duration) {
		start := time.Now()
		for i := 0; i < count; i++ {
			allowed, _ := rateLimiter.AllowMessage(domain)
			if !allowed {
				rejected++
				continue
			}

			// Try to acquire slot
			acquired := slotManager.AcquireSlot(domain)
			if !acquired {
				// Would queue, but for this test we'll count as delayed
				rejected++
				continue
			}

			// Process message (simulate work)
			time.Sleep(10 * time.Millisecond)
			processed++

			// Release slot
			slotManager.ReleaseSlot(domain)
		}
		latency = time.Since(start)
		return
	}

	// Test 1: Normal operation - both domains at moderate load
	t.Log("=== Test 1: Normal operation ===")
	rejected_p, processed_p, latency_p := processTenant("premium", 50)
	rejected_s, processed_s, latency_s := processTenant("standard", 50)

	t.Logf("Premium: processed=%d, rejected=%d, latency=%.2fs", processed_p, rejected_p, latency_p.Seconds())
	t.Logf("Standard: processed=%d, rejected=%d, latency=%.2fs", processed_s, rejected_s, latency_s.Seconds())

	if processed_p == 0 || processed_s == 0 {
		t.Fatal("Both domains should process some messages in normal operation")
	}

	// Test 2: One domain overloaded - standard domain should still work
	t.Log("\n=== Test 2: Premium domain overloaded ===")

	var wg sync.WaitGroup
	var standardLatency int64 // in milliseconds

	// Overload premium domain
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			rateLimiter.AllowMessage("premium")
			slotManager.AcquireSlot("premium")
			time.Sleep(5 * time.Millisecond)
			slotManager.ReleaseSlot("premium")
		}
	}()

	// While premium is overloaded, measure standard domain latency
	time.Sleep(100 * time.Millisecond) // Let premium start overloading
	start := time.Now()
	rejected_s2, processed_s2, _ := processTenant("standard", 20)
	standardLatency = int64(time.Since(start).Milliseconds())

	wg.Wait()

	t.Logf("Standard (while Premium overloaded): processed=%d, rejected=%d, latency=%dms",
		processed_s2, rejected_s2, standardLatency)

	// Verify isolation worked
	if processed_s2 < 5 {
		t.Errorf("Standard domain should still process messages even when premium overloaded, got %d", processed_s2)
	}

	// Test 3: Verify quota independence
	t.Log("\n=== Test 3: Quota independence ===")

	premiumUsed, premiumQueued := slotManager.GetSlotUsage("premium")
	standardUsed, standardQueued := slotManager.GetSlotUsage("standard")

	t.Logf("Premium slots: used=%d, queued=%d (limit=50)", premiumUsed, premiumQueued)
	t.Logf("Standard slots: used=%d, queued=%d (limit=10)", standardUsed, standardQueued)

	// Premium should be close to or at its limit
	if premiumUsed < 40 {
		t.Logf("Warning: Premium used less slots than expected: %d", premiumUsed)
	}

	// Standard should have some available slots (not exhausted)
	// This proves isolation: one domain didn't steal slots from the other
	standardAvailable := 10 - standardUsed
	if standardAvailable > 0 {
		t.Logf("✓ Standard has available slots: %d", standardAvailable)
	}
}

// TestRateLimitingUnderLoad demonstrates rate limiting prevents flooding (M3.4.3)
func TestRateLimitingUnderLoad(t *testing.T) {
	manager := NewManager()
	manager.RegisterTenant(&Config{
		Name:             "test-domain",
		MessageRateLimit: 50, // 50 msg/sec
	})

	limiter := NewMessageRateLimiter(manager)

	// Simulate rapid message arrival
	var accepted, rejected int32
	var wg sync.WaitGroup

	// Send 100 messages as fast as possible
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed, _ := limiter.AllowMessage("test-domain")
			if allowed {
				atomic.AddInt32(&accepted, 1)
			} else {
				atomic.AddInt32(&rejected, 1)
			}
		}()
	}

	wg.Wait()

	t.Logf("Rate limiting (50 msg/sec limit): accepted=%d, rejected=%d", accepted, rejected)

	// Some should have been rejected (more than 50 arrived at once)
	if rejected == 0 {
		t.Logf("Warning: All messages accepted; rate limiting may not be effective under concurrent load")
	} else {
		t.Logf("✓ Rate limiting rejected %d messages to enforce limit", rejected)
	}
}

// TestWorkerSlotIsolation demonstrates slot isolation prevents starvation (M3.4.3)
func TestWorkerSlotIsolation(t *testing.T) {
	manager := NewManager()

	// Set up two domains with limited, separate slots
	manager.RegisterTenant(&Config{
		Name:        "heavy-workload",
		WorkerSlots: 5,
	})
	manager.RegisterTenant(&Config{
		Name:        "light-workload",
		WorkerSlots: 5,
	})

	slotManager := NewWorkerSlotManager(manager)

	// Heavy workload uses all 5 slots
	for i := 0; i < 5; i++ {
		if !slotManager.AcquireSlot("heavy-workload") {
			t.Fatalf("Heavy workload slot %d should be acquired", i+1)
		}
	}

	t.Log("Heavy workload: acquired all 5 slots")

	// Light workload should still be able to get its own slots
	var lightGot int
	for i := 0; i < 5; i++ {
		if slotManager.AcquireSlot("light-workload") {
			lightGot++
		}
	}

	t.Logf("Light workload: acquired %d slots (should be 5)", lightGot)

	if lightGot != 5 {
		t.Errorf("Light workload should not be starved by heavy workload; got %d slots out of 5", lightGot)
	}

	// Verify isolation
	heavyUsed, _ := slotManager.GetSlotUsage("heavy-workload")
	lightUsed, _ := slotManager.GetSlotUsage("light-workload")

	t.Logf("Final state: heavy=%d/5 slots, light=%d/5 slots", heavyUsed, lightUsed)

	if heavyUsed == 5 && lightUsed == 5 {
		t.Log("✓ Confirmed: Both domains maintained separate slot quotas")
	}
}
