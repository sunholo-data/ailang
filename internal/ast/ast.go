package ast

import (
	"fmt"
	"strings"
)

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

// Identifier represents a variable or function name
type Identifier struct {
	Name string
	Pos  Pos
}

func (i *Identifier) String() string { return i.Name }
func (i *Identifier) Position() Pos  { return i.Pos }
func (i *Identifier) exprNode()      {}
func (i *Identifier) patternNode()   {}

// Literal represents a literal value
type Literal struct {
	Kind  LiteralKind
	Value interface{}
	Pos   Pos
}

type LiteralKind int

const (
	IntLit LiteralKind = iota
	FloatLit
	StringLit
	BoolLit
	UnitLit
)

func (l *Literal) String() string {
	return fmt.Sprintf("%v", l.Value)
}
func (l *Literal) Position() Pos { return l.Pos }
func (l *Literal) exprNode()     {}
func (l *Literal) patternNode()  {}

// BinaryOp represents a binary operation
type BinaryOp struct {
	Left  Expr
	Op    string
	Right Expr
	Pos   Pos
}

func (b *BinaryOp) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Op, b.Right)
}
func (b *BinaryOp) Position() Pos { return b.Pos }
func (b *BinaryOp) exprNode()     {}

// UnaryOp represents a unary operation
type UnaryOp struct {
	Op   string
	Expr Expr
	Pos  Pos
}

func (u *UnaryOp) String() string {
	return fmt.Sprintf("(%s %s)", u.Op, u.Expr)
}
func (u *UnaryOp) Position() Pos { return u.Pos }
func (u *UnaryOp) exprNode()     {}

// Lambda represents a lambda expression
type Lambda struct {
	Params  []*Param
	Body    Expr
	Effects []string // Effect annotations
	Pos     Pos
}

type Param struct {
	Name string
	Type Type
	Pos  Pos
}

func (l *Lambda) String() string {
	params := []string{}
	for _, p := range l.Params {
		params = append(params, p.Name)
	}
	return fmt.Sprintf("\\%s. %s", strings.Join(params, " "), l.Body)
}
func (l *Lambda) Position() Pos { return l.Pos }
func (l *Lambda) exprNode()     {}

// FuncLit represents an anonymous function literal (func expression)
// Syntax: func(x: int, y: int) -> int { x + y }
// This desugars to Lambda in the elaboration phase
type FuncLit struct {
	Params     []*Param
	ReturnType Type     // Optional return type annotation
	Effects    []string // Effect annotations
	Body       Expr
	Pos        Pos
}

func (f *FuncLit) String() string {
	params := []string{}
	for _, p := range f.Params {
		if p.Type != nil {
			params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
		} else {
			params = append(params, p.Name)
		}
	}
	retType := ""
	if f.ReturnType != nil {
		retType = fmt.Sprintf(" -> %s", f.ReturnType)
	}
	effects := ""
	if len(f.Effects) > 0 {
		effects = fmt.Sprintf(" ! {%s}", strings.Join(f.Effects, ", "))
	}
	return fmt.Sprintf("func(%s)%s%s { %s }", strings.Join(params, ", "), retType, effects, f.Body)
}
func (f *FuncLit) Position() Pos { return f.Pos }
func (f *FuncLit) exprNode()     {}

// FuncCall represents a function application
type FuncCall struct {
	Func Expr
	Args []Expr
	Pos  Pos
}

func (f *FuncCall) String() string {
	args := []string{}
	for _, a := range f.Args {
		args = append(args, a.String())
	}
	return fmt.Sprintf("(%s %s)", f.Func, strings.Join(args, " "))
}
func (f *FuncCall) Position() Pos { return f.Pos }
func (f *FuncCall) exprNode()     {}

// Let represents a let binding
type Let struct {
	Name  string
	Type  Type // Optional type annotation
	Value Expr
	Body  Expr
	Pos   Pos
}

func (l *Let) String() string {
	return fmt.Sprintf("(let %s = %s in %s)", l.Name, l.Value, l.Body)
}
func (l *Let) Position() Pos { return l.Pos }
func (l *Let) exprNode()     {}

// LetRec represents a recursive let binding
// Syntax: letrec name = value in body
// The name is in scope in the value expression (for recursion)
type LetRec struct {
	Name  string
	Type  Type // Optional type annotation
	Value Expr
	Body  Expr
	Pos   Pos
}

func (l *LetRec) String() string {
	return fmt.Sprintf("(letrec %s = %s in %s)", l.Name, l.Value, l.Body)
}
func (l *LetRec) Position() Pos { return l.Pos }
func (l *LetRec) exprNode()     {}

