# Sprint Plan — `m-fmt-unpinned-hunks` (iteration 288)

**Rows covered (both, in one sprint):**
1. `m-fmt-measurement-att-isolation-unpinned` — `internal/format/width.go:36` `att: nil`
2. `m-fmt-measurementerr-propagation-no-killer` — `internal/format/format.go:105-107,149-151`

**Worktree:** `/Users/voightkampff/dev/sunholo-data/.wt-iter288` (branch `sprint/iter288-fmt-unpinned-hunks`, base `9f089d3ebfc10b4cacdbd063aa8ea826efc88734`)
**Executor:** `codex:gpt-5.6-sol`, `workspace-write`, **NO git write.** Do not commit, do not branch, do not `git checkout --`. Restore mutated files by `cp` from a backup taken before mutating.
**Milestones:** 3. Snapshot each into `.snap/M<k>/`.

---

## 0. Planner reconnaissance — MEASURED THIS SESSION (re-verify anything you lean on)

All on the pristine worktree; `internal/format/width.go` sha256 `8a0982d1cab0d8f35ae7f2a4defa453fbf435a4f65d26db32ca4977903747758`, `git diff` empty before and after.

| # | Fact | How measured |
|---|---|---|
| P1 | A white-box printer built with a populated `att` index and asked for `inlineWidth(chain)` returns the **isolated** width; a hand-built *inheriting* shadow printer returns a **different** width. Fixture A: isolated `27`, inherited `12`. Fixture B (trailing comment): isolated `27`, inherited `30`. | temp test, `newAttachIndex` + `expr(n, precLowest)` |
| P2 | **Row (1) is latent, not live — now measured at the measurement site.** Corpus differential over `examples/` + `std/` (450 files, 443 parse-clean, 405 formatted, 38 fail-closed): **88 measurements, 0 divergent**. | temp `inlineWidthProbe` hook in `inlineWidth`, reverted |
| P3 | **P2 is NOT vacuous.** Of those 88 measurements, **82 ran with a POPULATED attachment index** (`attNil=0 attEmpty=6 attPOPULATED=82`). So real comments were in the index and inheriting them still changed nothing. | same sweep, att-population counter |
| P4 | **Second, independent mask (new — not in the iter-287 framing).** Comments owned by *nested interior* nodes are **not representable at all**: `AttachComments` fails closed on all 6 placements tried (list / record / tuple / if / lambda interiors, nested-let body) with `comment at byte N could not be attached to any boundary`. So the V6 caveat (exact-owner `hasAnyAttachment` does not recurse) cannot currently be turned into a divergence, because the offending attachment cannot exist. | temp test, 6 fixtures |
| P5 | **Row (2) redundancy is now MEASURED, not reasoned.** Fault injection of `&ast.Error{}` into every `Let.Value` and `Let.Body` across the corpus at widths {120, 40, 20} = **312 injections**. Joint cell: `measErr+fileOK = 0` (**the live cell is EMPTY**), `measErr+fileErr = 312`, `noMeasErr+fileErr = 0`. | temp test calling `p.file()` directly, inspecting `p.measurementErr` |
| P6 | **P5's instrument is alive.** The same sweep on the *pristine* corpus gives `1329` renders with `measErr` set **0** times and `file()` erroring **0** times; injection moves all 312 into `measErr+fileErr`. So the empty live cell is a measurement, not a dead pipeline. | same test |
| P7 | `interp.go:167-169` **discards** a `sub.expr` error without setting `p.measurementErr`; only `interp.go:170-171` propagates, and it requires `sub.expr` to *succeed* with `measurementErr` set — the same empty cell as P5. | code read, consistent with P5 |

**Conclusion carried into the plan:** row (1) = *unpinned invariant, latent, double-masked* → **kill the mutant with a targeted unit test + add a tripwire for when the masks lift.** Row (2) = *measured-redundant* → **declare it, and pin the neighbouring live hunk (`width.go:46`) that the same sweep does kill.**

---

## 1. Acceptance-command baselines (run by the planner on the pristine tree)

| Command | Baseline | Usable as a gate? |
|---|---|---|
| `go build ./internal/format/` | **rc=0** | YES |
| `go vet ./internal/format/` | **rc=0** | YES |
| `gofmt -l internal/format` | **rc=0, empty output** | YES |
| `go test ./internal/format/` | **rc=0, 0 FAIL, ~21s** | YES |
| `go build ./...` | **rc=1** (cmd/wasm, gen/main have no native main) | **NO — red at base** |
| `make check-file-sizes` | **rc=2** | **NO — red at base** |
| `go test ./internal/format/ -run TestDoesNotExist` | **rc=0, `[no tests to run]`** | **NO — VACUOUS TRAP** |

> **TRAP, measured:** `go test -run <Name>` is **rc=0 when `<Name>` does not exist**. Every per-test AC below therefore uses
> `go test ./internal/format/ -run '<Name>' -v 2>&1 | grep -c -- '--- PASS: <Name>'`
> which is **rc=1** when the test is missing or red. Baselined: on the pristine tree that form returns `0` / rc=1 for each new test name. Do not substitute a bare `-run`.

