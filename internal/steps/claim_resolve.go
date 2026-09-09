package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// ClaimResolveStep resolves claim-checked payloads back to full content
// Used by downstream steps/adapters that need the original payload
type ClaimResolveStep struct {
	store           claimcheck.ClaimCheckStore
	ticketPath      string // Where to find the ticket in message (default: "_claim_check")
	outputFieldPath string // Where to put resolved payload (default: "body")
}

// NewClaimResolveStep creates a new claim-resolve step
func NewClaimResolveStep(spec *config.ClaimResolveSpec, store claimcheck.ClaimCheckStore) (*ClaimResolveStep, error) {
	if spec == nil {
		return nil, fmt.Errorf("claim resolve spec is required")
	}

	if store == nil {
		return nil, fmt.Errorf("claim resolve: store is required")
	}

	ticketPath := spec.TicketFieldPath
	if ticketPath == "" {
		ticketPath = "_claim_check"
	}

	outputPath := spec.PayloadFieldPath
	if outputPath == "" {
		outputPath = "body"
	}

	return &ClaimResolveStep{
		store:           store,
		ticketPath:      ticketPath,
		outputFieldPath: outputPath,
	}, nil
}

// Execute processes a message through claim-resolve
// It looks for a claim-check ticket, retrieves the payload, and restores it
func (s *ClaimResolveStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, fmt.Errorf("claim resolve: message is nil")
	}

	// Extract claim-check ticket from message
	bodyMap, ok := msg.Body.(map[string]interface{})
	if !ok {
		// Not a map body, return as-is
		return msg, nil
	}

	ticketData, ok := bodyMap[s.ticketPath].(map[string]interface{})
	if !ok {
		// No claim-check ticket, return as-is
		return msg, nil
	}

	ticket, ok := ticketData["ticket"].(string)
	if !ok || ticket == "" {
		return nil, fmt.Errorf("claim resolve: invalid or missing ticket")
	}

	// Retrieve payload from store
	payload, meta, err := s.store.Retrieve(ctx, ticket)
	if err != nil {
		return nil, fmt.Errorf("claim resolve: failed to retrieve payload: %w", err)
	}

	// Verify integrity if hash is available
	if meta.Hash != "" {
		valid, err := s.store.Verify(ctx, ticket, meta.Hash)
		if err != nil {
			return nil, fmt.Errorf("claim resolve: verification failed: %w", err)
		}
		if !valid {
			return nil, fmt.Errorf("claim resolve: payload integrity check failed")
		}
	}

	// Restore payload to message
	outMsg := msg.Copy()

	// Deserialize payload based on content type
	var restoredPayload interface{}
	if meta.ContentType == "application/json" {
		// For JSON, keep as string representation
		// In production, would use proper JSON unmarshaling
		restoredPayload = string(payload)
	} else {
		// For other types, keep as bytes
		restoredPayload = payload
	}

	// Restore to body
	outMsg.Body = restoredPayload

	return outMsg, nil
}
