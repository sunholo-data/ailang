package bytecode

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// StdADTTag hard-codes constructor ordinals that the compiler derives from
// declaration order. If std/option.ail or std/result.ail ever reorders its
// constructors, the bridge would hand the VM Some where it expects None. This
// reads the declarations and checks the table against them.
func TestStdADTTagMatchesDeclarationOrder(t *testing.T) {
	cases := []struct{ file, module, typ string }{
		{"option.ail", "std/option", "Option"},
		{"result.ail", "std/result", "Result"},
	}
	for _, c := range cases {
		src, err := os.ReadFile(filepath.Join("..", "..", "std", c.file))
		if err != nil {
			t.Fatal(err)
		}
		re := regexp.MustCompile(`(?m)^export type ` + c.typ + `\[[^\]]*\]\s*=\s*(.+)$`)
		m := re.FindSubmatch(src)
		if m == nil {
			t.Fatalf("%s: no `export type %s` declaration found", c.file, c.typ)
		}
		for i, alt := range strings.Split(string(m[1]), "|") {
			ctor := strings.TrimSpace(alt)
			if p := strings.IndexByte(ctor, '('); p >= 0 {
				ctor = ctor[:p]
			}
			got, ok := StdADTTag(c.module, c.typ, ctor)
			if !ok || got != i {
				t.Errorf("%s.%s: StdADTTag = (%d, %v), declaration order says %d", c.typ, ctor, got, ok, i)
			}
		}
	}
	if _, ok := StdADTTag("user/mod", "Option", "Some"); ok {
		t.Error("a user type named Option must not map to the std ordinals")
	}
}
