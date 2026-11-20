// Package daemon runs a route configuration: it builds the routes, starts their
// adapters and executors, records their state in the Tier 1 viewer, and applies
// hot reloads on SIGHUP. `dimctl run` and `dimd` both run through it, so there is
// one implementation of startup, reload and shutdown.
package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/observability/viewer"
	"github.com/naren-chakraview/dim/internal/tenant"
)

// ViewerAddrEnv overrides the viewer's listen address (default :8081).
const ViewerAddrEnv = "DIM_VIEWER_ADDR"

// viewerRingSize is the number of recent messages each route's viewer keeps.
const viewerRingSize = 100

// Serve runs cfg until SIGINT or SIGTERM: observability, the Tier 1 viewer, the
// routes, and hot reload from configPath on SIGHUP.
func Serve(configPath string, cfg *config.RouteConfig) error {
	obsConfig := observability.FromRouteConfig(cfg)
	if err := obsConfig.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}

	var metricsServer *http.Server
	if obsConfig.MetricsEnabled {
		collector := observability.NewMetricsCollector(obsConfig.PrometheusAddr)
		metricsServer = startMetricsServer(context.Background(), obsConfig.PrometheusAddr, collector)
		log.Printf("[INFO] metrics enabled on %s/metrics", obsConfig.PrometheusAddr)
	}

	var tracing *observability.TracingProvider
	if obsConfig.TracingEnabled {
		tracing = observability.NewTracingProvider(obsConfig.OtelExporter, obsConfig.JaegerEndpoint, obsConfig.SampleRate)
		log.Printf("[INFO] tracing enabled with %s exporter (sample_rate=%.2f)", obsConfig.OtelExporter, obsConfig.SampleRate)
	}

	viewerAddr := os.Getenv(ViewerAddrEnv)
	if viewerAddr == "" {
		viewerAddr = ":8081"
	}
	viewerSrv, viewerServer, err := viewer.StartServer(viewerAddr)
	if err != nil {
		return fmt.Errorf("failed to start viewer server: %w", err)
	}
	log.Printf("[INFO] Tier 1 viewer enabled on http://%s/debug/routes", viewerAddr)

	runErr := Run(Options{
		ConfigPath: configPath,
		Config:     cfg,
		Tracing:    tracing,
		Tenants:    tenant.NewManager(),
		Viewer:     viewerSrv,
	})

	if metricsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		metricsServer.Shutdown(shutdownCtx)
		cancel()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	viewerServer.Shutdown(shutdownCtx)
	cancel()
	return runErr
}

// Options configures Run.
type Options struct {
	ConfigPath string
	Config     *config.RouteConfig
	Tracing    *observability.TracingProvider
	Tenants    *tenant.Manager
	// Viewer, if set, receives the outcome of every message on every route.
	Viewer *viewer.ViewerServer
}

// Run builds and starts every route in opts.Config and blocks until SIGINT or
// SIGTERM, or until an executor fails.
func Run(opts Options) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gens, router, sources, sinks, err := factory.BuildMultiRoutePipelineWithTracing(ctx, opts.Config, opts.Tracing, opts.Tenants)
	if err != nil {
		return fmt.Errorf("failed to build pipeline: %w", err)
	}
	log.Printf("[INFO] configured %d route(s)", len(gens))

	for name, gm := range gens {
		gen := gm.GetActiveGeneration()
		if gen == nil {
			return fmt.Errorf("no active generation for route %q", name)
		}
		attachViewer(opts.Viewer, name, gen.Executor)
	}

	// Adapters block in Start while they serve (an HTTP source serves until its
	// context is cancelled), so each one runs in its own goroutine. A start
	// failure while the daemon is running is reported on adapterErrs.
	adapterErrs := make(chan error, len(sources)+len(sinks))
	for _, source := range sources {
		go func(source factory.SourceAdapter) {
			if err := source.Start(ctx); err != nil && ctx.Err() == nil {
				adapterErrs <- fmt.Errorf("source failed: %w", err)
			}
		}(source)
	}
	for _, sink := range sinks {
		go func(sink factory.SinkAdapter) {
			if err := sink.Start(ctx); err != nil && ctx.Err() == nil {
				adapterErrs <- fmt.Errorf("sink failed: %w", err)
			}
		}(sink)
	}
	if err := router.Start(ctx); err != nil {
		return fmt.Errorf("failed to start message router: %w", err)
	}

	state := &reloadState{
		cfg:     opts.Config,
		gens:    gens,
		viewer:  opts.Viewer,
		exits:   make(chan exitResult, 256),
		cancels: make(map[*engine.Executor]context.CancelFunc),
	}
	for name, gm := range gens {
		state.startExecutor(ctx, name, gm.GetActiveGeneration().Executor)
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	hupCh := make(chan os.Signal, 1)
	signal.Notify(hupCh, syscall.SIGHUP)
	reloadDone := make(chan struct{})
	go func() {
		defer close(reloadDone)
		for range hupCh {
			state.reload(ctx, opts.ConfigPath)
		}
	}()
	log.Printf("[INFO] routes running; send SIGHUP to reload %s", opts.ConfigPath)

	var runErr error
wait:
	for {
		select {
		case <-sigCh:
			log.Printf("[INFO] shutdown signal received")
			break wait
		case e := <-state.exits:
			// An executor that exits with "context canceled" was stopped on purpose
			// after a reload drained it. Only a real failure ends the daemon.
			if e.err != nil && !strings.Contains(e.err.Error(), "context canceled") {
				runErr = fmt.Errorf("route %q executor failed: %w", e.route, e.err)
				break wait
			}
		case err := <-adapterErrs:
			runErr = err
			break wait
		}
	}

	signal.Stop(sigCh)
	signal.Stop(hupCh)
	close(hupCh)
	<-reloadDone

	// Stop the router and executors, drain in-flight work, then close adapters.
	cancel()
	for name, gm := range gens {
		drainCtx, drainCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := gm.DrainAll(drainCtx, 10*time.Second); err != nil {
			log.Printf("[WARN] route %q drain error: %v", name, err)
		}
		drainCancel()
	}
	for _, source := range sources {
		if err := source.Stop(); err != nil {
			log.Printf("[WARN] error stopping source: %v", err)
		}
	}
	for _, sink := range sinks {
		if err := sink.Stop(); err != nil {
			log.Printf("[WARN] error stopping sink: %v", err)
		}
	}
	log.Printf("[INFO] shutdown complete")
	return runErr
}

