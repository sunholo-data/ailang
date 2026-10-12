package types_test

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
	"strings"
	"testing"
)

func TestIFCScopedAuthority(t *testing.T) {
	for _, tc := range []struct {
		label, scope string
		fail         bool
	}{{"email", "email", false}, {"secret", "email", true}, {"secret", "", false}} {
		t.Run(tc.label+tc.scope, func(t *testing.T) {
			eff := "Declassify"
			if tc.scope != "" {
				eff += "[label=" + tc.scope + "]"
			}
			src := fmt.Sprintf("module test/scoped\nfunc reveal(x: string<%s>) -> string<clean> ! {%s} { x }", tc.label, eff)
			errs := types.CheckModuleIFC(parseIFC(t, src))
			if tc.fail {
				if len(errs) != 1 || !strings.Contains(errs[0].Error(), "email") {
					t.Fatalf("expected scoped rejection: %v", errs)
				}
			} else if len(errs) != 0 {
				t.Fatal(errs)
			}
		})
	}
}
func TestIFCPositiveParameterCoverage(t *testing.T) {
	for _, tc := range []struct {
		label string
		fail  bool
	}{{"arg", true}, {"secret", true}, {"sqlsafe", false}, {"", false}} {
		annot := "string"
		if tc.label != "" {
			annot += "<" + tc.label + ">"
		}
		src := fmt.Sprintf("module test/positive\nfunc sink(x: string<sqlsafe>) -> string ! {} { x }\nfunc caller(x: %s) -> string ! {} { sink(x) }", annot)
		errs := types.CheckModuleIFC(parseIFC(t, src))
		if tc.fail {
			if len(errs) != 1 || string(errs[0].Kind) != "param_label_cover" {
				t.Fatalf("expected call edge rejection: %v", errs)
			}
		} else if len(errs) != 0 {
			t.Fatal(errs)
		}
	}
}

func TestIFCScopedConservativeFlows(t *testing.T) {
	for _, body := range []string{"x ++ y", "let f = \\u. y in f(0)", "let z = reveal(x) in y"} {
		src := "module test/mixed\nfunc reveal(x: string<email>) -> string<clean> ! {Declassify[label=email]} { x }\nfunc caller(x: string<email>, y: string<secret>) -> string<clean> ! {Declassify[label=email]} { " + body + " }"
		errs := types.CheckModuleIFC(parseIFC(t, src))
		if len(errs) != 1 || errs[0].Kind != types.DeclassifyRequiredError || !strings.Contains(errs[0].Error(), "secret") {
			t.Fatalf("%s: %v", body, errs)
		}
	}
}
func TestIFCPositiveConservativeArguments(t *testing.T) {
	for _, body := range []string{"sink(x ++ y)", "let f = \\u. y in sink(f(0))", "sink(y)", "sink(\\u. y)"} {
		src := "module test/mixed\nfunc sink(x: string<email>) -> string ! {} { x }\nfunc caller(x: string<email>, y: string<secret>) -> string ! {} { " + body + " }"
		errs := types.CheckModuleIFC(parseIFC(t, src))
		if len(errs) != 1 || errs[0].Kind != types.ParamLabelCoverError {
			t.Fatalf("%s: %v", body, errs)
		}
	}
}
func TestIFCPositiveSilentWalk(t *testing.T) {
	src := `module test/silent
func sink(x: string<email>) -> string ! {} { x }
func forward(y: string<secret>) -> string ! {} { sink(y) }
func caller(y: string<secret>) -> string ! {} { forward(y) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected one diagnostic, got %v", errs)
	}
}

func TestIFCScopedClosureReturn(t *testing.T) {
	src := `module test/closure
func closure(s: string<secret>) -> string<clean> ! {Declassify[label=email]} { \u. s }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 || errs[0].Kind != types.DeclassifyRequiredError {
		t.Fatal(errs)
	}
}
func TestIFCPositiveNestedDeclaredArgument(t *testing.T) {
	src := `module test/nested
type Payload = { raw: string<secret> }
func sink(x: Payload<email>) -> string ! {} { "ok" }
func caller(x: Payload) -> string ! {} { sink(x) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 || errs[0].Kind != types.ParamLabelCoverError {
		t.Fatal(errs)
	}
}
func TestIFCDuplicateDeclassifyRejected(t *testing.T) {
	for _, eff := range []string{"Declassify[label=email], Declassify[label=secret]", "Declassify, Declassify[label=email]"} {
		p := parser.New(lexer.New("module test/duplicate\nfunc f() -> string ! {"+eff+"} { \"ok\" }", "duplicate.ail"))
		p.ParseFile()
		if len(p.Errors()) == 0 || !strings.Contains(fmt.Sprint(p.Errors()), "PAR_EFF001_DUP") {
			t.Fatalf("expected duplicate diagnostic: %v", p.Errors())
		}
	}
}

func TestIFCScopedUnlabelledReturn(t *testing.T) {
	for _, body := range []string{"s", "let f = \\u. s in f(0)", "\\u. s"} {
		src := "module test/unlabelled\nfunc f(s: string<secret>) -> string ! {Declassify[label=email]} { " + body + " }"
		errs := types.CheckModuleIFC(parseIFC(t, src))
		if len(errs) != 1 || errs[0].Kind != types.DeclassifyRequiredError {
			t.Fatalf("scoped bottom laundering %s: %v", body, errs)
		}
	}
}

func TestIFCScopedUnlabelledParameterCannotLaunder(t *testing.T) {
	for _, arg := range []string{"s", "\\u. s"} {
		src := `module test/actual
func reveal(x: string) -> string ! {Declassify[label=email]} { x }
func sink(x: string{not secret}) -> string ! {} { x }
func caller(s: string<secret>) -> string ! {Declassify[label=email]} { sink(reveal(` + arg + `)) }`
		errs := types.CheckModuleIFC(parseIFC(t, src))
		if len(errs) != 2 || errs[0].Kind != types.SinkRefinementError || errs[1].Kind != types.DeclassifyRequiredError {
			t.Fatalf("scoped actual argument %s lost label: %v", arg, errs)
		}
	}
}

func TestIFCScopedTypedRecordArgumentKeepsDeepLabel(t *testing.T) {
	src := `module test/deep_actual
type Payload = { raw: string<secret> }
func reveal(x: Payload) -> string ! {Declassify[label=email]} { "constant" }
func sink(x: string{not secret}) -> string ! {} { x }
func caller(s: Payload) -> string ! {Declassify[label=email]} { sink(reveal(s)) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 2 {
		t.Fatalf("type-matched record lost deep label: %v", errs)
	}
}
func TestIFCScopedUnlabelledParameterAuthorizedAndBareControls(t *testing.T) {
	for _, tc := range []struct{ eff, label string }{{"Declassify[label=email]", "email"}, {"Declassify", "secret"}} {
		src := fmt.Sprintf("module test/control\nfunc reveal(x: string) -> string ! {%s} { x }\nfunc caller(s: string<%s>) -> string<clean> ! {%s} { reveal(s) }", tc.eff, tc.label, tc.eff)
		if errs := types.CheckModuleIFC(parseIFC(t, src)); len(errs) != 0 {
			t.Fatal(errs)
		}
	}
}
