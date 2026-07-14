// validate_manifest.go validates examples/manifest.json against reality.
// It ensures documentation stays in sync with the codebase:
//
//  1. Schema: the manifest loads under internal/manifest (valid statuses, no
//     duplicate paths, .ail extensions, consistent statistics).
//  2. Modules drift: every entry's committed `modules` field matches the actual
//     std/* imports discovered by the SHARED parser-backed extractor
//     (scripts/internal/importextract) — the same function the backfill uses, so
//     the CI authority cannot disagree with the language. A stale/missing
//     `modules` entry fails with the exact regeneration command.
//
// It deliberately does NOT re-run each example (verify_examples.go owns actual
// program output). Files listed in the manifest but absent on disk are reported
// as warnings (pre-existing stale entries), not hard failures, so a legacy drift
// doesn't wedge CI — modules drift on a RESOLVABLE file is the hard gate.
//
//	go run ./scripts/validate_manifest.go            # report
//	go run ./scripts/validate_manifest.go --ci       # non-zero on drift
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sunholo-data/ailang/internal/manifest"
	"github.com/sunholo-data/ailang/scripts/internal/importextract"
)

var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
)

const regenCmd = "go run ./scripts/backfill_manifest_modules.go"

func main() {
	var (
		manifestPath = flag.String("manifest", "examples/manifest.json", "Path to manifest file")
		examplesDir  = flag.String("dir", "examples", "Examples directory")
		ciMode       = flag.Bool("ci", false, "CI mode (non-zero exit on any drift)")
		verbose      = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Deterministic environment for any downstream tooling.
	os.Setenv("LC_ALL", "C.UTF-8")
	os.Setenv("TZ", "UTC")
	os.Setenv("AILANG_SEED", "0")

	// (1) Schema: loading validates statuses, dup paths, extensions, statistics.
	m, err := manifest.Load(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s manifest failed schema validation: %v\n", red("✗"), err)
		os.Exit(1)
	}

	fmt.Printf("%s AILANG Manifest Validator\n", bold("🔍"))
	fmt.Printf("Manifest: %s\n", *manifestPath)
	fmt.Printf("Examples: %d (working %d, broken %d, experimental %d)\n\n",
		m.Statistics.Total, m.Statistics.Working, m.Statistics.Broken, m.Statistics.Experimental)

	// (2) Modules drift, via the shared parser-backed extractor.
	driftCount := 0
	missingCount := 0
	checked := 0
	for _, ex := range m.Examples {
		full, ok := importextract.ResolvePath(*examplesDir, ex.Path)
		if !ok {
			missingCount++
			fmt.Printf("%s %s: file not found on disk (stale manifest entry)\n", yellow("⚠"), ex.Path)
			continue
		}
		actual, err := importextract.ExtractModules(full)
		if err != nil {
			// Unparseable — already red via the verify gate; do not double-fail.
			fmt.Printf("%s %s: unparseable (covered by verify-examples): %v\n", yellow("⚠"), ex.Path, err)
			continue
		}
		checked++
		if !importextract.Equal(ex.Modules, actual) {
			driftCount++
			fmt.Printf("%s %s: modules drift\n    manifest: %v\n    actual:   %v\n",
				red("✗"), ex.Path, ex.Modules, actual)
		} else if *verbose {
			fmt.Printf("%s %s %v\n", green("✓"), ex.Path, actual)
		}
	}

	fmt.Printf("\n%s modules checked, %s drift, %s missing-on-disk\n",
		green(fmt.Sprintf("%d", checked)),
		func() string {
			if driftCount > 0 {
				return red(fmt.Sprintf("%d", driftCount))
			}
			return green("0")
		}(),
		yellow(fmt.Sprintf("%d", missingCount)))

	if driftCount > 0 {
		fmt.Fprintf(os.Stderr, "\n%s %d manifest `modules` entr(ies) are out of date.\n", red("DRIFT:"), driftCount)
		fmt.Fprintf(os.Stderr, "Regenerate with:\n    %s\n", regenCmd)
		if *ciMode {
			os.Exit(1)
		}
	} else {
		fmt.Printf("%s manifest `modules` field is in sync with actual imports\n", green("✓"))
	}
}
