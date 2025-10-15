// Package linked provides the linking phase of compilation
package linked

import (
	"fmt"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// Program represents a linked program
type Program struct {
	Main core.CoreExpr
}

// Linker performs dictionary linking
type Linker struct{}

// NewLinker creates a new linker
func NewLinker() *Linker {
	return &Linker{}
}

// Link resolves dictionary references
func (l *Linker) Link(expr core.CoreExpr, dictReg *types.DictionaryRegistry) (core.CoreExpr, error) {
	// For now, just pass through
	// Full implementation would resolve DictRef nodes to actual dictionaries
	return linkExpr(expr, dictReg), nil
}

func linkExpr(expr core.CoreExpr, dictReg *types.DictionaryRegistry) core.CoreExpr {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *core.DictRef:
		// Look up dictionary in registry
		dictKey := fmt.Sprintf("%s[%s]", e.ClassName, e.TypeName)
		// For now, just verify the key format is valid
		// Full implementation would resolve to actual dictionary
		_ = dictKey
		return e

	case *core.DictApp:
		return &core.DictApp{
			CoreNode: e.CoreNode,
			Dict:     linkExpr(e.Dict, dictReg),
			Method:   e.Method,
			Args:     linkExprs(e.Args, dictReg),
		}

	case *core.Let:
		return &core.Let{
			CoreNode: e.CoreNode,
			Name:     e.Name,
			Value:    linkExpr(e.Value, dictReg),
			Body:     linkExpr(e.Body, dictReg),
		}

	case *core.LetRec:
		var bindings []core.RecBinding
		for _, b := range e.Bindings {
			bindings = append(bindings, core.RecBinding{
				Name:  b.Name,
				Value: linkExpr(b.Value, dictReg),
			})
		}
		return &core.LetRec{
			CoreNode: e.CoreNode,
			Bindings: bindings,
			Body:     linkExpr(e.Body, dictReg),
		}

	case *core.Lambda:
		return &core.Lambda{
			CoreNode: e.CoreNode,
			Params:   e.Params,
			Body:     linkExpr(e.Body, dictReg),
		}

	case *core.App:
		return &core.App{
			CoreNode: e.CoreNode,
			Func:     linkExpr(e.Func, dictReg),
			Args:     linkExprs(e.Args, dictReg),
		}

	case *core.BinOp:
		return &core.BinOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Left:     linkExpr(e.Left, dictReg),
			Right:    linkExpr(e.Right, dictReg),
		}

	case *core.UnOp:
		return &core.UnOp{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Operand:  linkExpr(e.Operand, dictReg),
		}

	case *core.If:
		return &core.If{
			CoreNode: e.CoreNode,
			Cond:     linkExpr(e.Cond, dictReg),
			Then:     linkExpr(e.Then, dictReg),
			Else:     linkExpr(e.Else, dictReg),
		}

	case *core.Match:
		var arms []core.MatchArm
		for _, arm := range e.Arms {
			arms = append(arms, core.MatchArm{
				Pattern: arm.Pattern,
				Body:    linkExpr(arm.Body, dictReg),
			})
		}
		return &core.Match{
			CoreNode:   e.CoreNode,
			Scrutinee:  linkExpr(e.Scrutinee, dictReg),
			Arms:       arms,
			Exhaustive: e.Exhaustive,
		}

	case *core.Record:
		fields := make(map[string]core.CoreExpr)
		for k, v := range e.Fields {
			fields[k] = linkExpr(v, dictReg)
		}
		return &core.Record{
			CoreNode: e.CoreNode,
			Fields:   fields,
		}

	case *core.RecordAccess:
		return &core.RecordAccess{
			CoreNode: e.CoreNode,
			Record:   linkExpr(e.Record, dictReg),
			Field:    e.Field,
		}

	case *core.List:
		return &core.List{
			CoreNode: e.CoreNode,
			Elements: linkExprs(e.Elements, dictReg),
		}

	case *core.Intrinsic:
		// Intrinsic nodes pass through - they'll be handled by OpLowering pass
		return &core.Intrinsic{
			CoreNode: e.CoreNode,
			Op:       e.Op,
			Args:     linkExprs(e.Args, dictReg),
		}

	// Atomic expressions - return as is
	case *core.Var, *core.Lit, *core.DictAbs, *core.VarGlobal:
		return expr

	default:
		// Unknown type - return as is
		return expr
	}
}

func linkExprs(exprs []core.CoreExpr, dictReg *types.DictionaryRegistry) []core.CoreExpr {
	var result []core.CoreExpr
	for _, e := range exprs {
		result = append(result, linkExpr(e, dictReg))
	}
	return result
}
