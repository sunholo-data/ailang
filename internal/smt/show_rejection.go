package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/core"
)

// M-SMT-INTERP-SHOW M3: the skip message for a `show` that survives
// normalization. Split out of encodable.go to keep that file under the 800-line
// AI-maintainability CI gate (make check-file-sizes) — no logic changes.

// encodableShowArgs lists the show argument types ShowNormalizer can turn into
// encodable Core. Kept in sync with internal/pipeline/show_normalize.go.
const encodableShowArgs = "string, bool, int"

// showAwareRejection builds the skip reason for an unencodable builtin.
//
// For `show` specifically it can do better than naming the builtin. After
// M-SMT-INTERP-SHOW's normalization pass, a surviving `show` is necessarily
// applied to a type with no SMT encoding — and the pass recorded WHICH type in
// DeclMeta.ShowResidue. This layer has no CoreTypeInfo of its own, so without
// that note it cannot know the type and must not invent one.
//
// The message states only what was measured. It does NOT say the call came from
// a "${...}" hole: the identical Core shape is produced by an explicit user
// `show(x)` call, and nothing in Core distinguishes them, so an origin claim
// would misdiagnose that case. The interpolation fact goes in the Hint instead,
// phrased as advice that is true either way — which is the part a reader needs,
// since for an interpolation there is no `show` call in the source to find.
func showAwareRejection(funcName, blocker string, meta *core.DeclMeta) SMTRejectionReason {
	if blocker == "show" {
		if argType := firstShowResidueType(meta); argType != "" {
			return SMTRejectionReason{
				Code: RejectUnencodable,
				Message: fmt.Sprintf(
					"Function %q applies show to a %s; Z3 has no string encoding for that type. Encodable show arguments: %s.",
					funcName, argType, encodableShowArgs),
				Hint: fmt.Sprintf(
					"\"${x}\" desugars to show(x), so this may come from an interpolation hole rather than a show call you wrote. Convert the %s to one of %s before interpolating it, or narrow the function's contracts.",
					argType, encodableShowArgs),
			}
		}
	}
	return SMTRejectionReason{
		Code:    RejectUnencodable,
		Message: fmt.Sprintf("Function %q uses an unencodable builtin: %s", funcName, blocker),
		Hint:    fmt.Sprintf("Z3 has no SMT-LIB encoding for %s. Either remove its use, refactor to use a supported builtin, or narrow the function's contracts.", blocker),
	}
}

// firstShowResidueType returns the argument type of the first `show` the
// normalizer left in this function, or "" if it recorded none (the pass was
// disabled, or this Core predates it).
func firstShowResidueType(meta *core.DeclMeta) string {
	if meta == nil || len(meta.ShowResidue) == 0 {
		return ""
	}
	return meta.ShowResidue[0].ArgType
}
