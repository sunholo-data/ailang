# Type checker rejects a valid program depending on unrelated code in the file (fresh α-name collision across the imported-scheme boundary)

- **Date**: 2026-11-06
- **Class**: bug (typechecker soundness — false rejects; wrong-accept risk unquantified)
- **Recommend**: design-doc
- **Searched**: `unify|unification|tyvar|type variable` across design_docs/; `ls design_docs/planned/` and `ailang-core-triage/`; `M-TVAR-COLLISION-FIX` (code + v1-mission-log-archive.md:165 — a *prior, related* fix); `alpha-normal` (→ m-iface-digest-alpha-normalization.md, related but distinct); `is_newer` in std/sem.ail. No existing doc covers cross-module substitution-name conflation.
- **Estimate**: n/a (design-doc)

The reporter's repro is deterministic (10/10) and the three perturbations (shrink a list
literal, add an unrelated decl, note that annotations don't help) are exactly the signature
of a **name-keyed substitution conflating two distinct type variables that happen to share
an α-number** — whichever unify binds the name first wins, and the other use site is
mis-substituted. Code-read verifies every link in that chain at repo HEAD (`01baf90c`):

1. `Builder.generalizeType` (`internal/iface/builder.go:503`) has an admitted gap —
   `freeVars := []string{}` under "TODO: Implement proper free variable collection" — so
   exported monotypes whose bodies still contain free `TVar2`/row variables serialize those
   variables **un-quantified** under the *defining* module's α-numbering
   (`MarshalScheme`, `internal/types/json_scheme.go:8`, keeps `type_vars` and the raw body).
2. At the import site, `Scheme.InstantiateWithConstraints`
   (`internal/types/types_v2.go:520`) renames **only** `s.TypeVars` — empty or stale here —
   so the exporter's α-names leak verbatim into the importer's variable namespace.
3. Substitutions are keyed by variable *name* (`Substitution map[string]Type`,
   `internal/types/unification_core.go:5`; TVar2 binds by `Name`), and the importer's own
   fresh variables are also `α%d` (`internal/types/inference.go:528`), with the counter made
   persistent **within one module only** by the earlier M-TVAR-COLLISION-FIX
   (`internal/types/typechecker_core.go:72`). Nothing namespaces imported-scheme variables
   against local fresh ones — or against another imported scheme's.
4. The observed error text pins the conflation site: `failed to unify field 'ver': cannot
   unify record with unexpandable type constructor int` is raised at
   `internal/types/unification_records.go:143` — a field type that should be `int` has been
   substituted with a *record* by a binding that belonged to a different variable of the
   same name. Whether the file layout hides or exposes the collision depends only on how
   many fresh variables were allocated before the `is_newer` application, which is why a
   3-element vs 1-element list literal or an extra `pure func k2` flips the verdict, and why
   annotations can't help (the substitution happens after annotation checking).

Caveat: this environment has no Go toolchain, so the repro was not re-run here; the
verification above is a code-read of the exact error path plus the reporter's deterministic
10/10 repro at `f20e6dea2` (repo is now 2 commits past it; none of the three commits touch
`internal/types/unification*`, `internal/iface/builder.go` or scheme instantiation).

