package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestFileSourceLocalPolling tests that the file source polls a local directory
// and sends file contents as messages.
func TestFileSourceLocalPolling(t *testing.T) {
	// Create a temporary directory for polling
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.json")
	testContent := `{"id": "test-123", "value": 42}`

	// Create output channel
	outChan := engine.NewChannel("test-file-source", 10)

	// Create file source with 100ms poll interval
	source, err := NewFileSource(tmpDir, "100ms", outChan)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start polling in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- source.Start(ctx)
	}()

	// Give the source time to start initial poll
	time.Sleep(200 * time.Millisecond)

	// Write test file
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Wait for message to be sent
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()
	msg, err := outChan.Recv(waitCtx)
	if err != nil {
		t.Fatal("timeout waiting for file message")
	}

	// Verify message content
	if msg.Body != testContent {
		t.Errorf("message body mismatch: got %q, want %q", msg.Body, testContent)
	}

	if msg.Headers["source_file"] != "test.json" {
		t.Errorf("source_file header mismatch: got %q, want 'test.json'", msg.Headers["source_file"])
	}

	// Cancel context to stop polling
	cancel()

	// Wait for source to close
	<-time.After(1 * time.Second)

	// Close source
	if err := source.Close(); err != nil {
		t.Errorf("failed to close source: %v", err)
	}
}

// TestFileSourceDeduplication tests that the file source doesn't re-send
// unchanged files on subsequent polls.
func TestFileSourceDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "dedup-test.json")
	testContent := `{"id": "dedup-123"}`

	// Write initial file
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	outChan := engine.NewChannel("test-file-source-dedup", 10)
	source, err := NewFileSource(tmpDir, "50ms", outChan)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start polling
	errChan := make(chan error, 1)
	go func() {
		errChan <- source.Start(ctx)
	}()

	msgCount := 0

	// First message should arrive (initial poll)
	msgCtx, msgCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer msgCancel()
	_, recvErr := outChan.Recv(msgCtx)
	if recvErr != nil {
		t.Fatal("timeout waiting for first message")
	}
	msgCount++

	// Wait for two more poll cycles without new files
	time.Sleep(300 * time.Millisecond)

	// Collect any additional messages (should be none for unchanged file)
	checkCtx, checkCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer checkCancel()
	_, err2 := outChan.Recv(checkCtx)
	if err2 == nil {
		msgCount++
	}

	if msgCount > 1 {
		t.Errorf("file source re-sent unchanged file: got %d messages, want 1", msgCount)
	}

	cancel()
	<-time.After(1 * time.Second)
	source.Close()
}

// TestFileSourceMultipleFiles tests file source with multiple files in directory.
func TestFileSourceMultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	files := map[string]string{
		"file1.json": `{"id": "1"}`,
		"file2.json": `{"id": "2"}`,
		"file3.txt":  `not json`,
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write %s: %v", filename, err)
		}
	}

	outChan := engine.NewChannel("test-file-source-multi", 30)
	source, err := NewFileSource(tmpDir, "50ms", outChan)
	if err != nil {
		t.Fatalf("failed to create file source: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		source.Start(ctx)
	}()

	// Collect all messages
	messages := make([]*engine.Message, 0, len(files))

	for i := 0; i < len(files); i++ {
		collectCtx, collectCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer collectCancel()
		msg, err := outChan.Recv(collectCtx)
		if err != nil {
			t.Fatalf("timeout: only got %d/%d messages", len(messages), len(files))
		}
		messages = append(messages, msg)
	}

	// Verify all files were sent
	if len(messages) != len(files) {
		t.Errorf("message count mismatch: got %d, want %d", len(messages), len(files))
	}

	// Verify each message has the expected source_file header
	sentFiles := make(map[string]bool)
	for _, msg := range messages {
		if sourceFile, ok := msg.Headers["source_file"].(string); ok {
			sentFiles[sourceFile] = true
		}
	}

	for filename := range files {
		if !sentFiles[filename] {
			t.Errorf("missing message for file: %s", filename)
		}
	}

	cancel()
	<-time.After(1 * time.Second)
	source.Close()
}
