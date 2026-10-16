// validate_manifest.go validates examples/manifest.json against reality.
// It ensures documentation stays in sync with the codebase:
//
//  1. Schema: the manifest loads under internal/manifest (valid statuses, no
//     duplicate paths, .ail extensions, consistent statistics).
//
//  2. Modules drift: every entry's committed `modules` field matches the actual
//     std/* imports discovered by the SHARED parser-backed extractor
//     (scripts/internal/importextract) — the same function the backfill uses, so
//     the CI authority cannot disagree with the language. A stale/missing
//     `modules` entry fails with the exact regeneration command.
//
//  3. Coverage: runnable files must be listed or explicitly grandfathered.
//
// It deliberately does NOT re-run each example (verify_examples.go owns actual
// program output). Files listed in the manifest but absent on disk are reported
// as warnings (pre-existing stale entries), not hard failures, so a legacy drift
// doesn't wedge CI — modules drift on a RESOLVABLE file is the hard gate.
//
// Enumerating ZERO modules is itself a hard failure, in both modes: a run that
// checked nothing is a broken instrument, not a clean manifest.
//
//	go run ./scripts/validate_manifest.go            # report
//	go run ./scripts/validate_manifest.go --ci       # non-zero on drift
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

const regenCmd = "make sync-examples-manifest"

// Baseline omissions measured at 482a9a4e (48; design measured 47).
// Disposition each as a documented example, helper, or removal; tracked in #1835.
var unmanifestedRunnable = map[string]string{
	"runnable/ai_call_result.ail":                    "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/ai_image_generation.ail":               "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/ai_tool_loop.ail":                      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/batch_processing.ail":                  "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/cross_module_types.ail":      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/cross_module_types_lib.ail":  "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/ensures_violation_demo.ail":  "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/inbox_injection.ail":         "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/inbox_injection_v2.ail":      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/record_discovery_verify.ail": "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/contracts/recursive_verify.ail":        "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/crypto_hashing.ail":                    "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/demos/adt_pipeline.ail":                "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/demos/hello_io.ail":                    "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/directory_listing.ail":                 "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/directory_ops.ail":                     "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/json_array_extraction.ail":             "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/json_bool_encoding.ail":                "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/json_helpers.ail":                      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/jwt_decode.ail":                        "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/list_extremes.ail":                     "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/list_helpers.ail":                      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/list_safe_indexing.ail":                "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/module_let_helpers.ail":                "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/reserved_keyword_names.ail":            "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/secrets/gated_secret.ail":              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/secrets/leak_attempt.ail":              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/secrets/secret_demo.ail":               "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_mcp_app.ail":                 "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_mcp_file.ail":                "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_mcp_header_auth.ail":         "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_mcp_hints.ail":               "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_mcp_oauth.ail":               "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/serve_api_named_params.ail":            "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/set_operations.ail":                    "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/split_argument_order.ail":              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/stream_sse.ail":                        "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/stream_websocket.ail":                  "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/string_chars.ail":                      "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/string_interp_nested.ail":              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/string_interpolation.ail":              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/string_repeat.ail":                     "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/syntax_ai_forgiving.ail":               "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/uuid.ail":                              "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/xml_build.ail":                         "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/xml_escape.ail":                        "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/xml_lenient.ail":                       "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
	"runnable/xml_zip_roundtrip.ail":                 "pre-existing unmanifested at 482a9a4e; disposition pending #1835",
}

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
		fmt.Fprintf(os.Stderr, "Regenerate with: %s\n", regenCmd)
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

	// Match physical paths so legacy bare names and runnable/ names agree.
	listed := make(map[string]bool)
	for _, ex := range m.Examples {
		if full, ok := importextract.ResolvePath(*examplesDir, ex.Path); ok {
			abs, err := filepath.Abs(full)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			listed[filepath.ToSlash(filepath.Clean(abs))] = true
		}
	}
	unlisted := 0
	err = filepath.WalkDir(filepath.Join(*examplesDir, "runnable"), func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.IsDir() || filepath.Ext(path) != ".ail" || strings.HasPrefix(item.Name(), "_") {
			return nil
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(*examplesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !listed[filepath.ToSlash(filepath.Clean(abs))] && unmanifestedRunnable[rel] == "" {
			unlisted++
			fmt.Printf("unlisted runnable example: %s (add an authored manifest entry, then run %s)\n", rel, regenCmd)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "enumerate runnable examples: %v\n", err)
		os.Exit(1)
	}
	if unlisted > 0 && *ciMode {
		os.Exit(1)
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

	// Anti-vacuity floor: a sweep that enumerated nothing cannot testify that the
	// manifest is in sync. Unconditional on --ci — this is not a drift-tolerance
	// question, it is a broken instrument.
	if checked == 0 {
		fmt.Fprintf(os.Stderr, "\n%s validate_manifest enumerated 0 modules (%d entries in manifest, %d missing on disk)\n",
			red("INSTRUMENT FAILURE:"), len(m.Examples), missingCount)
		fmt.Fprintf(os.Stderr, "A zero-enumeration means the instrument is broken, not that the manifest is clean.\n")
		os.Exit(1)
	}

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
