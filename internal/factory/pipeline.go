package factory

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/naren-chakraview/dim/internal/adapters/file"
	"github.com/naren-chakraview/dim/internal/adapters/http"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/ordering"
	"github.com/naren-chakraview/dim/internal/steps"
	"github.com/naren-chakraview/dim/internal/tenant"
)

// getHTTPPort returns the port for HTTP source from DIM_HTTP_PORT env var, or 8080 as default
func getHTTPPort() int {
	if portStr := os.Getenv("DIM_HTTP_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port < 65536 {
			return port
		}
	}
	return 8080
}

// SourceAdapter wraps a source adapter with lifecycle methods
type SourceAdapter interface {
	// Start begins listening for messages
	Start(ctx context.Context) error

	// GetOutputChannel returns the channel to which messages are written
	GetOutputChannel() *engine.Channel

	// Stop gracefully shuts down the adapter
	Stop() error
}

// SinkAdapter wraps a sink adapter with lifecycle methods
type SinkAdapter interface {
	// Start begins listening to the input channel and processing messages
	Start(ctx context.Context) error

	// Stop gracefully shuts down the adapter
	Stop() error
}

// HTTPSourceAdapter wraps the HTTPSource to implement SourceAdapter
type HTTPSourceAdapter struct {
	source *http.HTTPSource
	outCh  *engine.Channel
}

func (a *HTTPSourceAdapter) Start(ctx context.Context) error {
	return a.source.Start(ctx)
}

func (a *HTTPSourceAdapter) GetOutputChannel() *engine.Channel {
	return a.outCh
}

func (a *HTTPSourceAdapter) Stop() error {
	return a.source.Close()
}

// FileSinkAdapter wraps the FileSink to implement SinkAdapter
type FileSinkAdapter struct {
	sink *file.FileSink
}

func (a *FileSinkAdapter) Start(ctx context.Context) error {
	a.sink.Start(ctx)
	return nil
}

func (a *FileSinkAdapter) Stop() error {
	return a.sink.Close()
}

// MessageRouter routes messages from a source to the correct executor based on route name.
// It reads from a source channel and sends to the appropriate executor's input channel.
// Updated for M0.2.10: works with GenerationManager for hot reload support.
type MessageRouter struct {
	inputs         []RouterInput
	generationMgrs map[string]*engine.GenerationManager
	defaultRoute   string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// RouterInput is one source's output channel and the routes that read from it.
type RouterInput struct {
	Ch *engine.Channel
	// Routes are the routes bound to this source (its `from:` routes). A message
	// goes to each of them; with several, each gets its own copy.
	Routes []string
	// HonorMessageRoute: the message's own Metadata.Route selects the route (the HTTP
	// source sets it from the X-Route header). Other sources do not set it.
	HonorMessageRoute bool
}

// NewMessageRouter creates a router with one HTTP-style input: messages go to the
// route named in Metadata.Route, or to defaultRoute when none is named.
func NewMessageRouter(sourceOutCh *engine.Channel, generationMgrs map[string]*engine.GenerationManager, defaultRoute string) *MessageRouter {
	return NewSourceRouter([]RouterInput{{Ch: sourceOutCh, HonorMessageRoute: true}}, generationMgrs, defaultRoute)
}

// NewSourceRouter creates a router that reads each input and sends its messages to
// the routes bound to that input. An input with no bound routes falls back to
// defaultRoute. Each route's generation manager is used at send time, so a hot
// reload takes effect for the next message.
func NewSourceRouter(inputs []RouterInput, generationMgrs map[string]*engine.GenerationManager, defaultRoute string) *MessageRouter {
	return &MessageRouter{
		inputs:         inputs,
		generationMgrs: generationMgrs,
		defaultRoute:   defaultRoute,
	}
}

// Start starts one routing goroutine per input.
func (r *MessageRouter) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)
	for _, in := range r.inputs {
		in := in
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.routeMessages(in)
		}()
	}
	return nil
}

// Stop gracefully stops the router.
func (r *MessageRouter) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	return nil
}

// targets returns the routes a message from in goes to.
func (r *MessageRouter) targets(in RouterInput, msg *engine.Message) []string {
	if in.HonorMessageRoute && msg.Metadata.Route != "" {
		return []string{msg.Metadata.Route}
	}
	if len(in.Routes) == 0 {
		return []string{r.defaultRoute}
	}
	return in.Routes
}

