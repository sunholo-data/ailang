package types

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
)

// TypeChecker is the main type checking interface
type TypeChecker struct {
	errors []error
}

// NewTypeChecker creates a new type checker
func NewTypeChecker() *TypeChecker {
	return &TypeChecker{
		errors: []error{},
	}
}

// CheckProgram type checks an entire program
func (tc *TypeChecker) CheckProgram(program *ast.Program) (*TypedProgram, error) {
	typed := &TypedProgram{
		Statements: make([]TypedStatement, 0),
	}
	
	// Create global environment with builtins
	globalEnv := NewTypeEnvWithBuiltins()
	
	if program.Module != nil {
		for _, decl := range program.Module.Decls {
			typedStmt, env, err := tc.checkDecl(decl, globalEnv)
			if err != nil {
				tc.errors = append(tc.errors, err)
				continue
			}
			if typedStmt != nil {
				typed.Statements = append(typed.Statements, typedStmt)
			}
			globalEnv = env // Update environment with new bindings
		}
	}
	
	if len(tc.errors) > 0 {
		var errList ErrorList
		for _, err := range tc.errors {
			if typeErr, ok := err.(*TypeCheckError); ok {
				errList = append(errList, typeErr)
			}
		}
		return nil, errList
	}
	
	return typed, nil
}

// checkDecl type checks a declaration
func (tc *TypeChecker) checkDecl(decl ast.Node, env *TypeEnv) (TypedStatement, *TypeEnv, error) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		// Type check function body
		ctx := NewInferenceContext()
		ctx.env = env
		
		// Add parameters to environment
		paramTypes := make([]Type, len(d.Params))
		for i, param := range d.Params {
			// If type annotation exists, use it; otherwise fresh var
			var paramType Type
			if param.Type != nil {
				paramType = tc.astTypeToType(param.Type)
			} else {
				paramType = ctx.freshTypeVar()
			}
			paramTypes[i] = paramType
			ctx.env = ctx.env.Extend(param.Name, paramType)
		}
		
		// Infer body type
		bodyType, bodyEffects, err := ctx.Infer(d.Body)
		if err != nil {
			return nil, env, err
		}
		
		// Solve constraints
		sub, unsolved, err := ctx.SolveConstraints()
		if err != nil {
			return nil, env, err
		}
		
		// Report unsolved class constraints
		if len(unsolved) > 0 {
			for _, c := range unsolved {
				tc.errors = append(tc.errors, 
					NewUnsolvedConstraintError(c.Class, c.Type, c.Path))
			}
		}
		
		// Apply substitution
		for i := range paramTypes {
			paramTypes[i] = ApplySubstitution(sub, paramTypes[i])
		}
		bodyType = ApplySubstitution(sub, bodyType)
		bodyEffects = ApplySubstitution(sub, bodyEffects).(*Row)
		
		// Create function type
		fnType := &TFunc2{
			Params:    paramTypes,
			EffectRow: bodyEffects,
			Return:    bodyType,
		}
		
		// Generalize if pure
		var binding interface{}
		if d.IsPure || isValue(d.Body) {
			binding = ctx.generalize(fnType, EmptyEffectRow())
		} else {
			binding = fnType
		}
		
		// Add to environment
		newEnv := env
		if scheme, ok := binding.(*Scheme); ok {
			newEnv = env.ExtendScheme(d.Name, scheme)
		} else {
			newEnv = env.Extend(d.Name, binding.(Type))
		}
		
		return &TypedFunctionDeclaration{
			Name:       d.Name,
			Params:     nil, // Convert params
			ParamTypes: paramTypes,
			Body:       d.Body,
			BodyType:   bodyType,
			Effects:    bodyEffects,
		}, newEnv, nil
		
	case ast.Expr:
		// Expression as a top-level declaration
		typedExpr, err := tc.checkExpression(d, env)
		if err != nil {
			return nil, env, err
		}
		return &TypedExpressionStatement{
			Expression: typedExpr,
		}, env, nil
		
	default:
		return nil, env, fmt.Errorf("type checking not implemented for %T", decl)
	}
}

