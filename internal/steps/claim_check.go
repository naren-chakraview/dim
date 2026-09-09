package steps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// ClaimCheckStep implements the Claim Check EIP
// It stores large payloads externally and replaces them with references
type ClaimCheckStep struct {
	store             claimcheck.ClaimCheckStore
	payloadField      string // JSONata field path to extract payload (e.g., "body.attachment")
	removePayload     bool   // Whether to remove original payload from body
	ticketFieldPath   string // Where to store the claim-check ticket in the message
}

// NewClaimCheckStep creates a new claim-check step
func NewClaimCheckStep(spec *config.ClaimCheckSpec, store claimcheck.ClaimCheckStore) (*ClaimCheckStep, error) {
	if spec == nil {
		return nil, fmt.Errorf("claim check spec is required")
	}

	if store == nil {
		// Use in-memory store as default for testing
		store = claimcheck.NewInMemoryClaimCheckStore()
	}

	if spec.PayloadField == "" {
		spec.PayloadField = "body"
	}

	if spec.TicketFieldPath == "" {
		spec.TicketFieldPath = "_claim_check"
	}

	return &ClaimCheckStep{
		store:           store,
		payloadField:    spec.PayloadField,
		removePayload:   spec.RemovePayload,
		ticketFieldPath: spec.TicketFieldPath,
	}, nil
}

// Execute processes a message through claim-check
// It extracts the payload, stores it, and replaces with a reference
func (s *ClaimCheckStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("claim check: message is nil")
	}

	// For simplicity, store the entire body
	payloadBytes, err := marshalPayload(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("claim check: failed to marshal payload: %w", err)
	}

	// Create metadata with hash
	hash := sha256.Sum256(payloadBytes)
	metadata := claimcheck.ClaimCheckMetadata{
		ContentType: "application/json",
		Size:        int64(len(payloadBytes)),
		Hash:        "sha256:" + hex.EncodeToString(hash[:]),
	}

	// Store payload
	ticket, err := s.store.Store(ctx, payloadBytes, metadata)
	if err != nil {
		return nil, fmt.Errorf("claim check: failed to store payload: %w", err)
	}

	// Create output message
	outMsg := msg.Copy()

	// Create claim-check ticket object
	claimCheckInfo := map[string]interface{}{
		"ticket":       ticket,
		"size":         metadata.Size,
		"content_type": metadata.ContentType,
		"hash":         metadata.Hash,
	}

	// Replace body with lightweight message
	if s.removePayload {
		outMsg.Body = map[string]interface{}{
			s.ticketFieldPath: claimCheckInfo,
		}
	} else {
		// Keep body but add ticket
		if bodyMap, ok := msg.Body.(map[string]interface{}); ok {
			newBody := make(map[string]interface{})
			for k, v := range bodyMap {
				newBody[k] = v
			}
			newBody[s.ticketFieldPath] = claimCheckInfo
			outMsg.Body = newBody
		}
	}

	// Record in metadata
	if outMsg.Metadata.SplitterFacet == nil {
		// Use a generic facet for now (ideally would have ClaimCheckFacet)
		outMsg.Metadata.SplitterFacet = &SplitterFacet{
			SplitExpr:     "claim_check",
			TotalElements: 1,
			ElementIndex:  0,
			SplitTrigger:  "payload_checked",
		}
	}

	return outMsg, nil
}

// Helper to marshal payload
func marshalPayload(data interface{}) ([]byte, error) {
	if data == nil {
		return []byte("null"), nil
	}

	// Simple approach: convert to map and serialize
	switch v := data.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		// In production, would use JSON marshaling
		return []byte(fmt.Sprintf("%v", v)), nil
	}
}

// ResolveClaimCheck is a helper to retrieve a checked payload
func ResolveClaimCheck(ctx context.Context, store claimcheck.ClaimCheckStore, msg *engine.Message) (interface{}, error) {
	if msg == nil {
		return nil, fmt.Errorf("message is nil")
	}

	bodyMap, ok := msg.Body.(map[string]interface{})
	if !ok {
		return msg.Body, nil // Not a claim-checked message
	}

	claimCheckInfo, ok := bodyMap["_claim_check"].(map[string]interface{})
	if !ok {
		return msg.Body, nil // No claim-check ticket
	}

	ticket, ok := claimCheckInfo["ticket"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid claim-check ticket")
	}

	// Retrieve from store
	payload, _, err := store.Retrieve(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve claim-checked payload: %w", err)
	}

	// Restore and return
	if len(payload) == 0 {
		return nil, nil
	}

	// In production, would deserialize based on content_type
	return payload, nil
}
