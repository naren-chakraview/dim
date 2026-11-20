package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLNodeWrapper preserves the original node tree along with parsed data
type YAMLNodeWrapper struct {
	RootNode *yaml.Node             // Original parsed node tree (preserves order/comments)
	Data     map[string]interface{} // Parsed map for editing
	FilePath string                 // Source file path for reference
}

// ParseYAMLWithPreservation parses YAML while preserving the node tree
func ParseYAMLWithPreservation(yamlStr string) (*YAMLNodeWrapper, error) {
	// Parse to node tree (preserves order and comments)
	var rootNode yaml.Node
	if err := yaml.Unmarshal([]byte(yamlStr), &rootNode); err != nil {
		return nil, err
	}

	// Also parse to map for editing
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
		return nil, err
	}

	return &YAMLNodeWrapper{
		RootNode: &rootNode,
		Data:     data,
	}, nil
}

// errNoSourceNode is returned whenever a route would be re-serialized without
// its original node tree. Re-marshalling from a plain map drops comments and
// reorders keys, so the studio refuses instead of silently doing it.
var errNoSourceNode = errors.New("refusing to re-serialize route without its original node tree (would drop comments and reorder keys)")

// ReconstructYAMLFromNode writes an edited route back into its original node tree,
// so key order, comments and indentation survive. This is the single serializer
// used for every file the studio writes to disk.
//
// Structural edits are rejected with an error, never silently dropped: adding or
// removing a field at any depth, adding or removing list items, and changing a
// scalar/list/map into a different kind all return an error and nothing is written.
func ReconstructYAMLFromNode(wrapper *YAMLNodeWrapper) (string, error) {
	if wrapper == nil || wrapper.RootNode == nil {
		return "", errNoSourceNode
	}
	if err := applyDataToNode(wrapper.RootNode, wrapper.Data, "$"); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	// 2-space indentation matches the route files; yaml.v3's default is 4.
	enc.SetIndent(2)
	if err := enc.Encode(wrapper.RootNode); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// applyDataToNode walks the edited data alongside the original node tree and
// writes changed scalar values in place. It returns an error for any change it
// cannot represent in the node tree.
func applyDataToNode(node *yaml.Node, value interface{}, path string) error {
	if node == nil {
		return fmt.Errorf("%s: missing source node", path)
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return fmt.Errorf("%s: empty document", path)
		}
		return applyDataToNode(node.Content[0], value, path)
	}
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("%s: cannot edit a YAML alias in the studio; edit the file directly", path)
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if node.Kind != yaml.MappingNode {
			return kindChangeError(path, node, "map")
		}
		return applyMappingToNode(node, v, path)

	case []interface{}:
		if node.Kind != yaml.SequenceNode {
			return kindChangeError(path, node, "list")
		}
		if len(node.Content) != len(v) {
			return fmt.Errorf("%s: cannot add/remove list items in the studio (was %d, now %d); add/remove operations cannot be preserved in node-based round-trip, edit the file directly", path, len(node.Content), len(v))
		}
		for i := range v {
			if err := applyDataToNode(node.Content[i], v[i], fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil

	default:
		if node.Kind != yaml.ScalarNode {
			return kindChangeError(path, node, "scalar")
		}
		return setScalarNode(node, value, path)
	}
}

// applyMappingToNode checks that the field set matches exactly, then recurses.
func applyMappingToNode(node *yaml.Node, data map[string]interface{}, path string) error {
	if len(node.Content)%2 != 0 {
		return fmt.Errorf("%s: malformed mapping node", path)
	}
	inNode := make(map[string]bool, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		inNode[key] = true
		if _, ok := data[key]; !ok {
			return fmt.Errorf("cannot remove field %q at %s via studio editor - add/remove operations cannot be preserved in node-based round-trip", key, path)
		}
	}
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !inNode[key] {
			return fmt.Errorf("cannot add field %q at %s via studio editor - add/remove operations cannot be preserved in node-based round-trip", key, path)
		}
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if err := applyDataToNode(node.Content[i+1], data[key], path+"."+key); err != nil {
			return err
		}
	}
	return nil
}

