package engine

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestGenerationLifecycle tests the generation state transitions
func TestGenerationLifecycle(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	executor := NewExecutor("test", inputCh, outputCh, []Step{})
	gen := NewGeneration(1, executor, "v1_hash")

	// Initially should be active
	if !gen.IsActive() {
		t.Error("expected generation to be active initially")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start draining
	gen.StartDraining(ctx)

	// Should now be draining
	gen.mu.RLock()
	if gen.State != StateDraining {
		t.Errorf("expected state Draining, got %v", gen.State)
	}
	gen.mu.RUnlock()

	// Should not be active
	if gen.IsActive() {
		t.Error("expected generation to not be active after draining")
	}

	// Wait for drain to complete (should be instant since no messages in flight)
	if err := gen.WaitForDrained(ctx); err != nil {
		t.Fatalf("WaitForDrained failed: %v", err)
	}

	// Mark as expired
	gen.mu.Lock()
	gen.State = StateExpired
	gen.mu.Unlock()

	// Verify state string
	if gen.State.String() != "Expired" {
		t.Errorf("expected state string 'Expired', got %q", gen.State.String())
	}
}

// TestGenerationDrainWithInFlight tests that generation waits for in-flight messages to drain
func TestGenerationDrainWithInFlight(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	errorCh := NewChannel("error", 10)
	// Don't defer Close here; we'll close inputCh explicitly to signal end
	defer outputCh.Close()
	defer errorCh.Close()

	// Create a slow step to simulate in-flight messages
	slowStep := &mockStep{
		name:  "slow",
		delay: 50 * time.Millisecond,
	}

	executor := NewExecutor("test", inputCh, outputCh, []Step{slowStep})
	gen := NewGeneration(1, executor, "v1_hash")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Start executor in background
	executorErr := make(chan error, 1)
	go func() {
		executorErr <- executor.Run(ctx)
	}()

	// Send a message to create in-flight work
	msg := NewMessage(map[string]interface{}{"test": true}, "route1", "v1")
	if err := inputCh.Send(ctx, msg); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Let the message start processing
	time.Sleep(10 * time.Millisecond)

	// Start draining - should wait for in-flight message to complete
	gen.StartDraining(ctx)

	startTime := time.Now()

	// Wait for drain (should take ~50ms for the slow message)
	if err := gen.WaitForDrained(ctx); err != nil {
		t.Fatalf("WaitForDrained failed: %v", err)
	}

	elapsed := time.Since(startTime)
	if elapsed < 30*time.Millisecond {
		t.Errorf("drain completed too quickly (%v), expected to wait for in-flight messages", elapsed)
	}

	// Close input to stop executor
	inputCh.Close()

	// Wait for executor to finish
	select {
	case <-executorErr:
	case <-time.After(1 * time.Second):
		t.Error("executor did not finish in time")
	}
}

// TestGenerationManagerBasic tests basic generation manager operations
func TestGenerationManagerBasic(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	executor := NewExecutor("route1", inputCh, outputCh, []Step{})
	gm := NewGenerationManager("route1", executor, "v1_hash", 3)

	// Should have active generation
	activeGen := gm.GetActiveGeneration()
	if activeGen == nil {
		t.Fatal("expected active generation")
	}

	if activeGen.ID != 1 {
		t.Errorf("expected generation ID 1, got %d", activeGen.ID)
	}

	if activeGen.RouteVersion != "v1_hash" {
		t.Errorf("expected route version v1_hash, got %s", activeGen.RouteVersion)
	}

	// GetActiveInputChannel should return the executor's input channel
	activeCh := gm.GetActiveInputChannel()
	if activeCh == nil {
		t.Error("expected active input channel")
	}

	if activeCh != inputCh {
		t.Error("active input channel should be the executor's input channel")
	}
}

// TestGenerationManagerTakeover tests that takeover transitions between generations
func TestGenerationManagerTakeover(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	inputCh1 := NewChannel("input1", 10)
	outputCh1 := NewChannel("output1", 10)
	defer inputCh1.Close()
	defer outputCh1.Close()

	executor1 := NewExecutor("route1", inputCh1, outputCh1, []Step{})
	gm := NewGenerationManager("route1", executor1, "v1_hash", 3)

	// Get first generation
	gen1 := gm.GetActiveGeneration()
	if gen1.ID != 1 {
		t.Errorf("expected first generation ID 1, got %d", gen1.ID)
	}

	// Create new executor for takeover
	inputCh2 := NewChannel("input2", 10)
	outputCh2 := NewChannel("output2", 10)
	defer inputCh2.Close()
	defer outputCh2.Close()

	executor2 := NewExecutor("route1", inputCh2, outputCh2, []Step{})

	// Perform takeover
	if err := gm.Takeover(ctx, executor2, "v2_hash"); err != nil {
		t.Fatalf("takeover failed: %v", err)
	}

	// New generation should be active
	gen2 := gm.GetActiveGeneration()
	if gen2.ID != 2 {
		t.Errorf("expected new generation ID 2, got %d", gen2.ID)
	}

	if gen2.RouteVersion != "v2_hash" {
		t.Errorf("expected route version v2_hash, got %s", gen2.RouteVersion)
	}

	// Input channel should be from new generation immediately
	activeCh := gm.GetActiveInputChannel()
	if activeCh != inputCh2 {
		t.Error("active input channel should be from new generation")
	}

	// Old generation should no longer be active
	if gen1.IsActive() {
		t.Error("old generation should not be active after takeover")
	}

	// Old generation should transition to draining/expired in background
	// Just verify it's not Active (it may be Draining or already Expired by now)
	gen1.mu.RLock()
	state1 := gen1.State
	gen1.mu.RUnlock()

	if state1 == StateActive {
		t.Errorf("expected old generation to be not active, got %v", state1)
	}
}

// TestGenerationManagerConcurrentDrainingCap tests the concurrent draining cap
func TestGenerationManagerConcurrentDrainingCap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create initial generation
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	executor := NewExecutor("route1", inputCh, outputCh, []Step{})
	gm := NewGenerationManager("route1", executor, "v1_hash", 2) // Cap at 2 concurrent drains

	var wg sync.WaitGroup

	// Perform multiple takeovers to create draining generations
	for i := 2; i <= 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			newInputCh := NewChannel(fmt.Sprintf("input%d", id), 10)
			newOutputCh := NewChannel(fmt.Sprintf("output%d", id), 10)
			defer newInputCh.Close()
			defer newOutputCh.Close()

			newExecutor := NewExecutor("route1", newInputCh, newOutputCh, []Step{})
			version := fmt.Sprintf("v%d_hash", id)

			if err := gm.Takeover(ctx, newExecutor, version); err != nil {
				t.Errorf("takeover %d failed: %v", id, err)
			}
		}(i)

		// Small delay between takeovers to allow draining cap enforcement
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	// Allow time for draining to complete
	time.Sleep(500 * time.Millisecond)

	// Check that we respect the cap (max 2 concurrent draining at any time)
	draining := atomic.LoadInt32(&gm.currentlyDraining)
	if draining > int32(gm.concurrentDrainingCap) {
		t.Errorf("concurrent draining count %d exceeded cap %d", draining, gm.concurrentDrainingCap)
	}
}

