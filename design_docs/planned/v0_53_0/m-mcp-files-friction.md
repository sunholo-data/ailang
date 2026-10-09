# M-MCP-FILES-FRICTION — routing and decisions for the sunholo/mcp_files 0.1.0 friction grab-bag (Refs #1618)

**Status**: Planned
**Target**: v0.53.0
**Priority**: P2 (issue label `priority:P2`)
**Estimated**: 0.5–1 day per slice, ≈4–5 days total across five independently shippable slices (item 3 shipped directly with this doc — see below)
**Dependencies**: none blocking; items 4 and 5 extend implemented docs (M-IFC-DECLARED-RECORD-LABELS v0.52.0, M-SMT-INTERP-SHOW v0.36.0)
**Refs**: #1618 (this doc does not open a new issue — it splits the grab-bag and routes each item)
**Triage**: `design_docs/planned/ailang-core-triage/mcp-files-0.1.0-friction-batch.md` (task-88b82b4d, PR #1619)
**Source feature**: `design_docs/planned/v0_53_0/m-mcp-file-handoff.md` (M-MCP-FILE-HANDOFF F2, the feature whose dogfood run produced the report)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | All fixes are deterministic functions of source (stripper spans, fmt envelope, originpolicy); no new nondeterminism |
| A2: Replayability | 0 | No trace/runtime behavior changes except CORS admission (server config, not language semantics) |
| A3: Effect Legibility | 0 | No effect-surface changes |
| A4: Type Safety | +1 | Item 4 closes a silent hole in the IFC checker (a refinement that parses but is dropped) |
| A5: Purity | 0 | No change |
| A6: Explicit Capability | +1 | Item 6 keeps the origin allowlist an explicit, validated grant instead of the only working option being `--cors` (allow all) |
| A7: Machine Decidability | +1 | Items 1/2/5 remove parse/verify failures that currently force agents into generator+drift-check workarounds |
| A8: Semantic Transparency | +1 | Item 3 fixes a prompt that taught the wrong timestamp units; item 4 stops silent dropping of written refinements |
| A9: Composability | 0 | No change |
| A10: Simplicity | 0 | Slices reuse existing machinery (envelope string spans, originpolicy, ShowNormalizer) rather than new subsystems |
| A11: Observability | 0 | Existing skip/error reporting surfaces are reused |
| A12: Performance | 0 | Stripper/fmt changes are one-pass; no hot-path risk |

**Net: +5. No hard violations (A1/A3/A4/A7 all ≥ 0).**

## Problem Statement

While building `sunholo/mcp_files` 0.1.0 (M-MCP-FILE-HANDOFF F2), the package hit six AILANG toolchain
frictions and worked around each one (issue #1618). The triage batch doc (PR #1619) recommended a short
batch design doc that **routes** rather than re-files: four of six items already have triage rows or
implemented-doc frames, and the genuinely new decisions are small.

This doc re-verifies every item live at HEAD and gives each one a decision or a pointer, per the task
directive: *"Re-verify each item and give each live one a decision or a pointer to an existing doc."*

**Re-verification environment** (recorded for reproducibility):
- Workspace: dev HEAD `1dfd5615` (PR #1619 merged; only doc commits after the binary's commit)
- Behavior verified with the prebuilt `ailang` v0.52.5, commit `7200786`, built 2026-10-07
  (`ailang version`; no Go toolchain is installed in this session, so mechanisms were additionally
  verified by reading HEAD source — every mechanism claim below cites the file and line read)
- Prior triage claim re-checked: the directive noted **item 1 "did not reproduce"** on 2026-10-08 —
  that non-repro is **contradicted** here with a 9-line reproducer (see item 1; the boundary between
  balanced and unbalanced braces explains the divergent result)

**Current State (live verification summary):**

| # | Item (from #1618) | Live at HEAD? | Evidence | Routing |
|---|---|---|---|---|
| 1 | `ailang test` extraction cuts test body at unbalanced `{`/`}` in a string literal | **YES — reproduces** (contradicts the 2026-10-08 non-repro) | `ailang test` → 0 passed, 2 failed | Fix; extends triage row `source-strip-string-brace-skip-ranges.md` |
| 2 | `ailang fmt` flattens multi-line string literals to `\n`-escapes | **YES** | `ailang fmt --write` rewrites the file | Decision + contract amendment; extends `m-ailang-fmt.md` rule 6 |
| 3 | `ailang prompt` says datetime/clock are Unix seconds; runtime is milliseconds | **YES** | `now()` = 1791482846844 (13 digits = ms); `makeDate(2026,10,8)` = 1791417600000 | **Shipped directly** with this doc (prompt text fix, authorized by the task directive) |
| 4 | `{not secret}` refinement on a record field's function type silently ignored | **YES — and broader than reported** (plain fields too) | `ailang check` → `✓ No errors found!` on all three shapes | Decision; extends `m-ifc-declared-record-labels.md` |
| 5 | `"${x}"` in `ensures` unencodable; `std/string.concat` unencodable | **YES — both halves** | `ailang verify` → encode error; → SKIPPED | Interpolation half: pointer to triage row (already scoped); concat half: decision here |
| 6 | serve-api CORS is global + exact-origin only; MCP App widget origins can't be expressed | **YES** | `--cors-origin "https://*.claudemcpcontent.com"` accepted at startup, then 403 on preflight and POST | Feature decision; extends `originpolicy` |

**Impact:** every item forced the package into a workaround (test strings moved into functions, an
assets/ generator plus drift check, wrapper functions plus a lint rule, contract rewrites). These are
exactly the "codegen-beats-tooling" retreats the harness is supposed to prevent, and items 1 and 6
defeat a security-relevant gate (test isolation and origin admission respectively).

## Goals

**Primary Goal:** give each of the six #1618 items a verified live status and a routing decision —
direct-fix, follow-up-doc pointer, or shipped-here — so no item needs re-triage.

**Success Metrics:**
- The item 1 reproducer (`/tmp` fixture below) passes: 2 tests, 2 passed after the fix
- `ailang fmt` round-trips a multi-line string literal with its layout intact (per the frozen decision)
- `ailang prompt` (and its four derived surfaces) states millisecond units — shipped in this PR
- A secret reaching a `{not secret}`-refined record field (plain or function-typed) is either rejected or the annotation is rejected — never silent
- `ailang verify` accepts `"${x}"` inside `ensures` (normalization, per the triage row's M5 scope)
- A declared `*.claudemcpcontent.com` origin pattern admits widget preflights/POSTs on the declared routes, and wildcard-shaped exact origins are refused at startup

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Stripper fix shape — drive `testAndPropertySkipRanges` from the lexer's string-aware scan rather than raw rune counting (not AST spans) | Decides whether lexical knowledge is duplicated in a third place ( stripper vs comment_scan) | agent | design | med |
| D2: fmt contract amendment — preserve as-written layout of string literals that were multi-line in source (envelope-anchored, like comments), instead of always flattening | Amends documented rule 6 of the implemented formatter contract (`m-ailang-fmt.md`); changes what `--check` accepts | human | design | high |
| D3: IFC — enforce `{not ℓ}` refinements inside field types now vs fail-loud-reject unenforceable positions first | Security semantics of the checker; scope found broader than the issue (plain fields too) | human | design | high |
| D4: `_str_join` in contracts — encode a list-fold of `str.++` for fixed-shape lists vs keep honest SKIPPED and document interpolation as the encodable form | Adds SMT surface vs teaches around it; the prompt already prefers `"${...}"` | human | design | med |
| D5: CORS — add host-suffix origin matching to the shared `originpolicy` (+ reject wildcard-shaped exact origins at validation); per-route scoping deferred | Public CLI/server surface; shared with `ailang server`; smallest shape that satisfies the documented Claude requirement | agent | design | med |
| D6: Prompt unit fix ships directly in this PR (amend unfrozen head v0.16.6 in place per registry discipline) | Text-only, no semantics; coordinator directive says prompt text fixes ship directly | agent | shipped | low |

### Design Freeze

Before implementation begins (slices 1, 2, 4, 5), these must be resolved:

- [x] D2 ratified: does fmt preserve multi-line string layout as-written? (Recommended: yes — see Solution Design; the alternative is documenting flatten as canonical and pointing embedded sources at assets/) — **Ruled 2026-10-08 by Mark: the recommended option, as written.**
- [x] D3 ratified: enforce vs reject-first for `{not ℓ}` inside field types (recommended: enforce for plain-field writes and fn-domain call-through; reject with an error where enforcement is not yet expressible) — **Ruled 2026-10-08 by Mark: the recommended option, as written.**
- [x] D4 ratified: encode `_str_join` vs document-around — **Ruled 2026-10-08 by Mark: the recommended option, as written.**
- [ ] D1/D5 are agent-decidable from the frames below; no human input required beyond this doc's approval

## Solution Design

### Overview

Six items, four implementable slices, one direct fix shipped here, one already-scoped pointer. The
splits are independent; each can ship as its own PR/changelog entry. Nothing here needs a new
subsystem — every slice reuses machinery that already exists at HEAD (verified below).

### Item 3 — prompt timestamp units (SHIPPED IN THIS PR)

**Verified live (before the fix):**
- `prompts/v0.16.6.md:988` — "`now() -> int ! {Clock}` - Current Unix timestamp (seconds since epoch)" — FALSE: `internal/builtins/clock.go:64` uses `time.Now().UnixMilli()`; `std/clock.ail:18` `now() = _clock_now()`. Live: `now()` printed `1791482846844` — 13 digits, milliseconds (a seconds value would be 10 digits, `~1.79e9`).
- `prompts/v0.16.6.md:1002` — "**std/datetime** — pure date arithmetic on Unix timestamps (int seconds)" — FALSE: `std/datetime.ail:5` "All timestamps are Unix milliseconds in UTC". Live: `makeDate(2026,10,8)` = `1791417600000` (ms), and `year()` round-trips on that scale. The reporter's "expiresAt printing as 1970" is exactly a seconds-scale value read by ms-scale `year()`.

**The fix (this PR, per the task directive "Prompt text fixes can ship directly"):** amend the
unfrozen head v0.16.6 **in place** — this is the registry's documented amendment discipline (v0.16.6
carries no banked corpus evidence, `ailang prompt freeze v0.16.6` refuses; see `prompts/versions.json`
notes for the M-STD-BASE64URL-ENCODE and later in-place precedents):
- `prompts/v0.16.6.md:988` → "(milliseconds since epoch)"
- `prompts/v0.16.6.md:1002` → "(int milliseconds)"
- Same two lines in the derived surfaces kept byte-consistent: `cmd/ailang/prompts/v0.16.6.md`
  (embedded copy, must stay identical — verified `sha256sum` equal pre-fix),
  `docs/docs/prompts/current.md` (Docusaurus active copy), and the three `llms.txt` bundles
  (`llms.txt`, `docs/llms.txt`, `docs/static/llms.txt`; `jq` is unavailable in this session so the
  generators were reproduced by editing the identical lines — regenerating via `make docs` at merge
  time is the belt-and-braces check)
- `prompts/versions.json` + `cmd/ailang/prompts/versions.json`: hash updated to the new sha256 of
  `prompts/v0.16.6.md` and an amendment note appended to the v0.16.6 entry (both registries must
  agree — `check-prompt-freeze` compares them)
- Historical prompt copies (v0.16.1–v0.16.5 in `docs/docs/prompts/`) are frozen archives and stay untouched
- Propagation note: the running v0.52.5 binary serves its compile-time embedded copy, so `ailang
  prompt` picks the fix up on the next `make build`; the deployed MCP prompt host
  (mcp.ailang.sunholo.com, the `--source auto` first choice) is version-pinned separately and needs
  its redeploy — recorded here so the propagation is not assumed done when only the repo changed

**Why direct:** text-only, no language semantics, verified against source + live behavior before
editing, and the failure class (prompt teaching wrong units) is the measured "eval model punished for
believing us" pattern the dialect-alignment amendment (v0.16.4) already established as ship-direct.

### Item 1 — named-test stripper vs braces in string literals (slice 1)

**Live reproduction (contradicts the 2026-10-08 non-repro):**

```ailang
module brace/main
import std/string (length)
pure func helper() -> int = 42
test "has unbalanced brace in string" {
  let s = "click { here";
  assert length(s) > 0
}
test "sibling passes" {
  assert helper() == 42
}
```
`ailang test` → `2 tests: 0 passed, 2 failed` — the first test fails with
`PAR017 at …:7:25: ';' is only valid inside { } block bodies` and `PAR_NO_PREFIX_PARSE … :8:3: unexpected
token in expression: assert`, and the **sibling fails too** ("Every test in the module then fails", as #1618
reported). Boundary characterized live: **balanced** braces in strings (`"click { here } ok"`) pass
2/2 — a balanced pair returns depth to 0 within the line, which is the likely source of the triage's
"did not reproduce" result. The defect needs an *unbalanced* brace.

**Mechanism (read at HEAD):** `testAndPropertySkipRanges`
(`internal/testing/source_strip.go:77-107`) scans raw runes of each source line counting `{`/`}` with
no string-literal or comment awareness. An unbalanced `{` inside a literal keeps depth ≠ 0, so the
skip range collapses and later test-body lines leak into the stripped base every sibling test is
compiled against → module-wide `PAR_NO_PREFIX_PARSE`/`PAR017`. Function decls are immune because
`functionSkipRanges` uses AST spans (`f.Span.End.Line`, `internal/testing/source_strip.go:52-56`) —
which is exactly why the reporter's workaround (move such strings into functions) works.

**Decision (D1):** fix the scan, and fix it once — reuse the lexer's existing string/comment-skipping
machinery (`internal/lexer/comment_scan.go` already implements literal-skip subroutines with 2–3-char
lookahead for `--`/`//`/`"""` boundaries) so the stripper stops duplicating lexical knowledge a third
time. **Not** AST spans: `ast.TestDecl`/`ast.PropertyDecl` carry only `Pos`, no `End` span
(`internal/ast/ast_decl.go:145-149, 156-160`), so the span route means a parser/AST surface change for
a stripper-local defect. Extends (and closes) the triage row
`design_docs/planned/ailang-core-triage/source-strip-string-brace-skip-ranges.md`, whose three options
this decides between.

**Estimate:** ~20–40 LOC in `internal/testing/source_strip.go` + tests (same as the triage row).

### Item 2 — fmt flattens multi-line string literals (slice 2)

**Live reproduction:**
```ailang
export pure func page() -> string = "<html>
  <body>
    <p>hello</p>
  </body>
</html>"
```
`ailang fmt --write` rewrites it to `= "<html>\n  <body>\n    <p>hello</p>\n  </body>\n</html>"` —
semantically identical, layout destroyed; round-trip then reports canonical (`--check` passes).

**Mechanism (read at HEAD):** `escapeString` (`internal/format/literal.go:21-46`) maps every decoded
`\n` to the two-character escape — there is no multi-line emission mode; the AST stores the decoded
payload only, so the original layout is gone by the time the printer runs. This is **documented,
deliberate** behavior — rule 6 of `design_docs/implemented/v0_30_0/m-ailang-fmt.md` ("All literals are
escaped canonically… one escaping routine… never reinterpret quasiquote template contents"). So this
is not a bug to silently fix; it is a **contract decision** (D2).

**Decision (D2, recommended):** amend rule 6 — a string literal that is multi-line **in the source**
keeps its interior layout verbatim, exactly like comments (already preserved losslessly via the
token-anchored envelope) and exactly like quasiquote templates (the printer already emits
`n.Template` verbatim, `internal/format/expr.go:525-533`). Feasibility is already in place: the
envelope computes the full byte span of every string literal, including interpolation holes
(`internal/format/envelope.go:145-172`, `inStringSpan`/`stringSpans`). Enforcement:
- Preserve only when the original token bytes contain a raw newline AND re-lexing the preserved
  bytes decodes to the same payload as the AST value — otherwise refuse/canonicalize (fail-closed,
  like the comment-attachment refusals)
- Idempotence is preserved (a preserved literal re-formats to itself; an escaped one-liner stays a
  one-liner), so `--check` and the drift gates keep working; what changes is that two layouts of the
  same payload can both be canonical — the same relaxation comments already made
- The alternative (keep flatten as canonical; embedded HTML/JS/CSS lives in `assets/` + generator,
  as mcp_files does today) is the documented status quo and remains valid if D2 is not ratified

Also recorded for the future: the `ast.QuasiQuote` scaffolding (kinds sql/html/shell/url/json/regex
per `internal/lexer/lexer.go:391-397`) has a verbatim printer but **no parser production** — grep for
`QuasiQuote|parseQuasi` in `internal/parser/` is empty, and live `html"""…"""` fails with
`PAR_NO_PREFIX_PARSE`. Completing it is a separate language-surface change and is **not** proposed
here; the as-written-preservation rule makes it unnecessary for this report.

**Estimate:** ~40–60 LOC in `internal/format/` (literal.go + envelope plumbing) + corpus tests.

### Item 4 — `{not secret}` refinements inside record field types silently dropped (slice 3)

**Live reproduction (three shapes, all silent):**
- Function-typed field, domain refinement: `type Handler = { run: (string{not secret}) -> string }`,
  then `h.run(secret("ref","purpose"))` → `✓ No errors found!` — the #1618 shape
- Plain field: `type Box = { payload: string{not secret} }`, then
  `let b = { payload: secret("ref","purpose") }` → `✓ No errors found!` — **broader than the issue**
- Function-typed field, codomain refinement: `type Maker = { mk: () -> string{not secret} }` → silent

**Contrast (live, the boundary that proves the drop):** the same refinement on a top-level function
parameter IS enforced — `pure func logLine(s: string{not secret}) -> string` called with
`secret("ref","purpose")` fails: `information-flow violation: value labelled <secret> reaches
parameter "s" which forbids it (string{not secret})` with the declassify suggestion.

**Mechanism (read at HEAD):** `walkType` (`internal/types/ifc_static_type.go:100-131`) reports only
source labels — the `LabelledType` case reads `tt.Label` and never `tt.Refinement`
(`internal/ast/ast_type.go:160`: `Refinement *RefinementExpr // set for T{not IDENT} syntax`); and the
`FuncType` case walks **only `tt.Return`** (lines 129-130), so function domains are invisible to the
declared-label machinery in both directions. Sink refinements are read only from top-level function
signatures (`buildIFCSig`, `internal/types/ifc_check.go:128-129`). This is the deliberate residue of
M-IFC-DECLARED-RECORD-LABELS (`design_docs/planned/v0_52_0/m-ifc-declared-record-labels.md:112`:
"function type → deep(return) only") — that doc fixed **source** labels (`<ℓ>`); **sink** refinements
(`{not ℓ}`) written inside field types were out of its scope and remain unread anywhere.

**Decision (D3, recommended):** enforce `{not ℓ}` inside field types at the two places the checker
already controls, and fail loudly where it cannot yet enforce:
- Plain-field writes: a `<ℓ>`-labelled value flowing into a `{not ℓ}`-refined field of a record
  literal / declared type is a violation (mirror of the enforced top-level parameter rule; the error
  kind `SinkRefinementError` already exists, `internal/types/ifc_check.go:559`)
- Function-typed field, domain refinement: enforced at **call-through** — `r.f(secretVal)` with a
  domain `T{not ℓ}` checks the argument exactly like a direct call to a declared function does
- Function-typed field, codomain refinement: **not enforceable locally** (the field's function value
  comes from outside; its body is not visible) — reject the *annotation* with an explicit error
  ("`{not ℓ}` on a function-typed record field's codomain is not enforceable here; refine the
  producer's return type instead") rather than dropping it. #1618's own words: "enforced, or rejected
  with an error, **not silently dropped**"
- Scoped as a follow-up section of/extension to M-IFC-DECLARED-RECORD-LABELS (its fixtures p1–p13,
  r1–r4 methodology carries over; add the three shapes above as fixtures 18–20)

**Estimate:** ~30–60 LOC in `internal/types/` + fixtures.

### Item 5 — contracts: interpolation in `ensures` + `std/string.concat` (slice 4)

**Live reproduction (both halves):**
- Half A (already triaged): `pure func tagged(x: string) -> string ensures { result == "id:${x}" }`
  with body `"id:${x}"` → `ailang verify` fails: `encoding error: cannot encode ensures clause: let
  value: unsupported application: $builtin.show([x])`. Control: the **identical interpolation in the
  body** with `ensures { true }` → `✓ VERIFIED` in 298 ms. Same repro shape as the triage row.
- Half B (new scope): `ensures { result == concat(["id:", x]) }` (body identical) → `⚠ SKIPPED …
  Reason: Function "tagged" uses an unencodable builtin: std/string.concat / Hint: Z3 has no
  SMT-LIB encoding for std/string.concat`.

**Mechanism (read at HEAD):** `std/string.ail:112` — `concat(xs) = _str_join(xs, "")`; grep for
`_str_join|str_join` across `internal/smt/` is **empty** — only `OpConcat` is encodable
(`internal/smt/codegen_operators.go:19`: `core.OpConcat: "str.++"`; `internal/smt/encodable.go:557`).
So `concat` has no SMT-LIB encoding, and functions using it are skipped (honest, with an explicit
reason — fail-loud, not silent). Half A's mechanism (ShowNormalizer walks body exprs only; contract
clauses live on `core.Func.Contracts` and reach the encoder raw) is verified in the triage row
`show-in-ensures-clause-unencodable.md` and re-confirmed live above.

**Decision:**
- **Half A — pointer, no new design:** route the sprint exactly as the triage row scopes it: run the
  existing M-SMT-INTERP-SHOW M1 normalization over `Contract.Expr`s of each decl
  (`internal/pipeline/show_normalize.go`), extend the residue hint in
  `internal/smt/show_rejection.go` with the interpolation origin. That row already carries the
  repro, the mechanism, and the LOC estimate (~20–40). This doc adds nothing and re-files nothing.
- **Half B — D4, recommended: keep the honest SKIPPED and document interpolation as the encodable
  form.** Rationale: the interpolation the prompt already teaches (`"${...}"` preferred since
  M-CONCAT-DISAMBIG) desugars to `concat_String` chains that ARE encodable as `str.++` — the control
  above proves the encodable path verifies today. Encoding `_str_join` for arbitrary lists needs a
  fold, which SMT-LIB has no direct primitive for (fixed-arity expansion or declared recursive
  definitions — a genuinely new decision with quantifier risk). Cheap complementary fix if D4 goes
  the other way: teach the SKIPPED hint to point at `"${...}"` (e.g. "rewrite concat([a,b]) as
  "${…}" — encodable today"), which is a one-line hint change consistent with M-SMT-INTERP-SHOW's
  residue-diagnostic pattern.

**Estimate:** half A ~20–40 LOC (per the triage row); half B either 0 (doc + hint line) or a new
follow-up doc if encoding is ratified.

### Item 6 — serve-api CORS: per-route / suffix origins (slice 5)

**Live reproduction:**
- Flags (read at HEAD, `cmd/ailang/serve_api.go:19-21`): only `--cors` (allow all) and
  `--cors-origin` (exact match, repeatable). No per-route form; no wildcard/suffix form.
- Startup accepts a wildcard-shaped value: `ailang serve-api --port 18099 --cors-origin
  "https://*.claudemcpcontent.com" serve/main.ail` **starts cleanly** — `Validate`
  (`internal/platform/originpolicy/originpolicy.go:46-60`) only checks
  `scheme://host[:port]` shape, and `https://*.claudemcpcontent.com` parses as scheme+host
  (`u.Host == "*.claudemcpcontent.com"`) so it passes.
- Runtime then refuses everything it was meant to admit: `OPTIONS` preflight and `POST /uploads`
  from `https://abc123def456abc123def456abc123de.claudemcpcontent.com` both return **403 Forbidden**
  (exact-match allowlist contains a string no real Origin header will ever equal). The MCP App
  widget origin is `<first 32 hex of sha256(connector URL)>.claudemcpcontent.com` — unknowable before
  deployment, so the allowlist can never list it. Claude's own docs instruct developers to
  "allow requests whose `Origin` matches `*.claudemcpcontent.com`" (quoted with sources in
  `design_docs/planned/v0_51_0/m-serveapi-directory-ready-file-handoff-research.md`, MCP Apps row).
  Today the only working expression of that instruction is `--cors` — allow **every** origin on
  **every** route.

**Mechanism (read at HEAD):** one immutable global `Policy` per server
(`internal/apiserver/server.go:66,232` → `originpolicy.New(cfg.CORS, cfg.CORSOrigins, …)`), applied
uniformly by `corsWrap` to every wrapped route (`internal/apiserver/routes.go:462`); there is no
route-keyed origin state anywhere (grep for per-route origin config is empty). The policy itself
lives in `internal/platform/originpolicy`, shared with `ailang server` (M-SERVER-ORIGIN-POLICY), so a
suffix form benefits both listeners.

**Decision (D5):**
1. **Suffix origins in the shared policy** — accept an explicit suffix form, e.g.
   `--cors-origin-suffix claudemcpcontent.com` (repeatable) or `--cors-origin
   https://*.claudemcpcontent.com` with `*` recognized as exactly one leading host label:
   - host-suffix match at a **dot boundary** (`abc.claudemcpcontent.com` matches;
     `evilclaudemcpcontent.com` does NOT), scheme still pinned http/https
   - `Listed`-equivalent admission + echo of the concrete request Origin in
     `Access-Control-Allow-Origin` (a wildcard value is not sent — echoing the concrete origin is
     what the Claude guidance asks for)
   - **Validation (fail-loud):** a wildcard-shaped `--cors-origin` value that is not in the
     recognized suffix form must be **rejected at startup** — today it is silently dead config (the
     live repro above). No silent fallbacks (CLAUDE.md §2)
2. **Per-route scoping (deferred, not this slice):** a route-keyed allowlist (`--route-origin
   POST /uploads=…`) is a coherent later extension once the suffix form lands; it needs config
   plumbing through `ServerConfig` and per-route wrap lookup. The suffix form alone satisfies the
   reported widget case on the declared `POST /uploads` route.
3. Tests: extend the originpolicy table tests with suffix admits/dot-boundary-refusals, and a
   serve-api integration test replaying the live repro (startup → widget-origin preflight → POST
   200/403 matrix).

**Estimate:** ~60–100 LOC across `internal/platform/originpolicy` (+tests) and
`cmd/ailang/serve_api.go` flag surface; `internal/apiserver/cors.go` unchanged (mode pick only).

### Implementation Plan

**Phase 0: shipped with this doc** (~0.5 hours, done)
- [x] Item 3 prompt text fix across source, embedded, docs-sync and llms.txt surfaces + registry hash/note (D6)

**Phase 1: stripper (item 1)** (~0.5 day)
- [ ] Reuse the lexer literal-skip scan in `testAndPropertySkipRanges`; keep behavior for balanced cases identical
- [ ] Regression test: the 9-line reproducer above (unbalanced `{` and `}` variants; comment-bearing bodies)

**Phase 2: fmt as-written multi-line strings (item 2)** (~1 day, after D2 ratification)
- [ ] Envelope-anchored preservation for literals whose original bytes contain a raw newline, with re-lex validation and fail-closed refusal
- [ ] Corpus + roundtrip tests (`internal/format/roundtrip_soundness_test.go` family)

**Phase 3: IFC field-type refinements (item 4)** (~1 day, after D3 ratification)
- [ ] Enforce plain-field writes and fn-domain call-through; reject codomain-position refinements with an explicit error
- [ ] Fixtures 18–20 (the three live shapes) added to the M-IFC fixture family

**Phase 4: contracts (item 5)** (~0.5 day + optional follow-up)
- [ ] Half A per the triage row: normalize `Contract.Expr`s, residue hint
- [ ] Half B per D4: SKIPPED hint points at `"${…}"`, or a follow-up doc if encoding is ratified

**Phase 5: CORS suffix origins (item 6)** (~0.5–1 day)
- [ ] Suffix form in `originpolicy` + startup rejection of dead wildcard-shaped exact origins
- [ ] Integration test replaying the live 403 repro

### Files to Modify/Create

**Modified:**
- `prompts/v0.16.6.md`, `cmd/ailang/prompts/v0.16.6.md`, `docs/docs/prompts/current.md`, `llms.txt`, `docs/llms.txt`, `docs/static/llms.txt` — two lines each (shipped)
- `prompts/versions.json`, `cmd/ailang/prompts/versions.json` — hash + amendment note (shipped)
- `internal/testing/source_strip.go` — string-aware skip ranges (~20–40 LOC)
- `internal/format/literal.go` (+ envelope plumbing) — multi-line preservation (~40–60 LOC)
- `internal/types/ifc_static_type.go`, `internal/types/ifc_check.go` — field refinements (~30–60 LOC)
- `internal/pipeline/show_normalize.go`, `internal/smt/show_rejection.go` — contract normalization (per triage row)
- `internal/smt/` encodable set + hint — only if D4 ratifies encoding
- `internal/platform/originpolicy/originpolicy.go` (+tests), `cmd/ailang/serve_api.go` — suffix origins (~60–100 LOC)

## Conflict Surface Analysis

Required for the `internal/types/`, `internal/format/` (parser-adjacent printer), and `internal/testing/` changes:

1. **Positions extended:**
   - Stripper: test/property body line ranges — the scan currently sees comments and string interiors as brace sources; the fix makes only real braces count. Other constructs in those positions: block comments containing braces (currently ALSO miscounted — the fix changes their treatment too; that is in-scope and tested), nested records in test bodies (unaffected: braces still counted outside literals).
   - fmt: the emitted form of `ast.StringLit` in every expression position (patterns, contracts, annotations, interpolants) — the preservation rule is keyed on the original token, so ALL positions change coherently, not just top-level bindings.
   - IFC: `{not ℓ}` inside every field-type position — plain fields, fn domains, fn codomains, nested records, ADT constructor fields, list/tuple elements (the walk is structural; only the three fixture shapes get dedicated tests).
   - originpolicy: the origin admission predicate — every wrapped route, both `serve-api` and `ailang server`, REST and the WebSocket Origin check (`CheckWebSocket` consults the same allowlist, `internal/apiserver/routes_ws.go:243` — a suffix form must be considered there too, noted in Phase 5).
2. **Other constructs already in those positions:** balanced braces in strings (must keep passing — live control 2/2); `\n`-escaped one-liners (must stay canonical, not exploded); `<ℓ>` source labels inside field types (M-IFC behavior must not regress — its fixtures p1–p13/r1–r4 must still fail/pass respectively); exact origins and same-origin requests (admission unchanged).
3. **Disambiguation:** stripper keys on lexer literal boundaries (no new lookahead); fmt keys on original-token newline presence + re-lex equality; IFC keys on the existing `LabelledType.Label` vs `.Refinement` split (`internal/ast/ast_type.go:152-160`); originpolicy keys on `*.` prefix syntax at flag-parse time.
4. **Programs that MUST still work (fixtures, all verified to exist):**
   - `std/trace_test.ail` — named-test-shaped module shipped in-tree
   - `tests/record_update_regression_test.ail`, `tests/stdlib/list_drop_test.ail` — named tests with records and multiple tests per module
   - `examples/serveapi_ws_bridge.ail` — `@route("WS", "/live")` + serve-api surface
   - `examples/runnable/contracts/inbox_injection_v2.ail` — the M-IFC worked example (cited by m-ifc-declared-record-labels.md as its regression example)
   - M-IFC fixtures p1–p13, r1–r4 (methodology in the implemented doc; re-derive from its table)
5. **Deliberate changes (intentional incompatibilities):** startup rejection of wildcard-shaped `--cors-origin` values (today: silently dead config — the live repro); rejection of codomain-position `{not ℓ}` on function-typed record fields (today: silently dropped — #1618 explicitly asks for the error). Both are fail-loud tightening, documented as such in the changelog entries.

## Examples

### Example 1: the reporter's module, after all slices

```ailang
-- mcp_files, post-fix: strings with braces in tests, multi-line HTML, refinements, contracts
test "json template renders" {
  let s = "{\"partial\": true";        -- unbalanced brace in a literal: now fine
  assert length(s) > 0
}
export pure func page() -> string = "<html>
  <body>…</body>
</html>"                               -- fmt keeps this layout
export type Handler = { run: (string{not secret}) -> string }   -- enforced at call-through
export pure func tagged(x: string) -> string
  ensures { result == "id:${x}" }      -- verifies (normalized like the body)
```

### Example 2: MCP App widget upload, post-D5

```bash
ailang serve-api --port 8080 --cors-origin-suffix claudemcpcontent.com --cors-origin https://app.example.com svc.ail
# widget preflight from https://<hash>.claudemcpcontent.com → 204 + concrete-origin echo
# curl / server-to-server (no Origin) → unchanged
# POST from https://evil-claudemcpcontent.attacker.tld → 403 (no dot-boundary match)
```

## Success Criteria

- [ ] Item 1 reproducer passes 2/2 after the fix (today: 0/2)
- [ ] Balanced-brace and comment fixtures unchanged (live control re-run)
- [ ] `ailang fmt --write` on the multi-line example leaves layout intact; `--check` idempotent; corpus/roundtrip gates green (`make test` fmt corpus)
- [ ] `ailang prompt | grep -i "timestamp"` says milliseconds; all four derived surfaces agree; `make check-prompt-freeze` green (both registries agree)
- [ ] The three IFC shapes from the live repro are rejected (or the annotation is, per D3); M-IFC fixtures p1–p13/r1–r4 unchanged
- [ ] `"${x}"` inside `ensures` verifies (control body case stays green); concat-using functions either verify or skip with the interpolation hint
- [ ] Suffix-origin server: widget preflight 204, dot-boundary refusal 403, exact/same-origin behavior unchanged; wildcard-shaped exact `--cors-origin` refused at startup
- [ ] All tests passing (`make test`), docs updated, changelog entries per slice

## Testing Strategy

- Every slice gets the live reproducer from this doc as its regression test (they are all ≤ 12 lines and were run against HEAD here)
- Stripper: add `}`-in-string variant and brace-in-comment variant (same mechanism class)
- fmt: add to the existing corpus/roundtrip gates rather than new harnesses
- IFC: extend the fixture family of M-IFC-DECLARED-RECORD-LABELS (fixtures 18–20)
- originpolicy: table tests for suffix/dot-boundary; serve-api integration test for the 403 matrix
- Contracts: the triage row's repro shape plus the SKIPPED-hint assertion

## Verification Log (hard gate — every claim above, verified 2026-10-08)

| Claim | Verification | Result |
|---|---|---|
| Item 1 reproduces (contradicts 2026-10-08 non-repro) | `ailang test` on the 9-line module above (binary v0.52.5) | `2 tests: 0 passed, 2 failed`; errors `PAR017`, `PAR_NO_PREFIX_PARSE` |
| Item 1 boundary: balanced braces fine | same module with `"click { here } ok"` | `2 passed, 0 failed` |
| Stripper is string-blind | read `internal/testing/source_strip.go:77-107` | rune scan `{`/`}`, no literal/comment awareness (code read) |
| Test decls lack end spans | read `internal/ast/ast_decl.go:145-160` | `TestDecl`/`PropertyDecl` have `Pos` only — rules out the AST-span fix shape |
| Item 2: fmt flattens | `ailang fmt --write` on the multi-line example; `--check` after | file rewritten to `\n`-escapes; then canonical |
| Flatten is deliberate (contract) | read `m-ailang-fmt.md` rule 6 (`design_docs/implemented/v0_30_0/`) | "All literals are escaped canonically… one escaping routine" |
| Escape always flattens | read `internal/format/literal.go:21-46` | `'\n' → \`\\n\``; no multi-line mode |
| Envelope tracks string byte spans | read `internal/format/envelope.go:145-172` | `inStringSpan`/`stringSpans` compute full literal spans incl. holes |
| QuasiQuote printer verbatim | read `internal/format/expr.go:525-533` | emits `n.Template` verbatim |
| **No parser path produces QuasiQuote** (negative) | `grep -rn "QuasiQuote\|parseQuasi" internal/parser/` → empty; live `html"""…"""` | `PAR_NO_PREFIX_PARSE` — scaffolding only, not a user escape hatch |
| Item 3: now() is milliseconds | `ailang run` live; read `internal/builtins/clock.go:64`, `std/clock.ail:18` | `1791482846844` (13 digits); `time.Now().UnixMilli()` |
| Item 3: datetime is milliseconds | read `std/datetime.ail:5`; live `makeDate(2026,10,8)` = `1791417600000` | "All timestamps are Unix milliseconds in UTC" |
| Item 3: prompt claims seconds | read `prompts/v0.16.6.md:988,1002`; `ailang prompt \| grep` | "(seconds since epoch)", "(int seconds)" — both fixed in this PR |
| Item 4: fn-field domain refinement silent | `ailang check` on `Handler = { run: (string{not secret}) -> string }` + secret call | `✓ No errors found!` |
| Item 4: plain-field refinement silent | `ailang check` on `Box = { payload: string{not secret} }` + secret write | `✓ No errors found!` — broader than #1618 reported |
| Item 4: codomain refinement silent | `ailang check` on `Maker = { mk: () -> string{not secret} }` | `✓ No errors found!` |
| Item 4: top-level param refinement enforced (contrast) | `ailang check` on `logLine(s: string{not secret})` + secret call | `information-flow violation … reaches parameter "s" which forbids it` |
| walkType drops refinements (negative) | read `internal/types/ifc_static_type.go:100-131` | `LabelledType` case reads `tt.Label` only; no `Refinement` case anywhere |
| FuncType walks Return only | read `internal/types/ifc_static_type.go:129-130` | domain invisible in both directions |
| Refinement AST node exists | read `internal/ast/ast_type.go:144-170` | `RefinementExpr`; `LabelledType.Refinement` set for `T{not IDENT}` |
| Sink error kind exists (no new code needed) | read `internal/types/ifc_check.go:559` | `SinkRefinementError` — reuse, do not allocate |
| Item 5A: ensures interpolation unencodable | `ailang verify` live | `cannot encode ensures clause … $builtin.show([x])` |
| Item 5A control: body interpolation verifies | `ailang verify` with `ensures { true }` | `✓ VERIFIED tagged` (298 ms) |
| Item 5B: concat skipped, honest | `ailang verify` with `concat(["id:", x])` | `⚠ SKIPPED … unencodable builtin: std/string.concat` + hint |
| concat resolves to `_str_join` | read `std/string.ail:112` | `export pure func concat(xs) = _str_join(xs, "")` |
| **No `_str_join` SMT encoding** (negative) | `grep -rn "str_join" internal/smt/` → empty; `OpConcat → "str.++"` at `codegen_operators.go:19`, `encodable.go:557` | only `OpConcat` encodable |
| Item 6: only `--cors`/`--cors-origin` flags | read `cmd/ailang/serve_api.go:19-21` | no per-route, no suffix form |
| Item 6: policy global, per-server | read `internal/apiserver/server.go:66,232`, `routes.go:462`, `cors.go:28` | one `Policy` via `corsWrap` on every route; **no route-keyed origin state** (negative, grep) |
| Item 6: wildcard-shaped value accepted at startup | live `serve-api --cors-origin "https://*.claudemcpcontent.com"` | server started cleanly (validation passes) |
| Item 6: then 403s real widget origins | live `curl` OPTIONS + POST with `Origin: https://<32hex>.claudemcpcontent.com` | both `HTTP/1.1 403 Forbidden` — dead config, silent |
| Claude requires `*.claudemcpcontent.com` origin allowance | `design_docs/planned/v0_51_0/m-serveapi-directory-ready-file-handoff-research.md` MCP Apps row (with source URLs) | "Configure your infrastructure to allow requests whose `Origin` matches `*.claudemcpcontent.com`" |
| mcp_files workaround = mcp_oauth T7 pattern (systemic) | #1618 item 4 + `m-mcp-file-handoff.md` context | wrapper-functions-plus-lint is the second package to need it |
| Prebuilt binary vs HEAD: no code drift | `ailang version` → v0.52.5 `7200786d` (2026-10-07); workspace HEAD `1dfd5615` = PR #1619 (docs only, verified via `git show --stat` file list: 1 design-doc file) | behavior reads valid for HEAD |
| create_planned_doc.sh aborts on empty search results (session friction, once) | `bash -x` run: `merge_results` → `grep` exit 1 under `set -e -o pipefail` | scaffold created manually from the script's template; fix not attempted (not twice yet) |

## Deferred Decisions

- **Per-route CORS scoping** (`--route-origin METHOD /path=origin`): coherent follow-up to D5; needs `ServerConfig` + per-route wrap plumbing. Deferred because the suffix form alone covers the reported widget case on the declared route.
- **QuasiQuote parser completion** (`html"""…"""` in expression position): would give embedded sources a dedicated verbatim home; separate language-surface change with its own Conflict Surface. Deferred; unnecessary if D2 is ratified.
- **`_str_join` SMT encoding** (if D4 keeps SKIPPED): fixed-arity `str.++` expansion for literal list arguments is a cheap future win; quantifier/recursive-def risk for general lists is the reason it is not the default.
- **WebSocket Origin check under suffix origins**: `CheckWebSocket` (`internal/apiserver/routes_ws.go:97,243`) consults the exact allowlist; the D5 slice must decide whether suffix entries apply to socket admission too — left to the slice's sprint with a default of yes (consistency).

## Non-Goals

- Re-filing items 1/3/5A as new triage rows (the triage batch doc already rules them duplicates; this doc confirms)
- Any change to `now()`/datetime **semantics** (milliseconds is the designed, documented behavior — only the prompt was wrong)
- New error-code allocations (existing kinds reused: `SinkRefinementError`; stripper/CORS reuse existing PAR/flag error surfaces)
- Opening new GitHub issues (this doc tracks #1618)

## Timeline

| Slice | Estimate | Notes |
|---|---|---|
| 0 — prompt units (item 3) | 0.5 day | **shipped with this doc** |
| 1 — stripper (item 1) | 0.5 day | no ratification needed (D1 agent-decidable) |
| 2 — fmt multi-line (item 2) | 1 day | after D2 ratification |
| 3 — IFC field refinements (item 4) | 1 day | after D3 ratification |
| 4 — contracts (item 5) | 0.5 day (+opt) | half A is fully scoped by the triage row |
| 5 — CORS suffix (item 6) | 0.5–1 day | D5 agent-decidable; flag/validation doc updates |

(Realistic total ≈ 4–5 days across slices; slices 1 and 5 can proceed immediately on approval.)

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| D2 relaxation weakens "one canonical form" (two layouts both canonical) | Same relaxation comments already made; idempotence + round-trip gates retained; refuse (not canonicalize) on re-lex mismatch |
| D3 enforcement false-positives on legitimate secret-carrying record fields | Enforcement mirrors the already-enforced top-level parameter rule; fixtures p1–p13 must not change behavior |
| Suffix origins widen admission beyond intent | Dot-boundary match only; scheme pinned; echo concrete origin; table tests for the evil-twin host |
| Stripper change shifts line maps (error-position mapping) | `stripWithLineMap` orig indices are span-based for functions; tests assert mapped positions for the repro |
| Prompt amendment drift across the four derived surfaces | Byte-identical edits + registry hash updates; `make check-prompt-freeze` + `make docs` at merge verify |

## Related Documents

**Implemented (may inform design):**
- `design_docs/implemented/v0_30_0/m-ailang-fmt.md` — formatter contract, rule 6 (item 2 amends it)
- `design_docs/implemented/v0_36_0/m-smt-interp-show.md` (+ sprint plan) — body-side show normalization; item 5A extends it to contract clauses
- `design_docs/planned/v0_52_0/m-ifc-declared-record-labels.md` (status: implemented on dev 2026-10-02) — source-label fix; item 4 extends its machinery to sink refinements
- `design_docs/implemented/v0_44_0/m-serveapi-bind-host-cors.md` + `design_docs/planned/v0_44_0/m-server-origin-policy.md` — the CORS modes and the shared origin policy item 6 extends

**Planned (check for overlap):**
- `design_docs/planned/ailang-core-triage/mcp-files-0.1.0-friction-batch.md` — the triage that recommended this doc (routing source)
- `design_docs/planned/ailang-core-triage/source-strip-string-brace-skip-ranges.md` — item 1's triage row (mechanism + options; this doc decides D1)
- `design_docs/planned/ailang-core-triage/show-in-ensures-clause-unencodable.md` — item 5A's triage row (fully scoped fix)
- `design_docs/planned/ailang-core-triage/datetime-clock-ms-vs-seconds-docs.md` — item 3's triage row (superseded by the direct fix here)
- `design_docs/planned/v0_53_0/m-mcp-file-handoff.md` — the source feature and its iteration loop
- `design_docs/planned/v0_51_0/m-serveapi-directory-ready-file-handoff-research.md` — Claude `*.claudemcpcontent.com` requirement, with sources

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Issue #1618 — the grab-bag this doc routes (do not open a new issue)
- `docs/LIMITATIONS.md` — current language limitations register

## Future Work

- QuasiQuote parser completion as the verbatim home for embedded non-AILANG sources (see Deferred Decisions)
- Per-route CORS scoping (see Deferred Decisions)
- `_str_join` SMT encoding for fixed-arity literal lists (see Deferred Decisions)
- Session friction: `create_planned_doc.sh` dies on empty search results under `set -e -o pipefail`
  (grep exit 1 in `merge_results`) — first occurrence; a `.pi/extensions` fix is authorized if it bites twice

---

**Last updated**: 2026-10-08
