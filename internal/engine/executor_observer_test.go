package engine

import (
	"context"
	"errors"
	"testing"
	"time"
)

type recordingObserver struct {
	records []struct {
		id      string
		success bool
	}
}

func (r *recordingObserver) RecordMessage(correlationID string, latency time.Duration, success bool) {
	r.records = append(r.records, struct {
		id      string
		success bool
	}{correlationID, success})
}

type passStep struct{}

func (passStep) Execute(ctx context.Context, msg *Message) (*Message, error) { return msg, nil }

type failStep struct{}

func (failStep) Execute(ctx context.Context, msg *Message) (*Message, error) {
	return nil, errors.New("boom")
}

// Each processed message is reported once, with its real outcome.
func TestExecutorReportsOutcomeToObserver(t *testing.T) {
	obs := &recordingObserver{}
	ok := NewExecutor("ok", NewChannel("in", 1), NewChannel("out", 10), []Step{passStep{}})
	ok.SetObserver(obs)
	ctx := context.Background()

	msg := NewMessage(map[string]interface{}{"a": 1}, "r", "v1")
	msg.Metadata.CorrelationID = "c-1"
	ok.processMessageIteration(ctx, 0, msg)

	bad := NewExecutor("bad", NewChannel("in", 1), NewChannel("out", 10), []Step{failStep{}})
	bad.SetObserver(obs)
	msg2 := NewMessage(map[string]interface{}{"a": 2}, "r", "v1")
	msg2.Metadata.CorrelationID = "c-2"
	bad.processMessageIteration(ctx, 0, msg2)

	if len(obs.records) != 2 {
		t.Fatalf("observer got %d records, want 2", len(obs.records))
	}
	if obs.records[0].id != "c-1" || !obs.records[0].success {
		t.Errorf("first record = %+v, want c-1 success", obs.records[0])
	}
	if obs.records[1].id != "c-2" || obs.records[1].success {
		t.Errorf("second record = %+v, want c-2 failure", obs.records[1])
	}
}
