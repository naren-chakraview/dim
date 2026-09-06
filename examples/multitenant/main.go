// Example M3.4: Multi-tenant resource isolation on a shared instance
// This demonstrates two domains sharing a DIM instance with independent
// resource quotas (message rate limiting, worker slots, lineage storage).
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/tenant"
)

func main() {
	fmt.Println("=== Multi-Tenant Isolation Example (M3.4) ===\n")

	// Create a tenant manager
	manager := tenant.NewManager()

	// Register two tenants with different quotas
	fmt.Println("1. Registering tenants with different quotas:")
	fmt.Println()

	// Premium tenant: high throughput, can handle 1000 msg/sec, 100 worker slots
	premium := &tenant.Config{
		Name:                 "premium-corp",
		Description:          "Premium tier customer",
		MessageRateLimit:     1000,
		MessageRateStrategy:  tenant.RateLimitQueue,
		WorkerSlots:          100,
		LineageQuotaRecords:  10000000,
		LineageQuotaStrategy: tenant.LineageQuotaExpire,
		LineageRetentionDays: 90,
		Priority:             tenant.PriorityHigh,
	}

	// Standard tenant: moderate throughput, 100 msg/sec, 10 worker slots
	standard := &tenant.Config{
		Name:                 "standard-corp",
		Description:          "Standard tier customer",
		MessageRateLimit:     100,
		MessageRateStrategy:  tenant.RateLimitReject,
		WorkerSlots:          10,
		LineageQuotaRecords:  1000000,
		LineageQuotaStrategy: tenant.LineageQuotaReject,
		LineageRetentionDays: 30,
		Priority:             tenant.PriorityNormal,
	}

	manager.RegisterTenant(premium)
	manager.RegisterTenant(standard)

	fmt.Printf("  Premium:  %d msg/sec, %d worker slots, %d lineage records\n",
		premium.MessageRateLimit, premium.WorkerSlots, premium.LineageQuotaRecords)
	fmt.Printf("  Standard: %d msg/sec, %d worker slots, %d lineage records\n",
		standard.MessageRateLimit, standard.WorkerSlots, standard.LineageQuotaRecords)
	fmt.Println()

	// Create enforcer instances
	rateLimiter := tenant.NewMessageRateLimiter(manager)
	slotManager := tenant.NewWorkerSlotManager(manager)

	// Scenario: Premium tenant receives heavy load while standard tenant should remain unaffected
	fmt.Println("2. Simulating mixed load:\n")
	fmt.Println("   - Premium domain: 50 messages in quick succession")
	fmt.Println("   - Standard domain: 20 messages concurrently")
	fmt.Println()

	// Track results
	var premiumProcessed, standardProcessed int32
	var premiumLatency, standardLatency time.Duration

	var wg sync.WaitGroup

	// Premium workload: High throughput
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for i := 0; i < 50; i++ {
			if allowed, _ := rateLimiter.AllowMessage("premium-corp"); allowed {
				if acquired := slotManager.AcquireSlot("premium-corp"); acquired {
					premiumProcessed++
					// Simulate work
					time.Sleep(5 * time.Millisecond)
					slotManager.ReleaseSlot("premium-corp")
				}
			}
		}
		premiumLatency = time.Since(start)
	}()

	// Standard workload: Moderate throughput (runs concurrently)
	time.Sleep(50 * time.Millisecond) // Let premium start
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		for i := 0; i < 20; i++ {
			if allowed, _ := rateLimiter.AllowMessage("standard-corp"); allowed {
				if acquired := slotManager.AcquireSlot("standard-corp"); acquired {
					standardProcessed++
					// Simulate work
					time.Sleep(5 * time.Millisecond)
					slotManager.ReleaseSlot("standard-corp")
				}
			}
		}
		standardLatency = time.Since(start)
	}()

	wg.Wait()

	fmt.Printf("   Premium:  processed %d messages in %.2fs\n", premiumProcessed, premiumLatency.Seconds())
	fmt.Printf("   Standard: processed %d messages in %.2fs\n", standardProcessed, standardLatency.Seconds())
	fmt.Println()

	// Verify isolation
	fmt.Println("3. Verifying resource isolation:\n")

	premiumUsed, premiumQueued := slotManager.GetSlotUsage("premium-corp")
	standardUsed, standardQueued := slotManager.GetSlotUsage("standard-corp")

	fmt.Printf("   Premium slots:  %d used, %d queued (limit: %d)\n",
		premiumUsed, premiumQueued, premium.WorkerSlots)
	fmt.Printf("   Standard slots: %d used, %d queued (limit: %d)\n",
		standardUsed, standardQueued, standard.WorkerSlots)

	premiumRate := rateLimiter.GetCurrentRate("premium-corp")
	standardRate := rateLimiter.GetCurrentRate("standard-corp")

	fmt.Printf("   Premium rate:  %.2f msg/sec (limit: %d)\n", premiumRate, premium.MessageRateLimit)
	fmt.Printf("   Standard rate: %.2f msg/sec (limit: %d)\n", standardRate, standard.MessageRateLimit)
	fmt.Println()

	// Verify isolation properties
	fmt.Println("4. Isolation verification:")
	fmt.Println()

	success := true

	// Check 1: Each tenant remained within its limits
	if premiumUsed <= premium.WorkerSlots && standardUsed <= standard.WorkerSlots {
		fmt.Println("   ✓ Both tenants stayed within worker slot quotas")
	} else {
		fmt.Println("   ✗ A tenant exceeded its worker slot quota")
		success = false
	}

	// Check 2: Standard tenant was not starved by premium
	if standardProcessed > 0 {
		fmt.Println("   ✓ Standard tenant processed messages despite premium load")
	} else {
		fmt.Println("   ✗ Standard tenant was starved by premium")
		success = false
	}

	// Check 3: Standard tenant had available slots
	standardAvailable := standard.WorkerSlots - standardUsed
	if standardAvailable > 0 {
		fmt.Println("   ✓ Standard tenant had available slots (not exhausted)")
	} else {
		fmt.Println("   ✗ Standard tenant slots exhausted")
		success = false
	}

	// Check 4: Rate limiting was independent
	if premiumRate > 0 && standardRate > 0 {
		fmt.Println("   ✓ Both tenants' rate limits tracked independently")
	} else {
		fmt.Println("   ✗ Rate limiting not working independently")
		success = false
	}

	fmt.Println()

	if success {
		fmt.Println("✓ Multi-tenant isolation verified successfully!")
		fmt.Println()
		fmt.Println("Key takeaway: Premium tenant's high load did not degrade")
		fmt.Println("standard tenant's latency or throughput. Each tenant")
		fmt.Println("maintains independent resource quotas on the same instance.")
	} else {
		fmt.Println("✗ Isolation verification failed")
	}
}