// Block represents a sequence of expressions separated by semicolons
// The last expression is the return value, others are evaluated for effects
type Block struct {
	Exprs []Expr
	Pos   Pos
}

func (b *Block) String() string {
	var parts []string
	for _, expr := range b.Exprs {
		parts = append(parts, expr.String())
	}
	return fmt.Sprintf("{ %s }", strings.Join(parts, "; "))
}
func (b *Block) Position() Pos { return b.Pos }
func (b *Block) exprNode()     {}

// If represents a conditional expression
type If struct {
	Condition Expr
	Then      Expr
	Else      Expr
	Pos       Pos
}

func (i *If) String() string {
	return fmt.Sprintf("(if %s then %s else %s)", i.Condition, i.Then, i.Else)
}
func (i *If) Position() Pos { return i.Pos }
func (i *If) exprNode()     {}

// Match represents pattern matching
type Match struct {
	Expr  Expr
	Cases []*Case
	Pos   Pos
}

type Case struct {
	Pattern Pattern
	Guard   Expr // Optional guard clause
	Body    Expr
	Pos     Pos
}

func (m *Match) String() string {
	cases := []string{}
	for _, c := range m.Cases {
		cases = append(cases, fmt.Sprintf("%s => %s", c.Pattern, c.Body))
	}
	return fmt.Sprintf("(match %s { %s })", m.Expr, strings.Join(cases, " | "))
}
func (m *Match) Position() Pos { return m.Pos }
func (m *Match) exprNode()     {}

// List represents a list literal
type List struct {
	Elements []Expr
	Pos      Pos
}

func (l *List) String() string {
	elems := []string{}
	for _, e := range l.Elements {
		elems = append(elems, e.String())
	}
	return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
}
func (l *List) Position() Pos { return l.Pos }
func (l *List) exprNode()     {}

// Tuple represents a tuple
type Tuple struct {
	Elements []Expr
	Pos      Pos
}

func (t *Tuple) String() string {
	elems := []string{}
	for _, e := range t.Elements {
		elems = append(elems, e.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(elems, ", "))
}
func (t *Tuple) Position() Pos { return t.Pos }
func (t *Tuple) exprNode()     {}

// Record represents a record literal
type Record struct {
	Fields []*Field
	Pos    Pos
}

type Field struct {
	Name  string
	Value Expr
	Pos   Pos
}

func (r *Record) String() string {
	fields := []string{}
	for _, f := range r.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", f.Name, f.Value))
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}
func (r *Record) Position() Pos { return r.Pos }
func (r *Record) exprNode()     {}

// RecordAccess represents field access
type RecordAccess struct {
	Record Expr
	Field  string
	Pos    Pos
}

func (r *RecordAccess) String() string {
	return fmt.Sprintf("%s.%s", r.Record, r.Field)
}
func (r *RecordAccess) Position() Pos { return r.Pos }
func (r *RecordAccess) exprNode()     {}

// RecordUpdate represents functional record update: {base | field: value, ...}
type RecordUpdate struct {
	Base   Expr      // The base record expression
	Fields []*Field  // Fields to update
	Pos    Pos
}

func (r *RecordUpdate) String() string {
	fields := []string{}
	for _, f := range r.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", f.Name, f.Value))
	}
	return fmt.Sprintf("{ %s | %s }", r.Base, strings.Join(fields, ", "))
}
func (r *RecordUpdate) Position() Pos { return r.Pos }
func (r *RecordUpdate) exprNode()     {}

// Error represents a parse error node (placeholder for error recovery)
type Error struct {
	Pos Pos
	Msg string
}

func (e *Error) exprNode()       {}
func (e *Error) Literal() string { return "<error>" }
func (e *Error) Position() Pos   { return e.Pos }
func (e *Error) String() string {
	if e.Msg != "" {
		return fmt.Sprintf("<error: %s>", e.Msg)
	}
	return "<error>"
}

// QuasiQuote represents typed quasiquotes
type QuasiQuote struct {
	Kind           string // sql, html, regex, json, shell, url
	Template       string
	Interpolations []*Interpolation
	Pos            Pos
}

type Interpolation struct {
	Name string
	Expr Expr
	Type Type // Optional type annotation
	Pos  Pos
}

func (q *QuasiQuote) String() string {
	return fmt.Sprintf("%s\"\"\"%s\"\"\"", q.Kind, q.Template)
}
func (q *QuasiQuote) Position() Pos { return q.Pos }
func (q *QuasiQuote) exprNode()     {}

// Channel operations
type Send struct {
	Channel Expr
	Value   Expr
	Pos     Pos
}

func (s *Send) String() string {
	return fmt.Sprintf("%s <- %s", s.Channel, s.Value)
}
func (s *Send) Position() Pos { return s.Pos }
func (s *Send) exprNode()     {}

