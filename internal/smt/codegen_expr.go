package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// EncodeExpr translates a Core AST expression to an SMT-LIB expression string.
func EncodeExpr(expr core.CoreExpr) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("nil expression")
	}

	switch e := expr.(type) {
	case *core.Lit:
		return encodeLit(e)

	case *core.Var:
		return e.Name, nil

	case *core.VarGlobal:
		// For builtins, return the builtin name (will be used as function in App)
		if e.Ref.Module == "$builtin" {
			return e.Ref.Name, nil
		}
		// For ADT constructors, strip make_Type_ prefix
		return stripConstructorPrefix(e.Ref.Name), nil

	case *core.App:
		return encodeApp(e)

	case *core.If:
		cond, err := EncodeExpr(e.Cond)
		if err != nil {
			return "", fmt.Errorf("if condition: %w", err)
		}
		then, err := EncodeExpr(e.Then)
		if err != nil {
			return "", fmt.Errorf("if then: %w", err)
		}
		els, err := EncodeExpr(e.Else)
		if err != nil {
			return "", fmt.Errorf("if else: %w", err)
		}
		return fmt.Sprintf("(ite %s %s %s)", cond, then, els), nil

	case *core.Let:
		return encodeLet(e)

	case *core.Match:
		return encodeMatch(e)

	case *core.Intrinsic:
		// Pre-lowered intrinsic (shouldn't appear after op_lowering, but handle gracefully)
		return encodeIntrinsic(e)

	case *core.BinOp:
		return encodeBinOp(e)

	case *core.UnOp:
		return encodeUnOp(e)

	case *core.DictApp:
		return encodeDictApp(e)

	case *core.DictAbs:
		// Dictionary abstraction: transparently encode the body
		return EncodeExpr(e.Body)

	case *core.DictRef:
		// Dictionary reference: these are type class instances, skip in SMT
		return "", fmt.Errorf("dictionary reference cannot be encoded directly in SMT-LIB")

	case *core.Lambda:
		return "", fmt.Errorf("lambda expressions cannot be encoded in SMT-LIB (higher-order)")

	case *core.LetRec:
		return "", fmt.Errorf("recursive let bindings cannot be encoded in SMT-LIB")

	case *core.Tuple:
		return "", fmt.Errorf("tuple expressions cannot be encoded in SMT-LIB")

	case *core.Record:
		return encodeRecord(e)

	case *core.RecordAccess:
		return encodeRecordAccess(e)

	case *core.RecordUpdate:
		return encodeRecordUpdate(e)

	case *core.List:
		return encodeList(e)

	default:
		return "", fmt.Errorf("unsupported Core expression type %T", expr)
	}
}

// encodeLit encodes a literal value.
func encodeLit(lit *core.Lit) (string, error) {
	switch lit.Kind {
	case core.IntLit:
		v, ok := lit.Value.(int64)
		if !ok {
			return "", fmt.Errorf("IntLit with non-int64 value: %T", lit.Value)
		}
		if v < 0 {
			return fmt.Sprintf("(- %d)", -v), nil
		}
		return fmt.Sprintf("%d", v), nil
	case core.FloatLit:
		v, ok := lit.Value.(float64)
		if !ok {
			return "", fmt.Errorf("FloatLit with non-float64 value: %T", lit.Value)
		}
		if v < 0 {
			return fmt.Sprintf("(- %g)", -v), nil
		}
		return fmt.Sprintf("%g", v), nil
	case core.BoolLit:
		v, ok := lit.Value.(bool)
		if !ok {
			return "", fmt.Errorf("BoolLit with non-bool value: %T", lit.Value)
		}
		if v {
			return "true", nil
		}
		return "false", nil
	case core.UnitLit:
		return "", fmt.Errorf("unit literals cannot be encoded in SMT-LIB")
	case core.StringLit:
		v, ok := lit.Value.(string)
		if !ok {
			return "", fmt.Errorf("StringLit with non-string value: %T", lit.Value)
		}
		// SMT-LIB string literals are enclosed in double quotes with escaping
		return encodeStringLiteral(v), nil
	default:
		return "", fmt.Errorf("unknown literal kind: %d", lit.Kind)
	}
}

// encodeStringLiteral converts a Go string to an SMT-LIB string literal.
// SMT-LIB strings use "" for quotes and standard escapes.
func encodeStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, ch := range s {
		switch ch {
		case '"':
			b.WriteString(`""`) // SMT-LIB escapes " as ""
		case '\\':
			b.WriteString(`\\`)
		default:
			b.WriteRune(ch)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// stripConstructorPrefix removes the Core "make_TypeName_" prefix from constructor names.
// Core represents ADT constructors as "make_Season_LOW_SEASON" but SMT-LIB uses "LOW_SEASON".
func stripConstructorPrefix(name string) string {
	if strings.HasPrefix(name, "make_") {
		// Format: make_TypeName_ConstructorName
		// Find second underscore to extract constructor name
		rest := name[5:] // Remove "make_"
		if idx := strings.Index(rest, "_"); idx >= 0 {
			return rest[idx+1:]
		}
	}
	return name
}
