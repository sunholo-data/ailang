package format

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	aitesting "github.com/sunholo-data/ailang/internal/testing"
)

// combined_pipeline_test.go is the M2 acceptance-gated integration test for
// m-fmt-properties-printer-roundtrip. It locks the combined contracts+properties
// case end to end against testdata/contracts_and_properties.ail:
//
//	(a) checks clean — the fixture elaborates without error;
//	(b) the contract reaches contract verification — DeclMeta.Contracts holds
//	    EXACTLY one RequiresKind + one EnsuresKind and NOTHING from the forall;
//	(c) the forall reaches ONLY the property pipeline — fn.Properties is EXACTLY
//	    [RequiresKind, EnsuresKind, PropertyKind] and the collector emits EXACTLY
//	    one PropertyCase whose Property.Kind is PropertyKind (the forall);
//	(d) no duplication/omission/panic — all counts are exact (== not >=), the
//	    test completing proves panic-freedom, and the fmt round-trip asserts AST
//	    identity with the requires clause present in the output.
const combinedFixturePath = "testdata/contracts_and_properties.ail"

func TestCombinedContractsAndPropertiesPipeline(t *testing.T) {
	data, err := os.ReadFile(combinedFixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	src := string(data)

	// --- Parse: source-of-truth ordering (c, parse-level) ---
	p := parser.New(lexer.New(src, combinedFixturePath))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse errors: %v", errs)
	}
	fn := findFunc(t, file, "f")

	wantKinds := []ast.ContractKind{ast.RequiresKind, ast.EnsuresKind, ast.PropertyKind}
	if len(fn.Properties) != len(wantKinds) {
		t.Fatalf("fn.Properties = %v, want exactly [Requires, Ensures, Property]", propKinds(fn.Properties))
	}
	for i, want := range wantKinds {
		if fn.Properties[i].Kind != want {
			t.Fatalf("fn.Properties[%d].Kind = %v, want %v (full: %v)", i, fn.Properties[i].Kind, want, propKinds(fn.Properties))
		}
	}

	// --- Elaborate: (a) checks clean + (b) contract reaches DeclMeta.Contracts ---
	elab := elaborate.NewElaborator()
	prog, err := elab.ElaborateFile(file)
	if err != nil {
		t.Fatalf("elaboration error (fixture must check clean): %v", err)
	}
	meta, ok := prog.Meta["f"]
	if !ok {
		t.Fatalf("no DeclMeta for f")
	}
	var reqN, ensN, otherN int
	for _, c := range meta.Contracts {
		switch c.Kind {
		case core.RequiresKind:
			reqN++
		case core.EnsuresKind:
			ensN++
		default:
			otherN++
		}
	}
	if reqN != 1 || ensN != 1 || otherN != 0 || len(meta.Contracts) != 2 {
		t.Fatalf("DeclMeta.Contracts = %d total (requires=%d ensures=%d other=%d), want exactly 1 requires + 1 ensures + 0 from the forall",
			len(meta.Contracts), reqN, ensN, otherN)
	}

	// --- Collect: (c) forall reaches ONLY the property pipeline ---
	suite := aitesting.NewCollector(combinedFixturePath).Collect(file)
	var forallN, contractCaseN int
	for _, pc := range suite.Properties {
		if pc.FunctionCtx != "f" {
			continue
		}
		if pc.Property.Kind == ast.PropertyKind {
			forallN++
		} else {
			contractCaseN++
		}
	}
	if forallN != 1 {
		t.Fatalf("collector emitted %d PropertyKind property cases for f, want exactly 1 (the forall)", forallN)
	}
	// The collector also emits a PropertyCase per requires/ensures entry (today's
	// pre-existing behavior, audit site 5); those are Kind-filtered downstream by
	// the runner. We assert they are exactly the two contract entries — no more, no
	// fewer — proving no duplication or omission through the collector.
	if contractCaseN != 2 {
		t.Fatalf("collector emitted %d non-property (contract) property cases for f, want exactly 2 (requires+ensures)", contractCaseN)
	}

	// --- fmt round-trip: (d) AST identity + requires clause survives in output ---
	out := assertIdempotentAndRoundTrips(t, src, combinedFixturePath)
	if !strings.Contains(out, "requires {") {
		t.Errorf("fmt output lost the requires clause:\n%s", out)
	}
	if !strings.Contains(out, "ensures {") {
		t.Errorf("fmt output lost the ensures clause:\n%s", out)
	}
	if !strings.Contains(out, "properties [") {
		t.Errorf("fmt output lost the properties block:\n%s", out)
	}

	// Belt-and-suspenders: re-parse the formatted output and confirm the kind
	// partition is still exactly [Requires, Ensures, Property] (no clobber, no dup).
	rp := parser.New(lexer.New(out, combinedFixturePath))
	refile := rp.ParseFile()
	if errs := rp.Errors(); len(errs) > 0 {
		t.Fatalf("re-parse of formatted output failed: %v", errs)
	}
	refn := findFunc(t, refile, "f")
	if diff := cmp.Diff(propKinds(fn.Properties), propKinds(refn.Properties)); diff != "" {
		t.Errorf("property-kind order changed across fmt round-trip (-orig +formatted):\n%s", diff)
	}
}

func findFunc(t *testing.T, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name == name {
			return fd
		}
	}
	t.Fatalf("func %q not found", name)
	return nil
}
