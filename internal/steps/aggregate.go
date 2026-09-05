package steps

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/expr"
)

// AggregateStep collects related messages and emits combined output (M2.1.2, Phase 2)
type AggregateStep struct {
	correlationEvaluator *expr.Evaluator    // Compiled correlation key expression
	completionStrategy   string             // "count" or "time_window"
	count                int                // for count strategy
	timeoutMs            int                // for count strategy: fallback timeout
	windowMs             int                // for time_window strategy
	maxMessages          int                // for time_window strategy: safety valve
	outputEvaluator      *expr.Evaluator    // Compiled output transformation (optional)
	onNullCorrelation    string             // "error_path" or "skip"

	// State
	groups     map[string]*AggregationGroup
	mu         sync.RWMutex
	timers     map[string]*time.Timer
	drainCaps  chan struct{} // Concurrent drain cap (M2.1.2)
}

// AggregationGroup tracks messages being aggregated for a correlation key
type AggregationGroup struct {
	CorrelationKey string
	Messages       []*engine.Message
	FirstIngested  time.Time
	LastIngested   time.Time
	Drained        bool // Marked during hot reload drain
}

// AggregateSpec defines aggregator configuration
type AggregateSpec struct {
	CorrelationKey     string `yaml:"correlation_key" json:"correlation_key"`
	CompletionStrategy string `yaml:"completion_strategy" json:"completion_strategy"` // "count" or "time_window"

	// Count strategy
	Count     int `yaml:"count,omitempty" json:"count,omitempty"`
	TimeoutMs int `yaml:"timeout_ms,omitempty" json:"timeout_ms,omitempty"`

	// Time-window strategy
	WindowMs    int `yaml:"window_ms,omitempty" json:"window_ms,omitempty"`
	MaxMessages int `yaml:"max_messages,omitempty" json:"max_messages,omitempty"`

	// Output transformation
	OutputExpr string `yaml:"output_expr,omitempty" json:"output_expr,omitempty"`

	// Error handling
	OnNullCorrelation string `yaml:"on_null_correlation,omitempty" json:"on_null_correlation,omitempty"` // "error_path" default
	OnError           string `yaml:"on_error,omitempty" json:"on_error,omitempty"`
}

// NewAggregateStep creates an aggregator step
func NewAggregateStep(spec *AggregateSpec) (*AggregateStep, error) {
	if spec.CorrelationKey == "" {
		return nil, fmt.Errorf("correlation_key is required")
	}

	if spec.CompletionStrategy != "count" && spec.CompletionStrategy != "time_window" {
		return nil, fmt.Errorf("completion_strategy must be 'count' or 'time_window', got %s", spec.CompletionStrategy)
	}

	if spec.CompletionStrategy == "count" {
		if spec.Count < 1 {
			return nil, fmt.Errorf("count strategy requires count >= 1, got %d", spec.Count)
		}
		if spec.TimeoutMs < 100 {
			spec.TimeoutMs = 60000 // 60s default
		}
	} else {
		if spec.WindowMs < 100 {
			return nil, fmt.Errorf("time_window strategy requires window_ms >= 100, got %d", spec.WindowMs)
		}
		if spec.MaxMessages < 1 {
			spec.MaxMessages = 1000 // 1000 default
		}
	}

	if spec.OnNullCorrelation == "" {
		spec.OnNullCorrelation = "error_path"
	}

	// Compile correlation key expression
	correlationEval, err := expr.CompileExpression(spec.CorrelationKey)
	if err != nil {
		return nil, fmt.Errorf("failed to compile correlation_key expression: %w", err)
	}

	// Compile optional output expression
	var outputEval *expr.Evaluator
	if spec.OutputExpr != "" {
		var err error
		outputEval, err = expr.CompileExpression(spec.OutputExpr)
		if err != nil {
			return nil, fmt.Errorf("failed to compile output_expr expression: %w", err)
		}
	}

	return &AggregateStep{
		correlationEvaluator: correlationEval,
		completionStrategy:   spec.CompletionStrategy,
		count:                spec.Count,
		timeoutMs:            spec.TimeoutMs,
		windowMs:             spec.WindowMs,
		maxMessages:          spec.MaxMessages,
		outputEvaluator:      outputEval,
		onNullCorrelation:    spec.OnNullCorrelation,
		groups:               make(map[string]*AggregationGroup),
		timers:               make(map[string]*time.Timer),
		drainCaps:            make(chan struct{}, 10), // Max 10 concurrent drains
	}, nil
}

