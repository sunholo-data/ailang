package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// moduleDoc represents documentation for a stdlib module
type moduleDoc struct {
	Name        string      // e.g., "std/io"
	Description string      // First comment block (module description)
	Exports     []exportDoc // Exported functions
	Examples    []string    // Usage examples from comments
	FilePath    string      // Full path to .ail file
}

// exportDoc represents an exported function
type exportDoc struct {
	Name      string // Function name
	Signature string // Full signature line
	DocLine   string // Comment line above export
}

// docsCommand implements `ailang docs` command
func docsCommand() {
	// Check for subcommands first
	args := flag.Args()[1:]
	if len(args) > 0 && args[0] == "search" {
		docsSearchCommand(args[1:])
		return
	}
	if len(args) > 0 && args[0] == "embed-warmup" {
		docsEmbedWarmupCommand(args[1:])
		return
	}

	// Parse subcommand flags
	docsFlags := flag.NewFlagSet("docs", flag.ExitOnError)
	listFlag := docsFlags.Bool("list", false, "List all available stdlib modules")
	examplesFlag := docsFlags.Bool("examples", false, "Show usage examples")
	helpFlag := docsFlags.Bool("help", false, "Show help for docs command")

	if err := docsFlags.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if *helpFlag {
		printDocsHelp()
		return
	}

	// Find stdlib directory
	stdlibPath, err := findStdlibDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		fmt.Fprintf(os.Stderr, "\nTip: Set AILANG_STDLIB_PATH or run from project root\n")
		os.Exit(1)
	}

	// List mode
	if *listFlag {
		listModules(stdlibPath)
		return
	}

	// View specific module
	if docsFlags.NArg() >= 1 {
		moduleName := docsFlags.Arg(0)
		showModuleDocs(stdlibPath, moduleName, *examplesFlag)
		return
	}

	// No arguments - show help
	printDocsHelp()
}

// findStdlibDir finds the stdlib directory
func findStdlibDir() (string, error) {
	// Priority order:
	// 1. AILANG_STDLIB_PATH environment variable
	// 2. ./std (current directory)
	// 3. ../std (parent directory)

	if envPath := os.Getenv("AILANG_STDLIB_PATH"); envPath != "" {
		if isStdlibDir(envPath) {
			return envPath, nil
		}
	}

	// Check ./std
	if isStdlibDir("std") {
		absPath, _ := filepath.Abs("std")
		return absPath, nil
	}

	// Check ../std (for running from cmd/ailang)
	if isStdlibDir("../std") {
		absPath, _ := filepath.Abs("../std")
		return absPath, nil
	}

	return "", fmt.Errorf("stdlib directory not found")
}

// isStdlibDir checks if path looks like stdlib directory
func isStdlibDir(path string) bool {
	// Check for existence of common stdlib files
	ioPath := filepath.Join(path, "io.ail")
	if _, err := os.Stat(ioPath); err == nil {
		return true
	}
	return false
}

// listModules lists all available stdlib modules
func listModules(stdlibPath string) {
	modules := discoverModules(stdlibPath)

	fmt.Println("Available stdlib modules:")
	fmt.Println()

	for _, mod := range modules {
		desc := mod.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Printf("  %-14s %s\n", mod.Name, desc)
	}

	fmt.Println()
	fmt.Printf("Use 'ailang docs <module>' for details (e.g., ailang docs std/io)\n")
}

// discoverModules finds all stdlib modules
func discoverModules(stdlibPath string) []moduleDoc {
	var modules []moduleDoc

	entries, err := os.ReadDir(stdlibPath)
	if err != nil {
		return modules
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".ail") {
			continue
		}

		filePath := filepath.Join(stdlibPath, entry.Name())
		mod := parseModuleFile(filePath)
		if mod.Name != "" {
			modules = append(modules, mod)
		}
	}

	// Sort by name
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	return modules
}

