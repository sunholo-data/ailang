package ast

import (
	"fmt"
	"sort"
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

// EffectParam is a single key=value pair on a parameterised effect.
// Used in row markers like AI[mode=routeable], Rand[mode=crypto, scope=identity].
// Both Key and Value are bare identifiers (no quoting); structured values are
// out of scope for Phase 1 of M-EFFECT-REFINEMENT.
type EffectParam struct {
	Key   string
	Value string
	Pos   Pos
}

// EffectAnnotation represents an effect with optional parameters and budget constraints.
// Syntax:
//
//	IO                                   -- bare effect
//	Rand[mode=crypto]                    -- parameterised
//	Rand[mode=crypto, scope=identity]    -- multiple params
//	IO @limit=5                          -- bare with budget
//	AI[mode=routeable] @limit=10         -- parameterised + budget
//
// Row variables: lowercase identifiers like 'e' in ! {e} for effect polymorphism.
type EffectAnnotation struct {
	Name     string        // Effect name (e.g., "IO", "FS", "Net") or row variable (e.g., "e")
	IsRowVar bool          // True if this is an effect row variable (lowercase identifier)
	Params   []EffectParam // Optional parameter list (e.g. [mode=crypto]); empty/nil for bare effects
	Budget   *int          // Optional budget limit / max (nil = unlimited)
	Min      *int          // Optional minimum usage requirement (nil = no minimum) (M-DX25 M4)
	Pos      Pos
}

// String formats the effect annotation for display.
// Parameters are emitted in alphabetical-by-key order for golden-file determinism.
func (e *EffectAnnotation) String() string {
	head := e.Name
	if len(e.Params) > 0 {
		// Sort a copy so we never mutate caller-provided data.
		sorted := make([]EffectParam, len(e.Params))
		copy(sorted, e.Params)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
		paramParts := make([]string, len(sorted))
		for i, p := range sorted {
			paramParts[i] = fmt.Sprintf("%s=%s", p.Key, p.Value)
		}
		head = fmt.Sprintf("%s[%s]", e.Name, strings.Join(paramParts, ", "))
	}

	parts := []string{head}
	if e.Min != nil {
		parts = append(parts, fmt.Sprintf("@min=%d", *e.Min))
	}
	if e.Budget != nil {
		parts = append(parts, fmt.Sprintf("@limit=%d", *e.Budget))
	}
	if len(parts) == 1 {
		return head
	}
	return strings.Join(parts, " ")
}

// EffectNames extracts just the effect names from annotations (for backward compatibility)
func EffectNames(effects []EffectAnnotation) []string {
	names := make([]string, len(effects))
	for i, e := range effects {
		names[i] = e.Name
	}
	return names
}

// FormatEffects formats a slice of effect annotations for display
func FormatEffects(effects []EffectAnnotation) string {
	if len(effects) == 0 {
		return ""
	}
	parts := make([]string, len(effects))
	for i, e := range effects {
		parts[i] = e.String()
	}
	return fmt.Sprintf("! {%s}", strings.Join(parts, ", "))
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
	Path          string            // Module path to import
	Symbols       []string          // Selective imports (empty = whole module)
	ModuleAlias   string            // Module alias: "import std/list as List" -> "List"
	SymbolAliases map[string]string // Symbol aliases: original -> alias (e.g., "length" -> "stringLength")
	IsPackage     bool              // True if import uses pkg/ prefix (external package)
	PackageName   string            // Package name (vendor/name) extracted from pkg/ import path
	IsRelative    bool              // True if import uses ./ prefix (intra-package sibling)
	RelativePath  string            // Relative portion after ./ (e.g., "plan" from "./plan")
	Pos           Pos
	Span          Span
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
	var result string
	if i.ModuleAlias != "" {
		result = fmt.Sprintf("import %s as %s", i.Path, i.ModuleAlias)
	} else {
		result = fmt.Sprintf("import %s", i.Path)
	}
	if len(i.Symbols) > 0 {
		// Format symbols with aliases
		var symStrs []string
		for _, sym := range i.Symbols {
			if alias, ok := i.SymbolAliases[sym]; ok {
				symStrs = append(symStrs, fmt.Sprintf("%s as %s", sym, alias))
			} else {
				symStrs = append(symStrs, sym)
			}
		}
		result += fmt.Sprintf(" (%s)", strings.Join(symStrs, ", "))
	}
	return result
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
