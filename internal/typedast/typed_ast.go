package typedast

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
)

// TypedExpr is the base for all typed expressions
// It carries monomorphic type and effect information
type TypedExpr struct {
	NodeID    uint64
	Span      ast.Pos
	Type      interface{}   // types.Type - Always monomorphic
	EffectRow interface{}   // *types.Row - Effect row with kind
	Core      core.CoreExpr // Underlying core expression
}

// Typed node types - mirror Core AST but with type annotations

// TypedVar represents a typed variable reference
type TypedVar struct {
	TypedExpr
	Name string
}

// TypedLit represents a typed literal
type TypedLit struct {
	TypedExpr
	Kind  core.LitKind
	Value interface{}
}

// TypedLambda represents a typed lambda
type TypedLambda struct {
	TypedExpr
	Params     []string
	ParamTypes []interface{} // []types.Type
	Body       TypedNode
}

// TypedLet represents a typed let binding
// Only let bindings carry Schemes (generalized types)
type TypedLet struct {
	TypedExpr
	Name   string
	Scheme interface{} // *types.Scheme - Generalized type (only here!)
	Value  TypedNode
	Body   TypedNode
}

// TypedLetRec represents typed recursive bindings
type TypedLetRec struct {
	TypedExpr
	Bindings []TypedRecBinding
	Body     TypedNode
}

// TypedRecBinding represents a recursive binding with scheme
type TypedRecBinding struct {
	Name   string
	Scheme interface{} // *types.Scheme - Generalized type for recursive binding
	Value  TypedNode
}

// TypedApp represents typed function application
type TypedApp struct {
	TypedExpr
	Func TypedNode
	Args []TypedNode
}

// TypedIf represents typed conditional
type TypedIf struct {
	TypedExpr
	Cond TypedNode
	Then TypedNode
	Else TypedNode
}

// TypedMatch represents typed pattern matching
type TypedMatch struct {
	TypedExpr
	Scrutinee  TypedNode
	Arms       []TypedMatchArm
	Exhaustive bool
}

// TypedMatchArm represents a typed match arm
type TypedMatchArm struct {
	Pattern TypedPattern
	Guard   TypedNode // Optional
	Body    TypedNode
}

// TypedBinOp represents typed binary operation
type TypedBinOp struct {
	TypedExpr
	Op    string
	Left  TypedNode
	Right TypedNode
}

// TypedUnOp represents typed unary operation
type TypedUnOp struct {
	TypedExpr
	Op      string
	Operand TypedNode
}

// TypedRecord represents typed record construction
type TypedRecord struct {
	TypedExpr
	Fields map[string]TypedNode
}

// TypedRecordAccess represents typed field access
type TypedRecordAccess struct {
	TypedExpr
	Record TypedNode
	Field  string
}

// TypedList represents typed list construction
type TypedList struct {
	TypedExpr
	Elements []TypedNode
}

// TypedTuple represents typed tuple construction
type TypedTuple struct {
	TypedExpr
	Elements []TypedNode
}

func (t TypedTuple) String() string {
	return fmt.Sprintf("(...) : %s", t.Type)
}

// TypedNode is the interface for all typed nodes
type TypedNode interface {
	GetNodeID() uint64
	GetSpan() ast.Pos
	GetType() interface{}      // types.Type
	GetEffectRow() interface{} // *types.Row
	GetCore() core.CoreExpr
	String() string
}

// Implement TypedNode interface for TypedExpr
func (t TypedExpr) GetNodeID() uint64         { return t.NodeID }
func (t TypedExpr) GetSpan() ast.Pos          { return t.Span }
func (t TypedExpr) GetType() interface{}      { return t.Type }
func (t TypedExpr) GetEffectRow() interface{} { return t.EffectRow }
func (t TypedExpr) GetCore() core.CoreExpr    { return t.Core }

// String methods for typed nodes
func (t TypedVar) String() string { return t.Name }
func (t TypedLit) String() string { return fmt.Sprintf("%v", t.Value) }
func (t TypedLambda) String() string {
	return fmt.Sprintf("λ%v. %s : %s", t.Params, t.Body, t.Type)
}

