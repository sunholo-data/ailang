package types_test

import (
	"github.com/sunholo-data/ailang/internal/types"
	"reflect"
	"strings"
	"testing"
)

func TestIFCSummaries(t *testing.T) {
	file := parseIFC(t, `module test/summary
func gate(x: string<email>, y: string{not secret}) -> string<email> ! {} { x }
func reveal(x: string<email>) -> string<clean> ! {Declassify[label=email]} { x }
func bare(x: string<secret>) -> string<clean> ! {Declassify} { x }
func first(n: int) -> string { second(n) }
func second(n: int) -> string { if n == 0 then secret("key") else first(n - 1) }
func wrapper(n: int) -> string { imported(n) }
`)
	imported := types.EmptyIFCSummary()
	imported.Intrinsic = types.EncodeIFCLabel(types.LabelJoin(types.LabelConst("secret"), types.LabelVar("L")))
	sigs, err := types.AnalyzeModuleIFC(file, map[string]*types.ImportedIFCSig{"imported": imported})
	if err != nil {
		t.Fatal(err)
	}
	gate := sigs["gate"]
	if gate.Params[0].Name != "x" || gate.Params[1].Forbidden[0] != "secret" || !types.LabelEqual(gate.Params[0].Label.Decode(), types.LabelConst("email")) || !gate.HasReturnLabel {
		t.Fatalf("gate: %+v", gate)
	}
	if !reflect.DeepEqual(sigs["reveal"].Declassify, []string{"email"}) || !reflect.DeepEqual(sigs["bare"].Declassify, []string{"*"}) || sigs["gate"].Declassify == nil {
		t.Fatal("authority lost")
	}
	for _, name := range []string{"first", "second"} {
		if !types.LabelSubsumes(sigs[name].Intrinsic.Decode(), types.LabelConst("secret")) {
			t.Fatalf("recursion lost taint: %s", name)
		}
	}
	if !reflect.DeepEqual(sigs["wrapper"].Intrinsic, imported.Intrinsic) {
		t.Fatal("imported join lost")
	}
}

func TestIFCSummaryValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*types.ImportedIFCSig)
	}{
		{"params missing", func(s *types.ImportedIFCSig) { s.Params = nil }},
		{"authority missing", func(s *types.ImportedIFCSig) { s.Declassify = nil }},
		{"constituents missing", func(s *types.ImportedIFCSig) { s.Intrinsic.Constants = nil }},
		{"duplicate", func(s *types.ImportedIFCSig) { s.Intrinsic.Constants = []string{"email", "email"} }},
		{"unsorted", func(s *types.ImportedIFCSig) { s.Declassify = []string{"secret", "email"} }},
		{"wildcard mixed", func(s *types.ImportedIFCSig) { s.Declassify = []string{"*", "email"} }},
		{"reserved source", func(s *types.ImportedIFCSig) { s.Intrinsic.Constants = []string{"*"} }},
		{"undeclared return", func(s *types.ImportedIFCSig) { s.Return = types.EncodeIFCLabel(types.LabelConst("secret")) }},
		{"param name missing", func(s *types.ImportedIFCSig) {
			s.Params = []types.IFCParam{{Applied: types.EncodeIFCLabel(types.LabelBottom()), Forbidden: []string{}, Label: types.EncodeIFCLabel(types.LabelBottom())}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := types.EmptyIFCSummary()
			tc.mutate(s)
			if s.Validate() == nil {
				t.Fatal("malformed summary accepted")
			}
		})
	}
	for _, authority := range [][]string{{}, {"*"}, {"email"}, {"email", "secret"}} {
		s := types.EmptyIFCSummary()
		s.Declassify = authority
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	var absent *types.ImportedIFCSig
	if absent.Validate() == nil {
		t.Fatal("missing summary accepted")
	}
}

func TestIFCSummaryShadowingAndBottom(t *testing.T) {
	file := parseIFC(t, `module test/shadow
func local(x: string) -> string { x }
func forward(x: string) -> string { local(x) }
func shadow(x: string) -> string { let imported = \u. "clean" in imported(x) }
`)
	source := types.EmptyIFCSummary()
	source.Intrinsic = types.EncodeIFCLabel(types.LabelConst("secret"))
	sigs, err := types.AnalyzeModuleIFC(file, map[string]*types.ImportedIFCSig{"imported": source, "local": source})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"local", "forward", "shadow"} {
		if !types.LabelEqual(sigs[name].Intrinsic.Decode(), types.LabelBottom()) {
			t.Fatalf("%s should be bottom", name)
		}
	}
}

