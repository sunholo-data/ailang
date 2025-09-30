package parser

import (
	"testing"
)

// TestModuleDeclarations tests module declaration parsing
func TestModuleDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"simple_module",
			"module Foo",
			"module/simple_module",
		},
		{
			"nested_module",
			"module Foo/Bar",
			"module/nested_module",
		},
		{
			"deep_nested_module",
			"module Foo/Bar/Baz",
			"module/deep_nested_module",
		},
		{
			"module_with_statement",
			"module Test\nlet x = 1",
			"module/module_with_statement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestImportDeclarations tests import statement parsing
func TestImportDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		// NOTE: Bare imports like "import Foo" trigger IMP012_UNSUPPORTED_NAMESPACE
		// Only selective imports with parentheses are supported
		{
			"import_with_symbols",
			"import Foo (bar, baz)",
			"module/import_with_symbols",
		},
		{
			"import_single_symbol",
			"import Foo (bar)",
			"module/import_single_symbol",
		},
		{
			"multiple_imports",
			"import Foo (bar)\nimport Baz (qux)",
			"module/multiple_imports",
		},
		{
			"import_with_code",
			"import Foo (bar)\nlet x = 1",
			"module/import_with_code",
		},
		{
			"import_multiple_symbols",
			"import Foo (a, b, c)",
			"module/import_multiple_symbols",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestModuleWithImports tests module declarations combined with imports
func TestModuleWithImports(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		golden string
	}{
		{
			"module_and_import",
			"module Foo\nimport Bar (x)",
			"module/module_and_import",
		},
		{
			"module_multiple_imports",
			"module Foo\nimport Bar (x)\nimport Baz (y)",
			"module/module_multiple_imports",
		},
		{
			"module_import_code",
			"module Foo\nimport Bar (x)\nlet x = 1",
			"module/module_import_code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := parseAndPrint(t, tt.input)
			goldenCompare(t, tt.golden, output)
		})
	}
}

// TestInvalidModuleSyntax tests error handling for invalid module syntax
func TestInvalidModuleSyntax(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"module_no_name", "module"},
		{"module_trailing_slash", "module Foo/"},
		{"module_leading_slash", "module /Foo"},
		{"module_double_slash", "module Foo//Bar"},
		{"import_no_name", "import"},
		{"import_bare", "import Foo"}, // IMP012: namespace imports not supported
		{"import_empty_parens", "import Foo ()"},
		{"import_trailing_comma", "import Foo (bar,)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := mustParseError(t, tt.input)
			if len(errs) == 0 {
				t.Errorf("Expected parse error for %q, but got none", tt.input)
			}
		})
	}
}
