package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPrimitiveTypes tests basic type constructors
func TestPrimitiveTypes(t *testing.T) {
	b := NewBuilder()

	tests := []struct {
		name     string
		builder  func() Type
		expected string
	}{
		{"String", b.String, "string"},
		{"Int", b.Int, "int"},
		{"Bool", b.Bool, "bool"},
		{"Float", b.Float, "float"},
		{"Unit", b.Unit, "()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			typ := tt.builder()
			assert.Equal(t, tt.expected, typ.String())
		})
	}
}

// TestPrimitiveCasing enforces lowercase primitive types to match AILANG source code
func TestPrimitiveCasing(t *testing.T) {
	b := NewBuilder()

	// All primitives MUST be lowercase (except unit which is "()")
	primitives := []struct {
		typ      Type
		name     string
		mustNot  string
	}{
		{b.String(), "string", "String"},
		{b.Int(), "int", "Int"},
		{b.Bool(), "bool", "Bool"},
		{b.Float(), "float", "Float"},
	}

	for _, tc := range primitives {
		t.Run(tc.name, func(t *testing.T) {
			formatted := tc.typ.String()
			assert.Equal(t, tc.name, formatted, "Type must be lowercase")
			assert.NotContains(t, formatted, tc.mustNot, "Type must NOT contain uppercase variant")
		})
	}
}

// TestVar tests type variable construction
func TestVar(t *testing.T) {
	b := NewBuilder()

	a := b.Var("a")
	assert.IsType(t, &TVar2{}, a)
	assert.Equal(t, "a", a.(*TVar2).Name)

	b2 := b.Var("b")
	assert.Equal(t, "b", b2.(*TVar2).Name)
}

// TestCon tests type constructor
func TestCon(t *testing.T) {
	b := NewBuilder()

	result := b.Con("Result")
	assert.IsType(t, &TCon{}, result)
	assert.Equal(t, "Result", result.(*TCon).Name)
}

// TestList tests list type construction
func TestList(t *testing.T) {
	b := NewBuilder()

	listInt := b.List(b.Int())
	assert.IsType(t, &TApp{}, listInt)

	app := listInt.(*TApp)
	assert.IsType(t, &TCon{}, app.Constructor)
	assert.Equal(t, "List", app.Constructor.(*TCon).Name)
	assert.Len(t, app.Args, 1)
	assert.Equal(t, "int", app.Args[0].(*TCon).Name)
}

// TestApp tests multi-argument type application
func TestApp(t *testing.T) {
	b := NewBuilder()

	// Result<String, Error>
	resultType := b.App("Result", b.String(), b.Con("Error"))

	assert.IsType(t, &TApp{}, resultType)
	app := resultType.(*TApp)
	assert.Equal(t, "Result", app.Constructor.(*TCon).Name)
	assert.Len(t, app.Args, 2)
	assert.Equal(t, "string", app.Args[0].(*TCon).Name)
	assert.Equal(t, "Error", app.Args[1].(*TCon).Name)
}

// TestAppZeroArgs tests that App with no args returns just the constructor
func TestAppZeroArgs(t *testing.T) {
	b := NewBuilder()

	typ := b.App("Option")
	assert.IsType(t, &TCon{}, typ)
	assert.Equal(t, "Option", typ.(*TCon).Name)
}

// TestRecordWithFields tests record construction with FieldSpec
func TestRecordWithFields(t *testing.T) {
	b := NewBuilder()

	rec := b.Record(
		Field("name", b.String()),
		Field("age", b.Int()),
		Field("active", b.Bool()),
	)

	assert.IsType(t, &TRecord2{}, rec)
	record := rec.(*TRecord2)

	assert.Len(t, record.Row.Labels, 3)
	assert.Equal(t, "string", record.Row.Labels["name"].(*TCon).Name)
	assert.Equal(t, "int", record.Row.Labels["age"].(*TCon).Name)
	assert.Equal(t, "bool", record.Row.Labels["active"].(*TCon).Name)
}

// TestRecordDuplicateField tests that duplicate field names panic
func TestRecordDuplicateField(t *testing.T) {
	b := NewBuilder()

	assert.Panics(t, func() {
		b.Record(
			Field("name", b.String()),
			Field("name", b.Int()), // Duplicate!
		)
	})
}

// TestRecWithPairs tests Rec convenience method
func TestRecWithPairs(t *testing.T) {
	b := NewBuilder()

	rec := b.Rec(
		"x", b.Int(),
		"y", b.Int(),
	)

	assert.IsType(t, &TRecord2{}, rec)
	record := rec.(*TRecord2)

	assert.Len(t, record.Row.Labels, 2)
	assert.Equal(t, "int", record.Row.Labels["x"].(*TCon).Name)
	assert.Equal(t, "int", record.Row.Labels["y"].(*TCon).Name)
}

