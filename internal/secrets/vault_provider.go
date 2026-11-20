package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// VaultProvider reads secrets from a HashiCorp Vault (or OpenBao) KV version 2 engine.
// A secret at <path> is stored under <mount>/data/<path> with a "value" field.
type VaultProvider struct {
	addr   string
	token  string
	mount  string
	client *http.Client
}

// NewVaultProvider returns a provider for the Vault at addr (for example
// http://127.0.0.1:8200), authenticated with token, reading the KV v2 engine at mount.
func NewVaultProvider(addr, token, mount string) (*VaultProvider, error) {
	addr = strings.TrimSuffix(addr, "/")
	if addr == "" {
		return nil, fmt.Errorf("vault provider needs an address")
	}
	if token == "" {
		return nil, fmt.Errorf("vault provider needs a token")
	}
	if mount == "" {
		mount = "secret"
	}
	return &VaultProvider{
		addr:   addr,
		token:  token,
		mount:  strings.Trim(mount, "/"),
		client: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// Get returns the "value" field of the KV v2 secret at path.
func (p *VaultProvider) Get(ctx context.Context, path string) (string, error) {
	body, status, err := p.do(ctx, http.MethodGet, p.mount+"/data/"+path, nil)
	if err != nil {
		return "", fmt.Errorf("%w: get %s: %v", ErrProviderUnavailable, path, err)
	}
	if status == http.StatusNotFound {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, path)
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("%w: get %s: HTTP %d", ErrProviderUnavailable, path, status)
	}
	var doc struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("%w: get %s: unreadable response: %v", ErrProviderUnavailable, path, err)
	}
	value, ok := doc.Data.Data["value"].(string)
	if !ok {
		return "", fmt.Errorf("secret %s has no string \"value\" field", path)
	}
	return value, nil
}

// List returns the names of the secrets directly under prefix (which ends with "/").
func (p *VaultProvider) List(ctx context.Context, prefix string) ([]string, error) {
	dir := strings.Trim(prefix, "/")
	body, status, err := p.do(ctx, "LIST", p.mount+"/metadata/"+dir, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: list %s: %v", ErrProviderUnavailable, prefix, err)
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: list %s: HTTP %d", ErrProviderUnavailable, prefix, status)
	}
	var doc struct {
		Data struct {
			Keys []string `json:"keys"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("%w: list %s: unreadable response: %v", ErrProviderUnavailable, prefix, err)
	}
	var names []string
	for _, k := range doc.Data.Keys {
		if !strings.HasSuffix(k, "/") { // directories end with "/"
			names = append(names, dir+"/"+k)
		}
	}
	return names, nil
}

func (p *VaultProvider) do(ctx context.Context, method, apiPath string, body io.Reader) ([]byte, int, error) {
	u := p.addr + "/v1/" + (&url.URL{Path: apiPath}).EscapedPath()
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-Vault-Token", p.token)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.StatusCode, err
}
