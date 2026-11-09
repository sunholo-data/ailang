package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-IFC-CROSS-MODULE M3 (#1134): an imported function's exported IFC summary
// is enforced at the importer's call sites — its {not ℓ} sinks (Check A), its
// positive parameter labels (Check C) and its result label.

const xmodLib = `module lib
export func gate(input: string{not secret}) -> bool ! {} { input != "" }
export func leak() -> string<secret> ! {} { "hunter2" }
export func wrapped() -> string ! {} { leak() }
export func echo(s: string) -> string ! {} { s }
export func plain() -> string ! {} { "ok" }
export func sql(q: string<sqlsafe>) -> string ! {} { q }
export func redact(s: string) -> string ! {Declassify} { "***" }
export func unmask(s: string) -> string ! {Declassify[label=email]} { s }
export func apply1(f: (string) -> bool, x: string) -> bool ! {} { f(x) }
`

// libSigs analyses a library module and returns its export summaries.
func libSigs(t *testing.T, src string, imports types.IFCImports) map[string]*types.ImportedIFCSig {
	t.Helper()
	sigs, err := types.AnalyzeModuleIFC(parseIFC(t, src), imports.Sigs())
	if err != nil {
		t.Fatal(err)
	}
	return sigs
}

// bindAll binds every export of module under prefix+name ("" = selective
// import of everything, "L." = module alias L).
func bindAll(module, prefix string, sigs map[string]*types.ImportedIFCSig) types.IFCImports {
	out := types.IFCImports{}
	for name, s := range sigs {
		out[prefix+name] = types.IFCImport{Module: module, Name: name, Sig: s}
	}
	return out
}

func checkCaller(t *testing.T, body string, imports types.IFCImports) []*types.TypeCheckError {
	t.Helper()
	return types.CheckModuleIFCWithImports(parseIFC(t, "module app\n"+body), imports)
}

func TestIFCImportedSinkRefinement(t *testing.T) {
	lib := bindAll("lib", "", libSigs(t, xmodLib, nil))
	secretArg := "func s() -> string<secret> ! {} { \"k\" }\n"
	for _, tc := range []struct {
		name, body, want string
	}{
		{"direct", `export func main() -> bool ! {} { gate(s()) }`, `reaches parameter "input" of imported function lib.gate, which forbids it (string{not secret})`},
		{"let alias", `export func main() -> bool ! {} { let g = gate in g(s()) }`, `of imported function lib.gate (via g), which forbids it`},
		{"higher-order local", `func ap(f: (string) -> bool, x: string) -> bool ! {} { f(x) }
export func main() -> bool ! {} { ap(gate, s()) }`, `of imported function lib.gate`},
		// A wrapper's inferred demand is replayed where the concrete label is
		// supplied; the message names the imported callee and the route.
		{"local wrapper", `func w(y: string) -> bool ! {} { gate(y) }
export func main() -> bool ! {} { w(s()) }`, `of lib.gate (via w → gate) which forbids it`},
		{"lambda wrapper", `export func main() -> bool ! {} { let w = \y. gate(y) in w(s()) }`, `of lib.gate (via w → gate) which forbids it`},
		{"imported secret producer", `export func main() -> bool ! {} { gate(leak()) }`, `value labelled <secret> reaches parameter "input" of imported function lib.gate`},
		{"intrinsic producer", `export func main() -> bool ! {} { gate(wrapped()) }`, `of imported function lib.gate`},
		{"transparent imported", `export func main() -> bool ! {} { gate(echo(s())) }`, `of imported function lib.gate`},
		{"scoped declassify does not cover secret", `export func main() -> bool ! {} { gate(unmask(s())) }`, `of imported function lib.gate`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			errs := checkCaller(t, secretArg+tc.body, lib)
			if len(errs) != 1 {
				t.Fatalf("want 1 error, got %d: %v", len(errs), errs)
			}
			if errs[0].Kind != types.SinkRefinementError || !strings.Contains(errs[0].Message, tc.want) {
				t.Fatalf("kind %s, message %q; want %q", errs[0].Kind, errs[0].Message, tc.want)
			}
			if strings.Contains(tc.want, "imported function") && !strings.Contains(errs[0].Suggestion, "lib.gate(input)") {
				t.Fatalf("suggestion does not name the edge: %q", errs[0].Suggestion)
			}
		})
	}
}

