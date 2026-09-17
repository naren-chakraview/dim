package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLNodeWrapper preserves the original node tree along with parsed data
type YAMLNodeWrapper struct {
	RootNode *yaml.Node                 // Original parsed node tree (preserves order/comments)
	Data     map[string]interface{}     // Parsed map for editing
	FilePath string                     // Source file path for reference
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

// ReconstructYAMLFromNode rebuilds YAML from a node wrapper, preserving structure and comments
func ReconstructYAMLFromNode(wrapper *YAMLNodeWrapper) (string, error) {
	if wrapper == nil || wrapper.RootNode == nil {
		// Fallback to marshaling if no node available
		return marshallYAML(wrapper.Data)
	}

	// Update node values in-place from the edited data
	updateNodeFromData(wrapper.RootNode, wrapper.Data)

	// Encode the modified node tree back to YAML
	out, err := yaml.Marshal(wrapper.RootNode)
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// updateNodeFromData recursively updates node values to match edited data
// This preserves the original node structure, order, and comments
func updateNodeFromData(node *yaml.Node, data map[string]interface{}) {
	if node == nil {
		return
	}

	if node.Kind == yaml.MappingNode && len(node.Content)%2 == 0 {
		// Process key-value pairs
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]
			key := keyNode.Value

			if newValue, exists := data[key]; exists {
				// Key still exists in edited data - update its value
				updateNodeValue(valueNode, newValue)
			}
		}
	} else if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		// Handle document root
		updateNodeFromData(node.Content[0], data)
	}
}

// updateNodeValue recursively updates a node to match a new value
func updateNodeValue(node *yaml.Node, value interface{}) {
	if node == nil {
		return
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if node.Kind == yaml.MappingNode {
			// Recursively update nested mapping
			updateNodeFromData(node, v)
		} else {
			// Node type mismatch - would need to replace node
			// For now, this shouldn't happen in well-formed edits
		}

	case []interface{}:
		if node.Kind == yaml.SequenceNode {
			// Update sequence items
			updateNodeSequence(node, v)
		}

	default:
		// Scalar value - update the node's Value
		node.Value = fmt.Sprintf("%v", v)
		node.Kind = yaml.ScalarNode
	}
}

// updateNodeSequence updates sequence nodes to match array data
func updateNodeSequence(node *yaml.Node, items []interface{}) {
	if node.Kind != yaml.SequenceNode {
		return
	}

	// For simplicity, if lengths differ, we'd need to add/remove nodes
	// For now, just update existing nodes
	for i, item := range items {
		if i < len(node.Content) {
			updateNodeValue(node.Content[i], item)
		}
	}
}

// ReconstructYAML rebuilds YAML from data (fallback, no node preservation)
func ReconstructYAML(data map[string]interface{}, wrapper *YAMLNodeWrapper) (string, error) {
	// If we have a wrapper with a node, use node-based reconstruction
	if wrapper != nil && wrapper.RootNode != nil {
		wrapper.Data = data
		return ReconstructYAMLFromNode(wrapper)
	}

	// Fallback: marshal the data (loses order and comments)
	return marshallYAML(data)
}

// marshallYAML is a helper that marshals map to YAML
func marshallYAML(data map[string]interface{}) (string, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
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
