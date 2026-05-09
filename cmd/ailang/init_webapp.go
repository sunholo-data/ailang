package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/apiserver/templates"
)

func initCommand(args []string) error {
	flagSet := flag.NewFlagSet("init", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || flagSet.NArg() < 1 {
		printInitHelp()
		return nil
	}

	kind := flagSet.Arg(0)

	switch kind {
	case "web-app":
		name := "my-ailang-app"
		if flagSet.NArg() >= 2 {
			name = flagSet.Arg(1)
		}
		return initWebApp(name)
	case "package":
		return initPackageCommand(flagSet.Args()[1:])
	case "motoko-extension":
		return initMotokoExtensionCommand(flagSet.Args()[1:])
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown init type %q\n", red("Error"), kind)
		printInitHelp()
		os.Exit(1)
		return nil
	}
}

func initWebApp(name string) error {
	// Check target directory doesn't exist
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("directory %q already exists", name)
	}

	fmt.Printf("Creating AILANG web app: %s\n", name)

	// Copy embedded template files
	err := copyEmbeddedDir(templates.WebAppFS, "web_app", name)
	if err != nil {
		return fmt.Errorf("failed to scaffold project: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Created %s/\n", name)
	fmt.Println()
	fmt.Println("  Get started:")
	fmt.Printf("    cd %s\n", name)
	fmt.Println("    cd ui && npm install && cd ..")
	fmt.Println("    make dev")
	fmt.Println()
	fmt.Println("  This starts:")
	fmt.Println("    - AILANG API server on http://localhost:8080")
	fmt.Println("    - React dev server on http://localhost:5173 (proxies /api)")
	fmt.Println()
	fmt.Println("  Your AILANG API modules are in api/")
	fmt.Println("  Your React frontend is in ui/")

	return nil
}

// copyEmbeddedDir recursively copies files from an embedded filesystem to disk.
func copyEmbeddedDir(embeddedFS fs.FS, srcRoot string, dstRoot string) error {
	return fs.WalkDir(embeddedFS, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path from srcRoot
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}

		dst := filepath.Join(dstRoot, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}

		// Read embedded file
		content, err := fs.ReadFile(embeddedFS, path)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", path, err)
		}

		// Write to disk
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}

		return os.WriteFile(dst, content, 0644)
	})
}

func printInitHelp() {
	fmt.Println("Usage: ailang init <type> [name]")
	fmt.Println()
	fmt.Println("Scaffold a new AILANG project.")
	fmt.Println()
	fmt.Println("Types:")
	fmt.Println("  web-app    Create a web app with AILANG API backend + React frontend")
	fmt.Println("  package    Create an ailang.toml package manifest in the current directory")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang init web-app myproject")
	fmt.Println("  ailang init web-app")
	fmt.Println("  ailang init package --name sunholo/mylib")
}
