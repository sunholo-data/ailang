# Sprint Plan — M-FMT-PRINTER-LINE-WIDTH-LIMIT

**Design doc**: [m-fmt-printer-line-width-limit.md](m-fmt-printer-line-width-limit.md) (575 lines, quorum rounds 1+2 BLOCKED-then-refined, both external reviewers present each round)
**Sprint ID**: `m-fmt-printer-line-width-limit`
**Target**: v0.35.0
**Planner**: mission loop iteration 285, SPRINT-PLANNER role
**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter285`
**Branch**: `sprint/iter285-fmt-width-limit`
**Base commit**: `2c8498886` (== `origin/dev` at spawn)
**Executor lane**: `codex:gpt-5.6-sol`, `workspace-write` sandbox — **no loopback sockets, no git writes**
**Estimate**: **3.5 days** (doc said 3; revised +0.5 — justification in §2)
**Risk**: medium

---

## 0. What the planner re-measured, and what it found

Per the mission skill's Gate-2 rule, every inherited claim was re-checked first-party in this
worktree at base `2c8498886` before a milestone was built on it. The design doc's V-log ran at
`f45d4f0fe`; HEAD has moved 18 commits, so the corpus numbers were **re-measured, not inherited**.

### 0.1 Doc claims CONFIRMED at this base

| Claim (doc) | Command run here | Observed |
|---|---|---|
| Corpus = 450 `.ail` files under `examples/` + `std/` | `find examples std -name '*.ail' \| wc -l` | `450` ✓ |
| >120 = 159, >100 = 282, max = 1315 @ `list_extremes.ail:24`, total 22,378 lines | `find … \| xargs -0 awk` (max/FNR + counts) | `max=1315 @ examples/runnable/list_extremes.ail:24`, `lines>120=159`, `lines>100=282`, `total=22378` ✓ **byte-for-byte identical to V2** |
| Let-chains are 20 of the 159 | `awk 'length>120 && / in let /'` over corpus | `20` ✓ |
| The catastrophic tail (>400) is **entirely** chains | `awk 'length>400'` + read each | 3 lines — `1315` `list_extremes.ail:24`, `496` `set_operations.ail:6`, `485` `directory_listing.ail:7` — all three begin `= let _ = … in let …` ✓ |
| Predicate has exactly THREE non-test call sites | `grep -rn hasAnyAttachment internal/format/*.go \| grep -v _test` | `decl.go:174`, `decl.go:572`, `expr.go:266` (+ def `format.go:181`) ✓ |
| `letChainMultiline` exists and is the multi-line emitter | `grep -rn letChainMultiline` + read `expr.go:290-330` | defined `expr.go:306`, called from `expr.go:267`, `decl.go:179`, `decl.go:577` ✓ |
| `decl.go:174` consults the predicate BEFORE `p.w.write(" =")` | read `decl.go:155-195` | predicate at `:174`, `p.w.write(" =")` at `:175` ✓ → **`pendingPrefix` is genuinely needed** (gpt5-6-sol's round-2 objection stands) |
| `newWriter` leaves `depth` zero; `write` emits indent `for i:=0;i<w.depth;i++` | read `doc.go:22-40` | exact ✓ → gemini-3-1-pro's double-count objection **refuted as stated**, invariant still worth pinning |
| No width concept anywhere in the printer | two-armed grep over non-test `internal/format/*.go` | arm1 (`MaxWidth\|maxWidth\|lineWidth\|exceedsWidth\|inlineWidth`) = **0 files**; **control** (`printer`) = **10 files** — the zero is a measurement, not a broken instrument ✓ |
| V10 two-armed repro: comment-free chain collapses; one interior comment → multi-line | wrote both fixtures, `ailang check` + `ailang fmt` with a freshly built `bin/ailang` | arm A: rc=0, one **88-char** line; arm B: rc=0, continuation layout, max **33** chars ✓ |
| V24 depth-60 value-position nested chain parses and collapses | generated depth-60 file, `ailang check`, `ailang fmt` | `✓ No errors found!` rc=0; `fmt` → one **1069-char** line ✓ — the AC11 fixture is real |
| Corpus comment total = 7865 | `lexer.CollectComments` over all 450 files | `files=450 comments=7865` ✓ **exact match** |
| Attach-refusals / parse-failures fail closed and stay untouched | per-file `ailang fmt --check` over all 450 | 405 canonical / **38 rc=2 attach-refusal** / **7 rc=3 parse-failure** ✓ |
| `go build ./...` is rc=1 on pristine dev — do not gate on it | `go build ./...` | `# …/cmd/wasm: function main is undeclared in the main package`, **rc=1** ✓ the doc is right to exclude it |

### 0.2 Doc claims the planner **REFUTED** (with the refuting command)

Four. None touches the design direction; all are AC-level defects that would have produced a
vacuous pass. **The doc wins as the reviewed artifact**, so each is carried as a *correction* to
the AC's wording, not a change of intent.

**R1 — AC12's constant `88` is the wrong number for `inlineWidth`.**
AC12 says "`inlineWidth` returns exactly the rune count of its one-line rendering (88)". But
`inlineWidth(node)` renders the **node**, and the node here is the chain, not the whole decl line.

```
$ ailang fmt /tmp/i285v10/b.ail | awk '/^export func main/ {…decompose at " = "…}'
LINE=88
SIG=31  [export func main() -> () ! {IO}]
PENDING=3
BODY(inlineWidth of chain node)=54  [let a = 1 in let b = 2 in let c = 3 in println("done")]
check: 31+3+54=88
```

`inlineWidth(chain) = 54`; **88 is `p.w.col + pendingPrefix + inlineWidth`**. Asserting `88` on
`inlineWidth` would fail a *correct* implementation. **Correction**: AC12 asserts BOTH
`inlineWidth(chain) == 54` AND the full formula `31 + 3 + 54 == 88`. The second is strictly
stronger than the doc's version and preserves its 88.

**R2 — AC10's "(expected zero)" is wrong; the count is 1.**
The active teaching prompt is `prompts/v0.16.6.md` (proved: `ailang prompt --source=embedded`
output is byte-identical to that file — `diff -q` matched, 92,749 bytes).

```
$ grep -c 'in let ' /tmp/i285_prompt.txt      →  1
$ sed -n '22p' /tmp/i285_prompt.txt | awk '{print length($0)}'  →  112
```

Line 22 teaches `func f() ! {IO} = let x = a in let y = b in finalExpr`. It is **112 chars < 120**,
so the new predicate leaves it alone and it does **not** contradict the new canonical layout — but
the doc's prediction of zero hits is false, and an executor that greps, sees `1`, and trusts the
parenthetical will either "fix" a correct line or silently drop the AC. **Correction**: AC10 is
restated as a width-qualified check that can fail (below), with `1` as the recorded baseline.

**R3 — `make check-file-sizes` is RED at base. It must not appear in any AC.**

```
$ make check-file-sizes
✗ cmd/ailang/eval_suite.go:      826 lines (exceeds 800)
make: *** [check-file-sizes] Error 1
```

The doc does not cite it, but it is a standing repo gate and an executor may add it reflexively. A
gate already red at base measures the repo, not the change. **Correction**: the sprint uses a
**scoped** file-size assertion over `internal/format/` only (largest file today: `decl.go` 583).

**R4 — `Options.MaxWidth` must be resolved at TWO printer construction sites, not "printer
construction" singular.**
The doc's M0 says "resolved once at printer construction". There are two:

- `format.go:94` — `Source(...)`: `p := &printer{w: newWriter(indent)}`
- `format.go:131` — `SourceWithComments(...)`: `p := &printer{w: newWriter(indent), att: …}`

`cmd/ailang/fmt.go:130` calls `SourceWithComments` — **the CLI path is the second one**. Resolving
only in `Source` leaves the entire CLI on `maxWidth == 0`, i.e. every line "exceeds", i.e. total
corpus explosion; resolving only in `SourceWithComments` leaves the comment-free unit tests on a
different width than the CLI. **Correction**: M0 AC requires the resolution in a single helper
called by both, plus an assertion that both entry points produce identical output for
comment-free input (the byte-identity invariant already documented at `format.go:181`).

### 0.3 Planner-added hazard (NOT in the doc): `col` must count RUNES

`inlineWidth` is specified in **runes** ("the rune count of the rendering's FIRST line"). `writer.col`
is NEW in M0 and the doc leaves its unit unspecified. The natural Go implementation —
`w.col += len(s)` alongside `w.buf.WriteString(s)` — counts **bytes**. The predicate then mixes units.

This is not hypothetical on this corpus. Two-armed scan (the `-P` arm returned 0 with a *firing*
control, so it was discarded as a broken instrument and re-run byte-wise under `LC_ALL=C`):

```
$ find examples std -name '*.ail' -print0 | xargs -0 env LC_ALL=C grep -l '[^ -~\t]' | wc -l
140                                   # 140 of 450 files carry non-ASCII
$ … grep -ln '"[^"]*[^ -~\t][^"]*"' | wc -l
18                                    # 18 files carry non-ASCII INSIDE string literals
$ … awk 'length($0)>120' | grep -c '[^ -~\t]'
3                                     # 3 of the 159 over-long lines are non-ASCII
```

And there is a ready-made fixture already in the corpus — a let chain in an equation body:

```
examples/string_replace.ail:19
pure func utf8FindSubstring() -> string = let s = "café world" in let pos = find(s, "world") in substring(s, pos, length(s))
bytes=125  runes=124
```

**AC15 (planner addition, labelled as such)** pins the unit. See §4.

---

## 1. Milestones

Four, exactly as the doc names them. The doc's own round-count note anticipates splitting M0 out if
a third quorum round blocks on it; M0 is already the standalone first milestone, so no restructuring
is needed. Milestone order is also the **sharpest bisect order**: M0 adds a mechanism nothing calls,
M1 wires it at the three chain sites, M2 wires the two non-chain arms, M3 moves 450 files.

| # | Name | Impl LOC | Test LOC | Days | Closes |
|---|---|---|---|---|---|
| **M0** | Width plumbing + the mode-locked measurement printer | ~150 | ~250 | 1.0 | AC11, AC12, AC14, AC15 |
| **M1** | Width-widened chain predicate at all three sites | ~25 | ~200 | 0.75 | AC1, AC2, AC13 |
| **M2** | Continuation layout for long equation bodies | ~50 | ~160 | 0.75 | AC13 (non-chain arms) |
| **M3** | The second corpus reformat + evidence | ~140 (tooling) | — | 1.0 | AC3, AC4, AC5, AC6, AC8, AC9, AC10 |
| | **AC7 (suites) is asserted at EVERY milestone boundary** | | | | AC7 |
| | **Total** | **~365** | **~610** | **3.5** | |

Plus the corpus rewrite itself (M3) — a mechanical diff over ~400 files, not counted as LOC.

### 2. Why 3.5 days, not the doc's 3

The doc's estimate predates quorum round 2. Round 2 added `pendingPrefix` (a per-site API surface)
and two new failure-capable ACs (AC13, AC14) with named mutation arms; this plan adds AC15 and the
two-entry-point resolution of R4. Test LOC in this repo runs ≈70% of impl LOC, and the mutation arms
are **execution** work, not just assertion work — each mutant must be applied, proven applied, its
result read, and reverted byte-identical. M0 alone carries three mutation arms. **No AC is cut**;
+0.5 day lands on M0, which grows from the doc's 0.5 to 1.0.

---

## 3. Day-by-day

### Day 1 — M0: width plumbing and the measurement printer

**All new code lands in a NEW file `internal/format/width.go`.** Rationale: `decl.go` is 583 lines,
`expr.go` 571, `format.go` 536; a new file keeps every file under the 800-line ceiling and gives the
evaluator a single place to read the mechanism. Edits to existing files in M0 are only: the two
struct definitions, the two construction sites, and `doc.go`'s stale package comment.

1. **`Options.MaxWidth int`** in `format.go` (currently `Options` has only `Indent`).
   Zero value → 120 via one unexported helper `resolveMaxWidth(o Options) int`.
2. **Call `resolveMaxWidth` at BOTH construction sites** — `format.go:94` (`Source`) and
   `format.go:131` (`SourceWithComments`). Per R4 this is the CLI-correctness point.
3. **`writer.col int`** in `doc.go`. Maintained by `write` (**including the indentation it emits**
   when `atBOL`), zeroed by `hardline`. **Counted in RUNES** — `utf8.RuneCountInString`, never
   `len` (R-hazard §0.3).
4. **`printer.maxWidth int` and `printer.measuring bool`** (new fields).
5. **`pendingPrefix`** as a named per-site constant set in `width.go`, part of the M0 API so a new
   site cannot silently omit it: `prefixLetIn = 0`, `prefixEquationBody = len(" = ")`,
   `prefixTopLevelLetValue = len(" = ")`.
6. **`newMeasurementPrinter(p *printer) *printer`** — the ONLY construction site of a measurement
   printer, and it lives inside `exceedsWidth`. Mode table, enforced field-by-field:
   `w: newWriter(p.w.indent)` with **`depth` left at its zero value — never seeded from the parent**
   (the AC14 invariant); `att: nil` (never `p.att`); `measuring: true`; `maxWidth` not consulted.
   Add a one-line comment at the site naming `interp.go:160` (`holeText`) as the precedent that does
   the OPPOSITE deliberately, so the next reader does not "fix" this by copying it.
7. **`inlineWidth(n ast.Expr) int`** — render `n` with the measurement printer, return the rune
   count up to the first `\n` (or of the whole output when there is none).
8. **`exceedsWidth(n ast.Expr, pending int) bool`** — returns `false` **unconditionally and before
   constructing anything** when `p.measuring`; otherwise
   `p.w.col + pending + p.inlineWidth(n) > p.maxWidth`.
9. **Depth hook**: a package-level test hook that **fails fast the instant a measurement printer is
   constructed at nesting depth 2**, rather than recording a max. Fail-fast is load-bearing: under
   AC11's `measuring=false` mutant the depth-60 fixture blows up geometrically, so a
   record-the-max hook would hang the suite instead of failing it.
10. **Fix `doc.go`'s package comment** — it currently claims the writer "tracks the current
    indentation depth and column" and that "soft-line and group primitives are provided". Column
    tracking becomes true in this commit; soft-line/group remain absent. Same commit, per the doc.
11. **Tests** (`internal/format/width_test.go`): column tracking incl. the indent-emitting first
    write; the same chain fitting at col 0 and not at col 40; AC11; AC12; AC14; AC15.

**M0 exit**: `.snap/M0/` written; gates in §5 green.

### Day 2 (first half) — M1: the chain predicate at all three sites

One-line changes, three sites, one meaning — the systemic form (audit-before-patch: V4 enumerates
the sites and the planner re-confirmed there are no others).

- `expr.go:266` → `if n.Body != nil && (p.hasAnyAttachment(n) || p.exceedsWidth(n, prefixLetIn))`
- `decl.go:174` → `… && (p.hasAnyAttachment(let) || p.exceedsWidth(let, prefixEquationBody))`
- `decl.go:572` → `… && (p.hasAnyAttachment(val) || p.exceedsWidth(val, prefixTopLevelLetValue))`

**Attachment is checked FIRST** (Go `||` short-circuits) so no attached case can regress to inline —
comment behaviour is untouched by construction, and the short-circuit also means `exceedsWidth` is
never even called on attached nodes, which is a free performance property.

Fixture churn is expected and legitimate: `inline_interior_test.go` /
`inline_interior_shape_test.go` pin byte-exact output for the attached-chain feature. Any fixture
whose **comment-free** arm renders wider than 120 flips deliberately. **Testing policy: update the
fixture, never weaken the assert** (`.claude/rules/coding-standards.md`). Every flip must be listed
in `.snap/M1/FLIPS.md` with old/new widths — an unexplained flip is a defect, not churn.

`TestFmtOutputMatchesTaughtDialect` (`format_test.go:326`) must stay green **unmodified** — its
snippets are short; red there means the width change leaked into the dialect.

### Day 2 (second half) — M2: continuation layout for long equation bodies

In `funcBody` (equation form) and `topLevelLet`, the **non-chain** arms. Predicate order makes the
chain/non-chain split explicit: the M1 guard is evaluated first, and M2's branch is its `else`, so a
wide chain body takes M1's `letChainMultiline` and never M2's generic continuation.

```go
// funcBody, len(blk.Exprs)==1, after the M1 chain branch:
if p.exceedsWidth(blk.Exprs[0], prefixEquationBody) {
    p.w.write(" =")
    p.w.hardline()
    var err error
    p.w.indented(func() { err = p.expr(blk.Exprs[0], precLowest) })
    return err
}
p.w.write(" = ")
return p.expr(blk.Exprs[0], precLowest)
```

Same shape in `topLevelLet` for `d.Value` when `d.Body == nil`. (`d.Body != nil` returns to
`p.letIn(d)` at `decl.go:558` before anything is written, so that path is owned by `expr.go:266`.)

### Day 3 — M3: the second corpus reformat

Executed with iteration 282's evidence discipline reproduced in full. Steps 1, 2, 4, 5 are
executor-runnable; step 3's `make verify-examples` is a **CONTROLLER** gate (§6).

1. Build a fresh ldflags-stamped binary to a scratch dir, PATH-prepend it. **Never
   `make quick-install`** — this rig is shared and concurrent agents depend on `~/go/bin/ailang`.
2. `ailang fmt --write` over all 450 files. Files that fail closed (38 rc=2 attach-refusals,
   7 rc=3 parse-failures, measured at base) stay untouched — assert that their bytes are unchanged.
3. Comment totals before/after via `lexer.CollectComments`, per-file join. Baseline **7865**.
   **The poisoned control arm must FIRE**: mutate one file's comment count, show the instrument
   reports a mismatch, revert. A control that does not fire is not a control.
4. `ailang check` rc unchanged on every joined pair.
5. Width metrics before/after: >120 count, >100 count, max line, **and the per-construct residual
   classification** using the same classifier as V11 — that table is AC8 and is the sole input to
   the follow-on reflow-engine decision.
6. Second `fmt --write` pass → `git diff --stat` must be **empty** (AC4).

---

## 4. Acceptance criteria — baselined, mapped, and mutation-checked

Every command below was **run on the pristine tree at `2c8498886`** and its result recorded. Anything
already red at base is disqualified as a gate.

### 4.1 Gate baselines (pristine tree)

| Command | Base result | Usable as a gate? |
|---|---|---|
| `go build ./internal/format/...` | **rc=0** | ✅ narrowest gate that can fail for this diff |
| `go build ./...` | **rc=1** (`cmd/wasm`: `function main is undeclared`) | ❌ **disqualified** |
| `go vet ./internal/format/...` | rc=0 | ✅ |
| `go test ./internal/format/...` | rc=0, `ok … 55.143s` | ✅ (V15 said 43s; 55s here — budget 3 min) |
| `gofmt -l internal/format/` | empty output | ✅ |
| `golangci-lint run ./internal/format/...` | rc=0, `0 issues.` | ✅ |
| `make verify-examples` | **rc=0** | ✅ but **CONTROLLER-ONLY** — needs live network (§6) |
| `make verify-stdlib` | rc=0, `✓ All 45 stdlib interfaces stable` | ✅ |
| `make test-stdlib-ail` | rc=0, `✓ stdlib .ail suites and run-fixtures pass` | ✅ |
| `make check-file-sizes` | **rc≠0** (`cmd/ailang/eval_suite.go: 826`) | ❌ **disqualified (R3)** |
| `wc -l internal/format/*.go` max | 583 (`decl.go`) | ✅ scoped substitute for the above |

### 4.2 Can each AC fail? (requirement: no vacuous passes)

| AC | Milestone | Baseline observed NOW (so it can fail) | Kills which mutation, via which observable |
|---|---|---|---|
| **AC1** predicate two-armed | M1 | `fmt` on the V10 fixture emits the 88-char one-liner **regardless** — arm 1 fails today | Mechanism: the `\|\| p.exceedsWidth(…)` disjunct. Observable: emitted bytes (inline vs multi-line). **Downstream** of the branch, not set alongside it. Arm 2 (large `MaxWidth` → byte-identical to today) is the byte-identity control and fails if the predicate is unconditional. |
| **AC2** tail kill | M1 | `ailang fmt examples/runnable/set_operations.ail` → `max=496, lines>120=1`; `list_extremes.ail` → `max=1315`. Required: `set_operations` no line >120, `list_extremes` max ≤150 | Same mechanism/observable as AC1 on committed corpus files. Both targets confirmed by the planner to be `let`-chains (they are 2 of the 3 lines >400), so **M1 alone can close this** — no dependence on M2. |
| **AC3** corpus reduction | M3 | `lines>120 = 159`, `max = 1315`. Required: ≤105 and ≤350 | Fails loudly if the predicate is unwired: the values simply stay 159/1315. |
| **AC4** idempotence | M3 | n/a (post-M3 property) | Observable: `git diff --stat` after a second `fmt --write` = empty. |
| **AC5** comment safety | M3 | `files=450 comments=7865` | Observable: the before/after total. **The poisoned-control arm is what makes this non-vacuous** — a join that silently compares nothing also reports "unchanged". |
| **AC6** semantics | M3 | 405 canonical / 38 rc=2 / 7 rc=3 | Observable: per-file `ailang check` rc, joined pairwise. |
| **AC7** suites | M0/M1/M2/M3 | all green at base (§4.1) | Green-at-base is what makes a later red attributable to the diff. |
| **AC8** residual table | M3 | V11 classifier reproduced: `FUNC_DECL_ONELINE 62 / LET_SINGLE 23 / LET_CHAIN_2PLUS 20 / …` sums to 159 | Not pass/fail — a required artifact. Its absence fails the milestone. |
| **AC9** no gate wiring (negative) | M3 | `internal/cihygiene/gate_wiring_test.go` sha256-prefix `8e805c026…` | Observable: that file's hash unchanged + `git diff` touches no `.github/workflows/` and no `make/ci.mk` gate list. A negative AC with a recorded baseline hash **can** fail. |
| **AC10** teaching surface | M3 | **`grep -c 'in let ' prompts/v0.16.6.md` = 1** (not 0 — R2). The hit is 112 chars | **Restated so it can fail**: every `prompts/` occurrence of a collapsed chain whose rendered width **exceeds 120** must be replaced with an `ailang check`-verified multi-line spelling. Baseline: 1 occurrence, width 112, **0 requiring change**. Recording the 1 is what distinguishes "checked and clear" from "grep silently returned nothing". |
| **AC11** bounded measurement | M0 | mechanism does not exist yet | Mechanism: `exceedsWidth` returns `false` when `measuring`. Observable: measurement-printer **construction nesting depth**, asserted never to reach 2, over (a) the full 450-file corpus and (b) the committed depth-60 fixture (**expressibility re-verified: `check` rc=0, `fmt` → one 1069-char line**). Downstream: the counter observes the *consequence* of the short-circuit (a construction that should not happen), not the flag itself. **Mutation arm**: build the measurement printer with `measuring: false` → must trip the depth assertion. Fail-fast at depth 2 is required so the mutant **fails** rather than hanging. Wall-clock ceiling: sub-second on the fixture. |
| **AC12** measurement correctness | M0 | mechanism does not exist yet | Mechanism: `maxWidth` unconsulted while measuring. Observable: the integer `inlineWidth` returns. **CORRECTED per R1** — assert `inlineWidth(chain) == 54` AND `col(31) + pendingPrefix(3) + 54 == 88`. Identical under parent `MaxWidth` 40 and 120; if the measurement printer wrapped, the 40-arm would differ. Second case: a chain >120 wide returns the un-wrapped inline width, not the length of any wrapped block. |
| **AC13** `pendingPrefix` boundaries | M1 (+M2) | mechanism does not exist yet | Mechanism: `pendingPrefix(site)` ∈ {0, 3, 3}. Observable: inline-vs-multi-line at exactly `MaxWidth` and at `MaxWidth+1`, per site = a **6-bit vector**. A single fixture's binary observable is coarser than the mechanism's value set, which is precisely why the AC needs all six points plus the **asymmetry**. **Two mutation arms, not one** — the doc names only the first: (a) all prefixes → `0` must flip the two decl sites and must **NOT** flip `letIn`; (b) **planner addition** — all prefixes → `3` must flip `letIn` and must NOT flip the decl sites. Without arm (b) a constant-3 implementation survives, and "site-specific rather than a constant fudge" would be unproven in the other direction. M2 extends the same six points to the non-chain arms of the two decl sites. |
| **AC14** zero measurement depth | M0 | `newWriter` leaves `depth` zero (`doc.go:22`); `write` indents `for i:=0;i<w.depth;i++` (`doc.go:34`) — re-read first-party | Mechanism: the measurement writer's `depth` field. Observable: the integer `inlineWidth` returns, with the parent at `depth` 0 vs `depth` 4. Downstream (depth → indent emission → first-line rune count → return value). **Mutation arm** = literally the implementation gemini-3-1-pro predicted (`depth: p.w.depth`): the two values must then differ by exactly `4 × len(indent)` = **8**. Correct arm: exactly equal. |
| **AC15** width is measured in RUNES | M0 | **PLANNER ADDITION** — 140/450 files carry non-ASCII; 18 carry it inside string literals; 3 of the 159 over-long lines are non-ASCII; `examples/string_replace.ail:19` is a let chain at **125 bytes / 124 runes** | Mechanism: `writer.col` and `inlineWidth` both counting runes. Observable: the predicate's inline/multi-line outcome on a paired fixture — an ASCII body of exactly `MaxWidth` runes, and the same body with one ASCII char swapped for `é` (same rune width, +1 byte). Both must stay inline. A byte-counting `col` wraps the second and not the first. Downstream of the counter; the observable's value set (binary per fixture, 2 fixtures) is matched to the mechanism's (one unit choice). |

---

## 5. Per-milestone exit gates (what the executor runs)

Run **in this order** at every milestone boundary. All are green at base, so any red is attributable
to the diff.

```bash
gofmt -l internal/format/                      # must print nothing
go build ./internal/format/...                 # rc=0   (NEVER `go build ./...` — rc=1 at base)
go vet ./internal/format/...                   # rc=0
go test ./internal/format/... -count=1         # rc=0   (~1 min; budget 3 min)
golangci-lint run ./internal/format/...        # "0 issues."
wc -l internal/format/*.go | sort -rn | head -3   # no file >800 (max at base: decl.go 583)
```

Additionally at **M2** and **M3**:

```bash
make verify-stdlib        # rc=0 at base
make test-stdlib-ail      # rc=0 at base
```

**`make verify-examples` is NOT in this list** — see §6.

---

## 6. Sandbox reality: what the codex lane can and cannot do

The executor is `codex:gpt-5.6-sol` under `workspace-write`. **No loopback sockets, no git writes.**

| Gate | Lane | Evidence |
|---|---|---|
| `gofmt` / `go build ./internal/format/...` / `go vet` / `go test ./internal/format/...` | **EXECUTOR** | pure compute + repo-local temp; no sockets |
| `golangci-lint run ./internal/format/...` | **EXECUTOR** | local binary at `~/go/bin/golangci-lint`, rc=0 at base |
| `make verify-stdlib`, `make test-stdlib-ail` | **EXECUTOR (attempt), CONTROLLER (confirm)** | run `./bin/ailang` over `std/` + `tests/stdlib`; pure by inspection, but they do execute AILANG, so the controller re-runs before the milestone is accepted |
| `make verify-examples` | **CONTROLLER ONLY** | **Proven to make live outbound HTTPS.** `examples/runnable/http_simple.ail` is `status=working, skip=false` in `examples/manifest.json` and is NOT on `scripts/verify_examples.go`'s `skippedExamples` list, so `verify-examples` runs it. Direct control: `./bin/ailang run --caps IO,Net examples/runnable/http_simple.ail` → `Fetching: https://httpbin.org/get` / `Response length: 265 bytes`. Seven examples carry the `Net` effect. In a no-network sandbox this gate goes red for reasons unrelated to the diff. |
| M3's `fmt --write` over 450 files | **EXECUTOR** | repo-local writes only |
| M3's fresh ldflags-stamped build to a scratch dir | **EXECUTOR** | `go build` + `$TMPDIR`; **not** `make quick-install` (shared rig) |
| `git add` / `commit` / `branch` / `stash` / `checkout` | **CONTROLLER ONLY** | executor performs **no** git writes |

---

## 7. Snapshot protocol (mandatory — the sandbox blocks committing)

The executor cannot commit, so each milestone's state must survive as files.

After each milestone `k` ∈ {M0, M1, M2, M3}, write **`.snap/M<k>/`** containing the **cumulative**
post-milestone content of **every file created or modified so far in this sprint** — not a delta.
Mirror the repo-relative path inside the snapshot dir, e.g.:

```
.snap/M1/internal/format/width.go
.snap/M1/internal/format/doc.go
.snap/M1/internal/format/format.go
.snap/M1/internal/format/expr.go
.snap/M1/internal/format/decl.go
.snap/M1/internal/format/width_test.go
.snap/M1/FLIPS.md          # M1 only: every deliberately-updated byte-exact fixture, old→new width
.snap/M1/GATES.md          # the §5 command list with each observed rc, verbatim
```

`M2` repeats every M0+M1 file at its M2 content, and so on. `GATES.md` in each dir is required —
a snapshot without recorded gate output is not evidence.

`.snap/` is a scratch directory for the controller to read; the controller decides what is committed.

---

## 8. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| `col` implemented in bytes | **high** (it is the natural Go spelling) | AC15 + the `string_replace.ail:19` fixture |
| `MaxWidth` resolved at only one of the two construction sites | **high** (the doc says "printer construction", singular) | R4: one shared helper + a both-entry-points byte-identity assertion |
| Measurement writer seeded with `depth: p.w.depth` | medium (the "make it resemble its parent" instinct; gemini-3-1-pro predicted it) | AC14 with the exact-8 mutation arm |
| Byte-exact fixture churn read as breakage | medium | `.snap/M1/FLIPS.md` with old→new widths; policy is update-the-fixture, never weaken |
| AC11's mutation arm hangs instead of failing | medium | depth hook **fails fast at depth 2**, plus a sub-second wall-clock ceiling |
| The write-hook (`scripts/hooks/format_ail.sh:69`) starts emitting the new layout in concurrent agent sessions the moment M1 lands | **certain**, benign | Schedule M3 immediately after M2 — same sprint, same day |
| `verify-examples` red in the sandbox for network reasons | high if attempted | Controller-only gate (§6) |
| M3 rewrites so many files that a real regression hides in the diff | medium | The 38 rc=2 + 7 rc=3 fail-closed files must be **byte-unchanged** — a cheap, sharp invariant over 45 of the 450 |
| Manifest drift turning `verify-examples` red | low (reformatting does not change imports) | Controller re-runs; known repo failure mode |

---

## 9. Explicitly out of scope (from the doc's Non-Goals — do not drift into these)

No general reflow engine (groups/soft-lines — `doc.go`'s comment notwithstanding, **they do not
exist**); no string-literal splitting, ever; no comment rewrapping; no import-list wrapping (new form
+ new attach list = the iteration-281 silent-comment-loss hazard class); no type-decl body layout;
no function-signature wrapping; **no fmt gate wiring or freeze** (`D-39` sequencing, AC9); no CLI
`--width` flag.

---

## 10. Note for the controller when committing

`.ailang/state/sprints/` is matched by `.gitignore:82` (`.ailang/`). The 59 sprint JSONs already
tracked there were force-added, so the ignore rule only affects the **new** file. Committing this
sprint's JSON therefore requires:

```bash
git add -f .ailang/state/sprints/sprint_m-fmt-printer-line-width-limit.json
```

A plain `git add .` will silently skip it — verified: `git check-ignore -v` on the new file returns
`.gitignore:82:.ailang/`, and `git status --porcelain` does not list it.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_35_0/m-fmt-printer-line-width-limit-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_m-fmt-printer-line-width-limit.json` (needs `git add -f`)
