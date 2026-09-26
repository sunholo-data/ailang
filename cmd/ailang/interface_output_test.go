package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

const preRefactorModuleGreetJSON = "{\n  \"module\": \"examples/docs/module_greet\",\n  \"types\": [],\n  \"funcs\": [\n    {\n      \"name\": \"greet\",\n      \"type\": \"(string)-\\u003e()!{IO}\",\n      \"effects\": [\n        \"IO\"\n      ],\n      \"pure\": true\n    }\n  ],\n  \"schema\": \"ailang.iface/v1\"\n}\n"

// repoRoot returns the repository root, two levels above cmd/ailang.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	return root
}

// TestOutputInterface_ByteIdenticalToPreRefactor pins the stdout bytes of
// `ailang iface` against a golden captured from the pre-refactor binary.
//
// It deliberately routes through outputInterfacePackageDir rather than a
// literal, so it exercises the argument outputInterface actually passes.
func TestOutputInterface_ByteIdenticalToPreRefactor(t *testing.T) {
	t.Chdir(repoRoot(t))
	got, err := pipeline.BuildCanonicalJSON(context.Background(), outputInterfacePackageDir, "examples/docs/module_greet.ail")
	if err != nil {
		t.Fatalf("BuildCanonicalJSON: %v", err)
	}
	got = append(got, '\n') // outputInterface prints the normalized JSON with one trailing newline.
	if string(got) != preRefactorModuleGreetJSON {
		t.Fatalf("outputInterface bytes changed from pre-refactor golden\ngot:\n%s\nwant:\n%s", got, preRefactorModuleGreetJSON)
	}
}

// TestOutputInterface_ResolvesIntraPackageImportsFromRepoRoot is the
// discriminating arm the golden above cannot be.
//
// module_greet.ail has no intra-package imports, so it produces identical bytes
// whether packageDir is "" or "."; only a module that imports a sibling can
// tell those two apart. This arm runs from the REPOSITORY ROOT — i.e. outside
// the fixture's own package — which is exactly the ailang#671 scenario:
// packageSearchDir must anchor the ailang.toml search at the entry file's
// directory, and a non-empty packageDir defeats that by winning outright.
//
// Kills the mutant `outputInterfacePackageDir = "."`, which was measured to
// take `ailang iface examples/intra_package_imports/service.ail` from
// rc=0/235 bytes to rc=1/0 bytes.
func TestOutputInterface_ResolvesIntraPackageImportsFromRepoRoot(t *testing.T) {
	t.Chdir(repoRoot(t))
	got, err := pipeline.BuildCanonicalJSON(context.Background(), outputInterfacePackageDir, "examples/intra_package_imports/service.ail")
	if err != nil {
		t.Fatalf("intra-package interface must resolve from the repo root (ailang#671): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected a non-empty interface JSON")
	}
}