// exitResult reports that a route's executor returned.
type exitResult struct {
	route string
	err   error
}

// reloadState is the running configuration, its generation managers and the
// executors' cancel functions. mu serializes reloads; the cancel map has its own
// lock because drained executors are stopped from a background goroutine.
type reloadState struct {
	mu       sync.Mutex
	cfg      *config.RouteConfig
	gens     map[string]*engine.GenerationManager
	viewer   *viewer.ViewerServer
	exits    chan exitResult
	cancelMu sync.Mutex
	cancels  map[*engine.Executor]context.CancelFunc
}

// startExecutor runs an executor under its own context, so that it can be
// stopped on its own once a reload has drained it.
func (s *reloadState) startExecutor(ctx context.Context, route string, ex *engine.Executor) {
	execCtx, cancel := context.WithCancel(ctx)
	s.cancelMu.Lock()
	s.cancels[ex] = cancel
	s.cancelMu.Unlock()
	go func() {
		s.exits <- exitResult{route: route, err: ex.Run(execCtx)}
	}()
}

// stopExecutor cancels an executor's run loop. Called once its generation has drained.
func (s *reloadState) stopExecutor(ex *engine.Executor) {
	s.cancelMu.Lock()
	cancel := s.cancels[ex]
	delete(s.cancels, ex)
	s.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// reload re-reads the config file and takes over each route whose route_version
// changed. A failed reload leaves the running routes as they are.
func (s *reloadState) reload(ctx context.Context, configPath string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if configPath == "" {
		log.Printf("[WARN] reload requested but config path not provided")
		return
	}
	newCfg, err := config.LoadRouteConfig(configPath)
	if err != nil {
		log.Printf("[ERROR] reload failed: %v", err)
		return
	}
	log.Printf("[INFO] hot reload triggered, checking %d route(s)", len(newCfg.Routes))

	for routeName, newRoute := range newCfg.Routes {
		oldRoute, ok := s.cfg.Routes[routeName]
		if !ok {
			log.Printf("[WARN] route %q added (not reloading, requires restart)", routeName)
			continue
		}
		if oldRoute.RouteVersion == newRoute.RouteVersion {
			log.Printf("[INFO] route %q unchanged (version=%s)", routeName, truncateVersion(newRoute.RouteVersion))
			continue
		}

		log.Printf("[INFO] reloading route %q (version %s → %s)",
			routeName, truncateVersion(oldRoute.RouteVersion), truncateVersion(newRoute.RouteVersion))

		gm := s.gens[routeName]
		previous := gm.GetActiveGeneration()

		// The sinks are bound to the running channels, so an error_path change cannot
		// take effect on a live route. Refuse it and say so.
		if (oldRoute.ErrorPath == nil) != (newRoute.ErrorPath == nil) ||
			(newRoute.ErrorPath != nil && oldRoute.ErrorPath.Target != newRoute.ErrorPath.Target) {
			log.Printf("[WARN] route %q error_path changed (not reloading, requires restart)", routeName)
			continue
		}

		newExecutor, err := BuildExecutorForRoute(ctx, routeName, newRoute,
			previous.Executor.GetOutputChannel(), previous.Executor.GetErrorChannel())
		if err != nil {
			log.Printf("[ERROR] failed to build executor for %q: %v", routeName, err)
			continue
		}
		attachViewer(s.viewer, routeName, newExecutor)

		if err := gm.Takeover(ctx, newExecutor, newRoute.RouteVersion); err != nil {
			log.Printf("[ERROR] takeover failed for %q: %v", routeName, err)
			continue
		}
		// Takeover only makes the new generation active. Start its executor, or
		// its input is never read. Stop the previous executor once it has drained.
		s.startExecutor(ctx, routeName, newExecutor)
		go func(prev *engine.Generation) {
			if err := prev.WaitForDrained(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[WARN] generation %d of %q did not drain: %v", prev.ID, routeName, err)
			}
			s.stopExecutor(prev.Executor)
		}(previous)
		s.cfg.Routes[routeName] = newRoute
	}
}

// attachViewer makes a route's executor report its messages to the viewer.
func attachViewer(v *viewer.ViewerServer, route string, ex *engine.Executor) {
	if v == nil || ex == nil {
		return
	}
	ex.SetObserver(v.GetOrCreateViewer(route, viewerRingSize))
}
