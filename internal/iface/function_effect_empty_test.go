package iface

import (
	"bytes"
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"strings"
	"testing"
)

func TestExplicitEmptyIfaceConversionAndRestore(t *testing.T) {
	astFn := &ast.FuncType{Params: []ast.Type{&ast.SimpleType{Name: "int"}}, Return: &ast.SimpleType{Name: "int"}, Effects: []ast.EffectAnnotation{}}
	check := func(t *testing.T, fn *types.TFunc2) {
		t.Helper()
		if fn.EffectRow == nil || fn.EffectRow.Tail != nil || len(fn.EffectRow.Labels) != 0 || !fn.ConcreteEffectContract || fn.ImplicitEffectContract {
			t.Fatalf("empty contract lost: %+v", fn)
		}
	}
	check(t, astTypeToInternalType(astFn).(*types.TFunc2))
	inferred := &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: &types.Row{Kind: types.EffectRow, Labels: map[string]types.Type{}, Tail: &types.RowVar{Name: "old", Kind: types.EffectRow}}, ImplicitEffectContract: true}
	check(t, restoreNestedEffectRow(inferred, astFn).(*types.TFunc2))
	outer := &types.TFunc2{Params: []types.Type{inferred}, Return: inferred, EffectRow: types.EmptyEffectRow()}
	restored := applyLabelsFromAST(outer, &ast.FuncDecl{Params: []*ast.Param{{Name: "f", Type: astFn}}, ReturnType: astFn}).(*types.TFunc2)
	check(t, restored.Params[0].(*types.TFunc2))
	check(t, restored.Return.(*types.TFunc2))
}

func TestExplicitEmptyIfaceContractIdentity(t *testing.T) {
	callback := func(concrete bool) *types.TFunc2 {
		return &types.TFunc2{Params: []types.Type{types.TInt}, Return: types.TInt, EffectRow: types.EmptyEffectRow(), ConcreteEffectContract: concrete, ImplicitEffectContract: !concrete}
	}
	makeIface := func(alias, ctor, exported bool) *Iface {
		i := NewIface("contract")
		i.AddType("Callback", 0)
		i.TypeAliases["Callback"] = callback(alias)
		i.AddType("Box", 0)
		i.AddConstructor("Box", "Box", []types.Type{callback(ctor)}, &types.TCon{Name: "Box"})
		i.AddExport("keep", &types.Scheme{Type: &types.TFunc2{Params: []types.Type{callback(exported)}, Return: callback(exported), EffectRow: types.EmptyEffectRow()}}, true)
		return i
	}
	raw, _ := makeIface(false, false, false).ToNormalizedJSON()
	var before InterfaceJSON
	if err := json.Unmarshal(raw, &before); err != nil {
		t.Fatal(err)
	}
	beforeHash, _ := HashProjection(&before)
	for _, tt := range []struct {
		name                  string
		alias, ctor, exported bool
	}{
		{"alias_only", true, false, false}, {"constructor_only", false, true, false}, {"export_only", false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			changed, _ := makeIface(tt.alias, tt.ctor, tt.exported).ToNormalizedJSON()
			if bytes.Equal(raw, changed) {
				t.Fatal("contract disappeared from normalized JSON")
			}
			var after InterfaceJSON
			if err := json.Unmarshal(changed, &after); err != nil {
				t.Fatal(err)
			}
			marker := after.Types[1].Alias
			if tt.ctor {
				marker = after.Types[0].Ctors[0]
			}
			if tt.exported {
				marker = after.Funcs[0].Type
			}
			if !strings.Contains(marker, "!{}") {
				t.Fatalf("missing explicit-empty marker: %s", changed)
			}
			afterHash, _ := HashProjection(&after)
			sameHash := bytes.Equal(beforeHash, afterHash)
			sameSet := strings.Join(SignatureSet(&before), "\n") == strings.Join(SignatureSet(&after), "\n")
			// Public signatures intentionally exclude alias bodies. Compile dependency
			// identity covers them separately via aliasDigest and warm-edit tests.
			if sameHash != tt.alias || sameSet != tt.alias {
				t.Fatalf("alias exclusion / signature identity mismatch: hashSame=%v setSame=%v", sameHash, sameSet)
			}
		})
	}
}
