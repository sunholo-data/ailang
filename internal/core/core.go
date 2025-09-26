package core

import (
	"fmt"
	"strings"
	"github.com/sunholo/ailang/internal/ast"
)

// Core AST - A-Normal Form with explicit recursion
// All complex expressions are decomposed into let-bindings

// CoreNode is the base for all Core AST nodes
type CoreNode struct {
	NodeID   uint64   // Stable identifier assigned by elaborator
	CoreSpan ast.Pos  // Position in Core AST
	OrigSpan ast.Pos  // Original surface position for diagnostics
}

// CoreExpr is the base interface for Core expressions
type CoreExpr interface {
	ID() uint64
	Span() ast.Pos       // Core span
	OriginalSpan() ast.Pos  // Surface origin
	String() string
	coreExpr()
}

// Ensure CoreNode implements base methods
func (n CoreNode) ID() uint64 { return n.NodeID }
func (n CoreNode) Span() ast.Pos { return n.CoreSpan }
func (n CoreNode) OriginalSpan() ast.Pos { return n.OrigSpan }

// Atomic expressions (can appear in any position)

// Var represents a variable reference
type Var struct {
	CoreNode
	Name string
}

func (v *Var) coreExpr() {}
func (v *Var) String() string { return v.Name }

// Lit represents a literal value
type Lit struct {
	CoreNode
	Kind  LitKind
	Value interface{}
}

type LitKind int

const (
	IntLit LitKind = iota
	FloatLit
	StringLit
	BoolLit
	UnitLit
)

func (l *Lit) coreExpr() {}
func (l *Lit) String() string { return fmt.Sprintf("%v", l.Value) }

// Lambda represents a function value
type Lambda struct {
	CoreNode
	Params []string
	Body   CoreExpr
}

func (l *Lambda) coreExpr() {}
func (l *Lambda) String() string {
	return fmt.Sprintf("λ%v. %s", l.Params, l.Body)
}

// Complex expressions (must be let-bound in ANF)

// Let represents a non-recursive let binding
type Let struct {
	CoreNode
	Name  string
	Value CoreExpr  // In ANF: atomic or simple call
	Body  CoreExpr
}

func (l *Let) coreExpr() {}
func (l *Let) String() string {
	return fmt.Sprintf("let %s = %s in %s", l.Name, l.Value, l.Body)
}

// LetRec represents mutually recursive bindings
type LetRec struct {
	CoreNode
	Bindings []RecBinding
	Body     CoreExpr
}

type RecBinding struct {
	Name  string
	Value CoreExpr  // Usually Lambda for recursion
}

func (l *LetRec) coreExpr() {}
func (l *LetRec) String() string {
	return fmt.Sprintf("let rec %v in %s", l.Bindings, l.Body)
}

// App represents function application (in ANF, args are atomic)
type App struct {
	CoreNode
	Func CoreExpr
	Args []CoreExpr  // All must be atomic in ANF
}

func (a *App) coreExpr() {}
func (a *App) String() string {
	return fmt.Sprintf("%s(%v)", a.Func, a.Args)
}

// If represents conditional (in ANF, condition is atomic)
type If struct {
	CoreNode
	Cond CoreExpr  // Must be atomic in ANF
	Then CoreExpr
	Else CoreExpr
}

func (i *If) coreExpr() {}
func (i *If) String() string {
	return fmt.Sprintf("if %s then %s else %s", i.Cond, i.Then, i.Else)
}

// Match represents pattern matching
type Match struct {
	CoreNode
	Scrutinee  CoreExpr  // Must be atomic in ANF
	Arms       []MatchArm
	Exhaustive bool  // Set by elaborator/typechecker
}

type MatchArm struct {
	Pattern CorePattern
	Guard   CoreExpr  // Optional, must be atomic
	Body    CoreExpr
}

func (m *Match) coreExpr() {}
func (m *Match) String() string {
	return fmt.Sprintf("match %s { %v }", m.Scrutinee, m.Arms)
}

// BinOp represents binary operations (in ANF, operands are atomic)
type BinOp struct {
	CoreNode
	Op    string
	Left  CoreExpr  // Must be atomic in ANF
	Right CoreExpr  // Must be atomic in ANF
}

func (b *BinOp) coreExpr() {}
func (b *BinOp) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, b.Op, b.Right)
}

// UnOp represents unary operations (in ANF, operand is atomic)
type UnOp struct {
	CoreNode
	Op      string
	Operand CoreExpr  // Must be atomic in ANF
}

func (u *UnOp) coreExpr() {}
func (u *UnOp) String() string {
	return fmt.Sprintf("%s%s", u.Op, u.Operand)
}

// Record represents record construction (fields are atomic in ANF)
type Record struct {
	CoreNode
	Fields map[string]CoreExpr  // All values must be atomic
}

