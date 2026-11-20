package factory

import (
	"net"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
)

// Reachability is reported from real probes. A closed port is unreachable, a live
// listener is reachable, a missing path is unreachable, and anything the probe cannot
// tell is unchecked, never reachable.
func TestCheckSourceReachabilityReportsRealProbes(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	live := listener.Addr().(*net.TCPAddr).Port

	closedL, _ := net.Listen("tcp", "127.0.0.1:0")
	closed := closedL.Addr().(*net.TCPAddr).Port
	closedL.Close()

	dir := t.TempDir()
	cfg := &config.RouteConfig{
		Sources: map[string]config.SourceSpec{
			"landing": {Type: "file", Path: dir},
			"gone":    {Type: "file", Path: filepath.Join(dir, "missing")},
			"live":    {Type: "kafka", Extra: map[string]interface{}{"brokers": []interface{}{"127.0.0.1:" + strconv.Itoa(live)}}},
			"dead":    {Type: "kafka", Extra: map[string]interface{}{"brokers": []interface{}{"127.0.0.1:" + strconv.Itoa(closed)}}},
			"bucket":  {Type: "s3"},
		},
		Routes: map[string]config.RouteSpec{
			"r": {From: "landing"},
		},
	}
	want := map[string]string{
		"landing": ReachabilityReachable,
		"gone":    ReachabilityUnreachable,
		"live":    ReachabilityReachable,
		"dead":    ReachabilityUnreachable,
		"bucket":  ReachabilityUnchecked,
	}
	for _, st := range CheckSourceReachability(cfg, time.Second) {
		if st.Reachability != want[st.Name] {
			t.Errorf("source %q: reachability %q (%s), want %q", st.Name, st.Reachability, st.Detail, want[st.Name])
		}
		if st.Name == "landing" && len(st.Route) != 1 {
			t.Errorf("landing should list the route that reads it, got %v", st.Route)
		}
	}
}
