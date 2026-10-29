package iface

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
)

// astTypeToInternalType converts an AST type to an internal type
// This is used during interface building to get actual constructor field types
func astTypeToInternalType(t ast.Type) types.Type {
	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return types.TInt
		case "float":
			return types.TFloat
		case "string":
			return types.TString
		case "bool":
			return types.TBool
		case "()":
			return types.TUnit
		case "bytes":
			return types.TBytes
		default:
			// Type variable or constructor
			if len(typ.Name) > 0 && typ.Name[0] >= 'a' && typ.Name[0] <= 'z' {
				return &types.TVar2{Name: typ.Name, Kind: types.Star}
			}
			return &types.TCon{Name: typ.Name}
		}

	case *ast.FuncType:
		paramTypes := make([]types.Type, len(typ.Params))
		for i, p := range typ.Params {
			paramTypes[i] = astTypeToInternalType(p)
		}
		var effectRow *types.Row
		if len(typ.Effects) > 0 {
			labels := make(map[string]types.Type)
			var tail *types.RowVar
			for _, e := range typ.Effects {
				if e.IsRowVar {
					tail = &types.RowVar{Name: e.Name, Kind: types.EffectRow}
				} else {
					labels[e.Name] = types.TUnit
				}
			}
			effectRow = &types.Row{
				Kind:   types.EffectRow,
				Labels: labels,
				Tail:   tail,
			}
		} else {
			effectRow = types.EmptyEffectRow()
		}
		return &types.TFunc2{
			Params:                 paramTypes,
			EffectRow:              effectRow,
			Return:                 astTypeToInternalType(typ.Return),
			ImplicitEffectContract: typ.Effects == nil,
			ConcreteEffectContract: typ.Effects != nil && !hasRowVarEffects(typ.Effects),
		}

	case *ast.ListType:
		return &types.TList{
			Element: astTypeToInternalType(typ.Element),
		}

	case *ast.ArrayType:
		return &types.TArray{
			Element: astTypeToInternalType(typ.Element),
		}

	case *ast.TupleType:
		elements := make([]types.Type, len(typ.Elements))
		for i, e := range typ.Elements {
			elements[i] = astTypeToInternalType(e)
		}
		return &types.TTuple{Elements: elements}

	case *ast.RecordType:
		// Convert record type fields
		labels := make(map[string]types.Type)
		for _, f := range typ.Fields {
			labels[f.Name] = astTypeToInternalType(f.Type)
		}
		return &types.TRecord{Fields: labels, Row: nil}

	case *ast.TypeVar:
		// Type variables in constructor fields (e.g., 'a' in Ok(a), 'e' in Err(e))
		return &types.TVar2{Name: typ.Name, Kind: types.Star}

	case *ast.TypeApp:
		// Type applications (e.g., Option[int], Result[a, e])
		args := make([]types.Type, len(typ.Args))
		for i, a := range typ.Args {
			args[i] = astTypeToInternalType(a)
		}
		return &types.TApp{
			Constructor: &types.TCon{Name: typ.Constructor},
			Args:        args,
		}

	case *ast.LabelledType:
		// IFC label — wrap the base type in TLabelled so the label survives
		// iface serialisation (M6). Refinements (`{not LABEL}`) are sink-side
		// only and are not stored on the type itself.
		base := astTypeToInternalType(typ.Base)
		if typ.Label != nil {
			return types.WithLabel(base, types.LabelConst(typ.Label.Name))
		}
		return base

	default:
		// No silent fallback — fail loudly per CLAUDE.md Section 2
		panic(fmt.Sprintf("astTypeToInternalType: unhandled ast.Type variant: %T", t))
	}
}
