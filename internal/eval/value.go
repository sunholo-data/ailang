package eval

import (
	"fmt"
	"sort"
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

// BytesValue represents raw binary data (e.g., file content, uploads).
type BytesValue struct {
	Value    []byte
	Filename string // original filename from upload (empty if not from upload)
	MimeType string // detected or declared MIME type (empty if unknown)
}

func (b *BytesValue) Type() string { return "bytes" }
func (b *BytesValue) String() string {
	if b.Filename != "" {
		return fmt.Sprintf("<bytes:%d:%s:%s>", len(b.Value), b.MimeType, b.Filename)
	}
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
	var b strings.Builder
	b.WriteByte('[')
	for i, elem := range l.Elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.String())
	}
	b.WriteByte(']')
	return b.String()
}

// ArrayValue represents an array with O(1) indexed access
type ArrayValue struct {
	Elements []Value
}

func (a *ArrayValue) Type() string { return "array" }
func (a *ArrayValue) String() string {
	var b strings.Builder
	b.WriteString("#[")
	for i, elem := range a.Elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.String())
	}
	b.WriteByte(']')
	return b.String()
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
	var b strings.Builder
	b.WriteByte('(')
	for i, elem := range t.Elements {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(elem.String())
	}
	b.WriteByte(')')
	return b.String()
}

// RecordValue represents a record (struct) value
type RecordValue struct {
	Fields map[string]Value
}

func (r *RecordValue) Type() string { return "record" }
func (r *RecordValue) String() string {
	// Sort keys for deterministic output (Go map iteration is non-deterministic)
	keys := make([]string, 0, len(r.Fields))
	for k := range r.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(r.Fields[k].String())
	}
	b.WriteByte('}')
	return b.String()
}

// ContractSpec represents a requires/ensures contract for a function (M-VERIFY-CONTRACTS)
type ContractSpec struct {
	Kind     string      // "requires" or "ensures"
	Expr     interface{} // The contract expression (ast.Expr or core.CoreExpr)
	Message  string      // Auto-generated message from source (e.g., "limit > 0")
	Location string      // Source location for error messages (e.g., "api.ail:15")
}

// FunctionValue represents a function value
type FunctionValue struct {
	Params           []string
	Body             interface{} // Can be ast.Expr, core.CoreExpr, or typedast.TypedNode
	Env              *Environment
	Typed            bool            // Whether Body is typed
	EffectBudgets    map[string]int  // Budget max limits per effect (from @limit annotation)
	EffectMinBudgets map[string]int  // Budget min limits per effect (from @min annotation, M-DX25 M4)
	Preconditions    []*ContractSpec // requires blocks (M-VERIFY-CONTRACTS)
	Postconditions   []*ContractSpec // ensures blocks (M-VERIFY-CONTRACTS)
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
		return t.CtorName
	}
	var b strings.Builder
	b.WriteString(t.CtorName)
	b.WriteByte('(')
	for i, field := range t.Fields {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(field.String())
	}
	b.WriteByte(')')
	return b.String()
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
