# M-LIST-ACCESSOR-API: LC-2 — The Accessor Seam on `*eval.ListValue` + the `listrep` Ratchet Analyzer

**Status**: Planned — **quorum-cleared via the narrow-refinement carve-out at mission iteration 251** (round 1 blocked 2-of-2 → designer revision; round 2 blocked 2-of-2 → both premises measured, carve-out applied; ratified iter-98). See the two quorum logs below.
LC-2 of the quorum-cleared
[m-list-cons-cells-decomposition](m-list-cons-cells-decomposition.md) programme (roadmap line 178
is the authoritative scope contract; this doc refines it and does not re-open it).
**Target**: v0.35.0 (the programme's target release; LC-1 landed in v0.34.0)
**Priority**: P0 — every migration lane (LC-3a/b/c) and the swap (LC-4) is denominated in this
piece's analyzer count; nothing downstream can start or be acceptance-tested without it
**Estimated**: 3–4 days (the roadmap's own estimate, which already includes +1 day for the
permanent assignment rule and its seeded positive/negative fixtures). The round-2 quorum fixes
consume the Day-4 buffer — 4 days is now the expected case, not the ceiling breach (see Timeline).
**Dependencies**: LC-1 (**COMPLETE**, verdict GO — M6 report in
`design_docs/implemented/v0_34_0/`); **D-22 = `C1`** (Mark, 2026-08-22, verified L2 below).
No other code dependencies.
**Planner-Lane**: codex-ok (the design decisions are made in this doc; execution is well-specified
Go. The roadmap reserves opus-required for LC-1/LC-4 only.)

## What this doc is

The roadmap's LC-2 section (`m-list-cons-cells-decomposition.md:178-200`) names two halves, one
seam:

1. **The accessor layer** on `*eval.ListValue`, implemented over the **unchanged** slice
   representation as thin wrappers, whose exact surface "comes from LC-1's draft spec and is
   decided in this piece's own doc" — i.e. here, now, **for C1**.
2. **The `listrep` type-aware `go/analysis` tool**: the instrument that produces the TRUE
   migration count (grep cannot — roadmap N4), emits a per-package baseline, gates CI as a
   ratchet, and **permanently** rejects writes to representation fields except fresh-value
   initialization inside explicitly configured constructors.

This is the **round-2 revision** of the doc: the round-1 quorum BLOCKED it 2-of-2 (both
reviewers present), with two narrow, reviewer-authored fixes and no dispute of the design
direction. Both fixes are applied in full here; the accessor surface, D-19, and D-22 are
unchanged. See "Quorum verification log — round 1" below.

This doc does **not** migrate any consumer, does **not** change representation or behaviour
anywhere, and does **not** design LC-3 or LC-4. It contains **no AILANG snippets** — every code
block is Go — so the `ailang check` hard gate has nothing to check and nothing is asserted about
AILANG syntax.

### The two rulings that bound this design

- **D-19 = B** (Mark, 2026-08-19, roadmap N1): true cons cells / structural sharing. The
  front-slack arena is declined as the permanent answer.
- **D-22 = C1** (Mark, 2026-08-22, verified first-party at L2): **plain cons cells** —
  LC-1's candidate (i), `{head Value, tail *cell, n int}` with cached length. **Not chunked.**
  Consequence for this design: the accessor surface is sized for a plain-cons future — there is
  no chunk, no leading-chunk copy, and no chunk-boundary invariant anywhere in this API or in
  what the analyzer must protect. C1 is the shape the roadmap's 15.5–21.5 person-day scope was
  written around, so D-22 **confirms** this piece's 3–4-day estimate rather than re-basing it.

## Problem Statement

`::` copies the whole tail (`internal/builtins/list.go:98-104`, L7), so prepend-built lists are
Θ(n²) — the live user-reported OOM #676. The fix (LC-4) swaps `ListValue`'s representation, but
**903 textual `.Elements` references and 388 `ListValue{...}` constructions** exist under
`internal/` + `cmd/` at HEAD (L5) and `Elements` is a field name owned by ≥22 struct types across
9 packages (roadmap N4) — `ArrayValue`, `TupleValue`, and five AST/core/typedast families all have
one (L4, L6). Two consequences:

- **No migration can start** until consumers have something to migrate *to*: today `*ListValue`
  exposes exactly two methods, `Type()` and `String()` (L3) — no length, no indexing, no
  iteration, no construction discipline.
- **No migration can be sized or acceptance-tested by grep**: a textual count cannot distinguish
  `ListValue.Elements` from `TupleValue.Elements`, and only `ListValue` changes. Every migration
  AC in the programme is denominated in a type-checker-driven count. That instrument does not
  exist yet.

**Impact**: LC-3a/b/c (three parallel migration lanes, ~6–8 person-days) and LC-4 (the swap) are
all blocked on this piece. #676 stays open until the chain completes.

## Goals

**Primary Goal**: land the complete accessor seam and the `listrep` CI ratchet so that every
downstream lane has (a) a target API and (b) a mechanical, type-aware definition of "done", with
zero behaviour change anywhere.

**Success Metrics**:
- `*eval.ListValue` accessor surface: 2 methods → 8 methods + 3 package-level constructors,
  each with slice-equivalence tests
- The TRUE `ListValue`-typed migration count is measured and committed as a per-package baseline
  (the 903 textual count is a contaminated upper bound, L5; the analyzer's number replaces it)
- CI reds on: a seeded new `.Elements` site (including one seeded OUTSIDE `internal/`+`cmd/` —
  proving the gate LOOKS, not just fires), a seeded non-constructor field assignment, mutation
  seeded inside an accessor body, a parameter-rooted write seeded inside a constructor, a
  configured-field composite literal in a non-constructor, a neutered analyzer, and a baseline
  mismatch in either direction — all proven by fixtures
- The escape-site census (roadmap constraint: sizes LC-3a/b/c) is enumerated from analyzer
  output, not grep

## Verification Log

Rows L1–L24 first-party round 1; rows L25–L26 measured by the mission controller (opus) during
the round-1 quorum triage. All in the iteration worktree at `684ebc23e` (= `origin/dev`;
`git log -1 --format='%H %D'` → `684ebc23e7694102d8bd2f90a33c16607b589ef6 HEAD ->
sprint/iter251-list-accessor-api, origin/dev, origin/HEAD`; `git merge-base HEAD origin/dev` =
same SHA; `git status --short` clean). Scope travels with every count. Negative/empty results
carry a same-call control per the hard gate.

| # | Claim | Command | Observed |
|---|---|---|---|
| L1 | Tree state as above | `git log -1 --format='%H %D'`; `git status --short`; `git merge-base HEAD origin/dev` | as quoted; status empty; merge-base = HEAD SHA |
| L2 | **D-22 ratified as `C1`** | `gh api repos/sunholo-data/ailang/issues/745/comments --paginate --jq '.[] \| select(.created_at >= "2026-08-21") \| {body, created_at, user: .user.login}'` | final comment: `{"body":"C1 ","created_at":"2026-08-22T11:36:26Z","user":"MarkEdmondson1234"}` — preceded by the controller's consequences comment (08:42:58Z) that framed the exact C1/C2K32 choice |
| L3 | `*ListValue` has exactly 2 methods; no accessor layer exists | `grep -n 'func (.*ListValue)' internal/eval/*.go`; control `grep -c 'func (' internal/eval/value.go` | 2 hits: `value.go:88` `Type()`, `:89` `String()`; control 42 `func (` in the same file |
| L4 | `ArrayValue` and `TupleValue` also own `Elements []Value` — the same-name fields the analyzer must NOT count | read `internal/eval/value.go:84-86` (ListValue), `:103-105` (ArrayValue), `:249-252` (TupleValue) | all three declare `Elements []Value`; ArrayValue's doc comment says "O(1) indexed access" |
| L5 | Textual blast radius at HEAD (contaminated upper bounds, same scope as roadmap N3) | `grep -rn '\.Elements' --include='*.go' internal/ cmd/ \| wc -l`; same for `'ListValue{'`; control `'func '` | **903** `.Elements`; **388** `ListValue{`; control 20,548 — consistent with N3's 902/386/20,361 at `dedf3b91f` |
| L6 | Pattern-match call sites that size the API: exact-length compare, indexed head access, bounds check, O(1) tail alias; plus a same-file `TupleValue.Elements` use proving the discrimination problem is real in one screenful | `grep -n 'len(listVal.Elements)\|len(p.Elements)\|listVal.Elements\[i\]\|tailElements' internal/eval/eval_patterns.go` | `:218` exact-len `!=`; `:224`,`:244` `listVal.Elements[i]`; `:238` `<` bounds; `:255` `tailElements := listVal.Elements[len(p.Elements):]` + `:256` wrap in `&ListValue{...}`; `:161` is `tupleVal.Elements` — same field name, different type |
| L7 | `::` copies the whole tail; `_list_nth` is bounds-checked | read `internal/builtins/list.go:87-106` (`listConsImpl`); `grep -n 'out of bounds' internal/builtins/list.go` | cons: `make([]eval.Value, 0, 1+len(tail.Elements))` + two appends; nth: `:320` `"_list_nth: index %d out of bounds for list of length %d"` |
| L8 | `tools/linters/` does not exist (negative + control) | `test -d tools/linters`; control `ls -d tools/*/` | "tools/linters ABSENT"; control lists 18 existing directories (`tools/eval-elo/`, `tools/ci/`, `tools/internal/`, …) |
| L9 | `golang.org/x/tools` is NOT a direct module requirement — the analyzer adds a new direct dependency | `grep -n "x/tools" go.mod` → rc=1; control `grep -n "go-sqlite3\|testify" go.mod`; `grep -c "x/tools" go.sum` | go.mod: no hit (rc=1); control fires (`:23` go-sqlite3, `:28` testify); go.sum has 9 x/tools lines (transitive graph only, not importable without a `require`) |
| L10 | `iter.Seq` is available and has exactly 4 in-repo uses, all in the LC-1 spike — the accessor layer is its first `internal/` use | `head -4 go.mod`; `grep -rn "iter\.Seq" --include='*.go' internal/ cmd/ tools/` | `go 1.26.6`; 4 hits, all under `tools/internal/spike-listrep/` |
| L11 | `make ci` is an aggregate that GitHub CI never invokes — the gate needs its own ci.yml step, not just a ci.mk edit | `grep -n "make ci\b" .github/workflows/ci.yml` → rc=1; same-file control: `grep -n "make \|run:" .github/workflows/ci.yml` | rc=1 for `make ci`; control fires 30+ times (`:43 make deps`, `:130 make check-file-sizes`, …) — ci.yml runs individual targets |
| L12 | Repo gate precedent: script-backed `check-*` targets live in `make/code-health.mk`, and gates ship with their own gate-self-test CI step | `grep -n -A8 "^check-boundaries:" make/*.mk`; ci.yml lines 146–163 | `code-health.mk:139-147`: `check-boundaries`/`check-changelog`/`check-autoclose`, each `@bash scripts/...`; ci.yml runs both `make check-autoclose` and `make test-check-autoclose` |
| L13 | Direct `.Elements` MUTATION at HEAD: 2 non-test assignment sites, both parser AST nodes (not eval values); index-assignment 0, with a firing regex control | `grep -rn '\.Elements *=' --include='*.go' internal/ cmd/ \| grep -v _test.go \| grep -v '=='`; `grep -rnE '\.Elements\[[^]]+\] *=[^=]' ...` → rc=1; control `echo 'x.Elements[0] = y' \| grep -cE <same regex>` → 1 | `parser_literals.go:248` (`list.Elements = append(...)` on `*ast.ListLiteral`), `:282` (on `*ast.ArrayLiteral`); index-assign: zero hits, control fires — so Rule 2 starts from a clean tree *for direct syntax* (aliased writes are a named non-instrument, see Unverified) |
| L14 | No symbol collision for the proposed names in `internal/eval` | `grep -rn "func NewList\|func EmptyList\|func Cons(\|func (l \*ListValue) Len" --include='*.go' internal/eval/`; control `grep -rc "func NewCoreEvaluator" internal/eval/eval_evaluator.go` | only near-miss is `builtin_errors.go:149` `EmptyListError` (different symbol); control = 2 |
| L15 | `internal/eval/value.go` is 437 lines — headroom under the 800-line gate, but a sibling file is still chosen (constructor confinement, see Design) | `wc -l internal/eval/value.go internal/eval/eval_patterns.go` | 437 and 781 |
| L16 | LC-1's as-built accessor surface (the D3 draft this doc finalizes) | read `tools/internal/spike-listrep/list.go:11-18` | `List` interface = `Len() int`, `At(int) (eval.Value, bool)`, `All() iter.Seq[eval.Value]`, `ToSlice() []eval.Value`, `Uncons() (eval.Value, List, bool)`, `DropPrefix(int) List`; constructors are package-level per arm (M6 §6 divergence 1) |
| L17 | LC-1's evidence artifact exists and is structured | `test -e tools/internal/spike-listrep/testdata/m6-matrix.json`; `python3 -c "import json; d=json.load(open(...)); print(list(d.keys())[:6])"` | EXISTS; dict keys `metadata, partial, ac1_cells, b_len_cells, candidates, overall` |
| L18 | C1's measured margins (context for API sizing; read from the M6 report, which is generated from L17's JSON — trials NOT re-run here) | read `design_docs/implemented/v0_34_0/m-list-repr-spike-M6-report.md` §2, §5 | C1: clause (a) 1.2416/1.2229 (≤1.5), (b) 1.1065/**0.9496** (≤2.0 — faster than slice at worst-n), (c) 1.9524 (≤2.5; 32.001 vs C0's measured 16.418 B/element), (d) 0.9987, (e) pass |
| L19 | Duplicate/coverage gate | `ailang docs search "list accessor API analyzer migration ratchet" --neural` | Neural unavailable — CLI fell back (header `🔍 SimHash search`, 1415 docs). Top hits are keyword noise (an unrelated `++`-inference bug doc at 1.00). The genuinely related docs are this programme's own (roadmap, LC-1, superseded arena doc) — all in Related Documents; same fallback recorded at roadmap N20 and LC-1 V13 |
| L20 | Nothing in flight collides with the seam: open PRs are 3 dependabot + 1 stale coordinator branch | `gh pr list --state open --json number,title,headRefName` | `750` (actions bump), `695` (old coordinator task), `627`/`431` (ui deps) — none touch `internal/eval`, `tools/`, `make/`, or ci.yml |
| L21 | The CONCURRENT array-codegen sprint (M-ARRAY-SHOW M4 pending) has a disjoint file surface | `grep -nE '(internal\|cmd\|tools)/[a-z_/]+\.go' design_docs/planned/m-array-show-diverges-run-vs-compile-sprint-plan.md \| grep -io "internal/[a-z_/]*\|cmd/[a-z_/]*" \| sort -u` | `internal/builtins/registry_codegen_json`, `internal/gen/golang/codegen`, `internal/gen/golang/codegen_runtime_slices`, `internal/types/unification` — zero overlap with this seam (`internal/eval`, `tools/linters/`, `make/`, ci.yml) |
| L22 | The LC-1 spike itself holds 7 `.Elements` references (its C0 control arm wraps the production type). Round 1 read this as pinning the scan scope to `internal/` + `cmd/`; round 2 keeps the spike visible-but-exempt inside a whole-module scope instead (L25) | `grep -rn '\.Elements' tools/internal/spike-listrep/*.go` | 7 hits, all in `slicelist.go` (`:27,:38,:44,:49,:58,:65,:75`) |
| L23 | `make test` compiles and unit-tests everything under `tools/` too — the analyzer's own tests run in CI with no extra wiring | read `make/test.mk:27-30` | `$(GOTEST) -v $$($(GOCMD) list ./... \| grep -v /scripts \| grep -v /examples/agents)` — module-wide |
| L24 | `internal/eval/value.go` is low-churn (collision risk of the additive seam) | `git log --oneline -5 -- internal/eval/value.go` | most recent touch is `e9ac3f2ed` (effects replay-contract work, well before this programme) |
| L25 | **Controller census (round-1 quorum triage):** zero `.Elements` in `tools/` outside the LC-1 spike, zero in `examples/` — widening the scan scope to `./...` changes the LC-2 baseline by **ZERO** sites today | scopes asserted first: `test -d internal cmd tools examples` → all four exist. `grep -rn '\.Elements' tools --include='*.go' \| wc -l` → **7**; `… \| grep -c spike-listrep` → **7** (ALL of them); `… \| grep -v spike-listrep \| wc -l` → **0**; `grep -rn '\.Elements' examples --include='*.go' \| wc -l` → **0**. Same-scope known-positive controls: `tools/` files containing `package ` → 31; `find examples -name '*.go' \| wc -l` → 18. Negative control, fresh literal, same scopes: `zzqxAbsent251` → 0 | 7/7/0/0 with firing positive controls and a clean negative control — the round-1 doc never measured this (gemini-3-1-pro's catch was correct); the measured answer is zero |
| L26 | **Round-1 Rule 2 inherited Rule 1's exemption set**, which contains every accessor body — so `l.Elements[0] = x` inside `At` was exempt from the very mutation rule that exists to forbid it; and composite-literal coverage lived only in Rule 1, which retires at LC-4, leaving the retargeted cell fields unprotected | read the round-1 doc's own text: Rule 1 class (c) — "sites inside the constructor allowlist (`eval.NewList`, `eval.EmptyList`, `eval.Cons`, and the accessor method bodies in `value_list.go`) are exempt"; Rule 2 — "outside the constructor allowlist" | confirmed first-party by the controller — gpt5-6-sol's objection is a correct reading, not a misreading; fixed in this revision (Rule 2's exemption set is now independent and structural) |
| L27 | **Controller re-measure of gemini-3-1-pro's round-2 premise (CONFIRMED exactly): 388 `ListValue{…}` composite-literal sites exist at HEAD**, so a zero-tolerance Rule-2 class covering them would flag all 388 on LC-2 day one, in a piece that migrates no consumers | `grep -rn "ListValue{" internal cmd --include='*.go' \| wc -l`; split by `_test.go`; same-scope control `grep -rn 'ArrayValue{' internal cmd --include='*.go' \| wc -l`; negative control `ZzqxValue251{` | **388** total (**291** `_test.go`, **97** non-test); control **14**; negative control **0** — objection confirmed, fix applied verbatim |
| L28 | **Controller re-measure of gemini-3-1-pro's round-2 catch (PARTIALLY REFUTED, in the doc's favour): L13's `grep -v _test.go` filter was hiding nothing** — test files are mutation-free by the same syntax check | `grep -rnE "\.Elements(\[[^]]*\])? *(=|\+=)[^=]" internal cmd --include='*.go'`, WITHOUT the test filter, then split | **2** hits total, **0** in `_test.go`, **2** non-test — the same two parser-AST-node hits L13 already reports and Rule 2's type filter excludes. The reviewer was right that the doc had not verified it; the measured answer is that there is nothing there. No triage-on-first-run declaration is needed |

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Accessor surface = LC-1's as-built D3 (6 methods + `Type`/`String` + 3 package-level constructors), sized for **C1** | Every LC-3 migration targets exactly this surface; LC-4 reimplements exactly these bodies | this doc (bounded by D-22) | design | high |
| `At` returns `(Value, bool)`, never an error or panic | Shapes every migrated indexed call; error formatting stays at the caller (e.g. `_list_nth`'s message, L7) | this doc | design | med |
| `NewList([]Value)` **consumes ownership of `elems`**; caller use after the call is invalid, and the doc gives NO guarantee about copying or aliasing | Representation-independent by construction, so LC-4 can build cells without an observable contract change (`gpt5-6-sol` round-2, applied verbatim under the carve-out). The zero-copy implementation is still what LC-2 ships — it preserves today's cost at all 388 sites — but it is an implementation fact, not a pinned API promise | this doc | design | med |
| Ratchet semantics = **exact-match** per-package baseline (any drift fails, both directions) | "May only decrease" with a stale baseline permits re-growth inside an improved package; exact-match locks every decrease in AND makes a suddenly-empty result loud (0 ≠ baseline → red) | this doc | design | med |
| Rule 2 (mutation rule) is **permanent**, config-driven by `(type, field)` pairs, with **structural write-context validation and an exemption set independent of Rule 1's** | LC-4 retargets the same rule from `ListValue.Elements` to the cell fields without a new tool; a function-identifier allowlist is too coarse for Rule 2 — round 1 let it inherit Rule 1's exemptions, exempting every accessor body from the invariant Rule 2 enforces (L26, gpt5-6-sol) | roadmap quorum (inherited) + quorum round 1 | design | high |
| Analyzer scan scope = `./...` (the whole module); the LC-1 spike is an explicit **package exemption** (`github.com/sunholo-data/ailang/tools/internal/spike-listrep`, both rules), not a scope carve-out | A scope restriction is invisible to a ratchet BY CONSTRUCTION — a `.Elements` use added to `tools/` or `examples/` tomorrow can never be seen by a gate that does not load those packages. Measured, not predicted: widening changes today's baseline by ZERO sites (L25), which is what makes the fix free | quorum round 1 (gemini-3-1-pro) + this doc | design | med |
| `golang.org/x/tools` becomes a direct dependency | `go/analysis` + `go/packages` + `analysistest` are the canonical toolchain; not currently required (L9) | this doc | design | low |
| Gate wiring = target in `make/code-health.mk` + `ci:` aggregate in `make/ci.mk` + **its own ci.yml steps** | ci.yml never runs `make ci` (L11), so a ci.mk-only edit would be a gate that never gates; refines the roadmap's "make/ci.mk wiring" seam wording to match how every existing gate actually ships (L12) | this doc | design | med |

### Design Freeze

Resolved before implementation starts (all resolved in this doc):

- [x] Accessor surface and names (see table below; collision-checked at L14)
- [x] `NewList` consumes ownership of `elems`; no copying/aliasing guarantee either way (round-3 carve-out). Caller use after the call is invalid. AC-3 pins order and pre-transfer non-mutation, NOT post-transfer caller mutation
- [x] Exact-match ratchet semantics + `make listrep-baseline` regeneration flow
- [x] Rule 2 target config = `(type, field)` pairs; Rule 2 write-context validation =
      **structural** (fresh-allocation inside a configured constructor), NOT the Rule-1
      function-identifier allowlist — the two rules' exemption sets are independent (round 2)
- [x] Rule 1 allowlist = function identifiers (package path + name), not file paths
- [x] Scan scope `./...` — whole module; spike-listrep visible-but-exempt via a
      package-path exemption that applies to both rules (control arm; disposed at LC-4)
      (round 2)
- [x] D-22 = C1: no chunk-aware anything anywhere in this piece

## Deferred Decisions

Intentionally left open for the implementer (`agent may choose` unless noted):

- Exact diagnostic message wording (must include the site class and the fix direction, e.g.
  "migrate to All()/ToSlice(); see m-list-accessor-api.md")
- baseline.json field layout beyond the required content (per-package counts, analyzer version,
  scope string; deterministic ordering so diffs are stable)
- Internal analyzer code organization (single pass vs two passes sharing one `go/packages` load)
- Whether `EmptyList()` returns a fresh value or a shared singleton — **fresh at LC-2**
  (pointer identity must not become observable before LC-4's identity audit, roadmap
  "Unverified" table); LC-4 may revisit
- Names of the make targets (`check-listrep` / `test-listrep-gate` / `listrep-baseline` are the
  defaults; keep the `check-*`/`test-check-*` house pattern, L12)

**NOT deferred — an LC-4 OBLIGATION carried forward from this doc's round-2 quorum.** When LC-4
retargets Rule 2 to the new cell `(type, field)` pairs, it MUST add the **configured-field
composite-literal class to Rule 2** at that point. Rule 1 retires at LC-4 and Rule 1 is what
reports composite literals during LC-2/LC-3, so without this the retargeted cell fields lose
composite-literal protection at exactly the moment the swap makes it matter (`gpt5-6-sol`
round-2). It is safe to add only then, because the Rule-1 ratchet has driven the count to zero by
construction (`gemini-3-1-pro` round-2, L27). AC-13's fixtures already exercise the class under a
simulated LC-4 config, so LC-4 inherits a proven fixture rather than a note.

## Solution Design

### Half 1 — the accessor layer (`internal/eval/value_list.go`, new file)

A new sibling file in `package eval` — NOT edits inside `value.go` — for two reasons: (a) the
constructor allowlist is easiest to audit when every representation-field write the constructors
perform lives in one file (mirrors the spike's AC-4 leg-3 discipline); (b) `value.go` stays the
value-type inventory it is today (437 lines, L15). This is a refinement of the roadmap's
"`internal/eval/value.go` (additive methods only)" seam wording: same package, same additive-only
constraint, one file over.

The surface is LC-1's D3 **as built and benchmarked** (L16), with the two M6 divergences settled
and the two draft caveats decided:

| Signature | Today (slice) | After LC-4 (C1 cells) | Call-site justification |
|---|---|---|---|
| `func (l *ListValue) Len() int` | `len(l.Elements)`, O(1) | cached `n`, O(1) | exact-length patterns `eval_patterns.go:218`, bounds `:238` (L6) |
| `func (l *ListValue) At(i int) (Value, bool)` | index, O(1) | spine walk, O(i) | `_list_nth` `list.go:320` (L7); indexed pattern heads `eval_patterns.go:224,:244` (L6) |
| `func (l *ListValue) All() iter.Seq[Value]` | range over slice, O(n) | pointer chase, O(n) | every spine walk (`String()`, map/fold family); first `internal/` use of `iter.Seq` (L10) — stdlib-adopted per LC-1's reuse decision |
| `func (l *ListValue) ToSlice() []Value` | fresh copy, O(n), **documented** | fresh copy, O(n) | the materialize target for the escape sites (roadmap N13/N22); never aliases |
| `func (l *ListValue) Uncons() (Value, *ListValue, bool)` | O(1) subslice alias | O(1) shared tail pointer | pattern tail extraction `eval_patterns.go:255-256` (L6) — the seam where sharing will pay; must not regress to a copy TODAY (AC-2) |
| `func (l *ListValue) DropPrefix(k int) (*ListValue, bool)` | O(1) subslice alias | O(k) walk | multi-element `[a, b, ...rest]` tails at offset `len(p.Elements)` (L6). M6 flagged "give it a row or fold it": **kept** — folding into repeated `Uncons` would turn one O(1) alias into k allocations today. `(nil, false)` on `k < 0 \|\| k > Len()`, symmetric with `At` |
| `func NewList(elems []Value) *ListValue` | consumes ownership of `elems` | consumes ownership of `elems` | the 388 construction sites (L5). **Contract (round-3 carve-out, `gpt5-6-sol`'s verbatim text):** *"`NewList` takes ownership of `elems`; callers must not access or mutate it afterward. The implementation may wrap or materialize the input, so LC-4 can build cells without an observable contract change."* The contract is therefore representation-INDEPENDENT: no copying/aliasing guarantee is given in either direction, so LC-4's spine build is not a contract change |
| `func EmptyList() *ListValue` | fresh `&ListValue{}` | canonical empty | empty-literal constructions |
| `func Cons(head Value, tail *ListValue) *ListValue` | copy-prepend (mirrors `listConsImpl`, L7), O(n) | one cell, O(1) | `::` — the point of the programme |

Constructors are package-level functions, not methods — M6 divergence 1, kept: it is what makes
"field write inside a constructor" mechanically distinguishable for Rule 2, and constructors
cannot sit on the value they construct. `At` returns `(Value, bool)` — draft caveat settled:
error text belongs to callers (the builtin's message at `list.go:320` stays exactly where it is).
Receiver contract: non-nil, same as the existing `String()`.

**No consumer is migrated.** The new methods have zero callers outside their own tests at LC-2
merge. Behaviour change: none, by construction (additive file + tests only).

### Half 2 — the `listrep` analyzer (`tools/linters/listrep/`, new)

A `golang.org/x/tools/go/analysis` analyzer with two rules over one shared `go/packages` load of
`./...` — **the whole module** — plus a driver binary and a committed baseline.

**Scan scope (revised in round 2).** Round 1 restricted the load to `./internal/... ./cmd/...`
to keep the LC-1 spike's 7 control-arm refs (L22) out of the count. The quorum caught what that
trades away: a scope restriction is invisible to a ratchet **by construction** — a `.Elements`
use added to `tools/` (18 directories, e.g. `tools/eval-elo/`) or to a Go package under
`examples/` (18 `.go` files exist, L25 controls) tomorrow could never be seen by a gate that
does not load those packages; it would miss the LC-3 migration lanes, silently bypass Rule 2,
and surface as a surprise compile break at LC-4. A removal proves a check FIRES; only an
addition proves it LOOKS. So the scope is `./...`, and the spike is handled the other way
around: `github.com/sunholo-data/ailang/tools/internal/spike-listrep` is an explicit **package
exemption in the analyzer's configuration** (the reviewer's fix names Rule 1's config; because
the widened scope exposes the spike to Rule 2 as well, the package exemption covers both rules
— see Rule 1 class (c)) — visible-and-exempt, not invisible. Widening changes the
LC-2 baseline by **ZERO sites** (measured, not predicted — L25: the spike holds all 7 `.Elements`
refs under `tools/`, and `examples/` holds none), which is exactly why the fix costs nothing
today; the reason to take it is tomorrow.

**Rule 1 — migration census (ratcheted; retires at LC-4 into "field must not exist").**
Reports three site classes, each tagged in the diagnostic:

- **(a) selector**: any `SelectorExpr` `x.Elements` where `x` type-checks as
  `github.com/sunholo-data/ailang/internal/eval.ListValue` or `*...ListValue`. This includes
  aliasing reads like `s := lv.Elements` — each is a countable migration site.
- **(b) composite literal**: any `CompositeLit` of type `eval.ListValue` (keyed, positional, or
  empty — `&ListValue{Elements: r}`, `ListValue{r}`, `&ListValue{}` all construct without going
  through `NewList`/`EmptyList`). Without this class the 388 construction sites (L5) would be
  invisible to the ratchet and surface only as LC-4 compile breaks.
- **(c) allowlist exemption**: sites inside the constructor allowlist (`eval.NewList`,
  `eval.EmptyList`, `eval.Cons`, and the accessor method bodies in `value_list.go`) are exempt —
  they ARE the seam — as is the whole LC-1 spike package
  (`…/tools/internal/spike-listrep`, the round-2 scope decision above). Two distinct exemption
  kinds, deliberately not conflated: the **function-identifier allowlist** (constructors +
  accessor bodies; package path + name, not file paths) belongs to **Rule 1 ONLY** — it is a
  migration-census concern, and Rule 2 does not inherit it (L26). The **spike package
  exemption** is package-path-scoped and applies to **both rules**: the spike is LC-1's control
  arm, whose entire purpose is direct representation manipulation, so the gate must be
  insensitive to it; the exemption is one package path, recorded in the baseline as exempt, and
  disposed together with the spike at LC-4.

Same-name fields on other types — `ArrayValue.Elements`, `TupleValue.Elements`,
`ast`/`core`/`typedast` `Elements` (L4, L6, roadmap N4) — are **not reported**; the seeded
fixture proves this negatively (AC-7).

**Rule 2 — mutation rule (permanent; zero-tolerance, never ratcheted; exemption set
INDEPENDENT of Rule 1's — revised in round 2).**

Round 1 wrote "outside the constructor allowlist", inheriting Rule 1's exemption set — which
contains every accessor body, so `l.Elements[0] = x` inside `At` was exempt from the very
invariant Rule 2 exists to enforce, and whole-constructor exemption allowed
`tail.Elements[0] = head` inside `Cons` (L26, gpt5-6-sol — confirmed against the doc's own
text). This is the programme's own named recurring shape: a guard is not a gate until something
reds when you remove it, and round-1 Rule 2 could not red for the case it exists for. Revised:

Rule 2 has **no blanket accessor exemption and no whole-function constructor exemption**. It
reports, for each configured `(type, field)` pair — at LC-2, exactly
`(eval.ListValue, Elements)`:

- assignment or compound assignment whose LHS is (or indexes/slices into) the field selector
  (`lv.Elements = …`, `lv.Elements[i] = …`, `lv.Elements = append(lv.Elements, …)`)
- `IncDecStmt` on an element
- address-take of the field (`&lv.Elements`, `&lv.Elements[i]`)
- `copy(lv.Elements…, …)` or `append(x, …)` where the **first** argument is rooted at the field
  selector (both can write through the backing array; `append(dst, lv.Elements...)` — the spread
  READ used everywhere today, L7 — is not flagged)
**Composite-literal initializers are deliberately NOT a Rule-2 class at LC-2** (round-3
carve-out, applying `gemini-3-1-pro`'s verbatim round-2 fix). Round 2 added them here to close
the post-LC-4 hole; measured, that would have broken CI on day one. There are **388**
`ListValue{…}` construction sites at HEAD (L5, re-measured as L27: 388 total in `internal`+`cmd`,
291 in `_test.go`, 97 non-test; same-scope control `ArrayValue{` = 14; negative control 0), and
LC-2 migrates **no** consumers by design — so a zero-tolerance, never-ratcheted rule carrying this
class would flag all 388 immediately. Rule 1, which IS ratcheted, tracks them through LC-2 and
LC-3 and drives the count to zero. **LC-4 adds composite-literal protection to Rule 2 when it
retargets the rule to the new cell fields**, i.e. after the Rule-1 count has reached zero — at
which point the class costs nothing and the post-retirement hole `gpt5-6-sol` identified is
closed exactly when it opens. This is recorded as an LC-4 obligation in Deferred Decisions, not
as an aspiration.

The only accepted write context is validated **structurally**, not by function name alone: a
flagged write is exempt iff it initializes a **newly allocated value inside an explicitly
configured constructor** (`eval.NewList`, `eval.EmptyList`, `eval.Cons`) — i.e. the write's
base object is a value allocated in that same constructor body (`&ListValue{…}`,
`new(ListValue)`, or a composite literal bound to a local) that did not exist before the call.
Even inside a configured constructor, writes rooted at **parameters, receivers, globals, or any
previously existing value are rejected**: `tail.Elements[0] = head` inside `Cons` is a
violation exactly as it would be anywhere else, and accessor bodies get no exemption at all —
`l.Elements[0] = x` inside `At` is a violation. The single package-scoped exception is the LC-1
spike (see Rule 1 class (c)): whether its 7 refs include Rule-2-shaped writes is unmeasured (L22
counts selectors, not write contexts), and the package exemption makes the gate insensitive to
that either way — it is the control arm, and it dies at LC-4. Constructors are deliberately kept in the
trivially analyzable allocate → initialize → return shape (they are ours, in one ~130-LOC
file); if a constructor shape ever defeats the structural check, the constructor is
restructured, not the rule loosened.

Direct-syntax mutation at HEAD is already zero for eval values (L13 — the two hits are parser
AST nodes Rule 2's type filter excludes), so this rule goes in green; if the sprint's first
full run finds any `copy`/`append`-rooted hit, each is triaged in-sprint (latent bug → fixed as
its own commit, or a genuine constructor-shaped write → made to satisfy the structural check,
with a comment). **What Rule 2 is not**: it catches mutation *syntax rooted at the selector*, not
writes through a previously-copied slice header (`s := lv.Elements; s[0] = x` flags the alias
under Rule 1 but the write is invisible to Rule 2). That residual is bounded by the ratchet
driving Rule 1's count — including every alias site — to zero in LC-3, and is named in
Unverified rather than papered over. This is the enforcement design the programme quorum already
ratified: unexporting + analyzer **confines** mutation, it does not eliminate it; nothing in this
doc claims compiler-enforced immutability, and immutability/safe-publication remain required
properties verified at LC-4 (roadmap N21).

**Driver and ratchet** (`tools/linters/listrep/cmd/listrep/main.go`):

1. **Self-test first, every run**: before scanning the real tree, the driver runs the analyzer
   over its own `testdata/` fixture module and requires the exact expected diagnostic set (N
   Rule-1 sites of each class, M Rule-2 violations, 0 false positives on the Array/Tuple/ast
   decoys). Any mismatch → exit 3, "instrument failure", no verdict. A broken or neutered
   analyzer can therefore never green the gate — this is the empty-result trap converted into a
   mechanism, and it runs unconditionally, not only in tests.
2. **Scan** `./...` (the whole module); aggregate Rule-1 counts per package. The spike package
   appears in the baseline as explicitly exempt, not silently absent; every other `tools/` and
   `examples/` package is ratcheted and denominated like the rest of the tree (today at 0, L25).
3. **Compare** to `tools/linters/listrep/baseline.json` — exact match per package. Higher →
   fail listing the new sites. Lower → fail with "progress! run `make listrep-baseline` and
   commit" (locks the decrease in). Any Rule-2 diagnostic → fail unconditionally.
4. `-write-baseline` regenerates the file (deterministic ordering). The baseline records the
   scope string (`./...`), the exemption configuration, and the analyzer version so a scope or
   exemption drift is a visible diff, not a silent renumber.

**The census deliverable** (the roadmap's FIRST deliverable for LC-2): the initial baseline run's
per-package table goes verbatim into this doc's implemented-report, replacing the 903/388 textual
upper bounds as the programme's denominator. From the same output, the report enumerates the
**escape classification**: every Rule-1 site whose selector value flows out of the owning
function as a raw slice (hand-audited over the bounded site list, starting from the three known
escapes — `builtins/safe_cast.go:97`, `embed/convert.go:346`, `testctx/mock_context.go:353-355`,
roadmap N13 — which must all appear in the analyzer's output or the run is an instrument
failure). This enumeration is what sizes LC-3a/b/c; the roadmap's "3 currently known
syntax-matched escapes" wording is upgraded to a measured list by this census, not before.

### CI wiring

Per L11/L12, three edits: `make/code-health.mk` gains `check-listrep` (runs the driver),
`listrep-baseline` (regenerates), and `test-listrep-gate` (the gate-of-the-gate, below);
`make/ci.mk`'s `ci:` aggregate gains `check-listrep`; `.github/workflows/ci.yml` gains two steps
(`make check-listrep`, `make test-listrep-gate`) following the check-autoclose pattern at ci.yml
lines 146–163.

`test-listrep-gate` (`scripts/test_listrep_gate.sh`) proves the gate fires, both directions:
in a temp copy of the fixture module it (i) seeds one extra `.Elements` selector → driver must
exit non-zero naming it; (ii) seeds one non-constructor assignment → non-zero; (iii) confirms
the unmodified fixtures + real tree pass; (iv) runs the driver with `-self-test-only` against a
deliberately emptied expectation → must exit 3. Arm (iii) is the seeded-ACCEPTED case: the
fresh-allocation constructor writes in `value_list.go` and the fixture constructor are present
and produce zero diagnostics — proving the analyzer is not simply rejecting everything. Round 2
adds **arm (v), the scope-coverage seed**: create a small throwaway package under `tools/`
(e.g. `tools/listrep-gate-seed/`) containing one `*eval.ListValue.Elements` selector, run the
driver with the SAME `./...` invocation CI uses → it must exit non-zero naming the seeded site;
then delete the seed directory. This is the arm that would have FAILED under the round-1 scope
— it proves the gate looks outside `internal/`+`cmd/`, not merely that it fires where it
already looked.

## Files to Modify/Create

**New files:**
- `internal/eval/value_list.go` (~130 LOC) — accessor methods + `NewList`/`EmptyList`/`Cons`
- `internal/eval/value_list_test.go` (~250 LOC) — slice-equivalence, aliasing pins, ownership pin
- `tools/linters/listrep/listrep.go` (~360 LOC) — the analyzer, both rules, config; Rule 2's
  structural write-context validation (fresh-allocation tracking inside configured constructors)
- `tools/linters/listrep/listrep_test.go` (~140 LOC) — `analysistest` over testdata, both configs
- `tools/linters/listrep/testdata/src/fixture/…` (~230 LOC) — seeded positives (both rules, all
  site classes), the four AC-13 structural Rule-2 fixtures, seeded accepted constructor,
  Array/Tuple/ast-decoy negatives; plus a second config fixture with a cell-shaped type
  (`{head Value; tail *cell; n int}`) exercising the same four cases under the planned LC-4
  retarget
- `tools/linters/listrep/cmd/listrep/main.go` (~180 LOC) — driver: self-test, scan, baseline
  compare, `-write-baseline`
- `tools/linters/listrep/baseline.json` (generated at sprint time; committed; whole-module scope)
- `scripts/test_listrep_gate.sh` (~110 LOC) — gate red/green/neuter/scope-coverage test, five arms

**Modified files:**
- `make/code-health.mk` (+~14 LOC) — three targets
- `make/ci.mk` (+1 LOC) — `ci:` aggregate
- `.github/workflows/ci.yml` (+~10 LOC) — two steps
- `go.mod` / `go.sum` — `golang.org/x/tools` becomes direct (L9)
- `changelogs/` current file — entry under Unreleased (root `CHANGELOG.md` stays an index per
  the check-changelog gate, L12)

## Examples

Migration shape (executed by LC-3, enabled here) — today's pattern-tail site
(`eval_patterns.go:255-256`, L6):

```go
// Before (LC-3b will replace; shown to prove the surface fits the hardest O(1) site):
tailElements := listVal.Elements[len(p.Elements):]
tailList := &ListValue{Elements: tailElements}

// After — same O(1) alias today, O(k)-walk-to-shared-tail after LC-4, zero copies either way:
tailList, ok := listVal.DropPrefix(len(p.Elements))
```

The ratchet in action, after LC-2 lands:

```
$ make check-listrep
listrep: internal/builtins: 138 sites (baseline 138) ✓
listrep: internal/eval:      41 sites (baseline  41) ✓
...
$ # someone adds a new lv.Elements use in a PR:
listrep: internal/effects: 34 sites (baseline 33) — NEW SITE:
  internal/effects/encode.go:88: eval.ListValue.Elements selector (class a)
  migrate to All()/ToSlice(); see design_docs/planned/m-list-accessor-api.md
make: *** [check-listrep] Error 1
```

(Counts illustrative — the committed baseline is the analyzer's first-run output, which is the
census deliverable; the textual 903 is not it.)

## Conflict Surface

This piece touches `internal/eval` (the runtime's central value type) and adds a required CI
gate, so this section is mandatory. It is an API/CI conflict surface, not a syntactic one — no
parser, lexer, or grammar position is touched and no `.ail` surface changes.

**Positions touched and what else lives there:**

| Position | What already lives there | Why the seam is safe |
|---|---|---|
| Method set of `*eval.ListValue` | exactly `Type()`, `String()` (L3) | Adding methods cannot un-satisfy any existing interface; the 8 new names collide with nothing in the package (L14, nearest miss `EmptyListError`) |
| Package-level names in `eval` | `New`, `NewSimple`, `NewCoreEvaluator`, … (L14) | `NewList`/`EmptyList`/`Cons` are free; `Cons` is package-scoped so it cannot shadow anything outside `eval` |
| The `Elements` field NAME | owned by ≥22 types across 9 packages — `ArrayValue`/`TupleValue` in the same file (L4), `tupleVal.Elements` 57 lines above the list-pattern code (L6) | The analyzer discriminates by `types.Named` identity, never text; AC-7's decoy fixtures make a false positive a red test |
| CI pipeline (every future PR) | 30+ individual make-target steps (L11); gate precedent with self-tests (L12) | Exact-match baseline gives a clear both-directions failure message + one-command regeneration; the self-test makes analyzer rot loud rather than silently green |
| Concurrent mission work | array-codegen sprint M4: `internal/gen/golang`, `internal/builtins/registry_codegen_json`, `internal/types/unification` (L21); open PRs all dependabot/stale (L20); `value.go` low-churn (L24) | Zero file overlap with this seam. `internal/gen`'s 36 textual `.Elements` are compile-time types the analyzer excludes by construction. ci.yml/ci.mk edits are append-shaped |
| The LC-1 spike | 7 `.Elements` refs in its C0 control arm (L22) | Inside the round-2 `./...` scan scope but carried as an explicit package exemption in Rule 1's config — visible-and-exempt, never invisible. The spike's disposition (delete vs keep as LC-4's before/after instrument) is LC-4's decision, and LC-4's field deletion will surface these 7 refs at compile time then |

**Programs that MUST still work** (regression fixtures — trivially, since no consumer changes;
their role is to catch accidental non-additivity): the programme's five named fixtures —
`examples/first_non_repeat.ail`, `examples/inline_tests_recursive.ail`,
`examples/record_cons_pattern.ail`, `examples/record_list_extraction.ail`,
`examples/pattern_matching_adt.ail` — plus `make test` and `make verify-examples`, byte-identical
output (AC-10).

**What deliberately changes:** nothing at runtime. CI gains one required gate (the point). The
`go.mod` dependency set grows by `golang.org/x/tools` (L9).

## Testing Strategy

**Unit — accessors** (`value_list_test.go`): table/property tests over generated lists (empty,
1, 2, 47, 1000 elements): `Len()==len(Elements)`; `At(i)` equals `Elements[i]` in-bounds and
`(nil,false)` out; `All()` yields exactly the slice order (including early-break); `ToSlice()`
deep-equals and does NOT share a backing array; `Uncons`/`DropPrefix` equal the subslice AND (the
aliasing pin) share the source's backing array today — asserted via `&tail.Elements[0] ==
&src.Elements[k]`; `NewList` does not copy (ownership pin); `Cons` output equals
`listConsImpl`'s for the same inputs.

**Unit — analyzer** (`analysistest` over `testdata/`): every Rule-1 site class flagged at its
`// want` marker; Array/Tuple/ast decoys produce zero diagnostics; and the four **structural
Rule-2 fixtures** (round 2, gpt5-6-sol's list), each with the mutation that makes it red:

1. `l.Elements[0] = x` inside a fixture `At` method body → MUST be reported. *Reds when:* an
   accessor-body exemption leaks into Rule 2 (the round-1 defect, L26).
2. `tail.Elements[0] = head` inside a fixture `Cons` constructor (write rooted at a parameter,
   inside a configured constructor) → MUST be reported. *Reds when:* the constructor exemption
   is whole-function rather than structural.
3. A configured-field composite literal (`&ListValue{Elements: xs}`) in a non-constructor
   function → MUST be reported **by Rule 1** at LC-2 (it is a countable migration site, and
   Rule 1 is ratcheted so the 388 existing sites do not break CI), and MUST be reported **by
   Rule 2** under the simulated LC-4 cell-field config, where the migration count is zero.
   *Reds when:* the class is absent from Rule 1 (388 sites become invisible to the census), or
   absent from the LC-4 Rule-2 config (the hole re-opens when Rule 1 retires).
4. Fresh-object initialization inside a configured constructor (`c := &ListValue{Elements: xs};
   return c`) → MUST produce zero Rule-2 diagnostics. *Reds when:* the structural check rejects
   the one legitimate write context.

All four cases are then repeated under a **second testdata config** targeting a fixture
cell-shaped type (`{head Value; tail *cell; n int}`), demonstrating the planned LC-4 retarget
preserves the constructor-only write discipline with no tool change (AC-13).

**Gate-level** (`test-listrep-gate`, in CI): the five arms in the CI-wiring section — seeded
violation reds, seeded acceptance greens, real tree greens, neutered analyzer reds with the
distinct instrument-failure code, and the out-of-scope seed under `tools/` reds (arm (v),
AC-12).

**Regression-surface**: AC-10's five fixtures + full suite, byte-identical.

## Success Criteria

Each names the observation that reds it.

- [ ] **AC-1 (slice equivalence):** every accessor unit test above passes. *Red if:* any wrapper
  diverges from direct-slice semantics on any tested shape, including the empty list.
- [ ] **AC-2 (aliasing pins):** `Uncons`/`DropPrefix` results share the source backing array;
  `ToSlice` does not. *Red if:* a copy appears on the O(1) paths (the LC-3b pattern-tail
  microbench AC would inherit a regression) or `ToSlice` starts aliasing (the escape-site
  materialize contract breaks before LC-3 uses it).
- [ ] **AC-3 (construction correctness — round-3 carve-out, replacing the round-2 ownership
  pin):** tests verify (a) that a list built by `NewList(elems)` yields exactly `elems`' element
  order under `Len`/`At`/`All`/`ToSlice`, and (b) that no later list operation mutates a
  caller-owned slice **before** it is transferred. The test suite deliberately does **NOT**
  exercise post-transfer caller mutation — that is undefined behaviour under the contract above,
  and pinning it would make a representation artifact part of the tested API and force a
  behavioural contract change at LC-4 (`gpt5-6-sol` round-2, applied verbatim).
  *Red if:* construction reorders or drops elements, or a list operation writes through a slice
  the caller still owns.
- [ ] **AC-4 (true count + hand audit):** `baseline.json` is committed from the first full
  `./...` run — every module package denominated, the spike listed as exempt (expected zero
  non-exempt sites outside `internal/`+`cmd/` per L25, but the baseline RECORDS that rather
  than assuming it); its count for ONE sampled package (`internal/embed` — small, contains a
  known escape) matches
  a by-hand audit of that package's sites, and every one of the three known escape sites
  (roadmap N13) appears in analyzer output. *Red if:* hand-audit disagrees, any known escape is
  missing (instrument failure), or the committed baseline totals 0 anywhere the textual grep
  shows eval-value usage (e.g. `internal/builtins`).
- [ ] **AC-5 (ratchet fires, both directions):** `test-listrep-gate` arms (i)–(iii) pass —
  seeded new selector reds, seeded non-constructor assignment reds, unmodified tree greens.
  *Red if:* any arm fails — a gate that has never fired is a claim, not a gate.
- [ ] **AC-6 (instrument-failure is loud):** arm (iv) — with expectations emptied (equivalent to
  a neutered analyzer), the driver exits 3 and the gate reds. *Red if:* a no-op analyzer can
  produce a green gate.
- [ ] **AC-7 (false-positive controls):** decoy fixtures (`ArrayValue.Elements`,
  `TupleValue.Elements`, an `ast`-style `Elements` struct) produce zero diagnostics. *Red if:*
  the analyzer counts a non-ListValue site — the census would be contaminated exactly the way
  grep is.
- [ ] **AC-8 (CI actually gates):** the PR's own CI run shows `check-listrep` and
  `test-listrep-gate` steps executing (job log evidence, not target existence). *Red if:* the
  targets exist but no ci.yml step runs them — the L11 trap.
- [ ] **AC-9 (zero behaviour change):** full `make test` + `make verify-examples` green; the
  five named fixtures byte-identical to base; `git diff` against merge-base shows `internal/eval`
  changes are the additive `value_list*.go` files only. *Red if:* any existing file under
  `internal/eval` is edited or any fixture output moves.
- [ ] **AC-10 (census handoff):** the implemented-report contains the per-package analyzer
  table and the escape classification enumerating ≥ the 3 known escapes with site IDs, explicitly
  labelled as LC-3a/b/c's sizing input. *Red if:* the census is missing, or migration ACs
  anywhere in it are stated in grep counts.
- [ ] **AC-11 (docs):** changelog entry under Unreleased; this doc moved to implemented with the
  report. *Red if:* the check-changelog gate or the move is missing.
- [ ] **AC-12 (the gate LOOKS outside `internal/`+`cmd/` — round 2):** gate-test arm (v) seeds
  a `*eval.ListValue.Elements` selector in a throwaway package under `tools/` and the driver's
  standard `./...` invocation reds naming the seeded site. *Red if:* the seed is invisible —
  the exact failure the round-1 scope would have produced. A removal proves a check fires; only
  an addition proves it looks.
- [ ] **AC-13 (Rule-2 structural fixtures — round 2):** the four fixtures in Testing Strategy
  pass under BOTH configs (LC-2 `ListValue.Elements` and the simulated LC-4 cell fields).
  *Red if:* mutation inside `At` goes unreported (accessor exemption leaked into Rule 2);
  `tail.Elements[0] = head` inside `Cons` goes unreported (constructor exemption is
  whole-function, not structural); a non-constructor configured-field composite literal goes
  unreported by Rule 2 (the post-LC-4 hole re-opens); or the fresh-allocation constructor case
  IS reported (the rule cannot green its one legitimate write context).

## Non-Goals

- **Migrating any consumer** — LC-3a (`internal/builtins`), LC-3b (eval-internal/effects/
  runtime/telemetry), LC-3c (periphery). The accessor methods have zero production callers at
  LC-2 merge, by design.
- **Changing representation, semantics, or performance anywhere** — LC-4.
- **Any chunk-aware design** — D-22 = C1 forecloses it (L2). If a chunk boundary appears
  anywhere in this piece's code, the ruling was misread.
- **Claiming immutability or safe publication** — required properties, verified at LC-4
  (roadmap N21). This piece builds the confinement instrument, not the proof.
- **Deleting or wrapping the `Elements` field** — the field stays public and untouched until
  LC-4's deletion proves migration completeness.
- **Statement-level symmetric-switch census consumption** — the analyzer's output enables it;
  acting on it is LC-3's per-lane work.
- **VM/bridge** — decoupled, documented divergence (roadmap N11).

## Timeline

- **Day 1:** `value_list.go` + full unit/aliasing/ownership tests (AC-1..3). `x/tools` dep in.
- **Day 2:** analyzer Rule 1 (three site classes, allowlist) + `analysistest` fixtures incl.
  decoys (AC-7); first full scan; triage anything surprising in the count.
- **Day 3 (the heaviest day — both round-2 fixes land here):** Rule 2 with structural
  write-context validation + the four AC-13 fixtures under both configs; self-test +
  driver/baseline/ratchet over `./...`; `make` targets; gate-level test script incl. arm (v)
  (AC-5, AC-6, AC-12, AC-13); measure analyzer wall-clock at full-module scope; commit baseline
  (AC-4 hand-audit of `internal/embed`).
- **Day 4 (handoff):** ci.yml wiring proven on the PR's own run (AC-8); census table + escape
  classification for the report (AC-10); changelog; fixture byte-identity check (AC-9).

Total: 4 days. The roadmap's band was 3–4 (Rule 2 and its fixtures already priced in); the
round-2 fixes — structural validation instead of an allowlist lookup, dual-config fixtures, and
arm (v) — consume what was Day 4's buffer. Stated explicitly rather than silently absorbed: 3
days is no longer the expected case, and 4 is still inside the roadmap band, so no re-estimate
of the programme is triggered.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Analyzer CI cost (a full `go/packages` type-check of the WHOLE module per PR — round 2 widened the scope from `internal/`+`cmd/` to `./...`) | Medium | One shared load for both rules; measure on Day 3 at the widened scope — if >~2 min, cache via `go vet -vettool` integration is the named fallback (same analyzer, vet's build cache). Note `make test` already compiles the whole module per PR (L23), so the marginal cost is one type-check, not a new build |
| Rule 2's structural freshness check (is this write's base allocated in this function?) proves harder than local syntactic tracking — e.g. an allocation reaching the write through multiple rebindings | Medium | Constructor bodies are ours, three functions in one ~130-LOC file, deliberately kept allocate → initialize → return; if a constructor shape defeats the checker, restructure the constructor, never loosen the rule. AC-13's four fixtures pin the behaviour in both directions |
| Baseline churn across concurrent worktrees (exact-match means any count-touching PR regenerates) | Medium | Deterministic JSON ordering makes conflicts textual and small; regeneration is one make target; LC-3 lanes are package-disjoint so their baseline diffs don't overlap |
| Rule 2 first full run finds `copy`/`append`-rooted hits L13's greps could not see | Low | Triage in-sprint: latent bug → own commit; legitimate constructor → allowlist entry with comment. Zero direct-assign sites measured (L13) bounds the surprise |
| Aliased-write blind spot read as a safety claim | Medium | Stated in-doc (Rule 2 scope), in Unverified, and in the driver's own `-help` text; the programme-level answer is Rule 1's count → 0 + LC-4's race matrix, not this rule |
| `NewList` ownership contract misused by future callers (slice accessed after transfer) | Low | The doc comment states the contract in the reviewer's verbatim terms (caller use after the call is invalid; no copying/aliasing guarantee), so misuse is undefined behaviour rather than a pinned semantic LC-4 would have to break. AC-3 tests order and pre-transfer non-mutation only. Rule 1 counts any direct `.Elements` alias a caller would need in order to observe the difference pre-LC-4 |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | 0 | No semantic change; additive Go API + CI gate only |
| A2 Replayability | 0 | No trace changes |
| A3 Effect Legibility | 0 | No effects |
| A4 Explicit Authority | 0 | No capabilities |
| A5 Bounded Verification | +1 | Converts "migration done?" from a textual guess (903, contaminated) into a mechanically checked, type-aware, self-testing gate with per-package denominators |
| A6 Safe Concurrency | 0 | No claims made; the confinement instrument this piece ships is a prerequisite for LC-4's verification, not a substitute for it |
| A7 Machines First | 0 | No model-facing change in this piece (the programme scores the payoff) |
| A8 Minimal Syntax | 0 | No syntax |
| A9 Cost Visibility | +1 | Every accessor carries its slice-cost AND its post-C1 cost in the doc comment (`At` O(1)→O(i), `ToSlice` documented copy, `Cons` O(n)→O(1)) — the costs stop being implicit before they change |
| A10 Composability | +1 | LC-3's three lanes and LC-4 all compose against this one seam; the analyzer config retargets to cell fields without a new tool |
| A11 Structured Failure | 0 | No error-handling changes (At's `(Value, bool)` keeps error formatting at callers) |
| A12 System Boundary | 0 | No boundary changes; escape materialization is LC-3's |

**Net Score: +3** → Proceed (hard-violation check A1/A3/A4/A7: all non-negative).

## Unverified / not claimed — named, with owners

| Claim | Status | Owner |
|---|---|---|
| The TRUE `ListValue`-typed site count (903/388 are contaminated textual upper bounds, L5) | Unknown until the analyzer's first run — producing it IS this piece's census deliverable | LC-2 execution (AC-4) |
| Exhaustiveness of the escape list ("3 currently known" is a lower bound; syntax greps cannot prove exhaustiveness) | Upgraded to a measured list by the census hand-audit over analyzer output; not asserted before | LC-2 execution (AC-10) |
| Immutability and safe publication of shared lists | Required properties, NOT established facts; Rule 2 confines direct-syntax mutation, it does not eliminate mutation (Go has no immutable fields) | LC-4 (roadmap N21: publication-path → happens-before-edge → race-test matrix) |
| Whether any alias-header write (`s := lv.Elements; s[0] = x`) exists at HEAD | Not measured (needs data-flow, which is exactly what Rule 2 does not do); bounded by Rule 1 counting every alias site and LC-3 driving the count to 0; old-doc V32/V33 measured zero aliased writes at its base | LC-3 lanes (each lane's zero-count AC) + LC-4 |
| Analyzer wall-clock in CI at the round-2 `./...` scope | Measured Day 3; ~<2 min budget with a named fallback (vettool) | LC-2 execution |
| Rule 2's structural freshness validation is implementable with local, single-function analysis (no SSA/data-flow) | Design intent, not yet demonstrated in code; constructors are written to stay in the trivially analyzable shape, and AC-13's fixtures are the check that it worked | LC-2 execution (Day 3) |
| The simulated LC-4 cell-field config matches LC-4's real cell type exactly | The fixture cell type is modeled on D-22's C1 shape (`{head, tail *cell, n}`), but LC-4 owns the real definition; the fixture proves retargetability of the mechanism, not the final config values | LC-4 (re-run the four fixtures against the real config) |
| Whether the spike's 7 `.Elements` refs include Rule-2-shaped writes | Unmeasured — L22 counted selectors, not write contexts; moot for the gate because the spike's package exemption covers both rules, but named here rather than silently assumed clean | LC-4 (spike disposition surfaces them at compile time) |
| `copy`/`append`-first-arg-rooted mutation sites at HEAD (L13 covers direct assign/index only) | Unmeasured; Rule 2's first full run measures; triage path predeclared | LC-2 execution (Day 3) |
| Pointer-identity observability of `*ListValue` (whether `EmptyList` may become a singleton) | Inherited unverified (roadmap table); fresh-value choice here makes it moot for LC-2 | LC-4 mandatory re-audit |
| LC-1's C1 numbers (1.9524×/0.9496×/etc., L18) | Read from the M6 report, which is generated from the recorded trials JSON (L17); trials not re-run this session | Settled evidence (v0.34.0 implemented report) |

## Quorum verification log — round 1

**Verdict: BLOCKED 2-of-2.** Both external reviewers PRESENT (`absent_reviewers` empty) — a
genuine block, not a degraded quorum. Neither objection disputed the design direction; both were
narrow refinements with reviewer-authored fixes, applied in full in this round-2 revision.
Cost: **$0.104328** total ($0.07285 gpt5-6-sol / $0.031478 gemini-3-1-pro); 26,331 tokens in /
663 out. Artifact under `.ailang/state/mission-quorum/`.

**gemini-3-1-pro — scan scope restricted to `internal/`+`cmd/`.** Controller measurement
verdict: **PARTIALLY REFUTED on today's count, CONFIRMED as a durable hole.** The reviewer was
right that round 1 never verified `tools/`/`examples/` were clean — L22 verified the spike's 7
refs but not that the rest of those trees held zero, and L5's grep filtered `tools/` out
entirely. The controller measured it (commands in L25): all 7 `.Elements` refs under `tools/`
are the spike's, `examples/` holds zero, with firing positive controls (31 `tools/` files with
`package `, 18 `examples/` `.go` files) and a clean fresh-literal negative control. So the
implied live miss does not exist today — but the structural objection stands: a scope
restriction is invisible to a ratchet by construction, and only a seeded addition proves the
gate looks. Fix applied anyway (scope `./...`, spike as package exemption, AC-12); the measured
zero delta is what makes it free.

**gpt5-6-sol — Rule 2 does not enforce its stated invariant.** Controller verdict:
**CONFIRMED**, first-party against the round-1 doc's own text (L26): Rule 1 class (c) exempted
"the accessor method bodies in `value_list.go`", and Rule 2 was scoped "outside the constructor
allowlist" — inheriting that set, so `l.Elements[0] = x` inside `At` was exempt from the
permanent mutation rule, and whole-function constructor exemption permitted
`tail.Elements[0] = head` inside `Cons`. The second half is equally real: Rule 1 (which retires
at LC-4) was the only rule reporting composite literals, leaving the retargeted cell fields
unprotected after the swap. Fix applied in full: Rule 2's exemption set is independent of
Rule 1's, write context is validated structurally, Rule 2 carries the composite-literal class
permanently, and the four fixtures land with their red conditions (AC-13).

## Quorum verification log — round 2 + NARROW-REFINEMENT CARVE-OUT (mission iteration 251)

**Round 2 — `2026-08-22T14:31Z` — SYNTHESIS: BLOCKED (rc=3).** Cost **$0.132845** (34,771 in /
638 out tok). **`absent_reviewers` is EMPTY and both external reviewers report `present=true`** —
a genuine 2-of-2 external block, not an N−1 degrade (the cap was raised to $0.35 pre-emptively
because the doc had grown from 44.6 KB to 61 KB in revision, which is exactly the condition under
which a reviewer drops out on `budget`). Artifact under `.ailang/state/mission-quorum/`
(untracked, `.gitignore:82` excludes `.ailang/`).

| Reviewer | Verdict | Cost |
|---|---|---|
| `gpt5-6-sol` | **reject** | $0.092745 |
| `gemini-3-1-pro` | **reject** | $0.040100 |
| controller (in-session, not an API call) | pass | $0 |

Cumulative quorum spend across both rounds: **$0.237173**.

### Disposition — NARROW-REFINEMENT CARVE-OUT (ratified iter-98), both premises MEASURED not forwarded

Both round-2 objections carry a concrete reviewer-authored `proposed_fix` and **neither disputes
the design DIRECTION** (the accessor seam, the `listrep` ratchet, C1). Gate 2's carve-out therefore
applies after the one re-quorum, and Gate 2 rule 3f applies first: the controller MEASURES a
premise objection rather than buying a revision round to answer a question one command settles.
Both were measured first-party at `684ebc23e`, and the two came out **opposite ways** — which is
the whole reason the rule exists.

**Objection 1 — `gemini-3-1-pro` — CONFIRMED EXACTLY, and it caught a defect the round-2 revision
had just introduced.**

> The round 2 revision adds 'configured-field composite-literal initializers' to Rule 2 (which is
> zero-tolerance and never ratcheted). However, L5 confirms there are 388 existing ListValue{...}
> sites at HEAD. Because LC-2 deliberately migrates no consumers, Rule 2 will unconditionally flag
> all 388 sites on day 1 and instantly break CI.

Measured (L27): **388** exactly, with the same-scope control firing (**14**) and the negative
control at **0**. The reviewer's number is right and its consequence is right. Note the shape:
round 2 added the class to close `gpt5-6-sol`'s post-LC-4 hole, and in doing so created a
day-one CI break — **two correct objections pointing opposite ways**, resolved not by choosing
between them but by putting the class where its cost is zero. `gemini-3-1-pro`'s `proposed_fix` is
applied verbatim: the class leaves Rule 2, Rule 1's ratchet tracks the 388 through LC-2/LC-3, and
**LC-4 adds it to Rule 2 when it retargets** — after the count is zero by construction. Recorded
as an explicit LC-4 obligation in Deferred Decisions, with AC-13's fixtures already exercising the
class under a simulated LC-4 config, so LC-4 inherits a proven fixture rather than a note.

**Objection 2 — `gemini-3-1-pro`, second half — PARTIALLY REFUTED by measurement.**

> the claim that Rule 2 'goes in green' based on L13 is an unverified premise: L13 explicitly
> filtered out test files (grep -v _test.go), but the analyzer's ./... scan includes them.

Correct that the doc had not verified it. Measured (L28), re-running L13 **without** the test
filter: **2** hits total, **0** in `_test.go`. The filter was hiding nothing, so neither of the
reviewer's two remedies is needed — no test-file triage declaration, and "Rule 2 goes in green"
stands, now on a measurement instead of a filtered grep.

**Objection 3 — `gpt5-6-sol` — CONFIRMED as an internal inconsistency; the fix is applied verbatim
and it makes the contract representation-independent.**

> The `NewList` contract is internally inconsistent and not representation-independent: it forbids
> callers from retaining or mutating the transferred slice, yet AC-3 deliberately performs that
> forbidden mutation and requires it to remain observable through the list. The doc then plans to
> change this pinned behavior at LC-4 when C1 builds cells. That makes a representation artifact
> part of the tested API and knowingly requires a behavioral contract change during the swap.

This is a defect the doc could not have measured its way out of — it is a contradiction between a
stated contract and the test pinning it, visible only by reading both. The reviewer's replacement
text is adopted **verbatim** as the contract, AC-3 is replaced with the reviewer's specified tests
(element order; no pre-transfer mutation by list operations; **no** post-transfer caller mutation),
and the HID/Design-Freeze rows follow. The zero-copy implementation LC-2 ships is unchanged; what
changed is that it is now an implementation fact rather than a pinned API promise, so LC-4's spine
build is not a contract change. The reviewer's conditional clause — *"If zero-copy migration safety
is still claimed, add a type-aware audit row classifying every intended constructor migration by
slice provenance and retained aliases"* — is **not triggered**, because the revised contract no
longer claims it; the provenance question is instead named in Unverified with LC-3 as owner.

**Carve-out conditions, checked explicitly:** every remaining blocking objection (a) carries a
concrete reviewer-authored `proposed_fix` — yes, all three, quoted above; and (b) does not dispute
the design DIRECTION — yes, all three are completeness/consistency/scope objections. No objection
was overridden and no controller-invented resolution was substituted: where a reviewer supplied
replacement wording it is used verbatim. This SATISFIES the objections; it is not force-passing.

**Status: quorum-cleared via the narrow-refinement carve-out. Routing: sprint-planner.**


## Related Documents

- [m-list-cons-cells-decomposition.md](m-list-cons-cells-decomposition.md) — parent roadmap; the
  LC-2 contract (lines 178–200), the grep-cannot-size finding (N4), the quorum constraints this
  doc inherits: analyzer retained permanently, escapes enumerated-not-migrated, analyzer-count
  denomination.
- [m-list-repr-spike.md](m-list-repr-spike.md) + the
  [M6 report](../implemented/v0_34_0/m-list-repr-spike-M6-report.md) — LC-1; source of the D3
  draft this doc finalizes, the two as-built divergences (constructors package-level; DropPrefix
  disposition), and C1's measured margins.
- [m-list-cons-quadratic.md](m-list-cons-quadratic.md) — superseded arena doc; evidence appendix
  (V32/V33 zero-aliased-writes base).
- Duplicate gate: L19 — neural search unavailable (SimHash fallback, keyword noise); the related
  set is exactly the programme's own docs above.

## References

- [#676](https://github.com/sunholo-data/ailang/issues/676) — the user-reported OOM this
  programme permanently fixes
- [#745](https://github.com/sunholo-data/ailang/issues/745) — carries `D-19 : B` (roadmap N1) and
  `D-22 : C1` (L2)
- [design_docs/PROGRAM.md](../PROGRAM.md) — routing lane: AILANG fix

---

**Document created**: 2026-08-22 (mission iteration 251)
**Revised**: 2026-08-22, same iteration — round 2, both round-1 quorum fixes applied in full
(scan scope `./...` + spike exemption + AC-12; Rule 2 structural/independent + AC-13)
**Authored at**: `684ebc23e` (= `origin/dev`), branch `sprint/iter251-list-accessor-api`
