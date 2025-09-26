package main

import (
	"fmt"
	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/types"
)

func main() {
	fmt.Println("AILANG Type Inference Demo")
	fmt.Println("===========================\n")

	// Test 1: Simple literals
	testLiteral()

	// Test 2: Lambda functions
	testLambda()

	// Test 3: Let polymorphism
	testLetPolymorphism()

	// Test 4: Row polymorphism
	testRowPolymorphism()

	// Test 5: Type class constraints
	testTypeClasses()
}

func testLiteral() {
	fmt.Println("Test 1: Simple Literals")
	fmt.Println("-----------------------")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// Integer literal
	intLit := &ast.Literal{Kind: ast.IntLit, Value: 42}
	typ, eff, _ := ctx.Infer(intLit)
	fmt.Printf("42 : %s ! %s\n", typ, eff)

	// String literal
	strLit := &ast.Literal{Kind: ast.StringLit, Value: "hello"}
	typ, eff, _ = ctx.Infer(strLit)
	fmt.Printf("\"hello\" : %s ! %s\n", typ, eff)

	fmt.Println()
}

func testLambda() {
	fmt.Println("Test 2: Lambda Functions")
	fmt.Println("------------------------")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// Identity function: \x. x
	idLambda := &ast.Lambda{
		Params: []*ast.Param{{Name: "x"}},
		Body:   &ast.Identifier{Name: "x"},
	}

	typ, _, _ := ctx.Infer(idLambda)
	sub, _, _ := ctx.SolveConstraints()
	finalType := types.ApplySubstitution(sub, typ)
	fmt.Printf("\\x. x : %s\n", finalType)
	fmt.Printf("(Identity function is polymorphic)\n")

	fmt.Println()
}

func testLetPolymorphism() {
	fmt.Println("Test 3: Let Polymorphism")
	fmt.Println("------------------------")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// let id = \x. x in {id(42), id(true)}
	idLambda := &ast.Lambda{
		Params: []*ast.Param{{Name: "x"}},
		Body:   &ast.Identifier{Name: "x"},
	}

	letExpr := &ast.Let{
		Name:  "id",
		Value: idLambda,
		Body: &ast.Tuple{
			Elements: []ast.Expr{
				&ast.FuncCall{
					Func: &ast.Identifier{Name: "id"},
					Args: []ast.Expr{&ast.Literal{Kind: ast.IntLit, Value: 42}},
				},
				&ast.FuncCall{
					Func: &ast.Identifier{Name: "id"},
					Args: []ast.Expr{&ast.Literal{Kind: ast.BoolLit, Value: true}},
				},
			},
		},
	}

	typ, _, _ := ctx.Infer(letExpr)
	sub, _, _ := ctx.SolveConstraints()
	finalType := types.ApplySubstitution(sub, typ)
	fmt.Printf("let id = \\x.x in {id(42), id(true)} : %s\n", finalType)
	fmt.Printf("(Polymorphic function used at different types)\n")

	fmt.Println()
}

func testRowPolymorphism() {
	fmt.Println("Test 4: Row Polymorphism")
	fmt.Println("------------------------")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// \r. r.name - record selection
	getName := &ast.Lambda{
		Params: []*ast.Param{{Name: "r"}},
		Body: &ast.RecordAccess{
			Record: &ast.Identifier{Name: "r"},
			Field:  "name",
		},
	}

	typ, _, _ := ctx.Infer(getName)
	sub, _, _ := ctx.SolveConstraints()
	finalType := types.ApplySubstitution(sub, typ)
	fmt.Printf("\\r. r.name : %s\n", finalType)
	fmt.Printf("(Works with any record containing 'name' field)\n")

	fmt.Println()
}

func testTypeClasses() {
	fmt.Println("Test 5: Type Class Constraints")
	fmt.Println("------------------------------")

	ctx := types.NewInferenceContext()
	ctx.SetEnv(types.NewTypeEnvWithBuiltins())

	// \x. \y. x + y
	addLambda := &ast.Lambda{
		Params: []*ast.Param{{Name: "x"}},
		Body: &ast.Lambda{
			Params: []*ast.Param{{Name: "y"}},
			Body: &ast.BinaryOp{
				Left:  &ast.Identifier{Name: "x"},
				Op:    "+",
				Right: &ast.Identifier{Name: "y"},
			},
		},
	}

	typ, _, _ := ctx.Infer(addLambda)
	sub, unsolved, _ := ctx.SolveConstraints()
	finalType := types.ApplySubstitution(sub, typ)
	fmt.Printf("\\x. \\y. x + y : %s\n", finalType)
	if len(unsolved) > 0 {
		fmt.Printf("Unsolved constraints: ")
		for _, c := range unsolved {
			fmt.Printf("%s ", c)
		}
		fmt.Printf("\n(Type classes not yet implemented)\n")
	}

	fmt.Println()
}