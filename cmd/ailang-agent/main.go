package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const version = "0.4.5-dev"

func main() {
	// CLI flags
	instanceID := flag.String("instance-id", "", "Agent instance ID (required)")
	dbPath := flag.String("db", ".ailang/state/collaboration.db", "Path to collaboration database")
	pollInterval := flag.Int("poll-interval", 2, "Polling interval in seconds")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ailang-agent version %s\n", version)
		os.Exit(0)
	}

	if *instanceID == "" {
		log.Fatal("Error: --instance-id is required\n\nUsage: ailang-agent --instance-id <id>\n")
	}

	// Create agent
	agent, err := NewAgent(*instanceID, *dbPath, *pollInterval)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	defer agent.Close()

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start agent in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- agent.Run(ctx)
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
		cancel()
		// Wait for agent to finish
		if err := <-errChan; err != nil && err != context.Canceled {
			log.Printf("Agent error during shutdown: %v", err)
		}
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			log.Fatalf("Agent error: %v", err)
		}
	}

	log.Println("Agent shutdown complete")
}
