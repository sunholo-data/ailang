package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/smt"
)

// M-PKG-QUALITY-LADDER M1 — `ailang verify --package <dir>`.
//
// Verifies every exported module of a package with the SAME compile
// configuration `check --package` uses (DryLink, MOD010 relaxed, the manifest
// validating module names) so a flat tarball whose settle.ail declares
// `module vendor/name/settle` verifies instead of dying on MOD010 — which is
// what the registry validator's per-file `ailang verify <abs file>` did for
// every package ever published.

// verifyPackageOptions carries the flags that apply per function/package.
type verifyPackageOptions struct {
	Timeout        time.Duration // per-function Z3 timeout
	RecursiveDepth int
	Verbose        bool
	WallCap        time.Duration // whole-package cap; zero = unbounded
}

// verifyPackage compiles and verifies each exported module in sorted order.
// Compile failures are recorded per module (Errors=1) rather than aborting,
// so one broken module cannot hide the others' proofs.
func verifyPackage(dir string, opts verifyPackageOptions) (*pkg.PackageVerifyReport, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	manifest, err := pkg.LoadManifest(absDir)
	if err != nil {
		return nil, fmt.Errorf("cannot load ailang.toml in %s: %w", absDir, err)
	}

	report := &pkg.PackageVerifyReport{
		Schema:  pkg.PackageVerifySchema,
		Package: manifest.Package.Name,
		Version: manifest.Package.Version,
		Modules: []pkg.ModuleVerifyReport{},
	}

	modules := append([]string(nil), manifest.Exports.Modules...)
	sort.Strings(modules)
	start := time.Now()
	for _, modulePath := range modules {
		if opts.WallCap > 0 && time.Since(start) > opts.WallCap {
			report.WallCapHit = true
			report.Add(pkg.ModuleVerifyReport{Module: modulePath, Skipped: 1,
				CompileError: fmt.Sprintf("skipped: package wall cap %s exceeded", opts.WallCap)})
			continue
		}
		report.Add(verifyOneModule(absDir, manifest, modulePath, opts))
	}
	report.WallSeconds = time.Since(start).Seconds()
	return report, nil
}

func verifyOneModule(absDir string, manifest *pkg.PackageManifest, modulePath string, opts verifyPackageOptions) pkg.ModuleVerifyReport {
	mr := pkg.ModuleVerifyReport{Module: modulePath}
	file := pkg.ResolveModuleToFile(absDir, manifest.Package.Name, modulePath)
	if file == "" {
		mr.Errors = 1
		mr.CompileError = "exported module has no source file"
		return mr
	}
	rel, _ := filepath.Rel(absDir, file)
	mr.File = filepath.ToSlash(rel)

	content, err := os.ReadFile(file)
	if err != nil {
		mr.Errors = 1
		mr.CompileError = err.Error()
		return mr
	}
	cfg := pipeline.Config{
		DryLink:      true,
		RelaxModules: true, // package mode: the manifest validates module names
		PackageDir:   absDir,
	}
	result, err := pipeline.Run(cfg, pipeline.Source{Code: string(content), Filename: file})
	if err != nil {
		mr.Errors = 1
		mr.CompileError = err.Error()
		return mr
	}
	if len(result.Errors) > 0 {
		mr.Errors = 1
		mr.CompileError = fmt.Sprintf("%v", result.Errors[0])
		return mr
	}
	if result.Artifacts.Core == nil || result.Artifacts.Core.Meta == nil {
		return mr
	}
	vr := smt.Verify(result.Artifacts.Core, result.Artifacts.AST, verifyModulesFromPipeline(result), smt.VerifyOptions{
		Timeout:        opts.Timeout,
		RecursiveDepth: opts.RecursiveDepth,
		Verbose:        opts.Verbose,
	})
	mr.Verified, mr.Counterexample, mr.Skipped, mr.Errors, mr.Uncontracted =
		vr.Verified, vr.Counterexample, vr.Skipped, vr.Errors, vr.Uncontracted
	for _, r := range vr.Results {
		mr.Results = append(mr.Results, pkg.FunctionVerifyItem{Function: r.Function, Status: r.Status, Reason: r.Reason})
	}
	return mr
}

func printPackageVerifyJSON(report *pkg.PackageVerifyReport) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: JSON encoding error: %v\n", red("Error"), err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func printPackageVerifyHuman(report *pkg.PackageVerifyReport) {
	fmt.Printf("%s Verifying package %s@%s (%d exported modules)\n\n", cyan("→"), report.Package, report.Version, len(report.Modules))
	for _, m := range report.Modules {
		switch {
		case m.CompileError != "":
			fmt.Printf("  %s %s — %s\n", red("✗"), m.Module, m.CompileError)
		case m.Counterexample > 0:
			fmt.Printf("  %s %s — %d verified, %d counterexample\n", red("✗"), m.Module, m.Verified, m.Counterexample)
		case m.Verified == 0 && m.Uncontracted > 0:
			fmt.Printf("  %s %s — no contracts (%d exported)\n", yellow("·"), m.Module, m.Uncontracted)
		default:
			fmt.Printf("  %s %s — %d/%d verified\n", green("✓"), m.Module, m.Verified, m.Verified+m.Counterexample+m.Skipped+m.Errors)
		}
	}
	fmt.Printf("\n%d verified, %d counterexample, %d skipped, %d errors, %d uncontracted (%.1fs)\n",
		report.Verified, report.Counterexample, report.Skipped, report.Errors, report.Uncontracted, report.WallSeconds)
	if report.WallCapHit {
		fmt.Printf("%s package wall cap hit — remaining modules skipped\n", yellow("⚠"))
	}
}
