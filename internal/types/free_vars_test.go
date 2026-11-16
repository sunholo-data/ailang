package types

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
)

// typeImplementations parses this package's non-test sources and returns the
// name of every type with a Substitute(map[string]Type) Type method: every
// implementation of Type.
func typeImplementations(t *testing.T) []string {
	t.Helper()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var out []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != "Substitute" || len(fn.Recv.List) != 1 {
				continue
			}
			recv := fn.Recv.List[0].Type
			if star, ok := recv.(*ast.StarExpr); ok {
				recv = star.X
			}
			if id, ok := recv.(*ast.Ident); ok {
				out = append(out, id.Name)
			}
		}
	}
	sort.Strings(out)
	return out
}

// switchCases returns the type names T the named function of file dispatches
// on: `case *T` clauses of its type switches and `x.(*T)` assertions.
func switchCases(t *testing.T, file, funcName string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), file, src, 0)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName {
			continue
		}
		ast.Inspect(fn, func(n ast.Node) bool {
			if ta, ok := n.(*ast.TypeAssertExpr); ok {
				if star, ok := ta.Type.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok {
						cases[id.Name] = true
					}
				}
				return true
			}
			cc, ok := n.(*ast.CaseClause)
			if !ok {
				return true
			}
			for _, e := range cc.List {
				if star, ok := e.(*ast.StarExpr); ok {
					if id, ok := star.X.(*ast.Ident); ok {
						cases[id.Name] = true
					}
				}
			}
			return true
		})
	}
	return cases
}

// varUnder wraps one type variable "v" and one record row "ρ" in each Type
// constructor. Keyed by constructor name; the test below fails when a Type
// implementation has no entry, so a new constructor cannot be added without
// deciding how its variables are found.
var varUnder = map[string]func() Type{
	"TVar":  func() Type { return &TVar{Name: "v"} },
	"TVar2": func() Type { return &TVar2{Name: "v", Kind: Star} },
	"TCon":  func() Type { return &TTuple{Elements: []Type{TInt, &TVar2{Name: "v", Kind: Star}}} }, // a leaf; carried by a tuple
	"RowVar": func() Type {
		return &TTuple{Elements: []Type{&TVar2{Name: "v", Kind: Star}, &RowVar{Name: "ρ", Kind: RecordRow}}}
	},
	"Row": func() Type {
		return &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"f": &TVar2{Name: "v", Kind: Star}}, Tail: &RowVar{Name: "ρ", Kind: RecordRow}}}
	},
	"TFunc2": func() Type {
		return &TFunc2{Params: []Type{&TVar2{Name: "v", Kind: Star}}, Return: TInt, EffectRow: &Row{Kind: EffectRow, Labels: map[string]Type{}, Tail: &RowVar{Name: "ε", Kind: EffectRow}}}
	},
	"TList":  func() Type { return &TList{Element: &TVar2{Name: "v", Kind: Star}} },
	"TArray": func() Type { return &TArray{Element: &TVar2{Name: "v", Kind: Star}} },
	"TMap":   func() Type { return &TMap{Key: TString, Value: &TVar2{Name: "v", Kind: Star}} },
	"TTuple": func() Type { return &TTuple{Elements: []Type{&TVar2{Name: "v", Kind: Star}}} },
	"TApp": func() Type {
		return &TApp{Constructor: &TCon{Name: "Option"}, Args: []Type{&TVar2{Name: "v", Kind: Star}}}
	},
	"TRecord": func() Type {
		return &TRecord{Fields: map[string]Type{"name": &TVar2{Name: "v", Kind: Star}}, Row: &RowVar{Name: "ρ", Kind: RecordRow}}
	},
	"TRecord2": func() Type {
		return &TRecord2{Row: &Row{Kind: RecordRow, Labels: map[string]Type{"name": &TVar2{Name: "v", Kind: Star}}}}
	},
	"TRecordOpen": func() Type {
		return &TRecordOpen{Fields: map[string]Type{"name": &TVar2{Name: "v", Kind: Star}}, Row: &RowVar{Name: "ρ", Kind: RecordRow}}
	},
	"TLabelled": func() Type { return &TLabelled{Inner: &TList{Element: &TVar2{Name: "v", Kind: Star}}} },
}