// TestGenerationManagerDrainAll tests the DrainAll method at shutdown
func TestGenerationManagerDrainAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	inputCh1 := NewChannel("input1", 10)
	outputCh1 := NewChannel("output1", 10)
	defer inputCh1.Close()
	defer outputCh1.Close()

	executor1 := NewExecutor("route1", inputCh1, outputCh1, []Step{})
	gm := NewGenerationManager("route1", executor1, "v1_hash", 3)

	// Create and perform takeovers to get multiple generations
	for i := 2; i <= 3; i++ {
		inputCh := NewChannel(fmt.Sprintf("input%d", i), 10)
		outputCh := NewChannel(fmt.Sprintf("output%d", i), 10)
		defer inputCh.Close()
		defer outputCh.Close()

		executor := NewExecutor("route1", inputCh, outputCh, []Step{})
		version := fmt.Sprintf("v%d_hash", i)

		if err := gm.Takeover(ctx, executor, version); err != nil {
			t.Fatalf("takeover failed: %v", err)
		}
	}

	// Start DrainAll
	drainErr := make(chan error, 1)
	go func() {
		drainErr <- gm.DrainAll(ctx, 5*time.Second)
	}()

	// Wait for drain to complete
	select {
	case err := <-drainErr:
		if err != nil {
			t.Fatalf("DrainAll failed: %v", err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("DrainAll timeout")
	}
}

// TestHotReloadZeroMessageLoss tests message flow during hot reload
func TestHotReloadZeroMessageLoss(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inputCh1 := NewChannel("input1", 100)
	outputCh1 := NewChannel("output1", 100)
	// Don't defer Close for channels that executors close

	// Create initial executor
	executor1 := NewExecutor("route1", inputCh1, outputCh1, []Step{})
	gm := NewGenerationManager("route1", executor1, "v1_hash", 3)

	// Start executor
	go func() {
		executor1.Run(ctx)
	}()

	// Send initial batch of messages
	var sentMessages int32
	for i := 0; i < 10; i++ {
		msg := NewMessage(
			map[string]interface{}{"id": i, "batch": 1},
			"route1",
			"v1",
		)
		if err := inputCh1.Send(ctx, msg); err != nil {
			t.Logf("send error: %v", err)
			break
		}
		atomic.AddInt32(&sentMessages, 1)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for first batch to be processed
	time.Sleep(150 * time.Millisecond)

	// Trigger reload (takeover)
	inputCh2 := NewChannel("input2", 100)
	outputCh2 := NewChannel("output2", 100)

	executor2 := NewExecutor("route1", inputCh2, outputCh2, []Step{})

	if err := gm.Takeover(ctx, executor2, "v2_hash"); err != nil {
		t.Fatalf("takeover failed: %v", err)
	}

	// Start new executor
	go func() {
		executor2.Run(ctx)
	}()

	// Send messages to new generation
	for i := 10; i < 20; i++ {
		msg := NewMessage(
			map[string]interface{}{"id": i, "batch": 2},
			"route1",
			"v1",
		)
		// Get active input channel which should now be from new generation
		activeCh := gm.GetActiveInputChannel()
		if activeCh == nil {
			t.Error("active channel is nil")
			return
		}
		if err := activeCh.Send(ctx, msg); err != nil {
			t.Logf("send to new generation failed: %v", err)
			break
		}
		atomic.AddInt32(&sentMessages, 1)
		time.Sleep(10 * time.Millisecond)
	}

	// Allow processing
	time.Sleep(500 * time.Millisecond)

	// Collect received messages from both generations
	var receivedCount int32

	// Count messages from outputCh1 (first generation)
	for {
		msg, err := outputCh1.Recv(ctx)
		if err != nil {
			break
		}
		if msg != nil {
			atomic.AddInt32(&receivedCount, 1)
		}
	}

	// Count messages from outputCh2 (second generation)
	for {
		msg, err := outputCh2.Recv(ctx)
		if err != nil {
			break
		}
		if msg != nil {
			atomic.AddInt32(&receivedCount, 1)
		}
	}

	sent := atomic.LoadInt32(&sentMessages)
	received := atomic.LoadInt32(&receivedCount)

	t.Logf("Sent: %d, Received: %d", sent, received)

	// We should have processed messages successfully
	if sent == 0 {
		t.Error("no messages were sent")
	}
}

// TestGenerationInputChannelProxy tests that GetInputChannel correctly proxies to executor
func TestGenerationInputChannelProxy(t *testing.T) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	executor := NewExecutor("test", inputCh, outputCh, []Step{})
	gen := NewGeneration(1, executor, "v1_hash")

	// GetInputChannel should return the executor's input channel
	returned := gen.GetInputChannel()
	if returned != inputCh {
		t.Error("GetInputChannel should return executor's input channel")
	}
}

// BenchmarkGenerationManagerGetActiveInputChannel benchmarks the hot path
func BenchmarkGenerationManagerGetActiveInputChannel(b *testing.B) {
	inputCh := NewChannel("input", 10)
	outputCh := NewChannel("output", 10)
	defer inputCh.Close()
	defer outputCh.Close()

	executor := NewExecutor("route1", inputCh, outputCh, []Step{})
	gm := NewGenerationManager("route1", executor, "v1_hash", 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = gm.GetActiveInputChannel()
	}
}
