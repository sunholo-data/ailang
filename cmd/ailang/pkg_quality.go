package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/check"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
)

// M-PKG-QUALITY-LADDER M3 — `ailang pkg quality [--json] [--strict] [--no-run] <dir>`.
//
// The publisher-side measurement runner. Everything `server`-sourced is
// measured exactly as the validator measures it (same check config, the same
// `verify --package` report, the same InterfaceHashV2); tests and _smoke.ail
// are executed HERE and only here, and travel to the validator as an attested
// block it banks but never gates on.

// pkgQualityCommand is the CLI arm.
func pkgQualityCommand(args []string) error {
	fs := flag.NewFlagSet("pkg quality", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "Emit the report as JSON (schema ailang.package-quality/v1)")
	strict := fs.Bool("strict", false, "Promote warn-level badges to gates (exit 2)")
	noRun := fs.Bool("no-run", false, "Skip executing tests and _smoke.ail (server-side view only)")
	timeout := fs.Duration("timeout", 5*time.Second, "Per-function Z3 timeout")
	help := fs.Bool("help", false, "Show help")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fmt.Println("Usage: ailang pkg quality [--json] [--strict] [--no-run] [<package-dir>]")
		fmt.Println()
		fmt.Println("Static evidence inventory for a package — the same checks the registry")
		fmt.Println("validator runs on upload, plus the tests and _smoke.ail the validator")
		fmt.Println("never executes (they are attested by you, not proved by the registry).")
		fmt.Println()
		fmt.Println("Sections: compile · contracts (Z3) · interface identity (v2) · effects ·")
		fmt.Println("          release · docs · style (pure-export ratio) · tests · smoke")
		fmt.Println("Exit: 0 clean · 2 one or more gates (PUBnnn) · 1 usage/infra error")
		fmt.Println()
		fmt.Println("  --json      machine output; identical `server` sections to the validator's")
		fmt.Println("  --strict    warn-level badges become gates")
		fmt.Println("  --no-run    do not execute tests/smoke")
		return nil
	}
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	manifest, inputs, err := measurePackageQuality(dir, qualityMeasureOptions{RunAttested: !*noRun, Z3Timeout: *timeout})
	if err != nil {
		return err
	}
	report := pkg.BuildQualityReport(manifest, pkg.ModePublisher, inputs, *strict)
	if *jsonOut {
		data, mErr := json.MarshalIndent(report, "", "  ")
		if mErr != nil {
			return mErr
		}
		fmt.Println(string(data))
	} else {
		printQualityHuman(report)
	}
	if report.HasGates() {
		os.Exit(2)
	}
	return nil
}

type qualityMeasureOptions struct {
	RunAttested bool
	Z3Timeout   time.Duration
}

