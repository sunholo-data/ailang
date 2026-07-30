package testing

import "testing"

func TestAddPropertyResult_VacuousSkipClassification(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want int
	}{
		{name: "no generator", kind: SkipKindNoGenerator, want: 1},
		{name: "unsupported", kind: SkipKindUnsupported, want: 1},
		{name: "empty fails closed", kind: "", want: 1},
		{name: "out of contract forgiven", kind: SkipKindOutOfContract, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NewSuiteResult("test.ail")
			result.AddPropertyResult(PropertyResult{
				Status:   StatusSkip,
				SkipKind: tc.kind,
			})
			if result.VacuousSkips != tc.want {
				t.Fatalf("expected %d vacuous skips, got %d", tc.want, result.VacuousSkips)
			}
		})
	}
}
