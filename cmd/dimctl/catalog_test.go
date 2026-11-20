package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApicurioQueryContracts(t *testing.T) {
	// Mock Apicurio server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := ApicurioResponse{
			Count: 1,
			Artifacts: []ApicurioArtifact{
				{
					ID:   "payment-schema",
					Type: "AVRO",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &ApicurioClient{
		baseURL: server.URL,
		client:  server.Client(),
	}

	results, err := client.QueryContracts(context.Background(), "payment", "")
	if err != nil {
		t.Fatalf("QueryContracts failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Type != "contract" {
		t.Errorf("expected type 'contract', got '%s'", results[0].Type)
	}

	if results[0].Name != "payment-schema" {
		t.Errorf("expected name 'payment-schema', got '%s'", results[0].Name)
	}
}

func TestOpenLineageQueryDatasets(t *testing.T) {
	// Mock OpenLineage server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		datasets := []OpenLineageDataset{
			{
				Name:      "orders_processed",
				Namespace: "payments",
			},
		}
		json.NewEncoder(w).Encode(datasets)
	}))
	defer server.Close()

	client := &OpenLineageClient{
		baseURL: server.URL,
		client:  server.Client(),
	}

	results, err := client.QueryDatasets(context.Background(), "payments")
	if err != nil {
		t.Fatalf("QueryDatasets failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	if results[0].Type != "product" {
		t.Errorf("expected type 'product', got '%s'", results[0].Type)
	}

	if results[0].Name != "orders_processed" {
		t.Errorf("expected name 'orders_processed', got '%s'", results[0].Name)
	}

	if results[0].Domain != "payments" {
		t.Errorf("expected domain 'payments', got '%s'", results[0].Domain)
	}
}

func TestSearchGracefulFallback(t *testing.T) {
	// Test with unavailable Apicurio
	client := &ApicurioClient{
		baseURL: "http://invalid-apicurio:8080",
		client:  &http.Client{},
	}

	_, err := client.QueryContracts(context.Background(), "test", "")
	// Should return error gracefully
	if err == nil {
		t.Error("expected error for unavailable Apicurio")
	}
}

func TestSearchOutputFormats(t *testing.T) {
	t.Run("JSON output", func(t *testing.T) {
		results := &SearchResults{
			Query:   "test",
			Count:   0,
			Results: []CatalogResult{},
		}

		// Should be JSON marshallable
		data, err := json.Marshal(results)
		if err != nil {
			t.Fatalf("failed to marshal JSON: %v", err)
		}

		var unmarshalled SearchResults
		if err := json.Unmarshal(data, &unmarshalled); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v", err)
		}

		if unmarshalled.Query != "test" {
			t.Errorf("query mismatch: %s", unmarshalled.Query)
		}
	})
}

func TestCatalogCommand(t *testing.T) {
	// Verify the command is properly configured
	if catalogCmd == nil {
		t.Error("catalogCmd is nil")
	}

	if catalogCmd.Use != "catalog" {
		t.Errorf("expected command use 'catalog', got '%s'", catalogCmd.Use)
	}

	if searchCmd == nil {
		t.Error("searchCmd is nil")
	}

	if searchCmd.Use != "search <query>" {
		t.Errorf("expected search use 'search <query>', got '%s'", searchCmd.Use)
	}

	// Verify flags are configured
	if searchCmd.Flags().Lookup("type") == nil {
		t.Error("--type flag not configured")
	}

	if searchCmd.Flags().Lookup("domain") == nil {
		t.Error("--domain flag not configured")
	}

	if searchCmd.Flags().Lookup("json") == nil {
		t.Error("--json flag not configured")
	}
}
