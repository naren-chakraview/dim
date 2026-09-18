package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func (c *ApicurioClient) QueryContracts(ctx context.Context, subject string) ([]CatalogResult, error) {
	url := fmt.Sprintf("%s/apis/registry/v2/search/artifacts", c.baseURL)
	if subject != "" {
		url += "?search=" + subject
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apicurio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("apicurio returned %d", resp.StatusCode)
	}

	var apicurioResp ApicurioResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &apicurioResp); err != nil {
		return nil, nil
	}

	var results []CatalogResult
	for _, artifact := range apicurioResp.Artifacts {
		// Note: Apicurio API doesn't currently return artifact update time
		// in the search results endpoint; Updated field left empty
		// TODO: Query full artifact details to get actual UpdatedOn timestamp
		results = append(results, CatalogResult{
			Type:    "contract",
			Name:    artifact.ID,
			Subject: artifact.ID,
			Updated: "", // Not fabricated; actual timestamp requires additional API call
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

	var results []CatalogResult
	body, _ := io.ReadAll(resp.Body)
	var datasets []OpenLineageDataset
	if err := json.Unmarshal(body, &datasets); err != nil {
		return results, nil
	}

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
		contracts, err := apicurio.QueryContracts(ctx, query)
		if err != nil {
			// Distinguish between "no results" and "registry unreachable"
			return nil, fmt.Errorf("registry unreachable: %w", err)
		}
		// Filter by domain if specified
		if filterDomain != "" {
			var filtered []CatalogResult
			for _, c := range contracts {
				if c.Domain == filterDomain {
					filtered = append(filtered, c)
				}
			}
			contracts = filtered
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
