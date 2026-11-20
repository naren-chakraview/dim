package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/validation"
)

//go:embed web_dist/*
var webAssets embed.FS

type StudioServer struct {
	port    int
	workDir string
	watcher *fsnotify.Watcher
	routes  map[string]interface{}
}

func StartServer(port int, workDir string) error {
	absPath, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	server := &StudioServer{
		port:    port,
		workDir: absPath,
		routes:  make(map[string]interface{}),
	}

	// Set up file watcher for domains/*.yaml
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	server.watcher = watcher
	defer watcher.Close()

	// Watch domains directory
	domainsPath := filepath.Join(absPath, "domains")
	if err := watcher.Add(domainsPath); err != nil {
		log.Printf("Warning: could not watch domains directory: %v", err)
	}

	// Set up HTTP routes (the same mux the tests exercise)
	mux := server.newMux()

	// Start file watcher in background
	go server.watchFiles()

	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("Studio running at http://%s", addr)
	return http.ListenAndServe(addr, mux)
}

// newMux registers every studio route. StartServer serves this mux, so tests
// that drive it exercise exactly the routes the UI calls.
func (s *StudioServer) newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/routes", s.handleGetRoutes)
	mux.HandleFunc("/api/route", s.handleGetRoute)
	mux.HandleFunc("/api/save", s.handleSave)
	mux.HandleFunc("/api/schema", s.handleGetSchema)
	mux.HandleFunc("/api/validate", s.handleValidate)

	// Serve embedded web assets (React app)
	distFS, err := fs.Sub(webAssets, "web_dist")
	if err != nil {
		log.Printf("Warning: could not embed web assets: %v", err)
	} else {
		fsHandler := http.FileServer(http.FS(distFS))
		mux.HandleFunc("/", s.handleSPA(fsHandler))
	}
	return mux
}

// validateRouteData performs in-process validation of a route configuration.
// Returns (valid, errors, warnings, routeVersion, error).
// Uses the same validation as dimctl validate command.
//
// validationDir is the directory the route will live in. Relative imports
// (e.g. ../governance/fragments.yaml) resolve against the directory of the file
// being validated, so the temp file MUST live beside the real route file.
func (s *StudioServer) validateRouteData(route map[string]interface{}, wrapper *YAMLNodeWrapper, validationDir string) (bool, []string, []string, string, error) {
	// The temp file is only ever validated, never written to a route file. With a
	// source node we use the exact bytes the save would write; in-editor validation
	// (no file) has no node and uses the validation-only serializer.
	var yamlStr string
	var err error
	if wrapper != nil {
		yamlStr, err = ReconstructYAML(route, wrapper)
	} else {
		yamlStr, err = marshalForValidation(route)
	}
	if err != nil {
		return false, []string{fmt.Sprintf("Failed to reconstruct YAML: %v", err)}, []string{}, "", nil
	}

	// Write to a temporary file for validation
	if validationDir == "" {
		validationDir = s.workDir
	}
	tempFile := filepath.Join(validationDir, ".studio_validate_temp.yaml")
	if err := ioutil.WriteFile(tempFile, []byte(yamlStr), 0644); err != nil {
		return false, []string{fmt.Sprintf("Failed to write temp file: %v", err)}, []string{}, "", nil
	}
	defer os.Remove(tempFile)

	// Load and validate the route config using the same package as CLI
	cfg, err := config.LoadRouteConfig(tempFile)
	if err != nil {
		// Validation failed
		return false, []string{fmt.Sprintf("Validation error: %v", err)}, []string{}, "", nil
	}

	// Validate auth declarations (same as CLI)
	if err := config.ValidateAuthDeclarations(cfg, config.AuthValidationWarn); err != nil {
		return false, []string{fmt.Sprintf("Auth validation error: %v", err)}, []string{}, "", nil
	}

	// Perform static contract conformance checks (same as CLI)
	contractWarnings := validation.PerformStaticContractChecks(cfg)
	allWarnings := contractWarnings

	// Route version for lineage tracking
	routeVersion := ""
	if len(cfg.Routes) > 0 {
		for _, r := range cfg.Routes {
			if r.RouteVersion != "" {
				routeVersion = r.RouteVersion
				break
			}
		}
	}

	// All checks passed, return warnings (if any)
	return true, []string{}, allWarnings, routeVersion, nil
}

