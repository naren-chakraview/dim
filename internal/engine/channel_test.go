package engine

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestChannelSendRecv verifies basic send/receive functionality
func TestChannelSendRecv(t *testing.T) {
	ch := NewChannel("test", 10)
	defer ch.Close()

	msg := NewMessage("hello", "route", "v1")
	ctx := context.Background()

	// Send and receive
	err := ch.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	received, err := ch.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}

	if received.Body != "hello" {
		t.Errorf("Message body mismatch: got %v, want hello", received.Body)
	}
}

// TestChannelBackpressure verifies buffer size limit and blocking behavior
func TestChannelBackpressure(t *testing.T) {
	bufSize := 3
	ch := NewChannel("test", bufSize)
	defer ch.Close()

	ctx := context.Background()

	// Fill the buffer
	for i := 0; i < bufSize; i++ {
		msg := NewMessage(i, "route", "v1")
		if err := ch.Send(ctx, msg); err != nil {
			t.Fatalf("Send %d failed: %v", i, err)
		}
	}

	// Next send should block (simulate with timeout)
	sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	msg := NewMessage("overflow", "route", "v1")
	err := ch.Send(sendCtx, msg)
	if err == nil {
		t.Error("Expected timeout when buffer is full, but send succeeded")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded, got %v", err)
	}

	// Recv should unblock the send
	received, _ := ch.Recv(ctx)
	if received.Body.(int) != 0 {
		t.Error("Should have received first message")
	}

	// Now the overflow message should send without blocking
	err = ch.Send(ctx, msg)
	if err != nil {
		t.Fatalf("Send after recv should succeed, got %v", err)
	}
}

// TestChannelInFlightTracking verifies message counting
func TestChannelInFlightTracking(t *testing.T) {
	ch := NewChannel("test", 10)
	defer ch.Close()

	ctx := context.Background()

	if ch.InFlight() != 0 {
		t.Errorf("InFlight should start at 0, got %d", ch.InFlight())
	}

	// Send a message (increments inFlight)
	msg := NewMessage("test", "route", "v1")
	ch.Send(ctx, msg)

	if ch.InFlight() != 1 {
		t.Errorf("InFlight should be 1 after send, got %d", ch.InFlight())
	}

	// Receive it (decrements inFlight)
	ch.Recv(ctx)

	if ch.InFlight() != 0 {
		t.Errorf("InFlight should return to 0 after recv, got %d", ch.InFlight())
	}
}

// TestChannelTryRecv verifies non-blocking receive
func TestChannelTryRecv(t *testing.T) {
	ch := NewChannel("test", 10)
	defer ch.Close()

	// Empty channel should return nil
	msg := ch.TryRecv()
	if msg != nil {
		t.Error("TryRecv on empty channel should return nil")
	}

	// Send a message
	ctx := context.Background()
	original := NewMessage("test", "route", "v1")
	ch.Send(ctx, original)

	// Now TryRecv should return it
	received := ch.TryRecv()
	if received == nil {
		t.Error("TryRecv should return message when available")
	}
	if received.Body != "test" {
		t.Errorf("Received message body mismatch: got %v, want test", received.Body)
	}
}

// TestChannelContextCancellation verifies cancellation behavior
func TestChannelContextCancellation(t *testing.T) {
	ch := NewChannel("test", 1)
	defer ch.Close()

	ctx, cancel := context.WithCancel(context.Background())

	// Fill the buffer
	msg := NewMessage("test", "route", "v1")
	ch.Send(ctx, msg)

	// Try to send with cancelled context
	cancel()
	err := ch.Send(ctx, msg)
	if err != context.Canceled {
		t.Errorf("Expected Canceled error, got %v", err)
	}
}

// TestChannelConcurrentSendRecv verifies thread-safe concurrent operations
func TestChannelConcurrentSendRecv(t *testing.T) {
	ch := NewChannel("test", 100)
	defer ch.Close()

	ctx := context.Background()
	const numMessages = 1000

	var wg sync.WaitGroup
	received := 0
	var mu sync.Mutex

	// Sender goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg := NewMessage(i, "route", "v1")
			if err := ch.Send(ctx, msg); err != nil {
				t.Errorf("Send failed: %v", err)
			}
		}
	}()

	// Receiver goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			if msg, err := ch.Recv(ctx); err == nil && msg != nil {
				mu.Lock()
				received++
				mu.Unlock()
			}
		}
	}()

	wg.Wait()

	if received != numMessages {
		t.Errorf("Expected %d messages received, got %d", numMessages, received)
	}
}

// TestChannelNilMessage verifies nil messages are silently ignored
func TestChannelNilMessage(t *testing.T) {
	ch := NewChannel("test", 10)
	defer ch.Close()

	ctx := context.Background()

	// Sending nil should not error
	err := ch.Send(ctx, nil)
	if err != nil {
		t.Errorf("Send(nil) should not error, got %v", err)
	}

	// Channel should remain empty
	if ch.InFlight() != 0 {
		t.Error("Send(nil) should not increment InFlight")
	}
}
