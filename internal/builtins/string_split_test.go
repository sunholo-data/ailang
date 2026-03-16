package builtins

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sunholo/ailang/internal/effects/testctx"
	"github.com/sunholo/ailang/internal/eval"
)

// M-DOCPARSE-DX M3: String splitting tests

func TestStrWords(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"basic spaces", "hello world", []string{"hello", "world"}},
		{"multiple spaces", "hello  world", []string{"hello", "world"}},
		{"tabs", "hello\tworld\tfoo", []string{"hello", "world", "foo"}},
		{"mixed whitespace", "hello  world\tfoo\nbar", []string{"hello", "world", "foo", "bar"}},
		{"leading trailing", "  hello world  ", []string{"hello", "world"}},
		{"empty string", "", []string{}},
		{"only whitespace", "   \t\n  ", []string{}},
		{"single word", "hello", []string{"hello"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strWordsImpl(ctx.EffContext, []eval.Value{
				testctx.MakeString(tt.input),
			})
			assert.NoError(t, err)
			got := result.(*eval.ListValue).Elements
			assert.Equal(t, len(tt.expected), len(got), "length mismatch")
			for i, exp := range tt.expected {
				assert.Equal(t, exp, got[i].(*eval.StringValue).Value)
			}
		})
	}
}

func TestStrSplitAny(t *testing.T) {
	ctx := testctx.NewMockEffContext()

	tests := []struct {
		name       string
		input      string
		delimiters []string
		expected   []string
	}{
		{
			"comma and semicolon",
			"a,b;c",
			[]string{",", ";"},
			[]string{"a", "b", "c"},
		},
		{
			"single delimiter",
			"a,b,c",
			[]string{","},
			[]string{"a", "b", "c"},
		},
		{
			"no match",
			"abc",
			[]string{","},
			[]string{"abc"},
		},
		{
			"empty string",
			"",
			[]string{","},
			[]string{},
		},
		{
			"consecutive delimiters",
			"a,,b",
			[]string{","},
			[]string{"a", "b"},
		},
		{
			"multi-char delimiter uses all chars",
			"a:b-c",
			[]string{":-"},
			[]string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delimElems := make([]eval.Value, len(tt.delimiters))
			for i, d := range tt.delimiters {
				delimElems[i] = testctx.MakeString(d)
			}
			result, err := strSplitAnyImpl(ctx.EffContext, []eval.Value{
				testctx.MakeString(tt.input),
				&eval.ListValue{Elements: delimElems},
			})
			assert.NoError(t, err)
			got := result.(*eval.ListValue).Elements
			assert.Equal(t, len(tt.expected), len(got), "length mismatch")
			for i, exp := range tt.expected {
				assert.Equal(t, exp, got[i].(*eval.StringValue).Value)
			}
		})
	}
}
