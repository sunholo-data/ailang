package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/builtins"
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
	case "show":
		runBuiltinsShow()
	case "check-migration":
		runBuiltinsCheckMigration()
	default:
		fmt.Println("Usage: ailang builtins <subcommand>")
		fmt.Println()
		fmt.Println("Available subcommands:")
		fmt.Println("  list              List all registered builtins")
		fmt.Println("  show <name>       Show detailed documentation for a builtin")
		fmt.Println("  check-migration   Validate that all builtins have been migrated")
		os.Exit(1)
	}
}

func runBuiltinsList() {

	// Parse additional flags for list command
	listFlags := flag.NewFlagSet("list", flag.ExitOnError)
	byEffect := listFlags.Bool("by-effect", false, "Group by effect type")
	byModule := listFlags.Bool("by-module", false, "Group by module")
	verbose := listFlags.Bool("verbose", false, "Show full documentation including signatures")
	_ = listFlags.Parse(flag.Args()[2:]) // ExitOnError means this never returns an error we can handle

	// Get all specs from new registry (M-DX1 complete in v0.3.10)
	specs := builtins.AllSpecs()

	if len(specs) == 0 {
		fmt.Println("No builtins registered")
		return
	}

	// Choose display mode
	if *verbose {
		if *byModule {
			listBuiltinsVerboseByModule(specs)
		} else if *byEffect {
			listBuiltinsVerboseByEffect(specs)
		} else {
			listBuiltinsVerbose(specs)
		}
	} else if *byEffect {
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

// formatBuiltinSignature formats a builtin's type signature for display
func formatBuiltinSignature(spec *builtins.BuiltinSpec) string {
	// Get the type from the spec
	t := spec.Type()
	if t == nil {
		return spec.Name + ": <unknown type>"
	}
	return spec.Name + ": " + t.String()
}

// printBuiltinVerbose prints detailed documentation for a single builtin
func printBuiltinVerbose(spec *builtins.BuiltinSpec, indent string) {
	// Print signature
	fmt.Printf("%s%s\n", indent, formatBuiltinSignature(spec))

	// Show public wrapper name if this is an internal builtin
	if strings.HasPrefix(spec.Name, "_") && spec.Module != "" && spec.Module != "$builtin" {
		// Convert _rand_int -> rand_int for the public name
		publicName := strings.TrimPrefix(spec.Name, "_")
		fmt.Printf("%s  Usage: import %s (%s)\n", indent, spec.Module, publicName)
	}

	// Print description if available
	if spec.Metadata != nil && spec.Metadata.Description != "" {
		fmt.Printf("%s  %s\n", indent, spec.Metadata.Description)
	}

	// Print parameters if available
	if spec.Metadata != nil && len(spec.Metadata.Params) > 0 {
		fmt.Printf("%s  Parameters:\n", indent)
		for _, param := range spec.Metadata.Params {
			fmt.Printf("%s    %s: %s\n", indent, param.Name, param.Description)
		}
	}

	// Print return description if available
	if spec.Metadata != nil && spec.Metadata.Returns != "" {
		fmt.Printf("%s  Returns: %s\n", indent, spec.Metadata.Returns)
	}

	// Print examples if available
	if spec.Metadata != nil && len(spec.Metadata.Examples) > 0 {
		fmt.Printf("%s  Examples:\n", indent)
		for _, ex := range spec.Metadata.Examples {
			fmt.Printf("%s    %s", indent, ex.Code)
			if ex.Description != "" {
				fmt.Printf("  -- %s", ex.Description)
			}
			fmt.Println()
		}
	}

	fmt.Println()
}

// listBuiltinsVerbose lists all builtins with full documentation
func listBuiltinsVerbose(specs map[string]*builtins.BuiltinSpec) {
	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sortStrings(names)

	fmt.Printf("Total: %d builtins\n\n", len(specs))

	for _, name := range names {
		printBuiltinVerbose(specs[name], "")
	}
}

// listBuiltinsVerboseByModule lists builtins grouped by module with full documentation
func listBuiltinsVerboseByModule(specs map[string]*builtins.BuiltinSpec) {
	grouped := builtins.GroupByModule()

	modules := make([]string, 0, len(grouped))
	for module := range grouped {
		modules = append(modules, module)
	}
	sortStrings(modules)

	for _, module := range modules {
		names := grouped[module]
		fmt.Printf("# %s (%d)\n\n", module, len(names))
		for _, name := range names {
			printBuiltinVerbose(specs[name], "  ")
		}
	}
}

// listBuiltinsVerboseByEffect lists builtins grouped by effect with full documentation
func listBuiltinsVerboseByEffect(specs map[string]*builtins.BuiltinSpec) {
	grouped := builtins.GroupByEffect()

	effects := make([]string, 0, len(grouped))
	for effect := range grouped {
		effects = append(effects, effect)
	}
	sortStrings(effects)

	for _, effect := range effects {
		names := grouped[effect]
		fmt.Printf("# %s (%d)\n\n", effect, len(names))
		for _, name := range names {
			printBuiltinVerbose(specs[name], "  ")
		}
	}
}

// runBuiltinsShow shows detailed documentation for a specific builtin
func runBuiltinsShow() {
	if flag.NArg() < 3 {
		fmt.Println("Usage: ailang builtins show <name>")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang builtins show _rand_int")
		fmt.Println("  ailang builtins show _net_httpRequest")
		os.Exit(1)
	}

	name := flag.Arg(2)
	specs := builtins.AllSpecs()

	spec, ok := specs[name]
	if !ok {
		// Try to find partial matches
		var matches []string
		for n := range specs {
			if strings.Contains(strings.ToLower(n), strings.ToLower(name)) {
				matches = append(matches, n)
			}
		}
		sortStrings(matches)

		fmt.Printf("Builtin '%s' not found.\n", name)
		if len(matches) > 0 {
			fmt.Println("\nDid you mean:")
			for _, m := range matches {
				fmt.Printf("  %s\n", m)
			}
		}
		os.Exit(1)
	}

	// Print full documentation
	fmt.Println()
	fmt.Println(formatBuiltinSignature(spec))

	// Show public wrapper name and import if this is an internal builtin
	if strings.HasPrefix(spec.Name, "_") && spec.Module != "" && spec.Module != "$builtin" {
		publicName := strings.TrimPrefix(spec.Name, "_")
		fmt.Printf("\nUsage:\n  import %s (%s)\n  %s(...)\n", spec.Module, publicName, publicName)
	}
	fmt.Println()

	if spec.Metadata != nil {
		if spec.Metadata.Description != "" {
			fmt.Println("Description:")
			fmt.Printf("  %s\n\n", spec.Metadata.Description)
		}

		if spec.Metadata.LongDesc != "" {
			fmt.Println("Details:")
			// Indent each line of LongDesc
			for _, line := range strings.Split(spec.Metadata.LongDesc, "\n") {
				fmt.Printf("  %s\n", line)
			}
			fmt.Println()
		}

		if len(spec.Metadata.Params) > 0 {
			fmt.Println("Parameters:")
			for _, param := range spec.Metadata.Params {
				fmt.Printf("  %-12s %s\n", param.Name+":", param.Description)
			}
			fmt.Println()
		}

		if spec.Metadata.Returns != "" {
			fmt.Println("Returns:")
			fmt.Printf("  %s\n\n", spec.Metadata.Returns)
		}

		if len(spec.Metadata.Examples) > 0 {
			fmt.Println("Examples:")
			for _, ex := range spec.Metadata.Examples {
				fmt.Printf("  %-30s %s\n", ex.Code, ex.Description)
			}
			fmt.Println()
		}

		if spec.Metadata.Since != "" {
			fmt.Printf("Since: %s\n", spec.Metadata.Since)
		}

		fmt.Printf("Stability: %s\n", spec.Metadata.GetStabilityString())

		if len(spec.Metadata.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(spec.Metadata.Tags, ", "))
		}

		if len(spec.Metadata.SeeAlso) > 0 {
			fmt.Printf("See also: %s\n", strings.Join(spec.Metadata.SeeAlso, ", "))
		}
	} else {
		fmt.Println("(No documentation available)")
		fmt.Println()
	}

	fmt.Printf("Module: %s\n", spec.Module)

	effect := "pure"
	if !spec.IsPure {
		effect = spec.Effect
	}
	fmt.Printf("Effect: %s\n", effect)
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
