package steps

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
)

// ViolationInfo represents a contract violation (M0.3.6)
type ViolationInfo struct {
	ContractID string // contract that was violated
	Reason     string // validation error message
	Severity   string // "warning" or "error" based on strict flag
}

// ContractStep implements contract validation for data conformance checking.
// It validates message bodies against JSON Schema contracts and can either
// route violations to a separate sink or fail the pipeline in strict mode.
// Contract violations are permanent errors (don't retry).
type ContractStep struct {
	contractStore   *config.ContractStore
	routeName       string
	contractID      string
	strict          bool
	contract        *config.ContractSpec
	onViolationSink string
}

// NewContractStep creates a new contract validation step.
// The contract must exist in the store for the given route.
func NewContractStep(contractStore *config.ContractStore, routeName, contractID string) (*ContractStep, error) {
	if contractStore == nil {
		return nil, fmt.Errorf("contract store is nil")
	}

	if routeName == "" {
		return nil, fmt.Errorf("route name is empty")
	}

	if contractID == "" {
		return nil, fmt.Errorf("contract ID is empty")
	}

	// Look up the contract
	contract, err := contractStore.GetContractByID(routeName, contractID)
	if err != nil {
		return nil, err
	}

	// Get the compiled schema to verify it exists
	_, err = contractStore.GetCompiledSchema(routeName, contractID)
	if err != nil {
		return nil, err
	}

	return &ContractStep{
		contractStore:   contractStore,
		routeName:       routeName,
		contractID:      contractID,
		strict:          contract.Strict,
		contract:        contract,
		onViolationSink: contract.OnViolation,
	}, nil
}

// Execute validates the message body against the contract schema.
// Returns:
// - (msg with contract_version stamped, nil) if validation passes
// - (nil, error) if validation fails in strict mode
// - (msg with violation marker, nil) if validation fails in non-strict mode
//   (caller is responsible for routing to on_violation sink)
func (cs *ContractStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	if msg == nil {
		return nil, nil
	}

	// Validate the message body against the contract schema
	err := cs.contractStore.ValidateMessage(cs.routeName, cs.contractID, msg.Body)

	if err == nil {
		// Validation passed: stamp contract version and return
		msg.Metadata.ContractVersion = cs.contract.Version
		return msg, nil
	}

	// Validation failed: handle based on strict mode
	violation := &ViolationInfo{
		ContractID: cs.contractID,
		Reason:     err.Error(),
		Severity:   "warning",
	}

	if cs.strict {
		// Strict mode: return error to trigger error path routing
		violation.Severity = "error"
		return nil, &ContractViolationError{
			Msg:       fmt.Sprintf("contract_violation: %s", err.Error()),
			Violation: violation,
		}
	}

	// Non-strict mode: stamp violation marker and pass through for on_violation routing
	// Caller is responsible for routing based on onViolationSink
	msg.Metadata.ContractVersion = cs.contract.Version
	msg.Metadata.ContractViolation = violation
	return msg, nil
}

// GetOnViolationSink returns the sink name where violations should be routed
func (cs *ContractStep) GetOnViolationSink() string {
	return cs.onViolationSink
}

// ContractViolationError represents a contract validation error
type ContractViolationError struct {
	Msg       string
	Violation *ViolationInfo
}

func (e *ContractViolationError) Error() string {
	return e.Msg
}

// IsContractViolation checks if an error is a contract violation
func IsContractViolation(err error) bool {
	_, ok := err.(*ContractViolationError)
	return ok
}

// GetContractViolation extracts violation info from a contract error
func GetContractViolation(err error) *ViolationInfo {
	if cve, ok := err.(*ContractViolationError); ok {
		return cve.Violation
	}
	return nil
}