// measurePackageQuality performs every measurement the report needs. Server
// measurements never execute package code; attested ones do, and only when
// RunAttested is set.
func measurePackageQuality(dir string, opts qualityMeasureOptions) (*pkg.PackageManifest, pkg.QualityInputs, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, pkg.QualityInputs{}, err
	}
	manifest, err := pkg.LoadManifest(absDir)
	if err != nil {
		return nil, pkg.QualityInputs{}, fmt.Errorf("cannot load ailang.toml in %s: %w", absDir, err)
	}
	var in pkg.QualityInputs

	// compile — the check --package path, same config.
	os.Setenv("AILANG_QUIET_WARNINGS", "1")
	sourceFiles, _, err := check.DiscoverPackageSources(absDir)
	if err != nil {
		return nil, in, fmt.Errorf("discover package sources: %w", err)
	}
	cr := check.CheckPackageFiles(absDir, manifest, sourceFiles, check.PackageOptions{Timeout: 60 * time.Second})
	in.CompileOK = cr.Failed == 0
	in.CompileFiles = cr.Passed + cr.Failed
	if len(cr.Errors) > 0 {
		in.CompileError = cr.Errors[0]
	}

	// contracts — the same `verify --package` report the validator decodes.
	if in.CompileOK {
		if !smt.Z3Available() {
			in.VerifyErr = "z3 not available"
		} else if report, vErr := verifyPackage(absDir, verifyPackageOptions{Timeout: opts.Z3Timeout, RecursiveDepth: 2, WallCap: 120 * time.Second}); vErr != nil {
			in.VerifyErr = vErr.Error()
		} else {
			in.Verify = report
		}
	}

	// interface identity
	in.InterfaceHashV1 = pkg.InterfaceHash(manifest)
	if in.CompileOK {
		hash, sigs, v2Err := pkg.InterfaceHashV2(context.Background(), absDir, manifest, pkg.DefaultPublishLimits())
		if v2Err != nil {
			in.InterfaceV2Err = v2Err.Error()
		} else {
			in.InterfaceHashV2, in.Signatures = hash, sigs
		}
	}

	if st, sErr := os.Stat(filepath.Join(absDir, "AGENT.md")); sErr == nil && !st.IsDir() {
		in.HasAgentDoc = true
	}
	in.ChangelogNotes, in.HasChangelogSection = pkg.ChangelogSection(absDir, manifest.Package.Version)
	in.ReleaseGatesHard = pkg.ReleaseGatesHard(Version)

	if opts.RunAttested {
		in.Attested = runAttestedChecks(absDir, manifest)
	} else {
		// --no-run: still DISCOVER tests and the smoke file so the report says
		// "not run" rather than "none" — a badge that lies is worse than none.
		in.Attested = discoverAttestedChecks(absDir)
	}
	return manifest, in, nil
}

// discoverAttestedChecks counts test files and the smoke file without
// executing anything (the --no-run / server-side view).
func discoverAttestedChecks(absDir string) *pkg.AttestedBlock {
	testFiles, _ := filepath.Glob(filepath.Join(absDir, "*_test.ail"))
	tests := &pkg.TestsSection{Files: len(testFiles), Notes: "not run (--no-run)"}
	smoke := &pkg.SmokeSection{}
	if st, err := os.Stat(filepath.Join(absDir, pkg.SmokeFile)); err == nil && !st.IsDir() {
		smoke.Present = true
	}
	return &pkg.AttestedBlock{Tests: tests, Smoke: smoke}
}

// runAttestedChecks executes the package's tests and _smoke.ail on THIS
// machine. AttestedBy is left empty here: the validator stamps the API-key
// owner, which the client cannot know.
func runAttestedChecks(absDir string, manifest *pkg.PackageManifest) *pkg.AttestedBlock {
	block := &pkg.AttestedBlock{}
	bin, err := os.Executable()
	if err != nil {
		return block
	}

	// tests — `ailang test --package --format json` (files ∪ inline blocks once
	// m-package-test-discovery lands; today *_test.ail).
	tests := &pkg.TestsSection{}
	testFiles, _ := filepath.Glob(filepath.Join(absDir, "*_test.ail"))
	tests.Files = len(testFiles)
	if tests.Files > 0 {
		cmd := exec.Command(bin, "test", "--package", "--format", "json", ".")
		cmd.Dir = absDir
		cmd.Env = append(os.Environ(), "AILANG_QUIET_WARNINGS=1")
		out, _ := cmd.Output() // exit 1 on failures still prints the report
		var summary struct {
			Passed  int `json:"passed_tests"`
			Failed  int `json:"failed_tests"`
			Skipped int `json:"skipped_tests"`
		}
		if jErr := json.Unmarshal(out, &summary); jErr != nil {
			tests.Notes = "test run produced no report: " + strings.TrimSpace(string(out))
		} else {
			tests.Passed, tests.Failed, tests.Skipped = summary.Passed, summary.Failed, summary.Skipped
		}
	} else {
		tests.Notes = "no *_test.ail files; inline test blocks are not yet discovered in package mode (m-package-test-discovery)"
	}
	block.Tests = tests

	// smoke — the existing publish gate, reported rather than only enforced.
	smoke := &pkg.SmokeSection{}
	if st, sErr := os.Stat(filepath.Join(absDir, pkg.SmokeFile)); sErr == nil && !st.IsDir() {
		smoke.Present = true
		timeout := pkg.DefaultSmokeTimeout
		if manifest.Smoke.TimeoutSeconds > 0 {
			timeout = time.Duration(manifest.Smoke.TimeoutSeconds) * time.Second
		}
		if res, rErr := pkg.RunSmokeInTempDir(absDir, bin, timeout); rErr == nil {
			smoke.Passed = res.Passed
			smoke.Seconds = res.Duration.Seconds()
		}
	}
	block.Smoke = smoke
	return block
}

