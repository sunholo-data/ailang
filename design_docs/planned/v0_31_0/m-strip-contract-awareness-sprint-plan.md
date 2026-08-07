# Sprint Plan — `m-strip-decl-aware` (declaration-aware source stripping in `internal/testing`)

**Sprint ID**: `M-STRIP-DECL-AWARE`
**Mission**: V1 outer loop, iteration 127
**Planner**: Opus (sprint-planner role)
**Planner-Lane**: opus
**Date**: 2026-08-01
**Base**: `/tmp/wt-iter127`, detached at `origin/dev` `858b067d4`
**GitHub issues**: `#548` (closes), AC9 blocker for `#535` M2 (unblocks — `#535` itself stays OPEN)
**Design doc**: **NONE — bug-fix lane** (judgement in §1)
**Risk**: low-medium
**Estimated**: ~380 LOC (impl ~150, tests ~230), ~1.5 executor-days

---

## 0. Verification posture

Everything in §2 and §3 was measured first-party in this worktree with
`/tmp/wt-iter127/bin/ailang` (built from `858b067d4`). Every claim carries its command.
Fixture texts in §6 were validated with `ailang check` before being written down.

**Sandbox caveat (executor is `codex:gpt-5.6-sol`)**: the sandbox cannot bind loopback
sockets. Any `bind: operation not permitted` from `go test ./cmd/ailang/...` or any test that
starts a listener is **UNINFORMATIVE UNDER SANDBOX** — neither a pass nor a failure. Record it
as such and move on; the controller re-runs those gates outside the sandbox. Do **not** attempt
to "fix" a bind denial.

**Commit posture**: codex cannot `git commit` in a linked worktree. Leave the tree dirty and
clean; the controller finalizes commits.

---

## 1. JUDGEMENT: no design doc, no quorum — bug-fix lane

**Recommendation: proceed straight to execution. No design doc, no quorum round.**

Reasons, in order of weight:

1. **The FORUM RULE (charter line 158) scopes the full pipeline to FEATURE-shaped work.**
   This ships no new syntax, no new CLI flag, no new user-visible semantics. It makes an
   existing internal transform stop corrupting the source it produces. The observable delta is
   "programs that already had a defined meaning now get it".

2. **Precedent is direct**: `m-nightly-run-validity-gate` / `#524` ran doc-less as a bug fix
   inside one file. This is the same shape — one function (`stripNonPureFunctions`,
   `internal/testing/executor.go:475`) with three call sites in the same file.

3. **The one policy-ish decision has an in-repo ratified precedent, so it is convergence, not
   invention.** Treating an explicit empty effect annotation (`! {}`) as pure is already done,
   verbatim, at two sites:
   - `cmd/ailang/verify.go:160-168` — *"functions with `! {}` (empty effects) are semantically
     pure but the parser only sets `IsPure` for the explicit `pure` keyword"*
   - `cmd/ailang/ai_check.go:231-232` — same fixup
   Adopting the same local predicate inside `internal/testing` aligns a third consumer with an
   existing convention. A quorum round would be spent ratifying a decision the codebase already
   made twice.

If the controller disagrees and wants the gate, the item parks cleanly to the 08-03 re-arm —
but it then takes the AC9 blocker with it, and `#535` M2 cannot start.

---

## 2. Premises REFUTED

The controller supplied 8 measured premises. Six hold. **Two are refuted**, one of them in a way
that materially widens the fix.

### R1 — REFUTED (material): the root cause is NOT contract-specific. It is *any* multi-line declaration.

The controller's premise 3, and `#548`'s own root-cause paragraph, both say the bug is that
contracts are left orphaned. That is one **instance**. The actual defect is that
`stripNonPureFunctions` deletes exactly **one line** — the line matching `export func <name>` —
for a declaration that may span many lines. Everything else in the declaration is orphaned:
contracts, the body, and the closing brace.

Measured — a non-pure function with **no contract at all**, multi-line body, beside a named test:

```
$ /tmp/wt-iter127/bin/ailang test /tmp/iter127v/c2_multiline.ail
  ✗ big works
      PAR_NO_PREFIX_PARSE at .../_namedtest_body_2080421920.ail:4:1: unexpected token in expression: }
rc=1
```

Same class for a genuinely effectful (`! {IO}`) multi-line function (`c5_effectful.ail`, same
`unexpected token: }`), which the `! {}`-purity remedy alone would **not** fix. And a third
variant the naive span fix would still miss:

