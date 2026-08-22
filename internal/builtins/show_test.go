package builtins

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo-data/ailang/internal/effects/testctx"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestShow_AllEvalValueImplementationsHandled(t *testing.T) {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	builtinsDir := filepath.Dir(thisFile)

	valueTypes := evalValueImplementations(t, filepath.Join(builtinsDir, "..", "eval"))
	handledTypes := showValueCases(t, filepath.Join(builtinsDir, "show.go"))

	var unhandled []string
	for _, name := range valueTypes {
		if !handledTypes[name] {
			unhandled = append(unhandled, name)
		}
	}
	require.Empty(t, unhandled, "showValue must explicitly handle every eval.Value implementation")
}

func evalValueImplementations(t *testing.T, evalDir string) []string {
	t.Helper()
	methods := make(map[string]map[string]bool)
	fset := token.NewFileSet()
	files, err := filepath.Glob(filepath.Join(evalDir, "*.go"))
	require.NoError(t, err)
	require.NotEmpty(t, files)
	for _, filename := range files {
		if filepath.Ext(filename) != ".go" || len(filename) >= 8 && filename[len(filename)-8:] == "_test.go" {
			continue
		}
		file, parseErr := parser.ParseFile(fset, filename, nil, 0)
		require.NoError(t, parseErr)
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 || fn.Name == nil {
				continue
			}
			star, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			ident, ok := star.X.(*ast.Ident)
			if !ok {
				continue
			}
			if methods[ident.Name] == nil {
				methods[ident.Name] = make(map[string]bool)
			}
			methods[ident.Name][fn.Name.Name] = true
		}
	}
	var implementations []string
	for name, names := range methods {
		if names["Type"] && names["String"] {
			implementations = append(implementations, name)
		}
	}
	sort.Strings(implementations)
	return implementations
}

func showValueCases(t *testing.T, filename string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	require.NoError(t, err)
	handled := make(map[string]bool)
	var showValue *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "showValue" {
			showValue = fn
			break
		}
	}
	require.NotNil(t, showValue, "showValue declaration not found")
	ast.Inspect(showValue.Body, func(node ast.Node) bool {
		typeSwitch, ok := node.(*ast.TypeSwitchStmt)
		if !ok {
			return true
		}
		for _, statement := range typeSwitch.Body.List {
			clause := statement.(*ast.CaseClause)
			for _, expr := range clause.List {
				star, ok := expr.(*ast.StarExpr)
				if !ok {
					continue
				}
				selector, ok := star.X.(*ast.SelectorExpr)
				if ok {
					handled[selector.Sel.Name] = true
				}
			}
		}
		return false
	})
	return handled
}

func TestShow_Primitives(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		// Integers
		{"int positive", testctx.MakeInt(42), "42"},
		{"int negative", testctx.MakeInt(-17), "-17"},
		{"int zero", testctx.MakeInt(0), "0"},
		{"int large", testctx.MakeInt(1000000), "1000000"},

		// Floats
		{"float positive", testctx.MakeFloat(3.14), "3.14"},
		{"float negative", testctx.MakeFloat(-2.5), "-2.5"},
		{"float zero", testctx.MakeFloat(0.0), "0.0"},
		{"float integer-like", testctx.MakeFloat(5.0), "5.0"},
		{"float small", testctx.MakeFloat(0.001), "0.001"},
		{"float large", testctx.MakeFloat(123456.789), "123456.789"},

		// Booleans
		{"bool true", testctx.MakeBool(true), "true"},
		{"bool false", testctx.MakeBool(false), "false"},

		// Strings
		{"string empty", testctx.MakeString(""), ""},
		{"string simple", testctx.MakeString("hello"), "hello"},
		{"string with spaces", testctx.MakeString("hello world"), "hello world"},
		{"string with quotes", testctx.MakeString("hello \"world\""), "hello \"world\""},
		{"string with newlines", testctx.MakeString("line1\nline2"), "line1\nline2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_FloatSpecialValues(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		value    float64
		expected string
	}{
		{"NaN", math.NaN(), "NaN"},
		{"positive infinity", math.Inf(1), "Inf"},
		{"negative infinity", math.Inf(-1), "-Inf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{testctx.MakeFloat(tt.value)})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Lists(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"empty list",
			testctx.MakeList([]eval.Value{}),
			"[]",
		},
		{
			"single int",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(42),
			}),
			"[42]",
		},
		{
			"multiple ints",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(1),
				testctx.MakeInt(2),
				testctx.MakeInt(3),
			}),
			"[1, 2, 3]",
		},
		{
			"mixed types",
			testctx.MakeList([]eval.Value{
				testctx.MakeInt(42),
				testctx.MakeString("hello"),
				testctx.MakeBool(true),
			}),
			"[42, hello, true]",
		},
		{
			"nested lists",
			testctx.MakeList([]eval.Value{
				testctx.MakeList([]eval.Value{
					testctx.MakeInt(1),
					testctx.MakeInt(2),
				}),
				testctx.MakeList([]eval.Value{
					testctx.MakeInt(3),
					testctx.MakeInt(4),
				}),
			}),
			"[[1, 2], [3, 4]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Records(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"empty record",
			testctx.MakeRecord(map[string]eval.Value{}),
			"{}",
		},
		{
			"single field",
			testctx.MakeRecord(map[string]eval.Value{
				"x": testctx.MakeInt(42),
			}),
			"{x: 42}",
		},
		{
			"multiple fields (sorted)",
			testctx.MakeRecord(map[string]eval.Value{
				"name": testctx.MakeString("Alice"),
				"age":  testctx.MakeInt(30),
			}),
			"{age: 30, name: Alice}", // Sorted alphabetically
		},
		{
			"nested record",
			testctx.MakeRecord(map[string]eval.Value{
				"person": testctx.MakeRecord(map[string]eval.Value{
					"name": testctx.MakeString("Bob"),
					"age":  testctx.MakeInt(25),
				}),
			}),
			"{person: {age: 25, name: Bob}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_Constructors(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    eval.Value
		expected string
	}{
		{
			"constructor without args",
			&eval.TaggedValue{
				CtorName: "None",
				Fields:   []eval.Value{},
			},
			"None",
		},
		{
			"constructor with single arg",
			&eval.TaggedValue{
				CtorName: "Some",
				Fields:   []eval.Value{testctx.MakeInt(42)},
			},
			"Some(42)",
		},
		{
			"constructor with multiple args",
			&eval.TaggedValue{
				CtorName: "Pair",
				Fields: []eval.Value{
					testctx.MakeInt(1),
					testctx.MakeString("hello"),
				},
			},
			"Pair(1, hello)",
		},
		{
			"nested constructors",
			&eval.TaggedValue{
				CtorName: "Some",
				Fields: []eval.Value{
					&eval.TaggedValue{
						CtorName: "Just",
						Fields:   []eval.Value{testctx.MakeInt(42)},
					},
				},
			},
			"Some(Just(42))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := showImpl(ctx.EffContext, []eval.Value{tt.input})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, testctx.GetString(result))
		})
	}
}

