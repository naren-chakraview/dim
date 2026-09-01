package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/naren-chakraview/dim/internal/engine"
)

// FileSink reads messages from an input channel and writes them to a local file in JSONL format.
// It applies backpressure: the channel read blocks the sink until capacity is available.
// The sink spawns a goroutine that continuously reads from the channel and appends JSON lines to the file.
type FileSink struct {
	path      string          // filesystem path to the output file
	inChan    *engine.Channel // input channel to drain
	file      *os.File        // open file handle
	done      chan struct{}   // signal to stop the main loop
	closeOnce sync.Once       // ensure Close() can be called multiple times safely
}

// NewFileSink creates a new file sink that will write to the given path.
// It creates the file if it doesn't exist, and appends if it does.
// Parent directories are created if they don't exist.
// Returns an error if the path cannot be opened or directories cannot be created.
func NewFileSink(path string, inChan *engine.Channel) (*FileSink, error) {
	if inChan == nil {
		return nil, fmt.Errorf("input channel cannot be nil")
	}

	// Ensure parent directories exist
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create parent directories for %q: %w", path, err)
		}
	}

	// Open the file for writing: create if it doesn't exist, append if it does
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}

	return &FileSink{
		path:   path,
		inChan: inChan,
		file:   file,
		done:   make(chan struct{}),
	}, nil
}

// Start begins the main sink loop in a background goroutine.
// It reads messages from the input channel, marshals them to JSON, and writes them as JSONL to the file.
// The loop continues until ctx is cancelled or an error occurs.
// Start returns immediately; the goroutine runs in the background.
func (s *FileSink) Start(ctx context.Context) {
	go s.run(ctx)
}

// run is the main loop that processes messages from the channel.
// It blocks until the context is cancelled.
func (s *FileSink) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		default:
		}

		// Try to receive a message (this blocks until a message is available or ctx is cancelled)
		msg, err := s.inChan.Recv(ctx)
		if err != nil {
			// Context was cancelled or other error occurred
			return
		}

		// If the channel is closed, msg will be nil
		if msg == nil {
			return
		}

		// Marshal the message to JSON
		data, err := json.Marshal(msg)
		if err != nil {
			// Log the error but continue processing (don't crash the sink)
			// In a real implementation, this might be sent to an error handler
			fmt.Fprintf(os.Stderr, "failed to marshal message to JSON: %v\n", err)
			continue
		}

		// Write the JSON data and a newline to the file
		if _, err := s.file.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write to file: %v\n", err)
			return
		}

		if _, err := s.file.Write([]byte("\n")); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write newline to file: %v\n", err)
			return
		}
	}
}

// Close gracefully shuts down the sink, closes the done channel, and closes the file.
// It should be called before the context is cancelled to ensure all buffered data is flushed.
// Safe to call multiple times.
func (s *FileSink) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.done)
		closeErr = s.file.Close()
	})
	return closeErr
}
