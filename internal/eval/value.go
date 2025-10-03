package eval

import (
	"fmt"
)

// Value represents a runtime value in AILANG
type Value interface {
	Type() string
	String() string
}

// IntValue represents an integer value
type IntValue struct {
	Value int
}

func (i *IntValue) Type() string   { return "int" }
func (i *IntValue) String() string { return fmt.Sprintf("%d", i.Value) }

// FloatValue represents a floating-point value
type FloatValue struct {
	Value float64
}

func (f *FloatValue) Type() string   { return "float" }
func (f *FloatValue) String() string { return fmt.Sprintf("%g", f.Value) }

// StringValue represents a string value
type StringValue struct {
	Value string
}

func (s *StringValue) Type() string   { return "string" }
func (s *StringValue) String() string { return s.Value }

// BoolValue represents a boolean value
type BoolValue struct {
	Value bool
}

func (b *BoolValue) Type() string { return "bool" }
func (b *BoolValue) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

// UnitValue represents the unit value ()
type UnitValue struct{}

func (u *UnitValue) Type() string   { return "unit" }
func (u *UnitValue) String() string { return "()" }

// ListValue represents a list of values
type ListValue struct {
	Elements []Value
}

func (l *ListValue) Type() string { return "list" }
func (l *ListValue) String() string {
	result := "["
	for i, elem := range l.Elements {
		if i > 0 {
			result += ", "
		}
		result += elem.String()
	}
	result += "]"
	return result
}

// TupleValue represents a tuple of values
type TupleValue struct {
	Elements []Value
}

func (t *TupleValue) Type() string { return "tuple" }
func (t *TupleValue) String() string {
	result := "("
	for i, elem := range t.Elements {
		if i > 0 {
			result += ", "
		}
		result += elem.String()
	}
	result += ")"
	return result
}

// RecordValue represents a record (struct) value
type RecordValue struct {
	Fields map[string]Value
}

func (r *RecordValue) Type() string { return "record" }
func (r *RecordValue) String() string {
	result := "{"
	first := true
	for k, v := range r.Fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s: %s", k, v.String())
		first = false
	}
	result += "}"
	return result
}

// FunctionValue represents a function value
type FunctionValue struct {
	Params []string
	Body   interface{} // Can be ast.Expr, core.CoreExpr, or typedast.TypedNode
	Env    *Environment
	Typed  bool // Whether Body is typed
}

func (f *FunctionValue) Type() string   { return "function" }
func (f *FunctionValue) String() string { return "<function>" }

// BuiltinFunction represents a built-in function
type BuiltinFunction struct {
	Name string
	Fn   func(args []Value) (Value, error)
}

func (b *BuiltinFunction) Type() string   { return "builtin" }
func (b *BuiltinFunction) String() string { return fmt.Sprintf("<builtin: %s>", b.Name) }

// ErrorValue represents an error value
type ErrorValue struct {
	Message string
}

func (e *ErrorValue) Type() string   { return "error" }
func (e *ErrorValue) String() string { return fmt.Sprintf("Error: %s", e.Message) }

// TaggedValue represents an ADT constructor at runtime
type TaggedValue struct {
	ModulePath string  // Module where type is defined (e.g., "std/option") - prevents ambiguity
	TypeName   string  // The ADT name (e.g., "Option")
	CtorName   string  // Constructor name (e.g., "Some", "None")
	Fields     []Value // Constructor field values
}

func (t *TaggedValue) Type() string { return t.TypeName }
func (t *TaggedValue) String() string {
	if len(t.Fields) == 0 {
		// Nullary constructor: None
		return t.CtorName
	}
	// Constructor with fields: Some(42)
	result := t.CtorName + "("
	for i, field := range t.Fields {
		if i > 0 {
			result += ", "
		}
		result += field.String()
	}
	result += ")"
	return result
}