func TestShow_DepthLimit(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Create deeply nested list: [[[[42]]]]
	deepList := testctx.MakeInt(42)
	for i := 0; i < 5; i++ {
		deepList = testctx.MakeList([]eval.Value{deepList})
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{deepList})
	require.NoError(t, err)

	// Should hit depth limit and show "..."
	assert.Contains(t, testctx.GetString(result), "...")
}

func TestShow_LongStrings(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	// Create a very long list
	elements := make([]eval.Value, 50)
	for i := 0; i < 50; i++ {
		elements[i] = testctx.MakeInt(i)
	}
	longList := testctx.MakeList(elements)

	result, err := showImpl(ctx.EffContext, []eval.Value{longList})
	require.NoError(t, err)

	output := testctx.GetString(result)

	// Should truncate with "..." in the middle
	assert.Contains(t, output, "...")
	assert.LessOrEqual(t, len(output), maxWidth+10) // Some tolerance
}

func TestShow_FunctionValue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	funcVal := &eval.FunctionValue{
		Params: []string{"x"},
		Body:   nil, // Not important for show
		Env:    nil,
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{funcVal})
	require.NoError(t, err)
	assert.Equal(t, "<function>", testctx.GetString(result))
}

func TestShow_RemainingValueForms(t *testing.T) {
	initialized := &eval.IndirectValue{Cell: &eval.RefCell{Val: &eval.TupleValue{
		Elements: []eval.Value{testctx.MakeInt(1), testctx.MakeString("a")},
	}, Init: true}}
	tests := []struct {
		name     string
		value    eval.Value
		expected string
	}{
		{"tuple", &eval.TupleValue{Elements: []eval.Value{testctx.MakeInt(1), testctx.MakeString("a")}}, "(1, a)"},
		{"array", &eval.ArrayValue{Elements: []eval.Value{testctx.MakeInt(1), testctx.MakeInt(2)}}, "#[1, 2]"},
		{"map", &eval.MapValue{Entries: map[string]*eval.MapEntry{
			"s:b": {Key: testctx.MakeString("b"), Value: testctx.MakeInt(2)},
			"s:a": {Key: testctx.MakeString("a"), Value: testctx.MakeInt(1)},
		}}, "Map{a: 1, b: 2}"},
		{"bytes", &eval.BytesValue{Value: []byte{0x01, 0xab}}, "<bytes:01ab>"},
		{"builtin", &eval.BuiltinFunction{Name: "example"}, "<function>"},
		{"constructor closure", &eval.ConstructorClosure{CtorName: "Some", Arity: 1}, "<function>"},
		{"indirect", initialized, "(1, a)"},
		{"uninitialized indirect", &eval.IndirectValue{Cell: &eval.RefCell{}}, "<uninitialized>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, showValue(tt.value, 0))
		})
	}
}

func TestShow_ErrorValue(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	errVal := &eval.ErrorValue{
		Message: "something went wrong",
	}

	result, err := showImpl(ctx.EffContext, []eval.Value{errVal})
	require.NoError(t, err)
	assert.Equal(t, "Error: something went wrong", testctx.GetString(result))
}

func TestShow_TypeRegistration(t *testing.T) {
	// Verify show is registered in the builtin registry
	spec, exists := GetSpec("show")
	require.True(t, exists, "show should be registered")

	assert.Equal(t, "show", spec.Name)
	assert.Equal(t, "$builtin", spec.Module)
	assert.Equal(t, 1, spec.NumArgs)
	assert.True(t, spec.IsPure)
	assert.Equal(t, "", spec.Effect)
}
