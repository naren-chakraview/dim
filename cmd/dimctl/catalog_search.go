package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type ApicurioClient struct {
	baseURL string
	client  *http.Client
}

type OpenLineageClient struct {
	baseURL string
	client  *http.Client
}

func NewApicurioClient() *ApicurioClient {
	url := os.Getenv("APICURIO_URL")
	if url == "" {
		url = "http://localhost:8080"
	}
	return &ApicurioClient{
		baseURL: url,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

func NewOpenLineageClient() *OpenLineageClient {
	url := os.Getenv("OPENLINEAGE_URL")
	if url == "" {
		url = "http://localhost:5000"
	}
	return &OpenLineageClient{
		baseURL: url,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// QueryContracts searches the Apicurio registry. A non-empty group restricts the
// search to that registry group (verified against Apicurio 3.3.5's v2 search API);
// each result is labelled with the group the registry reports for it.
func (c *ApicurioClient) QueryContracts(ctx context.Context, subject, group string) ([]CatalogResult, error) {
	params := url.Values{}
	if subject != "" {
		params.Set("search", subject)
	}
	if group != "" {
		params.Set("group", group)
	}
	reqURL := fmt.Sprintf("%s/apis/registry/v2/search/artifacts", c.baseURL)
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apicurio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apicurio returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("apicurio response read failed: %w", err)
	}
	var apicurioResp ApicurioResponse
	if err := json.Unmarshal(body, &apicurioResp); err != nil {
		// An unreadable body must not look like "no contracts".
		return nil, fmt.Errorf("apicurio returned an unreadable response: %w", err)
	}

	var results []CatalogResult
	for _, artifact := range apicurioResp.Artifacts {
		// The registry reports each artifact's group and modification time.
		domain := artifact.GroupID
		if domain == "" {
			domain = group
		}
		results = append(results, CatalogResult{
			Type:    "contract",
			Name:    artifact.ID,
			Domain:  domain,
			Subject: artifact.ID,
			Updated: artifact.ModifiedOn,
			URL:     fmt.Sprintf("%s/ui/artifacts/%s", c.baseURL, artifact.ID),
		})
	}
	return results, nil
}

func (c *OpenLineageClient) QueryDatasets(ctx context.Context, domain string) ([]CatalogResult, error) {
	url := fmt.Sprintf("%s/api/v1/datasets", c.baseURL)
	if domain != "" {
		url = fmt.Sprintf("%s/api/v1/namespaces/%s/datasets", c.baseURL, domain)
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openlineage request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openlineage returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openlineage response read failed: %w", err)
	}
	var datasets []OpenLineageDataset
	if err := json.Unmarshal(body, &datasets); err != nil {
		return nil, fmt.Errorf("openlineage returned an unreadable response: %w", err)
	}

	var results []CatalogResult

	for _, ds := range datasets {
		results = append(results, CatalogResult{
			Type:    "product",
			Name:    ds.Name,
			Domain:  ds.Namespace,
			Updated: "", // Not fabricated; actual timestamp from OpenLineage would go here
		})
	}
	return results, nil
}

func Search(ctx context.Context, query, filterType, filterDomain, filterConnType string) (*SearchResults, error) {
	start := time.Now()
	results := &SearchResults{Query: query}

	apicurio := NewApicurioClient()
	openlineage := NewOpenLineageClient()

	if filterType == "" || filterType == "contract" {
		// The domain filter is applied by the registry (group), not by comparing a
		// field the query never fills in.
		contracts, err := apicurio.QueryContracts(ctx, query, filterDomain)
		if err != nil {
			// Distinguish between "no results" and "registry unreachable"
			return nil, fmt.Errorf("registry unreachable: %w", err)
		}
		results.Results = append(results.Results, contracts...)
	}

	if filterType == "" || filterType == "product" {
		products, err := openlineage.QueryDatasets(ctx, filterDomain)
		if err != nil {
			// Distinguish between "no results" and "registry unreachable"
			return nil, fmt.Errorf("registry unreachable: %w", err)
		}
		results.Results = append(results.Results, products...)
	}

	results.Count = len(results.Results)
	results.Took = time.Since(start)
	return results, nil
}