// TestRecOddArgs tests that Rec with odd number of args panics
func TestRecOddArgs(t *testing.T) {
	b := NewBuilder()

	assert.Panics(t, func() {
		b.Rec("x", b.Int(), "y") // Missing value for "y"
	})
}

// TestRecDuplicateField tests that Rec detects duplicate fields
func TestRecDuplicateField(t *testing.T) {
	b := NewBuilder()

	assert.Panics(t, func() {
		b.Rec(
			"x", b.Int(),
			"x", b.String(), // Duplicate!
		)
	})
}

// TestFuncBasic tests basic function type construction
func TestFuncBasic(t *testing.T) {
	b := NewBuilder()

	// (String, Int) -> Bool
	funcType := b.Func(b.String(), b.Int()).Returns(b.Bool()).Build()

	assert.IsType(t, &TFunc2{}, funcType)
	fn := funcType.(*TFunc2)

	assert.Len(t, fn.Params, 2)
	assert.Equal(t, "string", fn.Params[0].(*TCon).Name)
	assert.Equal(t, "int", fn.Params[1].(*TCon).Name)
	assert.Equal(t, "bool", fn.Return.(*TCon).Name)
	assert.Empty(t, fn.EffectRow.Labels) // Pure function
}

// TestFuncWithEffects tests function with effects
func TestFuncWithEffects(t *testing.T) {
	b := NewBuilder()

	// (String) -> () ! {IO}
	funcType := b.Func(b.String()).Returns(b.Unit()).Effects("IO")

	assert.IsType(t, &TFunc2{}, funcType)
	fn := funcType.(*TFunc2)

	assert.Len(t, fn.Params, 1)
	assert.Equal(t, "string", fn.Params[0].(*TCon).Name)
	assert.Equal(t, "()", fn.Return.(*TCon).Name)

	// Check effects
	assert.Len(t, fn.EffectRow.Labels, 1)
	assert.Contains(t, fn.EffectRow.Labels, "IO")
}

// TestFuncMultipleEffects tests function with multiple effects
func TestFuncMultipleEffects(t *testing.T) {
	b := NewBuilder()

	// (String) -> String ! {IO, Net}
	funcType := b.Func(b.String()).Returns(b.String()).Effects("IO", "Net")

	fn := funcType.(*TFunc2)
	assert.Len(t, fn.EffectRow.Labels, 2)
	assert.Contains(t, fn.EffectRow.Labels, "IO")
	assert.Contains(t, fn.EffectRow.Labels, "Net")
}

// TestFuncNoReturn tests that Build panics without Returns
func TestFuncNoReturn(t *testing.T) {
	b := NewBuilder()

	assert.Panics(t, func() {
		b.Func(b.String()).Build() // No Returns() call!
	})
}

// TestComplexType tests building a complex type like httpRequest
func TestComplexType(t *testing.T) {
	b := NewBuilder()

	// Header type: {name: String, value: String}
	headerType := b.Record(
		Field("name", b.String()),
		Field("value", b.String()),
	)

	// Response type: {status: Int, headers: List<Header>, body: String, ok: Bool}
	responseType := b.Record(
		Field("status", b.Int()),
		Field("headers", b.List(headerType)),
		Field("body", b.String()),
		Field("ok", b.Bool()),
	)

	// (String, String, List<Header>, String) -> Result<Response, NetError> ! {Net}
	httpRequestType := b.Func(
		b.String(),         // method
		b.String(),         // url
		b.List(headerType), // headers
		b.String(),         // body
	).Returns(
		b.App("Result", responseType, b.Con("NetError")),
	).Effects("Net")

	// Verify structure
	fn := httpRequestType.(*TFunc2)
	assert.Len(t, fn.Params, 4)
	assert.Equal(t, "string", fn.Params[0].(*TCon).Name)
	assert.Equal(t, "string", fn.Params[1].(*TCon).Name)

	// Check return type is Result
	returnType := fn.Return.(*TApp)
	assert.Equal(t, "Result", returnType.Constructor.(*TCon).Name)
	assert.Len(t, returnType.Args, 2)

	// Check effects
	assert.Contains(t, fn.EffectRow.Labels, "Net")
}

