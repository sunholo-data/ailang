// Package stmt defines the Statement IR — the ONLY representation that code
// generation emitters see. No emitter may import internal/core directly.
//
// Statement IR is target-agnostic: it knows about variable declarations,
// if/switch control flow, function calls, and field access — concepts that
// exist in every target language. It does NOT know about Go's interface{},
// Rust's Box<dyn Any>, or C's void*.
//
// The IR is produced by lowering passes in internal/gen/lower/ that transform
// Core AST + CoreTypeInfo into this representation.
package stmt

// Program represents a complete compilation unit ready for emission.
type Program struct {
	Package   string       // Target package/module name
	TypeDecls []TypeDecl   // ADT and record type declarations
	FuncDecls []FuncDecl   // Function declarations
	Imports   []ImportSpec // Required imports (e.g., "fmt", "strings")
}

// TypeDecl declares a named type.
type TypeDecl struct {
	Name     string
	Kind     TypeDeclKind
	Exported bool
}

// TypeDeclKind distinguishes ADTs from records.
type TypeDeclKind interface {
	typeDeclKind()
}

// ADTDecl declares an algebraic data type with variants.
type ADTDecl struct {
	Variants []ADTVariant
}

func (ADTDecl) typeDeclKind() {}

// ADTVariant is one constructor of an ADT.
type ADTVariant struct {
	Tag    string         // Constructor name (e.g., "Some", "None")
	Fields []ResolvedType // Field types (empty for nullary constructors)
}

// RecordDecl declares a record (struct) type.
type RecordDecl struct {
	Fields []RecordField
}

func (RecordDecl) typeDeclKind() {}

// RecordField is a named field in a record.
type RecordField struct {
	Name string
	Type ResolvedType
}

// TypeAliasDecl declares a type alias.
type TypeAliasDecl struct {
	Target ResolvedType
}

func (TypeAliasDecl) typeDeclKind() {}

// FuncDecl declares a function.
type FuncDecl struct {
	Name       string       // Function name (already namespaced by module)
	Params     []Param      // Parameters
	ReturnType ResolvedType // Return type
	Body       []Stmt       // Function body statements
	Return     Expr         // Final return expression
	Exported   bool         // Whether this function is public
	Module     string       // Source module name
	File       string       // Source file path (for diagnostics; optional)
	Line       int          // Source line of the func declaration (for diagnostics; 0 = unknown)
}

// Param is a function parameter.
type Param struct {
	Name string
	Type ResolvedType
}

// ImportSpec represents a required import.
type ImportSpec struct {
	Path  string // e.g., "strings", "fmt"
	Alias string // optional alias
}

// --- Statements ---

// Stmt is a statement in a function body.
type Stmt interface {
	stmt()
}

// VarDecl declares a variable with an initial value.
type VarDecl struct {
	Name  string
	Type  ResolvedType // may be nil if type is inferred
	Value Expr
	Line  int // source line (0 = unknown)
}

func (VarDecl) stmt() {}

// IfStmt is a conditional branch.
type IfStmt struct {
	Cond Expr
	Then []Stmt
	Else []Stmt // may be empty or contain another IfStmt for else-if chains
	Line int    // source line (0 = unknown)
}

func (IfStmt) stmt() {}

// SwitchStmt dispatches on a value (used for ADT tag matching).
type SwitchStmt struct {
	Scrutinee Expr
	ADTName   string // The ADT type name (e.g., "Color") — helps emitters generate typed constants
	Cases     []SwitchCase
	Default   []Stmt // may be empty; if non-empty, must end in panic or return
	Line      int    // source line (0 = unknown)
}

func (SwitchStmt) stmt() {}

// SwitchCase is one arm of a switch.
type SwitchCase struct {
	Tag      string    // ADT variant tag to match
	Bindings []Binding // Variables bound from the matched variant's fields
	Body     []Stmt
}

// Binding captures a field extraction from an ADT match.
type Binding struct {
	Name       string       // Variable name to bind
	FieldIndex int          // Which field of the variant (0-based)
	Type       ResolvedType // Type of the extracted field
}

// AssignStmt assigns a value to an existing variable.
type AssignStmt struct {
	Name  string
	Value Expr
	Line  int // source line (0 = unknown)
}

func (AssignStmt) stmt() {}

