package engine

import (
	"encoding/json"
	"fmt"
	"time"
)

// DeadLetterEnvelope wraps a failed message with error context for dead-letter routing
type DeadLetterEnvelope struct {
	// OriginalMessage is the message that failed
	OriginalMessage *Message `json:"original_message"`

	// Error is the error message from step execution
	Error string `json:"error"`

	// FailedStep is the index of the step that failed (0-based)
	FailedStep int `json:"failed_step"`

	// FailedStepType is the type of the failed step (e.g., "filter", "translate")
	FailedStepType string `json:"failed_step_type"`

	// ProcessedAt is the time when the error occurred (UTC)
	ProcessedAt time.Time `json:"processed_at"`

	// Metadata contains optional extra context about the error
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewDeadLetterEnvelope creates a new dead-letter envelope for a failed message
func NewDeadLetterEnvelope(msg *Message, err error, stepIndex int, stepType string) *DeadLetterEnvelope {
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &DeadLetterEnvelope{
		OriginalMessage: msg,
		Error:           errorMsg,
		FailedStep:      stepIndex,
		FailedStepType:  stepType,
		ProcessedAt:     time.Now().UTC(),
		Metadata:        make(map[string]interface{}),
	}
}

// MarshalJSON serializes the DeadLetterEnvelope to JSON
// This produces clean, readable JSON output suitable for JSONL format (one JSON object per line)
func (dle *DeadLetterEnvelope) MarshalJSON() ([]byte, error) {
	type Alias DeadLetterEnvelope
	return json.Marshal(&struct {
		*Alias
		ProcessedAt string `json:"processed_at"`
	}{
		Alias:       (*Alias)(dle),
		ProcessedAt: dle.ProcessedAt.Format(time.RFC3339Nano),
	})
}

// UnmarshalJSON deserializes JSON to a DeadLetterEnvelope
// Handles the time format conversion
func (dle *DeadLetterEnvelope) UnmarshalJSON(data []byte) error {
	type Alias DeadLetterEnvelope
	aux := &struct {
		ProcessedAt string `json:"processed_at"`
		*Alias
	}{
		Alias: (*Alias)(dle),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.ProcessedAt != "" {
		t, err := time.Parse(time.RFC3339Nano, aux.ProcessedAt)
		if err != nil {
			return fmt.Errorf("failed to parse processed_at timestamp: %w", err)
		}
		dle.ProcessedAt = t
	}

	return nil
}