// The bug behind the PR #2209 review finding: the environment walk and the
// generalisation walk disagreed about which type forms contain variables. Both
// are now filters over typeVarOccurrences. This test pins that, per
// constructor: whatever variable generalisation would quantify in a type, a
// binding of that type in the environment owns.
func TestFreeVars_EveryTypeConstructorIsWalked(t *testing.T) {
	impls := typeImplementations(t)
	if len(impls) < 15 {
		t.Fatalf("found only %d Type implementations (%v); the source scan is broken", len(impls), impls)
	}
	walked := switchCases(t, "free_vars.go", "walkType")
	renamed := switchCases(t, "scheme_canonical.go", "renameSchemeVars")
	for _, name := range impls {
		if !walked[name] {
			t.Errorf("varWalker.walkType has no case for *%s", name)
		}
		if !renamed[name] && name != "TCon" { // TCon is a leaf (the default arm)
			t.Errorf("renameSchemeVars has no case for *%s", name)
		}
		mk, ok := varUnder[name]
		if !ok {
			t.Errorf("varUnder has no entry for *%s: add one", name)
			continue
		}
		typ := mk()
		typeVars, recordRows := generalizableVars(typ)
		effectRows := freeEffectRowVarsInType(typ)
		if !slices.Contains(typeVars, "v") {
			t.Errorf("*%s: generalisation does not see the type variable: %v", name, typeVars)
		}
		if !freeTypeVars(typ)["v"] {
			t.Errorf("*%s: freeTypeVars does not see the type variable", name)
		}
		env := NewTypeEnv().Extend("p", typ)
		owned := env.ownedVars()
		for _, v := range typeVars {
			if !owned.types[v] {
				t.Errorf("*%s: type variable %s is generalisable in %s but not owned by an environment binding it", name, v, typ)
			}
		}
		for _, v := range recordRows {
			if !owned.rows[v] {
				t.Errorf("*%s: record row %s is generalisable in %s but not owned by an environment binding it", name, v, typ)
			}
		}
		rowWalk, effectWalk := map[string]bool{}, map[string]bool{}
		env.collectFreeRowVars(rowWalk)
		env.collectFreeEffectRowVars(effectWalk)
		if !sameVarSet(rowWalk, owned.rows) || !sameVarSet(effectWalk, owned.effectRows) {
			t.Errorf("*%s: single-set walks disagree with the cached sets: rows %v vs %v, effect rows %v vs %v", name, rowWalk, owned.rows, effectWalk, owned.effectRows)
		}
		if r, ok := typ.(*TRecord2); ok && name == "Row" && !freeRowVars(r.Row)["ρ"] {
			t.Errorf("*%s: freeRowVars does not see the row tail", name)
		}
		for v := range effectRows {
			if !owned.effectRows[v] || !owned.rows[v] {
				t.Errorf("*%s: effect row %s is generalisable in %s but not owned by an environment binding it", name, v, typ)
			}
		}
		// A scheme binding owns the same variables minus its quantifiers.
		s := &Scheme{TypeVars: []string{"v"}, Type: typ}
		if NewTypeEnv().ExtendScheme("p", s).ownedVars().types["v"] {
			t.Errorf("*%s: a quantified variable is reported free", name)
		}
	}
}

// The canonical traversal visits record fields in sorted label order, for open
// and closed records alike. Iterating a TRecordOpen's fields in map order left
// every other test green (PR #2209 review); here eight fields make a map-order
// walk produce a wrong name on almost every one of the 20 rounds.
func TestCanonicalizeScheme_RecordFieldsInLabelOrder(t *testing.T) {
	labels := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, open := range []bool{true, false} {
		for round := 0; round < 20; round++ {
			fields := map[string]Type{}
			var quantified []string
			for i, l := range labels {
				v := fmt.Sprintf("α%d", 100-i) // raw names in the opposite order
				fields[l] = &TVar2{Name: v, Kind: Star}
				quantified = append(quantified, v)
			}
			var typ Type = &TRecord{Fields: fields}
			if open {
				typ = &TRecordOpen{Fields: fields, Row: &RowVar{Name: "r", Kind: RecordRow}}
			}
			c, _ := CanonicalizeScheme(&Scheme{TypeVars: quantified, Type: &TFunc2{Params: []Type{typ}, Return: TInt}})
			got := c.Type.(*TFunc2).Params[0]
			gotFields := map[string]Type{}
			switch r := got.(type) {
			case *TRecord:
				gotFields = r.Fields
			case *TRecordOpen:
				gotFields = r.Fields
			}
			for i, l := range labels {
				want := fmt.Sprintf("'t%d", i)
				if name := gotFields[l].String(); name != want {
					t.Fatalf("open=%v round %d: field %s named %s, want %s (fields must be walked in label order)", open, round, l, name, want)
				}
			}
		}
	}
}

// varsUnderSubstitution: a variable the environment owns owns everything the
// substitution maps it to, through chains; nil sets are skipped.
func TestVarsUnderSubstitution(t *testing.T) {
	sub := Substitution{
		"α1": &TRecordOpen{Fields: map[string]Type{"name": &TVar2{Name: "α2", Kind: Star}}, Row: &RowVar{Name: "r", Kind: RecordRow}},
		"α2": &TList{Element: &TVar2{Name: "α3", Kind: Star}},
	}
	got := varsUnderSubstitution(sub, map[string]bool{"α1": true}, nil, map[string]bool{"ρ9": true})
	for _, v := range []string{"α1", "α3", "r", "ρ9"} {
		if !got[v] {
			t.Errorf("%s not owned: %v", v, got)
		}
	}
	if got["α2"] {
		t.Errorf("α2 is substituted away and should not be reported: %v", got)
	}
	if len(varsUnderSubstitution(nil)) != 0 {
		t.Error("no owned sets, no variables")
	}
}