func (s *StudioServer) handleSPA(fileServer http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Don't handle API routes with the SPA handler
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// For non-API routes, serve the SPA
		// If the file doesn't exist, serve index.html (SPA routing)
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to serve the file
		fileServer.ServeHTTP(w, r)
	}
}

func (s *StudioServer) handleGetRoutes(w http.ResponseWriter, r *http.Request) {
	routes, err := DiscoverRoutes(s.workDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to JSON-serializable format
	response := map[string]interface{}{
		"routes": routes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *StudioServer) watchFiles() {
	for {
		select {
		case event := <-s.watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Printf("File changed: %s", event.Name)
				// Reload routes
			}
		case err := <-s.watcher.Errors:
			log.Printf("Watch error: %v", err)
		}
	}
}

func (s *StudioServer) handleGetRoute(w http.ResponseWriter, r *http.Request) {
	routeName := r.URL.Query().Get("route")
	if routeName == "" {
		http.Error(w, "route parameter required", http.StatusBadRequest)
		return
	}

	routes, err := DiscoverRoutes(s.workDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	route, ok := routes[routeName]
	if !ok {
		http.Error(w, "route not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

func (s *StudioServer) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Route     map[string]interface{} `json:"route"`
		FilePath  string                 `json:"filePath"`
		RouteName string                 `json:"routeName"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		http.Error(w, "filePath required", http.StatusBadRequest)
		return
	}

	// Load the original route to get its node wrapper (for preserving order/comments)
	routes, err := DiscoverRoutes(s.workDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"valid":   false,
			"errors":  []string{fmt.Sprintf("Failed to load routes: %v", err)},
		})
		return
	}

	// The file must be an existing discovered route. There is no fallback that
	// re-serializes without the original node tree: that would drop comments.
	var wrapper *YAMLNodeWrapper
	for _, route := range routes {
		if route.FilePath == req.FilePath {
			wrapper = route.Node
			break
		}
	}
	if wrapper == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"valid":   false,
			"errors":  []string{fmt.Sprintf("save refused: %q is not a discovered route file; nothing was written", req.FilePath)},
		})
		return
	}

	// Validate beside the real route file so relative imports resolve correctly.
	valid, errs, warns, routeVersion, err := s.validateRouteData(req.Route, wrapper, filepath.Dir(req.FilePath))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"valid":   false,
			"errors":  []string{fmt.Sprintf("Validation error: %v", err)},
		})
		return
	}

	// If validation failed, don't write to disk
	if !valid {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":  false,
			"valid":    false,
			"errors":   errs,
			"warnings": warns,
		})
		return
	}

	// Convert route back to YAML, preserving order and comments
	yamlBytes, err := ReconstructYAML(req.Route, wrapper)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"valid":   false,
			"errors":  []string{fmt.Sprintf("Failed to convert to YAML: %v", err)},
		})
		return
	}

	// Write to file (validation passed)
	if err := ioutil.WriteFile(req.FilePath, []byte(yamlBytes), 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"valid":   false,
			"errors":  []string{fmt.Sprintf("Failed to write file: %v", err)},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"valid":         true,
		"errors":        []string{},
		"warnings":      warns,
		"route_version": routeVersion,
	})
}

func (s *StudioServer) handleGetSchema(w http.ResponseWriter, r *http.Request) {
	// Load schema from schemas/route.schema.json
	schemaPath := filepath.Join(s.workDir, "schemas", "route.schema.json")
	schemaData, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		http.Error(w, "schema not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(schemaData)
}

func (s *StudioServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Route map[string]interface{} `json:"route"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Perform real in-process validation (no wrapper for in-editor validation)
	valid, errs, warns, routeVersion, err := s.validateRouteData(req.Route, nil, "")
	if err != nil {
		http.Error(w, fmt.Sprintf("Validation error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":         valid,
		"errors":        errs,
		"warnings":      warns,
		"route_version": routeVersion,
		"timestamp":     "", // Frontend will set timestamp
	})
}
