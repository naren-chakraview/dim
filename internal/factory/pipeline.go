package factory

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/naren-chakraview/dim/internal/adapters/file"
	"github.com/naren-chakraview/dim/internal/adapters/http"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/ordering"
	"github.com/naren-chakraview/dim/internal/steps"
)

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
	sourceOutCh      *engine.Channel
	generationMgrs   map[string]*engine.GenerationManager
	defaultRoute     string
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewMessageRouter creates a new router for multi-route scenarios.
// It reads from sourceOutCh and routes to the appropriate generation manager based on message.Metadata.Route.
// M0.2.10: Updated to use GenerationManager for hot reload support.
func NewMessageRouter(sourceOutCh *engine.Channel, generationMgrs map[string]*engine.GenerationManager, defaultRoute string) *MessageRouter {
	return &MessageRouter{
		sourceOutCh:    sourceOutCh,
		generationMgrs: generationMgrs,
		defaultRoute:   defaultRoute,
	}
}

// Start begins the routing loop in a goroutine.
// It reads messages from the source channel and routes them to the correct executor.
func (r *MessageRouter) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.wg.Add(1)

	go func() {
		defer r.wg.Done()
		r.routeMessages()
	}()

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

// routeMessages reads from the source channel and routes messages to generation managers.
// M0.2.10: Updated to route to active generation for hot reload support.
func (r *MessageRouter) routeMessages() {
	for {
		// Read a message from the source
		msg, err := r.sourceOutCh.Recv(r.ctx)
		if err != nil {
			// Context cancelled or channel error
			log.Printf("[DEBUG] MessageRouter: recv error: %v", err)
			return
		}

		// If channel is closed, msg is nil
		if msg == nil {
			return
		}

		// Determine the target route
		routeName := msg.Metadata.Route
		if routeName == "" {
			routeName = r.defaultRoute
		}

		// Get the generation manager for this route
		gm, ok := r.generationMgrs[routeName]
		if !ok {
			log.Printf("[WARN] MessageRouter: no generation manager for route %q, correlation_id=%s", routeName, msg.Metadata.CorrelationID)
			continue
		}

		// Get the active generation's input channel and send the message
		inputCh := gm.GetActiveInputChannel()
		if inputCh == nil {
			log.Printf("[WARN] MessageRouter: no active generation for route %q, correlation_id=%s", routeName, msg.Metadata.CorrelationID)
			continue
		}

		if err := inputCh.Send(r.ctx, msg); err != nil {
			log.Printf("[WARN] MessageRouter: failed to send to route %q (active generation): %v", routeName, err)
		}
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
	return BuildSingleRoutePipelineWithTracing(ctx, cfg, nil)
}

// BuildSingleRoutePipelineWithTracing constructs the pipeline for a single route with tracing support (M0.5.1+).
// tracingProvider may be nil (no tracing).
// Exported for use in testing and direct single-route scenarios.
func BuildSingleRoutePipelineWithTracing(ctx context.Context, cfg *config.RouteConfig, tracingProvider *observability.TracingProvider) (
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

	// Create HTTP source
	var sources []SourceAdapter
	httpSrc, err := http.NewHTTPSource(8080, "/ingest", inputCh)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to create HTTP source: %w", err)
	}
	sources = append(sources, &HTTPSourceAdapter{
		source: httpSrc,
		outCh:  inputCh,
	})

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

	// Create executor with appropriate worker count, error channel, and tracing (M0.5.1+)
	var executor *engine.Executor
	if routeSpec.ErrorPath != nil {
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
	return BuildMultiRoutePipelineWithTracing(ctx, cfg, nil)
}

// BuildMultiRoutePipelineWithTracing constructs an end-to-end pipeline with multiple independent routes (M0.5.1+).
// Each route gets its own executor wrapped in a GenerationManager for hot reload support.
// Messages are routed based on Metadata.Route and always sent to the active generation.
//
// Returns (generationManagers, router, sources, sinks, error)
// M0.2.10: Updated to use GenerationManager for hot reload support (SIGHUP).
// M0.5.1+: Added tracing support
// tracingProvider may be nil (no tracing).
// Callers should manage the lifecycle of returned sources, sinks, managers, and router.
func BuildMultiRoutePipelineWithTracing(ctx context.Context, cfg *config.RouteConfig, tracingProvider *observability.TracingProvider) (
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
	sourceOutCh := engine.NewChannel("http-to-router", 100)

	// Create HTTP source (shared across all routes)
	var sources []SourceAdapter
	httpSrc, err := http.NewHTTPSource(8080, "/ingest", sourceOutCh)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to create HTTP source: %w", err)
	}
	sources = append(sources, &HTTPSourceAdapter{
		source: httpSrc,
		outCh:  sourceOutCh,
	})

	// Build generation managers for each route
	generationMgrs := make(map[string]*engine.GenerationManager)
	var sinks []SinkAdapter

	// Track which sinks we've already created to avoid duplicates
	createdSinks := make(map[string]bool)

	// First pass: create all route executors wrapped in generation managers and sinks
	for routeName, routeSpec := range cfg.Routes {
		// Create channels for this route
		outputCh := engine.NewChannel(fmt.Sprintf("executor-%s-to-sink", routeName), 100)
		var errorCh *engine.Channel
		if routeSpec.ErrorPath != nil {
			errorCh = engine.NewChannel(fmt.Sprintf("executor-%s-to-error-sink", routeName), 100)
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

		// Create executor with its own input channel and appropriate worker count (M0.5.1+: with tracing)
		executorInputCh := engine.NewChannel(fmt.Sprintf("router-to-executor-%s", routeName), 100)
		executor := engine.NewExecutorWithTracing(routeName, executorInputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers, retryPolicy, tracingProvider)

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

				errorSink, err := file.NewFileSink(errorSinkPath, errorCh)
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
	router := NewMessageRouter(sourceOutCh, generationMgrs, defaultRoute)

	return generationMgrs, router, sources, sinks, nil
}
