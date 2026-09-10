package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

//go:embed studio_ui/*
var uiAssets embed.FS

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
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/routes", server.handleGetRoutes)
	http.HandleFunc("/api/routes/", server.handleSaveRoute)

	// Serve embedded UI assets
	http.Handle("/ui/", http.FileServer(http.FS(uiAssets)))

	// Start file watcher in background
	go server.watchFiles()

	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("Studio running at http://%s", addr)
	return http.ListenAndServe(addr, nil)
}

func (s *StudioServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	// Serve index.html
	data, _ := uiAssets.ReadFile("studio_ui/index.html")
	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
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