```
$ /tmp/wt-iter127/bin/ailang test /tmp/iter127v/c7_annot.ail
      PAR_ATTR_REQUIRES_FUNC at _namedtest_body_1026338386.ail:4:3: annotations must be followed by a function declaration
      PAR_NO_PREFIX_PARSE at _namedtest_body_1026338386.ail:5:1: unexpected token in expression: }
```

Leading `@verify` / `@route` annotations sit **above** `FuncDecl.Span.Start` and are orphaned by
a span-based strip too. Consequence for the plan: the strip must delete
`min(annotation lines) .. Span.End.Line`, not "the declaration line plus its contracts".

Also, on the AST: contracts are **not** separate top-level forms. They are
`ast.FuncDecl.Properties []*Property` (`internal/ast/ast_decl.go:52`), attached by the parser
and merely *printed* on their own source lines. The controller's premise-6 wording ("reachable
from the AST") is right; premise-3's wording ("SEPARATE TOP-LEVEL FORMS") is wrong about the
AST, right about the text.

### R2 — REFUTED: `#548`'s "known-positive control" is weak; it passes for the wrong reason.

`#548`'s control is `test "trivial" { true }` — a body that **calls nothing**. It therefore
cannot detect that the function was deleted. With a control whose body actually calls the
function, the "no contract" case fails too, with a *different* error:

```
$ /tmp/wt-iter127/bin/ailang test /tmp/iter127v/c1_oneline.ail    # one-line body, no contract
      pipeline error: type error ... undefined variable: big at _namedtest_body_1446843359.ail:5:4
rc=1
```

So even when the strip produces *parseable* output, it removes the function the named test is
about. This is exactly the trap the controller's design question anticipated — and it is
already live today, not merely a hazard introduced by fixing the parse error. It is why M2
exists.

### Premises that HOLD (re-verified)

| # | Premise | Status |
|---|---|---|
| 1 | `#548` reproduces (`PAR_NO_PREFIX_PARSE` on `requires`/`ensures` in `_namedtest_body_*.ail`) | HOLDS — `named_test_contract.ail`, rc=1 |
| 2 | AC9 shape reproduces (`failed to extract function binding for big: … requires`) | HOLDS — `moduleless_contract.ail`, rc=1; module-ful twin rc=0 |
| 4 | `IsPure` set only by the `pure` keyword (`parser_func.go:22-24, 33`) | HOLDS |
| 5 | Adding `pure` makes the AC9 case pass | HOLDS — `ml_pure.ail` rc=0, `2 tests: 1 passed, 0 failed, 1 skipped` |
| 6 | `f.Properties` with `Kind`/`Pos` is reachable; brace-depth machinery reusable | HOLDS — but §3 uses `Span`, which is strictly better |
| 7 | Three call sites: `executor.go:197`, `:401`, `:682` | HOLDS |
| 8 | Module pipeline loads by filename; `src.Code` bypassed | HOLDS, and **sharper** — see §4 |

`#548`'s blast-radius claim also re-verified: 7 in-repo `.ail` files contain a named `test "`
block; the two live ones (`tests/record_update_regression_test.ail`,
`internal/pkg/testdata/multi_module/src/core_test.ail`) both pass at HEAD (rc=0). Latent trap,
zero live in-repo breakage. Confirmed.

---

## 3. The design decision (the question the controller said not to paper over)

**Chosen: (a′) — declaration-aware strip with a keep-set, plus a LOCAL effectively-pure
predicate. Option (b) (fix `IsPure` in the parser) is REJECTED on measured blast radius.**

### Why (b) is rejected

`ast.FuncDecl.IsPure` is read outside `internal/testing`:

```
$ grep -rn "IsPure" --include="*.go" internal/ cmd/ | grep -v _test.go
internal/format/decl.go:57          if d.IsPure {          <-- ailang fmt emits the `pure` keyword
internal/types/typechecker.go       if d.IsPure || isValue(d.Body) {
internal/ast/ast_decl.go            String()
internal/testing/executor.go:70,478
```

`internal/format/decl.go:57` is decisive. Flipping `IsPure` at the parser would make `ailang fmt`
**rewrite user source**, inserting a `pure` keyword the author never typed, on every
`! {}` function in the corpus. That is precisely the silent-rewrite class that cost this project
two weeks and −74% tokens on the fmt A/B. `internal/types/typechecker.go` reading it for the
value restriction is a second, independent generalization hazard.

