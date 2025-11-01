#!/usr/bin/env bash
set -euo pipefail

# List all AST node types in AILANG

echo "AILANG AST Node Types"
echo "===================="
echo ""

echo "Expression types:"
echo "----------------"
grep "^type.*struct" internal/ast/ast.go | grep -E "(Literal|List|Variable|FuncCall|Lambda|Block|If|Match)" || true

echo ""
echo "Type types:"
echo "----------"
grep "^type.*struct" internal/ast/ast.go | grep -E "(SimpleType|ListType|FuncType|TypeApp)" || true

echo ""
echo "Pattern types:"
echo "-------------"
grep "^type.*struct" internal/ast/ast.go | grep -E "(VarPattern|ConstructorPattern|LiteralPattern|WildcardPattern)" || true

echo ""
echo "Declaration types:"
echo "-----------------"
grep "^type.*struct" internal/ast/ast.go | grep -E "(FuncDecl|TypeDecl|LetDecl)" || true

echo ""
echo "Full list:"
echo "---------"
grep "^type.*struct" internal/ast/ast.go | head -30