// parseModuleFile parses a stdlib .ail file for documentation
func parseModuleFile(filePath string) moduleDoc {
	file, err := os.Open(filePath)
	if err != nil {
		return moduleDoc{}
	}
	defer file.Close()

	mod := moduleDoc{FilePath: filePath}
	scanner := bufio.NewScanner(file)

	var headerComments []string
	var currentDoc string
	inHeaderBlock := true
	exampleBlock := false
	var examples []string

	// Regex patterns
	moduleRe := regexp.MustCompile(`^module\s+(std/\w+)`)
	exportFuncRe := regexp.MustCompile(`^export\s+(?:pure\s+)?func\s+(\w+)`)
	// Match both short form (export func f(x) = ...) and block form (export func f(x) -> T { ... })
	exportSigRe := regexp.MustCompile(`^export\s+(?:pure\s+)?func\s+\w+\s*(?:\[[^\]]+\])?\s*\([^)]*\)(?:\s*->\s*[^{=]+)?`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Track example blocks
		if strings.Contains(trimmed, "Usage:") || strings.Contains(trimmed, "Example") {
			exampleBlock = true
		}

		// Collect header comments (module description)
		if inHeaderBlock && strings.HasPrefix(trimmed, "--") {
			comment := strings.TrimPrefix(trimmed, "--")
			comment = strings.TrimSpace(comment)
			headerComments = append(headerComments, comment)

			if exampleBlock && comment != "" && !strings.HasPrefix(comment, "=") {
				examples = append(examples, comment)
			}
			continue
		}

		// Module declaration
		if match := moduleRe.FindStringSubmatch(trimmed); match != nil {
			mod.Name = match[1]
			inHeaderBlock = false
			exampleBlock = false

			// Build description from header comments
			if len(headerComments) > 0 {
				// Skip file name comment (e.g., "-- std/io.ail")
				for _, c := range headerComments {
					if c != "" && !strings.HasPrefix(c, "std/") && !strings.HasPrefix(c, "=") {
						mod.Description = c
						break
					}
				}
			}
			continue
		}

		// Documentation comment before export
		if strings.HasPrefix(trimmed, "--") {
			currentDoc = strings.TrimSpace(strings.TrimPrefix(trimmed, "--"))
			continue
		}

		// Export declaration
		if match := exportFuncRe.FindStringSubmatch(trimmed); match != nil {
			funcName := match[1]

			// Join continuation lines for multi-line declarations
			// (e.g., params spanning multiple lines)
			fullLine := trimmed
			for strings.Count(fullLine, "(") > strings.Count(fullLine, ")") {
				if !scanner.Scan() {
					break
				}
				fullLine += " " + strings.TrimSpace(scanner.Text())
			}

			// Extract full signature from joined line
			sig := ""
			if sigMatch := exportSigRe.FindString(fullLine); sigMatch != "" {
				sig = sigMatch
			}

			export := exportDoc{
				Name:      funcName,
				Signature: sig,
				DocLine:   currentDoc,
			}
			mod.Exports = append(mod.Exports, export)
			currentDoc = ""
		}
	}

	mod.Examples = examples
	return mod
}

// showModuleDocs displays documentation for a specific module
func showModuleDocs(stdlibPath, moduleName string, showExamples bool) {
	// Normalize module name (allow both "std/io" and "io")
	if !strings.HasPrefix(moduleName, "std/") {
		moduleName = "std/" + moduleName
	}

	// Find the file
	fileName := strings.TrimPrefix(moduleName, "std/") + ".ail"
	filePath := filepath.Join(stdlibPath, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "%s: module '%s' not found\n", red("Error"), moduleName)
		fmt.Fprintf(os.Stderr, "\nUse 'ailang docs --list' to see available modules\n")
		os.Exit(1)
	}

	mod := parseModuleFile(filePath)

	// Print module header
	fmt.Printf("# %s\n", mod.Name)
	if mod.Description != "" {
		fmt.Printf("%s\n", mod.Description)
	}
	fmt.Println()

	// Print exports
	if len(mod.Exports) > 0 {
		fmt.Println("## Exports")
		fmt.Println()

		for _, exp := range mod.Exports {
			// Format signature nicely
			sig := formatExportSignature(exp.Signature)
			fmt.Printf("  %s\n", sig)
			if exp.DocLine != "" {
				fmt.Printf("    %s\n", exp.DocLine)
			}
			fmt.Println()
		}
	}

	// Print usage hint
	fmt.Println("## Usage")
	fmt.Println()
	shortName := strings.TrimPrefix(mod.Name, "std/")
	if len(mod.Exports) > 0 {
		// Show first export name
		firstExport := mod.Exports[0].Name
		fmt.Printf("  import %s (%s)\n", mod.Name, firstExport)
	}
	fmt.Printf("  -- or --\n")
	fmt.Printf("  import %s as %s\n", mod.Name, capitalize(shortName))

	// Print examples if requested
	if showExamples && len(mod.Examples) > 0 {
		fmt.Println()
		fmt.Println("## Examples")
		fmt.Println()
		for _, ex := range mod.Examples {
			fmt.Printf("  %s\n", ex)
		}
	}

	// Print a "Try it" section of registered runnable examples that import this
	// module (additive; only when --examples is passed).
	if showExamples {
		printTryItSection(mod.Name)
	}
}