// routeMessages reads one input and delivers each message to its target routes.
func (r *MessageRouter) routeMessages(in RouterInput) {
	for {
		msg, err := in.Ch.Recv(r.ctx)
		if err != nil {
			log.Printf("[DEBUG] MessageRouter: recv error: %v", err)
			return
		}
		if msg == nil {
			return
		}

		targets := r.targets(in, msg)
		for _, routeName := range targets {
			out := msg
			if len(targets) > 1 {
				// Each route gets its own copy, so one route's steps cannot change another's.
				cp := *msg
				cp.Metadata.Route = routeName
				out = &cp
			}
			r.deliver(routeName, out)
		}
	}
}

// deliver sends a message to a route's active generation.
func (r *MessageRouter) deliver(routeName string, msg *engine.Message) {
	gm, ok := r.generationMgrs[routeName]
	if !ok {
		log.Printf("[WARN] MessageRouter: no generation manager for route %q, correlation_id=%s", routeName, msg.Metadata.CorrelationID)
		return
	}
	inputCh := gm.GetActiveInputChannel()
	if inputCh == nil {
		log.Printf("[WARN] MessageRouter: no active generation for route %q, correlation_id=%s", routeName, msg.Metadata.CorrelationID)
		return
	}
	if err := inputCh.Send(r.ctx, msg); err != nil {
		log.Printf("[WARN] MessageRouter: failed to send to route %q (active generation): %v", routeName, err)
	}
}

// BuildPipeline constructs the end-to-end pipeline from a RouteConfig.
// Handles both single-route and multi-route scenarios:
// - Single route: returns executor, channels, sources, sinks (backward compatible)
// - Multiple routes: returns multiple executors coordinated via MessageRouter
//
// For single route, returns (executor, inputCh, outputCh, errorCh, sources, sinks, error)
// For multiple routes, returns (nil, nil, nil, nil, sources, sinks, error) and manages executors internally
//
// Deprecated: Use BuildMultiRoutePipeline for multi-route scenarios.
// This function maintains backward compatibility but delegates to BuildMultiRoutePipeline.
func BuildPipeline(ctx context.Context, cfg *config.RouteConfig) (
	*engine.Executor,
	*engine.Channel,
	*engine.Channel,
	*engine.Channel,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	if len(cfg.Routes) == 0 {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("no routes configured")
	}

	// For single route, use the old implementation
	if len(cfg.Routes) == 1 {
		return BuildSingleRoutePipeline(ctx, cfg)
	}

	// For multiple routes, use the new multi-route implementation
	// Note: This returns nil for executor/channels and manages them internally via router
	generationMgrs, router, sources, sinks, err := BuildMultiRoutePipeline(ctx, cfg)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	// Start the router (it will coordinate message routing)
	if err := router.Start(ctx); err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to start message router: %w", err)
	}

	// Return router as a special executor wrapper for compatibility
	// Actually, for multi-route we need a different approach in main.go
	// For now, we'll return nil and let main.go handle multiple generation managers directly
	log.Printf("[INFO] BuildPipeline: configured %d routes", len(generationMgrs))
	_ = router // router is managed by the caller

	return nil, nil, nil, nil, sources, sinks, nil
}

// BuildSingleRoutePipeline constructs the pipeline for a single route (original M0.1 implementation).
// Exported for use in testing and direct single-route scenarios.
// Deprecated: Use BuildSingleRoutePipelineWithTracing for tracing support
func BuildSingleRoutePipeline(ctx context.Context, cfg *config.RouteConfig) (
	*engine.Executor,
	*engine.Channel,
	*engine.Channel,
	*engine.Channel,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	return BuildSingleRoutePipelineWithTracing(ctx, cfg, nil, nil)
}

