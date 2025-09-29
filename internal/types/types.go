package types

import (
	"fmt"
	"strings"
)

// Type represents a type in the AILANG type system
type Type interface {
	String() string
	Equals(Type) bool
	Substitute(map[string]Type) Type
}

// Basic types

// TVar represents a type variable
type TVar struct {
	Name string
}

func (t *TVar) String() string {
	return t.Name
}

func (t *TVar) Equals(other Type) bool {
	if o, ok := other.(*TVar); ok {
		return t.Name == o.Name
	}
	return false
}

func (t *TVar) Substitute(subs map[string]Type) Type {
	if sub, ok := subs[t.Name]; ok {
		return sub
	}
	return t
}

// TCon represents a type constructor (int, bool, string, etc.)
type TCon struct {
	Name string
}

func (t *TCon) String() string {
	return t.Name
}

func (t *TCon) Equals(other Type) bool {
	if o, ok := other.(*TCon); ok {
		return t.Name == o.Name
	}
	return false
}

func (t *TCon) Substitute(subs map[string]Type) Type {
	return t
}

// TFunc represents a function type
type TFunc struct {
	Params  []Type
	Return  Type
	Effects []EffectType
}

func (t *TFunc) String() string {
	params := make([]string, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.String()
	}

	effectStr := ""
	if len(t.Effects) > 0 {
		effects := make([]string, len(t.Effects))
		for i, e := range t.Effects {
			effects[i] = e.String()
		}
		effectStr = fmt.Sprintf(" ! {%s}", strings.Join(effects, ", "))
	}

	if len(params) == 1 {
		return fmt.Sprintf("%s -> %s%s", params[0], t.Return.String(), effectStr)
	}
	return fmt.Sprintf("(%s) -> %s%s", strings.Join(params, ", "), t.Return.String(), effectStr)
}

