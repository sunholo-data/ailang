package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

// #2290: a constraint re-emitted at a use of another module's function is
// TC_CONSTRAINT_001 naming that function and module; a builtin's keeps the
// plain path (the builtin is not something the caller imported).
func TestGlobalUsePath(t *testing.T) {
	imported := globalUsePath(&core.VarGlobal{Ref: core.GlobalRef{Module: "q/lib", Name: "eqq"}}, "Eq")
	if !strings.Contains(imported, TC_CONSTRAINT_001+": `eqq` from q/lib requires Eq") {
		t.Errorf("imported use: %q", imported)
	}
	builtin := globalUsePath(&core.VarGlobal{Ref: core.GlobalRef{Module: "$builtin", Name: "_list_contains"}}, "Eq")
	if !strings.HasPrefix(builtin, "use of _list_contains at ") || strings.Contains(builtin, TC_CONSTRAINT_001) {
		t.Errorf("builtin use: %q", builtin)
	}
}