func TestIFCSyntheticSummary(t *testing.T) {
	s := types.IFCSummaryFromScheme(&types.Scheme{Type: &types.TFunc2{Params: []types.Type{types.WithLabel(types.TString, types.LabelConst("email"))}, Return: types.WithLabel(types.TString, types.LabelConst("secret"))}})
	if !s.HasReturnLabel || !types.LabelEqual(s.Return.Decode(), types.LabelConst("secret")) || !types.LabelEqual(s.Params[0].Label.Decode(), types.LabelConst("email")) {
		t.Fatalf("lost synthetic labels: %+v", s)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, scheme := range []*types.Scheme{nil, {Type: types.TInt}} {
		if err := types.IFCSummaryFromScheme(scheme).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIFCSummaryNestedDeclaredResult(t *testing.T) {
	file := parseIFC(t, `module test/nested_summary
 type Result = { email: string<email>, token: string<secret> }
 func make(x: Result) -> Result { x }
 `)
	sigs, err := types.AnalyzeModuleIFC(file, nil)
	if err != nil {
		t.Fatal(err)
	}
	intrinsic := sigs["make"].Intrinsic.Decode()
	if !types.LabelSubsumes(intrinsic, types.LabelConst("email")) || !types.LabelSubsumes(intrinsic, types.LabelConst("secret")) {
		t.Fatalf("declared result join lost: %s", intrinsic)
	}
}

func TestIFCExportSummaryDoesNotPublishRho(t *testing.T) {
	file := parseIFC(t, `module test/summary_rho
 func sink(x: string{not email}) -> string { x }
 func wrap(x: string) -> string { sink(x) }
 func apply(f: string -> string, x: string) -> string { f(x) }
 func captured(x: string<secret>) -> string -> string { \u. x }
 `)
	sigs, err := types.AnalyzeModuleIFC(file, nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, s := range sigs {
		for _, l := range []types.IFCLabel{s.Intrinsic, s.Return} {
			for _, v := range l.Variables {
				if strings.HasPrefix(v, "ρ") {
					t.Fatalf("%s published private variable %s", name, v)
				}
			}
		}
	}
}

func TestIFCExportImportedValueLookup(t *testing.T) {
	file := parseIFC(t, `module test/import_value_summary
 func alias(n: int) -> string { let saved = imported in saved(n) }
 func branch(n: int) -> string { let f = if n == 0 then imported else other in f(n) }
 func shadow(n: int) -> string { let imported = \x. "clean" in imported(n) }
 func choice(tag: string<secret>, n: int) -> string { let holder = {tag: tag, callable: clean} in holder.callable(n) }
 `)
	source := types.EmptyIFCSummary()
	source.Intrinsic = types.EncodeIFCLabel(types.LabelConst("secret"))
	other := types.EmptyIFCSummary()
	other.Intrinsic = types.EncodeIFCLabel(types.LabelConst("email"))
	sigs, err := types.AnalyzeModuleIFC(file, map[string]*types.ImportedIFCSig{"imported": source, "other": other, "clean": types.EmptyIFCSummary()})
	if err != nil {
		t.Fatal(err)
	}
	if !types.LabelSubsumes(sigs["alias"].Intrinsic.Decode(), types.LabelConst("secret")) {
		t.Fatal("aliased supplied producer lost intrinsic taint")
	}
	if got := sigs["branch"].Intrinsic.Decode(); !types.LabelSubsumes(got, types.LabelConst("secret")) || !types.LabelSubsumes(got, types.LabelConst("email")) {
		t.Fatalf("joined producer facts lost: %s", got)
	}
	if !types.LabelSubsumes(sigs["choice"].Intrinsic.Decode(), types.LabelConst("secret")) {
		t.Fatal("supplied producer discarded the callable container label")
	}
	if !types.LabelEqual(sigs["shadow"].Intrinsic.Decode(), types.LabelBottom()) {
		t.Fatal("lexical shadow used imported producer facts")
	}
}
