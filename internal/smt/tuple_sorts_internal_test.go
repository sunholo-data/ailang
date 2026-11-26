package smt

import "testing"

// A tuple value is shown and replayed as an AILANG tuple, its components at
// the sort's arguments (the datatype's own fields are type parameters).
func TestTupleModelValues(t *testing.T) {
	raw := `sat
(
  (define-fun $p_p () (Tuple2 Int String) (mk_Tuple2 (- 3) "a"))
  (define-fun $p_ps () (Seq (Tuple2 Int Int)) (seq.unit (mk_Tuple2 1 2)))
  (define-fun $p_t () (Tuple3 Bool Int Real) (mk_Tuple3 true 0 (/ 1.0 2.0)))
)`
	model := parseModel(raw)
	want := map[string]string{"p": `(-3, "a")`, "ps": "[(1, 2)]", "t": "(true, 0, 0.5)"}
	for _, b := range formatModelBindings(model, "") {
		if w, ok := want[b.Name]; ok && b.Value != w {
			t.Errorf("shown %s = %s, want %s", b.Name, b.Value, w)
		}
	}
	args, err := witnessArgs([]FunctionParam{{Name: "p"}, {Name: "ps"}, {Name: "t"}}, model, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range args {
		if got := a.Literal.String(); got != want[a.Name] {
			t.Errorf("replayed %s = %s, want %s", a.Name, got, want[a.Name])
		}
	}
}

// A value whose constructor does not fit its tuple sort is neither shown as a
// tuple nor replayed.
func TestTupleModelValueOfTheWrongArity(t *testing.T) {
	raw := `sat
(
  (define-fun $p_p () (Tuple2 Int Int) (mk_Tuple3 1 2 3))
)`
	model := parseModel(raw)
	for _, b := range formatModelBindings(model, "") {
		if b.Name == "p" && b.Value == "(1, 2, 3)" {
			t.Errorf("a 3-tuple value of a 2-tuple sort was shown as %s", b.Value)
		}
	}
	if _, err := witnessArgs([]FunctionParam{{Name: "p"}}, model, ""); err == nil {
		t.Error("a 3-tuple value of a 2-tuple sort was replayed")
	}
}

func TestTupleArity(t *testing.T) {
	for name, want := range map[string]int{"Tuple2": 2, "Tuple8": 8, "Tuple1": 0, "Tuple9": 0, "Tuple02": 0, "Tuple": 0, "Tagged": 0} {
		if got := tupleArity(name); got != want {
			t.Errorf("tupleArity(%q) = %d, want %d", name, got, want)
		}
	}
}

// An opaque helper is named only when the script applies it, never because a
// longer symbol starts with its name or because it is declared.
func TestReferencedOpaqueHelpers(t *testing.T) {
	hs := []OpaqueHelper{{Name: "f"}, {Name: "g"}, {Name: "h"}}
	decls := []string{
		"(declare-fun $spec_f (Int) Int)",
		"(declare-fun $spec_g () Int)",
		"(declare-fun $spec_h (Int) Int)",
		"(define-fun k (($a_x Int)) Int ($spec_foo $a_x))",
	}
	assertions := []string{"(assert (= result (+ $spec_g 1)))"}
	got := referencedOpaqueHelpers(hs, decls, assertions)
	if len(got) != 1 || got[0].Name != "g" {
		t.Fatalf("referenced = %+v, want only g", got)
	}
}