// TestNestedRecords tests nested record types
func TestNestedRecords(t *testing.T) {
	b := NewBuilder()

	// Inner record: {x: Int, y: Int}
	point := b.Record(
		Field("x", b.Int()),
		Field("y", b.Int()),
	)

	// Outer record: {center: Point, radius: Float}
	circle := b.Record(
		Field("center", point),
		Field("radius", b.Float()),
	)

	rec := circle.(*TRecord2)
	assert.Len(t, rec.Row.Labels, 2)

	// Check nested structure
	centerType := rec.Row.Labels["center"]
	assert.IsType(t, &TRecord2{}, centerType)
	innerRec := centerType.(*TRecord2)
	assert.Len(t, innerRec.Row.Labels, 2)
	assert.Equal(t, "int", innerRec.Row.Labels["x"].(*TCon).Name)
}

// TestRowTail tests effect row polymorphism (future feature)
func TestRowTail(t *testing.T) {
	b := NewBuilder()

	// (String) -> () ! {IO | ρ}
	funcType := b.Func(b.String()).
		Returns(b.Unit()).
		RowTail("ρ").
		Effects("IO")

	fn := funcType.(*TFunc2)
	assert.NotNil(t, fn.EffectRow.Tail)
	assert.Equal(t, "ρ", fn.EffectRow.Tail.Name)
	assert.Contains(t, fn.EffectRow.Labels, "IO")
}

// TestTypeString tests that types have reasonable string representations
func TestTypeString(t *testing.T) {
	b := NewBuilder()

	tests := []struct {
		name     string
		typ      Type
		contains string // String should contain this
	}{
		{
			"Simple function",
			b.Func(b.String()).Returns(b.Int()).Build(),
			"string",
		},
		{
			"List type",
			b.List(b.String()),
			"List",
		},
		{
			"Record type",
			b.Record(Field("name", b.String())),
			"name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.typ.String()
			assert.NotEmpty(t, str)
			if tt.contains != "" {
				assert.Contains(t, str, tt.contains)
			}
		})
	}
}

// TestBuilderReuse tests that builder can be reused safely
func TestBuilderReuse(t *testing.T) {
	b := NewBuilder()

	// Create multiple types with same builder
	t1 := b.String()
	t2 := b.Int()
	t3 := b.List(b.Bool())

	assert.Equal(t, "string", t1.(*TCon).Name)
	assert.Equal(t, "int", t2.(*TCon).Name)
	assert.IsType(t, &TApp{}, t3)
}

// TestCompareWithManualConstruction tests that builder produces same types as manual construction
func TestCompareWithManualConstruction(t *testing.T) {
	b := NewBuilder()

	// Manual construction
	manual := &TFunc2{
		Params: []Type{
			&TCon{Name: "string"},
		},
		Return: &TCon{Name: "int"},
		EffectRow: &Row{
			Kind:   KRow{ElemKind: KEffect{}},
			Labels: make(map[string]Type),
			Tail:   nil,
		},
	}

	// Builder construction
	builder := b.Func(b.String()).Returns(b.Int()).Build()

	// Compare structure (not using == because pointer equality)
	manualFn := manual
	builderFn := builder.(*TFunc2)

	assert.Equal(t, len(manualFn.Params), len(builderFn.Params))
	assert.Equal(t, manualFn.Params[0].String(), builderFn.Params[0].String())
	assert.Equal(t, manualFn.Return.String(), builderFn.Return.String())
	assert.Equal(t, len(manualFn.EffectRow.Labels), len(builderFn.EffectRow.Labels))
}

// TestLOCReduction demonstrates the LOC reduction from builder
func TestLOCReduction(t *testing.T) {
	// This test documents the improvement: complex types in ~10 lines vs 35 lines

	b := NewBuilder()

	// Before: 35+ lines of nested structs (see link/builtin_module.go)
	// After: ~10 lines with builder

	headerType := b.Rec("name", b.String(), "value", b.String())

	responseType := b.Rec(
		"status", b.Int(),
		"headers", b.List(headerType),
		"body", b.String(),
		"ok", b.Bool(),
	)

	httpRequestType := b.Func(
		b.String(), b.String(), b.List(headerType), b.String(),
	).Returns(
		b.App("Result", responseType, b.Con("NetError")),
	).Effects("Net")

	// Verify it actually works
	require.NotNil(t, httpRequestType)
	fn := httpRequestType.(*TFunc2)
	assert.Len(t, fn.Params, 4)
	assert.Contains(t, fn.EffectRow.Labels, "Net")

	// This type construction is:
	// - Self-documenting (clear structure)
	// - Compile-time safe (wrong field = error)
	// - Reusable (headerType extracted once)
	// - ~70% fewer lines than nested structs
}
