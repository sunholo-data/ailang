package types

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// TC_CONSTRAINT_001 codes a use of an imported function at a type that lacks
// a class instance the function's scheme requires (#2290): `add[a](x: a, y: a)
// -> a = x + y` exports `Num[a] => (a, a) -> a`, and `add("s", "t")` in another
// module is rejected at the call. Until v0.60.x the constraint was dropped at
// the module boundary, so such a call passed `check` and failed at run time (or,
// for `==` on two functions on the VM, answered false).
const TC_CONSTRAINT_001 = "TC_CONSTRAINT_001"

// globalUsePath is the diagnostic path of a class constraint that a use of the
// global v re-emits from its scheme. A function another module exports names
// that module and carries TC_CONSTRAINT_001, so the error reads as a broken
// promise of the call, not of an operator the caller never wrote. Builtins
// ($builtin) and synthesized constructors ($adt) keep the plain form.
func globalUsePath(v *core.VarGlobal, class string) string {
	if strings.HasPrefix(v.Ref.Module, "$") {
		return fmt.Sprintf("use of %s at %v", v.Ref.Name, v.Span())
	}
	return fmt.Sprintf("%v: %s: `%s` from %s requires %s for the type it is used at here",
		v.Span(), TC_CONSTRAINT_001, v.Ref.Name, v.Ref.Module, class)
}