// BuildSingleRoutePipelineWithTracing constructs the pipeline for a single route with tracing support (M0.5.1+).
// tracingProvider may be nil (no tracing).
// tenantManager may be nil (no multi-tenant isolation); if provided, enables per-domain rate limiting and worker slot allocation (M3.4+).
// Exported for use in testing and direct single-route scenarios.
func BuildSingleRoutePipelineWithTracing(ctx context.Context, cfg *config.RouteConfig, tracingProvider *observability.TracingProvider, tenantManager *tenant.Manager) (
	*engine.Executor,
	*engine.Channel,
	*engine.Channel,
	*engine.Channel,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	// Get the first (and only) route
	var routeName string
	var routeSpec config.RouteSpec
	for name, spec := range cfg.Routes {
		routeName = name
		routeSpec = spec
		break
	}

	// Create bounded channels
	inputCh := engine.NewChannel("http-to-executor", 100)
	outputCh := engine.NewChannel("executor-to-sink", 100)
	errorCh := engine.NewChannel("executor-to-error-sink", 100)

	// The route's source adapters (from its `sources:` entries) write into inputCh.
	resolver, err := secretResolver(ctx)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("secret provider: %w", err)
	}
	bindings, err := buildSourceBindings(ctx, cfg, resolver)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	var sources []SourceAdapter
	for _, b := range bindings {
		if !containsString(b.routes, routeName) {
			continue
		}
		sources = append(sources, b.adapter)
		go forwardMessages(ctx, b.out, inputCh)
	}

	// Build contract store from route config (M0.3.5)
	contractStore := config.NewContractStore()
	if len(routeSpec.Contracts) > 0 {
		if err := contractStore.LoadContracts(routeName, routeSpec.Contracts); err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to load contracts for route %q: %w", routeName, err)
		}
	}

	// Build steps from route config (dedupStore is nil for single-instance mode)
	stepsInstances, stepNames, err := steps.BuildStepsFromSpec(routeSpec.Steps, contractStore, routeName, nil)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to build steps: %w", err)
	}

	// Determine worker count based on ordering requirement
	// Default to 4 workers for parallel processing (M0.2+)
	// Constrain to 1 worker if ordering is required
	defaultWorkers := 4
	numWorkers := ordering.GetWorkerCount(&routeSpec, defaultWorkers)
	if ordering.IsOrderingRequired(&routeSpec) {
		log.Printf("[INFO] Route %q has ordering=required, constraining to 1 worker for serial processing", routeName)
	}

	// Extract retry policy from route config if present
	var retryPolicy *engine.RetryPolicy
	if routeSpec.Retry != nil {
		retryPolicy = &engine.RetryPolicy{
			MaxAttempts: routeSpec.Retry.MaxAttempts,
			BackoffMs:   routeSpec.Retry.BackoffMs,
			JitterMs:    routeSpec.Retry.JitterMs,
		}
	}

	// Extract domain for multi-tenant isolation (M3.4+)
	domain := routeSpec.Domain
	var rateLimiter *tenant.MessageRateLimiter
	var slotManager *tenant.WorkerSlotManager

	// Create tenant limiters if tenant manager is provided (M3.4+)
	if tenantManager != nil && domain != "" {
		tenantCfg := tenantManager.GetTenant(domain)
		rateLimiter = tenant.NewMessageRateLimiter(tenantManager)
		slotManager = tenant.NewWorkerSlotManager(tenantManager)
		log.Printf("[INFO] Route %q: tenant isolation enabled for domain %q (rate_limit=%d msg/sec, worker_slots=%d)", routeName, domain, tenantCfg.MessageRateLimit, tenantCfg.WorkerSlots)
	}

	// Create executor with appropriate worker count, error channel, tracing, and tenant isolation (M0.5.1+, M3.4+)
	var executor *engine.Executor
	if rateLimiter != nil || slotManager != nil {
		// Use tenant-aware executor
		executor = engine.NewExecutorWithTenantLimiting(routeName, inputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider, rateLimiter, slotManager, domain)
	} else if routeSpec.ErrorPath != nil {
		executor = engine.NewExecutorWithTracing(routeName, inputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider)
	} else {
		executor = engine.NewExecutorWithTracing(routeName, inputCh, outputCh, nil, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider)
	}

	// Create sinks
	var sinks []SinkAdapter

	// Output sink (required)
	outputSinkSpec, ok := cfg.Sinks["output"]
	if !ok {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("required sink 'output' not configured")
	}

	outputSinkPath := outputSinkSpec.Path
	if outputSinkPath == "" {
		outputSinkPath = "./output/messages.jsonl"
	}

	outputSink, err := file.NewFileSink(outputSinkPath, outputCh)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to create output sink: %w", err)
	}
	sinks = append(sinks, &FileSinkAdapter{sink: outputSink})

	// Error sink (optional)
	if routeSpec.ErrorPath != nil {
		errorSinkName := routeSpec.ErrorPath.Target
		errorSinkSpec, ok := cfg.Sinks[errorSinkName]
		if !ok {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("error sink '%s' not configured", errorSinkName)
		}

		errorSinkPath := errorSinkSpec.Path
		if errorSinkPath == "" {
			errorSinkPath = "./output/errors.jsonl"
		}

		errorSink, err := file.NewFileSink(errorSinkPath, errorCh)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to create error sink: %w", err)
		}
		sinks = append(sinks, &FileSinkAdapter{sink: errorSink})
	}

	return executor, inputCh, outputCh, errorCh, sources, sinks, nil
}