The two existing consumers that *want* empty-effects-means-pure both solved it **locally**, at
the point of consumption, without touching the parser (`verify.go:160-168`,
`ai_check.go:231-232`). We do the same.

### Why (a) alone is insufficient — and why the split is M1/M2, not one change

- Strip the whole declaration and nothing else (M1 alone): `#548`'s named-test parse error is
  gone, and the effectful/annotated variants (R1) are fixed. But AC9's `big` is deleted, so
  `ExtractFunctionBinding` returns *"function 'big' not found"* — the trade the controller
  predicted. And `c1_oneline`'s `undefined variable: big` (R2) is untouched.
- Add the effectively-pure predicate (M2): `big` (`! {}`) is no longer classified non-pure, so
  it survives the strip with its contracts attached, and AC9 clears. Independently, the
  function *under test* is force-kept at the extraction call sites, so an effectful
  function-under-test does not vanish either.

Both halves are needed. M1 is nonetheless independently landable and independently valuable
(§5), because it fixes a class M2 does not touch: genuinely effectful (`! {IO}`) multi-line
declarations, which must stay stripped and today corrupt the file when they are.

### Deliberately OUT of scope (recorded, not silently dropped)

A named test that **calls** a genuinely effectful function still gets `undefined variable`,
because M2 keeps effectful functions only at the *extraction* call sites, not in the named-test
base source. That is a real defect with a verified repro, but it is neither `#548` nor AC9, and
keeping `! {IO}` functions in a `ModeEval` temp module is an effect-checking risk this sprint
should not absorb. **Deliverable: file it as a new issue during landing** (repro:
`c1_oneline.ail`-shape with `! {IO}` instead of `! {}`). Do not fix it here.

---

## 4. Conflict surface / blast radius — the three call sites

`stripNonPureFunctions` has three callers. They do **not** share a fate, because the pipeline
routes on `Filename`, not `Code` (`internal/pipeline/pipeline.go:163-169`: any non-empty,
non-`<repl>` `Filename` → `runModuleWithContext`, which loads the file from disk).

| Site | Function | Temp file written? | Is the strip LIVE? | Affected by this sprint |
|---|---|---|---|---|
| `executor.go:197` | `EvaluateNamedTestBodyExprs` | **always** (`_namedtest_body_*.ail`) | **LIVE in all cases** | **YES** — `#548` lives here |
| `executor.go:401` | `ExtractFunctionBinding` | only when `sourceFile.Module == nil` | **LIVE only module-less**; dead when a `module` header is present | **YES** — AC9 lives here |
| `executor.go:682` | `ExtractPureClusterForFunction` | never (`Filename: e.modulePath`) | **DEAD in all cases** | M3 only |

**Sharpening of premise 8 — it is a latent trap, and it also mis-reports telemetry.**
In the module path, `src.Code` is referenced exactly once, and never lexed:

```
$ grep -n "src.Code" internal/pipeline/pipeline_module.go
38:  attribute.Int("file.size_bytes", len(src.Code)),
```

(`src.Code` *is* lexed in `pipeline_single.go:88`, but that path is unreachable with a non-empty
`Filename`.) So at site 682 the entire `stripNonPureFunctions` call is a no-op whose only
observable effect is that the `file.size_bytes` trace attribute reports the **stripped** length
instead of the file length. Verdict: **latent trap, not benign dead code** — a maintainer will
reasonably assume the strip applies there. M3 disposes of it explicitly.

**Also note premise 8's masking claim is site-specific**: it masks at site 401 only. It does
*not* mask at site 197 — a module header does not save the named-test path:

```
$ /tmp/wt-iter127/bin/ailang test /tmp/iter127v/c4_mod_multiline.ail   # HAS module header
      PAR_NO_PREFIX_PARSE at .../_namedtest_body_2122008266.ail:4:1: unexpected token in expression: }
rc=1
```

**File-size conflict**: `internal/testing/executor.go` is **739 lines** against the 800-line CI
cap (`make/code-health.mk:122`) — 61 lines of headroom. All new code goes in a **new file**
`internal/testing/source_strip.go`, and `stripNonPureFunctions` **moves** there (dropping
`executor.go` to ~670). `runner.go` (670) is untouched.

