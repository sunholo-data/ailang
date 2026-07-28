package pipeline

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

func TestEffectModeSubsumptionFinalMatrix(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
		want    string
	}{
		{"blocker", `module blocker
import std/rand (rand_int)
export func seeded_roll() -> int ! {Rand[mode=seeded]} = rand_int(1, 6)`, false, ""},
		{"c1", `module c1
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=os]} = rand_int(1, 6)`, false, ""},
		{"c2", `module c2
import std/rand (rand_int)
export func f() -> int ! {Rand[mode=crypto]} = rand_int(1, 6)`, false, ""},
		{"c3", `module c3
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand} = g(())`, true, "Effect mode mismatch: Rand requires mode=seeded; declaration provides mode=os"},
		{"c4", `module c4
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int = g(())`, true, "Missing effects: Rand"},
		{"c6", `module c6
import std/rand (rand_int)
export func g() -> int ! {Rand} = rand_int(1, 6)
export func f() -> int ! {Rand[mode=seeded]} = g(())`, false, ""},
		{"c7", `module c7
export func g() -> int ! {Rand[mode=seeded]} = 42
export func f() -> int ! {Rand[mode=os]} = g(())`, true, "Effect mode mismatch: Rand requires mode=seeded; declaration provides mode=os"},
		{"c8", `module c8
export func g() -> int ! {Rand[mode=crypto]} = 42
export func f() -> int ! {Rand} = g(())`, true, "Effect mode mismatch: Rand requires mode=crypto; declaration provides mode=os"},
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
			if err != nil && strings.Contains(err.Error(), "Missing effects:") &&
				!strings.Contains(err.Error(), "Missing effects: Rand") {
				t.Fatalf("rejection printed an empty/unknown missing-effects line: %q", err)
			}
			if err != nil && strings.Contains(err.Error(), "mode mismatch") &&
				strings.Contains(err.Error(), "Suggested fix:") {
				t.Fatalf("mode mismatch must not emit a suggested invalid row: %q", err)
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
				err := check386(t, "conflict_local", src)
				if declaredMode == "seeded" && err != nil {
					t.Fatalf("seeded must cover seeded+os requirements: %v", err)
				}
				if declaredMode == "os" && (err == nil ||
					!strings.Contains(err.Error(), "Rand requires mode=os|seeded; declaration provides mode=os")) {
					t.Fatalf("os conflict must name effect and modes: %v", err)
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
				err := check386(t, "conflict_imported", src)
				if declaredMode == "seeded" && err != nil {
					t.Fatalf("seeded must cover seeded+os requirements: %v", err)
				}
				if declaredMode == "os" && (err == nil ||
					!strings.Contains(err.Error(), "Rand requires mode=os|seeded; declaration provides mode=os")) {
					t.Fatalf("os conflict must name effect and modes: %v", err)
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

func TestLambdaModeMismatchUsesStructuredDiagnostic(t *testing.T) {
	err := formatLambdaEffectError("fixture:1:1", effectModeRow("seeded"), effectModeRow("os"))
	want := "Effect mode mismatch: Rand requires mode=seeded; declaration provides mode=os"
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("lambda diagnostic %q does not contain %q", err, want)
	}
	if strings.Contains(err.Error(), "Missing effects:") {
		t.Fatalf("lambda mode mismatch printed Missing effects: %q", err)
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
