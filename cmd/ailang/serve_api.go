package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/apiserver"
	"github.com/sunholo/ailang/internal/effects"
)

func serveAPICommand(args []string) error {
	fs := flag.NewFlagSet("serve-api", flag.ExitOnError)
	portFlag := fs.String("port", "8080", "HTTP server port")
	corsFlag := fs.Bool("cors", true, "Enable CORS for all origins")
	frontendFlag := fs.String("frontend", "", "Path to React/Vite project (proxies non-/api/ requests to Vite dev server)")
	staticFlag := fs.String("static", "", "Path to built frontend files (serve as static files)")
	watchFlag := fs.Bool("watch", false, "Watch .ail files for changes and hot-reload")
	capsFlag := fs.String("caps", "", "Capabilities to grant (comma-separated: IO,FS,Net,AI,Clock,Env,Stream)")
	aiModelFlag := fs.String("ai", "", "AI model for AI effect (e.g., gemini-2-5-flash, claude-sonnet-4-6)")
	aiStubFlag := fs.Bool("ai-stub", false, "Use stub AI handler (for testing)")
	verifyContractsFlag := fs.Bool("verify-contracts", false, "Enable runtime contract validation (requires/ensures)")
	mcpFlag := fs.Bool("mcp", false, "Run as MCP stdio server (for Claude Desktop, Cursor, etc.)")
	mcpHTTPFlag := fs.Bool("mcp-http", false, "Enable MCP HTTP endpoint at /mcp/")
	helpFlag := fs.Bool("help", false, "Show help for serve-api command")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		printServeAPIHelp()
		return nil
	}

	if fs.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "%s: missing path argument\n", red("Error"))
		printServeAPIHelp()
		os.Exit(1)
	}

	// Resolve all paths to absolute
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	paths := make([]string, fs.NArg())
	for i := 0; i < fs.NArg(); i++ {
		p := fs.Arg(i)
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		paths[i] = p
	}

	// Derive basePath from the provided files.
	// Read the first .ail file's module declaration to compute the project root.
	// e.g., file at /tmp/myproject/api/handlers.ail declaring "module api/handlers"
	// gives basePath = /tmp/myproject/
	basePath := cwd
	if declaredMod, absFile := findFirstModuleDecl(paths); declaredMod != "" {
		suffix := filepath.FromSlash(declaredMod) + ".ail"
		if strings.HasSuffix(absFile, suffix) {
			basePath = strings.TrimSuffix(absFile, suffix)
			basePath = strings.TrimRight(basePath, string(filepath.Separator))
		}
	}

	// Set up effect context if capabilities, AI, or contract flags are provided
	var effCtx *effects.EffContext
	if *capsFlag != "" || *aiModelFlag != "" || *aiStubFlag || *verifyContractsFlag {
		effCtx = effects.NewEffContext(nil)
		grantCapabilities(effCtx, *capsFlag)
		if err := setupAIHandler(effCtx, *aiStubFlag, *aiModelFlag); err != nil {
			return fmt.Errorf("AI handler setup failed: %w", err)
		}

		// Enable contract verification if requested
		if *verifyContractsFlag {
			effCtx.Contracts = effects.NewContractContextWithMode(effects.ContractModePanic)
			log.Println("Contract verification enabled (panic mode)")
		}

		// Initialize Stream context if Stream capability is granted
		if effCtx.HasCap("Stream") {
			effCtx.Stream = effects.NewStreamContext()
			effCtx.Stream.AllowHTTP = true      // serve-api typically proxies to known APIs
			effCtx.Stream.AllowLocalhost = true // local development
			log.Println("Stream capability enabled (SSE/WebSocket client)")
		}
	}

	cfg := apiserver.Config{
		Port:         *portFlag,
		CORS:         *corsFlag,
		FrontendPath: *frontendFlag,
		StaticPath:   *staticFlag,
		Watch:        *watchFlag,
		EffCtx:       effCtx,
		MCP:          *mcpHTTPFlag,
		MCPOnly:      *mcpFlag,
	}

	srv := apiserver.New(basePath, cfg)
	defer srv.Close()

	log.Println("Loading AILANG modules...")
	if err := srv.LoadModules(paths); err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
	}

	// Wire FnCaller for stream event handler dispatch (must happen after modules loaded)
	if effCtx != nil && effCtx.Stream != nil {
		effCtx.FnCaller = srv.GetEngine().GetCallValue()
		log.Println("Stream FnCaller wired for event handler dispatch")
	}

	return srv.Start()
}

