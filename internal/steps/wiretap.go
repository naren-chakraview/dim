package steps

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// WiretapStep implements non-blocking message duplication to a secondary sink.
// It copies messages to a tap sink without blocking the main flow.
// If the tap fails, the error is logged but doesn't fail the main message processing.
type WiretapStep struct {
	sinkName    string           // name of sink to tap messages to
	sinkChannel *engine.Channel  // reference to actual sink channel, set during pipeline build
}

// NewWiretapStep creates a new wiretap step that will duplicate messages to the named sink.
// The actual sink channel is wired up later via SetTapSink during pipeline build.
func NewWiretapStep(sinkName string) (*WiretapStep, error) {
	if sinkName == "" {
		return nil, fmt.Errorf("wiretap step: sink name cannot be empty")
	}

	return &WiretapStep{
		sinkName: sinkName,
	}, nil
}

// SetTapSink wires up the actual sink channel for this wiretap step.
// This is called during pipeline build to connect the named sink.
func (ws *WiretapStep) SetTapSink(ch *engine.Channel) {
	ws.sinkChannel = ch
}

// Execute copies the message to the tap sink without altering the main flow.
// Returns the original message unchanged.
// If tap send fails, logs the error but doesn't fail the main message.
//
// Non-blocking: Uses channel.TryRecv() equivalent logic or spawns goroutine for non-blocking send.
// For simplicity in Phase 0, we use a non-blocking select on the channel.
// If channel buffer is full, we log and continue (non-blocking by design).
func (ws *WiretapStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, nil
	}

	// If tap sink is not configured, just pass through
	if ws.sinkChannel == nil {
		log.Printf("[WARN] Wiretap step: sink channel not configured for sink %q", ws.sinkName)
		return msg, nil
	}

	// Deep copy the message
	msgCopy := msg.Copy()
	if msgCopy == nil {
		// If copy fails somehow, just log and continue with original
		log.Printf("[WARN] Wiretap step: failed to copy message for sink %q", ws.sinkName)
		return msg, nil
	}

	// Try to send the copy to the tap sink in a non-blocking way.
	// We spawn a goroutine to attempt the send so we don't block the main flow.
	go func() {
		// Use a timeout for the tap send to avoid hanging forever
		// If the original context has a deadline, use a fraction of that time
		var tapCtx context.Context
		var cancel context.CancelFunc

		if deadline, ok := ctx.Deadline(); ok {
			timeout := time.Until(deadline) / 2
			if timeout < 100*time.Millisecond {
				timeout = 100 * time.Millisecond
			}
			tapCtx, cancel = context.WithTimeout(context.Background(), timeout)
		} else {
			// Default timeout of 5 seconds if no deadline is set
			tapCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		}
		defer cancel()

		if err := ws.sinkChannel.Send(tapCtx, msgCopy); err != nil {
			log.Printf("[WARN] Wiretap step: failed to send copy to sink %q: %v", ws.sinkName, err)
			// Don't fail the main message; just log
			return
		}
	}()

	// Return the original message unchanged
	return msg, nil
}

