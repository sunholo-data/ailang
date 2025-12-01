package types

// TypeHead represents the head constructor of a type for operator lowering
type TypeHead int

const (
	HeadUnknown TypeHead = iota
	HeadInt
	HeadFloat
	HeadString
	HeadBool
	HeadList
	HeadRecord
	HeadFunc
	HeadUnit
	HeadBytes
)

// Head returns the head constructor of a type for type-guided operator lowering.
// This is used to determine which builtin variant to use (e.g., _str_concat vs _list_concat).
//
// Examples:
//   - Head(TInt) → HeadInt
//   - Head(TString) → HeadString
//   - Head(TApp{Constructor: TCon{Name: "List"}, ...}) → HeadList
//   - Head(TVar{...}) → HeadUnknown (type variable, need more inference)
func Head(t Type) TypeHead {
	if t == nil {
		return HeadUnknown
	}

	switch typ := t.(type) {
	case *TCon:
		switch typ.Name {
		case "int":
			return HeadInt
		case "float":
			return HeadFloat
		case "string":
			return HeadString
		case "bool":
			return HeadBool
		case "()":
			return HeadUnit
		case "bytes":
			return HeadBytes
		default:
			return HeadUnknown
		}

	case *TList:
		// TList is the dedicated list type
		return HeadList

	case *TApp:
		// Check if it's a List type application (alternative representation)
		if con, ok := typ.Constructor.(*TCon); ok {
			if con.Name == "List" {
				return HeadList
			}
		}
		return HeadUnknown

	case *TRecord, *TRecord2:
		return HeadRecord

	case *TFunc, *TFunc2:
		return HeadFunc

	case *TVar:
		// Type variable - cannot determine head
		return HeadUnknown

	default:
		return HeadUnknown
	}
}

// String returns a human-readable name for the type head
func (th TypeHead) String() string {
	switch th {
	case HeadInt:
		return "Int"
	case HeadFloat:
		return "Float"
	case HeadString:
		return "String"
	case HeadBool:
		return "Bool"
	case HeadList:
		return "List"
	case HeadRecord:
		return "Record"
	case HeadFunc:
		return "Func"
	case HeadUnit:
		return "Unit"
	case HeadBytes:
		return "Bytes"
	case HeadUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}
