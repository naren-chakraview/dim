package main

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// CommentMap stores comments indexed by key names for reconstruction
type CommentMap struct {
	HeadComments map[string]string // Comments above keys
	LineComments map[string]string // Inline comments after keys
	RawYAML      string             // Original YAML for fallback
}

// ParseYAMLWithComments parses YAML and tracks comments
func ParseYAMLWithComments(yamlStr string) (map[string]interface{}, map[string]string, error) {
	var data map[string]interface{}
	comments := make(map[string]string)

	// Parse with go-yaml's node representation to preserve comments
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(yamlStr), &node); err != nil {
		return nil, nil, err
	}

	// Walk nodes and extract comments using key names
	// The root node is a Document, the actual content is in node.Content[0]
	if len(node.Content) > 0 {
		extractCommentsFromNode(node.Content[0], comments)
	}

	// Also parse into map for easier access
	if err := yaml.Unmarshal([]byte(yamlStr), &data); err != nil {
		return nil, nil, err
	}

	return data, comments, nil
}

// extractCommentsFromNode recursively walks the YAML node tree and extracts comments
func extractCommentsFromNode(node *yaml.Node, comments map[string]string) {
	if node == nil {
		return
	}

	// Handle mapping nodes (key-value pairs)
	if node.Kind == yaml.MappingNode {
		// Process key-value pairs in the mapping
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			if keyNode.Value != "" {
				key := keyNode.Value

				// Store head comment (comment above the key)
				if keyNode.HeadComment != "" {
					comments[key+"_head"] = keyNode.HeadComment
				}

				// Store line comment (inline comment)
				if keyNode.LineComment != "" {
					comments[key+"_line"] = keyNode.LineComment
				}

				// Recursively process nested mappings
				if valueNode.Kind == yaml.MappingNode {
					extractCommentsFromNode(valueNode, comments)
				}
			}
		}
	}
}

// ReconstructYAML rebuilds YAML from data and comments
func ReconstructYAML(data map[string]interface{}, comments map[string]string) (string, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}

	result := string(out)

	// Reinsert head comments (comments above keys)
	for key, comment := range comments {
		if strings.HasSuffix(key, "_head") {
			actualKey := strings.TrimSuffix(key, "_head")
			// Find the key in the YAML and add comment above it
			pattern := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(actualKey) + `:)`)
			result = pattern.ReplaceAllString(result, comment+"\n$1")
		}
	}

	// Reinsert line comments (inline comments)
	for key, comment := range comments {
		if strings.HasSuffix(key, "_line") {
			actualKey := strings.TrimSuffix(key, "_line")
			// Find the key line and add inline comment
			pattern := regexp.MustCompile(`(?m)(` + regexp.QuoteMeta(actualKey) + `:\s*.*)$`)
			result = pattern.ReplaceAllString(result, "$1 "+comment)
		}
	}

	return result, nil
}

// RouteData represents a loaded route with its metadata and content
type RouteData struct {
	Name     string                 `json:"name"`
	Domain   string                 `json:"domain"`
	FilePath string                 `json:"filePath"`
	Data     map[string]interface{} `json:"data"`
	Comments map[string]string      `json:"comments"`
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

			data, comments, err := ParseYAMLWithComments(string(content))
			if err != nil {
				log.Printf("Warning: failed to parse %s: %v", filePath, err)
				continue
			}

			routes[routeName] = &RouteData{
				Name:     routeName,
				Domain:   domain,
				FilePath: filePath,
				Data:     data,
				Comments: comments,
			}
		}
	}

	return routes, nil
}
