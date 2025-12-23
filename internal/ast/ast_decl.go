package ast

import (
	"fmt"
	"strings"
)

// Top-level declarations

// FuncDecl represents a function declaration.
//
// Usage example:
//
//	funcDecl := &ast.FuncDecl{
//	    Name:       "factorial",
//	    TypeParams: []string{},
//	    Params: []*ast.Param{
//	        {Name: "n", Type: intType, Pos: pos},
//	    },
//	    ReturnType: intType,
//	    Effects:    []EffectAnnotation{},
//	    Tests:      []*ast.TestCase{},
//	    Properties: []*ast.Property{},
//	    Body:       bodyExpr,
//	    IsPure:     true,
//	    IsExport:   false,
//	    Pos:        p.curToken.Pos,
//	}
//
// Common parser pattern:
//
//	// Parse: func factorial(n: int) -> int { ... }
//	p.nextToken() // skip "func"
//	name := p.curToken.Literal
//	p.nextToken() // move to LPAREN
//	params := p.parseFunctionParams()
//	returnType := p.parseReturnType()
//	effects := p.parseEffects()
//	body := p.parseExpression(LOWEST)
//	return &ast.FuncDecl{
//	    Name: name, Params: params,
//	    ReturnType: returnType, Effects: effects,
//	    Body: body, Pos: startPos,
//	}
type FuncDecl struct {
	Name       string
	TypeParams []string // Generic type parameters
	Params     []*Param
	ReturnType Type
	Effects    []EffectAnnotation // Effect annotations with optional budgets
	Tests      []*TestCase
	Properties []*Property
	Body       Expr // nil for extern functions
	IsPure     bool
	IsExport   bool // Export flag
	IsExtern   bool // Extern flag - function implemented in Go, no body
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

// ContractKind distinguishes between property tests and contract clauses.
// This enables reuse of the Property struct for both forall-style tests
// and requires/ensures contracts (M-VERIFY).
type ContractKind int

const (
	PropertyKind  ContractKind = iota // Existing forall property-based tests
	RequiresKind                      // Precondition contract
	EnsuresKind                       // Postcondition contract
	InvariantKind                     // Type/module invariant contract
)

// String returns the string representation of a ContractKind
func (k ContractKind) String() string {
	switch k {
	case PropertyKind:
		return "property"
	case RequiresKind:
		return "requires"
	case EnsuresKind:
		return "ensures"
	case InvariantKind:
		return "invariant"
	default:
		return "unknown"
	}
}

type Property struct {
	Name    string
	Kind    ContractKind // Type of property/contract (M-VERIFY)
	Binders []*Binder    // forall bindings (empty for requires/ensures)
	Expr    Expr         // Boolean predicate
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

// TestDecl represents a top-level test block: test "name" { ... }
type TestDecl struct {
	Name string
	Body []Expr // Test body (assertions, expressions)
	Pos  Pos
}

func (t *TestDecl) String() string { return fmt.Sprintf("test %q", t.Name) }
func (t *TestDecl) Position() Pos  { return t.Pos }
func (t *TestDecl) stmtNode()      {}

// PropertyDecl represents a top-level property block: property "name" { forall(...) => expr }
type PropertyDecl struct {
	Name     string
	Property *Property // The property specification
	Pos      Pos
}

func (p *PropertyDecl) String() string { return fmt.Sprintf("property %q", p.Name) }
func (p *PropertyDecl) Position() Pos  { return p.Pos }
func (p *PropertyDecl) stmtNode()      {}

// AssertStmt represents an assertion: assert expr
type AssertStmt struct {
	Condition Expr
	Message   string // Optional failure message
	Pos       Pos
}

func (a *AssertStmt) String() string { return fmt.Sprintf("assert %s", a.Condition) }
func (a *AssertStmt) Position() Pos  { return a.Pos }
func (a *AssertStmt) stmtNode()      {}
func (a *AssertStmt) exprNode()      {} // AssertStmt can be used as expression in test bodies

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

// ConstructorField represents a field in an ADT constructor.
// Supports both named fields (x: int) and positional fields (int).
type ConstructorField struct {
	Name string // Field name (empty for positional fields)
	Type Type   // Field type
	Pos  Pos
}

type Constructor struct {
	Name   string
	Fields []*ConstructorField
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
