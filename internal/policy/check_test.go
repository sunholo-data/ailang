package policy

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func mkRow(labels []string, tail *types.RowVar) *types.Row {
	if len(labels) == 0 && tail == nil {
		return nil
	}
	m := make(map[string]types.Type, len(labels))
	for _, l := range labels {
		m[l] = types.Unit()
	}
	return &types.Row{
		Kind:   types.EffectRow,
		Labels: m,
		Tail:   tail,
	}
}

func TestCheck_PureAdmitted(t *testing.T) {
	p := &Policy{AllowedCaps: []string{}}
	d := Check(p, "main", nil)
	if !d.OK {
		t.Fatalf("pure should be admitted: %+v", d)
	}
}

func TestCheck_MonomorphicSubsetAdmitted(t *testing.T) {
	p := &Policy{AllowedCaps: []string{"Net", "Clock"}}
	row := mkRow([]string{"Net"}, nil)
	d := Check(p, "main", row)
	if !d.OK {
		t.Fatalf("Net ⊆ {Net,Clock} should admit: %+v", d)
	}
}

func TestCheck_PolicyViolationListsMissing(t *testing.T) {
	p := &Policy{AllowedCaps: []string{"Net"}}
	row := mkRow([]string{"FS", "Net"}, nil)
	d := Check(p, "main", row)
	if d.OK {
		t.Fatalf("{FS,Net} should not be admitted under {Net}")
	}
	if d.ErrorKind != KindPolicyViolation {
		t.Errorf("got error_kind=%q, want %q", d.ErrorKind, KindPolicyViolation)
	}
	if len(d.MissingFromPolicy) != 1 || d.MissingFromPolicy[0] != "FS" {
		t.Errorf("missing_from_policy = %v, want [FS]", d.MissingFromPolicy)
	}
}

func TestCheck_ParametricEntryRejected(t *testing.T) {
	// Open row {Net | e} — the e tail makes it parametric.
	p := &Policy{AllowedCaps: []string{"Net", "FS", "IO"}}
	tail := &types.RowVar{Name: "e", Kind: types.EffectRow}
	row := mkRow([]string{"Net"}, tail)
	d := Check(p, "main", row)
	if d.OK {
		t.Fatalf("parametric entry must be rejected even when concrete labels are subset")
	}
	if d.ErrorKind != KindParametricEntry {
		t.Errorf("got error_kind=%q, want %q", d.ErrorKind, KindParametricEntry)
	}
}

func TestCheck_DenyAllRejectsAnyEffect(t *testing.T) {
	p := DefaultPolicy() // AllowedCaps = []
	row := mkRow([]string{"IO"}, nil)
	d := Check(p, "main", row)
	if d.OK {
		t.Fatalf("deny-all should reject any effect")
	}
	if d.ErrorKind != KindPolicyViolation {
		t.Errorf("got %q, want %q", d.ErrorKind, KindPolicyViolation)
	}
}

func TestCheck_DeterministicLabelOrder(t *testing.T) {
	p := &Policy{AllowedCaps: []string{"Z", "A", "M"}}
	row := mkRow([]string{"Z", "A", "M"}, nil)
	d := Check(p, "main", row)
	if !d.OK {
		t.Fatalf("expected admit, got %+v", d)
	}
	want := []string{"A", "M", "Z"}
	if got := d.DeclaredEffects; !equalStrings(got, want) {
		t.Errorf("DeclaredEffects = %v, want sorted %v", got, want)
	}
	if got := d.AllowedCaps; !equalStrings(got, want) {
		t.Errorf("AllowedCaps = %v, want sorted %v", got, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
