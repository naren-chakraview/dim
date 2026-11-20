// Package schemas exposes the route JSON schema compiled into the binary.
// route.schema.json is the single source of truth for the capability catalog
// served to agents (see agent.BuildCapabilities).
package schemas

import _ "embed"

//go:embed route.schema.json
var RouteSchema []byte
