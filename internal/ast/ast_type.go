package ast

import (
	"fmt"
	"strings"
)

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
	Effects []EffectAnnotation // Effect annotations with optional budgets
	Pos     Pos
}

func (f *FuncType) String() string {
	params := []string{}
	for _, p := range f.Params {
		params = append(params, p.String())
	}
	effectStr := FormatEffects(f.Effects)
	if effectStr != "" {
		effectStr = " " + effectStr
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

// ArrayType represents array types Array[T]
type ArrayType struct {
	Element Type
	Pos     Pos
}

func (a *ArrayType) String() string { return fmt.Sprintf("Array[%s]", a.Element) }
func (a *ArrayType) Position() Pos  { return a.Pos }
func (a *ArrayType) typeNode()      {}

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

// TypeApp represents type application (generic types)
// Example: Option[int], Result[T, E], Map[string, int]
// M-TAPP-FIX: Added to preserve type arguments in annotations
type TypeApp struct {
	Constructor string // Type constructor name (e.g., "Option", "Result")
	Args        []Type // Type arguments (e.g., [int], [T, E])
	Pos         Pos
}

func (t *TypeApp) String() string {
	args := make([]string, len(t.Args))
	for i, arg := range t.Args {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s[%s]", t.Constructor, strings.Join(args, ", "))
}
func (t *TypeApp) Position() Pos { return t.Pos }
func (t *TypeApp) typeNode()     {}

// --- IFC label syntax (M-TAINT-TYPES) ---

// LabelExpr is the label constant in T<label> syntax.
type LabelExpr struct {
	Name string
	Pos  Pos
}

// RefinementExpr is the {not LABEL} refinement constraint on a type.
// MVP: only `not IDENT` is supported.
type RefinementExpr struct {
	NotLabel string // the label name that must NOT be present
	Pos      Pos
}

// LabelledType wraps any type with an optional label and/or refinement.
// Exactly one of Label or Refinement is non-nil (or both could be nil if only
// the base is retained after desugaring, but the parser always sets at least one).
//
//	string<email>   → LabelledType{Base: SimpleType{"string"}, Label: &LabelExpr{"email"}}
//	string{not pii} → LabelledType{Base: SimpleType{"string"}, Refinement: &RefinementExpr{"pii"}}
type LabelledType struct {
	Base       Type
	Label      *LabelExpr      // set for T<label> syntax
	Refinement *RefinementExpr // set for T{not IDENT} syntax
	Pos        Pos
}

func (t *LabelledType) String() string {
	base := t.Base.String()
	if t.Label != nil {
		return fmt.Sprintf("%s<%s>", base, t.Label.Name)
	}
	if t.Refinement != nil {
		return fmt.Sprintf("%s{not %s}", base, t.Refinement.NotLabel)
	}
	return base
}
func (t *LabelledType) Position() Pos { return t.Pos }
func (t *LabelledType) typeNode()     {}
