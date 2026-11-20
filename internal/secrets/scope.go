package secrets

import (
	"errors"
	"strings"
)

// ErrNoSecretContext is returned when a config value contains a ${SECRET:...}
// reference but the route has no domain or no resolver. Such references are refused,
// never read from the environment: an unscoped reference could reach another domain's
// secret.
var ErrNoSecretContext = errors.New("secret reference without a route domain or resolver: refused (no environment fallback)")

// ResolveScoped resolves ${SECRET:...} references in text for routeDomain. Text with
// no reference is returned unchanged. Resolution fails closed: see Resolver.ResolveInRoute.
func ResolveScoped(r *Resolver, routeDomain, text string) (string, error) {
	if !strings.Contains(text, "${SECRET:") {
		return text, nil
	}
	if r == nil || routeDomain == "" {
		return "", ErrNoSecretContext
	}
	return r.ResolveInRoute(routeDomain, text)
}
