package ast

import (
	"fmt"
	"strings"
)

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
