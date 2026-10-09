package types

import "testing"

func TestDeclassifyScopeValidation(t *testing.T) {
	for _, v := range []string{"email", "sqlsafe", "arg"} {
		if err := validateEffectParams("Declassify", map[string]string{"label": v}); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []map[string]string{{"mode": "email"}, {"label": ""}, {"label": "*"}, {"label": "a|b"}, {"label": "a b"}} {
		if validateEffectParams("Declassify", p) == nil {
			t.Fatalf("accepted %v", p)
		}
	}
}
func TestDeclassifyDirectionalCoverage(t *testing.T) {
	row := func(scope string) *Row {
		r := &Row{Kind: EffectRow, Labels: map[string]Type{"Declassify": TUnit}}
		if scope != "" {
			r.Params = map[string]map[string]string{"Declassify": {"label": scope}}
		}
		return r
	}
	for _, tc := range []struct {
		required, declared string
		want               bool
	}{{"email", "", true}, {"email", "email", true}, {"", "email", false}, {"secret", "email", false}, {"email|secret", "email", false}, {"email|secret", "", true}} {
		if got := SubsumeEffectRows(row(tc.required), row(tc.declared)); got != tc.want {
			t.Errorf("%q -> %q: %v", tc.required, tc.declared, got)
		}
	}
	if effectParamsCompatible(row("email"), row(""), "Declassify") {
		t.Fatal("function effects must stay invariant")
	}
}
