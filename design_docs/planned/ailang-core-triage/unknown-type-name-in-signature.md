# An undeclared type name in a signature becomes an opaque `TCon` and checks clean (#1485)

- **Date**: 2026-10-03
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `#1485`; "undeclared type", "unknown type name", "undefined type", `UNKNOWN_TYPE`, `TC_TYPE_UNKNOWN` across design_docs/ and internal/ (no hits that rule on it); `astTypeToInternalType`, `typeAliases`, `TC_TYPE_SHADOW_001` (hits: `design_docs/planned/v0_44_0/m-type-name-shadow-and-cache.md` — shadowing between *known* same-named types and its doc-only M4 "module-qualified identity"; `design_docs/implemented/v0_52_0/m-elaborator-lexical-scope.md` — value scope, not type scope). No doc rules on what happens to a type name that resolves to nothing.
- **Estimate**: omitted (design-doc)

Reproduced at origin/dev `790169359`: `export pure func f(r: Nope) -> float = 1.0` → `✓ No errors found!`. The
same holds inside a type declaration (`export type T = { x: Nope }` checks clean). The failure only surfaces
at a use site, far from the cause and in the wrong vocabulary: adding `export func main() -> float = f(3)`
reports `No instance for Num[Nope] in scope ... Nope is not numeric`, which reads as an arithmetic error.
The issue's renamed-import case (`import sim/types (Row as R)` then `f(r: Row)`) is the same defect: `Row`
is not in scope under that spelling, and nothing says so.

Mechanism: `internal/elaborate/file_funcs.go` `astTypeToInternalType` (~line 333) maps primitive names and
then, in the `default:` arm (~line 353), returns `&types.TCon{Name: typ.Name}` unconditionally — "Type
constructor (e.g., user-defined ADT)". `types.TCon` (`internal/types/types.go:57`) is a bare name with no
module and no resolution step, and the type-alias table it is later expanded against
(`Elaborator.typeAliases`, `internal/elaborate/core.go:28`; fed to the checker in
`internal/pipeline/pipeline_module_compile.go:120-140`) is a `map[string]types.Type` that simply misses. A miss
is indistinguishable from "an ADT declared elsewhere", so the opaque constructor unifies only with itself
and the program type-checks until something concrete meets it.

Why this needs a decision rather than a one-line error in the `default:` arm: the elaborator does not
currently have one authoritative "type names in scope" set at that point. Candidates that must count as
known include local `type` decls, selectively imported types (including `as` renames), builtin
constructors that are not declared in any `.ail` file (`Option`/`Result` when imported, `Json`, `Map`,
the stream/handle types std exposes), types reachable through a re-export, and type parameters written in
uppercase if any are accepted. Get the set wrong and a correct program fails; this is the same surface where
M-TYPE-NAME-SHADOW (dfa8d8b8e) just had to restrict the imported alias pull to the import closure.

Options:

1. **Scope-checked type names at elaboration (recommended).** Build the in-scope type-name set once per
   module (local decls ∪ import-closure exports under their *imported* spelling ∪ a named builtin list) and
   reject a `SimpleType` that is not in it with a new code (e.g. `TC_TYPE_UNKNOWN_001`) naming the type,
   its position, a did-you-mean over the in-scope set, and — when the name exists in a loaded but
   unimported module, or under its pre-rename name — the import that would bring it in. Applies to function
   signatures, lambda annotations and type-declaration bodies alike.
2. **Warn first, error later.** Same check, emitted as a warning for one release with a corpus sweep
   (`make verify-examples`, std, registry packages) to measure false positives before flipping it to an
   error. Cheaper to land safely; leaves the hole open for a release.
3. **Resolve `TCon` to a module-qualified identity** (M-TYPE-NAME-SHADOW M4). Fixes this and the
   collision class together, but is a much larger change to `types.TCon` and every consumer; not needed to
   close this issue.

Recommendation: option 1, gated by option 2's corpus sweep in the same PR (the sweep is the test that the
builtin list is complete). Acceptance: the issue's two repros fail at the signature with the unknown name;
the `Row as R` case suggests `R`; `type T = { x: Nope }` fails at the declaration; std and examples still
check clean. Issue: https://github.com/sunholo-data/ailang/issues/1485
