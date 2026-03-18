// Package apiserver provides an HTTP server that auto-generates REST endpoints
// from AILANG module exports. It wraps the embed.Engine to expose AILANG functions
// as JSON-in/JSON-out HTTP endpoints.
//
// Usage:
//
//	srv := apiserver.New(basePath, "8080")
//	srv.LoadModules([]string{"ecommerce/api/handlers.ail"})
//	srv.Start() // blocks
package apiserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sunholo/ailang/internal/embed"
	"github.com/sunholo/ailang/internal/iface"
	"github.com/sunholo/ailang/internal/pipeline"
)

// DefaultMaxUploadSize is the default maximum upload size (50MB).
const DefaultMaxUploadSize = 50 << 20

// Server is the API server that exposes AILANG functions as REST endpoints.
type Server struct {
	engine   *embed.Engine
	modules  map[string]*ModuleInfo // module path → info
	mu       sync.RWMutex
	port     string
	basePath string
	cors     bool

	// Frontend proxy
	frontendPath string // path to React project (optional)
	staticPath   string // path to built static files (optional)
	viteCmd      *exec.Cmd

	// Hot reload
	watch      bool // whether file watching is enabled
	watcher    *fsnotify.Watcher
	watchPaths []string // absolute paths of loaded .ail files (for reload mapping)

	// Protocol support
	mcpEnabled    bool   // serve MCP at /mcp/
	mcpOnly       bool   // stdio-only MCP mode (no HTTP)
	maxUploadSize int64  // maximum upload size in bytes (0 = use DefaultMaxUploadSize)
	apiKeyHeader  string // HTTP header name for API key auth
	apiKeyEnv     string // env var containing expected API key
}

// ModuleInfo holds metadata about a loaded AILANG module.
type ModuleInfo struct {
	Path    string       `json:"path"`
	Exports []ExportInfo `json:"exports"`
}