func TestIFCImportedCleanControls(t *testing.T) {
	lib := bindAll("lib", "", libSigs(t, xmodLib, nil))
	for _, tc := range []struct{ name, body string }{
		{"literal", `export func main() -> bool ! {} { gate("hello") }`},
		{"clean producer", `export func main() -> bool ! {} { gate(plain()) }`},
		{"bare declassify", `export func main() -> bool ! {} { gate(redact(leak())) }`},
		{"echo of clean", `export func main() -> bool ! {} { gate(echo("x")) }`},
		{"sql literal", `export func main() -> string ! {} { sql("select 1") }`},
		{"sql matching label", `func q() -> string<sqlsafe> ! {} { "select 1" }
export func main() -> string ! {} { sql(q()) }`},
		// A local declaration shadows the import of the same name.
		{"local shadows import", `func gate(input: string) -> bool ! {} { input != "" }
export func main() -> bool ! {} { gate(leak()) }`},
		// So does a lexical binding.
		{"lexical shadows import", `export func main() -> bool ! {} { let gate = \x. x != "" in gate(leak()) }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if errs := checkCaller(t, tc.body, lib); len(errs) != 0 {
				t.Fatalf("clean caller rejected: %v", errs)
			}
		})
	}
}

func TestIFCImportedResultTaintsCallerSink(t *testing.T) {
	lib := bindAll("lib", "", libSigs(t, xmodLib, nil))
	sink := "func sink(x: string{not secret}) -> string ! {} { x }\n"
	for _, body := range []string{
		`export func main() -> string ! {} { sink(leak()) }`,              // declared <secret> return
		`export func main() -> string ! {} { sink(wrapped()) }`,           // unannotated intrinsic taint
		`export func main() -> string ! {} { let k = leak() in sink(k) }`, // through a binding
		`export func main() -> string ! {} { sink(echo(leak())) }`,        // through a transparent import
		`export func main() -> string ! {} { let f = leak in sink(f()) }`, // an imported function value
		`export func main() -> string ! {} { sink(unmask(leak())) }`,      // scoped authority, unrelated label
	} {
		errs := checkCaller(t, sink+body, lib)
		if len(errs) != 1 || errs[0].Kind != types.SinkRefinementError || !strings.Contains(errs[0].Message, `<secret> reaches parameter "x"`) {
			t.Errorf("%s: want the local sink to reject the imported <secret>, got %v", body, errs)
		}
	}
}

func TestIFCImportedParamLabelCover(t *testing.T) {
	lib := bindAll("lib", "", libSigs(t, xmodLib, nil))
	errs := checkCaller(t, `export func main() -> string ! {} { sql(leak()) }`, lib)
	if len(errs) != 1 || errs[0].Kind != types.ParamLabelCoverError {
		t.Fatalf("want one ParamLabelCoverError, got %v", errs)
	}
	want := `information-flow violation: value labelled <secret> reaches parameter "q" of imported function lib.sql declared <sqlsafe>, which does not cover it`
	if errs[0].Message != want {
		t.Fatalf("message %q\nwant    %q", errs[0].Message, want)
	}
}

func TestIFCImportedIdentityForms(t *testing.T) {
	sigs := libSigs(t, xmodLib, nil)
	renamed := types.IFCImports{"check": {Module: "lib", Name: "gate", Sig: sigs["gate"]}, "leak": {Module: "lib", Name: "leak", Sig: sigs["leak"]}}
	errs := checkCaller(t, `export func main() -> bool ! {} { check(leak()) }`, renamed)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "imported function lib.gate") {
		t.Fatalf("renamed import: %v", errs)
	}
	qualified := bindAll("lib", "L.", sigs)
	errs = checkCaller(t, `export func main() -> bool ! {} { L.gate(L.leak()) }`, qualified)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "imported function lib.gate") {
		t.Fatalf("qualified import: %v", errs)
	}
	// A record binder named like the alias is a record, not the module (#1467).
	errs = checkCaller(t, `export func main() -> string ! {} { let L = {gate: "x"} in L.gate }`, qualified)
	if len(errs) != 0 {
		t.Fatalf("record shadowing the alias: %v", errs)
	}
}

// A library wrapping ANOTHER library's labelled producer exports the taint
// as its own intrinsic label (summaries built with the dependency's imports).
func TestIFCImportedTransitiveWrapper(t *testing.T) {
	lib := bindAll("lib", "", libSigs(t, xmodLib, nil))
	mid := libSigs(t, `module mid
export func fetch() -> string ! {} { leak() }
`, lib)
	if !types.LabelSubsumes(mid["fetch"].Intrinsic.Decode(), types.LabelConst("secret")) {
		t.Fatalf("transitive wrapper lost taint: %+v", mid["fetch"].Intrinsic)
	}
	errs := checkCaller(t, `export func main() -> bool ! {} { gate(fetch()) }`, types.IFCImports{
		"gate":  {Module: "lib", Name: "gate", Sig: lib["gate"].Sig},
		"fetch": {Module: "mid", Name: "fetch", Sig: mid["fetch"]},
	})
	if len(errs) != 1 {
		t.Fatalf("transitive taint into imported sink: %v", errs)
	}
}

func TestIFCImportedMissingMetadataFailsLoudly(t *testing.T) {
	errs := checkCaller(t, `export func main() -> bool ! {} { gate("x") }`, types.IFCImports{"gate": {Module: "lib", Name: "gate", Sig: nil}})
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "metadata missing or malformed") || !strings.Contains(errs[0].Message, "lib.gate") {
		t.Fatalf("missing summary must be an error, got %v", errs)
	}
}
