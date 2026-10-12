package lexer

import (
	"reflect"
	"testing"
)

func TestCollectStructuralBraces(t *testing.T) {
	cases := []struct {
		name, source string
		want         []StructuralBrace
	}{
		{"unbalanced_literal", "test \"open\" {\n\"click { here\"\n}\n", []StructuralBrace{{1, true}, {3, false}}},
		{"comment_and_nested_record", "test \"close\" {\n{x: \"}\", y: '{'} -- } {\n// }\n}\n", []StructuralBrace{{1, true}, {2, true}, {2, false}, {4, false}}},
		{"multiline_interpolation", "test \"é\" {\n\"first {\n${{x: \"}\"}.x} }\"\n}\n", []StructuralBrace{{1, true}, {4, false}}},
		{"escaped_quotes", "{\"escaped \\\" {\"}\n", []StructuralBrace{{1, true}, {1, false}}},
		{"quasiquote", "{html\"\"\"{\n}\"\"\"}\n", []StructuralBrace{{1, true}, {2, false}}},
		{"no_braces", "-- {\n\"}\"", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CollectStructuralBraces([]byte(tc.source)); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("braces = %v, want %v", got, tc.want)
			}
		})
	}
}
