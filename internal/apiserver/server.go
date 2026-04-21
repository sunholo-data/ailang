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
	"encoding/json"
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
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// DefaultMaxUploadSize is the default maximum upload size (50MB).
const DefaultMaxUploadSize = 50 << 20

// Server is the API server that exposes AILANG functions as REST endpoints.
type Server struct {
	engine *embed.Engine
	// modules is keyed by ModuleInfo.PhysicalPath (symlink-resolved absolute
	// file path). M-SERVEAPI-UNIFY: this is the single source of truth — no
	// other map or derivation writes to it. Every entry goes through
	// registerModule.
	modules map[string]*ModuleInfo
	mu      sync.RWMutex
	port    string
	// basePath is the user-supplied project root. Stored as the caller
	// passed it (for display / relative-path derivation convenience).
	basePath string
	// normalizedBasePath is basePath with filepath.Abs + filepath.EvalSymlinks
	// applied, plus a trailing separator. Used by registerModule's
	// under-basePath filter to compare against physical file paths.
	// Computed once at New() to avoid per-call symlink resolution.
	normalizedBasePath string
	cors               bool

	// Frontend proxy
	frontendPath string // path to React project (optional)
	staticPath   string // path to built static files (optional)
	viteCmd      *exec.Cmd

	// Hot reload
	watch      bool // whether file watching is enabled
	watcher    *fsnotify.Watcher
	watchPaths []string // absolute paths of loaded .ail files (for reload mapping)

	// Protocol support
	mcpEnabled    bool                // serve MCP at /mcp/
	mcpOnly       bool                // stdio-only MCP mode (no HTTP)
	a2aEnabled    bool                // serve A2A at /.well-known/agent.json and /a2a/
	maxUploadSize int64               // maximum upload size in bytes (0 = use DefaultMaxUploadSize)
	apiKeyHeader  string              // HTTP header name for API key auth
	apiKeyEnv     string              // env var containing expected API key
	effCtx        *effects.EffContext // for Debug output collection
	logLevel      int                 // minimum severity for Debug output
	routesOnly    bool                // only expose @route-annotated functions
}

// ModuleInfo holds metadata about a loaded AILANG module.
//
// M-SERVEAPI-UNIFY: ModuleInfo is the single source of truth for a local
// module's identity and all its projections. It is keyed in s.modules by
// PhysicalPath (symlink-resolved absolute file path) — NOT by the derived
// `Path` field. Every projection consumer (HTTP routes, OpenAPI, MCP,
// A2A, function dispatch) reads the field it needs from here, so drift
// between projection key derivations is structurally impossible.
//
// Non-JSON fields (PhysicalPath, File, Iface) are not serialized in the
// /modules endpoint response to keep the public API shape unchanged.
type ModuleInfo struct {
	// Path is the RelPath projection (forward-slash relative path with
	// .ail stripped). Kept as the primary JSON field for backwards compat.
	Path    string       `json:"path"`
	Exports []ExportInfo `json:"exports"`

	// M-SERVEAPI-UNIFY projection fields — populated once at registration
	// by registerModule. Read-only after that.

	// PhysicalPath is the symlink-resolved absolute file path. This is
	// the module's *identity*: two ModuleInfo values with the same
	// PhysicalPath refer to the same physical source file, regardless of
	// how they were discovered. This is the key used in s.modules.
	PhysicalPath string `json:"-"`

	// CanonicalID is the pipeline's canonical module ID (used by loader
	// cache and callFunction dispatch). Example: "docparse/services/mcp_tools".
	CanonicalID string `json:"-"`

	// DeclaredPath is the `module X` header as written in source. Used
	// to resolve imports from other local modules. Usually matches
	// CanonicalID; differs under `module_prefix` aliasing.
	DeclaredPath string `json:"-"`

	// File is the parsed AST for the source file. Used for doc comment
	// extraction and by the extract* helpers during registration.
	File *ast.File `json:"-"`

	// Iface is the type-checked interface from the pipeline. Used by
	// callFunction for signature lookup and by the MCP schema generator.
	Iface *iface.Iface `json:"-"`
}

