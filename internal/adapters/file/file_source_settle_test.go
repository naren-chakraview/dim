package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// collect returns every message the source emits within the window.
func collect(ch *engine.Channel, window time.Duration) []string {
	ctx, cancel := context.WithTimeout(context.Background(), window)
	defer cancel()
	var bodies []string
	for {
		msg, err := ch.Recv(ctx)
		if err != nil {
			return bodies
		}
		bodies = append(bodies, msg.Body.(string))
	}
}

func startLocal(t *testing.T, dir string, settle time.Duration) (*engine.Channel, context.CancelFunc) {
	t.Helper()
	out := engine.NewChannel("settle-test", 50)
	src, err := NewFileSource(dir, "100ms", out)
	if err != nil {
		t.Fatal(err)
	}
	src.settle = settle
	ctx, cancel := context.WithCancel(context.Background())
	go src.Start(ctx)
	return out, cancel
}

// A producer that writes a file slowly, in chunks, while the poller polls. The poller
// must emit exactly one message, with the complete content, and never a partial one.
func TestPollerNeverEmitsPartialSlowWrite(t *testing.T) {
	dir := t.TempDir()
	out, cancel := startLocal(t, dir, 600*time.Millisecond)
	defer cancel()

	chunks := []string{`{"id": "slow-`, `001", "value": `, `42}`}
	full := chunks[0] + chunks[1] + chunks[2]
	path := filepath.Join(dir, "slow.json")
	for i, c := range chunks {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			t.Fatal(err)
		}
		f.WriteString(c)
		f.Close()
		if i < len(chunks)-1 {
			time.Sleep(250 * time.Millisecond) // slower than one poll, shorter than the settle window
		}
	}

	got := collect(out, 3*time.Second)
	if len(got) != 1 {
		t.Fatalf("emitted %d messages, want exactly 1: %q", len(got), got)
	}
	if got[0] != full {
		t.Fatalf("emitted %q, want the complete content %q", got[0], full)
	}
}

// Empty files are not emitted as empty messages.
func TestPollerSkipsEmptyFile(t *testing.T) {
	dir := t.TempDir()
	out, cancel := startLocal(t, dir, 200*time.Millisecond)
	defer cancel()

	if err := os.WriteFile(filepath.Join(dir, "empty.json"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := collect(out, 1500*time.Millisecond); len(got) != 0 {
		t.Fatalf("empty file emitted %d message(s): %q", len(got), got)
	}
}
