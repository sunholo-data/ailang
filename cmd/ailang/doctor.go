package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/executor"
	_ "github.com/sunholo-data/ailang/internal/executor/managed_agents"
)

// runDoctor implements `ailang doctor <subcommand>` commands.
func runDoctor() {
	subcommand := ""
	if flag.NArg() >= 2 {
		subcommand = flag.Arg(1)
	}

	switch subcommand {
	case "builtins":
		// fall through to existing builtins-validation logic below
	case "managed_agents", "managed-agents":
		runDoctorManagedAgents()
		return
	default:
		fmt.Println("Usage: ailang doctor <subcommand>")
		fmt.Println()
		fmt.Println("Available subcommands:")
		fmt.Println("  builtins         Validate the builtin function registry")
		fmt.Println("  managed_agents   Check ADC for the Vertex AI Managed Agents API")
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
	jsonOut := listFlags.Bool("json", false, "Emit machine-readable JSON list")
	_ = listFlags.Parse(flag.Args()[2:]) // ExitOnError means this never returns an error we can handle

	// Get all specs from new registry (M-DX1 complete in v0.3.10)
	specs := builtins.AllSpecs()

	if len(specs) == 0 {
		if *jsonOut {
			fmt.Println(`{"count":0,"builtins":[]}`)
			return
		}
		fmt.Println("No builtins registered")
		return
	}

	if *jsonOut {
		emitBuiltinsListJSON(specs)
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
		fmt.Println("Usage: ailang builtins show <name> [--json]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang builtins show _rand_int")
		fmt.Println("  ailang builtins show _net_httpRequest")
		fmt.Println("  ailang builtins show concat_String --json")
		os.Exit(1)
	}

	// Scan args 2..N for the name (first non-flag) and --json toggle.
	name := ""
	jsonOut := false
	for i := 2; i < flag.NArg(); i++ {
		a := flag.Arg(i)
		switch a {
		case "--json", "-json":
			jsonOut = true
		default:
			if name == "" && !strings.HasPrefix(a, "-") {
				name = a
			}
		}
	}
	if name == "" {
		fmt.Println("Usage: ailang builtins show <name> [--json]")
		os.Exit(1)
	}
	specs := builtins.AllSpecs()

	spec, ok := specs[name]
	if !ok {
		// Also try the leading-underscore variant (so callers can pass the
		// public wrapper name like "concat_String" and still get a hit).
		if alt, ok2 := specs["_"+name]; ok2 {
			spec, ok = alt, true
		}
	}
	if !ok {
		// Try to find partial matches
		var matches []string
		for n := range specs {
			if strings.Contains(strings.ToLower(n), strings.ToLower(name)) {
				matches = append(matches, n)
			}
		}
		sortStrings(matches)

		if jsonOut {
			emitBuiltinShowError(name, matches)
			os.Exit(1)
		}
		fmt.Printf("Builtin '%s' not found.\n", name)
		if len(matches) > 0 {
			fmt.Println("\nDid you mean:")
			for _, m := range matches {
				fmt.Printf("  %s\n", m)
			}
		}
		os.Exit(1)
	}

	if jsonOut {
		emitBuiltinShowJSON(spec)
		return
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

// emitBuiltinsListJSON prints all registered builtins as a stable JSON list.
// Consumed by `tools/index_ailang_syntax.sh` to populate the ailang-builtins
// brain namespace.
func emitBuiltinsListJSON(specs map[string]*builtins.BuiltinSpec) {
	type item struct {
		Name        string `json:"name"`
		Module      string `json:"module"`
		Signature   string `json:"signature"`
		IsPure      bool   `json:"is_pure"`
		Effect      string `json:"effect,omitempty"`
		NumArgs     int    `json:"num_args"`
		Description string `json:"description,omitempty"`
	}
	names := make([]string, 0, len(specs))
	for n := range specs {
		names = append(names, n)
	}
	sortStrings(names)
	out := struct {
		Count    int    `json:"count"`
		Builtins []item `json:"builtins"`
	}{Count: len(names)}
	for _, n := range names {
		s := specs[n]
		it := item{
			Name:      s.Name,
			Module:    s.Module,
			Signature: formatBuiltinSignature(s),
			IsPure:    s.IsPure,
			Effect:    s.Effect,
			NumArgs:   s.NumArgs,
		}
		if s.Metadata != nil {
			it.Description = s.Metadata.Description
		}
		out.Builtins = append(out.Builtins, it)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
}

// emitBuiltinShowJSON prints a stable JSON envelope for a builtin spec.
// Consumed by `ailang micro-rag lint-builtin` for first-use nudges.
func emitBuiltinShowJSON(spec *builtins.BuiltinSpec) {
	type paramJSON struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}
	type exampleJSON struct {
		Code        string `json:"code"`
		Description string `json:"description,omitempty"`
	}
	type metaJSON struct {
		Description string        `json:"description,omitempty"`
		LongDesc    string        `json:"long_desc,omitempty"`
		Params      []paramJSON   `json:"params,omitempty"`
		Returns     string        `json:"returns,omitempty"`
		Examples    []exampleJSON `json:"examples,omitempty"`
		SeeAlso     []string      `json:"see_also,omitempty"`
		Tags        []string      `json:"tags,omitempty"`
		Since       string        `json:"since,omitempty"`
		Stability   string        `json:"stability,omitempty"`
	}
	out := struct {
		Name      string    `json:"name"`
		Module    string    `json:"module"`
		Signature string    `json:"signature"`
		IsPure    bool      `json:"is_pure"`
		Effect    string    `json:"effect,omitempty"`
		NumArgs   int       `json:"num_args"`
		Metadata  *metaJSON `json:"metadata,omitempty"`
	}{
		Name:      spec.Name,
		Module:    spec.Module,
		Signature: formatBuiltinSignature(spec),
		IsPure:    spec.IsPure,
		Effect:    spec.Effect,
		NumArgs:   spec.NumArgs,
	}
	if spec.Metadata != nil {
		m := &metaJSON{
			Description: spec.Metadata.Description,
			LongDesc:    spec.Metadata.LongDesc,
			Returns:     spec.Metadata.Returns,
			SeeAlso:     spec.Metadata.SeeAlso,
			Tags:        spec.Metadata.Tags,
			Since:       spec.Metadata.Since,
			Stability:   spec.Metadata.GetStabilityString(),
		}
		for _, p := range spec.Metadata.Params {
			m.Params = append(m.Params, paramJSON{Name: p.Name, Description: p.Description})
		}
		for _, ex := range spec.Metadata.Examples {
			m.Examples = append(m.Examples, exampleJSON{Code: ex.Code, Description: ex.Description})
		}
		out.Metadata = m
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
}

// emitBuiltinShowError prints a JSON error envelope for builtin lookup failures.
func emitBuiltinShowError(name string, suggestions []string) {
	out := struct {
		Error       string   `json:"error"`
		Name        string   `json:"name"`
		Suggestions []string `json:"suggestions,omitempty"`
	}{
		Error:       "not_found",
		Name:        name,
		Suggestions: suggestions,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
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

// runDoctorManagedAgents verifies the Vertex AI Managed Agents API is
// reachable via Application Default Credentials. It does NOT make a real
// interaction call (that would cost money and provision a sandbox) — it
// asks the executor's HealthCheck to validate the ADC token source.
//
// Output is plain text by default, JSON when --format=json is passed.
func runDoctorManagedAgents() {
	jsonOutput := false
	for _, arg := range flag.Args()[2:] {
		if arg == "--format=json" || arg == "-format=json" {
			jsonOutput = true
		}
	}

	type doctorResult struct {
		OK          bool   `json:"ok"`
		Subcommand  string `json:"subcommand"`
		Executor    string `json:"executor"`
		Detail      string `json:"detail"`
		ErrorString string `json:"error,omitempty"`
	}

	emit := func(r doctorResult) {
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			_ = enc.Encode(r)
			return
		}
		if r.OK {
			fmt.Printf("✅ %s: %s\n", r.Executor, r.Detail)
		} else {
			fmt.Printf("❌ %s: %s\n", r.Executor, r.Detail)
			if r.ErrorString != "" {
				fmt.Printf("   %s\n", r.ErrorString)
			}
			fmt.Println()
			fmt.Println("Likely fixes:")
			fmt.Println("  1. Run: gcloud auth application-default login")
			fmt.Println("  2. Verify: gcloud auth application-default print-access-token")
			fmt.Println("  3. Ensure GOOGLE_CLOUD_PROJECT is set (or use a models.yml entry with gcp_project)")
		}
	}

	factory := executor.GlobalFactory()
	exec, err := factory.GetExecutor("managed_agents")
	if err != nil {
		emit(doctorResult{
			OK:          false,
			Subcommand:  "managed_agents",
			Executor:    "managed_agents",
			Detail:      "executor not registered in factory",
			ErrorString: err.Error(),
		})
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := exec.HealthCheck(ctx); err != nil {
		emit(doctorResult{
			OK:          false,
			Subcommand:  "managed_agents",
			Executor:    exec.Name(),
			Detail:      "ADC health check failed",
			ErrorString: err.Error(),
		})
		os.Exit(1)
	}

	emit(doctorResult{
		OK:         true,
		Subcommand: "managed_agents",
		Executor:   exec.Name(),
		Detail:     "ADC token acquired; ready for Vertex Managed Agents API calls",
	})
}
