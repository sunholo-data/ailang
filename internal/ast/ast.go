package ast

import (
	"fmt"
	"strings"
)

// Package ast defines the Abstract Syntax Tree (AST) nodes for AILANG.
//
// # File Organization
//
// The AST package is split into several files by node type:
//   - ast.go: Core interfaces, base types, and File/Module nodes (THIS FILE)
//   - ast_expr.go: Expression nodes (Identifier, Literal, Lambda, etc.)
//   - ast_decl.go: Declaration nodes (FuncDecl, TypeDecl, TestDecl, etc.)
//   - ast_type.go: Type annotation nodes (SimpleType, FuncType, ListType, etc.)
//
// # Core Interfaces
//
// All AST nodes implement the Node interface:
//   - Node: Base interface with String() and Position()
//   - Expr: Expression nodes (exprNode() marker)
//   - Stmt: Statement nodes (stmtNode() marker)
//   - Type: Type annotation nodes (typeNode() marker)
//   - Pattern: Pattern matching nodes (patternNode() marker)
//
// # See Also
//
//   - internal/parser: Builds AST from tokens
//   - internal/elaborate: Converts Surface AST to Core AST
//   - internal/types: Type checking and inference

// Node is the base interface for all AST nodes
type Node interface {
	String() string
	Position() Pos
}

// Pos represents a position in the source code
type Pos struct {
	Line   int
	Column int
	File   string
	Offset int // Byte offset for SID calculation
}

// Span represents a range in source code
type Span struct {
	Start Pos
	End   Pos
}

func (p Pos) String() string {
	return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
}

// File represents a complete AILANG source file
type File struct {
	Module     *ModuleDecl   // Optional module declaration
	Imports    []*ImportDecl // Import declarations
	Decls      []Node        // Top-level declarations (deprecated, use Funcs/Statements)
	Funcs      []*FuncDecl   // Function declarations
	Statements []Node        // Top-level statements/expressions
	Path       string        // File path for validation
	Pos        Pos
}

// ModuleDecl represents a module declaration
type ModuleDecl struct {
	Path string // e.g., "foo/bar"
	Pos  Pos
	Span Span // For SID calculation
}

// ImportDecl represents an import declaration
type ImportDecl struct {
	Path    string   // Module path to import
	Symbols []string // Selective imports (empty = whole module)
	Pos     Pos
	Span    Span
}

func (f *File) String() string {
	parts := []string{}
	if f.Module != nil {
		parts = append(parts, fmt.Sprintf("module %s", f.Module.Path))
	}
	for _, imp := range f.Imports {
		parts = append(parts, imp.String())
	}
	for _, decl := range f.Decls {
		parts = append(parts, decl.String())
	}
	return strings.Join(parts, "\n")
}
func (f *File) Position() Pos { return f.Pos }

func (m *ModuleDecl) String() string {
	return fmt.Sprintf("module %s", m.Path)
}
func (m *ModuleDecl) Position() Pos { return m.Pos }

func (i *ImportDecl) String() string {
	if len(i.Symbols) > 0 {
		return fmt.Sprintf("import %s (%s)", i.Path, strings.Join(i.Symbols, ", "))
	}
	return fmt.Sprintf("import %s", i.Path)
}
func (i *ImportDecl) Position() Pos { return i.Pos }

// Expression nodes
type Expr interface {
	Node
	exprNode()
}

// Statement nodes (though AILANG is expression-based)
type Stmt interface {
	Node
	stmtNode()
}

// Type nodes
type Type interface {
	Node
	typeNode()
}

// Pattern nodes for pattern matching
type Pattern interface {
	Node
	patternNode()
}

// Program represents the entire program
type Program struct {
	File   *File   // New: Use File instead of Module
	Module *Module // Legacy: Keep for compatibility
}

func (p *Program) String() string {
	if p.Module != nil {
		return p.Module.String()
	}
	return "empty program"
}
