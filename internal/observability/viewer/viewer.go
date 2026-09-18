package viewer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Message represents a tracked message in the ring buffer
type Message struct {
	CorrelationID string    `json:"correlationId"`
	Timestamp     time.Time `json:"timestamp"`
	LatencyMs     int64     `json:"latencyMs"`
	Status        string    `json:"status"` // "success" or "error"
}

// RingBuffer is a thread-safe circular buffer for storing recent messages
type RingBuffer struct {
	messages []Message
	pos      int   // current write position
	count    int   // number of messages written (for distinguishing first rotation)
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the given capacity
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		capacity = 200 // default
	}
	return &RingBuffer{
		messages: make([]Message, capacity),
	}
}

// Add adds a message to the ring buffer
func (rb *RingBuffer) Add(msg Message) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.messages[rb.pos] = msg
	rb.pos = (rb.pos + 1) % len(rb.messages)
	rb.count++
}

// GetAll returns all messages in the ring buffer in chronological order
func (rb *RingBuffer) GetAll() []Message {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]Message, 0, len(rb.messages))

	if rb.count < len(rb.messages) {
		// Buffer not full yet, return messages in order from 0 to pos-1
		for i := 0; i < rb.pos; i++ {
			result = append(result, rb.messages[i])
		}
	} else {
		// Buffer is full, return messages in circular order starting from pos
		for i := 0; i < len(rb.messages); i++ {
			idx := (rb.pos + i) % len(rb.messages)
			result = append(result, rb.messages[idx])
		}
	}

	return result
}

// RouteStats tracks statistics for a route
type RouteStats struct {
	InFlight     int32
	TotalProcessed int64
	TotalErrors  int64
	Latencies    []int64 // raw latency samples for percentile calculation
	mu           sync.RWMutex
}

// RouteViewer tracks state for a single route
type RouteViewer struct {
	routeName  string
	ringBuffer *RingBuffer
	stats      *RouteStats
}

// NewRouteViewer creates a new viewer for a route
func NewRouteViewer(routeName string, ringBufferSize int) *RouteViewer {
	return &RouteViewer{
		routeName:  routeName,
		ringBuffer: NewRingBuffer(ringBufferSize),
		stats: &RouteStats{
			Latencies: make([]int64, 0, 10000),
		},
	}
}

// RecordMessage records a message completion
func (rv *RouteViewer) RecordMessage(correlationID string, latency time.Duration, success bool) {
	status := "success"
	if !success {
		status = "error"
	}

	latencyMs := latency.Milliseconds()

	msg := Message{
		CorrelationID: correlationID,
		Timestamp:     time.Now().UTC(),
		LatencyMs:     latencyMs,
		Status:        status,
	}

	rv.ringBuffer.Add(msg)

	// Update stats
	rv.stats.mu.Lock()
	defer rv.stats.mu.Unlock()

	rv.stats.TotalProcessed++
	if !success {
		rv.stats.TotalErrors++
	}
	rv.stats.Latencies = append(rv.stats.Latencies, latencyMs)
}

// GetState returns the current state of the route
func (rv *RouteViewer) GetState() RouteState {
	rv.stats.mu.RLock()
	defer rv.stats.mu.RUnlock()

	state := RouteState{
		Name:           rv.routeName,
		InFlight:       rv.stats.InFlight,
		TotalProcessed: rv.stats.TotalProcessed,
		TotalErrors:    rv.stats.TotalErrors,
		RecentMessages: rv.ringBuffer.GetAll(),
	}

	// Calculate latency statistics
	if len(rv.stats.Latencies) > 0 {
		sortedLatencies := make([]int64, len(rv.stats.Latencies))
		copy(sortedLatencies, rv.stats.Latencies)
		sort.Slice(sortedLatencies, func(i, j int) bool { return sortedLatencies[i] < sortedLatencies[j] })

		// Average
		sum := int64(0)
		for _, l := range sortedLatencies {
			sum += l
		}
		state.AvgLatencyMs = float64(sum) / float64(len(sortedLatencies))

		// P50
		state.P50LatencyMs = float64(sortedLatencies[len(sortedLatencies)/2])

		// P90
		p90Idx := (len(sortedLatencies) * 90) / 100
		if p90Idx >= len(sortedLatencies) {
			p90Idx = len(sortedLatencies) - 1
		}
		state.P90LatencyMs = float64(sortedLatencies[p90Idx])

		// P99
		p99Idx := (len(sortedLatencies) * 99) / 100
		if p99Idx >= len(sortedLatencies) {
			p99Idx = len(sortedLatencies) - 1
		}
		state.P99LatencyMs = float64(sortedLatencies[p99Idx])
	}

	return state
}

// SetInFlight sets the in-flight message count
func (rv *RouteViewer) SetInFlight(count int32) {
	rv.stats.mu.Lock()
	defer rv.stats.mu.Unlock()
	rv.stats.InFlight = count
}

