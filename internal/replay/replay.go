package replay

import (
	"strconv"
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
		"  Total:    " + strconv.Itoa(r.Total) + "\n" +
		"  Replayed: " + strconv.Itoa(r.Replayed) + "\n" +
		"  Failed:   " + strconv.Itoa(r.Failed) + "\n" +
		"  Skipped:  " + strconv.Itoa(r.Skipped) + "\n" +
		"  Duration: " + r.Duration.String()
}
