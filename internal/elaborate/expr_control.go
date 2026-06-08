package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// Control flow expression normalization: if, let, letrec, block

// normalizeIf handles conditionals
func (e *Elaborator) normalizeIf(ifExpr *ast.If) (core.CoreExpr, error) {
	// Check for let-without-body in branches (common mistake for FP users)
	// This pattern: if x then y else let v = ...; expr
	// requires explicit braces: if x then y else { let v = ...; expr }
	if letExpr, ok := ifExpr.Then.(*ast.Let); ok && letExpr.Body == nil {
		return nil, e.makeLetInBranchError(letExpr, "then", ifExpr)
	}
	if letExpr, ok := ifExpr.Else.(*ast.Let); ok && letExpr.Body == nil {
		return nil, e.makeLetInBranchError(letExpr, "else", ifExpr)
	}

	// Condition must be atomic
	cond, condBinds, err := e.normalizeToAtomic(ifExpr.Condition)
	if err != nil {
		return nil, err
	}

	// Branches can be complex
	thenBranch, err := e.normalize(ifExpr.Then)
	if err != nil {
		return nil, err
	}

	elseBranch, err := e.normalize(ifExpr.Else)
	if err != nil {
		return nil, err
	}

	result := &core.If{
		CoreNode: e.makeNode(ifExpr.Position()),
		Cond:     cond,
		Then:     thenBranch,
		Else:     elseBranch,
	}

	return e.wrapWithBindings(result, condBinds), nil
}

// makeLetInBranchError creates a helpful error message when a let-without-body
// is used directly in an if-else branch (common mistake for ML/Haskell users)
func (e *Elaborator) makeLetInBranchError(letExpr *ast.Let, branch string, ifExpr *ast.If) error {
	pos := letExpr.Position()
	ifPos := ifExpr.Position()

	return fmt.Errorf(`if-else branches require explicit braces when using let bindings

  %s:%d:%d: if expression starts here
  %s:%d:%d: let binding in '%s' branch has no body

The parser cannot determine where multi-statement branches end without braces.
The 'let %s = ...' is parsed as the entire %s branch expression.

Fix: Wrap the %s branch in braces:
    %s {
        let %s = ...;
        <your expression here>
    }

For more details, see: docs/LIMITATIONS.md#if-else-branches-require-explicit-braces`,
		ifPos.File, ifPos.Line, ifPos.Column,
		pos.File, pos.Line, pos.Column,
		branch,
		letExpr.Name, branch,
		branch,
		branch,
		letExpr.Name,
	)
}

// normalizeLet handles let bindings
func (e *Elaborator) normalizeLet(let *ast.Let) (core.CoreExpr, error) {
	// Handle let statements without body (e.g., "let x = 1;" in a block)
	// These are created by the parser for semicolon-terminated lets
	// They should only appear as part of a Block, which will sequence them properly
	if let.Body == nil {
		// Just normalize the value and return it wrapped in a Let that binds to Unit
		// The Block normalization will thread this through properly
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		// Return a Let that binds the value but returns Unit
		// This allows the value to be computed (for side effects) and the binding to be visible
		// in subsequent expressions in the block
		return &core.Let{
			CoreNode: e.makeNode(let.Position()),
			Name:     let.Name,
			Value:    value,
			Body: &core.Lit{
				CoreNode: e.makeNode(let.Position()),
				Kind:     core.UnitLit,
				Value:    "()",
			},
		}, nil
	}

	// Check if it's recursive (let rec)
	isRec := false
	// For now, detect recursion by checking if the value references the name
	// This is simplified - full implementation would analyze the value expression

	if isRec {
		// Handle recursive binding
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		return &core.LetRec{
			CoreNode: e.makeNode(let.Position()),
			Bindings: []core.RecBinding{{Name: let.Name, Value: value}},
			Body:     body,
		}, nil
	} else {
		// Non-recursive let with body (let x = 1 in x + 1)
		value, err := e.normalize(let.Value)
		if err != nil {
			return nil, err
		}

		body, err := e.normalize(let.Body)
		if err != nil {
			return nil, err
		}

		// ANF completion: if the value is a nested Let expression, flatten it.
		// This handles cases like: let npc = { pos: { x: 10, y: 20 } } where
		// the nested record normalization produces Let bindings in the value.
		//
		// Before flattening: Let npc = (Let $tmp1 = inner in outer) in body
		// After flattening:  Let $tmp1 = inner in Let npc = outer in body
		innerBindings, flattenedValue := extractLetBindings(value)

		if len(innerBindings) == 0 {
			// No nested lets - simple case
			node := &core.Let{
				CoreNode: e.makeNode(let.Position()),
				Name:     let.Name,
				Value:    flattenedValue,
				Body:     body,
			}
			// M-TYPE-LIST-ELEMENT-SOUNDNESS: preserve the let's type annotation so
			// `let xs: [string] = [42]` is actually type-checked (was dropped here).
			if let.Type != nil {
				if annot := e.astTypeToInternalType(let.Type); annot != nil {
					e.letTypeAnnots[node.ID()] = annot
				}
			}
			return node, nil
		}

		// Build flattened structure: inner bindings outermost, user binding innermost
		// Start with: Let name = flattenedValue in body
		result := &core.Let{
			CoreNode: e.makeNode(let.Position()),
			Name:     let.Name,
			Value:    flattenedValue,
			Body:     body,
		}
		// M-TYPE-LIST-ELEMENT-SOUNDNESS: the annotation belongs to the binding of
		// let.Name, which is this inner `result` node.
		if let.Type != nil {
			if annot := e.astTypeToInternalType(let.Type); annot != nil {
				e.letTypeAnnots[result.ID()] = annot
			}
		}

		// Wrap with inner bindings in reverse order (innermost binding becomes outermost let)
		for i := len(innerBindings) - 1; i >= 0; i-- {
			bind := innerBindings[i]
			result = &core.Let{
				CoreNode: e.makeNode(bind.Value.Span()),
				Name:     bind.Name,
				Value:    bind.Value,
				Body:     result,
			}
		}

		return result, nil
	}
}