---

## 2. Named mutants (referenced throughout)

| ID | Mutation | Base status |
|---|---|---|
| **A** | `width.go:36` `att: nil` → `att: p.att` | survives (0 FAIL) — parent V2 |
| **B** | `format.go:105,149` `if p.measurementErr != nil {` → `if false && p.measurementErr != nil {` (both) | survives (0 FAIL) — parent V3 |
| **C** | `width.go:46` `p.measurementErr = err` → delete the assignment (keep `return 0`) | **unknown — M3 must measure** |

---

## M1 — Kill mutant A, and tripwire the masks

**File:** new `internal/format/width_att_isolation_test.go`

**T1 `TestMeasurementIgnoresInheritedAttachments`**
Build from real source (do **not** hand-roll an `attachIndex`): `mustParse` → `NewEnvelope` → `AttachComments` → `newAttachIndex`. Fixtures, both verified to attach (P1):
- A: `"module m\nfunc f() =\n  let a = 1 in\n  -- inner note\n  let b = 2 in\n  b\n"`
- B: `"module m\nfunc f() =\n  let a = 1 in -- trailing note\n  let b = 2 in\n  b\n"`

For each fixture compute three widths on `chain := prog.File.Funcs[0].Body.(*ast.Block).Exprs[0]`:
- `reference` — `(&printer{w: newWriter("  "), maxWidth: 120}).inlineWidth(chain)` (att nil; **mutation-invariant**, since under A `p.att` is itself nil here)
- `isolated` — `(&printer{w: newWriter("  "), att: idx, maxWidth: 120}).inlineWidth(chain)`
- `inheriting` — an explicitly hand-built `&printer{w: newWriter("  "), att: idx, measuring: true, measurementDepth: 1}`, `expr(chain, precLowest)`, first line, `utf8.RuneCountInString`

Assertions:
1. **KILLER:** `isolated == reference`. (Base 27==27. Under **A**: 12 vs 27 / 30 vs 27 → RED.)
2. **CONTROL (anti-vacuity, mutation-invariant):** `inheriting != reference` — proves the fixture carries an observable comment, so assertion 1 is measuring something. Fail with an explicit "fixture no longer carries an observable attachment" message.
3. **CONTROL:** `len(atts) > 0`.

Assert *relations*, never the literals 27/12/30.

**T1b `TestNestedInteriorCommentsDoNotAttach`** — the tripwire for P4. Two fixtures from P4 (list-interior, if-interior). Assert `AttachComments` returns a **non-nil error** for each. Header comment must say, in the code: *this test exists so that widening the attachment boundary set (M2/M3) goes red here and forces re-checking `m-fmt-measurement-att-isolation-unpinned`, because nested-interior ownership is the one shape that could defeat the exact-owner `hasAnyAttachment` gate.*

**Also in M1:** replace the comment at `width.go:35` with a declared-scope comment stating (a) the isolation invariant, (b) that it is pinned by `TestMeasurementIgnoresInheritedAttachments`, (c) that at product level it is currently **double-masked** — the `hasAnyAttachment(X) || p.exceedsWidth(X, …)` short-circuit at `expr.go:266`, `decl.go:174`, `decl.go:572`, **and** the fail-closed attachment boundary set (P4) — with the measured evidence `88 measurements / 82 with a populated index / 0 divergent`, and (d) that M2's continuation layout must re-run that differential.

**AC-M1** (each must be run and its output recorded):
```
cd /Users/voightkampff/dev/sunholo-data/.wt-iter288
go build ./internal/format/                                              # rc=0
go vet ./internal/format/                                                # rc=0
gofmt -l internal/format                                                 # rc=0 AND empty
go test ./internal/format/ -run 'TestMeasurementIgnoresInheritedAttachments' -v 2>&1 | grep -c -- '--- PASS: TestMeasurementIgnoresInheritedAttachments'   # rc=0, prints 1
go test ./internal/format/ -run 'TestNestedInteriorCommentsDoNotAttach' -v 2>&1 | grep -c -- '--- PASS: TestNestedInteriorCommentsDoNotAttach'             # rc=0, prints 1
go test ./internal/format/                                               # rc=0
```
Snapshot to `.snap/M1/`.

---

## M2 — Measure-and-declare row (2), and pin `width.go:46`

**File:** new `internal/format/measurement_error_test.go`

**T2 `TestMeasurementErrorAlwaysAccompaniesRenderError`** — the P5/P6 sweep, made permanent. Walk `../../examples` and `../../std` for `*.ail`; skip files with parse errors. For each parse-clean file, for each `*ast.Func` whose `Body` is an `*ast.Block`, for each `*ast.Let` in `blk.Exprs` with `Body != nil`: inject `&ast.Error{Msg: "injected"}` into `Value`, then (restoring between) into `Body`. For widths `{120, 40, 20}`. Each injection: `p := &printer{w: newWriter("  "), maxWidth: w}; ferr := p.file(f)`.