func kindChangeError(path string, node *yaml.Node, want string) error {
	return fmt.Errorf("%s: cannot change this value to a %s in the studio (it is a different YAML kind); edit the file directly", path, want)
}

// setScalarNode writes value into a scalar node. Unchanged values are left
// alone so their original formatting (quoting, float precision) is kept.
func setScalarNode(node *yaml.Node, value interface{}, path string) error {
	var old interface{}
	if err := node.Decode(&old); err == nil && scalarEqual(old, value) {
		return nil
	}

	quoting := node.Style & (yaml.DoubleQuotedStyle | yaml.SingleQuotedStyle)
	switch v := value.(type) {
	case nil:
		node.Tag, node.Value, node.Style = "!!null", "null", 0
	case bool:
		node.Tag, node.Value, node.Style = "!!bool", strconv.FormatBool(v), 0
	case string:
		node.Tag, node.Value, node.Style = "!!str", v, quoting
	case float64:
		if v == math.Trunc(v) && math.Abs(v) < 1e15 {
			node.Tag, node.Value = "!!int", strconv.FormatInt(int64(v), 10)
		} else {
			node.Tag, node.Value = "!!float", strconv.FormatFloat(v, 'g', -1, 64)
		}
		node.Style = 0
	default:
		return fmt.Errorf("%s: unsupported value type %T", path, value)
	}
	return nil
}

// scalarEqual compares a decoded YAML scalar with an edited JSON scalar,
// treating all numeric kinds as float64.
func scalarEqual(a, b interface{}) bool {
	return reflect.DeepEqual(normalizeScalar(a), normalizeScalar(b))
}

func normalizeScalar(v interface{}) interface{} {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	}
	return v
}

// ReconstructYAML re-serializes data using the route's original node tree.
// A nil wrapper is an error: there is no fallback that drops comments and order.
func ReconstructYAML(data map[string]interface{}, wrapper *YAMLNodeWrapper) (string, error) {
	if wrapper == nil || wrapper.RootNode == nil {
		return "", errNoSourceNode
	}
	wrapper.Data = data
	return ReconstructYAMLFromNode(wrapper)
}

// marshalForValidation serializes a route with no source node. It is used ONLY
// to write a temp file for validation (in-editor /api/validate). Its output is
// never written to a route file on disk.
func marshalForValidation(data map[string]interface{}) (string, error) {
	var buf strings.Builder
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RouteData represents a loaded route with its metadata and content
// Note: Node is not JSON-serializable, so we only serialize the Data field
type RouteData struct {
	Name     string                 `json:"name"`
	Domain   string                 `json:"domain"`
	FilePath string                 `json:"filePath"`
	Data     map[string]interface{} `json:"data"`
	Node     *YAMLNodeWrapper       `json:"-"` // Preserved for round-trip fidelity
}

// MarshalJSON ensures RouteData serializes correctly (excluding Node)
func (r *RouteData) MarshalJSON() ([]byte, error) {
	type Alias RouteData
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}

// DiscoverRoutes finds all *.yaml files in domains/ (not test files)
func DiscoverRoutes(workDir string) (map[string]*RouteData, error) {
	routes := make(map[string]*RouteData)

	domainPath := filepath.Join(workDir, "domains")
	entries, err := os.ReadDir(domainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return routes, nil // domains dir doesn't exist yet
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		domain := entry.Name()
		domainDir := filepath.Join(domainPath, domain)
		files, _ := os.ReadDir(domainDir)

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".route_test.yaml") {
				continue
			}

			filePath := filepath.Join(domainDir, file.Name())
			routeName := strings.TrimSuffix(file.Name(), ".yaml")

			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("Warning: failed to read %s: %v", filePath, err)
				continue
			}

			wrapper, err := ParseYAMLWithPreservation(string(content))
			if err != nil {
				log.Printf("Warning: failed to parse %s: %v", filePath, err)
				continue
			}
			wrapper.FilePath = filePath

			routes[routeName] = &RouteData{
				Name:     routeName,
				Domain:   domain,
				FilePath: filePath,
				Data:     wrapper.Data,
				Node:     wrapper,
			}
		}
	}

	return routes, nil
}