// BuildMultiRoutePipeline constructs an end-to-end pipeline with multiple independent routes.
// Each route gets its own executor wrapped in a GenerationManager for hot reload support.
// Messages are routed based on Metadata.Route and always sent to the active generation.
//
// Returns (generationManagers, router, sources, sinks, error)
// M0.2.10: Updated to use GenerationManager for hot reload support (SIGHUP).
// Deprecated: Use BuildMultiRoutePipelineWithTracing for tracing support
// Callers should manage the lifecycle of returned sources, sinks, managers, and router.
func BuildMultiRoutePipeline(ctx context.Context, cfg *config.RouteConfig) (
	map[string]*engine.GenerationManager,
	*MessageRouter,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	return BuildMultiRoutePipelineWithTracing(ctx, cfg, nil, nil)
}

// BuildMultiRoutePipelineWithTracing constructs an end-to-end pipeline with multiple independent routes (M0.5.1+).
// Each route gets its own executor wrapped in a GenerationManager for hot reload support.
// Messages are routed based on Metadata.Route and always sent to the active generation.
//
// Returns (generationManagers, router, sources, sinks, error)
// M0.2.10: Updated to use GenerationManager for hot reload support (SIGHUP).
// M0.5.1+: Added tracing support
// M3.4+: Added multi-tenant isolation support
// tracingProvider may be nil (no tracing).
// tenantManager may be nil (no multi-tenant isolation); if provided, enables per-domain rate limiting and worker slot allocation (M3.4+).
// Callers should manage the lifecycle of returned sources, sinks, managers, and router.
func BuildMultiRoutePipelineWithTracing(ctx context.Context, cfg *config.RouteConfig, tracingProvider *observability.TracingProvider, tenantManager *tenant.Manager) (
	map[string]*engine.GenerationManager,
	*MessageRouter,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	if len(cfg.Routes) == 0 {
		return nil, nil, nil, nil, fmt.Errorf("no routes configured")
	}

	// Create a shared channel for the HTTP source
	// The router will read from this and distribute to individual generation managers
	// One adapter per configured source; each source feeds the routes that read from it.
	resolver, err := secretResolver(ctx)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("secret provider: %w", err)
	}
	bindings, err := buildSourceBindings(ctx, cfg, resolver)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	var sources []SourceAdapter
	var routerInputs []RouterInput
	for _, b := range bindings {
		sources = append(sources, b.adapter)
		routerInputs = append(routerInputs, RouterInput{
			Ch:                b.out,
			Routes:            b.routes,
			HonorMessageRoute: b.typ == "http" || b.typ == "",
		})
	}

	// Build generation managers for each route
	generationMgrs := make(map[string]*engine.GenerationManager)
	var sinks []SinkAdapter

	// Track which sinks we've already created to avoid duplicates
	createdSinks := make(map[string]bool)

	// Every route writes to one shared output channel, and to one shared error
	// channel per error sink. Each sink is created once and reads its shared
	// channel, so all routes reach it. (A channel per route would connect only the
	// first route to the sink, and the others' results would never be written.)
	outputCh := engine.NewChannel("executors-to-output-sink", 100)
	errorChs := make(map[string]*engine.Channel)

	// First pass: create all route executors wrapped in generation managers and sinks
	for routeName, routeSpec := range cfg.Routes {
		var errorCh *engine.Channel
		if routeSpec.ErrorPath != nil {
			target := routeSpec.ErrorPath.Target
			if errorChs[target] == nil {
				errorChs[target] = engine.NewChannel(fmt.Sprintf("executors-to-error-sink-%s", target), 100)
			}
			errorCh = errorChs[target]
		}

		// Build contract store from route config (M0.3.5)
		contractStore := config.NewContractStore()
		if len(routeSpec.Contracts) > 0 {
			if err := contractStore.LoadContracts(routeName, routeSpec.Contracts); err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to load contracts for route %q: %w", routeName, err)
			}
		}

		// Build steps from route config (dedupStore is nil for single-instance mode)
		stepsInstances, stepNames, err := steps.BuildStepsFromSpec(routeSpec.Steps, contractStore, routeName, nil)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to build steps for route %q: %w", routeName, err)
		}

		// Determine worker count based on ordering requirement
		// Default to 4 workers for parallel processing (M0.2+)
		// Constrain to 1 worker if ordering is required
		defaultWorkers := 4
		numWorkers := ordering.GetWorkerCount(&routeSpec, defaultWorkers)
		if ordering.IsOrderingRequired(&routeSpec) {
			log.Printf("[INFO] Route %q has ordering=required, constraining to 1 worker for serial processing", routeName)
		}

		// Extract retry policy from route config if present
		var retryPolicy *engine.RetryPolicy
		if routeSpec.Retry != nil {
			retryPolicy = &engine.RetryPolicy{
				MaxAttempts: routeSpec.Retry.MaxAttempts,
				BackoffMs:   routeSpec.Retry.BackoffMs,
				JitterMs:    routeSpec.Retry.JitterMs,
			}
		}

		// Extract domain for multi-tenant isolation (M3.4+)
		domain := routeSpec.Domain
		var rateLimiter *tenant.MessageRateLimiter
		var slotManager *tenant.WorkerSlotManager

		// Create tenant limiters if tenant manager is provided (M3.4+)
		if tenantManager != nil && domain != "" {
			tenantCfg := tenantManager.GetTenant(domain)
			rateLimiter = tenant.NewMessageRateLimiter(tenantManager)
			slotManager = tenant.NewWorkerSlotManager(tenantManager)
			log.Printf("[INFO] Route %q: tenant isolation enabled for domain %q (rate_limit=%d msg/sec, worker_slots=%d)", routeName, domain, tenantCfg.MessageRateLimit, tenantCfg.WorkerSlots)
		}

		// Create executor with its own input channel and appropriate worker count (M0.5.1+: with tracing, M3.4+: with tenant isolation)
		executorInputCh := engine.NewChannel(fmt.Sprintf("router-to-executor-%s", routeName), 100)
		var executor *engine.Executor
		if rateLimiter != nil || slotManager != nil {
			// Use tenant-aware executor
			executor = engine.NewExecutorWithTenantLimiting(routeName, executorInputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider, rateLimiter, slotManager, domain)
		} else {
			executor = engine.NewExecutorWithTracing(routeName, executorInputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider)
		}

		// Wrap executor in a GenerationManager with concurrent-draining cap of 3
		generationMgr := engine.NewGenerationManager(routeName, executor, routeSpec.RouteVersion, 3)
		generationMgrs[routeName] = generationMgr

		// Create output sink for this route (if not already created)
		outputSinkName := "output"
		if !createdSinks[outputSinkName] {
			outputSinkSpec, ok := cfg.Sinks[outputSinkName]
			if !ok {
				return nil, nil, nil, nil, fmt.Errorf("required sink 'output' not configured")
			}

			outputSinkPath := outputSinkSpec.Path
			if outputSinkPath == "" {
				outputSinkPath = "./output/messages.jsonl"
			}

			outputSink, err := file.NewFileSink(outputSinkPath, outputCh)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("failed to create output sink for route %q: %w", routeName, err)
			}
			sinks = append(sinks, &FileSinkAdapter{sink: outputSink})
			createdSinks[outputSinkName] = true
		}

		// Create error sink for this route (if configured)
		if routeSpec.ErrorPath != nil {
			errorSinkName := routeSpec.ErrorPath.Target
			if !createdSinks[errorSinkName] {
				errorSinkSpec, ok := cfg.Sinks[errorSinkName]
				if !ok {
					return nil, nil, nil, nil, fmt.Errorf("error sink '%s' not configured for route %q", errorSinkName, routeName)
				}

				errorSinkPath := errorSinkSpec.Path
				if errorSinkPath == "" {
					errorSinkPath = "./output/errors.jsonl"
				}

				errorSink, err := file.NewFileSink(errorSinkPath, errorChs[errorSinkName])
				if err != nil {
					return nil, nil, nil, nil, fmt.Errorf("failed to create error sink for route %q: %w", routeName, err)
				}
				sinks = append(sinks, &FileSinkAdapter{sink: errorSink})
				createdSinks[errorSinkName] = true
			}
		}
	}

	// Determine default route (first route in map)
	var defaultRoute string
	for routeName := range generationMgrs {
		defaultRoute = routeName
		break
	}

	// Create the message router with generation managers
	router := NewSourceRouter(routerInputs, generationMgrs, defaultRoute)

	return generationMgrs, router, sources, sinks, nil
}

// forwardMessages copies messages from a source's output to a route's input until ctx ends.
func forwardMessages(ctx context.Context, from, to *engine.Channel) {
	for {
		msg, err := from.Recv(ctx)
		if err != nil || msg == nil {
			return
		}
		if err := to.Send(ctx, msg); err != nil {
			return
		}
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
