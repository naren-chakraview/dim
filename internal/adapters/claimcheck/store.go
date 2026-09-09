package claimcheck

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// ClaimCheckMetadata holds payload metadata
type ClaimCheckMetadata struct {
	ContentType string
	Size        int64
	Hash        string
	Timestamp   time.Time
	ExpiresAt   *time.Time
}

// ClaimCheckStore is the interface for storing and retrieving payloads via references
type ClaimCheckStore interface {
	// Store saves a payload and returns a ticket reference
	Store(ctx context.Context, payload []byte, metadata ClaimCheckMetadata) (ticket string, err error)

	// Retrieve loads a payload using the ticket
	Retrieve(ctx context.Context, ticket string) (payload []byte, metadata ClaimCheckMetadata, err error)

	// Delete removes a stored payload
	Delete(ctx context.Context, ticket string) error

	// Verify checks integrity of a stored payload
	Verify(ctx context.Context, ticket string, expectedHash string) (bool, error)

	// Close gracefully shuts down the store
	Close(ctx context.Context) error
}

// InMemoryClaimCheckStore is a simple in-memory implementation for testing
type InMemoryClaimCheckStore struct {
	store map[string][]byte
}

// NewInMemoryClaimCheckStore creates a new in-memory claim-check store
func NewInMemoryClaimCheckStore() *InMemoryClaimCheckStore {
	return &InMemoryClaimCheckStore{
		store: make(map[string][]byte),
	}
}

// Store saves a payload in memory
func (s *InMemoryClaimCheckStore) Store(ctx context.Context, payload []byte, metadata ClaimCheckMetadata) (string, error) {
	// Generate ticket
	ticket := fmt.Sprintf("mem://claim-check/%d/%s", time.Now().Unix(), generateRandomID())

	// Store payload
	s.store[ticket] = append([]byte{}, payload...) // Copy to avoid mutation

	return ticket, nil
}

// Retrieve loads a payload from memory
func (s *InMemoryClaimCheckStore) Retrieve(ctx context.Context, ticket string) ([]byte, ClaimCheckMetadata, error) {
	payload, ok := s.store[ticket]
	if !ok {
		return nil, ClaimCheckMetadata{}, fmt.Errorf("ticket not found: %s", ticket)
	}

	metadata := ClaimCheckMetadata{
		Size:      int64(len(payload)),
		Hash:      computeHash(payload),
		Timestamp: time.Now(),
	}

	return append([]byte{}, payload...), metadata, nil // Return copy
}

// Delete removes a payload from memory
func (s *InMemoryClaimCheckStore) Delete(ctx context.Context, ticket string) error {
	delete(s.store, ticket)
	return nil
}

// Verify checks payload integrity
func (s *InMemoryClaimCheckStore) Verify(ctx context.Context, ticket string, expectedHash string) (bool, error) {
	payload, ok := s.store[ticket]
	if !ok {
		return false, fmt.Errorf("ticket not found: %s", ticket)
	}

	actualHash := computeHash(payload)
	return actualHash == expectedHash, nil
}

// Close is a no-op for in-memory store
func (s *InMemoryClaimCheckStore) Close(ctx context.Context) error {
	return nil
}

// Helper functions

func computeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func generateRandomID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if crypto/rand fails (shouldn't happen)
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
