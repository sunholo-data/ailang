package parser

import (
	"testing"
)

// TestLiterals tests parsing of all literal types
func TestLiterals(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// Integer literals
		{"int_zero", "0", "expr/int_zero"},
		{"int_positive", "42", "expr/int_positive"},
		{"int_negative", "-123", "expr/int_negative"},
		{"int_large", "9999999", "expr/int_large"},

		// Float literals
		{"float_simple", "3.14", "expr/float_simple"},
		{"float_zero", "0.0", "expr/float_zero"},
		{"float_negative", "-2.5", "expr/float_negative"},
		{"float_scientific", "1.5e10", "expr/float_scientific"},
		{"float_scientific_negative", "2.5e-3", "expr/float_scientific_negative"},

		// String literals
		{"string_empty", `""`, "expr/string_empty"},
		{"string_simple", `"hello"`, "expr/string_simple"},
		{"string_with_spaces", `"hello world"`, "expr/string_with_spaces"},
		{"string_with_escapes", `"hello\nworld"`, "expr/string_with_escapes"},
		{"string_with_quotes", `"say \"hi\""`, "expr/string_with_quotes"},

		// Boolean literals
		{"bool_true", "true", "expr/bool_true"},
		{"bool_false", "false", "expr/bool_false"},

		// Unit literal
		{"unit", "()", "expr/unit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestIdentifiers tests parsing of identifiers
func TestIdentifiers(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"ident_simple", "x", "expr/ident_simple"},
		{"ident_multichar", "foo", "expr/ident_multichar"},
		{"ident_with_underscore", "foo_bar", "expr/ident_with_underscore"},
		{"ident_with_number", "x1", "expr/ident_with_number"},
		{"ident_camelCase", "fooBar", "expr/ident_camelCase"},
		{"ident_PascalCase", "FooBar", "expr/ident_PascalCase"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestBinaryOperators tests all binary operators
func TestBinaryOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// Arithmetic
		{"add", "1 + 2", "expr/add"},
		{"subtract", "5 - 3", "expr/subtract"},
		{"multiply", "4 * 3", "expr/multiply"},
		{"divide", "10 / 2", "expr/divide"},
		{"modulo", "7 % 3", "expr/modulo"},

		// Comparison
		{"equal", "x == y", "expr/equal"},
		{"not_equal", "x != y", "expr/not_equal"},
		{"less_than", "x < y", "expr/less_than"},
		{"less_equal", "x <= y", "expr/less_equal"},
		{"greater_than", "x > y", "expr/greater_than"},
		{"greater_equal", "x >= y", "expr/greater_equal"},

		// Logical
		{"and", "x && y", "expr/and"},
		{"or", "x || y", "expr/or"},

		// String concatenation
		{"concat", `"hello" ++ "world"`, "expr/concat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestUnaryOperators tests unary operators
func TestUnaryOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"negate", "-x", "expr/negate"},
		{"negate_literal", "-42", "expr/negate_literal"},
		{"not", "!true", "expr/not"},
		{"not_variable", "!flag", "expr/not_variable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestLists tests list literal parsing
func TestLists(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"list_empty", "[]", "expr/list_empty"},
		{"list_one_element", "[1]", "expr/list_one_element"},
		{"list_multiple", "[1, 2, 3]", "expr/list_multiple"},
		{"list_nested", "[[1, 2], [3, 4]]", "expr/list_nested"},
		// TODO: Function calls in lists not yet supported
		// {"list_mixed_expr", "[x, 1 + 2, foo()]", "expr/list_mixed_expr"},
		{"list_trailing_comma", "[1, 2, 3,]", "expr/list_trailing_comma"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestTuples tests tuple literal parsing
func TestTuples(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"tuple_two", "(1, 2)", "expr/tuple_two"},
		{"tuple_three", "(1, 2, 3)", "expr/tuple_three"},
		{"tuple_mixed", "(x, \"hello\", true)", "expr/tuple_mixed"},
		{"tuple_nested", "((1, 2), (3, 4))", "expr/tuple_nested"},
		// TODO: Trailing commas in tuples not yet supported
		// {"tuple_trailing_comma", "(1, 2,)", "expr/tuple_trailing_comma"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestRecords tests record literal parsing
func TestRecords(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"record_empty", "{}", "expr/record_empty"},
		{"record_one_field", "{x: 1}", "expr/record_one_field"},
		{"record_multiple", "{x: 1, y: 2}", "expr/record_multiple"},
		{"record_nested", "{point: {x: 1, y: 2}}", "expr/record_nested"},
		{"record_mixed", "{name: \"Alice\", age: 30, active: true}", "expr/record_mixed"},
		{"record_trailing_comma", "{x: 1, y: 2,}", "expr/record_trailing_comma"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestRecordAccess tests field access
func TestRecordAccess(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"access_simple", "point.x", "expr/access_simple"},
		{"access_chain", "user.address.city", "expr/access_chain"},
		{"access_after_call", "getUser().name", "expr/access_after_call"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestLambdas tests lambda expressions
func TestLambdas(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"lambda_one_param", `\x. x + 1`, "expr/lambda_one_param"},
		{"lambda_two_params", `\x y. x + y`, "expr/lambda_two_params"},
		// TODO: Lambda type annotations not yet supported
		// {"lambda_with_types", `\(x: int) (y: int). x + y`, "expr/lambda_with_types"},
		{"lambda_nested", `\x. \y. x + y`, "expr/lambda_nested"},
		// {"lambda_return_type", `\(x: int) -> int. x * 2`, "expr/lambda_return_type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestFunctionCalls tests function application
func TestFunctionCalls(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"call_no_args", "foo()", "expr/call_no_args"},
		{"call_one_arg", "foo(1)", "expr/call_one_arg"},
		{"call_multiple_args", "foo(1, 2, 3)", "expr/call_multiple_args"},
		{"call_nested", "foo(bar(x))", "expr/call_nested"},
		{"call_with_operators", "foo(x + 1, y * 2)", "expr/call_with_operators"},
		{"call_chain", "foo().bar().baz()", "expr/call_chain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestLetExpressions tests let bindings
func TestLetExpressions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"let_simple", "let x = 1 in x", "expr/let_simple"},
		{"let_with_expr", "let x = 1 + 2 in x * 3", "expr/let_with_expr"},
		{"let_nested", "let x = 1 in let y = 2 in x + y", "expr/let_nested"},
		{"let_with_type", "let x: int = 42 in x", "expr/let_with_type"},
		// TODO: let rec syntax not yet supported
		// {"let_recursive", "let rec factorial = \\n. if n <= 1 then 1 else n * factorial(n - 1) in factorial", "expr/let_recursive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestIfExpressions tests conditional expressions
func TestIfExpressions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"if_simple", "if true then 1 else 2", "expr/if_simple"},
		{"if_with_comparison", "if x > 0 then \"pos\" else \"neg\"", "expr/if_with_comparison"},
		{"if_nested", "if x > 0 then if x > 10 then \"large\" else \"small\" else \"negative\"", "expr/if_nested"},
		{"if_with_let", "if x > 0 then let y = x * 2 in y else 0", "expr/if_with_let"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestMatchExpressions tests pattern matching
func TestMatchExpressions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"match_simple", "match x { 1 => \"one\", 2 => \"two\", _ => \"other\" }", "expr/match_simple"},
		{"match_with_guard", "match x { n if n > 0 => \"positive\", _ => \"other\" }", "expr/match_with_guard"},
		// TODO: List and tuple patterns not yet supported in match expressions
		// {"match_list", "match list { [] => \"empty\", [x] => \"one\", _ => \"many\" }", "expr/match_list"},
		// {"match_tuple", "match pair { (0, 0) => \"origin\", (x, y) => \"point\" }", "expr/match_tuple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestGroupedExpressions tests parenthesized expressions
func TestGroupedExpressions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{"grouped_simple", "(42)", "expr/grouped_simple"},
		{"grouped_operator", "(1 + 2)", "expr/grouped_operator"},
		{"grouped_nested", "((1 + 2) * 3)", "expr/grouped_nested"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestComplexExpressions tests combinations of expressions
func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"complex_arithmetic",
			"(a + b) * (c - d) / e",
			"expr/complex_arithmetic",
		},
		{
			"complex_with_calls",
			"foo(x + 1) + bar(y * 2)",
			"expr/complex_with_calls",
		},
		{
			"complex_nested_collections",
			"[{x: 1, y: 2}, {x: 3, y: 4}]",
			"expr/complex_nested_collections",
		},
		{
			"complex_lambda_call",
			"(\\x. x * 2)(21)",
			"expr/complex_lambda_call",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}
