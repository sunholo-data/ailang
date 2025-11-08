package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sunholo/ailang/internal/server"
)

func serveCommand(args []string) error {
	// Default values
	port := "8080"
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
			fmt.Println("  --port PORT   HTTP server port (default: 8080)")
			fmt.Println("  --db PATH     Database path (default: ~/.ailang/state/collaboration.db)")
			fmt.Println("  --help, -h    Show this help message")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  ailang serve")
			fmt.Println("  ailang serve --port 3001")
			fmt.Println("  ailang serve --db /tmp/collab.db")
			fmt.Println("")
			fmt.Println("Endpoints:")
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

	// Create and start server
	httpAddr := fmt.Sprintf("localhost:%s", port)
	srv, err := server.NewServer(dbPath, httpAddr)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	log.Printf("AILANG Collaboration Hub Server")
	log.Printf("Database: %s", dbPath)
	log.Printf("")
	log.Printf("Open the UI at: http://localhost:3000")
	log.Printf("(Make sure the UI dev server is running: cd ui && npm run dev)")
	log.Printf("")

	return srv.Start()
}
