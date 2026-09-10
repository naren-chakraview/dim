package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
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
	w.Header().Set("Content-Type", "application/json")
	// TODO: load routes from domains/*.yaml and return as JSON
	fmt.Fprintf(w, `{"routes":[]}`)
}

func (s *StudioServer) handleSaveRoute(w http.ResponseWriter, r *http.Request) {
	// TODO: parse JSON edit from request body, reconstruct YAML, validate, write to file
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok"}`)
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
