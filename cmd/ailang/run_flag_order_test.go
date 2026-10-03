package main

import (
	"flag"
	"testing"
)

func TestMisplacedRunFlag(t *testing.T) {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.String("args-json", "null", "")
	fs.String("entry", "main", "")
	fs.Bool("bytecode", false, "")

	cases := []struct {
		after []string
		want  string
	}{
		{[]string{"--args-json", "{}"}, "--args-json"},
		{[]string{"-entry=main"}, "-entry=main"},
		{[]string{"in.txt", "--bytecode"}, "--bytecode"},
		{[]string{"--", "--args-json", "x"}, ""}, // explicit program args
		{[]string{"--user-flag", "x"}, ""},       // not a run flag: the program's
		{[]string{"-", "in.txt"}, ""},            // stdin marker
		{nil, ""},
	}
	for _, c := range cases {
		if got := misplacedRunFlag(fs, c.after); got != c.want {
			t.Errorf("misplacedRunFlag(%q) = %q, want %q", c.after, got, c.want)
		}
	}
}
