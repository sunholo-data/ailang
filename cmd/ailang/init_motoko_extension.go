package main

// M-EXT-SCAFFOLD-AI-FIRST (v0.18.5): scaffold a working motoko_agent extension
// package in one CLI command. The generated package mirrors the canonical
// shape of motoko-ext-test-dummy (see ailang-packages repo): single
// register.ail with all 8 ExtensionHooks fields populated as no-op defaults,
// plus a placeholder types.ail that the user fills in. Output passes
// `ailang lock + ailang check` with zero edits.
//
// Audience: AI agents (and humans) authoring a new extension. Today, doing
// this by hand requires reading ~840 lines of docs + a 100-LOC reference
// impl. This scaffolder collapses that to ~500 tokens of generated stubs.

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

// motokoExtensionFlags holds parsed CLI flags for `ailang init motoko-extension`.
type motokoExtensionFlags struct {
	name      string   // e.g. "arniwesth/motoko_ext_openkb"
	tools     []string // e.g. ["OpenKBSearch", "OpenKBList"]
	effects   []string // e.g. ["FS", "Process"]
	outputDir string   // override of derived dir; empty = use derived
	help      bool
}

// canonicalEffects is the set of effect names the host can declare in
// [effects].max. Validated against this list to give the user a clear
// error for typos. Source: internal/effects (canonical effect list).
var canonicalEffects = map[string]bool{
	"IO":          true,
	"FS":          true,
	"Net":         true,
	"Env":         true,
	"Process":     true,
	"Clock":       true,
	"AI":          true,
	"SharedMem":   true,
	"Stream":      true,
	"SharedIndex": true,
	"Rand":        true,
}

// initMotokoExtensionCommand is the entry point dispatched by initCommand
// when kind == "motoko-extension". Parses flags, validates, scaffolds.
func initMotokoExtensionCommand(args []string) error {
	mef, err := parseInitMotokoExtensionFlags(args)
	if err != nil {
		return err
	}
	if mef.help {
		printInitMotokoExtensionHelp()
		return nil
	}

	// Derive short_name (e.g. "motoko_ext_openkb" → "openkb") and output dir
	// (e.g. "arniwesth/motoko_ext_openkb" → "packages/motoko-ext-openkb").
	shortName := deriveShortName(mef.name)
	outDir := mef.outputDir
	if outDir == "" {
		outDir = deriveOutputDir(mef.name)
	}

	// Refuse to overwrite an existing dir — matches initWebApp behavior.
	if _, statErr := os.Stat(outDir); statErr == nil {
		return fmt.Errorf("directory %q already exists; pass --output-dir to scaffold elsewhere", outDir)
	}

	if err := scaffoldMotokoExtension(outDir, mef.name, shortName, mef.tools, mef.effects); err != nil {
		return err
	}

	// Friendly post-scaffold pointer (mirrors initPackageCommand style).
	fmt.Printf("\n%s Scaffolded %s/\n", green("✓"), outDir)
	fmt.Println("  ├── ailang.toml          (registry deps + exports + effects)")
	fmt.Println("  ├── register.ail         (canonical register_with_config)")
	fmt.Println("  ├── types.ail            (placeholder — edit me)")
	fmt.Printf("  ├── %s.ail%s(8-hook ExtensionHooks, all no-op)\n", shortName, padTo(len(shortName)+4, 20))
	fmt.Println("  └── README.md            (links to tutorial + publishing guide)")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", outDir)
	fmt.Printf("  $EDITOR %s.ail        # implement your tool's logic\n", shortName)
	fmt.Println("  ailang lock && ailang check register.ail")
	return nil
}

// parseInitMotokoExtensionFlags parses CLI flags for the motoko-extension
// init type. Returns motokoExtensionFlags or a wrapped error with an
// actionable hint.
func parseInitMotokoExtensionFlags(args []string) (*motokoExtensionFlags, error) {
	fs := flag.NewFlagSet("init motoko-extension", flag.ContinueOnError)
	mef := &motokoExtensionFlags{}
	var toolsCSV, effectsCSV string

	fs.StringVar(&mef.name, "name", "", "Package name in <namespace>/<motoko_ext_xxx> form (REQUIRED)")
	fs.StringVar(&toolsCSV, "tools", "", "Comma-separated list of tool names this extension provides (e.g. \"OpenKBSearch,OpenKBList\")")
	fs.StringVar(&effectsCSV, "effects", "", "Comma-separated list of effect names this extension uses (e.g. \"FS,Process\")")
	fs.StringVar(&mef.outputDir, "output-dir", "", "Override output directory (default: packages/motoko-ext-<short>)")
	fs.BoolVar(&mef.help, "help", false, "Show help")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if mef.help {
		return mef, nil
	}

	if err := validatePackageName(mef.name); err != nil {
		return nil, err
	}

	mef.tools = splitCSV(toolsCSV)
	mef.effects = splitCSV(effectsCSV)

	if err := validateEffects(mef.effects); err != nil {
		return nil, err
	}

	// Auto-include Env + FS + IO in the effect ceiling.
	//   - Env + FS: required by register_with_config's effect annotation
	//     (`! {Env, FS}`) — matches canonical motoko-ext-* shape.
	//   - IO: required by _smoke.ail's println-based assertion logging.
	//     Without IO in [effects].max the publish sandbox rejects the
	//     smoke at type-check time. M-EXT-AUTHOR-DX M2 (v0.20.1).
	// User-supplied --effects extend this baseline, never replace it.
	mef.effects = ensureEffects(mef.effects, "Env", "FS", "IO")

	return mef, nil
}