// checkExpression type checks an expression
func (tc *TypeChecker) checkExpression(expr ast.Expr, env *TypeEnv) (*TypedExpression, error) {
	ctx := NewInferenceContext()
	ctx.env = env
	
	// Infer type
	typ, effects, err := ctx.Infer(expr)
	if err != nil {
		return nil, err
	}
	
	// Solve constraints
	sub, unsolved, err := ctx.SolveConstraints()
	if err != nil {
		return nil, err
	}
	
	// Report unsolved class constraints
	if len(unsolved) > 0 {
		for _, c := range unsolved {
			tc.errors = append(tc.errors, 
				NewUnsolvedConstraintError(c.Class, c.Type, c.Path))
		}
	}
	
	// Apply substitution
	finalType := ApplySubstitution(sub, typ)
	finalEffects := ApplySubstitution(sub, effects).(*Row)
	
	return &TypedExpression{
		Expr:    expr,
		Type:    finalType,
		Effects: finalEffects,
	}, nil
}

// astTypeToType converts an AST type to an internal type
func (tc *TypeChecker) astTypeToType(t ast.Type) Type {
	switch typ := t.(type) {
	case *ast.SimpleType:
		switch typ.Name {
		case "int":
			return TInt
		case "float":
			return TFloat
		case "string":
			return TString
		case "bool":
			return TBool
		case "()":
			return TUnit
		case "bytes":
			return TBytes
		default:
			// Type variable or constructor
			if isLowerCase(typ.Name) {
				return &TVar2{Name: typ.Name, Kind: Star}
			}
			return &TCon{Name: typ.Name}
		}
		
	case *ast.FuncType:
		paramTypes := make([]Type, len(typ.Params))
		for i, p := range typ.Params {
			paramTypes[i] = tc.astTypeToType(p)
		}
		
		// Handle effects
		var effectRow *Row
		if len(typ.Effects) > 0 {
			labels := make(map[string]Type)
			for _, e := range typ.Effects {
				labels[e] = TUnit
			}
			effectRow = &Row{
				Kind:   EffectRow,
				Labels: labels,
				Tail:   nil,
			}
		} else {
			effectRow = EmptyEffectRow()
		}
		
		return &TFunc2{
			Params:    paramTypes,
			EffectRow: effectRow,
			Return:    tc.astTypeToType(typ.Return),
		}
		
	case *ast.ListType:
		return &TList{
			Element: tc.astTypeToType(typ.Element),
		}
		
	case *ast.TupleType:
		elements := make([]Type, len(typ.Elements))
		for i, e := range typ.Elements {
			elements[i] = tc.astTypeToType(e)
		}
		return &TTuple{Elements: elements}
		
	default:
		// Unknown type, return type variable
		return &TVar2{Name: "unknown", Kind: Star}
	}
}

func isLowerCase(s string) bool {
	return len(s) > 0 && s[0] >= 'a' && s[0] <= 'z'
}

// TypedProgram represents a type-checked program
type TypedProgram struct {
	Statements []TypedStatement
}

// TypedStatement represents a type-checked statement
type TypedStatement interface {
	typedStatement()
	GetType() Type
}

// TypedExpressionStatement is a typed expression statement
type TypedExpressionStatement struct {
	Expression *TypedExpression
}

func (s *TypedExpressionStatement) typedStatement() {}
func (s *TypedExpressionStatement) GetType() Type {
	return s.Expression.Type
}

// TypedFunctionDeclaration is a typed function declaration
type TypedFunctionDeclaration struct {
	Name       interface{} // Can be ast.Expression or nil
	Params     []interface{} // ast.Expression items
	ParamTypes []Type
	Body       interface{} // ast.Expression
	BodyType   Type
	Effects    *Row
}

func (f *TypedFunctionDeclaration) typedStatement() {}
func (f *TypedFunctionDeclaration) GetType() Type {
	return &TFunc2{
		Params:    f.ParamTypes,
		EffectRow: f.Effects,
		Return:    f.BodyType,
	}
}

// TypedExpression wraps an expression with its type
type TypedExpression struct {
	Expr    interface{} // ast.Expression
	Type    Type
	Effects *Row
}

// TypeCheckFile is a convenience function to type check a file
func TypeCheckFile(filename string, program *ast.Program) error {
	tc := NewTypeChecker()
	_, err := tc.CheckProgram(program)
	return err
}