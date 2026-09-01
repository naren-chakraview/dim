package factory

import (
	"context"
	"fmt"

	"github.com/naren-chakraview/dim/internal/adapters/file"
	"github.com/naren-chakraview/dim/internal/adapters/http"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
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

// BuildPipeline constructs the end-to-end pipeline from a RouteConfig.
// For Phase 0, it assumes:
// - Exactly one route in cfg.Routes (first route is used)
// - Single HTTP source (type "http") named "http-source"
// - Single output sink and optional error sink
//
// Returns (executor, inputCh, outputCh, errorCh, sources, sinks, error)
// Callers should manage the lifecycle of returned sources and sinks.
func BuildPipeline(ctx context.Context, cfg *config.RouteConfig) (
	*engine.Executor,
	*engine.Channel,
	*engine.Channel,
	*engine.Channel,
	[]SourceAdapter,
	[]SinkAdapter,
	error,
) {
	// Phase 0: assume single route
	if len(cfg.Routes) == 0 {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("no routes configured")
	}

	// Get the first route (Phase 0 limitation)
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

	// Build steps from route config
	stepsInstances, stepNames, err := steps.BuildStepsFromSpec(routeSpec.Steps)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("failed to build steps: %w", err)
	}

	// Create executor with optional error channel
	var executor *engine.Executor
	if routeSpec.ErrorPath != nil {
		executor = engine.NewExecutorWithStepNames(routeName, inputCh, outputCh, errorCh, stepsInstances, stepNames)
	} else {
		executor = engine.NewExecutorWithStepNames(routeName, inputCh, outputCh, nil, stepsInstances, stepNames)
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
