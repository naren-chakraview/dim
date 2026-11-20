package secrets

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func vaultServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != token {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/dim/payments/db":
			w.Write([]byte(`{"data":{"data":{"value":"vault-db-pass"},"metadata":{}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/dim/payments/broken":
			w.Write([]byte(`{"data":{"data":{"other":1}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/secret/data/dim/payments/down":
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == "LIST" && r.URL.Path == "/v1/secret/metadata/dim/payments":
			w.Write([]byte(`{"data":{"keys":["api","db","nested/"]}}`))
		case r.Method == "LIST":
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestVaultProviderGetAndMissing(t *testing.T) {
	srv := vaultServer(t, "root-token")
	defer srv.Close()
	p, err := NewVaultProvider(srv.URL, "root-token", "secret")
	if err != nil {
		t.Fatal(err)
	}

	if v, err := p.Get(context.Background(), "dim/payments/db"); err != nil || v != "vault-db-pass" {
		t.Fatalf("Get: got %q, %v", v, err)
	}
	if _, err := p.Get(context.Background(), "dim/payments/absent"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("missing secret: got %v, want ErrSecretNotFound", err)
	}
}

func TestVaultProviderServerErrorIsUnavailable(t *testing.T) {
	srv := vaultServer(t, "root-token")
	defer srv.Close()
	p, _ := NewVaultProvider(srv.URL, "root-token", "secret")
	_, err := p.Get(context.Background(), "dim/payments/down")
	if !errors.Is(err, ErrProviderUnavailable) || errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("server error: got %v, want ErrProviderUnavailable only", err)
	}
}

func TestVaultProviderRejectsWrongToken(t *testing.T) {
	srv := vaultServer(t, "root-token")
	defer srv.Close()
	p, _ := NewVaultProvider(srv.URL, "wrong", "secret")
	if _, err := p.Get(context.Background(), "dim/payments/db"); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("wrong token: got %v, want ErrProviderUnavailable", err)
	}
}

func TestVaultProviderListsNames(t *testing.T) {
	srv := vaultServer(t, "root-token")
	defer srv.Close()
	p, _ := NewVaultProvider(srv.URL, "root-token", "secret")
	names, err := p.List(context.Background(), "dim/payments/")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "dim/payments/api" || names[1] != "dim/payments/db" {
		t.Fatalf("names = %v, want api and db (directories skipped)", names)
	}
}

func TestVaultProviderNeedsAddressAndToken(t *testing.T) {
	if _, err := NewVaultProvider("", "t", ""); err == nil {
		t.Fatal("empty address accepted")
	}
	if _, err := NewVaultProvider("http://x", "", ""); err == nil {
		t.Fatal("empty token accepted")
	}
}