---

## 5. Milestones

### M1 — Declaration-aware strip (mechanical; kills the corruption class)

**Independently landable. Independently valuable.** After M1 alone, effectful multi-line
declarations beside a named test stop producing parse errors in an invisible temp file.

**Files**
- NEW `internal/testing/source_strip.go` (~120 lines): `stripNonPureFunctions` moves here
  verbatim, then is rewritten.
- `internal/testing/executor.go`: delete the moved function (739 → ~670 lines).
- NEW `internal/testing/testdata/strip/*.ail`: fixtures from §6.
- NEW `internal/testing/source_strip_test.go` (~150 lines).

**Change**
1. For each `*ast.FuncDecl` selected for removal, delete the line range
   `startLine .. f.Span.End.Line`, where
   `startLine = min(f.Span.Start.Line, min over f.Annotations[i].Pos.Line)`.
   `f.Span` is populated at `internal/parser/parser_func.go:229` (block form) and `:323`
   (extern); `End` is the position of the closing `}` of the body.
2. **Defensive fallback, mandatory**: if `f.Span.End.Line == 0` or `< f.Span.Start.Line`
   (hand-built ASTs in unit tests; extern decls), fall back to deleting the single line
   `f.Pos.Line`. This must be its own unit test, not an untested branch.
3. Keep the existing `*ast.TestDecl` / `*ast.PropertyDecl` brace-depth `skipRanges` machinery
   unchanged — it already works and is orthogonal.
4. Delete the `containsPattern(line, "export func "+name)` substring matching entirely. It is
   the source of the bug and is also a false-positive hazard (`func fooBar` contains `func foo`).

**Acceptance criteria (runnable now, with the current flag set)**

All from repo root in `/tmp/wt-iter127`, after `go build -o bin/ailang ./cmd/ailang`.

| AC | Command | Expected |
|---|---|---|
| **AC1.1** | `./bin/ailang test internal/testing/testdata/strip/named_test_multiline.ail; echo rc=$?` | `rc=0`; output contains `1 passed` and `0 failed`; output does **not** contain `PAR_NO_PREFIX_PARSE`. *Pre-fix: rc=1, `PAR_NO_PREFIX_PARSE … unexpected token in expression: }`* |
| **AC1.2** | `./bin/ailang test internal/testing/testdata/strip/named_test_effectful.ail; echo rc=$?` | `rc=0`; `1 passed`, `0 failed`; no `PAR_NO_PREFIX_PARSE`. Proves a genuinely `! {IO}` declaration is removed *whole*. *Pre-fix: rc=1, `unexpected token in expression: }`* |
| **AC1.3** | `./bin/ailang test internal/testing/testdata/strip/named_test_annotated.ail; echo rc=$?` | `rc=0`; `1 passed`, `0 failed`; no `PAR_ATTR_REQUIRES_FUNC`. Proves leading annotations are removed with their declaration. *Pre-fix: rc=1, `PAR_ATTR_REQUIRES_FUNC … annotations must be followed by a function declaration`* |
| **AC1.4** | `./bin/ailang test internal/testing/testdata/strip/named_test_contract.ail; echo rc=$?` | `rc=0`; `0 failed`; no `PAR_NO_PREFIX_PARSE`. This is `#548` verbatim. *Pre-fix: rc=1, `3 tests: 1 passed, 1 failed, 1 skipped`* |
| **AC1.5** | `go test ./internal/testing/... 2>&1 \| tail -3` | `ok  github.com/sunholo-data/ailang/internal/testing`. Baseline at `858b067d4` is `ok` (measured) — no pre-existing red to explain away |
| **AC1.6** | `make check-file-sizes` | exits 0; `internal/testing/executor.go` reported under 800 |
| **AC1.7** | `./bin/ailang test tests/record_update_regression_test.ail && ./bin/ailang test internal/pkg/testdata/multi_module/src/core_test.ail` | both rc=0 — the two live in-repo named-test files, which pass today. Non-regression control |

> **AC1.1–AC1.4 do not pin skip counts or case counts.** Property case counts are wall-clock
> seeded (that is `#535`, still open); an AC that pinned them would be flaky by construction.
> `0 failed` + absence of the parse-error string is the crisp, stable assertion.