func printQualityHuman(r *pkg.QualityReport) {
	tick := func(ok bool) string {
		if ok {
			return green("✓")
		}
		return red("✗")
	}
	fmt.Printf("%s quality %s@%s (%s)\n", cyan("→"), r.Package, r.Version, orDash(r.Stability))
	fmt.Printf("  compile:   %s %d files\n", tick(r.Compile.OK), r.Compile.Files)
	if r.Contracts.Error != "" {
		fmt.Printf("  contracts: %s not run — %s\n", yellow("·"), r.Contracts.Error)
	} else {
		fmt.Printf("  contracts: %s %d/%d verified, %d refuted, %d uncontracted exports (%.1fs)\n", tick(r.Contracts.Counterexample == 0), r.Contracts.Verified, r.Contracts.Total, r.Contracts.Counterexample, r.Contracts.UncontractedExports, r.Contracts.WallSeconds)
	}
	if r.Interface.Error != "" {
		fmt.Printf("  interface: %s v2 not built — %s\n", yellow("⚠"), r.Interface.Error)
	} else {
		fmt.Printf("  interface: %s v2 %s (%d signatures)\n", green("✓"), shortHash(r.Interface.HashV2), r.Interface.Signatures)
	}
	fmt.Printf("  effects:   max %v  rank_max %s  ceiling_declared %v\n", r.Effects.Max, orDash(r.Effects.RankMax), r.Effects.CeilingDeclared)
	fmt.Printf("  release:   kind %s  changelog_section %v\n", orDash(r.Release.Kind), r.Release.ChangelogSection)
	fmt.Printf("  docs:      AGENT.md %v  ai_summary %v\n", r.Docs.AgentMD, r.Docs.AISummary)
	fmt.Printf("  style:     %d exported funcs, %d pure (%.0f%%)\n", r.Style.ExportedFuncs, r.Style.PureExports, r.Style.PureRatio*100)
	if r.Tests != nil {
		fmt.Printf("  tests:     [attested] %d files, %d passed, %d failed%s\n", r.Tests.Files, r.Tests.Passed, r.Tests.Failed, noteSuffix(r.Tests.Notes))
	}
	if r.Smoke != nil {
		fmt.Printf("  smoke:     [attested] present %v passed %v\n", r.Smoke.Present, r.Smoke.Passed)
	}
	fmt.Println()
	for _, g := range r.Gates {
		fmt.Printf("  %s %s %s\n", red("✗"), g.Code, g.Msg)
	}
	for _, b := range r.Badges {
		mark := yellow("·")
		if b.Level == "warn" {
			mark = yellow("⚠")
		}
		fmt.Printf("  %s %s %s\n", mark, b.Code, b.Msg)
	}
	if r.HasGates() {
		fmt.Printf("\n%s %d gate(s) — publish would be refused\n", red("✗"), len(r.Gates))
	} else {
		fmt.Printf("\n%s no gates\n", green("✓"))
	}
}

func noteSuffix(s string) string {
	if s == "" {
		return ""
	}
	return " — " + s
}

func shortHash(h string) string {
	h = strings.TrimPrefix(h, "sha256:ifacev2:")
	if len(h) > 12 {
		return h[:12] + "…"
	}
	return h
}
