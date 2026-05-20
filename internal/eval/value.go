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

// MapValue represents an immutable hash map with O(1) lookup (copy-on-write)
type MapValue struct {
	Entries map[string]*MapEntry // canonical key -> entry
}

// MapEntry stores the original key and value for a map entry
type MapEntry struct {
	Key   Value
	Value Value
}

func (m *MapValue) Type() string { return "map" }

func (m *MapValue) String() string {
	if len(m.Entries) == 0 {
		return "Map{}"
	}
	// Deterministic sorted output (Axiom A1)
	keys := make([]string, 0, len(m.Entries))
	for k := range m.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("Map{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		entry := m.Entries[k]
		b.WriteString(entry.Key.String())
		b.WriteString(": ")
		b.WriteString(entry.Value.String())
	}
	b.WriteByte('}')
	return b.String()
}

// MapKey returns a canonical, collision-free string key for Go map lookup.
// Distinct from String() which is for display only.
// Supported key types: int, string, bool. Returns error for unsupported types.
func MapKey(v Value) (string, error) {
	switch k := v.(type) {
	case *IntValue:
		return fmt.Sprintf("i:%d", k.Value), nil
	case *StringValue:
		return "s:" + k.Value, nil
	case *BoolValue:
		if k.Value {
			return "b:true", nil
		}
		return "b:false", nil
	default:
		return "", fmt.Errorf("unsupported map key type: %T (only int, string, bool supported)", v)
	}
}

// Lookup returns the value for a key, or (nil, false) if not found. O(1).
func (m *MapValue) Lookup(key Value) (Value, bool) {
	k, err := MapKey(key)
	if err != nil {
		return nil, false
	}
	entry, ok := m.Entries[k]
	if !ok {
		return nil, false
	}
	return entry.Value, true
}

// Insert returns a new map with the key-value pair added/updated. O(n) copy-on-write.
func (m *MapValue) Insert(key, val Value) (*MapValue, error) {
	k, err := MapKey(key)
	if err != nil {
		return nil, err
	}
	newEntries := make(map[string]*MapEntry, len(m.Entries)+1)
	for ek, ev := range m.Entries {
		newEntries[ek] = ev
	}
	newEntries[k] = &MapEntry{Key: key, Value: val}
	return &MapValue{Entries: newEntries}, nil
}

// Remove returns a new map without the given key. O(n) copy-on-write.
func (m *MapValue) Remove(key Value) *MapValue {
	k, err := MapKey(key)
	if err != nil {
		return m
	}
	if _, ok := m.Entries[k]; !ok {
		return m // key not present, return unchanged
	}
	newEntries := make(map[string]*MapEntry, len(m.Entries))
	for ek, ev := range m.Entries {
		if ek != k {
			newEntries[ek] = ev
		}
	}
	return &MapValue{Entries: newEntries}
}

// Size returns the number of entries. O(1).
func (m *MapValue) Size() int {
	return len(m.Entries)
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
	Resolver         GlobalResolver  // M-DX-XPKG-RESOLVE: resolver from defining module (for cross-package calls)
	Typed            bool            // Whether Body is typed
	EffectBudgets    map[string]int  // Budget max limits per effect (from @limit annotation)
	EffectMinBudgets map[string]int  // Budget min limits per effect (from @min annotation, M-DX25 M4)
	Preconditions    []*ContractSpec // requires blocks (M-VERIFY-CONTRACTS)
	Postconditions   []*ContractSpec // ensures blocks (M-VERIFY-CONTRACTS)

	// IsZeroArgExport marks functions that have a single implicit unit parameter
	// injected by the parser for `export func name() -> T` syntax (M-ZERO-ARG /
	// S-CALL0). External callers (apiserver, WASM ailangCall, bytecode entrypoint)
	// pass zero args; CallFunction injects a UnitValue automatically when this
	// flag is set. See design_docs/planned/v0_22_0/m-zero-arg-invocation-surfaces.md.
	IsZeroArgExport bool
}

func (f *FunctionValue) Type() string   { return "function" }
func (f *FunctionValue) String() string { return "<function>" }

// isZeroArgUnitInjection reports whether the call needs an implicit unit
// argument. Returns true when the caller passed no arguments and the function
// has the parser's zero-arg-export shape (single param named "_") -- see
// internal/parser/parser_func.go for the parser's injection convention. Used by
// CoreEvaluator.CallFunction (M-ZERO-ARG-SURFACES, v0.22.0).
func isZeroArgUnitInjection(fn *FunctionValue, args []Value) bool {
	return len(args) == 0 && len(fn.Params) == 1 && fn.Params[0] == "_"
}

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
