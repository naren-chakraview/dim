package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These tests call the real Search() that `dimctl catalog search --domain` runs,
// pointed at mock registries through their documented env vars.

// mockRegistries returns an Apicurio mock that honors the group query parameter
// (as the registry does) and an OpenLineage mock with one dataset per namespace.
func mockRegistries(t *testing.T) (apicurioURL, openlineageURL string) {
	t.Helper()
	artifacts := []struct{ id, group string }{
		{"order-schema", "payments"},
		{"fraud-score", "fraud"},
	}
	apicurio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/registry/v2/search/artifacts" {
			http.NotFound(w, r)
			return
		}
		group := r.URL.Query().Get("group")
		var out []ApicurioArtifact
		for _, a := range artifacts {
			if group == "" || a.group == group {
				out = append(out, ApicurioArtifact{ID: a.id, Type: "JSON"})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ApicurioResponse{Artifacts: out, Count: len(out)})
	}))
	t.Cleanup(apicurio.Close)

	ol := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		all := []OpenLineageDataset{
			{Name: "payment_events", Namespace: "payments"},
			{Name: "fraud_signals", Namespace: "fraud"},
		}
		out := all
		if strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/") {
			ns := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/namespaces/"), "/datasets")
			out = nil
			for _, d := range all {
				if d.Namespace == ns {
					out = append(out, d)
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(ol.Close)
	return apicurio.URL, ol.URL
}

// No filters: Search returns contracts and products together, through the real path.
func TestSearchNoFiltersReturnsBothKinds(t *testing.T) {
	a, o := mockRegistries(t)
	t.Setenv("APICURIO_URL", a)
	t.Setenv("OPENLINEAGE_URL", o)

	res, err := Search(context.Background(), "", "", "", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Count != 4 {
		t.Fatalf("no filters returned %d results, want 4 (2 contracts + 2 products)", res.Count)
	}
}

func TestSearchDomainFilterReturnsContractsInDomain(t *testing.T) {
	a, o := mockRegistries(t)
	t.Setenv("APICURIO_URL", a)
	t.Setenv("OPENLINEAGE_URL", o)

	res, err := Search(context.Background(), "", "contract", "payments", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Count != 1 {
		t.Fatalf("--domain payments returned %d contracts, want 1 (silent zero-result regression)", res.Count)
	}
	if got := res.Results[0]; got.Name != "order-schema" || got.Domain != "payments" {
		t.Errorf("result = %+v, want order-schema in domain payments", got)
	}
}

func TestSearchDomainFilterExcludesOtherDomains(t *testing.T) {
	a, o := mockRegistries(t)
	t.Setenv("APICURIO_URL", a)
	t.Setenv("OPENLINEAGE_URL", o)

	res, err := Search(context.Background(), "", "contract", "fraud", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Count != 1 || res.Results[0].Name != "fraud-score" {
		t.Fatalf("--domain fraud returned %+v, want only fraud-score", res.Results)
	}
}

func TestSearchDomainFilterAppliesToProducts(t *testing.T) {
	a, o := mockRegistries(t)
	t.Setenv("APICURIO_URL", a)
	t.Setenv("OPENLINEAGE_URL", o)

	res, err := Search(context.Background(), "", "product", "payments", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if res.Count != 1 || res.Results[0].Name != "payment_events" || res.Results[0].Domain != "payments" {
		t.Fatalf("product results = %+v, want payment_events in payments", res.Results)
	}
}

// A registry that answers with a non-JSON body must be an error, not an empty
// result that looks like "no contracts".
func TestApicurioMalformedResponseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>proxy error</html>"))
	}))
	defer srv.Close()
	client := &ApicurioClient{baseURL: srv.URL, client: srv.Client()}
	if _, err := client.QueryContracts(context.Background(), "", ""); err == nil {
		t.Fatal("malformed Apicurio response returned no error")
	}
}

func TestOpenLineageMalformedResponseIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	client := &OpenLineageClient{baseURL: srv.URL, client: srv.Client()}
	if _, err := client.QueryDatasets(context.Background(), ""); err == nil {
		t.Fatal("malformed OpenLineage response returned no error")
	}
}
