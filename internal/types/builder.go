package types

// Builder provides a fluent API for constructing type signatures
// This eliminates the need for verbose nested struct literals and provides:
//   - Readable, self-documenting type construction
//   - Compile-time safety (wrong field name = compile error)
//   - Future-proofing for polymorphism and row types
//
// Example usage:
//
//	T := NewBuilder()
//	httpReqType := T.Func(
//	    T.String(), T.String(),
//	    T.List(T.Record(Field("name", T.String()), Field("value", T.String()))),
//	    T.String(),
//	).Returns(
//	    T.App("Result", responseType, T.Con("NetError")),
//	).Effects("Net")
type Builder struct{}

// NewBuilder creates a new type builder
func NewBuilder() *Builder {
	return &Builder{}
}

// Primitive type constructors

// String returns the String type
func (b *Builder) String() Type {
	return &TCon{Name: "String"}
}

// Int returns the Int type
func (b *Builder) Int() Type {
	return &TCon{Name: "Int"}
}

// Bool returns the Bool type
func (b *Builder) Bool() Type {
	return &TCon{Name: "Bool"}
}

// Float returns the Float type
func (b *Builder) Float() Type {
	return &TCon{Name: "Float"}
}

// Unit returns the unit type ()
func (b *Builder) Unit() Type {
	return &TCon{Name: "()"}
}

// Type variable constructors (for polymorphism)

// Var creates a type variable (e.g., 'a', 'b')
// Used for polymorphic types like: forall a. a -> a
func (b *Builder) Var(name string) Type {
	return &TVar2{Name: name, Kind: KStar{}}
}

// Constructor type constructors

// Con creates a type constructor (e.g., "Result", "Option", "List")
func (b *Builder) Con(name string) Type {
	return &TCon{Name: name}
}

// App creates a type application (e.g., List<String>, Result<T, E>)
// For higher-kinded types like Result<Response, NetError>
func (b *Builder) App(con string, args ...Type) Type {
	if len(args) == 0 {
		return &TCon{Name: con}
	}

	return &TApp{
		Constructor: &TCon{Name: con},
		Args:        args,
	}
}

// Collection constructors

// List creates a list type: List<T>
func (b *Builder) List(elem Type) Type {
	return &TApp{
		Constructor: &TCon{Name: "List"},
		Args:        []Type{elem},
	}
}

// Record builders

// FieldSpec represents a record field specification
type FieldSpec struct {
	Name string
	Type Type
}

// Field creates a record field specification
// Used with Record() builder
func Field(name string, typ Type) FieldSpec {
	return FieldSpec{Name: name, Type: typ}
}

// Record creates a record type with the given fields
// Duplicate field names will panic at build time
//
// Example:
//
//	T.Record(
//	    Field("name", T.String()),
//	    Field("age", T.Int()),
//	)
func (b *Builder) Record(fields ...FieldSpec) Type {
	labels := make(map[string]Type)
	for _, f := range fields {
		if _, exists := labels[f.Name]; exists {
			panic("duplicate field name: " + f.Name)
		}
		labels[f.Name] = f.Type
	}

	return &TRecord2{
		Row: &Row{
			Kind:   KRow{ElemKind: KRecord{}},
			Labels: labels,
			Tail:   nil, // No row polymorphism yet
		},
	}
}

// Rec creates a record type with alternating key-value pairs
// This is a convenience method for simple records
//
// Example:
//
//	T.Rec("name", T.String(), "age", T.Int())
func (b *Builder) Rec(pairs ...interface{}) Type {
	if len(pairs)%2 != 0 {
		panic("Rec() requires even number of arguments (key-value pairs)")
	}

	labels := make(map[string]Type)
	for i := 0; i < len(pairs); i += 2 {
		name := pairs[i].(string)
		typ := pairs[i+1].(Type)

		if _, exists := labels[name]; exists {
			panic("duplicate field name: " + name)
		}
		labels[name] = typ
	}

	return &TRecord2{
		Row: &Row{
			Kind:   KRow{ElemKind: KRecord{}},
			Labels: labels,
			Tail:   nil,
		},
	}
}

// Function type builders

// FuncBuilder provides a fluent interface for building function types
type FuncBuilder struct {
	builder *Builder
	params  []Type
	ret     Type
	effects []string
	rowTail *RowVar // For effect row polymorphism (future)
}

// Func starts building a function type
// Takes the parameter types as arguments
//
// Example:
//
//	T.Func(T.String(), T.Int()).Returns(T.Bool())
func (b *Builder) Func(params ...Type) *FuncBuilder {
	return &FuncBuilder{
		builder: b,
		params:  params,
	}
}

// Returns sets the return type of the function
// This is required before calling Effects() or Build()
func (fb *FuncBuilder) Returns(ret Type) *FuncBuilder {
	fb.ret = ret
	return fb
}

// Effects adds effect annotations and builds the function type
// Multiple effects can be specified. Returns the final Type.
//
// Example:
//
//	typ := T.Func(T.String()).Returns(T.Unit()).Effects("IO", "Net")
func (fb *FuncBuilder) Effects(eff ...string) Type {
	fb.effects = eff
	return fb.Build()
}

// RowTail adds a row variable tail for effect row polymorphism
// This is for future extensibility
//
// Example:
//
//	typ := T.Func(T.String()).Returns(T.Unit()).RowTail("ρ").Effects("IO")
func (fb *FuncBuilder) RowTail(tailVar string) *FuncBuilder {
	fb.rowTail = &RowVar{
		Name: tailVar,
		Kind: KRow{ElemKind: KEffect{}},
	}
	return fb
}

// Build constructs the final function type
// Usually not needed as Effects() automatically builds
func (fb *FuncBuilder) Build() Type {
	// If no return type set, this is an error
	if fb.ret == nil {
		panic("function type must have a return type (call Returns() first)")
	}

	// Build effect row
	effectRow := &Row{
		Kind:   KRow{ElemKind: KEffect{}},
		Labels: make(map[string]Type),
		Tail:   fb.rowTail,
	}

	for _, eff := range fb.effects {
		effectRow.Labels[eff] = &TCon{Name: eff}
	}

	return &TFunc2{
		Params:    fb.params,
		EffectRow: effectRow,
		Return:    fb.ret,
	}
}
