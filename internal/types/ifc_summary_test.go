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

func TestIFCExportLocalProducerValues(t *testing.T) {
	file := parseIFC(t, `module test/local_producer_summary
 type Tagged = {secret: string<secret>, safe: string}
 pure func source() -> string<secret> { "secret" }
 pure func other() -> string<email> { "email" }
 pure func direct() -> string { source() }
 export pure func alias() -> string { let f = direct in f() }
 pure func joined(n: int) -> string { let f = if n == 0 then direct else other in f() }
 pure func mixed(n: int) -> string { let f = if n == 0 then direct else unknown in f() }
 pure func shadow() -> string { let direct = \u. "clean" in direct(()) }
 pure func stored() -> () -> string { direct }
 pure func returned() -> string { let f = stored() in f() }
 pure func clean() -> string { "clean" }
 pure func storedClean() -> () -> string { clean }
 pure func take(t: Tagged) -> string { t.safe }
 pure func handoff(t: Tagged) -> string { let f = take in f(t) }
 `)
	supplied := types.EmptyIFCSummary()
	supplied.Intrinsic = types.EncodeIFCLabel(types.LabelConst("other"))
	sigs, err := types.AnalyzeModuleIFC(file, map[string]*types.ImportedIFCSig{"direct": supplied})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"direct", "alias", "joined", "mixed", "returned", "stored"} {
		if !types.LabelSubsumes(sigs[name].Intrinsic.Decode(), types.LabelConst("secret")) {
			t.Fatalf("%s lost local producer intrinsic: %+v", name, sigs[name])
		}
	}
	if !types.LabelSubsumes(sigs["joined"].Intrinsic.Decode(), types.LabelConst("email")) {
		t.Fatal("local producer join dropped an alternative")
	}
	for _, name := range []string{"shadow", "storedClean", "take", "handoff"} {
		if !types.LabelEqual(sigs[name].Intrinsic.Decode(), types.LabelBottom()) {
			t.Fatalf("%s gained an unrelated intrinsic label: %+v", name, sigs[name])
		}
	}
}

func TestIFCExportProducerEtaParity(t *testing.T) {
	file := parseIFC(t, `module test/producer_eta
 type SecretRec = {value: string<secret>}
 pure func source() -> string<secret> { "secret" }
 pure func recSource() -> SecretRec { {value: "secret"} }
 pure func apply(f: () -> string) -> string { f() }
 pure func ignore(f: () -> string) -> string { "clean" }
 pure func ref() -> string { apply(source) }
 pure func eta() -> string { apply(\_. source()) }
 pure func ignoredRef() -> string { ignore(source) }
 pure func ignoredEta() -> string { ignore(\_. source()) }
 pure func storedRef() -> () -> string { source }
 pure func storedEta() -> () -> string { \_. source() }
 pure func recordRef() -> () -> SecretRec { recSource }
 pure func recordEta() -> () -> SecretRec { \_. recSource() }
 pure func importedRef() -> string { apply(imported) }
 pure func importedEta() -> string { apply(\_. imported()) }
 pure func importedStoredRef() -> () -> string { imported }
 pure func importedStoredEta() -> () -> string { \_. imported() }
 `)
	source := types.EmptyIFCSummary()
	source.Intrinsic = types.EncodeIFCLabel(types.LabelConst("secret"))
	sigs, err := types.AnalyzeModuleIFC(file, map[string]*types.ImportedIFCSig{"imported": source})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"ref", "eta"}, {"ignoredRef", "ignoredEta"}, {"storedRef", "storedEta"}, {"recordRef", "recordEta"}, {"importedRef", "importedEta"}, {"importedStoredRef", "importedStoredEta"}} {
		got, want := sigs[pair[0]].Intrinsic, sigs[pair[1]].Intrinsic
		if !reflect.DeepEqual(got, want) || !types.LabelSubsumes(got.Decode(), types.LabelConst("secret")) {
			t.Fatalf("producer eta parity %s/%s: got=%+v want=%+v", pair[0], pair[1], got, want)
		}
	}
}

