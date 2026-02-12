package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/types"
)

// SMTContext holds the state needed for SMT-LIB generation.
type SMTContext struct {
	// Declarations accumulates SMT-LIB type declarations.
	Declarations []string
	// Variables maps AILANG parameter names to their SMT-LIB types.
	Variables map[string]string
	// DeclaredTypes tracks which ADT types have been declared.
	DeclaredTypes map[string]bool
}

// NewSMTContext creates a new SMT encoding context.
func NewSMTContext() *SMTContext {
	return &SMTContext{
		Variables:     make(map[string]string),
		DeclaredTypes: make(map[string]bool),
	}
}

// MapType maps an AILANG type to its SMT-LIB sort name.
// Returns the SMT-LIB sort string and an error if the type is not encodable.
func MapType(t types.Type) (string, error) {
	if t == nil {
		return "", fmt.Errorf("nil type")
	}
	switch ty := t.(type) {
	case *types.TCon:
		return mapTCon(ty.Name)
	case *types.TVar:
		// Type variables are not directly encodable; they should be
		// monomorphized before SMT encoding.
		return "", fmt.Errorf("type variable %q cannot be encoded in SMT-LIB (needs monomorphization)", ty.Name)
	case *types.TFunc:
		return "", fmt.Errorf("function types cannot be encoded in SMT-LIB")
	case *types.TList:
		return "", fmt.Errorf("list types cannot be encoded in SMT-LIB")
	case *types.TRecord:
		return "", fmt.Errorf("record types cannot be encoded in SMT-LIB")
	case *types.TApp:
		// TApp with a TCon constructor may be a parameterized ADT
		if con, ok := ty.Constructor.(*types.TCon); ok {
			return con.Name, nil
		}
		return "", fmt.Errorf("parameterized type %v cannot be encoded in SMT-LIB", ty)
	default:
		return "", fmt.Errorf("unsupported type %T for SMT-LIB encoding", t)
	}
}

// mapTCon maps a type constructor name to SMT-LIB sort.
func mapTCon(name string) (string, error) {
	switch name {
	case "int":
		return "Int", nil
	case "float":
		return "Real", nil
	case "bool":
		return "Bool", nil
	case "string":
		return "", fmt.Errorf("string type cannot be encoded in SMT-LIB")
	case "unit":
		return "", fmt.Errorf("unit type cannot be encoded in SMT-LIB")
	default:
		// Assume it's an ADT name — return as-is for declare-datatype
		return name, nil
	}
}

// ADTVariant describes a single variant of an algebraic data type.
type ADTVariant struct {
	Name   string
	Fields []ADTField
}

// ADTField describes a field within an ADT variant.
type ADTField struct {
	Name string // Accessor name (auto-generated if not specified)
	Sort string // SMT-LIB sort name
}

// DeclareDatatype generates an SMT-LIB declare-datatype for an ADT.
// For enums (all variants have zero fields):
//
//	(declare-datatype Season ((LOW_SEASON) (HIGH_SEASON)))
//
// For ADTs with fields:
//
//	(declare-datatype Shape ((Circle (radius Int)) (Rect (width Int) (height Int))))
func DeclareDatatype(typeName string, variants []ADTVariant) string {
	var parts []string
	for _, v := range variants {
		if len(v.Fields) == 0 {
			parts = append(parts, fmt.Sprintf("(%s)", v.Name))
		} else {
			var fieldParts []string
			for _, f := range v.Fields {
				fieldParts = append(fieldParts, fmt.Sprintf("(%s %s)", f.Name, f.Sort))
			}
			parts = append(parts, fmt.Sprintf("(%s %s)", v.Name, strings.Join(fieldParts, " ")))
		}
	}
	return fmt.Sprintf("(declare-datatype %s (%s))", typeName, strings.Join(parts, " "))
}

// DeclareEnumDatatype is a convenience for enums (all nullary constructors).
func DeclareEnumDatatype(typeName string, variants []string) string {
	adtVariants := make([]ADTVariant, len(variants))
	for i, v := range variants {
		adtVariants[i] = ADTVariant{Name: v}
	}
	return DeclareDatatype(typeName, adtVariants)
}

// DeclareConst generates an SMT-LIB constant declaration.
//
//	(declare-const name sort)
func DeclareConst(name, sort string) string {
	return fmt.Sprintf("(declare-const %s %s)", name, sort)
}

// Assert generates an SMT-LIB assertion.
//
//	(assert expr)
func Assert(expr string) string {
	return fmt.Sprintf("(assert %s)", expr)
}

// AssertNot generates a negated SMT-LIB assertion.
//
//	(assert (not expr))
func AssertNot(expr string) string {
	return fmt.Sprintf("(assert (not %s))", expr)
}

// CheckSat returns the check-sat command.
func CheckSat() string {
	return "(check-sat)"
}

// GetModel returns the get-model command.
func GetModel() string {
	return "(get-model)"
}

// BuiltinToSMTOp maps lowered AILANG builtin names to SMT-LIB operators.
// After op_lowering, operators become App(VarGlobal($builtin.XXX)) calls.
// This map reverses that transformation for SMT encoding.
var BuiltinToSMTOp = map[string]string{
	// Arithmetic
	"add_Int":   "+",
	"add_Float": "+",
	"sub_Int":   "-",
	"sub_Float": "-",
	"mul_Int":   "*",
	"mul_Float": "*",
	"div_Int":   "div",
	"div_Float": "/",
	"mod_Int":   "mod",
	"mod_Float": "mod",

	// Comparison
	"eq_Int":   "=",
	"eq_Float": "=",
	"eq_Bool":  "=",
	"ne_Int":   "distinct",
	"ne_Float": "distinct",
	"ne_Bool":  "distinct",
	"lt_Int":   "<",
	"lt_Float": "<",
	"le_Int":   "<=",
	"le_Float": "<=",
	"gt_Int":   ">",
	"gt_Float": ">",
	"ge_Int":   ">=",
	"ge_Float": ">=",

	// Boolean
	"and_Bool": "and",
	"or_Bool":  "or",
	"not_Bool": "not",

	// Unary
	"neg_Int":   "-",
	"neg_Float": "-",
}
