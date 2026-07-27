package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEffectModePreservation_PreRelaxationMatrix(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
		want    string
	}{
		{"blocker", `module blocker
import std/rand (rand_int)
export func seeded_roll() -> int ! {Rand[mode=seeded]} = rand_int(1, 6)`, true, ""},
		{"c1", `module c1
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=os]} = rand_int(1, 6)`, false, ""},
		{"c2", `module c2
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=crypto]} = rand_int(1, 6)`, true, ""},
		{"c3", `module c3
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand} = g(())`, true, ""},
		{"c4", `module c4
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int = g(())`, true, "Missing effects: Rand"},
		{"c6", `module c6
import std/rand (rand_int)
export func g() -> int ! {Rand} = rand_int(1, 6)
export func f() -> int ! {Rand[mode=seeded]} = g(())`, true, ""},
		{"c7", `module c7
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand[mode=os]} = g(())`, true, ""},
		{"c8", `module c8
export func g() -> int ! {Rand[mode=crypto]} = 42
export func f() -> int ! {Rand} = g(())`, true, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := check386(t, tc.name, tc.src)
			if tc.wantErr && err == nil {
				t.Fatal("expected effect validation failure")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation failure: %v", err)
			}
			if err != nil && tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestConflictingLocalModesAreOrderIndependent(t *testing.T) {
	for _, declaredMode := range []string{"os", "seeded"} {
		for _, order := range []string{"seeded_first", "os_first"} {
			t.Run(declaredMode+"/"+order, func(t *testing.T) {
				first, second := "seeded(())", "osRand(())"
				if order == "os_first" {
					first, second = second, first
				}
				src := `module conflict_local
export func seeded() -> int ! {Rand[mode=seeded]} = 1
export func osRand() -> int ! {Rand[mode=os]} = 2
export func caller() -> int ! {Rand[mode=` + declaredMode + `]} {
  let ignored = ` + first + `;
  ` + second + `
}`
				if err := check386(t, "conflict_local", src); err == nil {
					t.Fatal("strict M1 must reject conflicting seeded and os requirements")
				}
			})
		}
	}
}

func TestConflictingImportedAndLocalModesAreOrderIndependent(t *testing.T) {
	for _, declaredMode := range []string{"os", "seeded"} {
		for _, order := range []string{"local_first", "imported_first"} {
			t.Run(declaredMode+"/"+order, func(t *testing.T) {
				first, second := "seeded(())", "rand_int(1, 6)"
				if order == "imported_first" {
					first, second = second, first
				}
				src := `module conflict_imported
import std/rand (rand_int)
export func seeded() -> int ! {Rand[mode=seeded]} = 1
export func caller() -> int ! {Rand[mode=` + declaredMode + `]} {
  let ignored = ` + first + `;
  ` + second + `
}`
				if err := check386(t, "conflict_imported", src); err == nil {
					t.Fatal("strict M1 must reject conflicting imported os and local seeded requirements")
				}
			})
		}
	}
}

func TestRequirementUnionPreservesConflictingModesDeterministically(t *testing.T) {
	osRow := effectModeRow("os")
	seededRow := effectModeRow("seeded")
	for _, merged := range []*types.Row{
		unionRequiredEffectRows(osRow, seededRow),
		unionRequiredEffectRows(seededRow, osRow),
	} {
		if got := merged.Params["Rand"]["mode"]; got != "os|seeded" {
			t.Fatalf("conflicting modes collapsed; got %q, want os|seeded", got)
		}
	}
}

func TestDeclaredRowsRemainImmutableDuringCollection(t *testing.T) {
	limit, minimum := 7, 2
	declared, err := types.ElaborateEffectRowWithBudgets([]ast.EffectAnnotation{{
		Name:   "Rand",
		Budget: &limit,
		Min:    &minimum,
		Params: []ast.EffectParam{{Key: "mode", Value: "seeded"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	before := types.FormatEffectRow(declared)
	_ = unionRequiredEffectRows(cloneEffectRow(declared), effectModeRow("os"))
	_ = unionRequiredEffectRows(cloneEffectRow(declared), effectModeRow("crypto"))
	if after := types.FormatEffectRow(declared); after != before {
		t.Fatalf("stored declaration mutated: before %q, after %q", before, after)
	}
	if *declared.Budgets["Rand"] != limit || *declared.MinBudgets["Rand"] != minimum {
		t.Fatalf("stored budgets mutated: %#v %#v", declared.Budgets, declared.MinBudgets)
	}
	if got := declared.Params["Rand"]["mode"]; got != "seeded" {
		t.Fatalf("stored params mutated: got mode=%q", got)
	}
}

func TestEffectRowVariableImportsStillValidate(t *testing.T) {
	src := `module row_tail_guard
import std/io (println)
import std/list (mapE)

export func main() -> [int] ! {IO} =
  mapE(func(x: int) -> int ! {IO} { println(show(x)); x }, [1, 2])`
	if err := check386(t, "row_tail_guard", src); err != nil {
		t.Fatalf("std/list mapE row-variable declaration regressed: %v", err)
	}
}

func effectModeRow(mode string) *types.Row {
	return &types.Row{
		Kind:   types.EffectRow,
		Labels: map[string]types.Type{"Rand": types.Unit()},
		Params: map[string]map[string]string{"Rand": {"mode": mode}},
	}
}
