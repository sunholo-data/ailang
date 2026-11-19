package repl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// unreachable(msg) type-checks from the builtin type environment; at the
// prompt it must also evaluate (RT010), not fail as an undefined variable.
func TestREPLUnreachable(t *testing.T) {
	r := New()
	r.initBuiltins()
	r.importModule("std/prelude", io.Discard)
	var out bytes.Buffer
	r.ProcessExpression(`unreachable("boom")`, &out)
	if !strings.Contains(out.String(), "RT010: unreachable code reached: boom") {
		t.Fatalf(`unreachable("boom") printed %q, want RT010`, out.String())
	}
}
