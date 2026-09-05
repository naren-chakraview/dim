package schema

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// ApicurioClient is a RegistryBackend implementation for Apicurio Registry.
type ApicurioClient struct {
	baseURL string
	client  *http.Client
}

// NewApicurioClient creates a new Apicurio Registry client.
func NewApicurioClient(baseURL string) (*ApicurioClient, error) {
	return &ApicurioClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}, nil
}

// Health checks if the Apicurio registry is accessible.
func (ac *ApicurioClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", ac.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry health check failed: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

// RegisterSchema registers a new schema version in Apicurio.
func (ac *ApicurioClient) RegisterSchema(ctx context.Context, group, subject string, schema []byte) (string, error) {
	if group == "" {
		group = "default"
	}

	payload := map[string]interface{}{
		"type":    "JSON",
		"name":    subject,
		"content": string(schema),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/apis/registry/v3/groups/%s/artifacts/%s/versions",
		ac.baseURL, group, subject)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusConflict {
		// Schema already exists, this is OK for updating
		return ac.getLatestVersionFromResponse(respBody)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to register schema: %d %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	// Extract version ID from response
	if versionID, ok := result["version"].(float64); ok {
		return fmt.Sprintf("%.0f", versionID), nil
	}

	return "", fmt.Errorf("failed to extract version from response: %s", string(respBody))
}

// GetSchema fetches a schema by group, subject, and version from Apicurio.
func (ac *ApicurioClient) GetSchema(ctx context.Context, group, subject, version string) ([]byte, string, error) {
	if group == "" {
		group = "default"
	}

	// Get artifact versions endpoint to find the version ID
	url := fmt.Sprintf("%s/apis/registry/v3/groups/%s/artifacts/%s/versions",
		ac.baseURL, group, subject)

	if version != "" && version != "latest" {
		// Specific version requested
		url = fmt.Sprintf("%s/%s", url, version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("schema not found: group=%s subject=%s version=%s", group, subject, version)
	}

	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("failed to fetch schema: %d %s", resp.StatusCode, string(body))
	}

	// Parse response to extract schema content
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", err
	}

	// If version is "latest" or unspecified, parse list response
	if (version == "" || version == "latest") && isVersionListResponse(result) {
		versions := result["versions"].([]interface{})
		if len(versions) == 0 {
			return nil, "", fmt.Errorf("no versions found for subject %s", subject)
		}
		// Get the first (latest) version
		latestVersion := versions[0].(map[string]interface{})
		body, err = json.Marshal(latestVersion)
		if err != nil {
			return nil, "", err
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, "", err
		}
	}

	schemaContent, ok := result["content"].(string)
	if !ok {
		return nil, "", fmt.Errorf("schema content not found in response")
	}

	versionID, ok := result["version"].(float64)
	if !ok {
		return nil, "", fmt.Errorf("version not found in response")
	}

	return []byte(schemaContent), fmt.Sprintf("%.0f", versionID), nil
}

// GetLatestVersion returns the version ID of the latest schema for a subject.
func (ac *ApicurioClient) GetLatestVersion(ctx context.Context, group, subject string) (string, error) {
	if group == "" {
		group = "default"
	}

	url := fmt.Sprintf("%s/apis/registry/v3/groups/%s/artifacts/%s/versions",
		ac.baseURL, group, subject)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := ac.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("subject not found: group=%s subject=%s", group, subject)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("failed to fetch version: %d %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	versions, ok := result["versions"].([]interface{})
	if !ok || len(versions) == 0 {
		return "", fmt.Errorf("no versions found for subject %s", subject)
	}

	// Versions are returned in descending order (latest first)
	latestVersion := versions[0].(map[string]interface{})
	if versionID, ok := latestVersion["version"].(float64); ok {
		return fmt.Sprintf("%.0f", versionID), nil
	}

	return "", fmt.Errorf("failed to parse version from response")
}

// getLatestVersionFromResponse extracts the version ID from a registry response.
func (ac *ApicurioClient) getLatestVersionFromResponse(resp []byte) (string, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}

	if versionID, ok := result["version"].(float64); ok {
		return fmt.Sprintf("%.0f", versionID), nil
	}

	return "", fmt.Errorf("failed to extract version from response")
}

// isVersionListResponse checks if the response is a list of versions.
func isVersionListResponse(result map[string]interface{}) bool {
	_, ok := result["versions"]
	return ok
}