**Why design-doc, not direct-fix**: there are at least three acceptable remedies and they
are not equivalent — (a) implement the free-var collection TODO so exports quantify all
their variables (fixes the leak at the source, but changes every iface and its digest);
(b) namespace imported scheme variables at `UnmarshalScheme`/import time (localized, but
doubles the variable space and must preserve digest stability); (c) move substitution to
pointer identity rather than names (touches every unify site — the largest blast radius).
The choice interacts with `design_docs/planned/v0_57_1/m-iface-digest-alpha-normalization.md`
(#2076), which is the **same α-naming fragility with a different symptom**: that doc scopes
itself to *digest determinism* ("no wrong verdicts observed", options explicitly unchosen);
this report is the *wrong-verdict* consequence the digest doc has not observed. The two
should be decided together — if #2076's chosen option alpha-normalizes scheme bodies at
generalization (not just digests), it likely fixes this collision as a corollary; if it only
canonicalizes digests, this bug survives it. That coupling, plus iface cache-format
compatibility, is a ruling, not a two-line patch.

**Suggested next step**: a small design doc (or an explicit scope extension of the #2076 doc
before its Design Freeze) that (1) picks the namespace/normalization option for
*substitution semantics*, not just digests, (2) decides whether un-quantified free variables
in an exported scheme are an export-time error (fail loudly) or a load-time rename, and
(3) pins the reporter's repro as a golden cross-module test — its 10/10 determinism makes it
an ideal regression.

---

## Smaller items from the same batch

### 1. `std/cognition` drain/subscribeMsg and `std/dom` applyBatch/applyPatch/subscribe unusable under `ailang run` (E_COG_NO_CONTEXT)

- **Class**: bug (docs/UX gap, fails loudly — not silent)
- **Recommend**: design-doc
- **Searched**: `E_COG_NO_CONTEXT` in internal/; `NewCogContext|NewMsgContext|NewDOMContext` in cmd/, internal/

Verified: only the REPL wires the cognition contexts (`internal/repl/repl.go:174,182` —
DOM and Msg; Cog likewise); the CLI run path leaves `EffContext.Cog/DOM/Msg` nil, so
`_cog_drain` correctly fails with `E_COG_NO_CONTEXT` (`internal/effects/cog_drain.go:115`).
The failure is loud and well-coded; the gap is that the stdlib advertises these functions
without a docs warning. Two acceptable remedies (document "REPL/WASM only" in std/cognition.ail
and std/dom.ail headers + docs, vs. wiring a no-op context into `ailang run`) diverge in
observable behavior — a no-op context would silently change what CLI users see, and which
effects get context under which entrypoint is a surface decision → design-doc, though a
small one. If documented, the fix is 2 doc-comment lines per module (would be direct-fix).

### 2. `ailang docs std/sharedindex` says findByEmbedding scores by cosine; it is (cos+1)/2

- **Class**: bug (docs)
- **Recommend**: direct-fix
- **Estimate**: 1–2 lines in `std/sharedindex.ail` (the `findByEmbedding` doc comment at :59;
  optionally the `upsert`-adjacent comment at :53)

Verified: `CosineUnit` (`internal/simhash/simhash.go:114`) rescales to `(cos+1)/2` "so
embedding scores compare with Similarity's scale" — intentional, degenerate inputs score 0.
`ailang docs` renders the source doc comment, so the one-line comment fix ("cosine
similarity rescaled to [0,1]: (cos+1)/2, orthogonal vectors score 0.5") fixes the CLI docs
at the source.

### 3. `validate_namespace` reports a space raw: `invalid character:  ` (trailing space)

- **Class**: bug (cosmetic error message)
- **Recommend**: direct-fix
- **Estimate**: 1 line in `std/sem.ail:39` (`find_invalid_char`)

Verified: tab and newline are escaped (`Some("\\t")`, `Some("\\n")`) but space is returned
raw (`Some(" ")` at std/sem.ail:39), so the composed message reads `invalid character:  `
with an invisible offender. Escape it like its siblings (`Some("\\s")` or `"space"`).

### 4. `ailang run /abs/path/file.ail` fails MOD010; only a module-root-relative path works

- **Class**: bug (DX; blocks the fine-tune extractor over most of examples/)
- **Recommend**: design-doc
- **Searched**: `MOD010` in internal/; `CanonicalModuleID` in internal/loader/

Verified the mechanism: `CanonicalModuleID` (`internal/loader/loader.go:375`) only strips a
leading `/` — it does **not** make an absolute path repo-relative — so
`/workspace/x/examples/runnable/stdlib/sem_frame_replicas` canonicalizes to
`workspace/x/examples/...`, which can never equal the declared module path
`examples/runnable/stdlib/sem_frame_replicas` in `validateModulePath`
(`internal/pipeline/pipeline_module.go:210`); a relative-from-root path is the only spelling
that matches. The fix needs a ruling on what the module root *is* (cwd? nearest
ailang.toml/git root? resolve-both-and-warn?) — that choice changes which existing workflows
silently start passing, and MOD010 is a gate whose contract is already nuanced (temp-path
auto-relax, `--relax-modules`, package-manifest resolution all special-cased in
`internal/pipeline/mod010.go`). The measured impact (extractor rejecting most of examples/)
makes it high-value, but it is not a two-line fix → design-doc.
