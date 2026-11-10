package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/observatory"
	"github.com/sunholo-data/ailang/internal/platform/originpolicy"
	otelplatform "github.com/sunholo-data/ailang/internal/platform/otel"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/server"
	"github.com/sunholo-data/ailang/internal/statedir"
	"github.com/sunholo-data/ailang/internal/storage"
)

func serverCommand(args []string) error {
	// Default values
	port := "1957"
	bindAddr := config.DefaultBindHost()         // 127.0.0.1; 0.0.0.0 when PORT is set (Cloud Run)
	dbPath := messaging.GetDefaultDatabasePath() // "" when no state dir resolves; --db overrides
	firebaseProject := ""                        // Firebase project ID for authentication
	var corsOrigins []string                     // --cors-origin (repeatable): exact cross-origin allowlist

	// Check PORT env var (Cloud Run convention) — overridden by --port flag
	if envPort := config.Port(); envPort != "" {
		port = envPort
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
		case "--cors-origin":
			if i+1 < len(args) {
				corsOrigins = append(corsOrigins, args[i+1])
				i++
			}
		case "--help", "-h":
			fmt.Println("Usage: ailang server [options]")
			fmt.Println("")
			fmt.Println("Start the AILANG dashboard API server (headless: telemetry ingest,")
			fmt.Println("observatory and chains reads, approvals intake, /benchmarks/ data)")
			fmt.Println("")
			fmt.Println("Alias: 'ailang serve' also works (backward compatibility)")
			fmt.Println("")
			fmt.Println("Options:")
			fmt.Println("  --port PORT              HTTP server port (default: 1957, or PORT env var)")
			fmt.Println("  --bind ADDR              Bind address (default: 127.0.0.1, 0.0.0.0 when PORT env set)")
			fmt.Println("  --db PATH                Database path (default: ~/.ailang/state/collaboration.db)")
			fmt.Println("  --firebase-project ID    Firebase project ID for authentication (optional)")
			fmt.Println("  --cors-origin ORIGIN     Let this exact origin (scheme://host[:port]) call the API and open")
			fmt.Println("                           the WebSocket cross-origin; repeatable. Default: same-origin only")
			fmt.Println("  --help, -h               Show this help message")
			fmt.Println("")
			fmt.Println("Environment variables:")
			fmt.Println("  PORT                     Server port (Cloud Run convention, overridden by --port)")
			fmt.Println("  AILANG_CONFIG            Config file path (overrides ~/.ailang/config.yaml)")
			fmt.Println("")
			fmt.Println("Authentication:")
			fmt.Println("  Bound to loopback (the default), every route is open to local processes.")
			fmt.Println("  On any other bind (Cloud Run, --bind 0.0.0.0) every /api/ route except machine")
			fmt.Println("  ingest, and /ws, require Authorization: Bearer $COORDINATOR_API_KEY (or a Firebase")
			fmt.Println("  Viewer/Approver session); with neither configured they return 503. Clients send")
			fmt.Println("  the key from AILANG_DASHBOARD_TOKEN.")
			fmt.Println("  To enable Firebase authentication:")
			fmt.Println("  1. Run 'gcloud auth application-default login' for ADC")
			fmt.Println("  2. Use --firebase-project or set firebase.project_id in ~/.ailang/config.yaml")
			fmt.Println("")
			fmt.Println("Examples:")
			fmt.Println("  ailang server")
			fmt.Println("  ailang server --port 8080")
			fmt.Println("  ailang server --firebase-project ailang-dev")
			fmt.Println("")
			fmt.Println("Endpoints (headless API, no UI):")
			fmt.Println("  WebSocket:   ws://localhost:PORT/ws")
			fmt.Println("  REST API:    http://localhost:PORT/api/")
			fmt.Println("  Health:      http://localhost:PORT/health")
			return nil
		}
	}

	if err := originpolicy.Validate(false, corsOrigins); err != nil {
		return err
	}

	// Check for Firebase project from config or environment if not specified via flag
	if firebaseProject == "" {
		// Try environment variable first
		firebaseProject = config.FirebaseProject()
	}
	if firebaseProject == "" {
		// Try config file
		firebaseProject = getFirebaseProjectFromConfig()
	}

	// Ensure database directory exists
	if dbPath == "" {
		return fmt.Errorf("no collaboration database path: set %s (or HOME), or pass --db", statedir.EnvVar)
	}
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("failed to create database directory: %w", err)
	}

	// Check if server is already running on this port
	httpAddr := net.JoinHostPort(bindAddr, port)
	if isPortInUse(httpAddr) {
		// Port is in use - check if it's our server
		healthURL := fmt.Sprintf("http://%s/health", httpAddr)
		if checkServerHealth(healthURL) {
			log.Printf("✓ AILANG server is already running on port %s (%s)", port, healthURL)
			return nil
		}

		// Port in use by something else
		return fmt.Errorf("port %s is already in use by another process. Use --port to specify a different port", port)
	}

	// Initialize OpenTelemetry (if configured via environment variables)
	ctx := context.Background()
	shutdownTelemetry, telemetryStatus, err := otelplatform.InitWithStatus(ctx, "ailang-server")
	if err != nil {
		log.Printf("Warning: Failed to initialize OpenTelemetry: %v", err)
	} else {
		defer shutdownTelemetry(ctx)
		log.Printf("Telemetry: %s", telemetryStatus)
	}

	// Build server options based on storage mode
	serverOpts := []server.ServerOption{
		server.WithVersion(Version),
		server.WithCORSOrigins(corsOrigins),
	}

	// Ingest auth (internal/server/ingest_auth.go): the shared secrets for the
	// hook/exec/telemetry routes and OTLP /v1/*, and the warn|enforce|off
	// rollout mode. Checked only on a non-loopback bind.
	serverOpts = append(serverOpts,
		server.WithHookToken(config.HubToken()),
		server.WithOTLPIngestToken(config.OTLPIngestToken()),
		server.WithIngestAuthMode(config.IngestAuthMode()),
	)

	// The dashboard's machine credential (COORDINATOR_API_KEY): on a
	// non-loopback bind, /api/ reads and the WebSocket require it (or a
	// Firebase Viewer/Approver session) — internal/server/read_auth.go.
	if apiKey := config.CoordinatorAPIKey(); apiKey != "" {
		serverOpts = append(serverOpts, server.WithAPIKey(apiKey))
		log.Printf("API key authentication enabled (Authorization: Bearer for /api/, ?token= for /ws)")
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
		project, projErr := config.CloudProject(ctx)
		topicPrefix := pubsub.TopicPrefixFromEnv()
		if projErr != nil {
			log.Printf("Warning: no Pub/Sub event streaming: %v", projErr)
		} else {
			psClient, psErr := pubsub.NewClient(ctx, project, topicPrefix)
			if psErr != nil {
				log.Printf("Warning: Failed to create Pub/Sub client for event streaming: %v", psErr)
				log.Printf("Live task events will not be available via WebSocket")
			} else {
				psSub := pubsub.NewSubscriber(psClient)
				subName := pubsub.SubEventsDashboard
				serverOpts = append(serverOpts, server.WithPubSubEvents(psSub, subName))
				defer psSub.Stop()

				// M-SECRET-REMOTE-APPROVAL-WIRING: a publisher on the same client
				// drives the secret-approval ntfy push (kind=approval → coordinator
				// bridge → ntfy → phone).
				approvalPub := pubsub.NewPublisher(psClient)
				serverOpts = append(serverOpts, server.WithApprovalPublisher(approvalPub))
				defer approvalPub.Stop()

				defer psClient.Close()
				log.Printf("Pub/Sub event streaming: subscription=%s-%s", topicPrefix, subName)
			}
		}

		log.Printf("Storage mode: %s", storageMode)
	} else {
		// Local mode: use SQLite paths (existing behavior)
		obsDbPath := observatory.DefaultDatabasePath()
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
		srv.SetCoordinatorStoreRaw(backends.Coordinator)
	} else {
		// Local mode: open SQLite coordinator store under the state dir
		coordDbPath, _ := statedir.Path("coordinator.db") // "" → NewSQLiteStore reports the resolution error
		coordStore, err := coordinator.NewSQLiteStore(coordDbPath)
		if err != nil {
			log.Printf("Warning: Could not connect to coordinator store: %v", err)
			log.Printf("Task event history will not be available")
		} else {
			srv.SetTaskEventStore(coordStore)
			srv.SetApprovalStore(coordStore)
			srv.SetCoordinatorStoreRaw(coordStore)
			defer coordStore.Close()
		}
		log.Printf("Coordinator DB: %s", coordDbPath)
	}

	// M-SECRET-REMOTE-APPROVAL-WIRING: enable signed single-use token auth on the
	// secret approve/reject endpoints when a signing key is configured, so the
	// iPhone ntfy action buttons can POST without Google IAM.
	if keyStr := config.ApprovalSigningKey(); keyStr != "" {
		if signer, err := approvaltoken.NewSigner([]byte(keyStr)); err == nil {
			srv.SetSecretApprovalAuth(signer)
			log.Printf("Secret approval token auth enabled")
		} else {
			log.Printf("WARNING: invalid AILANG_APPROVAL_SIGNING_KEY: %v", err)
		}
	}

	log.Printf("AILANG Observatory & Collaboration Hub (v%s)", Version)
	log.Printf("Collaboration DB: %s", dbPath)
	log.Printf("")

	return srv.Start()
}

// isPortInUse checks whether addr (the exact host:port the server will bind)
// is already taken. Probing the wildcard ":port" instead misses a holder on
// 127.0.0.1 on macOS, where both binds succeed and requests split between them.
func isPortInUse(addr string) bool {
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

func getFirebaseProjectFromConfig() string {
	cfg := coordinator.LoadFirebaseConfig()
	if cfg != nil {
		return cfg.ProjectID
	}
	return ""
}
