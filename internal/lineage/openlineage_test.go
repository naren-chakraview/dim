package lineage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestOpenLineageEmitter_EventMarshaling verifies events are correctly marshaled (M1.7.2)
func TestOpenLineageEmitter_EventMarshaling(t *testing.T) {
	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().Format(time.RFC3339),
		Producer:  "https://github.com/naren-chakraview/dim",
		Run: &Run{
			RunID: "urn:uuid:123",
		},
		Job: &Job{
			Namespace: "dim",
			Name:      "process-orders.v1",
		},
		Inputs: []Dataset{
			{
				Namespace: "dim://webhook",
				Name:      "order-events",
			},
		},
		Outputs: []Dataset{
			{
				Namespace: "dim://amqp",
				Name:      "orders-processed",
			},
		},
	}

	// Marshal to JSON
	payload, err := json.Marshal(map[string]interface{}{
		"events": []*OpenLineageEvent{event},
	})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(payload, &result)
	if err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	// Verify structure
	if events, ok := result["events"]; !ok || events == nil {
		t.Error("Missing 'events' field in marshaled output")
	}
}

// TestOpenLineageEmitter_EventPosting verifies events are posted to Marquez (M1.7.2)
func TestOpenLineageEmitter_EventPosting(t *testing.T) {
	// Mock Marquez server
	receivedPayload := []byte{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/lineage", func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedPayload, err = io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create emitter with very short flush interval
	emitter := NewOpenLineageEmitter(server.URL, 1, 100)
	emitter.Start(context.Background())

	// Emit an event
	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().Format(time.RFC3339),
		Producer:  "https://github.com/naren-chakraview/dim",
		Run: &Run{
			RunID: "urn:uuid:test",
		},
		Job: &Job{
			Namespace: "dim",
			Name:      "test-job",
		},
	}

	err := emitter.Emit(event)
	if err != nil {
		t.Fatalf("Failed to emit: %v", err)
	}

	// Wait for flush
	time.Sleep(200 * time.Millisecond)

	// Verify payload was received
	if len(receivedPayload) == 0 {
		t.Fatal("No payload received by mock server")
	}

	var payload map[string]interface{}
	err = json.Unmarshal(receivedPayload, &payload)
	if err != nil {
		t.Errorf("Invalid payload JSON: %v", err)
	}

	if events, ok := payload["events"]; !ok || events == nil {
		t.Error("Missing 'events' in received payload")
	}

	emitter.Stop(context.Background())
}

// TestOpenLineageEmitter_Batching verifies events are batched correctly (M1.7.4)
func TestOpenLineageEmitter_Batching(t *testing.T) {
	postCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/lineage", func(w http.ResponseWriter, r *http.Request) {
		postCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// Create emitter with batch size 5
	emitter := NewOpenLineageEmitter(server.URL, 5, 10000)
	emitter.Start(context.Background())

	// Emit 12 events
	for i := 0; i < 12; i++ {
		event := &OpenLineageEvent{
			EventType: "COMPLETE",
			EventTime: time.Now().Format(time.RFC3339),
			Producer:  "https://github.com/naren-chakraview/dim",
			Run: &Run{
				RunID: "urn:uuid:test",
			},
			Job: &Job{
				Namespace: "dim",
				Name:      "test-job",
			},
		}
		emitter.Emit(event)
	}

	// Wait for batches to be flushed (first batch at 5, second at 10)
	time.Sleep(200 * time.Millisecond)

	// Should have at least 2 POSTs (5 events per batch)
	if postCount < 2 {
		t.Errorf("Expected at least 2 POSTs, got %d", postCount)
	}

	emitter.Stop(context.Background())
}

