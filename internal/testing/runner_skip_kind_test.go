package testing

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestRunProperty_SkipTaxonomyIsTotal(t *testing.T) {
	runner := NewRunner("test.ail")
	cases := []struct {
		name string
		run  func() PropertyResult
		want string
	}{
		{
			name: "forall no generator",
			run: func() PropertyResult {
				return runner.runProperty(PropertyCase{
					Name: "forall",
					Property: &ast.Property{Binders: []*ast.Binder{
						{Name: "tree", Type: &ast.SimpleType{Name: "Tree"}},
					}},
				})
			},
			want: SkipKindNoGenerator,
		},
		{
			name: "ensures unsupported",
			run: func() PropertyResult {
				return runner.runProperty(PropertyCase{
					Name:     "ensures",
					Property: &ast.Property{Kind: ast.EnsuresKind},
				})
			},
			want: SkipKindUnsupported,
		},
		{
			name: "requires unsupported",
			run: func() PropertyResult {
				return runner.runProperty(PropertyCase{
					Name:     "requires",
					Property: &ast.Property{Kind: ast.RequiresKind},
				})
			},
			want: SkipKindUnsupported,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.run()
			if got.Status != StatusSkip {
				t.Fatalf("expected skip, got %s", got.Status)
			}
			if got.SkipKind != tc.want {
				t.Fatalf("expected skip kind %q, got %q", tc.want, got.SkipKind)
			}
		})
	}
}

func TestRunContractProperties_SkipKinds(t *testing.T) {
	src := `module skip_kinds

export pure func unsupported(p: ImportedPoint) -> int
  requires { true }
  ensures { result > 0 }
{
  1
}

export pure func outOfContract(x: int) -> int
  requires { false }
{
  x
}

export func main() -> int ! {} { 0 }
`
	result := runEnsuresFromSource(t, src)
	if len(result.Properties) != 3 {
		t.Fatalf("expected 3 property results, got %d", len(result.Properties))
	}

	want := []string{
		SkipKindNoGenerator,
		SkipKindNoGenerator,
		SkipKindOutOfContract,
	}
	for i, prop := range result.Properties {
		if prop.Status != StatusSkip {
			t.Fatalf("property %d: expected skip, got %s (%s)", i, prop.Status, prop.Error)
		}
		if prop.SkipKind != want[i] {
			t.Fatalf("property %d: expected skip kind %q, got %q", i, want[i], prop.SkipKind)
		}
	}
}
