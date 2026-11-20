package factory

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/naren-chakraview/dim/internal/adapters/amqp"
	"github.com/naren-chakraview/dim/internal/adapters/database"
	"github.com/naren-chakraview/dim/internal/adapters/file"
	"github.com/naren-chakraview/dim/internal/adapters/http"
	"github.com/naren-chakraview/dim/internal/adapters/kafka"
	"github.com/naren-chakraview/dim/internal/adapters/s3"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
)

// boundSource is one configured source, its adapter, and the routes that read from it.
type boundSource struct {
	name    string
	typ     string
	domain  string // the domain of every route that reads from this source
	routes  []string
	adapter SourceAdapter
	out     *engine.Channel
}

// secretResolver returns the resolver every source uses for ${SECRET:...} references.
// No secret store is populated in production yet, so any reference is refused: a
// reference that cannot be resolved in its domain is an error, never an env lookup.
func secretResolver() *secrets.Resolver {
	return secrets.NewResolver(secrets.NewInMemoryStore())
}

// buildSourceBindings creates one adapter per configured source, each with its own
// output channel, and binds each source to the routes whose `from:` names it.
//
// A source belongs to one domain: every route that reads from it must carry the same
// domain. Routes that name a source the config does not define are an error. When the
// config defines no sources at all, the HTTP ingest source is used, as before.
func buildSourceBindings(ctx context.Context, cfg *config.RouteConfig, resolver *secrets.Resolver) ([]*boundSource, error) {
	routeNames := make([]string, 0, len(cfg.Routes))
	for name := range cfg.Routes {
		routeNames = append(routeNames, name)
	}
	sort.Strings(routeNames)

	// Which routes read from which source, and in which domain.
	routesBySource := make(map[string][]string)
	domainBySource := make(map[string]string)
	for _, routeName := range routeNames {
		route := cfg.Routes[routeName]
		sourceName := route.From
		if len(cfg.Sources) == 0 {
			sourceName = defaultHTTPSourceName
		} else if _, ok := cfg.Sources[sourceName]; !ok {
			return nil, fmt.Errorf("route %q reads from source %q, which is not defined in sources", routeName, sourceName)
		}
		if prev, seen := domainBySource[sourceName]; seen && prev != route.Domain {
			return nil, fmt.Errorf("source %q is read by routes in domains %q and %q; a source must belong to one domain", sourceName, prev, route.Domain)
		}
		domainBySource[sourceName] = route.Domain
		routesBySource[sourceName] = append(routesBySource[sourceName], routeName)
	}

	sourceNames := make([]string, 0, len(cfg.Sources)+1)
	for name := range cfg.Sources {
		sourceNames = append(sourceNames, name)
	}
	if len(cfg.Sources) == 0 {
		sourceNames = append(sourceNames, defaultHTTPSourceName)
	}
	sort.Strings(sourceNames)

	var bindings []*boundSource
	for _, name := range sourceNames {
		routes := routesBySource[name]
		if len(routes) == 0 {
			log.Printf("[WARN] source %q is defined but no route reads from it; it will not be started", name)
			continue
		}
		spec, ok := cfg.Sources[name]
		if !ok {
			spec = config.SourceSpec{Type: "http"}
		}
		out := engine.NewChannel("source-"+name+"-to-router", 100)
		adapter, err := buildSourceAdapter(ctx, name, spec, out, domainBySource[name], resolver)
		if err != nil {
			return nil, fmt.Errorf("source %q: %w", name, err)
		}
		bindings = append(bindings, &boundSource{
			name:    name,
			typ:     spec.Type,
			domain:  domainBySource[name],
			routes:  routes,
			adapter: adapter,
			out:     out,
		})
	}
	return bindings, nil
}

// defaultHTTPSourceName names the HTTP ingest source used when the config has none.
const defaultHTTPSourceName = "http-source"

// funcSource adapts an adapter's start and stop functions to SourceAdapter.
type funcSource struct {
	out   *engine.Channel
	start func(context.Context) error
	stop  func() error
}

func (s *funcSource) Start(ctx context.Context) error   { return s.start(ctx) }
func (s *funcSource) GetOutputChannel() *engine.Channel { return s.out }
func (s *funcSource) Stop() error                       { return s.stop() }

// buildSourceAdapter constructs the adapter for one source type. Each adapter writes
// to out, and resolves its ${SECRET:...} config values for domain.
func buildSourceAdapter(ctx context.Context, name string, spec config.SourceSpec, out *engine.Channel, domain string, resolver *secrets.Resolver) (SourceAdapter, error) {
	x := spec.Extra
	switch spec.Type {
	case "http", "":
		src, err := http.NewHTTPSource(getHTTPPort(), "/ingest", out)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP source: %w", err)
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "file":
		schedule := stringOpt(x, "schedule", "5s")
		src, err := file.NewFileSourceWithDomain(spec.Path, schedule, out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "sftp":
		port := intOpt(x, "port", 22)
		password := stringOpt(x, "password", "")
		schedule := stringOpt(x, "schedule", "5s")
		src, err := file.NewSFTPSourceWithDomain(stringOpt(x, "host", ""), port, stringOpt(x, "user", ""), password, spec.Path, schedule, out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "kafka":
		brokers := stringListOpt(x, "brokers")
		src, err := kafka.NewKafkaSourceWithDomain(brokers, stringOpt(x, "topic", ""), stringOpt(x, "group_id", ""), out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "amqp":
		// url is a SourceSpec field (the yaml 'url' key), not an extra key.
		src, err := amqp.NewAMQPSourceWithDomain(spec.URL, stringOpt(x, "queue", ""), out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "s3":
		cfg := s3.SourceConfig{
			Endpoint:     stringOpt(x, "endpoint", ""),
			Bucket:       stringOpt(x, "bucket", ""),
			Prefix:       stringOpt(x, "prefix", ""),
			Region:       stringOpt(x, "region", ""),
			PollInterval: durationOpt(x, "poll_interval", 30*time.Second),
		}
		src, err := s3.NewS3SourceWithDomain(cfg, out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil

	case "database":
		cfg := &database.SourceConfig{
			Driver:           stringOpt(x, "driver", ""),
			DSN:              stringOpt(x, "dsn", ""),
			Query:            stringOpt(x, "query", ""),
			WatermarkColumn:  stringOpt(x, "watermark_column", ""),
			WatermarkType:    stringOpt(x, "watermark_type", ""),
			InitialWatermark: stringOpt(x, "initial_watermark", ""),
			Schedule:         durationOpt(x, "schedule", 30*time.Second),
			BatchSize:        intOpt(x, "batch_size", 0),
			TimeoutSec:       intOpt(x, "timeout_sec", 0),
		}
		src, err := database.NewDatabaseSourceWithDomain(cfg, out, domain, resolver)
		if err != nil {
			return nil, err
		}
		return &funcSource{out: out, start: src.Start, stop: src.Close}, nil
	}
	return nil, fmt.Errorf("unsupported source type %q", spec.Type)
}

func stringOpt(x map[string]interface{}, key, def string) string {
	if v, ok := x[key].(string); ok {
		return v
	}
	return def
}

func intOpt(x map[string]interface{}, key string, def int) int {
	switch v := x[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return def
}

func durationOpt(x map[string]interface{}, key string, def time.Duration) time.Duration {
	if s, ok := x[key].(string); ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return def
}

func stringListOpt(x map[string]interface{}, key string) []string {
	var out []string
	switch v := x[key].(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
	case string:
		for _, s := range strings.Split(v, ",") {
			out = append(out, strings.TrimSpace(s))
		}
	}
	return out
}