// recordRowsToWithhold: rows of the frames above the declaration's base, rows
// free in env but not in base, every free row without a base, nothing for a
// nil env.
func TestRecordRowsToWithhold(t *testing.T) {
	open := func(row string) Type {
		return &TRecordOpen{Fields: map[string]Type{"x": TInt}, Row: &RowVar{Name: row, Kind: RecordRow}}
	}
	base := NewTypeEnv().Extend("leaked", open("ρbase"))
	env := base.Extend("p", open("ρparam"))
	got := env.recordRowsToWithhold(base)
	if !got["ρparam"] || got["ρbase"] {
		t.Fatalf("withhold = %v, want the parameter's row only", got)
	}
	if all := env.recordRowsToWithhold(nil); !all["ρparam"] || !all["ρbase"] {
		t.Fatalf("no base: withhold = %v, want every free row", all)
	}
	unrelated := NewTypeEnv().Extend("q", open("ρother"))
	if got := env.recordRowsToWithhold(unrelated); !got["ρparam"] || !got["ρbase"] {
		t.Fatalf("base not an ancestor: withhold = %v, want rows free in env but not in base", got)
	}
	var nilEnv *TypeEnv
	if nilEnv.recordRowsToWithhold(base) != nil {
		t.Fatal("nil env withholds nothing")
	}
}

// generalizeWithConstraints withholds what an enclosing parameter owns: its
// record row, and everything the current substitution says its unsolved type
// variable stands for. A declaration-local variable is still generalised.
func TestGeneralize_WithholdsWhatTheEnclosingParameterOwns(t *testing.T) {
	tc := NewCoreTypeChecker()
	base := NewTypeEnv()
	baseFree := base.FreeTypeVars()

	// p : {name: string | ρp} — the parameter's record row is owned.
	env := base.Extend("p", &TRecordOpen{Fields: map[string]Type{"name": TString}, Row: &RowVar{Name: "ρp", Kind: RecordRow}})
	q := &TFunc2{
		Params: []Type{&TVar2{Name: "αlocal", Kind: Star}},
		Return: &TRecordOpen{Fields: map[string]Type{"name": TString}, Row: &RowVar{Name: "ρp", Kind: RecordRow}},
	}
	s := tc.generalizeWithConstraints(q, EmptyEffectRow(), nil, nil, env, baseFree, base, nil)
	if slices.Contains(s.RowVars, "ρp") {
		t.Errorf("record row owned by the parameter was generalised: %v", s)
	}
	if !slices.Contains(s.TypeVars, "αlocal") {
		t.Errorf("declaration-local variable not generalised: %v", s)
	}

	// p : α1 with α1 ↦ {name: α2 | r} — α2 and r are owned through the substitution.
	env = base.Extend("p", &TVar2{Name: "α1", Kind: Star})
	sub := Substitution{"α1": &TRecordOpen{Fields: map[string]Type{"name": &TVar2{Name: "α2", Kind: Star}}, Row: &RowVar{Name: "r", Kind: RecordRow}}}
	g := &TFunc2{
		Params: []Type{&TVar2{Name: "αu", Kind: Star}},
		Return: &TRecord{Fields: map[string]Type{"name": &TVar2{Name: "α2", Kind: Star}}},
	}
	s = tc.generalizeWithConstraints(g, EmptyEffectRow(), nil, sub, env, baseFree, base, nil)
	if slices.Contains(s.TypeVars, "α2") {
		t.Errorf("variable owned through the substitution was generalised: %v", s)
	}
	if !slices.Contains(s.TypeVars, "αu") {
		t.Errorf("declaration-local variable not generalised: %v", s)
	}
}

// nodeSet keeps the first nodes inline and moves to a map past that; both
// halves must answer membership by pointer identity, and a large cyclic type
// must still terminate with every variable found.
func TestNodeSet_SpillsToMapAndStaysCycleSafe(t *testing.T) {
	var s nodeSet
	nodes := make([]Type, 40)
	for i := range nodes {
		nodes[i] = &TVar2{Name: fmt.Sprintf("v%d", i), Kind: Star}
		if s.add(nodes[i]) {
			t.Fatalf("node %d reported present on first add", i)
		}
	}
	for i, n := range nodes {
		if !s.add(n) {
			t.Fatalf("node %d not found after spilling to the map", i)
		}
	}
	if s.add(&TVar2{Name: "v0", Kind: Star}) {
		t.Fatal("membership must be by identity, not by name")
	}

	fields := map[string]Type{}
	for i := 0; i < 30; i++ {
		fields[fmt.Sprintf("f%02d", i)] = &TList{Element: &TVar2{Name: fmt.Sprintf("α%d", i), Kind: Star}}
	}
	rec := &TRecord{Fields: fields}
	fields["self"] = &TList{Element: rec} // cycle
	if tv, _ := generalizableVars(rec); len(tv) != 30 {
		t.Fatalf("found %d type variables in a 30-variable cyclic record, want 30", len(tv))
	}
	free := map[string]bool{}
	collectFreeTypeVars(rec, free)
	if len(free) != 30 {
		t.Fatalf("unordered walk found %d, want 30", len(free))
	}
}