**Non-vacuity requirement (HARD)**: `source_strip_test.go` asserts absence of
`PAR_NO_PREFIX_PARSE` in several places. Every such negative assertion must sit in a table that
**also** contains `malformed_control.ail` (§6), asserting the string **is** present. If the
detector ever stops seeing parse errors, that row goes red and the whole test fails loudly.
A negative assertion without its positive control in the same test function is a review reject.

**Est**: ~120 impl + ~150 test = 270 LOC · ~1 day

---

### M2 — Strip policy: effectively-pure + keep-set (unblocks AC9, closes `#548` end-to-end)

**Depends on M1.**

**Files**
- `internal/testing/source_strip.go` (+~50 lines)
- `internal/testing/executor.go` (call sites 401 and 682: pass the function under test as the
  keep-set; ~6 lines)
- NEW `internal/testing/testdata/strip/moduleless_contract.ail`,
  `moduleless_contract_control.ail`
- `internal/testing/source_strip_test.go` (+~80 lines)

**Change**
1. Add, in `source_strip.go`, mirroring `cmd/ailang/verify.go:160-168` (cite it in the comment):
   ```go
   // isEffectivelyPure mirrors the fixup at cmd/ailang/verify.go:160-168 and
   // cmd/ailang/ai_check.go:231-232: a function with an EXPLICIT empty effect
   // annotation (`! {}`) is semantically pure, but the parser sets IsPure only
   // for the `pure` keyword (internal/parser/parser_func.go:22-24).
   // Deliberately LOCAL: ast.FuncDecl.IsPure is read by internal/format/decl.go:57,
   // where flipping it would make `ailang fmt` insert a `pure` keyword the author
   // never wrote.
   func isEffectivelyPure(f *ast.FuncDecl) bool {
       return f.IsPure || (f.Effects != nil && len(f.Effects) == 0)
   }
   ```
   Note the `f.Effects != nil` guard is load-bearing: a function with **no** effect annotation
   at all (`Effects == nil`) is *not* claimed pure, matching both precedent sites exactly.
2. Add a `keep map[string]bool` parameter (or variadic `keepFuncs ...string`) to the strip. A
   function whose name is in the keep set is never removed, regardless of purity.
3. `ExtractFunctionBinding` (`:401`) and `ExtractPureClusterForFunction` (`:682`) pass
   `functionName` as the keep-set. Site `:197` (named test) passes an empty keep-set — see the
   out-of-scope note in §3.

**Acceptance criteria**

| AC | Command | Expected |
|---|---|---|
| **AC2.1 (the AC9 blocker)** | `./bin/ailang test internal/testing/testdata/strip/moduleless_contract.ail; echo rc=$?` | `rc=0`; `0 failed`; output contains neither `failed to extract function binding` nor `PAR_NO_PREFIX_PARSE`; contains `big_property_2` with a `✓`. *Pre-fix: rc=1, `failed to extract function binding for big: … PAR_NO_PREFIX_PARSE … 3:1: requires`* |
| **AC2.2 (control)** | `./bin/ailang test internal/testing/testdata/strip/moduleless_contract_control.ail; echo rc=$?` | `rc=0`, `0 failed`. The module-ful twin of AC2.1. Passes **today** (measured: `2 tests: 1 passed, 0 failed, 1 skipped`) and must still pass. If AC2.1 and AC2.2 ever disagree, the module-less path has diverged again |
| **AC2.3** | `./bin/ailang test internal/testing/testdata/strip/named_test_contract.ail; echo rc=$?` | `rc=0`, `0 failed` — `#548` closed with the function actually present, not merely parseable |
| **AC2.4** | `go test -run 'TestStrip' ./internal/testing/... -v 2>&1 \| grep -E '^(=== RUN\|--- (PASS\|FAIL))'` | every `TestStrip*` shows `--- PASS`; **at least one `--- PASS` line must appear** (a zero-test run is a FAIL, not a pass) |
| **AC2.5** | `go test ./... 2>&1 \| grep -v '^ok\|no test files' \| head -20` | empty, **or** only `bind: operation not permitted` lines — which are **UNINFORMATIVE UNDER SANDBOX**, to be recorded verbatim and re-run by the controller outside |
| **AC2.6** | `./bin/ailang test examples/runnable/contracts/basic.ail; echo rc=$?` | rc unchanged from the pre-M1 baseline the executor records in step 1 of the sprint. Regression guard on the real contract corpus |

