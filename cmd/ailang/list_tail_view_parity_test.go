package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// #1814: the VM binds `rest` in `x :: rest` as a view of the parent list
// rather than a copy. The interpreter and the VM must agree on every entry,
// including programs that build new lists on a tail and then read the tail
// again. sumLarge recurses over 200,000 elements, which took ~25 s on the VM
// while each tail was a copy.
func TestListTailViewParity1814(t *testing.T) {
	bin := buildAilang(t)
	src := filepath.Join("cmd", "ailang", "testdata", "listtail", "views.ail")
	cases := []struct{ entry, arg, want string }{
		{"sumLarge", "200000", "19999900000"},
		{"reuse", "null", "[0, 2, 3, 4] [2, 3, 4, 9] [7, 2, 3, 4] [2, 3, 4]"},
		{"nested", "null", "[12, 34, 5]"},
	}
	for _, c := range cases {
		interp, ierr, icode := runWithStdin(t, bin, "", "run", "--quiet", "--entry", c.entry, "--args-json", c.arg, src)
		vm, verr, vcode := runWithStdin(t, bin, "", "run", "--quiet", "--bytecode", "--strict-bytecode", "--entry", c.entry, "--args-json", c.arg, src)
		if icode != 0 || vcode != 0 {
			t.Errorf("%s: exit interp=%d vm=%d\ninterp: %s\nvm: %s", c.entry, icode, vcode, ierr, verr)
			continue
		}
		if strings.TrimSpace(interp) != c.want || strings.TrimSpace(vm) != c.want {
			t.Errorf("%s: interpreter %q, VM %q, want %q", c.entry, strings.TrimSpace(interp), strings.TrimSpace(vm), c.want)
		}
	}
}