// printTryItSection prints registered runnable examples that import moduleName
// (e.g. "std/gzip"), read from examples/manifest.json's `modules` field. It
// reuses loadExamplesManifest -> findExamplesDir resolution, so it works both
// in-repo and from an installed binary (AILANG_EXAMPLES / ~/.ailang/examples).
//
// Behavior is never silently identical to flagless output:
//   - resolution failure -> prints the explicit findExamplesDir error as a note
//     under the doc (the doc is still shown above);
//   - no matches -> says so explicitly.
func printTryItSection(moduleName string) {
	fmt.Println()
	fmt.Println("## Try it")
	fmt.Println()

	m, err := loadExamplesManifest()
	if err != nil {
		// Explicit note, never a silent fallback (see findExamplesDir error text).
		fmt.Printf("  (could not load registered examples: %v)\n", err)
		return
	}

	var matches []ExampleEntry
	for _, e := range m.Examples {
		if e.Status != "working" {
			continue
		}
		for _, mod := range e.Modules {
			if mod == moduleName {
				matches = append(matches, e)
				break
			}
		}
	}

	if len(matches) == 0 {
		fmt.Printf("  No registered examples import %s yet\n", moduleName)
		return
	}

	// Deterministic order.
	sort.Slice(matches, func(i, j int) bool { return matches[i].Path < matches[j].Path })

	shown := 0
	for _, e := range matches {
		if shown >= 3 {
			break
		}
		path := exampleRunPath(e.Path)
		desc := e.Description
		if desc != "" {
			fmt.Printf("  %s — %s\n", path, desc)
		} else {
			fmt.Printf("  %s\n", path)
		}
		fmt.Printf("    ailang run %s\n", path)
		shown++
	}
	if len(matches) > shown {
		short := strings.TrimPrefix(moduleName, "std/")
		fmt.Printf("\n  (%d more: ailang examples search %s)\n", len(matches)-shown, short)
	}
}

// exampleRunPath normalizes a manifest entry path to a `examples/...`-rooted
// path suitable for `ailang run`. Legacy bare-name entries live under runnable/.
// Always slash-form: this is a display/run-command string, and `ailang run`
// accepts slash paths on every platform (Windows renders backslashes otherwise).
func exampleRunPath(entryPath string) string {
	if strings.HasPrefix(entryPath, "runnable/") ||
		strings.HasPrefix(entryPath, "experimental/") ||
		strings.Contains(entryPath, "/") {
		return filepath.ToSlash(filepath.Join("examples", entryPath))
	}
	return filepath.ToSlash(filepath.Join("examples", "runnable", entryPath))
}

// formatExportSignature formats an export signature for display
func formatExportSignature(sig string) string {
	// Remove "export" and "pure" keywords, keep just the function signature
	sig = strings.TrimPrefix(sig, "export ")
	sig = strings.TrimPrefix(sig, "pure ")
	sig = strings.TrimPrefix(sig, "func ")
	return sig
}

// printDocsHelp prints help for docs command
func printDocsHelp() {
	fmt.Println("Usage: ailang docs [options] [module]")
	fmt.Println()
	fmt.Println("Show documentation for AILANG stdlib modules.")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --list        List all available stdlib modules")
	fmt.Println("  --examples    Also print a 'Try it' section of registered runnable")
	fmt.Println("                examples that import the module (with run commands)")
	fmt.Println("  --help        Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang docs --list              List all modules")
	fmt.Println("  ailang docs std/io              Show std/io documentation")
	fmt.Println("  ailang docs io                  Short form (same as std/io)")
	fmt.Println("  ailang docs --examples gzip     Show std/gzip docs + runnable examples")
}