**Non-vacuity**: the AC9 unit test must assert `binding != nil` **and** that the returned binding
is named `big` — not merely that `ExtractFunctionBinding` returned no error. Pair it with a
negative control asserting `ExtractFunctionBinding("nonexistent", …)` still returns
`function 'nonexistent' not found`, so a keep-set that accidentally keeps everything fails loudly.

**Est**: ~50 impl + ~80 test = 130 LOC · ~0.5 day

---

### M3 — Dispose of the dead strip at call site 682 (small, honest)

**Depends on M2.** Do not skip: an untouched no-op that *looks* live is how the next reader
reintroduces this bug.

**Change**: in `ExtractPureClusterForFunction` (`executor.go:682`), the pipeline is invoked with
`Filename: e.modulePath`, so `strippedSource` is never lexed (§4). Choose ONE and state which in
the commit message:
- **(i) preferred** — stop computing it; pass `Code: string(sourceCode)` so the `file.size_bytes`
  telemetry attribute is truthful, and add a comment recording that the module pipeline loads by
  filename (`internal/pipeline/pipeline.go:163-169`) so a strip here would require a temp file.
- (ii) — write a temp file as `ExtractFunctionBinding` does, making the strip actually live.
  **Rejected as scope**: it changes cluster-extraction behaviour for every contract file in the
  corpus and is not needed by `#548` or AC9. Only take (ii) if (i) turns out to break a test.

**AC3.1**: `go test ./internal/testing/... ./internal/pipeline/...` → `ok` for both.
**AC3.2**: `./bin/ailang test examples/runnable/contracts/cross_function.ail; echo rc=$?` →
rc unchanged from the pre-M1 baseline (this file exercises the cluster path).

**Est**: ~15 impl + ~0 test (covered by AC3.1/AC3.2) · ~0.25 day

---

## 6. Fixtures (verified to parse before being written down)

Create under `internal/testing/testdata/strip/`. Each was written to disk and run through
`ailang check` and `ailang test` in `/tmp/iter127fx` at `858b067d4`; the "pre-fix" column is
measured, not predicted.

| File | Purpose | `ailang check` today | `ailang test` today |
|---|---|---|---|
| `named_test_contract.ail` | `#548` verbatim | OK | rc=1, `PAR_NO_PREFIX_PARSE … requires` |
| `named_test_multiline.ail` | R1: multi-line, **no** contract | OK | rc=1, `unexpected token: }` |
| `named_test_effectful.ail` | R1: genuinely `! {IO}` | OK | rc=1, `unexpected token: }` |
| `named_test_annotated.ail` | R1: leading `@verify` | OK | rc=1, `PAR_ATTR_REQUIRES_FUNC` |
| `moduleless_contract.ail` | AC9 shape | `MOD014 no 'module' declaration` — **expected**, the fixture is module-less by design | rc=1, `failed to extract function binding for big` |
| `moduleless_contract_control.ail` | module-ful twin, known-positive | OK | **rc=0**, `2 tests: 1 passed, 0 failed, 1 skipped` |
| `malformed_control.ail` | **instrument control** — genuinely broken source that must keep producing `PAR_NO_PREFIX_PARSE` | FAIL (by design) | rc=1 (by design) |

```ailang
-- named_test_contract.ail
module named_test_contract

test "trivial" { true }

export func guarded(x: int) -> bool ! {}
requires { x > 0 }
ensures { result == true } {
  x > 0
}
```

```ailang
-- named_test_multiline.ail
module named_test_multiline

export func big(x: int) -> bool ! {} {
  x > 10
}

test "big works" {
  big(20) == true
}
```

```ailang
-- named_test_effectful.ail
module named_test_effectful

import std/io (println)

export func shout(s: string) -> () ! {IO} {
  println(s)
}

export pure func big(x: int) -> bool {
  x > 10
}

test "big works" {
  big(20) == true
}
```

```ailang
-- named_test_annotated.ail
module named_test_annotated

import std/io (println)

@verify(depth: 3)
export func shout(s: string) -> () ! {IO} {
  println(s)
}

export pure func big(x: int) -> bool { x > 10 }

test "big works" {
  big(20) == true
}
```

```ailang
-- moduleless_contract.ail   (NO module header — that is the point)
export func big(x: int) -> bool ! {}
requires { x >= 0 }
ensures { result == (x > 10) } {
  x > 10
}
```

