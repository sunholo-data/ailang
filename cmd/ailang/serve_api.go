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
)

func serveAPICommand(args []string) error {
	fs := flag.NewFlagSet("serve-api", flag.ExitOnError)
	portFlag := fs.String("port", "8080", "HTTP server port")
	corsFlag := fs.Bool("cors", true, "Enable CORS for all origins")
	frontendFlag := fs.String("frontend", "", "Path to React/Vite project (proxies non-/api/ requests to Vite dev server)")
	staticFlag := fs.String("static", "", "Path to built frontend files (serve as static files)")
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

	cfg := apiserver.Config{
		Port:         *portFlag,
		CORS:         *corsFlag,
		FrontendPath: *frontendFlag,
		StaticPath:   *staticFlag,
	}

	srv := apiserver.New(basePath, cfg)
	defer srv.Close()

	log.Println("Loading AILANG modules...")
	if err := srv.LoadModules(paths); err != nil {
		return fmt.Errorf("failed to load modules: %w", err)
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
	fmt.Println("  --help               Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang serve-api api/handlers.ail")
	fmt.Println("  ailang serve-api ./api/ --port 3000")
	fmt.Println("  ailang serve-api ./api/ --frontend ./ui")
	fmt.Println("  ailang serve-api ./api/ --static ./ui/dist")
	fmt.Println()
	fmt.Println("Endpoints generated for each exported function:")
	fmt.Println("  POST /api/{module}/{function}")
	fmt.Println("    Body: {\"args\": [arg1, arg2, ...]}  OR  single JSON value")
	fmt.Println("    Response: {\"result\": ..., \"module\": \"...\", \"func\": \"...\", \"elapsed_ms\": 5}")
	fmt.Println()
	fmt.Println("Introspection endpoints:")
	fmt.Println("  GET  /api/_meta/modules           List all loaded modules and exports")
	fmt.Println("  GET  /api/_meta/modules/{path}    Details for a specific module")
	fmt.Println("  GET  /api/_health                 Health check")
}