// ExportInfo describes a single exported function from an AILANG module.
type ExportInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`                   // human-readable type signature
	Pure        bool     `json:"pure"`                   // whether the function is pure
	Arity       int      `json:"arity"`                  // number of parameters (-1 if not a function)
	ParamNames  []string `json:"param_names,omitempty"`  // parameter names in order (for named JSON binding)
	ParamTypes  []string `json:"param_types,omitempty"`  // parameter type strings in order (for zero-value padding)
	RouteMethod string   `json:"route_method,omitempty"` // custom HTTP method from @route annotation
	RoutePath   string   `json:"route_path,omitempty"`   // custom URL path from @route annotation
	IsRaw       bool     `json:"is_raw,omitempty"`       // @raw annotation: pass full HttpRequest record
	IsNowrap    bool     `json:"is_nowrap,omitempty"`    // @nowrap annotation: skip FunctionCallResponse envelope
	IsNoExpose  bool     `json:"is_no_expose,omitempty"` // @noexpose annotation: hide from HTTP endpoints
	MCPName     string   `json:"mcp_name,omitempty"`     // @mcp_name annotation: explicit MCP tool name override
	DocComment  string   `json:"doc_comment,omitempty"`  // doc comment (-- lines) preceding the function
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
	A2A           bool        // enable A2A endpoints (/.well-known/agent.json, /a2a/)
	MaxUploadSize int64       // max upload size in bytes (0 = DefaultMaxUploadSize)
	APIKeyHeader  string      // HTTP header for API key auth (empty = no auth)
	APIKeyEnv     string      // env var containing expected API key
	LogLevel      int         // minimum severity for Debug output (0=DEBUG, 1=INFO, 2=WARN, 3=ERROR, 4=NONE)
	RoutesOnly    bool        // only expose @route-annotated functions as HTTP endpoints
}

// New creates a new API server.
func New(basePath string, cfg Config) *Server {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	// Normalize basePath: absolute + symlink-resolved. registerModule
	// compares loader-resolved file paths against basePath to filter
	// package files from local files. Without consistent normalization,
	// macOS temp paths (/var/folders → /private/var/folders) break the
	// prefix check and either silently drop local files or register them
	// under wrong keys.
	if abs, err := filepath.Abs(basePath); err == nil {
		basePath = abs
	}
	if resolved, err := filepath.EvalSymlinks(basePath); err == nil {
		basePath = resolved
	}
	normalizedBase := filepath.Clean(basePath) + string(filepath.Separator)
	eng := embed.New(basePath)
	// Guard against Go's typed-nil interface gotcha: a *EffContext(nil) stored
	// in interface{} is != nil but causes a nil pointer dereference.
	var storedEffCtx *effects.EffContext
	if cfg.EffCtx != nil {
		if effCtx, ok := cfg.EffCtx.(*effects.EffContext); ok && effCtx != nil {
			eng.SetEffContext(cfg.EffCtx)
			storedEffCtx = effCtx
		}
	}
	maxUpload := cfg.MaxUploadSize
	if maxUpload == 0 {
		maxUpload = DefaultMaxUploadSize
	}
	return &Server{
		engine:             eng,
		modules:            make(map[string]*ModuleInfo),
		port:               cfg.Port,
		basePath:           basePath,
		normalizedBasePath: normalizedBase,
		cors:               cfg.CORS,
		frontendPath:       cfg.FrontendPath,
		staticPath:         cfg.StaticPath,
		watch:              cfg.Watch,
		mcpEnabled:         cfg.MCP,
		mcpOnly:            cfg.MCPOnly,
		a2aEnabled:         cfg.A2A,
		maxUploadSize:      maxUpload,
		apiKeyHeader:       cfg.APIKeyHeader,
		apiKeyEnv:          cfg.APIKeyEnv,
		effCtx:             storedEffCtx,
		logLevel:           cfg.LogLevel,
		routesOnly:         cfg.RoutesOnly,
	}
}

// flushDebugOutput collects Debug ghost effect logs and prints them to stderr,
// then resets the context for the next request. Respects s.logLevel for filtering.
func (s *Server) flushDebugOutput() {
	if s.effCtx == nil || s.effCtx.Debug == nil {
		return
	}
	out := s.effCtx.Debug.Collect()
	for _, l := range out.Logs {
		if s.logLevel > 0 {
			sev := extractServerSeverity(l.Message)
			if sev != "" && serverSeverityLevel(sev) < s.logLevel {
				continue
			}
		}
		log.Printf("[Debug] %s", l.Message)
	}
	for _, a := range out.Assertions {
		if !a.Passed {
			log.Printf("[Debug ASSERT FAIL] %s at %s", a.Message, a.Location)
		}
	}
	s.effCtx.Debug.Reset()
}

func extractServerSeverity(msg string) string {
	if len(msg) < 2 || msg[0] != '{' {
		return ""
	}
	var parsed struct {
		Severity string `json:"severity"`
	}
	if err := json.Unmarshal([]byte(msg), &parsed); err != nil {
		return ""
	}
	return parsed.Severity
}

func serverSeverityLevel(severity string) int {
	switch severity {
	case "DEBUG", "TRACE":
		return 0
	case "INFO":
		return 1
	case "WARNING":
		return 2
	case "ERROR":
		return 3
	default:
		return 1
	}
}

// LoadModules compiles and loads AILANG modules from the given paths.
// Each path can be a .ail file or a directory (scanned recursively for .ail files).
//
// M-SERVEAPI-UNIFY: both directory and file paths route through the
// unified loader. Directories go through LoadProject (project-wide
// single pass); single files go through loadFile, which compiles the
// file and registers every module from result.Modules via the SAME
// registerModule write site. There is no longer a separate dep-discovery
// loop — drift between projection key derivations is structurally
// impossible.
func (s *Server) LoadModules(paths []string) error {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", p, err)
		}

		if info.IsDir() {
			if err := s.LoadProject(context.Background(), p); err != nil {
				return err
			}
		} else {
			if err := s.loadFile(p); err != nil {
				return err
			}
		}
	}

	// Eagerly evaluate all loaded modules so they're fully initialized before
	// any HTTP requests arrive. This prevents deadlocks where concurrent requests
	// trigger lazy LoadAndEvaluate under the Engine's write lock.
	//
	// Iterate by CanonicalID (populated unconditionally by registerModule)
	// because engine.Load expects canonical IDs, not the PhysicalPath keys.
	s.mu.RLock()
	modPaths := make([]string, 0, len(s.modules))
	for _, entry := range s.modules {
		if entry == nil || entry.CanonicalID == "" {
			continue
		}
		modPaths = append(modPaths, entry.CanonicalID)
	}
	s.mu.RUnlock()

	for _, modPath := range modPaths {
		// Skip eager loading for package dependency modules. These are
		// already preloaded into the engine's loader cache via
		// PreloadModule. Re-running compileModule() for pkg/ paths can
		// overwrite the preloaded cache entries with modules compiled
		// from a different basePath, corrupting the canonical paths.
		if strings.HasPrefix(modPath, "pkg/") {
			continue
		}
		if err := s.engine.Load(modPath); err != nil {
			log.Printf("  Warning: eager load failed for %s: %v", modPath, err)
		}
	}

	return nil
}

// loadFile compiles a single .ail file and registers every module from
// the resulting result.Modules via registerModule. Used for the
// single-file form of LoadModules and for the watcher's reloadFile path.
//
// M-SERVEAPI-UNIFY: this is now a thin wrapper that mirrors LoadProject
// for one file. There is no dep-discovery loop and no second key
// derivation — registerModule is the SOLE write site to s.modules.
func (s *Server) loadFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", path, err)
	}
	if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
		absPath = resolved
	}

	cfg := pipeline.Config{
		Mode:         pipeline.ModeCheck,
		RelaxModules: true,
	}
	src := pipeline.Source{Filename: absPath}

	result, err := pipeline.RunWithContext(context.Background(), cfg, src)
	if err != nil {
		return fmt.Errorf("compilation error for %s: %w", path, err)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("compilation errors for %s: %v", path, result.Errors)
	}

	// Preload every transitively resolved module into the runtime, then
	// register every local one via registerModule. registerModule's
	// under-basePath filter handles local-vs-package classification.
	if result.Modules != nil {
		for modID, loaded := range result.Modules {
			s.engine.PreloadModule(modID, loaded)
			// Aliasing: also preload under the declared module path when
			// it differs from the resolved key (module_prefix support).
			if loaded.Path != "" && loaded.Path != modID {
				s.engine.PreloadModule(loaded.Path, loaded)
			}
			if _, _, regErr := s.registerModule(loaded); regErr != nil {
				return regErr
			}
		}
	}

	// Track absolute path for file watching.
	s.watchPaths = append(s.watchPaths, absPath)
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

	// A2A Agent Card (opt-in via --a2a flag)
	if s.a2aEnabled {
		mux.HandleFunc("/.well-known/agent.json", s.corsWrap(s.handleA2AAgentCard))
		mux.HandleFunc("/a2a/", s.corsWrap(s.handleA2ATask))
	}

	// MCP endpoint (streamable HTTP transport)
	if s.mcpEnabled {
		mcpSrv := NewMCPServer(s)
		mux.Handle("/mcp/", http.StripPrefix("/mcp", mcpSrv.HTTPHandler()))
	}

	// Build set of built-in paths to prevent @route collisions (Go 1.22+ panics on duplicates)
	builtinPaths := map[string]bool{
		"/api/_meta/modules":      true,
		"/api/_meta/modules/":     true,
		"/api/_health":            true,
		"/api/_meta/openapi.json": true,
		"/api/_meta/docs":         true,
		"/api/_meta/redoc":        true,
		"/api/":                   true,
	}
	if s.a2aEnabled {
		builtinPaths["/.well-known/agent.json"] = true
		builtinPaths["/a2a/"] = true
	}
	if s.mcpEnabled {
		builtinPaths["/mcp/"] = true
	}

	// Custom routes from @route annotations (registered before catch-all)
	// Auth middleware wraps custom routes and the catch-all
	s.registerCustomRoutes(mux, builtinPaths)

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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
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
	exposed, filtered := 0, 0
	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.Arity < 0 {
				continue
			}
			if !s.isExposed(exp) {
				filtered++
				continue
			}
			exposed++
			if exp.RoutePath != "" {
				log.Printf("    %s %s", exp.RouteMethod, exp.RoutePath)
			} else {
				log.Printf("    POST /api/%s/%s", mod.Path, exp.Name)
			}
		}
	}
	s.mu.RUnlock()

	if filtered > 0 {
		log.Printf("  (%d endpoints exposed, %d filtered)", exposed, filtered)
	}

	log.Println()
	log.Println("  Introspection:")
	log.Println("    GET  /api/_meta/modules")
	log.Println("    GET  /api/_meta/openapi.json")
	log.Println("    GET  /api/_meta/docs            (Swagger UI)")
	log.Println("    GET  /api/_meta/redoc           (ReDoc)")
	log.Println("    GET  /api/_health")
	log.Println()
	log.Println("  Protocols:")
	if s.a2aEnabled {
		log.Println("    GET  /.well-known/agent.json    (A2A Agent Card)")
		log.Println("    POST /a2a/                      (A2A JSON-RPC)")
	}
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