// ExportInfo describes a single exported function from an AILANG module.
type ExportInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`                   // human-readable type signature
	Pure        bool   `json:"pure"`                   // whether the function is pure
	Arity       int    `json:"arity"`                  // number of parameters (-1 if not a function)
	RouteMethod string `json:"route_method,omitempty"` // custom HTTP method from @route annotation
	RoutePath   string `json:"route_path,omitempty"`   // custom URL path from @route annotation
}

// Config holds configuration for the API server.
type Config struct {
	Port          string
	CORS          bool
	FrontendPath  string      // optional: React project path for Vite proxy
	StaticPath    string      // optional: built frontend files
	Watch         bool        // enable file watching for hot reload
	EffCtx        interface{} // optional: pre-configured effect context (*effects.EffContext)
	MCP           bool        // enable MCP endpoint at /mcp/
	MCPOnly       bool        // run as MCP stdio server only (no HTTP)
	MaxUploadSize int64       // max upload size in bytes (0 = DefaultMaxUploadSize)
	APIKeyHeader  string      // HTTP header for API key auth (empty = no auth)
	APIKeyEnv     string      // env var containing expected API key
}

// New creates a new API server.
func New(basePath string, cfg Config) *Server {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	eng := embed.New(basePath)
	if cfg.EffCtx != nil {
		eng.SetEffContext(cfg.EffCtx)
	}
	maxUpload := cfg.MaxUploadSize
	if maxUpload == 0 {
		maxUpload = DefaultMaxUploadSize
	}
	return &Server{
		engine:        eng,
		modules:       make(map[string]*ModuleInfo),
		port:          cfg.Port,
		basePath:      basePath,
		cors:          cfg.CORS,
		frontendPath:  cfg.FrontendPath,
		staticPath:    cfg.StaticPath,
		watch:         cfg.Watch,
		mcpEnabled:    cfg.MCP,
		mcpOnly:       cfg.MCPOnly,
		maxUploadSize: maxUpload,
		apiKeyHeader:  cfg.APIKeyHeader,
		apiKeyEnv:     cfg.APIKeyEnv,
	}
}

// LoadModules compiles and loads AILANG modules from the given paths.
// Each path can be a .ail file or a directory (scanned recursively for .ail files).
func (s *Server) LoadModules(paths []string) error {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", p, err)
		}

		if info.IsDir() {
			if err := s.loadDirectory(p); err != nil {
				return err
			}
		} else {
			if err := s.loadFile(p); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) loadDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".ail") {
			return s.loadFile(path)
		}
		return nil
	})
}

func (s *Server) loadFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", path, err)
	}

	// Compile through pipeline to get interface info
	cfg := pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: true,
	}
	src := pipeline.Source{
		Filename: absPath,
	}

	result, err := pipeline.RunWithContext(context.Background(), cfg, src)
	if err != nil {
		return fmt.Errorf("compilation error for %s: %w", path, err)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("compilation errors for %s: %v", path, result.Errors)
	}

	// Extract module interface
	if result.Interface == nil {
		return fmt.Errorf("no module interface for %s", path)
	}

	modInfo := extractModuleInfo(result.Interface)

	// Extract route annotations from AST
	if result.Artifacts.AST != nil {
		extractRouteAnnotations(modInfo, result.Artifacts.AST)
	}

	// Derive module path from file path relative to basePath.
	// This is more reliable than using the pipeline's canonical path which
	// may include filesystem prefixes (e.g., "var/folders/.../test/api/greet"
	// instead of "test/api/greet").
	modulePath := strings.TrimSuffix(filepath.Base(absPath), ".ail")

	relPath, relErr := filepath.Rel(s.basePath, absPath)
	if relErr == nil && !strings.HasPrefix(relPath, "..") {
		// File is under basePath - use relative path as module path
		modulePath = strings.TrimSuffix(relPath, ".ail")
		// Convert OS path separators to forward slashes for URL routing
		modulePath = filepath.ToSlash(modulePath)
	}

	modInfo.Path = modulePath

	s.mu.Lock()
	s.modules[modulePath] = modInfo
	s.mu.Unlock()

	// Don't eagerly load via engine.Load() - the engine will lazily compile
	// and load the module on the first Call(). This avoids path resolution
	// issues where pipeline canonical paths differ from engine basePath resolution.

	// Track absolute path for file watching
	s.watchPaths = append(s.watchPaths, absPath)

	log.Printf("  Loaded module: %s (%d exports)", modulePath, len(modInfo.Exports))
	return nil
}

// extractModuleInfo builds ModuleInfo from a module interface.
func extractModuleInfo(ifc *iface.Iface) *ModuleInfo {
	info := &ModuleInfo{
		Path:    ifc.Module,
		Exports: make([]ExportInfo, 0, len(ifc.Exports)),
	}

	for name, item := range ifc.Exports {
		export := ExportInfo{
			Name:  name,
			Pure:  item.Purity,
			Arity: -1, // default: not a function
		}

		if item.Type != nil {
			export.Type = item.Type.String()
			// Count arity by counting function arrows in the type
			export.Arity = countFunctionArity(item.Type.String())
		}

		info.Exports = append(info.Exports, export)
	}

	return info
}

// countFunctionArity estimates the number of parameters from a type string.
// e.g., "string -> string -> int" has arity 2.
func countFunctionArity(typeStr string) int {
	// Count " -> " occurrences as a simple heuristic
	count := strings.Count(typeStr, " -> ")
	if count == 0 {
		return -1 // not a function
	}
	return count
}

// StartMCP runs the server as an MCP stdio server (no HTTP). Blocks until done.
func (s *Server) StartMCP() error {
	mcpSrv := NewMCPServer(s)
	return mcpSrv.RunStdio(context.Background())
}

// Start starts the HTTP server. Blocks until shutdown signal.
func (s *Server) Start() error {
	// If MCPOnly, run stdio MCP server instead of HTTP.
	if s.mcpOnly {
		return s.StartMCP()
	}

	mux := s.buildRoutes()

	httpAddr := fmt.Sprintf(":%s", s.port)
	srv := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	// Start Vite dev server if frontend path specified
	if s.frontendPath != "" {
		if err := s.startViteProxy(); err != nil {
			log.Printf("Warning: failed to start Vite dev server: %v", err)
		}
	}

	// Start file watcher if enabled
	if s.watch {
		if err := s.startWatcher(); err != nil {
			log.Printf("Warning: failed to start file watcher: %v", err)
		} else {
			log.Println("  Hot reload enabled (watching for .ail file changes)")
		}
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-done
		log.Println("Shutting down...")
		if s.watcher != nil {
			_ = s.watcher.Close()
		}
		if s.viteCmd != nil && s.viteCmd.Process != nil {
			_ = s.viteCmd.Process.Kill()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	s.printStartupBanner()

	return srv.ListenAndServe()
}

func (s *Server) buildRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Meta/introspection endpoints
	mux.HandleFunc("/api/_meta/modules", s.corsWrap(s.handleListModules))
	mux.HandleFunc("/api/_meta/modules/", s.corsWrap(s.handleModuleDetail))
	mux.HandleFunc("/api/_health", s.corsWrap(s.handleHealth))

	// OpenAPI spec + interactive docs
	mux.HandleFunc("/api/_meta/openapi.json", s.corsWrap(s.handleOpenAPISpec))
	mux.HandleFunc("/api/_meta/docs", s.corsWrap(s.handleSwaggerUI))
	mux.HandleFunc("/api/_meta/redoc", s.corsWrap(s.handleReDoc))

	// A2A Agent Card
	mux.HandleFunc("/.well-known/agent.json", s.corsWrap(s.handleA2AAgentCard))
	mux.HandleFunc("/a2a/", s.corsWrap(s.handleA2ATask))

	// MCP endpoint (streamable HTTP transport)
	if s.mcpEnabled {
		mcpSrv := NewMCPServer(s)
		mux.Handle("/mcp/", http.StripPrefix("/mcp", mcpSrv.HTTPHandler()))
	}

	// Custom routes from @route annotations (registered before catch-all)
	// Auth middleware wraps custom routes and the catch-all
	s.registerCustomRoutes(mux)

	// Function call endpoints - catch-all under /api/
	mux.HandleFunc("/api/", s.corsWrap(s.authMiddleware(s.handleFunctionCall)))

	// Static files or frontend proxy
	if s.staticPath != "" {
		mux.Handle("/", http.FileServer(http.Dir(s.staticPath)))
	} else if s.frontendPath != "" {
		// Proxy to Vite dev server
		viteURL, _ := url.Parse("http://localhost:5173")
		proxy := httputil.NewSingleHostReverseProxy(viteURL)
		mux.Handle("/", proxy)
	}

	return mux
}

func (s *Server) corsWrap(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cors {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		handler(w, r)
	}
}

func (s *Server) startViteProxy() error {
	// Check if Vite config exists
	viteCfg := filepath.Join(s.frontendPath, "vite.config.ts")
	if _, err := os.Stat(viteCfg); os.IsNotExist(err) {
		viteCfg = filepath.Join(s.frontendPath, "vite.config.js")
		if _, err := os.Stat(viteCfg); os.IsNotExist(err) {
			return fmt.Errorf("no vite.config.ts or vite.config.js found in %s", s.frontendPath)
		}
	}

	s.viteCmd = exec.Command("npm", "run", "dev")
	s.viteCmd.Dir = s.frontendPath
	s.viteCmd.Stdout = os.Stdout
	s.viteCmd.Stderr = os.Stderr

	if err := s.viteCmd.Start(); err != nil {
		return fmt.Errorf("failed to start Vite: %w", err)
	}

	log.Printf("  Vite dev server starting in %s", s.frontendPath)
	return nil
}

func (s *Server) printStartupBanner() {
	log.Println()
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("  AILANG API Server")
	log.Printf("  http://localhost:%s", s.port)
	log.Println()
	log.Println("  Endpoints:")

	s.mu.RLock()
	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.Arity >= 0 {
				log.Printf("    POST /api/%s/%s", mod.Path, exp.Name)
			}
		}
	}
	s.mu.RUnlock()

	log.Println()
	log.Println("  Introspection:")
	log.Println("    GET  /api/_meta/modules")
	log.Println("    GET  /api/_meta/openapi.json")
	log.Println("    GET  /api/_meta/docs            (Swagger UI)")
	log.Println("    GET  /api/_meta/redoc           (ReDoc)")
	log.Println("    GET  /api/_health")
	log.Println()
	log.Println("  Protocols:")
	log.Println("    GET  /.well-known/agent.json    (A2A Agent Card)")
	log.Println("    POST /a2a/                      (A2A JSON-RPC)")
	if s.mcpEnabled {
		log.Println("    POST /mcp/                      (MCP Streamable HTTP)")
	}

	if s.frontendPath != "" {
		log.Println()
		log.Printf("  Frontend: proxying to Vite at %s", s.frontendPath)
	}
	if s.staticPath != "" {
		log.Println()
		log.Printf("  Frontend: serving static files from %s", s.staticPath)
	}
	if s.watch {
		log.Println()
		log.Printf("  Hot reload: watching %d directories for .ail changes", len(s.getWatchDirs()))
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println()
}

// GetModules returns a copy of the loaded module info map.
func (s *Server) GetModules() map[string]*ModuleInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]*ModuleInfo, len(s.modules))
	for k, v := range s.modules {
		result[k] = v
	}
	return result
}

// GetEngine returns the underlying embed.Engine for advanced wiring
// (e.g., connecting FnCaller for stream event handlers).
func (s *Server) GetEngine() *embed.Engine {
	return s.engine
}

// Close shuts down the server and releases resources.
func (s *Server) Close() error {
	if s.watcher != nil {
		_ = s.watcher.Close()
	}
	if s.viteCmd != nil && s.viteCmd.Process != nil {
		_ = s.viteCmd.Process.Kill()
	}
	return s.engine.Close()
}
