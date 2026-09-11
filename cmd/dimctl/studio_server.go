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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
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

	// Set up HTTP routes
	http.HandleFunc("/api/routes", server.handleGetRoutes)
	http.HandleFunc("/api/routes/", server.handleSaveRoute)
	http.HandleFunc("/api/route", server.handleGetRoute)
	http.HandleFunc("/api/save", server.handleSave)
	http.HandleFunc("/api/schema", server.handleGetSchema)
	http.HandleFunc("/api/validate", server.handleValidate)

	// Serve embedded web assets (React app)
	distFS, err := fs.Sub(webAssets, "web_dist")
	if err != nil {
		log.Printf("Warning: could not embed web assets: %v", err)
	} else {
		fsHandler := http.FileServer(http.FS(distFS))
		http.HandleFunc("/", server.handleSPA(fsHandler))
	}

	// Start file watcher in background
	go server.watchFiles()

	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("Studio running at http://%s", addr)
	return http.ListenAndServe(addr, nil)
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

func (s *StudioServer) handleSaveRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var editData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&editData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	routeName := r.URL.Query().Get("route")
	if routeName == "" {
		http.Error(w, "route parameter required", http.StatusBadRequest)
		return
	}

	// Reconstruct YAML
	yamlStr, err := ReconstructYAML(editData, make(map[string]string))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Write temp file and validate
	tempPath := filepath.Join(s.workDir, ".studio_temp.yaml")
	if err := ioutil.WriteFile(tempPath, []byte(yamlStr), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Run dimctl validate
	cmd := exec.Command("go", "run", "./cmd/dimctl", "validate", tempPath)
	cmd.Dir = s.workDir
	if err := cmd.Run(); err != nil {
		os.Remove(tempPath)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"validation_failed","error":"%s"}`, err.Error())
		return
	}

	// Validation passed; write to actual file
	// For now, we'll look for the route in the routes map and save to its file path
	routes, _ := DiscoverRoutes(s.workDir)
	if route, ok := routes[routeName]; ok {
		if err := ioutil.WriteFile(route.FilePath, []byte(yamlStr), 0644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"error","error":"route not found"}`)
		return
	}

	os.Remove(tempPath)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","message":"Route saved and validated"}`)
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
		Route    map[string]interface{} `json:"route"`
		FilePath string                 `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.FilePath == "" {
		http.Error(w, "filePath required", http.StatusBadRequest)
		return
	}

	// Convert route back to YAML and write to file
	yamlBytes, err := ReconstructYAML(req.Route, make(map[string]string))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to convert to YAML: %v", err),
		})
		return
	}

	// Write to file
	if err := ioutil.WriteFile(req.FilePath, []byte(yamlBytes), 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("Failed to write file: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route saved successfully",
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
		Route    map[string]interface{} `json:"route"`
		FilePath string                 `json:"filePath"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: Validate route using dimctl validate
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"errors": []string{},
		"warnings": []string{},
	})
}
