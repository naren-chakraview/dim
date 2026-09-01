package engine

import (
	"context"
	"sync/atomic"
)

// Channel is a bounded, backpressure-aware message queue between pipeline stages
type Channel struct {
	name  string
	queue chan *Message
	// inFlight tracks messages currently in transit for backpressure
	inFlight int32
	maxBuffer int
}

// NewChannel creates a bounded message channel
// bufferSize determines how many messages can be queued before blocking upstream
func NewChannel(name string, bufferSize int) *Channel {
	if bufferSize < 1 {
		bufferSize = 100 // reasonable default
	}
	return &Channel{
		name:      name,
		queue:     make(chan *Message, bufferSize),
		maxBuffer: bufferSize,
	}
}

// Send enqueues a message, blocking if the channel is full (backpressure)
func (ch *Channel) Send(ctx context.Context, msg *Message) error {
	if msg == nil {
		return nil
	}
	atomic.AddInt32(&ch.inFlight, 1)
	select {
	case ch.queue <- msg:
		return nil
	case <-ctx.Done():
		atomic.AddInt32(&ch.inFlight, -1)
		return ctx.Err()
	}
}

// Recv dequeues the next message, blocking if empty
// Returns nil if the channel is closed and empty
func (ch *Channel) Recv(ctx context.Context) (*Message, error) {
	select {
	case msg, ok := <-ch.queue:
		if !ok {
			// Channel closed
			return nil, nil
		}
		atomic.AddInt32(&ch.inFlight, -1)
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Close closes the channel and waits for all in-flight messages
// New Send calls will panic (like Go's chan behavior)
func (ch *Channel) Close() {
	close(ch.queue)
}

// InFlight returns the current number of messages in transit
func (ch *Channel) InFlight() int {
	return int(atomic.LoadInt32(&ch.inFlight))
}

// TryRecv attempts to receive without blocking
// Returns nil if no message is immediately available
func (ch *Channel) TryRecv() *Message {
	select {
	case msg := <-ch.queue:
		atomic.AddInt32(&ch.inFlight, -1)
		return msg
	default:
		return nil
	}
}
