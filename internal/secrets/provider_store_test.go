package secrets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeProvider is an in-memory Provider. err, when set, is returned by every call.
type fakeProvider struct {
	values map[string]string
	err    error
}

func (f *fakeProvider) Get(ctx context.Context, path string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	v, ok := f.values[path]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, path)
	}
	return v, nil
}

func (f *fakeProvider) List(ctx context.Context, prefix string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []string
	for p := range f.values {
		if strings.HasPrefix(p, prefix) {
			out = append(out, p)
		}
	}
	return out, nil
}

func newTestStore(t *testing.T, values map[string]string, err error) *ProviderStore {
	t.Helper()
	s, e := NewProviderStore(&fakeProvider{values: values, err: err}, "dim")
	if e != nil {
		t.Fatal(e)
	}
	return s
}

var _ SecretStore = (*ProviderStore)(nil) // the provider store is a SecretStore

func TestProviderStoreResolvesInDomainThenGlobal(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"dim/payments/db":    "payments-db",
		"dim/_global/shared": "global-shared",
	}, nil)

	if v, err := s.Resolve("payments", "db"); err != nil || v != "payments-db" {
		t.Fatalf("own domain: got %q, %v", v, err)
	}
	if v, err := s.Resolve("payments", "shared"); err != nil || v != "global-shared" {
		t.Fatalf("global fallback: got %q, %v", v, err)
	}
}

func TestProviderStoreDeniesExplicitCrossDomain(t *testing.T) {
	s := newTestStore(t, map[string]string{"dim/payments/db": "payments-db"}, nil)
	if _, err := s.Resolve("fraud", "payments.db"); !errors.Is(err, ErrCrossDomainDenied) {
		t.Fatalf("explicit cross-domain: got %v, want ErrCrossDomainDenied", err)
	}
}

func TestProviderStoreUnqualifiedNameStaysInDomain(t *testing.T) {
	s := newTestStore(t, map[string]string{"dim/payments/db": "payments-db"}, nil)
	if _, err := s.Resolve("fraud", "db"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("unqualified from another domain: got %v, want ErrSecretNotFound", err)
	}
}

// A provider that cannot be reached is an error, never "not found": the resolution
// fails closed instead of looking as if the secret is simply absent.
func TestProviderStoreProviderFailureIsNotNotFound(t *testing.T) {
	unavailable := fmt.Errorf("%w: connection refused", ErrProviderUnavailable)
	s := newTestStore(t, nil, unavailable)
	_, err := s.Resolve("payments", "db")
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("got %v, want ErrProviderUnavailable", err)
	}
	if errors.Is(err, ErrSecretNotFound) {
		t.Fatal("a provider failure was reported as not found")
	}
}

func TestProviderStoreIsReadOnly(t *testing.T) {
	s := newTestStore(t, nil, nil)
	if err := s.Register(SecretEntry{Name: "x", Value: "y", Domain: "payments"}); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("Register: got %v, want ErrReadOnly", err)
	}
}

func TestProviderStoreListsDomainNames(t *testing.T) {
	s := newTestStore(t, map[string]string{
		"dim/payments/db":  "a",
		"dim/payments/api": "b",
		"dim/fraud/other":  "c",
	}, nil)
	entries, err := s.ListByDomain("payments")
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
		if e.Domain != "payments" || e.Value != "" {
			t.Fatalf("entry %+v: want domain set and no value", e)
		}
	}
	if !names["db"] || !names["api"] || names["other"] {
		t.Fatalf("names = %v, want db and api only", names)
	}
}

func TestNewProviderStoreRejectsEmptyPrefix(t *testing.T) {
	if _, err := NewProviderStore(&fakeProvider{}, "/"); err == nil {
		t.Fatal("empty prefix accepted")
	}
}
