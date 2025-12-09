// Package golang provides Go code generation from AILANG Core AST.
package golang

import (
	"github.com/sunholo/ailang/internal/core"
)

// generateLet generates a Go variable binding.
// M-DX25.2: Use SEPARATE types for variable (value's type) and IIFE return (body's type).
// Bug fix: Originally used let's type for both, but let's type IS body's type, not value's type.
func (g *Generator) generateLet(let *core.Let) error {
	// M-DX26: In _impl functions, everything is interface{}
	inImplFunc := g.expectedReturnType == "interface{}"

	// M-DX25.2 FIX: Variable type comes from VALUE expression, not the let expression
	varType := "interface{}"

	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record expressions - infer type from fields
		// TypeMapper returns "struct{}" for TRecord, but we can get the proper
		// struct name by looking up the record type from its fields.
		if rec, isRec := let.Value.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				varType = "*" + recordType.Name // Records are generated as pointers
			}
		} else if g.coreTypeInfo != nil {
			valueNodeID := g.getExprNodeID(let.Value)
			if valueNodeID != 0 {
				if typ, ok := g.coreTypeInfo[valueNodeID]; ok {
					if goType, err := g.TypeMapper.MapType(typ); err == nil {
						varType = string(goType)
					}
				}
			}
		}
	}

	// M-DX25.2 FIX: Return type comes from LET expression (= body's type)
	returnType := "interface{}"
	// M-DX26: Skip type inference in _impl functions
	if !inImplFunc {
		// M-DX25.10: Special case for Record body - infer type from fields
		if rec, isRec := let.Body.(*core.Record); isRec {
			fieldNames := make(map[string]bool, len(rec.Fields))
			for name := range rec.Fields {
				fieldNames[name] = true
			}
			if recordType := g.GetRecordTypeByFields(fieldNames); recordType != nil {
				returnType = "*" + recordType.Name
			}
		} else if g.coreTypeInfo != nil {
			if typ, ok := g.coreTypeInfo[let.NodeID]; ok {
				if goType, err := g.TypeMapper.MapType(typ); err == nil {
					returnType = string(goType)
				}
			}
		}
	}

	g.writef("func() %s {\n", returnType)
	g.indent++
	g.writef("var %s %s = ", ToGoVarName(let.Name), varType)

	// Add type assertion if value produces interface{} but we need concrete varType
	needsValueAssertion := varType != "interface{}" && g.exprProducesInterface(let.Value)
	if err := g.generateExpr(let.Value); err != nil {
		return err
	}
	if needsValueAssertion {
		g.writef(".(%s)", varType)
	}
	g.writef("\n")

	g.writef("_ = %s // suppress unused\n", ToGoVarName(let.Name))
	g.writef("return ")

	// Add type assertion if body produces interface{} but we need concrete returnType
	// M-DX25.10: Special case - if body is just the variable we declared, we know its type
	needsBodyAssertion := false
	if v, isVar := let.Body.(*core.Var); isVar && v.Name == let.Name {
		// Body is just the variable we declared - its Go type is varType
		needsBodyAssertion = returnType != "interface{}" && varType == "interface{}"
	} else {
		needsBodyAssertion = returnType != "interface{}" && g.exprProducesInterface(let.Body)
	}
	if err := g.generateExpr(let.Body); err != nil {
		return err
	}
	if needsBodyAssertion {
		g.writef(".(%s)", returnType)
	}
	g.writef("\n")

	g.indent--
	g.write("}()")
	return nil
}

// generateLetRec generates recursive function bindings.
func (g *Generator) generateLetRec(letrec *core.LetRec) error {
	g.writef("func() interface{} {\n")
	g.indent++

	// Declare all bindings first
	for _, bind := range letrec.Bindings {
		g.writef("var %s func(...interface{}) interface{}\n", ToGoVarName(bind.Name))
	}

	// Assign values
	for _, bind := range letrec.Bindings {
		g.writef("%s = func(args ...interface{}) interface{} {\n", ToGoVarName(bind.Name))
		g.indent++
		g.writef("return ")
		if err := g.generateExpr(bind.Value); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
		g.writef("}\n")
	}

	g.writef("return ")
	if err := g.generateExpr(letrec.Body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.write("}()")
	return nil
}

// getExprNodeID extracts the NodeID from a CoreExpr.
// M-DX25.2: Used to look up value expression's type separately from let's type.
func (g *Generator) getExprNodeID(expr core.CoreExpr) uint64 {
	if expr == nil {
		return 0
	}
	return expr.ID()
}