// ReturnStmt returns a value from a function.
type ReturnStmt struct {
	Value Expr
	Line  int // source line (0 = unknown)
}

func (ReturnStmt) stmt() {}

// ExprStmt wraps an expression used as a statement (e.g., function call for side effects).
type ExprStmt struct {
	Value Expr
	Line  int // source line (0 = unknown)
}

func (ExprStmt) stmt() {}

// --- Expressions ---

// Expr is an expression that produces a value.
type Expr interface {
	expr()
}

// LitInt is an integer literal.
type LitInt struct{ Value int64 }

func (LitInt) expr() {}

// LitFloat is a float literal.
type LitFloat struct{ Value float64 }

func (LitFloat) expr() {}

// LitBool is a boolean literal.
type LitBool struct{ Value bool }

func (LitBool) expr() {}

// LitString is a string literal.
type LitString struct{ Value string }

func (LitString) expr() {}

// LitUnit is the unit value.
type LitUnit struct{}

func (LitUnit) expr() {}

// VarRef references a local variable.
type VarRef struct{ Name string }

func (VarRef) expr() {}

// GlobalRef references a module-qualified function or value.
type GlobalRef struct {
	Module string // Source module
	Name   string // Function/value name
}

func (GlobalRef) expr() {}

// BinOp is a binary operation.
type BinOp struct {
	Op    BinOpKind
	Left  Expr
	Right Expr
}

func (BinOp) expr() {}

// BinOpKind enumerates binary operators.
type BinOpKind int

const (
	OpAdd BinOpKind = iota
	OpSub
	OpMul
	OpDiv
	OpMod
	OpEq
	OpNeq
	OpLt
	OpLte
	OpGt
	OpGte
	OpAnd
	OpOr
	OpConcat // string concatenation
)

// UnOp is a unary operation.
type UnOp struct {
	Op      UnOpKind
	Operand Expr
}

func (UnOp) expr() {}

// UnOpKind enumerates unary operators.
type UnOpKind int

const (
	OpNeg UnOpKind = iota
	OpNot
)

// Call invokes a function.
type Call struct {
	Func Expr   // The function to call (VarRef, GlobalRef, or Lambda)
	Args []Expr // Arguments
}

func (Call) expr() {}

// FieldAccess accesses a field of a record.
type FieldAccess struct {
	Record Expr
	Field  string
}

func (FieldAccess) expr() {}

// RecordLit constructs a record value.
type RecordLit struct {
	TypeName string // The record type name
	Fields   []FieldInit
}

func (RecordLit) expr() {}

// FieldInit initializes one field of a record literal.
type FieldInit struct {
	Name  string
	Value Expr
}

// RecordUpdate creates a new record with some fields changed.
type RecordUpdate struct {
	Base   Expr
	Fields []FieldInit
}

func (RecordUpdate) expr() {}

// ListLit constructs a list value.
type ListLit struct {
	ElemType ResolvedType // Element type
	Elems    []Expr
}

func (ListLit) expr() {}

// Cons prepends an element to a list.
type Cons struct {
	Head Expr
	Tail Expr
}

func (Cons) expr() {}

// TupleLit constructs a tuple value.
type TupleLit struct {
	Elems []Expr
}

func (TupleLit) expr() {}

// ArrayLit constructs an array value.
type ArrayLit struct {
	ElemType ResolvedType
	Elems    []Expr
}

func (ArrayLit) expr() {}

// ADTConstructor constructs an ADT variant.
type ADTConstructor struct {
	TypeName string // The ADT type name (e.g., "Option")
	Tag      string // The variant tag (e.g., "Some")
	Args     []Expr // Constructor arguments
}

func (ADTConstructor) expr() {}

// Lambda is an anonymous function.
type Lambda struct {
	Params []Param
	Body   []Stmt
	Return Expr
}

func (Lambda) expr() {}

// TypeAssert asserts a value to a specific type (used at interface{} boundaries).
type TypeAssert struct {
	Value Expr
	Type  ResolvedType
}

func (TypeAssert) expr() {}

// IfExpr is a conditional expression (if/then/else that returns a value).
type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
}

func (IfExpr) expr() {}

// BuiltinCall calls a builtin/stdlib function by name.
type BuiltinCall struct {
	Name string // Builtin name (e.g., "_str_trim", "_list_map")
	Args []Expr
}

func (BuiltinCall) expr() {}
