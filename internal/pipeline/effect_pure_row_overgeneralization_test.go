package pipeline

// M-EFFECT-PURE-ROW-OVERGENERALIZATION (#1091).
//
// A function declared `pure` must export the CLOSED empty effect row that `pure`
// promises. Today, if its body calls a recursive function, the effect row never
// resolves — a recursive self-call deliberately shares (and does not bind) the
// enclosing function's effect-row variable, see the `inferApp` comment in
// internal/types/typechecker_functions.go — so with an empty declared row there is
// nothing concrete to close it against. `generalizeWithConstraints` then quantifies
// that leftover variable without ever consulting the declaration, and the module
// exports `(string, string) -> int ! {...ρ2}` with RowVars=[ρ2].
//
// An importer instantiates that row fresh, it unifies with whatever effect the
// surrounding inference supplies, and the effect checker reports the CALLER as
// requiring an effect nothing in the program performs ("Missing effects: FS" in
// #1091). The exported row is the defect; the false effect is one downstream
// symptom of it.
//
// These tests assert on the exported *Scheme*, because that is the only level where
// the defect is observable:
//   - `ailang check` shows only the downstream symptom, and reproducing that symptom
//     requires a large tangled module (the reporter failed to minimize it across four
//     attempts; a dependency-aware delta-debugger converged at 37/37 declarations
//     retained). That end-to-end reproduction is the sprint's M3 acceptance artifact.
//   - `ailang iface` is NOT usable here: its JSON projection flattens the row variable
//     away and reports a row-polymorphic export as `"effects": [], "pure": true`,
//     indistinguishable from a genuinely pure one.
//
// Both directions are pinned, so the fix cannot pass by simply closing every row:
// deliberate row polymorphism (`! {e}`) and concrete declared rows (`! {IO}`) must
// survive untouched.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// buildIface writes the given {relpath: content} files under a temp dir, runs the
// pipeline in ModeCheck against entry, and returns the resulting module interface.
func buildIface(t *testing.T, files map[string]string, entry string) *iface.Iface {
	t.Helper()

	tempDir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(tempDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	res, err := Run(Config{Mode: ModeCheck}, Source{Filename: entry})
	if err != nil {
		t.Fatalf("compiling %s failed: %v", entry, err)
	}
	if res.Interface == nil {
		t.Fatalf("compiling %s produced no module interface", entry)
	}
	return res.Interface
}

// exportedScheme returns the exported scheme for name, failing if absent.
func exportedScheme(t *testing.T, ifc *iface.Iface, name string) *iface.IfaceItem {
	t.Helper()
	item, ok := ifc.Exports[name]
	if !ok {
		t.Fatalf("module %s does not export %q (exports: %v)", ifc.Module, name, exportNames(ifc))
	}
	if item.Type == nil {
		t.Fatalf("export %q has a nil type scheme", name)
	}
	return item
}

func exportNames(ifc *iface.Iface) []string {
	names := make([]string, 0, len(ifc.Exports))
	for n := range ifc.Exports {
		names = append(names, n)
	}
	return names
}

// recursiveHelperModule is the #1091 shape, reduced to the one property that
// triggers it: an `export pure func` whose body delegates to a PRIVATE RECURSIVE
// scan. This mirrors docparse/services/pkg_template.pkgLastIndexOf.
const recursiveHelperModule = `module helper

import std/string (length, find, substring)

export pure func hLast(hay: string, needle: string) -> int =
  hScan(hay, needle, 0, -1)

pure func hScan(hay: string, needle: string, offset: int, best: int) -> int {
  let i = if offset >= length(hay) then -1
          else find(substring(hay, offset, length(hay)), needle);
  if i < 0 then best else hScan(hay, needle, offset + i + 1, offset + i)
}
`

// TestPureExport_RecursiveBody_HasNoQuantifiedEffectRow is the core regression.
//
// RED before the fix: RowVars=[ρ2], Type=(string, string) -> int ! {...ρ2}.
// GREEN after:        RowVars=[],   Type=(string, string) -> int
func TestPureExport_RecursiveBody_HasNoQuantifiedEffectRow(t *testing.T) {
	ifc := buildIface(t, map[string]string{"helper.ail": recursiveHelperModule}, "helper.ail")
	item := exportedScheme(t, ifc, "hLast")

	if len(item.Type.RowVars) != 0 {
		t.Fatalf("`pure func hLast` exported an EFFECT-POLYMORPHIC row: RowVars=%v, type=%s\n"+
			"A function declared `pure` must export the closed empty row. The quantified row "+
			"variable is instantiated fresh at every import and absorbs whatever effect the "+
			"importer's inference supplies — that is #1091, where a pure caller was told it was "+
			"\"Missing effects: FS\".",
			item.Type.RowVars, item.Type.Type)
	}
}

// TestPureExport_NonRecursiveBody_StaysClosed pins the control: the SAME shape with a
// non-recursive delegate already exports a closed row today. GREEN before and after —
// it proves the recursive case above fails for the reason claimed (recursion), and not
// because every cross-module `pure` export is broken.
func TestPureExport_NonRecursiveBody_StaysClosed(t *testing.T) {
	const src = `module helper

import std/string (find)

export pure func hLast(hay: string, needle: string) -> int = hFind(hay, needle)

pure func hFind(hay: string, needle: string) -> int = find(hay, needle)
`
	ifc := buildIface(t, map[string]string{"helper.ail": src}, "helper.ail")
	item := exportedScheme(t, ifc, "hLast")

	if len(item.Type.RowVars) != 0 {
		t.Fatalf("control case regressed: a non-recursive `pure func` exported RowVars=%v (type=%s); "+
			"this shape is closed at base, so the recursive test above is no longer isolating recursion",
			item.Type.RowVars, item.Type.Type)
	}
}

// TestDeclaredRowPolymorphicExport_KeepsItsRowVar pins the other direction: a
// DELIBERATELY effect-polymorphic signature must keep its quantified row variable.
//
// This is the shape of std/list.mapE (`mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e}`)
// and of the 12 other row-variable signatures shipped in std/. The fix must
// discriminate on the DECLARATION — a declared row variable stays polymorphic — and
// never on the row variable's name, which would break the first time a user writes a
// row named like a compiler-generated one.
//
// GREEN before and after. If the fix closes this row, imported uses stop being
// independently instantiable, which is the #386 regression this must not reintroduce.
func TestDeclaredRowPolymorphicExport_KeepsItsRowVar(t *testing.T) {
	const src = `module rowpoly

export func applyE[a, b, e](f: a -> b ! {e}, x: a) -> b ! {e} = f(x)
`
	ifc := buildIface(t, map[string]string{"rowpoly.ail": src}, "rowpoly.ail")
	item := exportedScheme(t, ifc, "applyE")

	if len(item.Type.RowVars) == 0 {
		t.Fatalf("declared row-polymorphic export `applyE[a, b, e](...) ! {e}` lost its quantified "+
			"row variable (RowVars=%v, type=%s). Effect polymorphism declared in the signature must "+
			"survive: without a quantified row var, separate imported uses share one row identity "+
			"(#386 Section B).",
			item.Type.RowVars, item.Type.Type)
	}
}

// TestConcreteEffectExport_RecursiveBody_StaysClosed pins the asymmetry that makes
// this a false REJECTION rather than a soundness hole: a recursive function with a
// CONCRETE declared row closes normally, because the recursive call unifies against
// the declared effect. GREEN before and after.
func TestConcreteEffectExport_RecursiveBody_StaysClosed(t *testing.T) {
	const src = `module loud

import std/io (println)

export func hLoud(n: int) -> int ! {IO} = hScan(n)

func hScan(n: int) -> int ! {IO} {
  let _ = println("tick");
  if n <= 0 then 0 else hScan(n - 1)
}
`
	ifc := buildIface(t, map[string]string{"loud.ail": src}, "loud.ail")
	item := exportedScheme(t, ifc, "hLoud")

	if len(item.Type.RowVars) != 0 {
		t.Fatalf("a recursive `! {IO}` export gained a quantified row var (RowVars=%v, type=%s); "+
			"a concrete declared row must stay closed",
			item.Type.RowVars, item.Type.Type)
	}
}

// TestStdlib_NoUnsharedQuantifiedEffectRows sweeps the whole stdlib for the defect
// shape: an exported function whose OWN effect row ends in a quantified variable that
// appears nowhere else in its type. Such a row is pure over-generalization — there is
// no callback for it to be polymorphic WITH — and every one of them is a latent #1091.
//
// The criterion is deliberately "unshared", not "absent". Measuring the stdlib showed
// six exports that legitimately keep a quantified row, all of them higher-order:
// std/option.map, .filter, .flatMap and std/result.map, .mapErr, .flatMap. Note that
// std/option.flatMap is declared `export pure func` and STILL carries an open outer
// row — shared with its callback. That is not a bug: `pure` constrains the function's
// own effects, while the shared row is what makes the combinator effect-transparent
// with respect to the callback the caller supplies. Demanding "no pure export has a
// quantified row" would require breaking those.
//
// NOTE ON WHAT THIS TEST IS: it is a FORWARD guard, not a #1091 regression test. It
// was verified to pass with the fix disabled — no stdlib module carries the defect
// shape today, because nothing in std/ pairs a `pure` declaration with a recursive
// body that has no callback. Its job is to stop a newly-added stdlib export from
// acquiring an over-generalized row unnoticed. The tests that actually go red without
// the fix are TestPureExport_RecursiveBody_HasNoQuantifiedEffectRow (the defect) and
// TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback (the over-broad fix).
func TestStdlib_NoUnsharedQuantifiedEffectRows(t *testing.T) {
	stdDir := findStdDir(t)

	var offenders []string
	err := filepath.Walk(stdDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".ail") {
			return nil
		}
		res, runErr := Run(Config{Mode: ModeCheck}, Source{Filename: p})
		if runErr != nil || res.Interface == nil {
			return nil // modules that need capabilities/deps are covered elsewhere
		}
		for name, item := range res.Interface.Exports {
			if item.Type == nil || len(item.Type.RowVars) == 0 {
				continue
			}
			fn, ok := item.Type.Type.(*types.TFunc2)
			if !ok || fn.EffectRow == nil || fn.EffectRow.Tail == nil {
				continue
			}
			tail := fn.EffectRow.Tail.Name
			elsewhere := types.FreeEffectRowVarNames(&types.TFunc2{
				Params: fn.Params, Return: fn.Return, EffectRow: types.EmptyEffectRow(),
			})
			shared := false
			for _, n := range elsewhere {
				if n == tail {
					shared = true
				}
			}
			if !shared {
				offenders = append(offenders, fmt.Sprintf("%s.%s : %s", res.Interface.Module, name, item.Type.Type))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", stdDir, err)
	}

	if len(offenders) > 0 {
		t.Fatalf("stdlib exports carry an UNSHARED quantified effect row (over-generalized, "+
			"latent #1091 — the row has no callback to be polymorphic with, so importers will "+
			"instantiate it fresh and absorb unrelated effects):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// findStdDir locates the repo's std/ directory from the test's working directory.
func findStdDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		candidate := filepath.Join(dir, "std")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("std/ directory not found from the test working directory")
	return ""
}

// TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback pins the case that a
// first, over-broad version of this fix actually broke.
//
// A higher-order function can be effect-polymorphic WITHOUT declaring `! {e}`: its
// callback row and its result row simply share an INFERRED row variable, and that
// sharing is what lets callers pass an effectful lambda. `std/list.flatMap` is exactly
// this — no effect annotation at all — and `docparse/services/epub_parser` calls it as
// `flatMap(\entry. epubParseContentFile(filepath, entry), contentFiles)` with an
// `! {FS}` lambda.
//
// The first attempt closed the declared-empty outer row by substituting ACROSS THE
// WHOLE TYPE, which also closed the shared callback row and made every such combinator
// strictly pure. That failed with:
//
//	failed to unify effect rows: incompatible closed rows:
//	r1 has extra labels [], r2 has extra labels [FS]
//
// so closure is now restricted to rows whose variable occurs nowhere else in the type.
// GREEN before the fix and after; RED for the over-broad version.
func TestInferredRowPolymorphicCombinator_AcceptsEffectfulCallback(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

import std/list (flatMap)
import std/fs (readFile)
import std/io (println)

func readOne(p: string) -> [string] ! {FS} = [readFile(p)]

export func gather(paths: [string]) -> [string] ! {FS} =
  flatMap(\p. readOne(p), paths)

export func main() -> () ! {IO, FS} = println("ok")
`,
	}

	if err := checkModules(t, files); err != nil {
		t.Fatalf("passing an `! {FS}` lambda to the inferred-row-polymorphic `std/list.flatMap` "+
			"was REJECTED: %v\n"+
			"flatMap declares no effect annotation, but its callback and result rows share an "+
			"inferred row variable — that sharing IS its effect polymorphism. Closing a "+
			"declared-empty row must never close a row variable that occurs elsewhere in the type.",
			err)
	}
}

// TestPureImporter_StillRejectedForGenuineEffect is the negative gate. A `pure`
// function calling a genuinely effectful cross-module function must STILL be
// rejected after the fix. Without this, "close every unresolved row" would pass every
// other test in this file while silently laundering real effects.
//
// GREEN before and after.
func TestPureImporter_StillRejectedForGenuineEffect(t *testing.T) {
	files := map[string]string{
		"helper.ail": `module helper

import std/io (println)

export func hLoud(n: int) -> int ! {IO} = hScan(n)

func hScan(n: int) -> int ! {IO} {
  let _ = println("tick");
  if n <= 0 then 0 else hScan(n - 1)
}
`,
		"main.ail": `module main

import std/io (println)
import helper (hLoud)

pure func cPure(n: int) -> int = hLoud(n)

export func main() -> () ! {IO} = println("${cPure(2)}")
`,
	}

	err := checkModules(t, files)
	if err == nil {
		t.Fatal("a `pure` function calling a cross-module `! {IO}` function was ACCEPTED; " +
			"effect requirements must still cross module boundaries — closing unresolved effect " +
			"rows must never close a row carrying a concrete effect label")
	}
}