// findFirstModuleDecl scans the provided paths for the first .ail file
// and extracts its module declaration. Returns (declaredModule, absFilePath).
func findFirstModuleDecl(paths []string) (string, string) {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			// Scan directory for first .ail file
			_ = filepath.Walk(p, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() || !strings.HasSuffix(path, ".ail") {
					return nil
				}
				p = path
				return filepath.SkipAll
			})
		}
		if !strings.HasSuffix(p, ".ail") {
			continue
		}

		mod := readModuleDecl(p)
		if mod != "" {
			absP, _ := filepath.Abs(p)
			return mod, absP
		}
	}
	return "", ""
}

// readModuleDecl reads the module declaration from an .ail file.
// Returns empty string if not found.
func readModuleDecl(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "--") || line == "" {
			continue // skip comments and blank lines
		}
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
		break // first non-comment, non-empty line is not a module decl
	}
	return ""
}

func printServeAPIHelp() {
	fmt.Println("Usage: ailang serve-api [options] <path...>")
	fmt.Println()
	fmt.Println("Start an HTTP server that exposes AILANG module exports as REST endpoints.")
	fmt.Println("Each exported function becomes a POST endpoint at /api/{module}/{function}.")
	fmt.Println()
	fmt.Println("Arguments:")
	fmt.Println("  <path...>            One or more .ail files or directories to load")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --port PORT          HTTP server port (default: 8080)")
	fmt.Println("  --cors               Enable CORS for all origins (default: true)")
	fmt.Println("  --frontend PATH      Path to React/Vite project for dev proxy")
	fmt.Println("  --static PATH        Path to built frontend files")
	fmt.Println("  --watch              Watch .ail files for changes and hot-reload")
	fmt.Println("  --caps CAPS          Capabilities to grant (comma-separated: IO,FS,Net,AI,Clock,Env)")
	fmt.Println("  --ai MODEL           AI model for AI effect (e.g., gemini-2-5-flash, claude-sonnet-4-6)")
	fmt.Println("  --ai-stub            Use stub AI handler (for testing)")
	fmt.Println("  --verify-contracts   Enable runtime contract validation (requires/ensures)")
	fmt.Println("  --mcp                Run as MCP stdio server (for Claude Desktop, Cursor)")
	fmt.Println("  --mcp-http           Enable MCP HTTP endpoint at /mcp/")
	fmt.Println("  --help               Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang serve-api api/handlers.ail")
	fmt.Println("  ailang serve-api ./api/ --port 3000")
	fmt.Println("  ailang serve-api ./api/ --frontend ./ui")
	fmt.Println("  ailang serve-api ./api/ --static ./ui/dist")
	fmt.Println("  ailang serve-api --watch ./api/")
	fmt.Println("  ailang serve-api --caps IO,AI --ai gemini-2-5-flash ./api/")
	fmt.Println("  ailang serve-api --mcp ./api/                        # MCP stdio server")
	fmt.Println("  ailang serve-api --mcp-http ./api/                   # HTTP + MCP at /mcp/")
	fmt.Println()
	fmt.Println("Endpoints generated for each exported function:")
	fmt.Println("  POST /api/{module}/{function}")
	fmt.Println("    Body: {\"args\": [arg1, arg2, ...]}  OR  single JSON value")
	fmt.Println("    Response: {\"result\": ..., \"module\": \"...\", \"func\": \"...\", \"elapsed_ms\": 5}")
	fmt.Println()
	fmt.Println("Introspection endpoints:")
	fmt.Println("  GET  /api/_meta/modules           List all loaded modules and exports")
	fmt.Println("  GET  /api/_meta/openapi.json      OpenAPI 3.1 spec")
	fmt.Println("  GET  /api/_meta/modules/{path}    Details for a specific module")
	fmt.Println("  GET  /api/_health                 Health check")
	fmt.Println()
	fmt.Println("Protocol endpoints:")
	fmt.Println("  GET  /.well-known/agent.json      A2A Agent Card")
	fmt.Println("  POST /a2a/                        A2A JSON-RPC task endpoint")
	fmt.Println("  POST /mcp/                        MCP streamable HTTP (with --mcp-http)")
}
