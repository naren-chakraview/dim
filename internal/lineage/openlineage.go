package lineage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenLineageEmitter publishes lineage events to Marquez via OpenLineage API (M1.7.2)
type OpenLineageEmitter struct {
	marquezURL string
	httpClient *http.Client
	eventCh    chan *OpenLineageEvent
	batchSize  int
	flushTicker *time.Ticker
}

// OpenLineageEvent represents a lineage event in OpenLineage format
type OpenLineageEvent struct {
	EventType  string      `json:"eventType"`  // START, COMPLETE, FAIL, ABORT
	EventTime  string      `json:"eventTime"`  // RFC3339 format
	Run        *Run        `json:"run"`
	Job        *Job        `json:"job"`
	Inputs     []Dataset   `json:"inputs,omitempty"`
	Outputs    []Dataset   `json:"outputs,omitempty"`
	Producer   string      `json:"producer"`  // e.g., "https://github.com/naren-chakraview/dim"
}

// Run represents a job run in OpenLineage
type Run struct {
	RunID  string                 `json:"runId"`
	Facets map[string]interface{} `json:"facets,omitempty"`
}

// Job represents a job in OpenLineage
type Job struct {
	Namespace string                 `json:"namespace"`
	Name      string                 `json:"name"`
	Facets    map[string]interface{} `json:"facets,omitempty"`
}

// Dataset represents an input or output dataset
type Dataset struct {
	Namespace string                 `json:"namespace"`
	Name      string                 `json:"name"`
	Facets    map[string]interface{} `json:"facets,omitempty"`
}

// SchemaDatasetFacet describes the schema of a dataset (M1.7.3)
type SchemaDatasetFacet struct {
	Fields []SchemaField `json:"fields"`
}

// SchemaField describes a field in a schema
type SchemaField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Nullable    bool   `json:"nullable"`
}

// NewOpenLineageEmitter creates a new emitter (M1.7.2)
func NewOpenLineageEmitter(marquezURL string, batchSize int, flushIntervalMs int) *OpenLineageEmitter {
	if batchSize < 1 {
		batchSize = 100
	}
	if flushIntervalMs < 100 {
		flushIntervalMs = 5000
	}

	return &OpenLineageEmitter{
		marquezURL:  marquezURL,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		eventCh:     make(chan *OpenLineageEvent, batchSize*2),
		batchSize:   batchSize,
		flushTicker: time.NewTicker(time.Duration(flushIntervalMs) * time.Millisecond),
	}
}

// Emit queues an event for publishing
func (ole *OpenLineageEmitter) Emit(event *OpenLineageEvent) error {
	select {
	case ole.eventCh <- event:
		return nil
	default:
		return fmt.Errorf("event queue full")
	}
}

// Start begins the batch export loop
func (ole *OpenLineageEmitter) Start(ctx context.Context) {
	go ole.batchExportLoop(ctx)
}

// batchExportLoop accumulates events and flushes them periodically
func (ole *OpenLineageEmitter) batchExportLoop(ctx context.Context) {
	batch := make([]*OpenLineageEvent, 0, ole.batchSize)

	for {
		select {
		case event := <-ole.eventCh:
			batch = append(batch, event)
			if len(batch) >= ole.batchSize {
				_ = ole.flushBatch(ctx, batch)
				batch = make([]*OpenLineageEvent, 0, ole.batchSize)
			}

		case <-ole.flushTicker.C:
			if len(batch) > 0 {
				_ = ole.flushBatch(ctx, batch)
				batch = make([]*OpenLineageEvent, 0, ole.batchSize)
			}

		case <-ctx.Done():
			if len(batch) > 0 {
				_ = ole.flushBatch(context.Background(), batch)
			}
			return
		}
	}
}

// flushBatch publishes accumulated events to Marquez
func (ole *OpenLineageEmitter) flushBatch(ctx context.Context, events []*OpenLineageEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Marshal events to JSON
	payload, err := json.Marshal(map[string]interface{}{
		"events": events,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	// POST to Marquez with retries
	url := fmt.Sprintf("%s/api/v1/lineage", ole.marquezURL)
	return ole.postWithRetry(ctx, url, payload)
}

// postWithRetry posts to Marquez with retry logic
func (ole *OpenLineageEmitter) postWithRetry(ctx context.Context, url string, payload []byte) error {
	const maxRetries = 3
	const backoffMs = 500

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(backoffMs*(1<<uint(attempt-1))) * time.Millisecond
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// POST request
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := ole.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Check status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			return nil
		}

		// Read error response
		body, _ := io.ReadAll(resp.Body)
		lastErr = fmt.Errorf("marquez returned %d: %s", resp.StatusCode, string(body))

		// Non-retryable error (e.g., 400, 404)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}
	}

	return lastErr
}

// Stop gracefully shuts down the emitter
func (ole *OpenLineageEmitter) Stop(ctx context.Context) error {
	ole.flushTicker.Stop()
	close(ole.eventCh)

	// Wait for remaining events to be flushed
	// (batchExportLoop will exit when eventCh is closed and empty)
	done := make(chan struct{})
	go func() {
		for range ole.eventCh {
			// Drain remaining events
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}
