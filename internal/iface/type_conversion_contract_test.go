package iface

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// Constructor fields and imported aliases use this converter rather than the
// source elaborator. Every container must retain the callback's source contract,
// including a declared effect-row variable and an IFC label around the callback.
func TestIfaceCallbackContractsThroughTypeContainers(t *testing.T) {
	for _, tt := range []struct {
		name               string
		effects            []ast.EffectAnnotation
		implicit, concrete bool
		io                 bool
		tail               string
	}{
		{name: "omitted", implicit: true},
		{name: "explicit_empty", effects: []ast.EffectAnnotation{}, concrete: true},
		{name: "concrete_io", effects: []ast.EffectAnnotation{{Name: "IO"}}, concrete: true, io: true},
		{name: "row_polymorphic", effects: []ast.EffectAnnotation{{Name: "IO"}, {Name: "e", IsRowVar: true}}, io: true, tail: "e"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			callback := &ast.FuncType{
				Params:  []ast.Type{&ast.SimpleType{Name: "int"}},
				Return:  &ast.SimpleType{Name: "int"},
				Effects: tt.effects,
			}
			input := &ast.RecordType{Fields: []*ast.RecordField{
				{Name: "list", Type: &ast.ListType{Element: callback}},
				{Name: "array", Type: &ast.ArrayType{Element: callback}},
				{Name: "tuple", Type: &ast.TupleType{Elements: []ast.Type{&ast.SimpleType{Name: "int"}, callback}}},
				{Name: "option", Type: &ast.TypeApp{Constructor: "Option", Args: []ast.Type{callback}}},
				{Name: "labelled", Type: &ast.LabelledType{Base: callback, Label: &ast.LabelExpr{Name: "pii"}}},
				{Name: "refined", Type: &ast.LabelledType{Base: callback, Refinement: &ast.RefinementExpr{NotLabel: "pii"}}},
			}}
			file := &ast.File{Decls: []ast.Node{
				&ast.TypeDecl{Name: "CallbackBundle", Exported: true, Definition: input},
				&ast.TypeDecl{Name: "Wrapped", Exported: true, Definition: &ast.AlgebraicType{Constructors: []*ast.Constructor{
					{Name: "Wrap", Fields: []*ast.ConstructorField{{Type: input}}},
				}}},
			}}
			built, err := BuildInterfaceWithTypesAndConstructors("contracts", &core.Program{}, types.NewTypeEnv(), file, nil)
			if err != nil {
				t.Fatal(err)
			}
			alias, ok := built.GetTypeAlias("CallbackBundle")
			if !ok {
				t.Fatal("exported callback bundle alias missing")
			}
			ctor, ok := built.GetConstructor("Wrap")
			if !ok || len(ctor.FieldTypes) != 1 {
				t.Fatalf("callback bundle constructor missing: %+v", ctor)
			}
			raw, err := built.ToNormalizedJSON()
			if err != nil {
				t.Fatal(err)
			}
			var normalized InterfaceJSON
			if err := json.Unmarshal(raw, &normalized); err != nil {
				t.Fatal(err)
			}
			if len(normalized.Types) != 2 {
				t.Fatalf("missing normalized alias or constructor: %s", raw)
			}
			for _, decl := range normalized.Types {
				text := decl.Alias + strings.Join(decl.Ctors, " ")
				wantEmpty := 0
				if tt.name == "explicit_empty" {
					wantEmpty = 6
				}
				if strings.Count(text, "!{}") != wantEmpty {
					t.Fatalf("normalized %s lost or invented empty contracts: %s", decl.Name, text)
				}
				if tt.io && !strings.Contains(text, "IO") {
					t.Fatalf("normalized %s lost IO: %s", decl.Name, text)
				}
			}
			for origin, bundle := range map[string]types.Type{"alias": alias, "constructor": ctor.FieldTypes[0]} {
				t.Run(origin, func(t *testing.T) {
					record, ok := bundle.(*types.TRecord)
					if !ok || len(record.Fields) != len(input.Fields) {
						t.Fatalf("callback bundle shape lost: %#v", record)
					}
					option := record.Fields["option"].(*types.TApp)
					if con, ok := option.Constructor.(*types.TCon); !ok || con.Name != "Option" || len(option.Args) != 1 {
						t.Fatalf("Option callback argument lost: %+v", option)
					}
					if !types.LabelEqual(types.LabelOf(record.Fields["labelled"]), types.LabelConst("pii")) {
						t.Fatal("IFC label lost while converting the callback contract")
					}
					if _, labelled := record.Fields["refined"].(*types.TLabelled); labelled {
						t.Fatal("sink-only refinement became a stored IFC label")
					}
					callbacks := map[string]types.Type{
						"list":     record.Fields["list"].(*types.TList).Element,
						"array":    record.Fields["array"].(*types.TArray).Element,
						"tuple":    record.Fields["tuple"].(*types.TTuple).Elements[1],
						"option":   option.Args[0],
						"labelled": types.StripLabel(record.Fields["labelled"]),
						"refined":  record.Fields["refined"],
					}
					for name, typ := range callbacks {
						t.Run(name, func(t *testing.T) {
							fn, ok := typ.(*types.TFunc2)
							if !ok || len(fn.Params) != 1 || fn.Params[0] != types.TInt || fn.Return != types.TInt {
								t.Fatalf("callback signature lost: %T %+v", typ, typ)
							}
							if fn.ImplicitEffectContract != tt.implicit || fn.ConcreteEffectContract != tt.concrete {
								t.Fatalf("callback ownership changed: implicit=%v concrete=%v", fn.ImplicitEffectContract, fn.ConcreteEffectContract)
							}
							row := fn.EffectRow
							if row == nil || row.Kind != types.EffectRow {
								t.Fatalf("missing callback effect row: %+v", row)
							}
							wantLabels := 0
							if tt.io {
								wantLabels = 1
								if row.Labels["IO"] != types.TUnit {
									t.Fatalf("IO label lost: %+v", row)
								}
							}
							if len(row.Labels) != wantLabels {
								t.Fatalf("unexpected callback effects: %+v", row)
							}
							if tt.tail == "" {
								if row.Tail != nil {
									t.Fatalf("closed iface contract acquired a tail: %+v", row)
								}
							} else if row.Tail == nil || row.Tail.Name != tt.tail || row.Tail.Kind != types.EffectRow {
								t.Fatalf("declared effect-row variable lost: %+v", row)
							}
						})
					}
				})
			}
		})
	}
}
