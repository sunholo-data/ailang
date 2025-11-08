package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/builtins"
)

// runDoctor implements `ailang doctor builtins` command
func runDoctor() {
	subcommand := ""
	if flag.NArg() >= 2 {
		subcommand = flag.Arg(1)
	}

	if subcommand != "builtins" {
		fmt.Println("Usage: ailang doctor builtins")
		fmt.Println()
		fmt.Println("Available subcommands:")
		fmt.Println("  builtins    Validate the builtin function registry")
		os.Exit(1)
	}

	// Validate builtins from new spec-based registry (M-DX1 complete in v0.3.10)
	errors := builtins.ValidateBuiltins()

	if len(errors) == 0 {
		fmt.Println("✅ All builtins are valid!")

		// Show stats
		stats := builtins.GetRegistryStats()
		fmt.Printf("\nRegistry Statistics:\n")
		fmt.Printf("  Total:      %d builtins\n", stats.Total)
		fmt.Printf("  Pure:       %d\n", stats.Pure)
		fmt.Printf("  Effectful:  %d\n", stats.Effect)
		fmt.Printf("\nBy Effect:\n")
		for effect, count := range stats.ByEffect {
			fmt.Printf("  %-10s %d\n", effect+":", count)
		}
		fmt.Printf("\nBy Module:\n")
		for module, count := range stats.ByModule {
			fmt.Printf("  %-20s %d\n", module+":", count)
		}
		return
	}

	// Report errors
	errorCount := 0
	warningCount := 0
	for _, err := range errors {
		if err.Severity == "error" {
			errorCount++
		} else {
			warningCount++
		}
	}

	fmt.Printf("❌ Found %d errors, %d warnings\n\n", errorCount, warningCount)

	for i, err := range errors {
		icon := "⚠️"
		if err.Severity == "error" {
			icon = "❌"
		}

		fmt.Printf("%d. %s %s: %s\n", i+1, icon, err.Builtin, err.Message)
		fmt.Printf("   Location: %s\n", err.Location)
		fmt.Printf("   Fix: %s\n", err.Fix)
		fmt.Println()
	}

	// Exit with error if any errors found
	if errorCount > 0 {
		os.Exit(1)
	}
}

// runBuiltins implements `ailang builtins list` command
func runBuiltins() {
	subcommand := ""
	if flag.NArg() >= 2 {
		subcommand = flag.Arg(1)
	}

	switch subcommand {
	case "list":
		runBuiltinsList()
	case "check-migration":
		runBuiltinsCheckMigration()
	default:
		fmt.Println("Usage: ailang builtins <subcommand>")
		fmt.Println()
		fmt.Println("Available subcommands:")
		fmt.Println("  list              List all registered builtins")
		fmt.Println("  check-migration   Validate that all builtins have been migrated")
		os.Exit(1)
	}
}

func runBuiltinsList() {

	// Parse additional flags for list command
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	byEffect := listFlags.Bool("by-effect", false, "Group by effect type")
	byModule := listFlags.Bool("by-module", false, "Group by module")
	_ = listFlags.Parse(flag.Args()[2:]) // ExitOnError means this never returns an error we can handle

	// Get all specs from new registry (M-DX1 complete in v0.3.10)
	specs := builtins.AllSpecs()

	if len(specs) == 0 {
		fmt.Println("No builtins registered")
		return
	}

	// Choose display mode
	if *byEffect {
		listBuiltinsByEffect(specs)
	} else if *byModule {
		listBuiltinsByModule(specs)
	} else {
		listAllBuiltins(specs)
	}
}

func listAllBuiltins(specs map[string]*builtins.BuiltinSpec) {
	// Get sorted names
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sortStrings(names)

	fmt.Printf("Total: %d builtins\n\n", len(specs))

	for _, name := range names {
		spec := specs[name]
		effect := "pure"
		if !spec.IsPure {
			effect = strings.ToLower(spec.Effect)
		}
		fmt.Printf("  %-30s [%s] %s\n", name, effect, spec.Module)
	}
}

func listBuiltinsByEffect(specs map[string]*builtins.BuiltinSpec) {
	grouped := builtins.GroupByEffect()

	// Sort effect names
	effects := make([]string, 0, len(grouped))
	for effect := range grouped {
		effects = append(effects, effect)
	}
	sortStrings(effects)

	for _, effect := range effects {
		names := grouped[effect]
		fmt.Printf("# %s (%d)\n", effect, len(names))
		for _, name := range names {
			spec := specs[name]
			fmt.Printf("  %-30s %s\n", name, spec.Module)
		}
		fmt.Println()
	}
}

func listBuiltinsByModule(specs map[string]*builtins.BuiltinSpec) {
	grouped := builtins.GroupByModule()

	// Sort module names
	modules := make([]string, 0, len(grouped))
	for module := range grouped {
		modules = append(modules, module)
	}
	sortStrings(modules)

	for _, module := range modules {
		names := grouped[module]
		fmt.Printf("# %s (%d)\n", module, len(names))
		for _, name := range names {
			spec := specs[name]
			effect := "pure"
			if !spec.IsPure {
				effect = strings.ToLower(spec.Effect)
			}
			fmt.Printf("  %-30s [%s]\n", name, effect)
		}
		fmt.Println()
	}
}

// sortStrings sorts a string slice in place
func sortStrings(s []string) {
	// Simple bubble sort for small slices
	n := len(s)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if s[j] > s[j+1] {
				s[j], s[j+1] = s[j+1], s[j]
			}
		}
	}
}

// runBuiltinsCheckMigration validates that all builtins have been migrated
func runBuiltinsCheckMigration() {
	// Get current working directory as project root
	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to get working directory: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Run migration validation
	report, err := builtins.ValidateMigration(projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: migration validation failed: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Display report
	fmt.Println(builtins.FormatReport(report))

	// Exit with error code if migration is incomplete
	if !report.IsClean {
		os.Exit(1)
	}
}
