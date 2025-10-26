package ast

import (
	"fmt"
	"strings"
)

// Expression nodes

// Identifier represents a variable or function name.
//
// Usage example:
//
//	ident := &ast.Identifier{
//	    Name: "factorial",
//	    Pos:  p.curToken.Pos,
//	}
//
// Common pattern in parser:
//
//	if p.curTokenIs(lexer.IDENT) {
//	    return &ast.Identifier{Name: p.curToken.Literal, Pos: p.curToken.Pos}
//	}
//
// Type assertions:
//
//	ident, ok := expr.(*ast.Identifier)
//	if ok {
//	    fmt.Println("Variable name:", ident.Name)
//	}
type Identifier struct {
	Name string
	Pos  Pos
}

func (i *Identifier) String() string { return i.Name }
func (i *Identifier) Position() Pos  { return i.Pos }
func (i *Identifier) exprNode()      {}
func (i *Identifier) patternNode()   {}

// Literal represents a literal value.
//
// ⚠️ CRITICAL GOTCHA: Lexer returns int64 for integers, NOT int!
//
// Usage example:
//
//	lit := &ast.Literal{
//	    Kind:  ast.IntLit,
//	    Value: int64(42),  // ⚠️ Must be int64, not int!
//	    Pos:   p.curToken.Pos,
//	}
//
// Type assertions (CORRECT):
//
//	lit := expr.(*ast.Literal)
//	if lit.Kind == ast.IntLit {
//	    val := lit.Value.(int64)  // ✅ CORRECT - int64
//	    fmt.Println("Integer:", val)
//	}
//
// Type assertions (WRONG):
//
//	val := lit.Value.(int)  // ❌ WRONG - will panic! Lexer returns int64
//
// Common parser pattern:
//
//	case lexer.INT:
//	    val, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
//	    return &ast.Literal{Kind: ast.IntLit, Value: val, Pos: p.curToken.Pos}
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

// Lambda represents a lambda expression.
//
// Usage example:
//
//	lambda := &ast.Lambda{
//	    Params: []*ast.Param{
//	        {Name: "x", Type: nil, Pos: pos1},
//	        {Name: "y", Type: nil, Pos: pos2},
//	    },
//	    Body:    bodyExpr,
//	    Effects: []string{"IO"},
//	    Pos:     p.curToken.Pos,
//	}
//
// Common parser pattern:
//
//	// Parse: \x y. x + y
//	p.nextToken() // skip \
//	params := p.parseParams()
//	p.expectToken(lexer.DOT)
//	body := p.parseExpression(LOWEST)
//	return &ast.Lambda{Params: params, Body: body, Pos: startPos}
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

// FuncCall represents a function application.
//
// Usage example:
//
//	call := &ast.FuncCall{
//	    Func: &ast.Identifier{Name: "factorial", Pos: pos},
//	    Args: []ast.Expr{
//	        &ast.Literal{Kind: ast.IntLit, Value: int64(5), Pos: pos},
//	    },
//	    Pos: p.curToken.Pos,
//	}
//
// Common parser pattern:
//
//	// Parse: factorial(5)
//	func := p.parseExpression(CALL) // parses "factorial"
//	p.nextToken() // move to LPAREN
//	args := p.parseCallArguments() // parses "(5)"
//	return &ast.FuncCall{Func: func, Args: args, Pos: startPos}
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

// List represents a list literal.
//
// Usage example:
//
//	list := &ast.List{
//	    Elements: []ast.Expr{
//	        &ast.Literal{Kind: ast.IntLit, Value: int64(1), Pos: pos},
//	        &ast.Literal{Kind: ast.IntLit, Value: int64(2), Pos: pos},
//	        &ast.Literal{Kind: ast.IntLit, Value: int64(3), Pos: pos},
//	    },
//	    Pos: p.curToken.Pos,
//	}
//
// Common parser pattern:
//
//	// Parse: [1, 2, 3]
//	p.nextToken() // skip [
//	elements := []ast.Expr{}
//	for !p.curTokenIs(lexer.RBRACKET) {
//	    elements = append(elements, p.parseExpression(LOWEST))
//	    if p.peekTokenIs(lexer.COMMA) {
//	        p.nextToken() // move to comma
//	        p.nextToken() // move past comma
//	    }
//	}
//	p.nextToken() // skip ]
//	return &ast.List{Elements: elements, Pos: startPos}
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
	Base   Expr     // The base record expression
	Fields []*Field // Fields to update
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