type Recv struct {
	Channel Expr
	Pos     Pos
}

func (r *Recv) String() string {
	return fmt.Sprintf("<- %s", r.Channel)
}
func (r *Recv) Position() Pos { return r.Pos }
func (r *Recv) exprNode()     {}

// Top-level declarations

// FuncDecl represents a function declaration
type FuncDecl struct {
	Name       string
	TypeParams []string // Generic type parameters
	Params     []*Param
	ReturnType Type
	Effects    []string
	Tests      []*TestCase
	Properties []*Property
	Body       Expr
	IsPure     bool
	IsExport   bool // Export flag
	Pos        Pos
	Span       Span   // For SID calculation
	SID        string // Stable ID (calculated post-parse)
	Origin     string // "func_decl" for metadata
}

type TestCase struct {
	Inputs   []Expr // Multiple inputs for multi-arg functions
	Expected Expr   // Expected output
	Pos      Pos
}

type Property struct {
	Name    string
	Binders []*Binder // forall bindings
	Expr    Expr
	Pos     Pos
}

type Binder struct {
	Name string
	Type Type
	Pos  Pos
}

func (f *FuncDecl) String() string {
	params := []string{}
	for _, p := range f.Params {
		params = append(params, p.Name)
	}
	pureStr := ""
	if f.IsPure {
		pureStr = "pure "
	}
	return fmt.Sprintf("%sfunc %s(%s) = %s", pureStr, f.Name, strings.Join(params, ", "), f.Body)
}
func (f *FuncDecl) Position() Pos { return f.Pos }
func (f *FuncDecl) stmtNode()     {}

// TypeDecl represents a type declaration
type TypeDecl struct {
	Name       string
	TypeParams []string
	Definition TypeDef
	Exported   bool // True if type was declared with 'export'
	Pos        Pos
}

type TypeDef interface {
	typeDefNode()
}

// AlgebraicType represents sum types
type AlgebraicType struct {
	Constructors []*Constructor
	Pos          Pos
}

type Constructor struct {
	Name   string
	Fields []Type
	Pos    Pos
}

func (a *AlgebraicType) typeDefNode() {}

// RecordType represents record types
type RecordType struct {
	Fields []*RecordField
	Pos    Pos
}

type RecordField struct {
	Name string
	Type Type
	Pos  Pos
}

func (r *RecordType) typeDefNode() {}
func (r *RecordType) typeNode()    {} // Also implements Type for nested record types
func (r *RecordType) String() string {
	fieldStrs := make([]string, len(r.Fields))
	for i, f := range r.Fields {
		fieldStrs[i] = fmt.Sprintf("%s: %s", f.Name, f.Type.String())
	}
	return fmt.Sprintf("{ %s }", strings.Join(fieldStrs, ", "))
}
func (r *RecordType) Position() Pos { return r.Pos }

// TypeAlias represents type aliases (not sum types)
// Used to distinguish `type Names = [string]` from `type Color = Red | Green`
type TypeAlias struct {
	Target Type // The aliased type expression
	Pos    Pos
}

func (t *TypeAlias) typeDefNode() {}

func (t *TypeDecl) String() string {
	return fmt.Sprintf("type %s", t.Name)
}
func (t *TypeDecl) Position() Pos { return t.Pos }
func (t *TypeDecl) stmtNode()     {}

// TypeClass represents a type class declaration
type TypeClass struct {
	Name       string
	TypeParam  string
	Superclass string // Optional superclass
	Methods    []*Method
	Pos        Pos
}

type Method struct {
	Name    string
	Type    Type
	Default Expr // Optional default implementation
	Pos     Pos
}

func (t *TypeClass) String() string {
	return fmt.Sprintf("class %s[%s]", t.Name, t.TypeParam)
}
func (t *TypeClass) Position() Pos { return t.Pos }
func (t *TypeClass) stmtNode()     {}

// Instance represents a type class instance
type Instance struct {
	ClassName string
	Type      Type
	Methods   map[string]Expr
	Pos       Pos
}

func (i *Instance) String() string {
	return fmt.Sprintf("instance %s[%s]", i.ClassName, i.Type)
}
func (i *Instance) Position() Pos { return i.Pos }
func (i *Instance) stmtNode()     {}

// Module represents a module
type Module struct {
	Name    string
	Imports []*Import
	Exports []string
	Decls   []Node
	Pos     Pos
}

type Import struct {
	Path         string
	Alias        string
	Symbols      []string // Specific imports
	Capabilities []string // Capability imports
	Pos          Pos
}

func (m *Module) String() string {
	return fmt.Sprintf("module %s", m.Name)
}
func (m *Module) Position() Pos { return m.Pos }

// Type system nodes