Assertions:
1. **KILLER (mutant C):** for **every** injection, `p.measurementErr != nil`. (Measured 312/312.)
2. **REDUNDANCY FACT:** for every injection, `ferr != nil` — i.e. the live cell `measErr set && ferr == nil` stays **empty**. If this ever fails, row (2) is **live**, the declaration below is **refuted**, and the executor must stop and report.
3. **CONTROL (anti-vacuity):** `injections >= 100` (measured 312).
4. **CONTROL (no ambient noise):** on the **pristine** pass, `p.measurementErr == nil` and `ferr == nil` for every file (measured 1329/1329).

Restore `let.Value` / `let.Body` after each injection; the AST is shared across widths.

**Also in M2:** at `format.go:105` and `format.go:149`, add the **declaration** the mission row asks for: this propagation is **redundant on the measured domain** — `expr()`'s only error sources are AST-shape errors, and injection of `ast.Error` at 312 sites across the corpus put `measurementErr` and a real `p.file()` error together **every time**, live cell empty (`TestMeasurementErrorAlwaysAccompaniesRenderError`). State that it is **retained deliberately** as fail-closed defence (silent wrong output is the formatter's worst failure), that mutant **B** is therefore *expected to survive*, and that **M2's continuation layout is the change most likely to make it live** — at which point T2's assertion 2 flips it red. Add the same cross-reference at `interp.go:170` (P7: `:167-169` discards, only `:170-171` propagates).

**AC-M2:**
```
cd /Users/voightkampff/dev/sunholo-data/.wt-iter288
go build ./internal/format/                                              # rc=0
go vet ./internal/format/                                                # rc=0
gofmt -l internal/format                                                 # rc=0 AND empty
go test ./internal/format/ -run 'TestMeasurementErrorAlwaysAccompaniesRenderError' -v 2>&1 | grep -c -- '--- PASS: TestMeasurementErrorAlwaysAccompaniesRenderError'   # rc=0, prints 1
go test ./internal/format/                                               # rc=0
```
Snapshot to `.snap/M2/`.

---

## M3 — Mutation verification (the milestone that proves M1/M2 pinned anything)

For each mutant **A**, **B**, **C**: `cp` the target file to a backup, apply the mutation, **assert it landed** (sha256 before != after) and that the intended edit is what changed (count the old and new token occurrences: old `1→0`, new `0→1`; for B, `2→0` / `0→2`), `go build ./internal/format/` (**rc=0** — a mutant that does not compile measures nothing), run `go test ./internal/format/`, record **rc and the full set of `--- FAIL:` test names**, then restore by `cp` from the backup and **assert the sha256 matches the pre-mutation value**.

Expected, with predicted blast radius — **record what actually happens; do not assert a red set you have not run**:

| Mutant | Expectation | Predicted blast radius |
|---|---|---|
| **A** | **RED**, and `TestMeasurementIgnoresInheritedAttachments` **must** be among the failures | plausibly **single-test** — the corpus differential says nothing else observes it (P2/P3). If other tests also go red, that is new information: record it. |
| **B** | **GREEN — survives, as declared.** T2 exercises `p.file()` directly, so it is untouched. **If B is RED, the M2 redundancy declaration is REFUTED** — stop, restore, and report rather than editing the declaration to match. | n/a |
| **C** | **RED**, and `TestMeasurementErrorAlwaysAccompaniesRenderError` **must** be among the failures | predicted **broader than one test**: dropping the assignment also disables the `if p.measurementErr != nil { return false }` bail at `width.go:58`, so `exceedsWidth` starts comparing against a width of `0`. Other width/corpus tests may go red too. Record the full set. |

**AC-M3:** a recorded table of, for each of A/B/C: pre-sha, post-sha, occurrence counts before/after, build rc, test rc, full `--- FAIL:` list, restored-sha == pre-sha. Then the tree must be pristine:
```
cd /Users/voightkampff/dev/sunholo-data/.wt-iter288
shasum -a 256 internal/format/width.go internal/format/format.go internal/format/interp.go
git status --short          # only the NEW/edited files of M1+M2 — no stray .bak, no .orig
go test ./internal/format/  # rc=0
```
Snapshot to `.snap/M3/`.

---

## 4. Out of scope (explicitly)

- **No production test hook** for the corpus att-differential. It would need a new `inlineWidthProbe` in `inlineWidth`; T1 already kills mutant A without touching production control flow. The differential stays a planner/M2-time instrument, and the plan records its numbers instead.
- **No deletion** of the `measurementErr` propagation. It is redundant *today, on the measured domain*; it is fail-closed defence and M2's continuation layout is the change that plausibly makes it live.
- **No M2/M3 continuation-layout work.** This sprint only clears the two unpinned hunks ahead of it, and leaves the tripwires (T1b, T2 assertion 2) that will fire when it lands.
- `~116` corpus lines still exceed 120 runes; nothing wraps within a line yet (parent V9). Unchanged by this sprint.