// TestOpenLineageEmitter_RetryPolicy verifies retries on server errors (M1.7.4)
func TestOpenLineageEmitter_RetryPolicy(t *testing.T) {
	attemptCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/lineage", func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
			// Fail first 2 attempts
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Succeed on 3rd attempt
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	emitter := NewOpenLineageEmitter(server.URL, 1, 100)
	emitter.Start(context.Background())

	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().Format(time.RFC3339),
		Producer:  "https://github.com/naren-chakraview/dim",
		Run: &Run{
			RunID: "urn:uuid:test",
		},
		Job: &Job{
			Namespace: "dim",
			Name:      "test-job",
		},
	}

	emitter.Emit(event)

	// Wait for retries
	time.Sleep(2 * time.Second)

	// Should have retried
	if attemptCount < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attemptCount)
	}

	emitter.Stop(context.Background())
}

// TestOpenLineageEmitter_SchemaFacet verifies schema facet creation (M1.7.3)
func TestOpenLineageEmitter_SchemaFacet(t *testing.T) {
	facet := &SchemaDatasetFacet{
		Fields: []SchemaField{
			{
				Name:        "order_id",
				Type:        "string",
				Description: "Unique order identifier",
				Nullable:    false,
			},
			{
				Name:        "amount",
				Type:        "number",
				Description: "Order amount in USD",
				Nullable:    false,
			},
			{
				Name:        "customer_id",
				Type:        "string",
				Description: "Customer identifier",
				Nullable:    true,
			},
		},
	}

	// Marshal to JSON
	payload, err := json.Marshal(facet)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify structure
	var result SchemaDatasetFacet
	err = json.Unmarshal(payload, &result)
	if err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if len(result.Fields) != 3 {
		t.Errorf("Expected 3 fields, got %d", len(result.Fields))
	}

	if result.Fields[0].Name != "order_id" {
		t.Errorf("Expected first field name 'order_id', got %q", result.Fields[0].Name)
	}

	if result.Fields[0].Type != "string" {
		t.Errorf("Expected first field type 'string', got %q", result.Fields[0].Type)
	}

	if result.Fields[2].Nullable != true {
		t.Error("Expected last field to be nullable")
	}
}

// TestOpenLineageEmitter_DatasetFacetInEvent verifies facets in dataset (M1.7.3)
func TestOpenLineageEmitter_DatasetFacetInEvent(t *testing.T) {
	// Create event with schema facet
	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().Format(time.RFC3339),
		Producer:  "https://github.com/naren-chakraview/dim",
		Run: &Run{
			RunID: "urn:uuid:test",
		},
		Job: &Job{
			Namespace: "dim",
			Name:      "test-job",
		},
		Outputs: []Dataset{
			{
				Namespace: "dim://amqp",
				Name:      "orders-processed",
				Facets: map[string]interface{}{
					"schema": SchemaDatasetFacet{
						Fields: []SchemaField{
							{
								Name: "order_id",
								Type: "string",
							},
						},
					},
				},
			},
		},
	}

	// Marshal to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Verify it contains schema
	var result map[string]interface{}
	err = json.Unmarshal(payload, &result)
	if err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	if outputs, ok := result["outputs"].([]interface{}); ok && len(outputs) > 0 {
		if output, ok := outputs[0].(map[string]interface{}); ok {
			if _, hasFacets := output["facets"]; !hasFacets {
				t.Error("Output missing 'facets' field")
			}
		}
	} else {
		t.Error("Missing or invalid 'outputs' field")
	}
}

// TestOpenLineageEmitter_QueueFull verifies behavior when queue is full (M1.7.4)
func TestOpenLineageEmitter_QueueFull(t *testing.T) {
	emitter := NewOpenLineageEmitter("http://localhost:5000", 1, 10000)

	// Fill the queue
	event := &OpenLineageEvent{
		EventType: "COMPLETE",
		EventTime: time.Now().Format(time.RFC3339),
		Producer:  "https://github.com/naren-chakraview/dim",
		Run:       &Run{RunID: "test"},
		Job: &Job{
			Namespace: "dim",
			Name:      "test",
		},
	}

	// Emit more events than the buffer can hold
	for i := 0; i < 100; i++ {
		err := emitter.Emit(event)
		if err != nil && i > 2 {
			// Expected to fail when queue full
			break
		}
	}

	emitter.Stop(context.Background())
}
