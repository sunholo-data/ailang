# Five friction points + serve-api CORS gap from sunholo/mcp_files 0.1.0 (M-MCP-FILE-HANDOFF F2)

- **Date**: 2026-10-07
- **Class**: bug (items 1–5) + feature (serve-api CORS)
- **Recommend**: design-doc
- **Searched**: `mcp.file.handoff|mcp_files` in design_docs/ (→ v0_53_0/m-mcp-file-handoff.md + sprint plan); prior triage rows on named tests, stripper, contracts, IFC labels, datetime units, CORS; `cors` in cmd/serveapi; `concat` in std/string.ail vs internal/smt/. Per-item coverage under "Cross-references" below.

## Why

F2 shipped (HEAD `c20f6605`, "F2 done — 0.1.0 published"), and this report is its measured
friction return trip — exactly the evidence the v0_53_0 doc's own iteration loop exists to
collect. Items 1–5 are AILANG defects; the serve-api ask is a feature. Verified cheaply
against HEAD where possible:

1. **Named-test stripper vs `{`/`}` in string literals** — prompt is current (`ailang
   prompt` header), and the mechanism was already reproduced locally by a prior triage:
   `testAndPropertySkipRanges` (`internal/testing/source_strip.go`) scans raw runes for
   braces with no string/comment awareness, so an unbalanced brace in a literal keeps depth
   ≠ 0 and body lines leak into the stripped base for every sibling test → module-wide
   `PAR_NO_PREFIX_PARSE`. **Already triaged: `ailang-core-triage/source-strip-string-brace-skip-ranges.md` (design-doc)**, mechanism refined in `ai-deadline-and-named-test-roundtrip.md` (defect 2a).
2. **`ailang fmt` flattens multi-line string literals to `\n`-escapes** — no existing doc
   found (`fmt|string` across triage rows; none match). Real gap, and the workaround
   (assets/ + generator + drift check) is the second time codegen-beats-tooling has been
   chosen. Surface decision (formatter fidelity contract vs preserve-as-written; row 3).
3. **Prompt says Unix SECONDS, runtime is MILLISECONDS** — re-confirmed on HEAD just now:
   `prompts/v0.16.6.md:988` "(seconds since epoch)", `:1002` "int seconds", while
   `std/datetime.ail:5` says "All timestamps are Unix milliseconds" and
   `internal/builtins/clock.go` (`_clock_now`) uses `UnixMilli()`. **Already triaged:
   `ailang-core-triage/datetime-clock-ms-vs-seconds-docs.md` (design-doc)** — a golden test
   catching it is new evidence, not new scope.
4. **`{not secret}` refinement on a record field's function type silently ignored** —
   adjacent-but-distinct from #1523 coverage: `m-ifc-declared-record-labels.md` (v0_52_0)
   fixes label *sources* for declared record fields but explicitly treats function types as
   `deep(return)` only — refinements inside a function-typed field's domain/codomain are
   still silently dropped. Also distinct from `ifc-labels-lost-across-module-boundary.md`
   (module edge) and `ifc-declared-record-field-label-loss.md` (plain fields, and it names
   #1523's field-loss scope as already being fixed). Enforce vs reject-with-error is a
   semantics decision (rows 3/4).
5. **Contract clause: `"${x}"` interpolation and `std/string.concat` unencodable** —
   the interpolation half is **already triaged, with the exact same repro shape:
   `ailang-core-triage/show-in-ensures-clause-unencodable.md` (design-doc)** — desugar
   inserts `$builtin.show`, `ShowNormalizer` (`internal/pipeline/show_normalize.go`) walks
   only body exprs, contract clauses live on `core.Func.Contracts` and reach the encoder
   raw. The `concat` half is new: `std/string.ail` concat resolves to `_str_join` (a
   builtin, not `++`), and `internal/smt/` has no `str_join` encoding — so it is a second
   unencodable builtin on the contract path, naturally in scope for the same M5 follow-up
   doc that triage proposes for M-SMT-INTERP-SHOW.
6. **serve-api CORS: global, exact-origin only** — confirmed in code: `cmd/ailang/serve_api.go:19-21`
   offers only `--cors` (allow-all) and `--cors-origin` (exact match, repeatable);
   `apiserver.ValidateCORSConfig` accepts no wildcard/suffix form and there is no per-route
   config. The ask (per-route CORS and/or suffix matching for MCP App widget origins like
   `*.claudemcpcontent.com`) has at least three acceptable shapes (suffix ACL vs per-route
   origins vs both), changes a public CLI/server surface, and spans flag parsing, config
   validation, and middleware — rows 3/4/5.

## Cross-references (what is a duplicate, what is not)

- Item 1 → duplicate-of `ailang-core-triage/source-strip-string-brace-skip-ranges.md` (the
  report's severity detail — *every* test in the module failing — matches the mechanism).
- Item 3 → duplicate-of `ailang-core-triage/datetime-clock-ms-vs-seconds-docs.md`.
- Item 5's interpolation half → duplicate-of
  `ailang-core-triage/show-in-ensures-clause-unencodable.md`; the `str_join`/concat half
  belongs in the same M-SMT-INTERP-SHOW follow-up.
- Item 4 is NOT the #1523 doc's scope (function-typed field interiors), and the workaround
  the reporter shipped mirrors `m-mcp-oauth-package.md` T7's wrapper-function pattern —
  evidence the systemic fix is overdue.
- Item 2 and item 6 have no existing coverage — terms searched listed in Searched.

## Why one batch design doc, not six files

Four of six items already have triage rows; writing new per-item files would duplicate
them. The genuinely new subjects (2, 4, 5-concat, 6) each independently clear the
design-doc bar, and items 1/3 are covered. Recommendation: a short batch doc (working
title M-MCP-FILE-FRICTION or similar, under v0_53_0 or as a follow-up set) that (a) treats
items 1/3/5-interpolation as confirmations of the existing rows and routes their sprints,
(b) scopes items 2, 4, and 5-concat as small decisions within the existing docs' frames
(formatter string-literal fidelity; IFC function-type interiors as an extension of
M-IFC-DECLARED-RECORD-LABELS; concat encoding inside the show-normalization M5), and (c)
scopes the serve-api CORS ask as its own section (it is a feature, not a bug, and touches
`cmd/ailang/serve_api.go`, `internal/apiserver` config/middleware). If the maintainer
prefers, items 2 and 6 can split into standalone docs — but do not re-file 1/3/5.

TRIAGE_FILE: design_docs/planned/ailang-core-triage/mcp-files-0.1.0-friction-batch.md
RECOMMEND: design-doc