func TestIFCExportKnownSeedEtaParity(t *testing.T) {
	const template = `module test/seed_eta
 pure func apply(f: string -> string) -> string { f("key") }
 pure func ignore(f: string -> string) -> string { "clean" }
 pure func ref() -> string { apply(SOURCE) }
 pure func eta() -> string { apply(\key. SOURCE(key)) }
 pure func ignoredRef() -> string { ignore(SOURCE) }
 pure func ignoredEta() -> string { ignore(\key. SOURCE(key)) }
 pure func storedRef() -> string -> string { SOURCE }
 pure func storedEta() -> string -> string { \key. SOURCE(key) }
 `
	for _, seed := range []string{"secret", "_secret_read"} {
		t.Run(seed, func(t *testing.T) {
			for _, local := range []bool{false, true} {
				src := strings.ReplaceAll(template, "SOURCE", seed)
				if local {
					src = strings.Replace(src, "module test/seed_eta", "module test/seed_eta\npure func "+seed+"(key: string) -> string { \"clean\" }", 1)
				}
				sigs, err := types.AnalyzeModuleIFC(parseIFC(t, src), nil)
				if err != nil {
					t.Fatal(err)
				}
				for _, pair := range [][2]string{{"ref", "eta"}, {"ignoredRef", "ignoredEta"}, {"storedRef", "storedEta"}} {
					got, want := sigs[pair[0]].Intrinsic, sigs[pair[1]].Intrinsic
					if !reflect.DeepEqual(got, want) || (local && !types.LabelEqual(got.Decode(), types.LabelBottom())) || (!local && !types.LabelSubsumes(got.Decode(), types.LabelConst("secret"))) {
						t.Fatalf("local=%v seed parity %s/%s: got=%+v want=%+v", local, pair[0], pair[1], got, want)
					}
				}
			}
			lexical := strings.ReplaceAll(`module test/seed_shadow
   pure func apply(f: string -> string) -> string { f("key") }
   pure func ref() -> string { let SOURCE = \key. "clean"; apply(SOURCE) }
   pure func eta() -> string { let SOURCE = \key. "clean"; apply(\key. SOURCE(key)) }
   `, "SOURCE", seed)
			sigs, err := types.AnalyzeModuleIFC(parseIFC(t, lexical), nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"ref", "eta"} {
				if !types.LabelEqual(sigs[name].Intrinsic.Decode(), types.LabelBottom()) {
					t.Fatalf("lexical %s shadow inherited %s source fact: %+v", name, seed, sigs[name])
				}
			}
		})
	}
}

func TestIFCExportBodylessDeclarations(t *testing.T) {
	file := parseIFC(t, `module test/extern_summary
 extern func plain(n: int) -> string
 extern func nativeSource(input: string<email>) -> string<secret>
 extern func sanitized(input: string{not secret}) -> string<clean> ! {Declassify[label=email]}
 pure func normal() -> string<secret> { "secret" }
 `)
	sigs, err := types.AnalyzeModuleIFC(file, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(sigs) != 4 {
		t.Fatalf("bodyless declarations omitted: %v", sigs)
	}
	for name, s := range sigs {
		if err := s.Validate(); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if !types.LabelEqual(sigs["plain"].Intrinsic.Decode(), types.LabelBottom()) {
		t.Fatal("bodyless plain declaration gained a fabricated body label")
	}
	s := sigs["nativeSource"]
	if !s.HasReturnLabel || !types.LabelSubsumes(s.Return.Decode(), types.LabelConst("secret")) || !types.LabelSubsumes(s.Params[0].Label.Decode(), types.LabelConst("email")) {
		t.Fatalf("bodyless declared facts lost: %+v", s)
	}
	if !reflect.DeepEqual(sigs["sanitized"].Params[0].Forbidden, []string{"secret"}) || !reflect.DeepEqual(sigs["sanitized"].Declassify, []string{"email"}) {
		t.Fatal("bodyless refinement/authority lost")
	}
	if !types.LabelSubsumes(sigs["normal"].Intrinsic.Decode(), types.LabelConst("secret")) {
		t.Fatal("ordinary labelled declaration lost")
	}
}

func TestIFCExportProducerResultPolicies(t *testing.T) {
	file := parseIFC(t, `module test/producer_result_policies
 pure func secretSource() -> string<secret> { "secret" }
 pure func emailSource() -> string<email> { "email" }
 func bare(x: string) -> string<clean> ! {Declassify} { x }
 func scoped(x: string) -> string<clean> ! {Declassify[label=email]} { x }
 pure func declared(x: string) -> string<clean> { "fixed" }
 func callBare() -> string { let f = bare in f(secretSource()) }
 func callScopedEmail() -> string { let f = scoped in f(emailSource()) }
 func callScopedSecret() -> string { let f = scoped in f(secretSource()) }
 func callDeclared() -> string { let f = declared in f(secretSource()) }
 `)
	sigs, err := types.AnalyzeModuleIFC(file, nil)
	if err != nil {
		t.Fatal(err)
	}
	clean := types.LabelConst("clean")
	for _, name := range []string{"callBare", "callScopedEmail", "callDeclared"} {
		if !types.LabelEqual(sigs[name].Intrinsic.Decode(), clean) {
			t.Fatalf("%s lost declared/authorized result policy: %+v", name, sigs[name])
		}
	}
	want := types.LabelJoin(clean, types.LabelConst("secret"))
	if !types.LabelEqual(sigs["callScopedSecret"].Intrinsic.Decode(), want) {
		t.Fatal("scoped authority erased unrelated producer label")
	}
}