// ensureEffects returns effects with each `required` entry present (idempotent).
// Preserves user ordering and only appends missing entries.
func ensureEffects(effects []string, required ...string) []string {
	have := make(map[string]bool, len(effects))
	for _, e := range effects {
		have[e] = true
	}
	out := append([]string(nil), effects...)
	for _, r := range required {
		if !have[r] {
			out = append(out, r)
		}
	}
	return out
}

// validatePackageName enforces the <namespace>/<motoko_ext_xxx> shape so the
// registry generator's short-name derivation works correctly. Common Arni-#8
// failure: name "sunholo/motoko_openkb" (missing _ext_) → ugly registry key.
func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("--name is required (e.g. --name arniwesth/motoko_ext_openkb)")
	}
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("--name must be in <namespace>/<package> form (got %q)", name)
	}
	if !strings.HasPrefix(parts[1], "motoko_ext_") {
		return fmt.Errorf("--name package portion must start with %q so the registry generator strips it cleanly (got %q). Example: %s/motoko_ext_yourthing",
			"motoko_ext_", parts[1], parts[0])
	}
	if parts[1] == "motoko_ext_" {
		return fmt.Errorf("--name package portion must have a name after the motoko_ext_ prefix (got %q)", parts[1])
	}
	return nil
}

// validateEffects checks each effect is in the canonical set. Empty list is
// allowed (a no-effect extension is valid).
func validateEffects(effects []string) error {
	for _, e := range effects {
		if !canonicalEffects[e] {
			validList := make([]string, 0, len(canonicalEffects))
			for k := range canonicalEffects {
				validList = append(validList, k)
			}
			return fmt.Errorf("--effects: %q is not a canonical effect name. Valid effects: %s",
				e, strings.Join(validList, ", "))
		}
	}
	return nil
}

// deriveOutputDir maps a package name to the conventional ailang-packages
// directory layout. Example: "arniwesth/motoko_ext_openkb" →
// "packages/motoko-ext-openkb". The directory name uses HYPHENS, while
// the package name field uses UNDERSCORES — mirrors the existing
// ailang-packages repo convention exactly.
func deriveOutputDir(packageName string) string {
	parts := strings.Split(packageName, "/")
	pkg := parts[len(parts)-1]
	// motoko_ext_openkb → motoko-ext-openkb
	dirName := strings.ReplaceAll(pkg, "_", "-")
	return path.Join("packages", dirName)
}

// splitCSV trims whitespace, drops empty entries. Returns nil for empty input
// (vs an empty slice) so callers can treat unset and empty identically.
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// padTo right-pads a label-display string with spaces so the next column
// aligns. Used only for the post-scaffold tree-print output.
func padTo(currentLen, target int) string {
	if currentLen >= target {
		return " "
	}
	return strings.Repeat(" ", target-currentLen)
}

// scaffoldMotokoExtension lives in init_motoko_extension_templates.go.
// The split keeps the pure CLI parsing logic separate from template
// rendering; M2 replaces the stub there with the real implementation.

func printInitMotokoExtensionHelp() {
	fmt.Println("Usage: ailang init motoko-extension [flags]")
	fmt.Println()
	fmt.Println("Scaffold a new motoko_agent extension as a reusable AILANG package.")
	fmt.Println("The generated package passes `ailang lock + ailang check` with zero edits.")
	fmt.Println()
	fmt.Println("Required flags:")
	fmt.Println("  --name <ns>/<pkg>      Package name. Pkg must start with 'motoko_ext_'.")
	fmt.Println("                         Example: --name arniwesth/motoko_ext_openkb")
	fmt.Println()
	fmt.Println("Optional flags:")
	fmt.Println("  --tools <csv>          Comma-separated tool names this extension provides")
	fmt.Println("                         Example: --tools \"OpenKBSearch,OpenKBList\"")
	fmt.Println("  --effects <csv>        Comma-separated effect names")
	fmt.Println("                         Example: --effects \"FS,Process,Env\"")
	fmt.Println("                         Valid: IO, FS, Net, Env, Process, Clock, AI,")
	fmt.Println("                                SharedMem, Stream, SharedIndex, Rand")
	fmt.Println("  --output-dir <path>    Override output dir (default: packages/motoko-ext-<short>)")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("  ailang init motoko-extension \\")
	fmt.Println("    --name arniwesth/motoko_ext_openkb \\")
	fmt.Println("    --tools \"OpenKBSearch\" \\")
	fmt.Println("    --effects \"FS,Process\"")
	fmt.Println()
	fmt.Println("See: https://ailang.sunholo.com/docs/guides/build-a-motoko-extension")
}
