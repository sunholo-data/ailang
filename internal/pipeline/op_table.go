// Package pipeline provides compilation passes for AILANG
package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// OpMapping defines how an intrinsic operation maps to builtins
type OpMapping struct {
	Builtin string   // Base builtin name (e.g., "add")
	Types   []string // Supported types (e.g., ["Int", "Float"])
}

// OperatorTable defines all operator to builtin mappings
var OperatorTable = map[core.IntrinsicOp]OpMapping{
	// Arithmetic operations
	core.OpAdd: {Builtin: "add", Types: []string{"Int", "Float"}},
	core.OpSub: {Builtin: "sub", Types: []string{"Int", "Float"}},
	core.OpMul: {Builtin: "mul", Types: []string{"Int", "Float"}},
	core.OpDiv: {Builtin: "div", Types: []string{"Int", "Float"}},
	core.OpMod: {Builtin: "mod", Types: []string{"Int", "Float"}},
	
	// Comparison operations
	core.OpEq: {Builtin: "eq", Types: []string{"Int", "Float", "String", "Bool"}},
	core.OpNe: {Builtin: "ne", Types: []string{"Int", "Float", "String", "Bool"}},
	core.OpLt: {Builtin: "lt", Types: []string{"Int", "Float", "String"}},
	core.OpLe: {Builtin: "le", Types: []string{"Int", "Float", "String"}},
	core.OpGt: {Builtin: "gt", Types: []string{"Int", "Float", "String"}},
	core.OpGe: {Builtin: "ge", Types: []string{"Int", "Float", "String"}},
	
	// String operations
	core.OpConcat: {Builtin: "concat", Types: []string{"String"}},
	
	// Boolean operations (short-circuit, handled specially)
	core.OpAnd: {Builtin: "and", Types: []string{"Bool"}},
	core.OpOr:  {Builtin: "or", Types: []string{"Bool"}},
	
	// Unary operations
	core.OpNot: {Builtin: "not", Types: []string{"Bool"}},
	core.OpNeg: {Builtin: "neg", Types: []string{"Int", "Float"}},
}

// GetBuiltinName returns the monomorphic builtin name for an operator and type
func GetBuiltinName(op core.IntrinsicOp, typ string) (string, error) {
	mapping, ok := OperatorTable[op]
	if !ok {
		return "", fmt.Errorf("unknown operator: %v", op)
	}
	
	// Verify type is supported
	found := false
	for _, t := range mapping.Types {
		if t == typ {
			found = true
			break
		}
	}
	if !found {
		opStr := GetOpSymbol(op)
		return "", fmt.Errorf("ELB_OP001: Operator '%s' has no implementation for type %s. Suggestion: Align operand types (e.g., add cast)", 
			opStr, typ)
	}
	
	return fmt.Sprintf("%s_%s", mapping.Builtin, typ), nil
}

// GetOpSymbol returns the string representation of an operator
func GetOpSymbol(op core.IntrinsicOp) string {
	symbols := map[core.IntrinsicOp]string{
		core.OpAdd: "+", core.OpSub: "-", core.OpMul: "*", core.OpDiv: "/", core.OpMod: "%",
		core.OpEq: "==", core.OpNe: "!=", core.OpLt: "<", core.OpLe: "<=", core.OpGt: ">", core.OpGe: ">=",
		core.OpConcat: "++", core.OpAnd: "&&", core.OpOr: "||", core.OpNot: "not", core.OpNeg: "-",
	}
	if sym, ok := symbols[op]; ok {
		return sym
	}
	return fmt.Sprintf("op_%d", op)
}

// GetAllBuiltinNames returns all registered builtin names (sorted)
func GetAllBuiltinNames() []string {
	seen := make(map[string]bool)
	for _, mapping := range OperatorTable {
		for _, typ := range mapping.Types {
			name := fmt.Sprintf("%s_%s", mapping.Builtin, typ)
			seen[name] = true
		}
	}
	
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// IsOperatorComplete checks if all IntrinsicOp values have mappings
func IsOperatorTableComplete() error {
	// Check that all operators from 0 to OpNeg are mapped
	// This assumes operators are defined contiguously
	for op := core.IntrinsicOp(0); op <= core.OpNeg; op++ {
		if _, ok := OperatorTable[op]; !ok {
			return fmt.Errorf("operator %v has no mapping in OperatorTable", op)
		}
	}
	return nil
}

// OperatorSemantics documents the semantics of each operator
var OperatorSemantics = map[string]string{
	"div_Int": "Integer division truncates toward zero (e.g., -7/2 = -3)",
	"mod_Int": "Integer modulo has the sign of the dividend (e.g., -7%3 = -1)",
	"div_Float": "Float division follows IEEE 754 (division by zero produces ±Inf)",
	"mod_Float": "Float modulo follows IEEE 754 (mod by zero produces NaN)",
	"eq_Float": "Float equality: NaN != NaN is false, all other comparisons standard",
	"ne_Float": "Float inequality: NaN != x is true for all x (including NaN)",
	"lt_Float": "Float less-than: any comparison with NaN is false",
	"and_Bool": "Boolean AND short-circuits: false && _ returns false without evaluating RHS",
	"or_Bool": "Boolean OR short-circuits: true || _ returns true without evaluating RHS",
}

// GetBuiltinType returns the type signature for a builtin
func GetBuiltinType(name string) string {
	// Parse builtin name (e.g., "add_Int" -> "Int -> Int -> Int")
	parts := strings.Split(name, "_")
	if len(parts) != 2 {
		return "?"
	}
	
	op := parts[0]
	typ := parts[1]
	
	// Determine signature based on operation
	switch op {
	case "add", "sub", "mul", "div", "mod":
		return fmt.Sprintf("%s -> %s -> %s", typ, typ, typ)
	case "eq", "ne", "lt", "le", "gt", "ge":
		return fmt.Sprintf("%s -> %s -> Bool", typ, typ)
	case "concat":
		return "String -> String -> String"
	case "and", "or":
		return "Bool -> Bool -> Bool"
	case "not":
		return "Bool -> Bool"
	case "neg":
		return fmt.Sprintf("%s -> %s", typ, typ)
	default:
		return "?"
	}
}