func (t TypedLet) String() string {
	typeStr := FormatType(t.Scheme)
	return fmt.Sprintf("let %s : %s = %s in %s", t.Name, typeStr, t.Value, t.Body)
}

func (t TypedLetRec) String() string {
	var binds []string
	for _, b := range t.Bindings {
		binds = append(binds, fmt.Sprintf("%s : %s", b.Name, FormatType(b.Scheme)))
	}
	return fmt.Sprintf("let rec %s in %s", binds, t.Body)
}

func (t TypedApp) String() string {
	return fmt.Sprintf("%s(%v) : %s", t.Func, t.Args, t.Type)
}

func (t TypedIf) String() string {
	return fmt.Sprintf("if %s then %s else %s : %s", t.Cond, t.Then, t.Else, t.Type)
}

func (t TypedMatch) String() string {
	return fmt.Sprintf("match %s { ... } : %s", t.Scrutinee, t.Type)
}

func (t TypedBinOp) String() string {
	return fmt.Sprintf("(%s %s %s) : %s", t.Left, t.Op, t.Right, t.Type)
}

func (t TypedUnOp) String() string {
	return fmt.Sprintf("%s%s : %s", t.Op, t.Operand, t.Type)
}

func (t TypedRecord) String() string {
	return fmt.Sprintf("{...} : %s", t.Type)
}

func (t TypedRecordAccess) String() string {
	return fmt.Sprintf("%s.%s : %s", t.Record, t.Field, t.Type)
}

func (t TypedList) String() string {
	return fmt.Sprintf("[...] : %s", t.Type)
}

// Typed patterns

type TypedPattern interface {
	patternNode()
	String() string
}

type TypedVarPattern struct {
	Name string
	Type interface{} // types.Type
}

func (p TypedVarPattern) patternNode()   {}
func (p TypedVarPattern) String() string { return p.Name }

type TypedLitPattern struct {
	Value interface{}
}

func (p TypedLitPattern) patternNode()   {}
func (p TypedLitPattern) String() string { return fmt.Sprintf("%v", p.Value) }

type TypedConstructorPattern struct {
	Name string
	Args []TypedPattern
}

func (p TypedConstructorPattern) patternNode() {}
func (p TypedConstructorPattern) String() string {
	return fmt.Sprintf("%s(%v)", p.Name, p.Args)
}

type TypedWildcardPattern struct{}

func (p TypedWildcardPattern) patternNode()   {}
func (p TypedWildcardPattern) String() string { return "_" }

type TypedTuplePattern struct {
	Elements []TypedPattern
}

func (p TypedTuplePattern) patternNode() {}
func (p TypedTuplePattern) String() string {
	parts := make([]string, len(p.Elements))
	for i, elem := range p.Elements {
		parts[i] = elem.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(parts, ", "))
}

type TypedListPattern struct {
	Elements []TypedPattern
	Tail     *TypedPattern // For spread patterns: [x, ...rest]
}

func (p TypedListPattern) patternNode() {}
func (p TypedListPattern) String() string {
	parts := make([]string, len(p.Elements))
	for i, elem := range p.Elements {
		parts[i] = elem.String()
	}
	if p.Tail != nil {
		return fmt.Sprintf("[%s, ...%s]", strings.Join(parts, ", "), (*p.Tail).String())
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// TypedProgram represents a typed program
type TypedProgram struct {
	Decls []TypedNode
}

// FormatType formats a type for display (interface{} version)
func FormatType(t interface{}) string {
	if t == nil {
		return "<unknown>"
	}
	// Use the String() method if available
	if stringer, ok := t.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%v", t)
}

// PrintTypedProgram pretty-prints a typed program
func PrintTypedProgram(prog *TypedProgram) string {
	var result string
	for _, decl := range prog.Decls {
		result += decl.String() + "\n"
	}
	return result
}
