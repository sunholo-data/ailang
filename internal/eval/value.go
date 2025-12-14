package eval

import (
	"fmt"
	"strings"
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

func (f *FloatValue) Type() string { return "float" }
func (f *FloatValue) String() string {
	s := fmt.Sprintf("%g", f.Value)
	// Ensure at least one decimal point for floats (5 -> 5.0)
	if !strings.Contains(s, ".") && !strings.Contains(s, "e") && !strings.Contains(s, "E") {
		s = s + ".0"
	}
	return s
}

// StringValue represents a string value
type StringValue struct {
	Value string
}

func (s *StringValue) Type() string   { return "string" }
func (s *StringValue) String() string { return s.Value }

// BytesValue represents a byte slice value
type BytesValue struct {
	Value []byte
}

func (b *BytesValue) Type() string { return "bytes" }
func (b *BytesValue) String() string {
	// Display as hex for readability, truncate if too long
	if len(b.Value) <= 32 {
		return fmt.Sprintf("<bytes:%x>", b.Value)
	}
	return fmt.Sprintf("<bytes:%x...>", b.Value[:32])
}

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

// ArrayValue represents an array with O(1) indexed access
type ArrayValue struct {
	Elements []Value
}

func (a *ArrayValue) Type() string { return "array" }
func (a *ArrayValue) String() string {
	result := "#["
	for i, elem := range a.Elements {
		if i > 0 {
			result += ", "
		}
		result += elem.String()
	}
	result += "]"
	return result
}

// Get returns the element at index i, or nil if out of bounds
func (a *ArrayValue) Get(i int64) (Value, bool) {
	if i < 0 || i >= int64(len(a.Elements)) {
		return nil, false
	}
	return a.Elements[i], true
}

// Set returns a new array with the element at index i replaced
func (a *ArrayValue) Set(i int64, v Value) *ArrayValue {
	if i < 0 || i >= int64(len(a.Elements)) {
		return a // Out of bounds, return unchanged
	}
	// Copy-on-write
	newElements := make([]Value, len(a.Elements))
	copy(newElements, a.Elements)
	newElements[i] = v
	return &ArrayValue{Elements: newElements}
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

// ConstructorClosure represents an ADT constructor that takes arguments
// When applied, it creates a TaggedValue with the provided fields.
// This is used in the test harness to inject constructor bindings.
type ConstructorClosure struct {
	TypeName string // The ADT name (e.g., "Option")
	CtorName string // Constructor name (e.g., "Some")
	Arity    int    // Number of fields expected
}

func (c *ConstructorClosure) Type() string { return "constructor" }
func (c *ConstructorClosure) String() string {
	return fmt.Sprintf("<constructor %s/%d>", c.CtorName, c.Arity)
}

// Apply creates a TaggedValue from the provided arguments
func (c *ConstructorClosure) Apply(args []Value) (*TaggedValue, error) {
	if len(args) != c.Arity {
		return nil, fmt.Errorf("constructor %s expects %d arguments, got %d", c.CtorName, c.Arity, len(args))
	}
	return &TaggedValue{
		TypeName: c.TypeName,
		CtorName: c.CtorName,
		Fields:   args,
	}, nil
}

// RefCell allows self/mutual recursion via indirection
// Used in LetRec evaluation for function-first semantics (OCaml/Haskell style)
type RefCell struct {
	Val      Value // The actual value (once initialized)
	Init     bool  // Has the value been set?
	Visiting bool  // Currently being evaluated? (for cycle detection)
}

// IndirectValue defers to the cell at read-time
// Environment bindings point to IndirectValue during LetRec evaluation
type IndirectValue struct {
	Cell *RefCell
}

func (iv *IndirectValue) Type() string { return "indirect" }
func (iv *IndirectValue) String() string {
	if iv.Cell.Init {
		return iv.Cell.Val.String()
	}
	return "<uninitialized>"
}

// Force resolves the indirection, checking for initialization and cycles
func (iv *IndirectValue) Force() (Value, error) {
	if !iv.Cell.Init {
		if iv.Cell.Visiting {
			return nil, fmt.Errorf("RT_REC_001: recursive value used before initialization (non-function RHS). Consider making it a function or introducing laziness")
		}
		return nil, fmt.Errorf("RT_REC_002: uninitialized recursive binding; this indicates an internal ordering bug")
	}
	return iv.Cell.Val, nil
}
