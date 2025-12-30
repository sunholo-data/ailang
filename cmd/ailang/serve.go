package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/server"
)

func serveCommand(args []string) error {
	// Default values
	port := "1957"
	dbPath := filepath.Join(os.Getenv("HOME"), ".ailang", "state", "collaboration.db")

	// Parse flags
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: ailang serve [options]")
			fmt.Println("")
			fmt.Println("Start the AILANG Collaboration Hub HTTP server")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --port PORT   HTTP server port (default: 1957)")
			fmt.Println("  --db PATH     Database path (default: ~/.ailang/state/collaboration.db)")
			fmt.Println("  --help, -h    Show this help message")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  ailang serve")
			fmt.Println("  ailang serve --port 8080")
			fmt.Println("  ailang serve --db /tmp/collab.db")
			fmt.Println("")
			fmt.Println("Endpoints:")
			fmt.Println("  UI:          http://localhost:PORT/")
			fmt.Println("  WebSocket:   ws://localhost:PORT/ws")
			fmt.Println("  REST API:    http://localhost:PORT/api/")
			fmt.Println("  Health:      http://localhost:PORT/health")
			return nil
		}
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Check if server is already running on this port
	httpAddr := fmt.Sprintf("localhost:%s", port)
	if isPortInUse(port) {
		// Port is in use - check if it's our server
		healthURL := fmt.Sprintf("http://%s/health", httpAddr)
		if checkServerHealth(healthURL) {
			uiURL := fmt.Sprintf("http://%s/", httpAddr)
			log.Printf("✓ AILANG server is already running on port %s", port)
			log.Printf("")
			log.Printf("Opening UI at: %s", uiURL)
			log.Printf("")

			// Open browser to UI
			if err := openBrowser(uiURL); err != nil {
				log.Printf("Could not open browser automatically: %v", err)
				log.Printf("Please open %s manually", uiURL)
			}

			return nil
		}

		// Port in use by something else
		return fmt.Errorf("port %s is already in use by another process. Use --port to specify a different port", port)
	}

	// Create and start server with version info
	srv, err := server.NewServer(dbPath, httpAddr, server.WithVersion(Version))
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	// Connect to coordinator store for task events (read-only access for historical replay)
	coordDbPath := filepath.Join(os.Getenv("HOME"), ".ailang", "state", "coordinator.db")
	coordStore, err := coordinator.NewSQLiteStore(coordDbPath)
	if err != nil {
		log.Printf("Warning: Could not connect to coordinator store: %v", err)
		log.Printf("Task event history will not be available")
	} else {
		srv.SetTaskEventStore(coordStore)
		srv.SetApprovalStore(coordStore)
		defer coordStore.Close()
	}

	log.Printf("AILANG Collaboration Hub Server (v%s)", Version)
	log.Printf("Database: %s", dbPath)
	log.Printf("Coordinator DB: %s", coordDbPath)
	log.Printf("")

	// Auto-open browser after a short delay (server needs to start first)
	uiURL := fmt.Sprintf("http://%s/", httpAddr)
	go func() {
		time.Sleep(500 * time.Millisecond)
		log.Printf("")
		log.Printf("Opening UI at: %s", uiURL)
		log.Printf("")
		if err := openBrowser(uiURL); err != nil {
			log.Printf("Could not open browser automatically: %v", err)
			log.Printf("Please open %s manually", uiURL)
		}
	}()

	return srv.Start()
}

// isPortInUse checks if a TCP port is already bound
func isPortInUse(port string) bool {
	addr := fmt.Sprintf(":%s", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return true // Port is in use
	}
	listener.Close()
	return false
}

// checkServerHealth verifies the server is running by checking the health endpoint
func checkServerHealth(healthURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// openBrowser opens the default browser to the specified URL
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
