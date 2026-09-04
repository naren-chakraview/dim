package replay

import (
	"time"
)

// Result summarizes the outcome of a replay operation
type Result struct {
	Total     int
	Replayed  int
	Failed    int
	Skipped   int
	Duration  time.Duration
	Errors    []string
	StartTime time.Time
}

// Summary returns a human-readable summary of the replay result
func (r *Result) Summary() string {
	return "" +
		"Replay complete:\n" +
		"  Total:    " + string(rune(r.Total)) + "\n" +
		"  Replayed: " + string(rune(r.Replayed)) + "\n" +
		"  Failed:   " + string(rune(r.Failed)) + "\n" +
		"  Skipped:  " + string(rune(r.Skipped)) + "\n" +
		"  Duration: " + r.Duration.String()
}
