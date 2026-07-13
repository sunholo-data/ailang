package pipeline

import (
	"strings"
	"testing"
)

// M-XMOD-ALIAS-POLY end-to-end regression pack.
//
// Parameterized type aliases (`type Box[a] = { items: [a] }`) must substitute
// their concrete arguments at use sites (`Box[int]`), single-module AND
// cross-module, while genuine parameterized ADTs (Option[a], Result[a,b], user
// `type Tree[a]`) stay strictly nominal. Uses the checkModules harness from
// cross_module_nonrecord_alias_test.go.

// --- Single-module positives ------------------------------------------------

func TestAliasPolyE2E_RecordSingleModule(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Box[a] = { items: [a] }

export func mk(xs: [int]) -> Box[int] { { items: xs } }

export func main() -> int { let b = mk([1, 2, 3]); 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Box[int] single-module failed to type-check: %v", err)
	}
}

func TestAliasPolyE2E_BareParamSingleModule(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Ident[a] = a

export func use(x: int) -> Ident[int] { x }

export func main() -> int { use(5) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Ident[int] single-module failed to type-check: %v", err)
	}
}

func TestAliasPolyE2E_TupleSingleModule(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Pair[a, b] = (a, b)

export func mk(x: int, y: string) -> Pair[int, string] { (x, y) }

export func main() -> int { let p = mk(1, "a"); 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Pair[int, string] single-module failed to type-check: %v", err)
	}
}

func TestAliasPolyE2E_FunctionSingleModule(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Fn[a, b] = (a) -> b

export func mk(f: (int) -> int) -> Fn[int, int] { f }

export func main() -> int { 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Fn[int, int] single-module failed to type-check: %v", err)
	}
}

func TestAliasPolyE2E_NestedSingleModule(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Box[a] = { items: [a] }

export func wrap(x: Box[int]) -> Box[Box[int]] { { items: [x] } }

export func main() -> int { 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Box[Box[int]] nested single-module failed to type-check: %v", err)
	}
}

// --- Arity mismatch ---------------------------------------------------------

func TestAliasPolyE2E_ArityMismatch(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Box[a] = { items: [a] }

export func mk(xs: [int]) -> Box[int, string] { { items: xs } }

export func main() -> int { 0 }
`,
	}
	err := checkModules(t, files)
	if err == nil {
		t.Fatal("Box[int, string] (arity 2 on 1-param alias) must fail")
	}
	if !strings.Contains(err.Error(), "TC_ALIAS_ARITY_001") {
		t.Errorf("expected coded TC_ALIAS_ARITY_001, got: %v", err)
	}
	if !strings.Contains(err.Error(), "but 2 provided") {
		t.Errorf("expected directional 'but 2 provided', got: %v", err)
	}
}

// --- Critical non-regression: real ADTs stay nominal ------------------------

// TestAliasPolyE2E_OptionStaysNominal: importing and using Option[int] must
// still work nominally (it is an ADT, absent from aliasEnv — expansion must not
// fire). This is the baseline that would break if keying were loose.
func TestAliasPolyE2E_OptionStaysNominal(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

import std/option (Option, Some, None)

export func wrap(x: int) -> Option[int] { Some(x) }

export func main() -> int { let o = wrap(5); 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("Option[int] must stay nominal and type-check, got: %v", err)
	}
}

// TestAliasPolyE2E_UserTreeADTStaysNominal: a user parameterized ADT with match.
// Expansion must NOT fire (Tree is registered as constructors, not an alias).
func TestAliasPolyE2E_UserTreeADTStaysNominal(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Tree[a] = Leaf(a) | Node(Tree[a], Tree[a])

export func size(t: Tree[int]) -> int {
    match t {
        Leaf(_) => 1,
        Node(_, _) => 2
    }
}

export func main() -> int { size(Leaf(5)) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("user Tree[a] ADT must stay nominal and type-check, got: %v", err)
	}
}

// TestAliasPolyE2E_TreeNotStructural: a bare int must NOT unify with Tree[int]
// — proves the ADT was not accidentally instantiated to its (nonexistent) body.
func TestAliasPolyE2E_TreeNotStructural(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Tree[a] = Leaf(a) | Node(Tree[a], Tree[a])

export func bad(x: int) -> Tree[int] { x }

export func main() -> int { 0 }
`,
	}
	if err := checkModules(t, files); err == nil {
		t.Fatal("REGRESSION: int unified with Tree[int] — ADT was wrongly instantiated")
	}
}

// --- PR #380 ordering lock: open-record over a parameterized alias body -----

// TestAliasPolyE2E_OpenRecordOverAliasBody: field access (which produces an open
// record / TRecordOpen) against a value whose type is a parameterized-alias
// record body must still unify. Expansion runs at Unify entry, BEFORE the type-
// switch dispatches to unifyRecord (M-LAMBDA-OPEN-RECORD-PATTERN, PR #380), so
// the open-record logic already sees the expanded { items: [int] }.
func TestAliasPolyE2E_OpenRecordOverAliasBody(t *testing.T) {
	files := map[string]string{
		"main.ail": `module main

type Box[a] = { items: [a], count: int }

export func mk(xs: [int]) -> Box[int] { { items: xs, count: 0 } }

export func getCount(b: Box[int]) -> int { b.count }

export func main() -> int { getCount(mk([1, 2, 3])) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("field access over parameterized-alias record body failed: %v", err)
	}
}

// --- M4: cross-module ------------------------------------------------------

// TestAliasPolyE2E_CrossModuleRecord: module shapes exports `type Box[a]`; main
// imports and uses Box[int]. Proves iface AliasParams threading composes with
// the M-XMOD-ALIAS cross-module path.
func TestAliasPolyE2E_CrossModuleRecord(t *testing.T) {
	files := map[string]string{
		"shapes/box.ail": `module shapes/box

export type Box[a] = { items: [a] }
`,
		"main.ail": `module main

import shapes/box (Box)

export func mk(xs: [int]) -> Box[int] { { items: xs } }

export func main() -> int { let b = mk([1, 2, 3]); 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("cross-module Box[int] failed to type-check: %v", err)
	}
}

// TestAliasPolyE2E_CrossModuleArityMismatch: an arity mismatch on an imported
// parameterized alias still emits the coded diagnostic.
func TestAliasPolyE2E_CrossModuleArityMismatch(t *testing.T) {
	files := map[string]string{
		"shapes/box.ail": `module shapes/box

export type Box[a] = { items: [a] }
`,
		"main.ail": `module main

import shapes/box (Box)

export func mk(xs: [int]) -> Box[int, string] { { items: xs } }

export func main() -> int { 0 }
`,
	}
	err := checkModules(t, files)
	if err == nil {
		t.Fatal("cross-module Box[int, string] must fail with arity error")
	}
	if !strings.Contains(err.Error(), "TC_ALIAS_ARITY_001") {
		t.Errorf("expected coded TC_ALIAS_ARITY_001 cross-module, got: %v", err)
	}
}
