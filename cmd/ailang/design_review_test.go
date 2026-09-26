package main

import (
	"reflect"
	"testing"
)

func TestHoistFlags(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "positional before value flag",
			in:   []string{"doc.md", "--reviewer", "gpt5-6-sol", "--json"},
			want: []string{"--reviewer", "gpt5-6-sol", "--json", "doc.md"},
		},
		{
			name: "already flags first",
			in:   []string{"--reviewer", "x", "doc.md"},
			want: []string{"--reviewer", "x", "doc.md"},
		},
		{
			name: "equals form keeps single token",
			in:   []string{"doc.md", "--max-cost-usd=0.05"},
			want: []string{"--max-cost-usd=0.05", "doc.md"},
		},
		{
			name: "double dash terminates flag parsing",
			in:   []string{"--json", "--", "-weird-file.md"},
			want: []string{"--json", "-weird-file.md"},
		},
		{
			name: "--author takes a value (the doc path is not the author)",
			in:   []string{"--author", "codex:gpt-6-astra", "doc.md"},
			want: []string{"--author", "codex:gpt-6-astra", "doc.md"},
		},
		{
			name: "--author after the doc",
			in:   []string{"doc.md", "--author", "claude:claude-opus-5-5"},
			want: []string{"--author", "claude:claude-opus-5-5", "doc.md"},
		},
		{
			name: "bare dash is positional (stdin)",
			in:   []string{"-", "--reviewer", "x"},
			want: []string{"--reviewer", "x", "-"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := hoistFlags(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("hoistFlags(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}
