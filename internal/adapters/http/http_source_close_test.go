package http

import (
	"fmt"
	"net"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
)

// A source that is built but never started (test_route does this) must release
// its port on Close. Before the fix, Close only shut down the http.Server, which
// does not know about a listener it never Serve()d, so the port stayed bound.
func TestCloseWithoutStartReleasesPort(t *testing.T) {
	probe, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()

	src, err := NewHTTPSource(port, "/ingest", engine.NewChannel("test", 1))
	if err != nil {
		t.Fatalf("NewHTTPSource: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("port %d still bound after Close on a never-started source: %v", port, err)
	}
	l.Close()
}
