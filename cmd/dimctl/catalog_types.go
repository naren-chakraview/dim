package main

import "time"

type CatalogResult struct {
	Type    string      `json:"type"`
	Name    string      `json:"name"`
	Domain  string      `json:"domain,omitempty"`
	Subject string      `json:"subject,omitempty"`
	Schema  interface{} `json:"schema,omitempty"`
	Updated string      `json:"updated"`
	URL     string      `json:"url,omitempty"`
}

type SearchResults struct {
	Query   string           `json:"query"`
	Count   int              `json:"count"`
	Results []CatalogResult  `json:"results"`
	Took    time.Duration    `json:"took_ms,omitempty"`
}

type ApicurioArtifact struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

type ApicurioResponse struct {
	Artifacts []ApicurioArtifact `json:"artifacts"`
	Count     int                `json:"count"`
}

type OpenLineageDataset struct {
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}