// Execute processes a message through the aggregator
func (as *AggregateStep) Execute(ctx context.Context, msg *engine.Message) (*engine.Message, error) {
	// Evaluate correlation key
	correlationKey, err := as.correlationEvaluator.Eval(msg.Body)
	if err != nil {
		return nil, fmt.Errorf("correlation_key evaluation failed: %w", err)
	}

	// Handle null correlation key
	if correlationKey == nil {
		if as.onNullCorrelation == "error_path" {
			return nil, fmt.Errorf("correlation_key evaluated to null")
		}
		// "skip" — don't aggregate, pass through
		return msg, nil
	}

	keyStr := fmt.Sprintf("%v", correlationKey)

	as.mu.Lock()
	group, exists := as.groups[keyStr]
	if !exists {
		group = &AggregationGroup{
			CorrelationKey: keyStr,
			Messages:       []*engine.Message{},
			FirstIngested:  time.Now().UTC(),
		}
		as.groups[keyStr] = group

		// Start timeout for this group (M2.1.2)
		if as.completionStrategy == "count" {
			timer := time.AfterFunc(time.Duration(as.timeoutMs)*time.Millisecond, func() {
				as.flushGroup(keyStr, "timeout")
			})
			as.timers[keyStr] = timer
		} else {
			timer := time.AfterFunc(time.Duration(as.windowMs)*time.Millisecond, func() {
				as.flushGroup(keyStr, "window_expired")
			})
			as.timers[keyStr] = timer
		}
	}

	group.Messages = append(group.Messages, msg)
	group.LastIngested = time.Now().UTC()

	// Check completion
	shouldFlush := false
	trigger := ""

	if as.completionStrategy == "count" && len(group.Messages) >= as.count {
		shouldFlush = true
		trigger = "count_reached"
	} else if as.completionStrategy == "time_window" && len(group.Messages) >= as.maxMessages {
		shouldFlush = true
		trigger = "max_messages_reached"
	}

	if shouldFlush {
		// Cancel timer and remove group
		if timer, ok := as.timers[keyStr]; ok {
			timer.Stop()
			delete(as.timers, keyStr)
		}
		groupToFlush := as.groups[keyStr]
		delete(as.groups, keyStr)
		as.mu.Unlock()

		return as.buildAggregateOutput(ctx, groupToFlush, trigger)
	}

	as.mu.Unlock()

	// Don't output yet — message is being aggregated
	return nil, nil
}

// flushGroup flushes a group when timeout or window expires
func (as *AggregateStep) flushGroup(keyStr string, trigger string) {
	as.mu.Lock()
	group, exists := as.groups[keyStr]
	if !exists {
		as.mu.Unlock()
		return
	}

	delete(as.groups, keyStr)
	if timer, ok := as.timers[keyStr]; ok {
		timer.Stop()
		delete(as.timers, keyStr)
	}
	as.mu.Unlock()

	// Build output (note: this doesn't propagate through Execute return;
	// in production, would queue to output channel)
	_, _ = as.buildAggregateOutput(context.Background(), group, trigger)
}

// buildAggregateOutput constructs the aggregated message
func (as *AggregateStep) buildAggregateOutput(ctx context.Context, group *AggregationGroup, trigger string) (*engine.Message, error) {
	// Build output structure
	msgBodies := make([]interface{}, len(group.Messages))
	for i, msg := range group.Messages {
		msgBodies[i] = map[string]interface{}{
			"body":    msg.Body,
			"headers": msg.Headers,
		}
	}

	output := map[string]interface{}{
		"messages":         msgBodies,
		"correlation_key":  group.CorrelationKey,
		"count":            len(group.Messages),
		"first_ingested":   group.FirstIngested.Format(time.RFC3339),
		"last_ingested":    group.LastIngested.Format(time.RFC3339),
		"completion_trigger": trigger,
	}

	// Apply optional output transformation
	if as.outputEvaluator != nil {
		transformed, err := as.outputEvaluator.Eval(output)
		if err != nil {
			return nil, fmt.Errorf("output_expr transformation failed: %w", err)
		}
		output = transformed.(map[string]interface{})
	}

	// Create output message with lineage to all inputs
	outMsg := engine.NewMessage(output, group.Messages[0].Metadata.Route, group.Messages[0].Metadata.RouteVersion)

	// Preserve first message's subject ID
	if group.Messages[0].Metadata.Principal != nil {
		outMsg.Metadata.Principal = group.Messages[0].Metadata.Principal
	}

	// Add aggregator facet to metadata
	outMsg.Metadata.AggregatorFacet = &AggregatorFacet{
		InputCount:         len(group.Messages),
		CorrelationKey:     group.CorrelationKey,
		CompletionStrategy: as.completionStrategy,
		CompletionTrigger:  trigger,
	}

	return outMsg, nil
}

// Drain flushes all in-flight groups on hot reload (M2.1.2)
func (as *AggregateStep) Drain(ctx context.Context, timeout time.Duration) error {
	// Acquire drain cap (max 10 concurrent drains)
	select {
	case as.drainCaps <- struct{}{}:
		defer func() { <-as.drainCaps }()
	case <-ctx.Done():
		return ctx.Err()
	}

	deadline := time.Now().Add(timeout)

	for {
		as.mu.Lock()
		if len(as.groups) == 0 {
			as.mu.Unlock()
			return nil
		}

		// Flush all groups
		for keyStr, group := range as.groups {
			if timer, ok := as.timers[keyStr]; ok {
				timer.Stop()
				delete(as.timers, keyStr)
			}
			groupToFlush := group
			delete(as.groups, keyStr)
			as.mu.Unlock()

			_, _ = as.buildAggregateOutput(context.Background(), groupToFlush, "drained")

			as.mu.Lock()
		}

		if len(as.groups) == 0 {
			as.mu.Unlock()
			return nil
		}

		as.mu.Unlock()

		if time.Now().After(deadline) {
			return fmt.Errorf("drain timeout exceeded with %d in-flight groups", len(as.groups))
		}

		time.Sleep(10 * time.Millisecond)
	}
}

// AggregatorFacet captures aggregation metadata for lineage
type AggregatorFacet struct {
	InputCount         int    `json:"input_count"`
	CorrelationKey     string `json:"correlation_key"`
	CompletionStrategy string `json:"completion_strategy"`
	CompletionTrigger  string `json:"completion_trigger"`
}