// normalizeLetRec handles recursive let bindings
// Syntax: letrec name = value in body
func (e *Elaborator) normalizeLetRec(letrec *ast.LetRec) (core.CoreExpr, error) {
	// Normalize value (which can reference the name being bound)
	value, err := e.normalize(letrec.Value)
	if err != nil {
		return nil, err
	}

	// Handle missing body (REPL case)
	if letrec.Body == nil {
		// Return a LetRec that binds the value but returns Unit
		return &core.LetRec{
			CoreNode: e.makeNode(letrec.Position()),
			Bindings: []core.RecBinding{{Name: letrec.Name, Value: value}},
			Body: &core.Lit{
				CoreNode: e.makeNode(letrec.Position()),
				Kind:     core.UnitLit,
				Value:    "()",
			},
		}, nil
	}

	// Normal case: letrec with body
	body, err := e.normalize(letrec.Body)
	if err != nil {
		return nil, err
	}

	return &core.LetRec{
		CoreNode: e.makeNode(letrec.Position()),
		Bindings: []core.RecBinding{{Name: letrec.Name, Value: value}},
		Body:     body,
	}, nil
}

// normalizeBlock converts a block of semicolon-separated expressions
// into nested Let expressions: { e1; e2; e3 } => let _ = e1 in let _ = e2 in e3
func (e *Elaborator) normalizeBlock(block *ast.Block) (core.CoreExpr, error) {
	// Empty block: should not happen but handle gracefully
	if len(block.Exprs) == 0 {
		// Return unit literal
		return &core.Lit{
			CoreNode: e.makeNode(block.Position()),
			Kind:     core.UnitLit,
			Value:    "()",
		}, nil
	}

	// Single expression: just normalize it directly
	if len(block.Exprs) == 1 {
		return e.normalize(block.Exprs[0])
	}

	// Multiple expressions: convert to nested Lets
	// Start with the last expression (the return value)
	result, err := e.normalize(block.Exprs[len(block.Exprs)-1])
	if err != nil {
		return nil, err
	}

	// Work backwards through the expressions, wrapping each in a Let
	for i := len(block.Exprs) - 2; i >= 0; i-- {
		expr := block.Exprs[i]

		// Special case: if this is a Let with nil body (statement form),
		// normalize its value and use the let's name directly
		if letExpr, ok := expr.(*ast.Let); ok && letExpr.Body == nil {
			value, err := e.normalize(letExpr.Value)
			if err != nil {
				return nil, err
			}

			letNode := &core.Let{
				CoreNode: e.makeNode(letExpr.Position()),
				Name:     letExpr.Name, // Use the actual let name, not _block_N
				Value:    value,
				Body:     result, // Thread through to next expression
			}
			// M-TYPE-LIST-ELEMENT-SOUNDNESS: preserve the statement-let's type
			// annotation (e.g. `let xs: [string] = [42];`) so it is type-checked.
			if letExpr.Type != nil {
				if annot := e.astTypeToInternalType(letExpr.Type); annot != nil {
					e.letTypeAnnots[letNode.ID()] = annot
				}
			}
			result = letNode
		} else {
			// Regular expression: normalize and bind to a wildcard
			value, err := e.normalize(expr)
			if err != nil {
				return nil, err
			}

			// Use a wildcard name for the binding since we're discarding the result
			// Generate unique names to avoid conflicts
			bindingName := fmt.Sprintf("_block_%d", i)

			result = &core.Let{
				CoreNode: e.makeNode(block.Position()),
				Name:     bindingName,
				Value:    value,
				Body:     result,
			}
		}
	}

	return result, nil
}

// normalizeForall handles bounded universal quantifier expressions.
// forall i: lo..hi => body  →  core.Forall{Var: "i", Lo: lo, Hi: hi, Body: body}
func (e *Elaborator) normalizeForall(fa *ast.ForallExpr) (core.CoreExpr, error) {
	lo, err := e.normalize(fa.Lo)
	if err != nil {
		return nil, fmt.Errorf("forall lower bound: %w", err)
	}

	hi, err := e.normalize(fa.Hi)
	if err != nil {
		return nil, fmt.Errorf("forall upper bound: %w", err)
	}

	body, err := e.normalize(fa.Body)
	if err != nil {
		return nil, fmt.Errorf("forall body: %w", err)
	}

	return &core.Forall{
		CoreNode: e.makeNode(fa.Position()),
		Var:      fa.Var,
		Lo:       lo,
		Hi:       hi,
		Body:     body,
	}, nil
}