```ailang
-- moduleless_contract_control.ail
module moduleless_contract_control

export func big(x: int) -> bool ! {}
requires { x >= 0 }
ensures { result == (x > 10) } {
  x > 10
}
```

`malformed_control.ail` is authored by the executor; the only requirement is that
`ailang test` on it emits `PAR_NO_PREFIX_PARSE`, verified before it is used as a control.

---

## 7. Execution order

1. **Baseline capture (before any edit)** — run and save to the sprint notes:
   `./bin/ailang test examples/runnable/contracts/basic.ail`,
   `…/cross_function.ail`, `…/invoice.ail`, `…/finance.ail` (rc + summary line each), and
   `go test ./internal/testing/... > /tmp/base_testing.txt`. AC2.6 and AC3.2 compare to this.
   Without it, "unchanged" is unfalsifiable.
2. M1: move + rewrite the strip; fixtures; tests; AC1.1–AC1.7.
3. M2: purity predicate + keep-set; AC2.1–AC2.6.
4. M3: dead-code disposition; AC3.1–AC3.2.
5. `make fmt && make lint && make test` — record any `bind:` denial as UNINFORMATIVE.
6. Leave the tree dirty; report to the controller.

---

## 8. Rollback

Fully self-contained and cheap:

- **M3 alone**: revert the ~15-line diff in `ExtractPureClusterForFunction`. Zero behavioural
  coupling to M1/M2.
- **M2 alone**: delete `isEffectivelyPure` and drop the keep-set argument (revert to
  `!f.IsPure`). M1's span-aware strip is untouched and still correct — the tree lands in the
  state described in §3 ("M1 alone"), i.e. `#548`'s parse error fixed, AC9 still blocked. This is
  a *valid shippable state*, so M2 can be reverted under time pressure without a re-plan.
- **M1+M2+M3**: `git revert` the sprint commit(s). `stripNonPureFunctions` returns to
  `executor.go`, which returns to 739 lines — still under the cap. No schema, no data, no
  on-disk format, no CLI surface is touched, so there is nothing to migrate back.
- **Blast radius if it goes wrong in the field**: `stripNonPureFunctions` is reachable only from
  `ailang test`. `make verify-examples` runs `ailang run`, never `ailang test`
  (`scripts/verify_examples.go:108-115`, verified in iteration 126), so a regression here cannot
  redden the examples gate — it would surface in `go test ./internal/testing/...` and in
  `ailang test` on the contracts corpus, both of which are ACs above.

---

## 9. What this sprint explicitly does NOT do

- **ZERO seed work.** No `--seed` flag, no change to `newRNG`'s wall-clock branch, no
  `internal/testing/config.go`, no `TestConfig.WorkspaceRoot`. `#535` stays **OPEN**. M2/M3 of
  `m-property-seed-determinism` remain date-gated to the 2026-08-03 07:00 re-arm.
- No change to `internal/parser`, `internal/format`, or `ast.FuncDecl.IsPure` (§3).
- No fix for "named test calls an effectful function" (§3, out-of-scope) — **file an issue**.
- No change to how parse errors in generated temp modules are *attributed* to the originating
  source file. `#548`'s secondary suggestion ("a user cannot act on a path that no longer
  exists") is a genuine diagnostics defect but is a separate, larger change to pipeline error
  reporting. **Record it in the landing comment as remaining open on `#548`'s thread**, or
  `#548` gets closed with half its ask unaddressed.

---

## 10. Effect on the AC9 blocker

After AC2.1 passes, `m-property-seed-determinism`'s §5.3 blocker is discharged in its **first**
form — "produce a module-less contract fixture that survives `ExtractFunctionBinding`'s temp-file
round-trip" — and `internal/testing/testdata/strip/moduleless_contract.ail` is that fixture. The
re-arm session then does **not** need to restate AC9 over module-bearing files under two absolute
roots (the weaker fallback), so AC9 keeps testing exactly what it was written to test: that the
absolute checkout directory is absent from the hash identity.

**Landing must update** `design_docs/planned/v0_31_0/m-property-seed-determinism-sprint-plan.md`
§0.3 and §5.3 to record the discharge and name the fixture. If that edit is skipped, the 08-03
session re-derives a blocker that no longer exists.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_31_0/m-strip-contract-awareness-sprint-plan.md`
