package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

	// A help flag AFTER the type never reaches helpFlag above: flag.Parse stops
	// at the first non-flag argument, so in `init web-app --help` everything
	// past "web-app" is positional and unparsed. Without this, --help became
	// the project NAME — `ailang init web-app --help` printed "Creating AILANG
	// web app: --help" and scaffolded a directory called "--help" (one is in
	// this repo, made 2026-09-18 01:55). Help is what was asked for; give it.
	for _, a := range flagSet.Args()[1:] {
		if a == "--help" || a == "-help" || a == "-h" {
			printInitHelp()
			return nil
		}
	}

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

// checkScaffoldName rejects a project name that is really a misplaced flag.
// Split out so the rule is one thing to state and one thing to test, and so
// any future `init <type>` shares it rather than re-deriving it.
func checkScaffoldName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is empty (an unset shell variable?)")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("refusing to create a directory named %q: it starts with a dash, "+
			"so it is a FLAG that landed in the project-name slot rather than a name.\n"+
			"`ailang init web-app` takes the name positionally, and flag parsing stops at "+
			"the type, so any option after it is read as the name.\n"+
			"For help: ailang init --help", name)
	}
	return nil
}

func initWebApp(name string) error {
	// Refuse to name a directory after a flag. The help flags are handled in
	// initCommand, so anything dash-leading reaching here is a flag that landed
	// in the positional slot — an unknown option, or a shell variable that
	// expanded empty. Creating the directory anyway is how "--help" ends up on
	// disk, and mkdir is the wrong moment to discover the typo.
	if err := checkScaffoldName(name); err != nil {
		return err
	}

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
