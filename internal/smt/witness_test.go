package smt

import (
	"testing"
)

// The line reader cut `(define-fun xs () (Seq Int)` at the first space and
// printed "(Seq = Int"; the S-expression reader keeps compound sorts and
// multi-line values whole.
func TestParseModel_CompoundSortAndMultilineValue(t *testing.T) {
	out := `sat
(
  (define-fun $p_xs () (Seq Int)
    (seq.++ (seq.unit 1)
            (seq.unit (- 2))))
  (define-fun $p_s () String
    "a""b\u{e9}")
  (define-fun f ((x Int)) Int
    x)
)`
	m := parseModel(out)
	if len(m) != 2 {
		t.Fatalf("want 2 nullary bindings, got %+v", m)
	}
	if m[0].Sort != "(Seq Int)" || m[0].Value != "(seq.++ (seq.unit 1) (seq.unit (- 2)))" {
		t.Fatalf("xs = %q : %q", m[0].Value, m[0].Sort)
	}
}

func TestWitnessArgs_ConvertsModelValues(t *testing.T) {
	script := `(declare-datatype Record_age_name ((mk_Record_age_name (age Int) (name String))))`
	out := `sat
(
  (define-fun $p_xs () (Seq Int) (seq.++ (seq.unit 1) (seq.unit (- 2))))
  (define-fun $p_x () Real (/ 1.0 4.0))
  (define-fun $p_s () String "a""b\u{e9}")
  (define-fun $p_ok () Bool false)
  (define-fun $p_p () Record_age_name (mk_Record_age_name 30 "ann"))
  (define-fun $p_e () (Seq String) (as seq.empty (Seq String)))
)`
	params := []FunctionParam{{Name: "xs"}, {Name: "x"}, {Name: "s"}, {Name: "ok"}, {Name: "p"}, {Name: "e"}}
	args, err := witnessArgs(params, parseModel(out), script)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"[1, -2]", "0.25", `"a\"bé"`, "false", `{ age: 30, name: "ann" }`, "[]"}
	for i, a := range args {
		if got := a.Literal.String(); got != want[i] {
			t.Errorf("%s: %s, want %s", a.Name, got, want[i])
		}
	}
}

// A value the converter does not understand refuses replay; it is never
// guessed (a guessed input could "confirm" a violation that does not exist).
func TestWitnessArgs_RefusesUnknownShapes(t *testing.T) {
	out := `sat
(
  (define-fun $p_o () (Option Int) ((as Some (Option Int)) 3))
)`
	if _, err := witnessArgs([]FunctionParam{{Name: "o"}}, parseModel(out), ""); err == nil {
		t.Fatal("want a refusal for an Option value (constructors are not replayed yet)")
	}
	if _, err := witnessArgs([]FunctionParam{{Name: "missing"}}, parseModel(out), ""); err == nil {
		t.Fatal("want a refusal for a parameter absent from the model")
	}
}

func TestWitnessArgs_ExpandsModelLets(t *testing.T) {
	out := `sat
(
  (define-fun $p_xs () (Seq Int) (let ((a!1 (seq.unit 3))) (seq.++ a!1 a!1 (seq.unit 4))))
)`
	args, err := witnessArgs([]FunctionParam{{Name: "xs"}}, parseModel(out), "")
	if err != nil {
		t.Fatal(err)
	}
	if got := args[0].Literal.String(); got != "[3, 3, 4]" {
		t.Fatalf("got %s", got)
	}
}
