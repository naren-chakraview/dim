package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/replay"
)

// ReplayCmd implements the dimctl replay command for recovering DLQ messages
type ReplayCmd struct {
	From     string
	Route    string
	Filter   string
	Limit    int
	DryRun   bool
	Parallel int
}

// ParseReplayCmd parses command-line arguments for replay command
func ParseReplayCmd(args []string) (*ReplayCmd, error) {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cmd := &ReplayCmd{
		Limit:    100,
		Parallel: 1,
	}

	fs.StringVar(&cmd.From, "from", "", "Source sink (DLQ name, required)")
	fs.StringVar(&cmd.Filter, "filter", "", "JSONata filter expression (optional)")
	fs.IntVar(&cmd.Limit, "limit", 100, "Max messages to replay")
	fs.BoolVar(&cmd.DryRun, "dry-run", false, "Preview without replaying")
	fs.IntVar(&cmd.Parallel, "parallel", 1, "Parallel replay threads")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	// Positional argument: target route
	if fs.NArg() < 1 {
		return nil, fmt.Errorf("usage: dimctl replay [options] <target-route>")
	}
	cmd.Route = fs.Arg(0)

	if cmd.From == "" {
		return nil, fmt.Errorf("--from is required")
	}
	if cmd.Parallel < 1 || cmd.Parallel > 16 {
		return nil, fmt.Errorf("--parallel must be between 1 and 16")
	}

	return cmd, nil
}

// Execute runs the replay command
// TODO: This function signature and implementation need updating for current engine API
func (rc *ReplayCmd) Execute(ctx context.Context) (*replay.Result, error) {
	// Load DLQ messages
	dlqMessages, err := loadDLQMessages(ctx, rc.From)
	if err != nil {
		return nil, fmt.Errorf("failed to load DLQ messages: %w", err)
	}

	// Apply filter if provided
	var filtered []*engine.Message
	for _, msg := range dlqMessages {
		if rc.Filter == "" {
			filtered = append(filtered, msg)
		} else {
			match, err := applyFilter(msg, rc.Filter)
			if err != nil {
				return nil, fmt.Errorf("filter error: %w", err)
			}
			if match {
				filtered = append(filtered, msg)
			}
		}
	}

	// Apply limit
	if len(filtered) > rc.Limit {
		filtered = filtered[:rc.Limit]
	}

	result := &replay.Result{
		Total:     len(filtered),
		StartTime: time.Now(),
	}

	// Dry-run preview
	if rc.DryRun {
		result.Replayed = len(filtered)
		result.Duration = time.Since(result.StartTime)
		return result, nil
	}

	// Parallel replay with concurrency control
	if err := rc.parallelReplay(ctx, filtered, result); err != nil {
		return result, err
	}

	result.Duration = time.Since(result.StartTime)
	return result, nil
}

// parallelReplay executes replayed with controlled concurrency
// TODO: This function needs updating for current engine API
func (rc *ReplayCmd) parallelReplay(ctx context.Context, messages []*engine.Message, result *replay.Result) error {
	// Function stub: replay API requires update for current engine version
	// engine.InjectMessage and message metadata APIs have changed
	_ = ctx
	_ = messages
	result.Failed = len(messages)
	result.Errors = append(result.Errors, "replay not implemented for current engine API")
	return fmt.Errorf("replay needs API update")
}

// loadDLQMessages loads messages from a DLQ sink
func loadDLQMessages(ctx context.Context, dlqName string) ([]*engine.Message, error) {
	// In production, this would:
	// 1. Query the sink configuration for the DLQ
	// 2. Load serialized messages from the sink (file, database, queue)
	// 3. Deserialize into Message structs
	// For now, stub implementation

	// Try to load from DLQ file output
	dlqPath := fmt.Sprintf("output/%s.jsonl", dlqName)
	data, err := os.ReadFile(dlqPath)
	if err != nil {
		return nil, fmt.Errorf("DLQ not found at %s: %w", dlqPath, err)
	}

	var messages []*engine.Message
	for _, line := range string(data) {
		if line == '\n' || line == 0 {
			continue
		}
		var msg engine.Message
		if err := json.Unmarshal([]byte(string(line)), &msg); err != nil {
			continue
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

// applyFilter evaluates a JSONata expression against a message
func applyFilter(msg *engine.Message, filterExpr string) (bool, error) {
	// In production, would use jsonata package for full JSONata support
	// Stub implementation handles simple expressions like: error_type == "timeout"

	switch filterExpr {
	case "error_type == \"timeout\"":
		return msg.Metadata.ErrorType == "timeout", nil
	case "error_type == \"authorization_denied\"":
		return msg.Metadata.ErrorType == "authorization_denied", nil
	default:
		return true, nil
	}
}