// SimpleType represents basic types
type SimpleType struct {
	Name string
	Pos  Pos
}

func (s *SimpleType) String() string { return s.Name }
func (s *SimpleType) Position() Pos  { return s.Pos }
func (s *SimpleType) typeNode()      {}

// TypeVar represents type variables
type TypeVar struct {
	Name string
	Pos  Pos
}

func (t *TypeVar) String() string { return t.Name }
func (t *TypeVar) Position() Pos  { return t.Pos }
func (t *TypeVar) typeNode()      {}

// FuncType represents function types
type FuncType struct {
	Params  []Type
	Return  Type
	Effects []string
	Pos     Pos
}

func (f *FuncType) String() string {
	params := []string{}
	for _, p := range f.Params {
		params = append(params, p.String())
	}
	effectStr := ""
	if len(f.Effects) > 0 {
		effectStr = fmt.Sprintf(" ! {%s}", strings.Join(f.Effects, ", "))
	}
	return fmt.Sprintf("(%s -> %s%s)", strings.Join(params, ", "), f.Return, effectStr)
}
func (f *FuncType) Position() Pos { return f.Pos }
func (f *FuncType) typeNode()     {}

// ListType represents list types
type ListType struct {
	Element Type
	Pos     Pos
}

func (l *ListType) String() string { return fmt.Sprintf("[%s]", l.Element) }
func (l *ListType) Position() Pos  { return l.Pos }
func (l *ListType) typeNode()      {}

// TupleType represents tuple types
type TupleType struct {
	Elements []Type
	Pos      Pos
}

func (t *TupleType) String() string {
	elems := []string{}
	for _, e := range t.Elements {
		elems = append(elems, e.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(elems, ", "))
}
func (t *TupleType) Position() Pos { return t.Pos }
func (t *TupleType) typeNode()     {}

// Pattern matching patterns

// WildcardPattern matches anything
type WildcardPattern struct {
	Pos Pos
}

func (w *WildcardPattern) String() string { return "_" }
func (w *WildcardPattern) Position() Pos  { return w.Pos }
func (w *WildcardPattern) patternNode()   {}

// ConsPattern matches list cons
type ConsPattern struct {
	Head Pattern
	Tail Pattern
	Pos  Pos
}

func (c *ConsPattern) String() string {
	return fmt.Sprintf("[%s, ...%s]", c.Head, c.Tail)
}
func (c *ConsPattern) Position() Pos { return c.Pos }
func (c *ConsPattern) patternNode()  {}

// ListPattern matches list literals
type ListPattern struct {
	Elements []Pattern
	Rest     Pattern // Optional rest pattern
	Pos      Pos
}

func (l *ListPattern) String() string {
	elems := []string{}
	for _, e := range l.Elements {
		elems = append(elems, e.String())
	}
	if l.Rest != nil {
		elems = append(elems, fmt.Sprintf("...%s", l.Rest))
	}
	return fmt.Sprintf("[%s]", strings.Join(elems, ", "))
}
func (l *ListPattern) Position() Pos { return l.Pos }
func (l *ListPattern) patternNode()  {}

// TuplePattern matches tuples
type TuplePattern struct {
	Elements []Pattern
	Pos      Pos
}

func (t *TuplePattern) String() string {
	elems := []string{}
	for _, e := range t.Elements {
		elems = append(elems, e.String())
	}
	return fmt.Sprintf("(%s)", strings.Join(elems, ", "))
}
func (t *TuplePattern) Position() Pos { return t.Pos }
func (t *TuplePattern) patternNode()  {}

// RecordPattern matches records
type RecordPattern struct {
	Fields []*FieldPattern
	Rest   bool // Has rest pattern ...
	Pos    Pos
}

type FieldPattern struct {
	Name    string
	Pattern Pattern
	Pos     Pos
}

func (r *RecordPattern) String() string {
	fields := []string{}
	for _, f := range r.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", f.Name, f.Pattern))
	}
	if r.Rest {
		fields = append(fields, "...")
	}
	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}
func (r *RecordPattern) Position() Pos { return r.Pos }
func (r *RecordPattern) patternNode()  {}

// ConstructorPattern matches algebraic type constructors
type ConstructorPattern struct {
	Name     string
	Patterns []Pattern
	Pos      Pos
}

func (c *ConstructorPattern) String() string {
	if len(c.Patterns) == 0 {
		return c.Name
	}
	patterns := []string{}
	for _, p := range c.Patterns {
		patterns = append(patterns, p.String())
	}
	return fmt.Sprintf("%s(%s)", c.Name, strings.Join(patterns, ", "))
}
func (c *ConstructorPattern) Position() Pos { return c.Pos }
func (c *ConstructorPattern) patternNode()  {}

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