func (t *TFunc) Equals(other Type) bool {
	if o, ok := other.(*TFunc); ok {
		if len(t.Params) != len(o.Params) {
			return false
		}
		for i := range t.Params {
			if !t.Params[i].Equals(o.Params[i]) {
				return false
			}
		}
		if !t.Return.Equals(o.Return) {
			return false
		}
		// Check effects
		if len(t.Effects) != len(o.Effects) {
			return false
		}
		for i := range t.Effects {
			if !t.Effects[i].Equals(o.Effects[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TFunc) Substitute(subs map[string]Type) Type {
	params := make([]Type, len(t.Params))
	for i, p := range t.Params {
		params[i] = p.Substitute(subs)
	}
	return &TFunc{
		Params:  params,
		Return:  t.Return.Substitute(subs),
		Effects: t.Effects, // Effects don't get substituted for now
	}
}

// TList represents a list type
type TList struct {
	Element Type
}

func (t *TList) String() string {
	return fmt.Sprintf("[%s]", t.Element.String())
}

func (t *TList) Equals(other Type) bool {
	if o, ok := other.(*TList); ok {
		return t.Element.Equals(o.Element)
	}
	return false
}

func (t *TList) Substitute(subs map[string]Type) Type {
	return &TList{Element: t.Element.Substitute(subs)}
}

// TTuple represents a tuple type
type TTuple struct {
	Elements []Type
}

func (t *TTuple) String() string {
	elems := make([]string, len(t.Elements))
	for i, e := range t.Elements {
		elems[i] = e.String()
	}
	return fmt.Sprintf("(%s)", strings.Join(elems, ", "))
}

func (t *TTuple) Equals(other Type) bool {
	if o, ok := other.(*TTuple); ok {
		if len(t.Elements) != len(o.Elements) {
			return false
		}
		for i := range t.Elements {
			if !t.Elements[i].Equals(o.Elements[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TTuple) Substitute(subs map[string]Type) Type {
	elems := make([]Type, len(t.Elements))
	for i, e := range t.Elements {
		elems[i] = e.Substitute(subs)
	}
	return &TTuple{Elements: elems}
}

// TRecord represents a record type with row polymorphism
type TRecord struct {
	Fields map[string]Type
	Row    Type // Row variable for extensibility
}

func (t *TRecord) String() string {
	var fields []string
	for name, typ := range t.Fields {
		fields = append(fields, fmt.Sprintf("%s: %s", name, typ.String()))
	}

	if t.Row != nil {
		fields = append(fields, fmt.Sprintf("...%s", t.Row.String()))
	}

	return fmt.Sprintf("{ %s }", strings.Join(fields, ", "))
}

func (t *TRecord) Equals(other Type) bool {
	if o, ok := other.(*TRecord); ok {
		if len(t.Fields) != len(o.Fields) {
			return false
		}
		for name, typ := range t.Fields {
			if oTyp, ok := o.Fields[name]; !ok || !typ.Equals(oTyp) {
				return false
			}
		}
		// Check row
		if t.Row == nil && o.Row == nil {
			return true
		}
		if t.Row != nil && o.Row != nil {
			return t.Row.Equals(o.Row)
		}
		return false
	}
	return false
}

func (t *TRecord) Substitute(subs map[string]Type) Type {
	fields := make(map[string]Type)
	for name, typ := range t.Fields {
		fields[name] = typ.Substitute(subs)
	}

	var row Type
	if t.Row != nil {
		row = t.Row.Substitute(subs)
	}

	return &TRecord{Fields: fields, Row: row}
}

// TApp represents type application (e.g., Maybe[int])
type TApp struct {
	Constructor Type
	Args        []Type
}

func (t *TApp) String() string {
	args := make([]string, len(t.Args))
	for i, a := range t.Args {
		args[i] = a.String()
	}
	return fmt.Sprintf("%s[%s]", t.Constructor.String(), strings.Join(args, ", "))
}

func (t *TApp) Equals(other Type) bool {
	if o, ok := other.(*TApp); ok {
		if !t.Constructor.Equals(o.Constructor) {
			return false
		}
		if len(t.Args) != len(o.Args) {
			return false
		}
		for i := range t.Args {
			if !t.Args[i].Equals(o.Args[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *TApp) Substitute(subs map[string]Type) Type {
	args := make([]Type, len(t.Args))
	for i, a := range t.Args {
		args[i] = a.Substitute(subs)
	}
	return &TApp{
		Constructor: t.Constructor.Substitute(subs),
		Args:        args,
	}
}

// EffectType represents an effect
type EffectType interface {
	String() string
	Equals(EffectType) bool
}

// SimpleEffect represents a basic effect (IO, FS, Net, etc.)
type SimpleEffect struct {
	Name string
}

func (e *SimpleEffect) String() string {
	return e.Name
}

func (e *SimpleEffect) Equals(other EffectType) bool {
	if o, ok := other.(*SimpleEffect); ok {
		return e.Name == o.Name
	}
	return false
}

// EffectRowOld represents a row of effects (for row polymorphism) - DEPRECATED
type EffectRowOld struct {
	Effects []EffectType
	Row     *EffectVar // Row variable for extensibility
}

func (e *EffectRowOld) String() string {
	effects := make([]string, len(e.Effects))
	for i, eff := range e.Effects {
		effects[i] = eff.String()
	}

	if e.Row != nil {
		effects = append(effects, fmt.Sprintf("...%s", e.Row.String()))
	}

	return fmt.Sprintf("{%s}", strings.Join(effects, ", "))
}

func (e *EffectRowOld) Equals(other EffectType) bool {
	if o, ok := other.(*EffectRowOld); ok {
		if len(e.Effects) != len(o.Effects) {
			return false
		}
		for i := range e.Effects {
			if !e.Effects[i].Equals(o.Effects[i]) {
				return false
			}
		}
		// Check row
		if e.Row == nil && o.Row == nil {
			return true
		}
		if e.Row != nil && o.Row != nil {
			return e.Row.Equals(o.Row)
		}
		return false
	}
	return false
}

// EffectVar represents an effect variable
type EffectVar struct {
	Name string
}

func (e *EffectVar) String() string {
	return e.Name
}

func (e *EffectVar) Equals(other EffectType) bool {
	if o, ok := other.(*EffectVar); ok {
		return e.Name == o.Name
	}
	return false
}

// TypeScheme represents a polymorphic type scheme
type TypeScheme struct {
	TypeVars   []string
	EffectVars []string
	Type       Type
}

func (t *TypeScheme) String() string {
	vars := append(t.TypeVars, t.EffectVars...)
	if len(vars) > 0 {
		return fmt.Sprintf("∀%s. %s", strings.Join(vars, " "), t.Type.String())
	}
	return t.Type.String()
}

// Instantiate creates a fresh instance of the type scheme
func (t *TypeScheme) Instantiate() Type {
	subs := make(map[string]Type)
	for _, v := range t.TypeVars {
		subs[v] = NewTypeVar()
	}
	return t.Type.Substitute(subs)
}

// Common predefined types
var (
	TInt    = &TCon{Name: "int"}
	TFloat  = &TCon{Name: "float"}
	TString = &TCon{Name: "string"}
	TBool   = &TCon{Name: "bool"}
	TUnit   = &TCon{Name: "()"}
	TBytes  = &TCon{Name: "bytes"}
)

// Common effects
var (
	EffectIO    = &SimpleEffect{Name: "IO"}
	EffectFS    = &SimpleEffect{Name: "FS"}
	EffectNet   = &SimpleEffect{Name: "Net"}
	EffectDB    = &SimpleEffect{Name: "DB"}
	EffectRand  = &SimpleEffect{Name: "Rand"}
	EffectClock = &SimpleEffect{Name: "Clock"}
	EffectTrace = &SimpleEffect{Name: "Trace"}
	EffectAsync = &SimpleEffect{Name: "Async"}
)

// Type variable generator
var typeVarCounter int

// NewTypeVar creates a fresh type variable
func NewTypeVar() Type {
	typeVarCounter++
	return &TVar{Name: fmt.Sprintf("t%d", typeVarCounter)}
}

// NewEffectVar creates a fresh effect variable
func NewEffectVar() *EffectVar {
	typeVarCounter++
	return &EffectVar{Name: fmt.Sprintf("e%d", typeVarCounter)}
}

// TypeClass represents a type class
type TypeClass struct {
	Name       string
	TypeParam  string
	Superclass string // Optional superclass
	Methods    map[string]*TypeScheme
}

// Instance represents a type class instance
type Instance struct {
	ClassName string
	Type      Type
	Methods   map[string]Type
}

// Constraint represents a type class constraint
type Constraint struct {
	Class string
	Type  Type
}

func (c *Constraint) String() string {
	return fmt.Sprintf("%s[%s]", c.Class, c.Type.String())
}

// Qualified represents a qualified type (type with constraints)
type Qualified struct {
	Constraints []Constraint
	Type        Type
}

func (q *Qualified) String() string {
	if len(q.Constraints) == 0 {
		return q.Type.String()
	}

	constraints := make([]string, len(q.Constraints))
	for i, c := range q.Constraints {
		constraints[i] = c.String()
	}

	return fmt.Sprintf("(%s) => %s", strings.Join(constraints, ", "), q.Type.String())
}

// Result type for type checking
type Result struct {
	Type   Type
	Errors []error
}

// Error types
type TypeError struct {
	Message string
	Pos     string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Message)
}

// UnificationError represents a unification failure
type UnificationError struct {
	Type1 Type
	Type2 Type
	Pos   string
}

func (e *UnificationError) Error() string {
	return fmt.Sprintf("%s: cannot unify %s with %s", e.Pos, e.Type1.String(), e.Type2.String())
}