// RouteState represents the current state of a route
type RouteState struct {
	Name           string    `json:"name"`
	InFlight       int32     `json:"inFlight"`
	TotalProcessed int64     `json:"totalProcessed"`
	TotalErrors    int64     `json:"totalErrors"`
	AvgLatencyMs   float64   `json:"avgLatencyMs"`
	P50LatencyMs   float64   `json:"p50LatencyMs"`
	P90LatencyMs   float64   `json:"p90LatencyMs"`
	P99LatencyMs   float64   `json:"p99LatencyMs"`
	RecentMessages []Message `json:"recentMessages"`
}

// RoutePair represents a message flow from one route to another
type RoutePair struct {
	SourceRoute      string `json:"sourceRoute"`      // Route that produces messages
	SourceSink       string `json:"sourceSink"`       // Sink that receives messages from source route
	TargetRoute      string `json:"targetRoute"`      // Route that receives messages
	TargetSource     string `json:"targetSource"`     // Source that receives messages from source sink
	MessagesFlowed   int64  `json:"messagesFlowed"`   // Number of messages that flowed from source to target
	LastSeenAt       time.Time `json:"lastSeenAt"`    // Last time a message flowed on this pair
}

// ViewerServer manages multiple route viewers and serves the HTTP endpoint
type ViewerServer struct {
	viewers map[string]*RouteViewer
	pairs   map[string]*RoutePair // key: "source_route:sink:target_route:source"
	mu      sync.RWMutex
}

// NewViewerServer creates a new viewer server
func NewViewerServer() *ViewerServer {
	return &ViewerServer{
		viewers: make(map[string]*RouteViewer),
		pairs:   make(map[string]*RoutePair),
	}
}

// GetOrCreateViewer gets or creates a viewer for a route
func (vs *ViewerServer) GetOrCreateViewer(routeName string, ringBufferSize int) *RouteViewer {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if viewer, exists := vs.viewers[routeName]; exists {
		return viewer
	}

	viewer := NewRouteViewer(routeName, ringBufferSize)
	vs.viewers[routeName] = viewer
	return viewer
}

// GetAllViewers returns all route viewers
func (vs *ViewerServer) GetAllViewers() []*RouteViewer {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	result := make([]*RouteViewer, 0, len(vs.viewers))
	for _, viewer := range vs.viewers {
		result = append(result, viewer)
	}
	return result
}

// RecordRoutePair records a message flow from one route's sink to another route's source
func (vs *ViewerServer) RecordRoutePair(sourceRoute, sourceSink, targetRoute, targetSource string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s:%s", sourceRoute, sourceSink, targetRoute, targetSource)

	if pair, exists := vs.pairs[key]; exists {
		pair.MessagesFlowed++
		pair.LastSeenAt = time.Now().UTC()
	} else {
		vs.pairs[key] = &RoutePair{
			SourceRoute:    sourceRoute,
			SourceSink:     sourceSink,
			TargetRoute:    targetRoute,
			TargetSource:   targetSource,
			MessagesFlowed: 1,
			LastSeenAt:     time.Now().UTC(),
		}
	}
}

// GetAllPairs returns all route pairs (pairing relationships)
func (vs *ViewerServer) GetAllPairs() []*RoutePair {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	result := make([]*RoutePair, 0, len(vs.pairs))
	for _, pair := range vs.pairs {
		result = append(result, pair)
	}
	return result
}

// GetPairsForRoute returns all routes that pair with a given route (either as source or target)
func (vs *ViewerServer) GetPairsForRoute(routeName string) []*RoutePair {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	var result []*RoutePair
	for _, pair := range vs.pairs {
		if pair.SourceRoute == routeName || pair.TargetRoute == routeName {
			result = append(result, pair)
		}
	}
	return result
}

// HandleRoutes is the HTTP handler for /debug/routes
func (vs *ViewerServer) HandleRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	viewers := vs.GetAllViewers()
	states := make([]RouteState, 0, len(viewers))

	for _, viewer := range viewers {
		states = append(states, viewer.GetState())
	}

	// Sort by route name for consistent output
	sort.Slice(states, func(i, j int) bool { return states[i].Name < states[j].Name })

	// Include route pairings (Tier 1 Viewer Pairing)
	pairs := vs.GetAllPairs()

	response := map[string]interface{}{
		"routes":    states,
		"pairs":     pairs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(response)
}

// HandleRoutePairs is the HTTP handler for /debug/route-pairs
func (vs *ViewerServer) HandleRoutePairs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	routeName := r.URL.Query().Get("route")
	var pairs []*RoutePair

	if routeName != "" {
		// Get pairings for a specific route
		pairs = vs.GetPairsForRoute(routeName)
	} else {
		// Get all pairings
		pairs = vs.GetAllPairs()
	}

	response := map[string]interface{}{
		"pairs":     pairs,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	json.NewEncoder(w).Encode(response)
}

// StartServer starts the HTTP server for the viewer
func StartServer(addr string) (*ViewerServer, *http.Server, error) {
	viewer := NewViewerServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/routes", viewer.HandleRoutes)
	mux.HandleFunc("/debug/route-pairs", viewer.HandleRoutePairs)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"healthy"}`)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("viewer server error: %v\n", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return viewer, server, nil
}
