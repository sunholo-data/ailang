package smt

import "testing"

func TestFormatModelBindings_AILANGSyntax(t *testing.T) {
	smtlib := `
(declare-datatype Option_Int ((None) (Some (Some_0 Int))))
(declare-datatype Record_age_name ((mk_Record_age_name (age Int) (name String))))
`
	raw := `sat
(
  (define-fun $p_xs () (Seq Int)
    (seq.++ (seq.unit (- 5)) (seq.unit 9223372036854775807)))
  (define-fun $p_empty () (Seq Int)
    (as seq.empty (Seq Int)))
  (define-fun $p_opt () Option_Int
    (Some 42))
  (define-fun $p_none () Option_Int
    None)
  (define-fun $p_rec () Record_age_name
    (mk_Record_age_name 36 "Ada"))
  (define-fun $p_frac () Real
    (/ (- 3) 2))
  (define-fun _map_spec_f_0$g () (Array Int Int)
    ((as const (Array Int Int)) 0))
  (define-fun result () Int
    9223372036854775808)
)`
	parsed := parseModel(raw)
	got := formatModelBindings(parsed, smtlib)

	want := map[string]string{
		"xs":     "[-5, 9223372036854775807]",
		"empty":  "[]",
		"opt":    "Some(42)",
		"none":   "None",
		"rec":    `{ age: 36, name: "Ada" }`,
		"frac":   "-1.5",
		"result": "9223372036854775808",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d bindings (%+v), want %d (Array stub should be filtered)", len(got), got, len(want))
	}
	for _, b := range got {
		if w, ok := want[b.Name]; !ok {
			t.Errorf("unexpected binding %q = %q", b.Name, b.Value)
		} else if b.Value != w {
			t.Errorf("binding %q: got %q, want %q", b.Name, b.Value, w)
		}
	}
}