func (r *Record) coreExpr() {}
func (r *Record) String() string {
	return fmt.Sprintf("{%v}", r.Fields)
}

// RecordAccess represents field access (record is atomic in ANF)
type RecordAccess struct {
	CoreNode
	Record CoreExpr  // Must be atomic in ANF
	Field  string
}

func (r *RecordAccess) coreExpr() {}
func (r *RecordAccess) String() string {
	return fmt.Sprintf("%s.%s", r.Record, r.Field)
}

// List represents list construction (elements are atomic in ANF)
type List struct {
	CoreNode
	Elements []CoreExpr  // All must be atomic in ANF
}

func (l *List) coreExpr() {}
func (l *List) String() string {
	return fmt.Sprintf("[%v]", l.Elements)
}

// Patterns for matching

type CorePattern interface {
	patternNode()
	String() string
}

type VarPattern struct {
	Name string
}

func (v *VarPattern) patternNode() {}
func (v *VarPattern) String() string { return v.Name }

type LitPattern struct {
	Value interface{}
}

func (l *LitPattern) patternNode() {}
func (l *LitPattern) String() string { return fmt.Sprintf("%v", l.Value) }

type ConstructorPattern struct {
	Name string
	Args []CorePattern
}

func (c *ConstructorPattern) patternNode() {}
func (c *ConstructorPattern) String() string {
	return fmt.Sprintf("%s(%v)", c.Name, c.Args)
}

type ListPattern struct {
	Elements []CorePattern
	Tail     *CorePattern  // For ... patterns
}

func (l *ListPattern) patternNode() {}
func (l *ListPattern) String() string {
	return fmt.Sprintf("[%v]", l.Elements)
}

type RecordPattern struct {
	Fields map[string]CorePattern
}

func (r *RecordPattern) patternNode() {}
func (r *RecordPattern) String() string {
	return fmt.Sprintf("{%v}", r.Fields)
}

type WildcardPattern struct{}

func (w *WildcardPattern) patternNode() {}
func (w *WildcardPattern) String() string { return "_" }

// Program represents a Core program
type Program struct {
	Decls []CoreExpr  // Top-level declarations
}

// Dictionary-passing nodes for type class resolution

// DictAbs represents dictionary abstraction at binders
// Used for polymorphic functions with type class constraints
type DictAbs struct {
	CoreNode
	Params []DictParam // Dictionary parameters in canonical order
	Body   CoreExpr    // Body with dictionaries available
}

func (d *DictAbs) coreExpr() {}
func (d *DictAbs) String() string {
	params := ""
	for i, p := range d.Params {
		if i > 0 {
			params += ", "
		}
		params += fmt.Sprintf("%s: %s[%s]", p.Name, p.ClassName, p.Type)
	}
	return fmt.Sprintf("DictAbs([%s], %s)", params, d.Body)
}

// DictApp represents dictionary application at use sites
// All method calls through type classes become DictApp nodes
type DictApp struct {
	CoreNode
	Dict   CoreExpr // Dictionary reference (must be a Var in ANF)
	Method string   // Method name: "add", "eq", "lt", etc.
	Args   []CoreExpr // Method arguments
}

func (d *DictApp) coreExpr() {}
func (d *DictApp) String() string {
	args := ""
	for i, a := range d.Args {
		if i > 0 {
			args += ", "
		}
		args += a.String()
	}
	return fmt.Sprintf("DictApp(%s.%s, [%s])", d.Dict, d.Method, args)
}

// DictRef represents a reference to a built-in dictionary
type DictRef struct {
	CoreNode
	ClassName string // e.g., "Num", "Ord"
	TypeName  string // Normalized type: "Int", "Float", etc.
}

func (d *DictRef) coreExpr() {}
func (d *DictRef) String() string {
	return fmt.Sprintf("dict_%s_%s", d.ClassName, d.TypeName)
}

// DictParam represents a dictionary parameter in DictAbs
type DictParam struct {
	Name      string // e.g., "dict_Num_α"
	ClassName string // e.g., "Num"
	Type      string // String representation of type
}

// Helper to check if expression is atomic (for ANF verification)
func IsAtomic(expr CoreExpr) bool {
	switch expr.(type) {
	case *Var, *Lit, *Lambda, *DictRef:
		return true
	default:
		return false
	}
}

// Pretty provides a basic string representation of Core programs
// This is a stub implementation for testing purposes
func Pretty(prog *Program) string {
	var parts []string
	for i, decl := range prog.Decls {
		parts = append(parts, fmt.Sprintf("decl_%d: %s", i, decl.String()))
	}
	return fmt.Sprintf("Program(\n  %s\n)", fmt.Sprintf(strings.Join(parts, "\n  ")))
}