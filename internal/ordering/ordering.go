package ordering

import (
	"github.com/naren-chakraview/dim/internal/config"
)

// OrderingMode represents the ordering requirement for a route
type OrderingMode int

const (
	// OrderingNone allows parallel processing of messages (default)
	OrderingNone OrderingMode = iota
	// OrderingRequired constrains processing to a single worker to preserve message order
	OrderingRequired
)

// String returns the string representation of the ordering mode
func (om OrderingMode) String() string {
	switch om {
	case OrderingNone:
		return "none"
	case OrderingRequired:
		return "required"
	default:
		return "unknown"
	}
}

// ParseOrdering converts a string to an OrderingMode
// "required" -> OrderingRequired
// "none" or empty string -> OrderingNone (default)
// invalid values default to OrderingNone
func ParseOrdering(s string) OrderingMode {
	switch s {
	case "required":
		return OrderingRequired
	case "none", "":
		return OrderingNone
	default:
		// Unknown values default to no ordering
		return OrderingNone
	}
}

// GetOrderingMode extracts the ordering mode from a RouteSpec
// Returns OrderingNone if the Ordering field is empty or invalid (backward compatible)
func GetOrderingMode(route *config.RouteSpec) OrderingMode {
	if route == nil {
		return OrderingNone
	}
	return ParseOrdering(route.Ordering)
}

// GetWorkerCount returns the number of workers for a route based on its ordering requirement
// If ordering is required, returns 1 (sequential processing)
// If ordering is none or unspecified, returns defaultWorkers (default: 4)
func GetWorkerCount(route *config.RouteSpec, defaultWorkers int) int {
	if route == nil {
		return defaultWorkers
	}

	// If ordering is required, use 1 worker (serial processing)
	if GetOrderingMode(route) == OrderingRequired {
		return 1
	}

	// Otherwise use the default worker count
	if defaultWorkers < 1 {
		return 4 // Fallback default if invalid
	}
	return defaultWorkers
}

// IsOrderingRequired checks if a route requires message ordering
func IsOrderingRequired(route *config.RouteSpec) bool {
	return GetOrderingMode(route) == OrderingRequired
}
