package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/naren-chakraview/dim/schemas"
)

// CapabilitySchemaVersion is the version of the served capability catalog.
const CapabilitySchemaVersion = "1.0.0"

// eipBindings maps each served EIP to the route-schema definition that
// implements it. BuildCapabilities fails if a bound definition is missing, so
// an EIP cannot be served without a real backing definition.
var eipBindings = []struct {
	Name        string
	Description string
	Definition  string
}{
	{"claim-check", "Externalize large message payloads to reduce memory usage", "claim_check_step"},
	{"content-based-router", "Route messages based on content evaluation", "route_step"},
	{"dead-letter-queue", "Capture and route failed messages", "error_path"},
	{"wiretap", "Clone messages for observation without affecting flow", "wiretap_step"},
}

// ServedCapabilities returns the capability catalog from the embedded route schema.
// This is the function GetCapabilities and `dimctl capabilities` both serve.
func ServedCapabilities(filterType string) ([]Capability, error) {
	return BuildCapabilities(schemas.RouteSchema, filterType)
}

// BuildCapabilities derives the capability catalog from a route JSON schema:
//   - adapters come from the source/sink "type" enums (one entry per type and direction)
//   - steps come from the "step" oneOf references, named "<key>-step"
//   - EIPs come from eipBindings, each bound to an existing schema definition
//
// It returns an error rather than a partial catalog if the schema is malformed.
func BuildCapabilities(schemaJSON []byte, filterType string) ([]Capability, error) {
	switch filterType {
	case "", "adapter", "step", "eip":
	default:
		return nil, fmt.Errorf("unknown capability filter %q (want adapter, step, or eip)", filterType)
	}

	var schema struct {
		Definitions map[string]map[string]interface{} `json:"definitions"`
	}
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		return nil, fmt.Errorf("parse route schema: %w", err)
	}
	defs := schema.Definitions

	caps := []Capability{}

	if filterType == "" || filterType == "adapter" {
		adapters, err := schemaAdapterTypes(defs)
		if err != nil {
			return nil, err
		}
		for _, adapter := range adapters {
			for _, dir := range []string{"source", "sink"} {
				def, ok := defs[dir]
				if !ok {
					return nil, fmt.Errorf("route schema has no %q definition", dir)
				}
				cfg, err := cloneSchemaMap(def)
				if err != nil {
					return nil, err
				}
				props, _ := cfg["properties"].(map[string]interface{})
				if props == nil {
					return nil, fmt.Errorf("%s definition has no properties", dir)
				}
				props["type"] = map[string]interface{}{
					"type": "string",
					"enum": []interface{}{adapter},
				}
				caps = append(caps, Capability{
					Name:         adapter + "-" + dir,
					Type:         "adapter",
					Description:  fmt.Sprintf("%s %s adapter", adapter, dir),
					ConfigSchema: cfg,
				})
			}
		}
	}

	if filterType == "" || filterType == "step" {
		oneOf, _ := defs["step"]["oneOf"].([]interface{})
		if len(oneOf) == 0 {
			return nil, fmt.Errorf("route schema definition \"step\" has no oneOf references")
		}
		for _, item := range oneOf {
			ref, _ := item.(map[string]interface{})["$ref"].(string)
			defName := strings.TrimPrefix(ref, "#/definitions/")
			if !strings.HasSuffix(defName, "_step") || defName == ref {
				return nil, fmt.Errorf("step reference %q is not a #/definitions/<name>_step", ref)
			}
			def, ok := defs[defName]
			if !ok {
				return nil, fmt.Errorf("step reference %q has no definition", ref)
			}
			cfg, err := cloneSchemaMap(def)
			if err != nil {
				return nil, err
			}
			caps = append(caps, Capability{
				// "-step" keeps step names distinct from EIP names (e.g. wiretap).
				Name:         strings.TrimSuffix(defName, "_step") + "-step",
				Type:         "step",
				Description:  stringField(def, "description"),
				ConfigSchema: cfg,
			})
		}
	}

	if filterType == "" || filterType == "eip" {
		for _, b := range eipBindings {
			def, ok := defs[b.Definition]
			if !ok {
				return nil, fmt.Errorf("EIP %q is bound to missing definition %q", b.Name, b.Definition)
			}
			cfg, err := cloneSchemaMap(def)
			if err != nil {
				return nil, err
			}
			caps = append(caps, Capability{
				Name:         b.Name,
				Type:         "eip",
				Description:  b.Description,
				ConfigSchema: cfg,
			})
		}
	}

	return caps, nil
}

// SchemaAdapterTypes returns the adapter type names the route schema accepts
// for sources and sinks. Callers that validate adapter names should use this
// instead of a local list.
func SchemaAdapterTypes() ([]string, error) {
	var schema struct {
		Definitions map[string]map[string]interface{} `json:"definitions"`
	}
	if err := json.Unmarshal(schemas.RouteSchema, &schema); err != nil {
		return nil, fmt.Errorf("parse route schema: %w", err)
	}
	return schemaAdapterTypes(schema.Definitions)
}

// schemaAdapterTypes reads the source and sink "type" enums and requires them to match.
func schemaAdapterTypes(defs map[string]map[string]interface{}) ([]string, error) {
	src, err := enumOf(defs, "source")
	if err != nil {
		return nil, err
	}
	snk, err := enumOf(defs, "sink")
	if err != nil {
		return nil, err
	}
	if strings.Join(src, ",") != strings.Join(snk, ",") {
		return nil, fmt.Errorf("source adapter enum %v does not match sink adapter enum %v", src, snk)
	}
	return src, nil
}

func enumOf(defs map[string]map[string]interface{}, dir string) ([]string, error) {
	def, ok := defs[dir]
	if !ok {
		return nil, fmt.Errorf("route schema has no %q definition", dir)
	}
	props, _ := def["properties"].(map[string]interface{})
	typeProp, _ := props["type"].(map[string]interface{})
	raw, _ := typeProp["enum"].([]interface{})
	if len(raw) == 0 {
		return nil, fmt.Errorf("%s.type has no enum", dir)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s.type enum has a non-string entry %v", dir, v)
		}
		out = append(out, s)
	}
	return out, nil
}

// cloneSchemaMap deep-copies a schema fragment so callers can narrow it safely.
func cloneSchemaMap(in map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stringField(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}
