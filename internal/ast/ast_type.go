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
