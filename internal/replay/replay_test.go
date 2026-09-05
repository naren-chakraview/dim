package replay

import (
	"strings"
	"testing"
	"time"
)

func TestResultSummary(t *testing.T) {
	result := &Result{
		Total:     5,
		Replayed:  3,
		Failed:    1,
		Skipped:   1,
		Duration:  2 * time.Second,
		StartTime: time.Now(),
	}

	summary := result.Summary()

	tests := []struct {
		name     string
		label    string
		expected string
	}{
		{"Total", "Total", "Total:    5"},
		{"Replayed", "Replayed", "Replayed: 3"},
		{"Failed", "Failed", "Failed:   1"},
		{"Skipped", "Skipped", "Skipped:  1"},
		{"Duration", "Duration", "Duration: 2s"},
	}

	for _, tt := range tests {
		if !strings.Contains(summary, tt.expected) {
			t.Errorf("%s: expected %q to contain %q", tt.name, summary, tt.expected)
		}
	}

	if !strings.Contains(summary, "Replay complete:") {
		t.Errorf("Summary: expected header 'Replay complete:' not found in %q", summary)
	}
}

func TestResultSummaryLargeNumbers(t *testing.T) {
	result := &Result{
		Total:     12345,
		Replayed:  10000,
		Failed:    234,
		Skipped:   2111,
		Duration:  1 * time.Minute,
		StartTime: time.Now(),
	}

	summary := result.Summary()

	tests := []struct {
		name     string
		label    string
		expected string
	}{
		{"Total", "Total", "Total:    12345"},
		{"Replayed", "Replayed", "Replayed: 10000"},
		{"Failed", "Failed", "Failed:   234"},
		{"Skipped", "Skipped", "Skipped:  2111"},
	}

	for _, tt := range tests {
		if !strings.Contains(summary, tt.expected) {
			t.Errorf("%s: expected %q to contain %q", tt.name, summary, tt.expected)
		}
	}
}

func TestResultSummaryZeros(t *testing.T) {
	result := &Result{
		Total:     0,
		Replayed:  0,
		Failed:    0,
		Skipped:   0,
		Duration:  0,
		StartTime: time.Now(),
	}

	summary := result.Summary()

	tests := []struct {
		name     string
		label    string
		expected string
	}{
		{"Total", "Total", "Total:    0"},
		{"Replayed", "Replayed", "Replayed: 0"},
		{"Failed", "Failed", "Failed:   0"},
		{"Skipped", "Skipped", "Skipped:  0"},
	}

	for _, tt := range tests {
		if !strings.Contains(summary, tt.expected) {
			t.Errorf("%s: expected %q to contain %q", tt.name, summary, tt.expected)
		}
	}
}
