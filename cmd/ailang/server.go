package main

import (
	"context"
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
	"github.com/sunholo/ailang/internal/pubsub"
	"github.com/sunholo/ailang/internal/server"
	"github.com/sunholo/ailang/internal/storage"
	"github.com/sunholo/ailang/internal/telemetry"
)

func serverCommand(args []string) error {
	// Default values
	port := "1957"
	bindAddr := "localhost" // Safe default for local development
	dbPath := filepath.Join(os.Getenv("HOME"), ".ailang", "state", "collaboration.db")
	firebaseProject := "" // Firebase project ID for authentication

	// Check PORT env var (Cloud Run convention) — overridden by --port flag
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
		bindAddr = "0.0.0.0" // Cloud Run requires binding to all interfaces
	}

	// Parse flags (--port/--bind override env vars)
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port":
			if i+1 < len(args) {
				port = args[i+1]
				i++
			}
		case "--bind":
			if i+1 < len(args) {
				bindAddr = args[i+1]
				i++
			}
		case "--db":
			if i+1 < len(args) {
				dbPath = args[i+1]
				i++
			}
		case "--firebase-project":
			if i+1 < len(args) {
				firebaseProject = args[i+1]
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: ailang server [options]")
			fmt.Println("")
			fmt.Println("Start the AILANG Observatory and Collaboration Hub server")
			fmt.Println("")
			fmt.Println("Alias: 'ailang serve' also works (backward compatibility)")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --port PORT              HTTP server port (default: 1957, or PORT env var)")
			fmt.Println("  --bind ADDR              Bind address (default: localhost, 0.0.0.0 when PORT env set)")
			fmt.Println("  --db PATH                Database path (default: ~/.ailang/state/collaboration.db)")
			fmt.Println("  --firebase-project ID    Firebase project ID for authentication (optional)")
			fmt.Println("  --help, -h               Show this help message")
			fmt.Println("")
			fmt.Println("Environment variables:")
			fmt.Println("  PORT                     Server port (Cloud Run convention, overridden by --port)")
			fmt.Println("  AILANG_CONFIG            Config file path (overrides ~/.ailang/config.yaml)")
			fmt.Println("")
			fmt.Println("Authentication:")
			fmt.Println("  To enable Firebase authentication:")
			fmt.Println("  1. Run 'gcloud auth application-default login' for ADC")
			fmt.Println("  2. Use --firebase-project or set firebase.project_id in ~/.ailang/config.yaml")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  ailang server")
			fmt.Println("  ailang server --port 8080")
			fmt.Println("  ailang server --firebase-project ailang-dev")
			fmt.Println("")
			fmt.Println("Endpoints:")
			fmt.Println("  UI:          http://localhost:PORT/")
			fmt.Println("  WebSocket:   ws://localhost:PORT/ws")
			fmt.Println("  REST API:    http://localhost:PORT/api/")
			fmt.Println("  Health:      http://localhost:PORT/health")
			return nil
		}
	}

	// Check for Firebase project from config or environment if not specified via flag
	if firebaseProject == "" {
		// Try environment variable first
		firebaseProject = os.Getenv("AILANG_FIREBASE_PROJECT")
	}
	if firebaseProject == "" {
		// Try config file
		firebaseProject = getFirebaseProjectFromConfig()
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Check if server is already running on this port
	httpAddr := fmt.Sprintf("%s:%s", bindAddr, port)
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

	// Initialize OpenTelemetry (if configured via environment variables)
	ctx := context.Background()
	shutdownTelemetry, err := telemetry.Init(ctx, "ailang-server")
	if err != nil {
		log.Printf("Warning: Failed to initialize OpenTelemetry: %v", err)
	} else {
		defer shutdownTelemetry(ctx)
		if telemetry.IsDualExportEnabled() {
			log.Printf("Dual telemetry export enabled:")
			log.Printf("  → Google Cloud Trace (project: %s)", telemetry.GoogleCloudProject())
			log.Printf("  → OTLP endpoint: %s", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		} else if telemetry.IsGoogleCloudEnabled() {
			log.Printf("Google Cloud Trace enabled (project: %s)", telemetry.GoogleCloudProject())
		} else if telemetry.IsEnabled() {
			log.Printf("OpenTelemetry OTLP export enabled")
		}
	}

	// Build server options based on storage mode
	serverOpts := []server.ServerOption{
		server.WithVersion(Version),
	}

	// Add hook token auth if configured (for cloud deployments)
	if hookToken := os.Getenv("AILANG_HUB_TOKEN"); hookToken != "" {
		serverOpts = append(serverOpts, server.WithHookToken(hookToken))
		log.Printf("Hook token authentication enabled for /api/hooks/*")
	}

	// Add WebSocket token auth if configured (reuses COORDINATOR_API_KEY)
	if wsToken := os.Getenv("COORDINATOR_API_KEY"); wsToken != "" {
		serverOpts = append(serverOpts, server.WithWebSocketToken(wsToken))
		log.Printf("WebSocket token authentication enabled (external clients require ?token= parameter)")
	}

	// Add Firebase auth if configured
	if firebaseProject != "" {
		log.Printf("Firebase authentication enabled for project: %s", firebaseProject)
		serverOpts = append(serverOpts, server.WithFirebaseAuth(firebaseProject))
	}

	storageMode := storage.GetMode()
	var backends *storage.Backends

	if storageMode != storage.ModeLocal {
		// Cloud or hybrid mode: use storage.NewBackends for all stores
		backends, err = storage.NewBackends(ctx)
		if err != nil {
			return fmt.Errorf("failed to create %s backends: %w", storageMode, err)
		}
		defer backends.Close()

		serverOpts = append(serverOpts,
			server.WithMessagingStore(backends.Messaging),
			server.WithObservatoryBackend(backends.Observatory),
		)

		// Create Pub/Sub subscriber for real-time event streaming.
		// Dashboard pulls from ailang-events-dashboard and broadcasts via WebSocket.
		project := os.Getenv("AILANG_CLOUD_PROJECT")
		topicPrefix := os.Getenv("AILANG_TOPIC_PREFIX")
		if topicPrefix == "" {
			topicPrefix = pubsub.DefaultTopicPrefix
		}
		if project != "" {
			psClient, psErr := pubsub.NewClient(ctx, project, topicPrefix)
			if psErr != nil {
				log.Printf("Warning: Failed to create Pub/Sub client for event streaming: %v", psErr)
				log.Printf("Live task events will not be available via WebSocket")
			} else {
				psSub := pubsub.NewSubscriber(psClient)
				subName := pubsub.SubEventsDashboard
				serverOpts = append(serverOpts, server.WithPubSubEvents(psSub, subName))
				defer psSub.Stop()
				defer psClient.Close()
				log.Printf("Pub/Sub event streaming: subscription=%s-%s", topicPrefix, subName)
			}
		}

		log.Printf("Storage mode: %s", storageMode)
	} else {
		// Local mode: use SQLite paths (existing behavior)
		obsDbPath := filepath.Join(os.Getenv("HOME"), ".ailang", "state", "observatory.db")
		serverOpts = append(serverOpts, server.WithObservatoryDB(obsDbPath))
		log.Printf("Storage mode: local")
		log.Printf("Observatory DB: %s", obsDbPath)
	}

	// Create and start server
	srv, err := server.NewServer(dbPath, httpAddr, serverOpts...)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}
	defer srv.Close()

	// Connect coordinator store for task events
	if backends != nil && backends.Coordinator != nil {
		// Cloud mode: coordinator store already created
		srv.SetTaskEventStore(backends.Coordinator)
		srv.SetApprovalStore(backends.Coordinator)
		srv.SetCoordinatorStore(&coordStoreAdapter{store: backends.Coordinator})
		srv.SetCoordinatorStoreRaw(backends.Coordinator)
	} else {
		// Local mode: open SQLite coordinator store
		coordDbPath := filepath.Join(os.Getenv("HOME"), ".ailang", "state", "coordinator.db")
		coordStore, err := coordinator.NewSQLiteStore(coordDbPath)
		if err != nil {
			log.Printf("Warning: Could not connect to coordinator store: %v", err)
			log.Printf("Task event history will not be available")
		} else {
			srv.SetTaskEventStore(coordStore)
			srv.SetApprovalStore(coordStore)
			srv.SetCoordinatorStore(&coordStoreAdapter{store: coordStore})
			srv.SetCoordinatorStoreRaw(coordStore)
			defer coordStore.Close()
		}
		log.Printf("Coordinator DB: %s", coordDbPath)
	}

	log.Printf("AILANG Observatory & Collaboration Hub (v%s)", Version)
	log.Printf("Collaboration DB: %s", dbPath)
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

// coordStoreAdapter adapts coordinator.Store to server.CoordinatorStore
type coordStoreAdapter struct {
	store coordinator.Store
}

// GetCoordinatorStats implements server.CoordinatorStore
func (a *coordStoreAdapter) GetCoordinatorStats() (*server.CoordinatorStats, error) {
	ctx := context.Background()
	stats, err := a.store.GetTaskStats(ctx)
	if err != nil {
		return nil, err
	}

	return &server.CoordinatorStats{
		Running:      true, // If we got here, coordinator store is available
		TasksRun:     stats.CompletedTasks,
		PendingTasks: stats.PendingTasks,
		RunningTasks: stats.RunningTasks,
		FailedTasks:  stats.FailedTasks,
		TotalCost:    stats.TotalCost,
		TotalTokens:  stats.TotalTokens,
	}, nil
}

// GetCostByProvider implements server.CoordinatorStore
func (a *coordStoreAdapter) GetCostByProvider() (map[string]float64, error) {
	return a.store.GetCostByProvider()
}

// getFirebaseProjectFromConfig loads Firebase project ID from ~/.ailang/config.yaml
func getFirebaseProjectFromConfig() string {
	cfg := coordinator.LoadFirebaseConfig()
	if cfg != nil {
		return cfg.ProjectID
	}
	return ""
}
