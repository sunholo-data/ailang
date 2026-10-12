package pipeline

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// Enumerate the real Type implementors from source, so adding a new variant
// without a traversal case fails this test rather than silently retaining names.
func TestAliasBodyClosure_AllTypeVariants(t *testing.T) {
	leaf := &types.TCon{Name: "Item"}
	row := func(x types.Type) *types.Row {
		return &types.Row{Kind: types.RecordRow, Labels: map[string]types.Type{"x": x}, Tail: &types.RowVar{Name: "r", Kind: types.RecordRow}}
	}
	wrappers := []func(types.Type) types.Type{
		func(x types.Type) types.Type { return &types.TList{Element: x} },
		func(x types.Type) types.Type { return &types.TArray{Element: x} },
		func(x types.Type) types.Type { return &types.TMap{Key: x, Value: x} },
		func(x types.Type) types.Type { return &types.TTuple{Elements: []types.Type{x}} },
		func(x types.Type) types.Type {
			return &types.TRecord{Fields: map[string]types.Type{"x": x}, Row: &types.TVar{Name: "r"}, TypeName: "Existing"}
		},
		func(x types.Type) types.Type {
			return &types.TRecordOpen{Fields: map[string]types.Type{"x": x}, Row: &types.TVar{Name: "r"}}
		},
		func(x types.Type) types.Type {
			return &types.TApp{Constructor: &types.TCon{Name: "list"}, Args: []types.Type{x}}
		},
		func(x types.Type) types.Type { return row(x) },
		func(x types.Type) types.Type { return &types.TRecord2{Row: row(x)} },
		func(x types.Type) types.Type {
			return &types.TFunc2{Params: []types.Type{x}, Return: x, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": x}}}
		},
		func(x types.Type) types.Type { return &types.TLabelled{Inner: x, L: types.LabelBottom()} },
	}
	covered := map[string]bool{}
	target := &types.TRecord{Fields: map[string]types.Type{"value": types.TInt}, TypeName: "Item"}
	c := newAliasBodyCloser(map[string]types.Type{"Item": target}, nil)
	for _, wrap := range wrappers {
		input, want := wrap(leaf), wrap(target)
		name := reflect.TypeOf(input).Elem().Name()
		covered[name] = true
		t.Run(name, func(t *testing.T) {
			before := input.String()
			got := c.walk(input)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v (%s), want %s", got, got, want)
			}
			if input.String() != before {
				t.Fatal("closure mutated input")
			}
			if !reflect.DeepEqual(newAliasBodyCloser(nil, nil).walk(input), input) {
				t.Fatal("empty environment changed type")
			}
		})
	}
	for _, terminal := range []types.Type{types.TInt, &types.TVar{Name: "a"}, &types.TVar2{Name: "b", Kind: types.Star}, &types.RowVar{Name: "r", Kind: types.RecordRow}} {
		covered[reflect.TypeOf(terminal).Elem().Name()] = true
		if c.walk(terminal) != terminal {
			t.Fatal("terminal identity changed")
		}
	}
	packages, err := parser.ParseDir(token.NewFileSet(), "../types", func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range packages["types"].Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "Substitute" || fn.Recv == nil {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			name := star.X.(*ast.Ident).Name
			if !covered[name] {
				t.Errorf("uncovered concrete types.Type: %s", name)
			}
		}
	}
}

func TestAliasBodyClosure_CyclesMemoizationAndNames(t *testing.T) {
	env := map[string]types.Type{
		"A":      &types.TRecord{Fields: map[string]types.Type{"next": &types.TCon{Name: "B"}}},
		"B":      &types.TRecord{Fields: map[string]types.Type{"next": &types.TCon{Name: "A"}}},
		"Self":   &types.TList{Element: &types.TCon{Name: "Self"}},
		"Item":   &types.TRecord{Fields: map[string]types.Type{"x": types.TInt}},
		"Tagged": &types.TRecord{Fields: map[string]types.Type{"x": types.TInt}, TypeName: "Original"},
	}
	before := env["A"].String()
	first, reverse := newAliasBodyCloser(env, nil), newAliasBodyCloser(env, nil)
	a, b := first.alias("A"), first.alias("B")
	rb, ra := reverse.alias("B"), reverse.alias("A")
	if !reflect.DeepEqual(a, ra) || !reflect.DeepEqual(b, rb) {
		t.Fatal("cycle expansion depends on root traversal order")
	}
	if len(first.memo) != 0 {
		t.Fatal("cyclic expansions were memoized")
	}
	if first.alias("Self").String() != "[Self]" {
		t.Fatal("self recursion did not remain opaque")
	}
	if first.alias("Item").(*types.TRecord).TypeName != "Item" || first.alias("Tagged").(*types.TRecord).TypeName != "Original" {
		t.Fatal("record tags lost")
	}
	if env["A"].String() != before || env["Item"].(*types.TRecord).TypeName != "" {
		t.Fatal("alias inputs mutated")
	}
	item := first.alias("Item")
	if first.alias("Item") != item {
		t.Fatal("acyclic expansions should be memoized")
	}
}

