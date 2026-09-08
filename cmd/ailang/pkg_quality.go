package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pkg"
	ailangTesting "github.com/sunholo-data/ailang/internal/testing"
)

//go:embed guides/package-authoring.md
var packageAuthoringGuide string

type packageQualityReport struct {
	Package         string   `json:"package"`
	Kind            string   `json:"kind"`
	Limitations     []string `json:"limitations"`
	Execution       string   `json:"execution"`
	Files           int      `json:"files"`
	NativeTests     int      `json:"native_tests"`
	Properties      int      `json:"properties"`
	ContractClauses int      `json:"contract_clauses"`
	Gaps            []string `json:"gaps"`
	TestSources     []string `json:"test_sources"`
}

func pkgQualityCommand(args []string) error {
	return runPackageQuality(args, os.Stdout, os.Stderr)
}

func runPackageQuality(args []string, out, errOut io.Writer) error {
	fs := flag.NewFlagSet("pkg quality", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonMode := fs.Bool("json", false, "Print machine-readable evidence inventory")
	strict := fs.Bool("strict", false, "Fail when authoring evidence is missing (not a proof gate)")
	fs.Usage = func() {
		fmt.Fprintln(errOut, "Usage: ailang pkg quality [--json] [--strict] [DIR]\nOffline declaration inventory; never executes tests or proves contracts.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("expected at most one package directory; put flags before DIR")
	}
	dir := "."
	if fs.NArg() == 1 {
		dir = fs.Arg(0)
	}
	report, err := inspectPackageQuality(dir)
	if err != nil {
		return err
	}
	if *jsonMode {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			return err
		}
	} else {
		fmt.Fprintf(out, "Package %s — declaration inventory\nTests: %d; properties: %d; contract clauses: %d; files: %d\n", report.Package, report.NativeTests, report.Properties, report.ContractClauses, report.Files)
		fmt.Fprintln(out, "Compilation, test execution, coverage and proofs: NOT ASSESSED. Effects are syntactic, not inferred.")
		for _, gap := range report.Gaps {
			fmt.Fprintln(out, "MISSING:", gap)
		}
		if len(report.Gaps) == 0 {
			fmt.Fprintln(out, "No declaration gaps found; run check/test/verify separately.")
		}
	}
	if *strict && len(report.Gaps) > 0 {
		return fmt.Errorf("%d authoring evidence gaps; see ailang docs package-authoring", len(report.Gaps))
	}
	return nil
}

func inspectPackageQuality(dir string) (*packageQualityReport, error) {
	sources, err := pkg.DiscoverPackageSources(dir)
	if err != nil {
		return nil, err
	}
	report := &packageQualityReport{
		Package:   sources.Manifest.Package.Name,
		Kind:      "declaration_inventory",
		Execution: "not_run",
		Limitations: []string{
			"Declarations only: compilation, test execution, coverage and proofs not assessed",
			"Effect annotations only: inferred purity and inferred effects not assessed",
			"Presence does not establish assertion quality or behavioral coverage",
		},
		Gaps:        []string{},
		TestSources: []string{},
	}
	if len(sources.Manifest.Exports.Modules) == 0 {
		report.Gaps = append(report.Gaps, "no exported modules declared")
	}
	agent, err := os.ReadFile(filepath.Join(sources.Dir, "AGENT.md"))
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if len(strings.TrimSpace(string(agent))) == 0 {
		report.Gaps = append(report.Gaps, "AGENT.md usage/effects/validation guidance")
	}
	paths := append(sources.AllSourcePaths(), sources.OrphanFiles...)
	paths = append(paths, sources.TestFiles...)
	sort.Strings(paths)
	seen := map[string]bool{}
	for _, path := range paths {
		if seen[path] {
			continue
		}
		seen[path] = true
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		p := parser.New(lexer.New(string(data), path))
		prog := p.Parse()
		if errs := p.Errors(); len(errs) > 0 {
			return nil, fmt.Errorf("%s: %v", path, errs[0])
		}
		if prog == nil || prog.File == nil {
			return nil, fmt.Errorf("%s: no AST", path)
		}
		rel, err := filepath.Rel(sources.Dir, path)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		report.Files++
		suite := ailangTesting.NewCollector(rel).Collect(prog.File)
		tests := len(suite.Tests)
		properties := 0
		for _, prop := range suite.Properties {
			if prop.Property.Kind == ast.PropertyKind {
				properties++
			}
		}
		report.NativeTests += tests
		report.Properties += properties
		if tests+properties > 0 {
			report.TestSources = append(report.TestSources, rel)
		}
		for _, fn := range prog.File.Funcs {
			contracts := 0
			for _, prop := range fn.Properties {
				if prop.Kind == ast.RequiresKind || prop.Kind == ast.EnsuresKind {
					contracts++
				}
			}
			report.ContractClauses += contracts
			if strings.HasSuffix(path, "_test.ail") || fn.IsExtern {
				continue
			}
			label := fmt.Sprintf("%s:%d %s", rel, fn.Pos.Line, fn.Name)
			if len(fn.Effects) == 0 && contracts == 0 {
				report.Gaps = append(report.Gaps, label+": contract (no declared effects; inferred purity not assessed)")
			}
			for _, eff := range fn.Effects {
				if !eff.IsRowVar && eff.Budget == nil {
					report.Gaps = append(report.Gaps, label+": budget for "+eff.Name)
				}
			}
		}
	}
	if report.NativeTests+report.Properties == 0 {
		report.Gaps = append(report.Gaps, "no native test cases or property tests (contracts and test_* names alone do not count)")
	}
	sort.Strings(report.Gaps)
	return report, nil
}
