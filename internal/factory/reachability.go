package factory

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
)

// Reachability values. "unchecked" means this check cannot tell; it is never reported as
// reachable.
const (
	ReachabilityReachable   = "reachable"
	ReachabilityUnreachable = "unreachable"
	ReachabilityUnchecked   = "unchecked"
)

// SourceStatus is the reachability of one configured source.
type SourceStatus struct {
	Name         string
	Type         string
	Route        []string // routes that read from this source
	Reachability string
	Detail       string
}

// CheckSourceReachability reports, for each configured source, whether it can be
// reached from here. The check is a cheap probe of the endpoint the source would use:
// a file path, or a TCP connection to a broker or host. Where a probe is not possible
// without credentials or a driver, the status is "unchecked" and the detail says why.
func CheckSourceReachability(cfg *config.RouteConfig, dialTimeout time.Duration) []SourceStatus {
	routesBySource := make(map[string][]string)
	for name, route := range cfg.Routes {
		routesBySource[route.From] = append(routesBySource[route.From], name)
	}

	names := make([]string, 0, len(cfg.Sources))
	for name := range cfg.Sources {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []SourceStatus
	for _, name := range names {
		spec := cfg.Sources[name]
		st := SourceStatus{Name: name, Type: spec.Type, Route: routesBySource[name]}
		st.Reachability, st.Detail = probeSource(spec, dialTimeout)
		out = append(out, st)
	}
	return out
}

func probeSource(spec config.SourceSpec, timeout time.Duration) (string, string) {
	x := spec.Extra // type-specific keys; path and url are fields of SourceSpec itself
	switch spec.Type {
	case "file":
		info, err := os.Stat(spec.Path)
		if err != nil {
			return ReachabilityUnreachable, fmt.Sprintf("path %s: %v", spec.Path, err)
		}
		if !info.IsDir() {
			return ReachabilityUnreachable, fmt.Sprintf("path %s is not a directory", spec.Path)
		}
		return ReachabilityReachable, "directory " + spec.Path + " exists"

	case "sftp":
		host := stringOpt(x, "host", "")
		port := intOpt(x, "port", 22)
		return dialProbe(host, port, timeout)

	case "kafka":
		brokers := stringListOpt(x, "brokers")
		if len(brokers) == 0 {
			return ReachabilityUnreachable, "no brokers configured"
		}
		var failed []string
		for _, b := range brokers {
			host, port, err := splitHostPort(b, 9092)
			if err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", b, err))
				continue
			}
			if status, detail := dialProbe(host, port, timeout); status != ReachabilityReachable {
				failed = append(failed, detail)
			}
		}
		if len(failed) > 0 {
			return ReachabilityUnreachable, fmt.Sprintf("%d of %d brokers: %v", len(failed), len(brokers), failed)
		}
		return ReachabilityReachable, fmt.Sprintf("%d broker(s) accept connections", len(brokers))

	case "amqp":
		raw := spec.URL
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return ReachabilityUnreachable, fmt.Sprintf("invalid amqp url %q", raw)
		}
		host, port, err := splitHostPort(u.Host, 5672)
		if err != nil {
			return ReachabilityUnreachable, err.Error()
		}
		return dialProbe(host, port, timeout)

	case "s3":
		return ReachabilityUnchecked, "reaching S3 needs AWS credentials; not probed"
	case "database":
		return ReachabilityUnchecked, "reaching the database needs its driver and DSN; not probed"
	case "http", "":
		return ReachabilityUnchecked, "the HTTP ingest source listens on this host; not probed"
	}
	return ReachabilityUnchecked, fmt.Sprintf("no probe for source type %q", spec.Type)
}

func dialProbe(host string, port int, timeout time.Duration) (string, string) {
	if host == "" {
		return ReachabilityUnreachable, "no host configured"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return ReachabilityUnreachable, fmt.Sprintf("%s: %v", addr, err)
	}
	conn.Close()
	return ReachabilityReachable, addr + " accepts connections"
}

// splitHostPort splits host[:port], using defPort when the port is absent.
func splitHostPort(hostport string, defPort int) (string, int, error) {
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport, defPort, nil
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q", hostport)
	}
	return host, port, nil
}