func TestAliasBodyClosure_ParameterizedHeadsAndRowMetadata(t *testing.T) {
	c := newAliasBodyCloser(map[string]types.Type{"Poly": &types.TList{Element: &types.TVar{Name: "a"}}, "Item": types.TInt}, map[string][]string{"Poly": {"a"}})
	head := &types.TCon{Name: "Poly"}
	got := c.walk(&types.TApp{Constructor: head, Args: []types.Type{&types.TCon{Name: "Item"}}}).(*types.TApp)
	if got.Constructor != head || got.Args[0] != types.TInt || c.walk(head) != head {
		t.Fatal("parameterized alias head expanded without substitution")
	}
	if c.alias("Poly").String() != "[a]" {
		t.Fatal("alias parameter lost")
	}
	budget := 3
	row := &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{"IO": types.TUnit}, Budgets: map[string]*int{"IO": &budget}, MinBudgets: map[string]*int{"IO": &budget}, Params: map[string]map[string]string{"IO": {"mode": "test"}}, Tail: &types.RowVar{Name: "e", Kind: types.EffectRow}}
	if !reflect.DeepEqual(c.row(row), row) {
		t.Fatal("effect row metadata lost")
	}
	if c.walk(nil) != nil || c.row(nil) != nil {
		t.Fatal("nil changed")
	}
}

func TestAliasBodyClosure_InterfaceCopiesAndDigest(t *testing.T) {
	ifc := iface.NewIface("a")
	scheme := &types.Scheme{TypeVars: []string{"a"}, RowVars: []string{"r"}, Constraints: []types.Constraint{{Class: "Eq", Type: &types.TVar{Name: "a"}}}, Type: &types.TFunc2{Params: []types.Type{&types.TCon{Name: "Item"}}, Return: &types.TVar{Name: "a"}}}
	ifc.AddExport("f", scheme, true)
	ifc.AddConstructor("Wrapped", "Wrap", []types.Type{&types.TCon{Name: "Item"}}, &types.TCon{Name: "Wrapped"})
	originalCtor := ifc.Constructors["Wrap"]
	if err := ifc.SetDerivedEq(nil); err != nil {
		t.Fatal(err)
	}
	oldDigest := ifc.Digest
	closeInterfaceTypes(ifc, nil, map[string]types.Type{"Item": &types.TRecord{Fields: map[string]types.Type{"x": types.TInt}}}, nil)
	if scheme.Type.(*types.TFunc2).Params[0].String() != "Item" || originalCtor.FieldTypes[0].String() != "Item" {
		t.Fatal("shared scheme or constructor mutated")
	}
	closed := ifc.Exports["f"].Type
	if !reflect.DeepEqual(closed.TypeVars, scheme.TypeVars) || !reflect.DeepEqual(closed.RowVars, scheme.RowVars) || !reflect.DeepEqual(closed.Constraints, scheme.Constraints) {
		t.Fatal("scheme binders/constraints changed")
	}
	if ifc.Constructors["Wrap"].ResultType.String() != "Wrapped" {
		t.Fatal("nominal result expanded")
	}
	if err := ifc.SetDerivedEq(nil); err != nil {
		t.Fatal(err)
	}
	if oldDigest == ifc.Digest {
		t.Fatal("digest does not reflect closure")
	}
	digest := ifc.Digest
	if err := ifc.SetDerivedEq(nil); err != nil {
		t.Fatal(err)
	}
	if ifc.Digest != digest {
		t.Fatal("unstable digest")
	}
	data, err := marshalIfaceFull(ifc)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := unmarshalIfaceFull(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := restored.SetDerivedEq(nil); err != nil {
		t.Fatal(err)
	}
	if restored.Digest != digest || !reflect.DeepEqual(restored.Constructors["Wrap"].FieldTypes, ifc.Constructors["Wrap"].FieldTypes) {
		t.Fatal("serialization changed closure/digest")
	}
}

func TestAliasBodyClosure_InterfaceAliasParamsAndImportedEnvironment(t *testing.T) {
	ifc := iface.NewIface("local")
	variable := &types.TVar{Name: "a"}
	ifc.AddTypeAlias("Poly", &types.TTuple{Elements: []types.Type{variable, &types.TCon{Name: "Private"}, &types.TCon{Name: "Imported"}}})
	ifc.AddTypeAliasParams("Poly", []string{"a"})
	imported := map[string]types.Type{"Private": types.TString, "Imported": types.TBool}
	local := map[string]types.Type{"Poly": ifc.TypeAliases["Poly"], "Private": types.TInt}
	closeInterfaceTypes(ifc, imported, local, ifc.AliasParams)
	got := ifc.TypeAliases["Poly"].(*types.TTuple)
	if got.Elements[0] != variable || got.Elements[1] != types.TInt || got.Elements[2] != types.TBool {
		t.Fatal("wrong local/imported environment or lost variable")
	}
	if !reflect.DeepEqual(ifc.AliasParams["Poly"], []string{"a"}) {
		t.Fatal("AliasParams changed")
	}
	if imported["Private"] != types.TString || local["Poly"].String() != "(a, Private, Imported)" {
		t.Fatal("alias environment mutated")
	}
	// Nil rows and empty composite slices occur in monomorphic interfaces too.
	c := newAliasBodyCloser(nil, nil)
	for _, typ := range []types.Type{&types.TRecord2{}, &types.TFunc2{Return: types.TUnit}, &types.TTuple{}, &types.TRecord{}} {
		if !reflect.DeepEqual(c.walk(typ), typ) {
			t.Fatalf("empty variant changed: %T", typ)
		}
	}
}
