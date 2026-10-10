package pipeline

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func TestImportedCallbackConcreteEffects(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	const host = `export type Host = { dispatch: (string, string, string) -> bool ! {FS, Process, Clock} }
`
	for _, imported := range []bool{false, true} {
		name := "inline"
		if imported {
			name = "imported"
		}
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct{ name, row, missing string }{
				{"declared", " ! {FS, Process, Clock}", ""},
				{"superset", " ! {FS, Process, Clock, Net}", ""},
				{"missing_process", " ! {FS, Clock}", "Process"},
				{"pure", "", "Clock, FS, Process"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					files := map[string]string{"main.ail": "module main\n"}
					if imported {
						files["minimal_abi.ail"] = "module minimal_abi\n" + host
						files["main.ail"] += "import minimal_abi (Host)\n"
					} else {
						files["main.ail"] += host
					}
					files["main.ail"] += "export func handle(h: Host) -> bool" + tt.row + ` {
  h.dispatch("site-rda", "poster", "request")
}
`
					err := checkModules(t, files)
					if tt.missing == "" {
						if err != nil {
							t.Fatalf("declared callback effects rejected: %v", err)
						}
						return
					}
					if err == nil || !strings.Contains(err.Error(), "function 'handle'") || !strings.Contains(err.Error(), "Missing effects: "+tt.missing) {
						t.Fatalf("expected missing %s on handle, got %v", tt.missing, err)
					}
					if strings.Contains(err.Error(), "Unresolved effect tail") {
						t.Fatalf("concrete callback introduced a spurious tail: %v", err)
					}
				})
			}
		})
	}
}

func TestAliasBodyClosurePreservesCallbackContracts(t *testing.T) {
	for _, tt := range []struct {
		name               string
		implicit, concrete bool
	}{
		{"implicit", true, false},
		{"concrete", false, true},
		{"polymorphic", false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			callback := &types.TFunc2{
				Params:                 []types.Type{types.TString},
				Return:                 types.TBool,
				ImplicitEffectContract: tt.implicit,
				ConcreteEffectContract: tt.concrete,
				EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"Process": types.TUnit},
					Tail: &types.RowVar{Name: "e", Kind: types.EffectRow}},
			}
			input := &types.TRecord{Fields: map[string]types.Type{"dispatch": callback}}
			got := newAliasBodyCloser(nil, nil).walk(input).(*types.TRecord)
			if !reflect.DeepEqual(got, input) {
				t.Fatalf("interface closure changed callback contract: got %#v, want %#v", got.Fields["dispatch"], callback)
			}
			if got == input || got.Fields["dispatch"] == callback {
				t.Fatal("interface closure must rebuild without modifying shared input types")
			}
		})
	}
}
