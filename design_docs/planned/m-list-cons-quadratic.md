# M-LIST-CONS-QUADRATIC: Amortized O(1) List Prepend

**Status**: PARKED — `needs-human-review` (quorum BLOCKED after the one revision pass; see [Quorum verification log](#quorum-verification-log-mission-iteration-228))
**Target**: v0.34.0 (next feature release)
**Priority**: P0 — blocks a downstream consumer (email-parse M9, [#676](https://github.com/sunholo-data/ailang/issues/676))
**Estimated**: 4–5 days (Sprint 1 of 2; Sprint 2 is a separate follow-up doc, see [Sprint Split](#sprint-split--ordering-f1-vs-f5))
**Dependencies**: None
**Planner-Lane**: opus-required (runtime core value representation)

All facts in this doc were re-derived first-party at `origin/dev` = `88631976e` in this session
(binary: `go build -o /tmp/ailang-iter228 ./cmd/ailang` from that tree). The triage comment on
#676 numbered these findings F1–F8; every F-row this doc relies on was re-run, none inherited.
See the [Verification Log](#verification-log).

**Revision (quorum round 2).** Both external reviewers rejected round 1, and both objections were
substantive; this revision changes the design, it does not re-argue it.
(1) *gpt5-6-sol*: Option B's safety rested on V5/V6, greps that cannot see an aliased write
(`e := lv.Elements; e[i] = v`). V5/V6 are **retired** (rows kept for audit, no claim cites them);
a widened instrument (V32/V33) shows zero aliased writes exist at HEAD — but that is a fact about
today, not an invariant, so safety is now **mechanically enforced**: a CI-gated `listmut`
analyzer with seeded-violation tests, plus copy-on-escape at the three raw-slice boundaries. See
[Immutability enforcement](#immutability-enforcement-mechanical).
(2) *gemini-3-1-pro*: as written, `concat_List`'s copy path returned nil-arena lists, so the
`[x] ++ rest` recursion in `std/list` could never bootstrap the fast path and stayed O(n²). The
design now **seeds**: every list returned by `::` or `++` is arena-backed (slow paths allocate
slack, geometric growth), which makes the chain bootstrap by construction. See
[Arena bootstrap across the `++` recursion](#arena-bootstrap-across-the--recursion). The
front-slack asymmetry the round-1 doc glossed is now stated explicitly: front slack accelerates
right-recursion (`[x] ++ rest`); it does nothing for left-append (`concat(big, [x])`), which is
why `reverse` is fixed by delegation instead (V35 enumerates both classes).

## Routing Lane (per design_docs/PROGRAM.md §4)

**Lane: AILANG fix.** Argued against the program's default bias ("if it can be an extension, it
is an extension"):

- The frozen core in PROGRAM.md is the **motoko agent-loop core**, not the AILANG runtime.
  PROGRAM.md §1 gives AILANG its own lane with the opposite trajectory: "Gaps (errors, builtins,
  dialect traps) get fixed as the loop finds them. Fix-rate → declining toward 0." §4 routes
  "bad/unfixable error, missing builtin, type-system gap" to **AILANG fix: design doc in
  `ailang/design_docs/` + regression test" — this doc.
- **The extension test fails twice.** (1) No motoko extension — prompt shaping, tool shaping,
  retrieval — can change the asymptotic complexity of `::` inside the interpreter or lift the
  evaluator's recursion cap. (2) The reporter (#676, email-parse) is not behind motoko at all;
  dialect coaching would never reach them. Coaching "never build lists by prepending" also fights
  the canonical functional idiom the language is designed around (Axiom A7, machines-first), and
  `std/list` itself is written in the quadratic shape (V17) — the defect is in the substrate, not
  in any agent's strategy.
- It is **not** the core-floor lane: that lane is for the motoko core's crash/overflow floor and
  triggers a re-freeze; neither applies here. Within the AILANG repo the change does touch the
  runtime's central value type, so it carries core-change discipline *internally*: the mandatory
  [Conflict Surface](#conflict-surface), a frozen representation invariant, and a regression
  surface pinned to named fixtures.

## Problem Statement

Building a list by prepending — the canonical functional accumulator idiom — costs **Θ(n²) time
and allocation** in the tree-walking evaluator, because `eval.ListValue` is a flat Go slice
(`internal/eval/value.go:84-86`) and `::` copies the entire tail on every call
(`internal/builtins/list.go:98-103`, V1/V2). A second, independent defect blocks the same idiom:
the tree-walking evaluator has **no tail-call elimination** (V3), so accumulator recursion is
capped at 10,000 frames by default — the repro at n=12,800 *fails* with `RT_REC_003`, it does not
merely get slow (V10). The RT_REC_003 message even advises "enable tail recursion", an option
that does not exist (V11).

**Current State (all first-party measurements at `88631976e`, V12/V13/V21):**

Repro (verbatim from #676; type-checks clean at HEAD, V9):

```ailang
module repro
import std/io   (println)
import std/list (length as llen)
export func main() -> () ! {IO} = println(show(llen(gen(6400, []))))
pure func gen(n: int, acc: [string]) -> [string] =
  if n <= 0 then acc else gen(n - 1, "constant" :: acc)
```

| Measurement | Value at base |
|---|---|
| Peak RSS, hello-world baseline | 46.4 MB |
| Peak RSS, repro n=1,600 / 3,200 / 6,400 | 76.0 / 152.4 / 428.7 MB |
| Peak RSS, repro n=12,800 (`--max-recursion-depth 200000`) | **1,467.9 MB** |
| Repro n=12,800 at default flags | **fails**: `RT_REC_003: max recursion depth 10000 exceeded` |
| Heap attribution at n=12,800 (pprof `alloc_space`) | **94.97% (1289.63 MB) flat in `builtins.listConsImpl`** |
| Predicted from representation: n(n+1)/2 × 16 B | 1,250.1 MB — 3.2% from observed: a derivation, not a correlation |
| `std/list.reverse` of a 2,000-element list | **10.2 s wall, 172.7 MB RSS** (V21) |

Wall-clock of the repro itself is NOT the harm at realistic n (0.05 s at 1,600 → 0.41 s at
12,800); the harm is allocation volume / peak RSS. The #676 reporter OOMed a real pipeline at
10,570 MB and pushed their machine into swap twice.

**The defect is systemic, not a one-off** (audit per CLAUDE.md principle 3):

- User-written cons recursion — any list built by prepending is quadratic. Recursion vs `map` is
  irrelevant; `map` is only fast because it delegates to an iterative Go builtin (V8).
- **`std/list` itself is written in the quadratic shape** (V16/V17): it contains zero `::`, but
  `reverse`, `concat`, `zip`, `take`, `mapE`, `filterE`, `sortBy`'s merge all build via
  `[x] ++ rest` — and `++` copies both operands (V18), the same O(tail)-per-step pattern.
- `reverse` is the worst case: `concat(reverse(rest), [x])` — measured 10.2 s for n=2,000 —
  while an iterative `_list_reverse` builtin **already exists** at `internal/builtins/list.go:578`
  and `std/list.ail` simply doesn't use it (V19/V20).
- The bytecode VM's `OpCons` is independently O(n) (`internal/vm/vm.go:436-438`, V22), but
  `ailang run` does not use the VM; it is a documented non-goal here.

**Impact:** any AILANG program that accumulates a list of more than a few thousand elements —
exactly the "replace the Python loader" workloads downstream consumers are trying to move to
AILANG (#676 M9). Eval-harness models writing idiomatic functional code hit the same wall.

## Verification Log

Provenance: **[R]** = re-derived/measured first-party this session at `88631976e`;
**[A✓]** = surfaced by the read-only inventory subagent, then spot-verified first-party by
reading the cited lines; **[A]** = subagent inventory, used for breadth only (no load-bearing
design decision rests on an [A] row).

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 [R] | `ListValue` is a flat slice | Read `internal/eval/value.go:84-86` | `type ListValue struct { Elements []Value }` |
| V2 [R] | `::` copies the whole tail | Read `internal/builtins/list.go:98-103` | `make([]eval.Value, 0, 1+len(tail.Elements))` + `append(result, tail.Elements...)` |
| V3 [R] | No TCO in tree-walking evaluator (negative + same-scope controls) | `test -d internal/eval` → YES; `grep -ril 'tail.\?call' internal/eval/` → empty (rc=1); control `grep -rl 'recursionDepth' internal/eval/` | control hits 2 files (`eval_evaluator.go`, `eval_operations.go`); tail-call grep 0 |
| V4 [R] | TCO machinery exists only in the VM/bytecode | `grep -ril 'tail.\?call' internal/vm/ internal/bytecode/ \| wc -l` | 11 files |
| V5 [R] — **RETIRED** | ~~No in-place writes to `.Elements[i]` anywhere~~ — the instrument only matches the literal pattern `X.Elements[i] =`; it **cannot see an aliased write** (`e := lv.Elements; e[i] = v`), as reviewer gpt5-6-sol correctly objected. Superseded by V32/V33. Row retained for audit trail only; no design claim may cite it. | `grep -rnE '\.Elements\[[^]]*\] *=[^=]' …` | (0 writes / 44 reads — true but insufficient) |
| V6 [R] — **RETIRED** | ~~Only `append(X.Elements, …)` sites are AST nodes~~ — same blind spot as V5 for aliased appends. Superseded by V32 + the capacity clip. Audit trail only. | `grep -rn 'append([A-Za-z_.]*\.Elements' …` | (2 AST-literal hits — true but insufficient) |
| V7 [R] | Blast radius of a representation change | `grep -rn 'ListValue{' … \| wc -l` etc., scope `internal/ cmd/ --include='*.go'` | 386 `ListValue{` constructions; 902 `.Elements` references; 750 `ListValue` total |
| V8 [R] | `map` is fast because it is an iterative Go builtin | `sed -n '56p' std/list.ail`; `grep -n listMapImpl internal/builtins/list_iterative.go` | `= _list_map(f, xs)`; impl at `list_iterative.go:68` |
| V9 [R] | Repro type-checks at HEAD | `/tmp/ailang-iter228 check /tmp/repro228/repro.ail` | `✓ No errors found!` |
| V10 [R] | n=12,800 fails at default flags | `/tmp/ailang-iter228 run --caps IO repro12800.ail` | `Error: execution failed: RT_REC_003: max recursion depth 10000 exceeded…` |
| V11 [R] | The advertised "enable tail recursion" option does not exist (negative + control) | `grep -rn 'enable tail recursion' internal/ cmd/ --include='*.go'`; `run --help \| grep -iE 'recursion\|depth\|tail'` | Only the message site `internal/eval/eval_operations.go:58`; help shows only `-max-recursion-depth` (control: the help grep does see that flag) |
| V12 [R] | Peak RSS scaling | `/usr/bin/time -l` on hello + n∈{1600,3200,6400,12800} | 46,432,256 / 75,988,992 / 152,354,816 / 428,654,592 / 1,467,875,328 B |
| V13 [R] | 95% of allocation is `listConsImpl` | `go tool pprof -top -sample_index=alloc_space -nodecount=8 /tmp/ailang-iter228 /tmp/repro228/mem228.prof` | `1289.63MB 94.97% … builtins.listConsImpl` of 1357.93MB total |
| V14 | Trace recorder ruled out as cause | controller-run A/B (`AILANG_TRACE=off` 1468.1 vs default 1467.6 MB); not re-run — V13 independently establishes cause | hypothesis dead |
| V15 [R] | Pattern-match tail `[x, ...rest]` is an O(1) subslice alias, not a copy | Read `internal/eval/eval_patterns.go:254-255` | `tailElements := listVal.Elements[len(p.Elements):]` then `&ListValue{Elements: tailElements}` |
| V16 [R] | `std/list.ail` contains zero `::` (negative + same-command control) | `grep -c '::' std/*.ail \| grep -v ':0$'` | hits only `embedding.ail:2`, `smoke.ail:2`, `string.ail:1` — the instrument sees positives in sibling files; `list.ail` absent |
| V17 [R] | `std/list` builds via `[x] ++ rest` / `concat(…, [x])` | `sed -n '29,55p;62,72p;99,121p;202,215p' std/list.ail` | `reverse` = `concat(reverse(rest), [x])`; `concat`/`zip`/`take`/`flatMap` = `[x] ++ recursive-call` |
| V18 [R] | `++` copies both operands | Read `listConcatImpl`, `internal/builtins/list.go:163-183` | `make(…, len(l1)+len(l2))` + two `append(…, ….Elements...)` |
| V19 [R] | Iterative `_list_reverse` builtin exists | Read `internal/builtins/list.go:578-589` | single `make([]Value, n)` + reverse-index loop |
| V20 [R] | `std/list.reverse` does not use it | V17 excerpt, `std/list.ail:29-35` | recursive `concat(reverse(rest), [x])` |
| V21 [R] | `reverse` base cost, n=2,000 | `/usr/bin/time -l` + `time` on revtest.ail (type-checked first: `✓`) | prints `2000`; **10.2 s real, 172,687,360 B peak RSS** |
| V22 [A✓] | VM `OpCons` is independently O(n) | Read `internal/vm/vm.go:429-440` | `make([]bytecode.Value, 0, len(old)+1)` + `append(elems, old...)` |
| V23 [A✓] | VM has its own list repr; eval↔VM bridge deep-copies both directions | Read `cmd/ailang/bytecode_bridge.go:132-142,191-200` | per-element copy loops both ways |
| V24 [A✓] | Fork shares module-level environments read-only across goroutines | Read `internal/eval/eval_evaluator.go:166-170`, `internal/eval/env.go:6-14` | Fork comment: "shares read-only state"; env comment: "module-level environments are shared (read) across goroutines" over `sync.RWMutex` |
| V25 [A✓] | `SafeAsList` leaks the raw `Elements` slice to callers | Read `internal/builtins/safe_cast.go:94-99` | `return lv.Elements, nil` (V34: zero non-test callers at HEAD; still hardened, see enforcement) |
| V26 [A✓] | Telemetry reads list **length only**, never elements | Read `internal/telemetry/effect_spans.go:74-77` | `attribute.Int("process.arg_count", len(listVal.Elements))` |
| V27 [R] | Base test status of the two touched packages | `go test ./internal/builtins/ ./internal/eval/` | `ok` / `ok` (5.8 s / 0.5 s) |
| V28 [R] | `docs/LIMITATIONS.md` has no depth-cap or quadratic-list entry (negative + same-file control) | `grep -in 'recursion\|cons\b\|quadratic' docs/LIMITATIONS.md` | 1 hit: line 21, the Y-combinator row (control: the grep sees that row; no cap/cons entry exists) |
| V29 [R] | Codegen/emit-go never touches `eval.ListValue` (negative + control) | `grep -rln "internal/eval" internal/gen/` → empty (rc=1); control `ls internal/gen/` | control lists `block/ emitgo/ golang/ …` — directory exists, grep sees it |
| V30 [R] | No test pins the RT_REC_003 hint text | `grep -rn 'RT_REC_003' --include='*_test.go' internal/ cmd/`; control: non-test grep | test asserts substring `"RT_REC_003"` only (`internal/eval/recursion_test.go:302`); non-test control hits `eval_operations.go` |
| V31 [R] | Cited regression fixtures exist and use `::` / list patterns | `grep -ln '::' examples/*.ail`; `ls examples/ \| grep -i pattern` | `first_non_repeat.ail`, `inline_tests_recursive.ail`, `inline_tests_best_practices.ail`, `record_cons_pattern.ail`, `record_list_extraction.ail`; `pattern_matching_adt.ail` |
| V32 [R] | Widened aliasing instrument (replaces V5/V6): bindings of a `.Elements` slice to a local, non-test Go, plus a control proving the write-matcher fires | `grep -rnE '(:=\|=) *[A-Za-z_][A-Za-z0-9_.]*\.Elements$' internal/ cmd/ --include='*.go' \| grep -v _test.go`; control: bare slice index-writes `grep -rnE '^[^=/]*\[[^]]+\] *= [^=]' internal/ cmd/ --include='*.go' \| grep -v _test.go \| wc -l` | 2 bindings: `internal/types/typechecker_patterns.go:272` (`tupleTy.Elements` — a `TupleType`, not `eval.ListValue`; out of scope) and `internal/testing/shrink.go:199` (the only `eval.ListValue` binding in the repo); control: 2,423 bare index-writes seen by the matcher |
| V33 [R] | The one `eval.ListValue` binding never writes through the alias | Read `internal/testing/shrink.go:193-247` | every write goes to a fresh `newElems := make(…)` + `copy(newElems, elems)` (lines 238-240) or a fresh `removed := make(…)`; other uses of `elems` are reads and read-only subslices ⇒ **zero aliased writes at HEAD** — an observation about today, not an invariant (hence the mechanical guard) |
| V34 [R] | Complete enumeration of raw-`Elements` escape sites + caller audit | `grep -rn '\.Elements, nil' internal/ cmd/ --include='*.go' \| grep -v _test.go`; `grep -rn 'SafeAsList' internal/ cmd/ --include='*.go'`; Read `internal/effects/testctx/mock_context.go:353-355` | 3 sites: `builtins/safe_cast.go:97` (8 refs, **zero non-test callers**), `embed/convert.go:346` (`ToList`, the embedding API — a genuine external boundary), `effects/testctx/mock_context.go:354` (`GetList`, test helper). Round-1 doc cited a wrong path for the third site; corrected |
| V35 [R] | `std/list` builder shapes, classified by slack direction | Read `std/list.ail` (whole file) | **Right-recursion** `[small] ++ recursive-result` (front slack helps): `concat`:40, `zip`:49, `mergeBy`:92-93, `take`:103, `flatMap`:205, `mapE`:235, `filterE`:246, `flatMapE`:268. **Left-append** `concat(growing, [x])` (front slack does NOT help): `reverse`:33 — the only one; fixed by delegation instead |
| V36 [R] | Base cost of the `++` right-recursion family (`take` over a consed list), for AC-9 | `/tmp/ailang-iter228 check` (✓, MOD010 temp-path note only), then `/usr/bin/time -l /tmp/ailang-iter228 run --caps IO takebench.ail` ×3 (min) + `-memprofile` + `go tool pprof -top -sample_index=alloc_space` — takebench = `llen(take(4000, gen(8000, [])))`, default flags | prints `4000`; min-of-3 peak RSS **784,531,456 B**; 0.45 s; pprof: `listConcatImpl` **130.62 MB (18.27%)** flat + `listConsImpl` 474.28 MB (66.33%) of 715.02 MB total |
| V37 [R] | No `listmut` analyzer or vettool wiring exists at base (negative + control) | `ls tools/linters` → no such dir; `grep -rn 'listmut\|vettool' Makefile make/` → 0 hits; control `grep -c 'lint' make/code-health.mk` | control: 19 — the instrument sees the existing lint wiring (`make lint` = golangci-lint, `make/code-health.mk:68`; `ci` runs `vet lint …`, `make/ci.mk:11`); no listmut anywhere |
| V38 [R] | Base status of `go test -race` on the two touched packages | `go test -race ./internal/eval/ ./internal/builtins/` | `ok` 1.9 s / `ok` 54.8 s — green at base; AC-6's improvement claim is therefore the NEW tests' existence + their demonstrated failure against a racy variant, not the flag itself |

## Goals

**Primary Goal:** building a list of n elements by prepending costs O(n) total allocation and
peak RSS, with zero observable semantic change.

**Success Metrics** (each is an Acceptance Criterion below, with its measured base value):
- Peak RSS of the repro at n=12,800 drops from 1,467.9 MB to ≤ 150 MB.
- pprof `alloc_space` attribution of `listConsImpl` at n=12,800 drops from 1,289.63 MB (94.97%) to < 50 MB.
- `std/list.reverse` of 2,000 elements drops from 10.2 s / 172.7 MB to < 1 s / ≤ 100 MB.
- The `[x] ++ rest` family (`std/list.take` over a consed list, V36) drops from 784.5 MB RSS /
  130.62 MB `listConcatImpl` flat alloc to ≤ 150 MiB RSS / < 20 MB flat — this is the metric that
  proves the arena chain bootstraps across the `++` recursion (quorum objection 2).
- RSS growth becomes linear: doubling n at most ~2.5× the RSS delta (base ratio: 3.72).
- An aliased write to a shared `Elements` slice cannot land silently: the `listmut` CI gate flags
  seeded violations (quorum objection 1; base: no such gate exists, V37).

## Sprint Split & Ordering (F1 vs F5)

**Decision: TWO sprints. This doc is Sprint 1 (quadratic representation, F1). Sprint 2 (no TCO,
F5) is a separate follow-up design doc, `m-eval-tail-calls`, to be authored after Sprint 1 lands.**

Why two, not one:

- **Different subsystems, different risk.** F1 lives in the value representation +
  `internal/builtins` (allocation behavior, concurrency invariant). F5 lives in the evaluator's
  control flow (`evalCoreApp`/`evalCoreIf` self-tail-call looping, with prior art in the VM — 11
  files, V4). Neither shares code with the other; bundling couples two independently revertible
  risks in one change.
- **Sizing.** Sprint 1 is honestly 4–5 days with tests, the enforcement gate, and benchmarks;
  F5 is 2–3 more; combined they exceed the sprint cap, and the decomposition rule (skill
  requirement 6) then applies anyway.
- **The interaction is real but bounded, and is why order matters.** Fixing F1 alone still leaves
  accumulator recursion capped at 10,000 frames by default (works at any n with
  `--max-recursion-depth`). Fixing F5 alone leaves quadratic memory — at large n the OOM lands
  before depth does (F4). They compose; neither subsumes the other.

Why F1 first:

1. F1 is the reported harm: #676's pipeline OOMed at 10,570 MB; RSS is the binding constraint at
   realistic n. F5 has a working user-facing mitigation today (`--max-recursion-depth`); F1's only
   mitigation is rewriting user code.
2. Sprint 1's stdlib fixes (`reverse`, `[x] ++ rest` fast path) deliver value regardless of TCO —
   `reverse` at n=2,000 is 10.2 s *today* with no recursion-cap involvement.
3. After Sprint 1, the depth cap is the *only* remaining wall for the repro, which makes Sprint
   2's acceptance criterion clean: repro n=12,800 at **default** flags goes from `RT_REC_003` to
   printing `12800`.

**What each unblocks on its own:** Sprint 1 unblocks email-parse-class workloads at linear memory
(any n with the flag; n ≤ 10,000 with defaults) and makes `std/list.reverse`/`take`/`zip` usable
at scale. Sprint 2 unblocks default-flag deep recursion for **all** accumulator recursion (not
just lists) and retires the misleading `RT_REC_003` hint for good.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `Elements []Value` stays the fully-materialized logical view; NO cons cells, NO accessor layer | Avoids a whole-repo break: 902 `.Elements` reads + 386 literal constructions stay valid (V7); Conflict Surface shrinks from "everything" to ~6 files + the analyzer | human | design | high |
| Amortized O(1) prepend via a hidden front-slack arena with an atomic-CAS claim; **the loser of any race copies** | The core mechanism; its concurrency soundness is the sharpest constraint in the repo (V24) | human | design | high |
| **Slow paths SEED: every list returned by `::` or `++` is arena-backed** (fresh arena + front slack on the copy path, geometric growth) | Without seeding, the `[x] ++ rest` recursion hits a nil-arena base case and the nil arena propagates up the whole chain — `std/list` stays O(n²) (quorum objection 2, gemini-3-1-pro: **correct as round 1 was written**). Seeding makes the bootstrap hold by construction; see the bootstrap section | human | design | high |
| **Element immutability is ENFORCED (listmut CI gate + copy-on-escape), not observed by grep** | `Elements` is a raw exported `[]Value`; Go has no read-only slices, and under a shared arena an aliased write corrupts *older logical versions* and can race readers — a strictly worse failure than under copy-on-cons (quorum objection 1, gpt5-6-sol: correct). "Nobody writes today" (V32/V33) is a fact, not an invariant | human | design | high |
| Literal `ListValue{Elements: …}` constructions (arena = nil zero value) always take the copy path | Zero-value is semantically correct ⇒ all 386 construction sites are safe-by-default; a missed site is a missed optimization, never a bug | human | design | med |
| The arena does NOT propagate through pattern-tail / subslice construction in Sprint 1 | Keeps `eval_patterns.go:255` and all aliasing sites untouched; bounds the invariant surface | human | design | med |
| `concat_List` gains the claim fast path for a small left operand **and the seeding slow path** | Together these — not `::` — are what make `std/list`'s right-recursion family (`[x] ++ rest`, V35) linear; front slack cannot help the left-append shape (`reverse`), which delegates instead | human | design | med |
| F1/F5 are two sprints; F1 first (this doc) | Sizing cap + independent risk; see Sprint Split | human | design | low |
| VM `OpCons` stays O(n) this sprint (documented divergence, V22) | `ailang run` doesn't use the VM; asymptotic parity across engines is deferred | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved (all are decided by this doc; the design
review / quorum gate ratifies them):

- [x] Representation: materialized `Elements` view + hidden `{arena *listArena, start int}` fields — not cons cells, not lazy materialization (see Options).
- [x] Concurrency protocol: single atomic CAS watermark per arena; claim-or-copy; **no path may ever write a buffer slot that any existing view can read**; a Go-memory-model comment and `-race` tests are part of the deliverable, not optional.
- [x] Zero-value safety: nil arena ⇒ copy path ⇒ today's exact behavior.
- [x] **Seeding: every list returned by `::` or `++` is arena-backed** — the copy/slow path allocates a fresh arena with front slack (geometric growth), never a bare slice. This is what makes the `[x] ++ rest` chain bootstrap (quorum objection 2).
- [x] **Immutability is enforced, not observed**: the `listmut` analyzer gates CI with seeded-violation tests, and the three raw-slice escape boundaries (V34) copy when the list is arena-backed (quorum objection 1).
- [x] No arena propagation through `eval_patterns.go:255` pattern tails or any other subslice-constructed list in Sprint 1.
- [x] Scope: `listConsImpl` + `listConcatImpl` (fast path **and** seeding slow path) + `listmut` gate + escape-boundary copies + `std/list.reverse` delegation + RT_REC_003 message text. Nothing else changes behaviorally.

## Solution Design

### Options considered

**Option A — true cons cells with structural sharing.** No aliased element write exists at HEAD
(V32/V33 — the widened instrument, not the retired V5/V6 greps), so sharing is *semantically* safe
today. But
`Elements` would no longer be a materialized slice: every one of the 902 `.Elements` references
and 386 literal constructions breaks or needs an accessor layer (V7); the inventory found ~15
switch statements where `ListValue`/`ArrayValue`/`TupleValue` arms are symmetric and would
diverge; indexed builtins (`_list_nth`, bounded map, JSON encoders) become O(n) or need
materialization caching. Not a 3–4 day change — a multi-week rewrite with a long regression tail.
**Rejected for Sprint 1** (remains the long-term "right" representation if the language ever adds
in-place mutation pressure; revisit only with data).

**Option B — front-slack arena behind the existing slice view (CHOSEN, with enforcement).** Keep
`Elements []Value` as the logical view; add hidden fields so a cons onto the most-recently-claimed
view writes into reserved slack in O(1), and *every other* cons/concat copies **into a freshly
seeded arena** (never a bare slice — the seeding is what makes `std/list`'s `++` recursion
bootstrap, see below). All 902 consumers compile and behave unchanged; blast radius is ~6 files
plus the analyzer. This was sketched (unverified) in the #676 triage; this doc adjudicates its
invariants against the re-derived facts and the Conflict Surface inventory — the concurrency
constraint (V24) forced the claim protocol to CAS-with-loser-copies.

Option B's safety does **not** rest on the observation that nobody writes `Elements` today
(V32/V33): that is a fact about HEAD, not an invariant, and the arena raises the cost of a future
violation — under copy-on-cons an accidental in-place write corrupts one list; under a shared
arena it corrupts an older logical version of the list too, and can race a reader. The design
therefore ships its own guard: a CI-mechanical analyzer plus copy-on-escape at the raw-slice
boundaries (see [Immutability enforcement](#immutability-enforcement-mechanical)). Why not the
stronger-sounding alternatives:

- *Unexport `Elements` behind accessors*: churns 902 references across 386 construction sites
  (V7) for **no type-level gain** — Go has no read-only slices, so a getter returning the raw
  slice enforces exactly as much as the field does. It creates a choke point the analyzer already
  provides without the churn. Rejected.
- *Copy-on-read at every access*: makes `_list_nth` and every indexed read O(n) — recreating the
  quadratic class this sprint removes. Rejected. Copying is confined to the three **escape**
  boundaries (V34), where the slice leaves the audited repo surface, and only for arena-backed
  lists (nil-arena lists — everything at HEAD — pay zero).
- *Confine arenas to provably non-escaping values*: no escape analysis exists over `eval.Value`
  and building one is not a 4–5-day change; the accumulator lists that matter here *do* flow
  through module boundaries. Rejected as the primary guard; the analyzer + boundary copies bound
  the same risk mechanically.

**Option C — hybrid representation with lazy materialisation.** A cons-chain that materializes to
a slice on first indexed access requires field access to trigger code; `Elements` is a plain
struct field read at ~230 sites, so this degenerates to Option A plus a cache. **Rejected**: same
blast radius as A with more state.

**Option D — do nothing to `::`; fix F5 and teach `foldr`/builtins in the prompt.** Refuted by
the systemic audit: `foldr` is itself user-level recursion (depth-capped at 10,000) and
quadratic-if-consing; **`std/list` itself builds quadratically** via `[x] ++ rest` (V17) — the
stdlib would be lying about the idiom it teaches; and #676 is not behind any prompt we control.
Its one good idea ships anyway: stdlib hygiene (`reverse` delegation, V19/V20) is Phase 3 of this
sprint. **Rejected as the primary fix.**

### Architecture (Option B)

```go
// internal/eval/value.go — additive; zero value of both new fields = today's behavior
type ListValue struct {
    Elements []Value    // the logical view — UNCHANGED for every existing consumer
    arena    *listArena // nil ⇒ this list never participates in the fast path
    start    int        // index of Elements[0] within arena.buf (0 when arena == nil)
}

// internal/eval/list_arena.go (new)
type listArena struct {
    buf []Value
    low atomic.Int64 // claim watermark: lowest buf index owned by any live view
}
```

**Prepend** (`eval.(*ListValue).Prepend(head Value) *ListValue`, called by `listConsImpl`):

1. **Fast path**: `l.arena != nil && l.start > 0 && l.arena.low.CompareAndSwap(l.start, l.start-1)`
   → write `buf[l.start-1] = head`; return
   `&ListValue{Elements: buf[l.start-1 : end : end], arena: l.arena, start: l.start - 1}`.
2. **Slow path — always seeds** (nil arena, exhausted slack, or lost CAS): allocate a fresh arena
   with front slack proportional to the new length (`slack = len+1`, geometric doubling across
   re-arenas ⇒ amortized O(1)); copy once; watermark = slack index. The result is arena-backed —
   **`Prepend` never returns a bare-slice list.**

**Why this is safe** (the invariants the Conflict Surface checks against):

- *Write-once below all views*: a slot is written exactly once, at claim time, at an index
  strictly below `start` of every pre-existing view (`low` only decreases; every view's `start`
  ≥ the watermark at its creation). No existing view can ever read a slot the fast path writes.
  That no *other* code writes elements is **enforced by the `listmut` CI gate** (below) — not
  assumed from the V32/V33 observation that no such write exists at HEAD.
- *Race behavior is copy, not corruption*: two goroutines consing onto the same shared list
  (possible — module-level lists are shared across `Fork()` goroutines, V24) race on one CAS;
  the loser takes the copy path. Publication of the winner's new view to another goroutine
  carries the normal happens-before edge of whatever channel/effect passes the value. A
  memory-model comment plus `-race` tests over a Fork-shared list are mandatory deliverables.
- *Capacity clipping*: views are constructed with the three-index form `buf[s:end:end]`, so any
  caller `append`ing to an escaped `Elements` slice reallocates instead of stomping the arena.
  The clip stops append-stomps only — it does **not** stop `xs[i] = v` through an alias (the
  round-1 doc over-claimed this; reviewer gpt5-6-sol was right). Index-writes are what the
  enforcement layer below exists for.
- *Retention bound*: a small suffix view pins its arena buffer (≤ ~2× the built list's size).
  This aliasing class already exists at base — the pattern-match tail (V15) pins its parent's
  backing array today. Documented in LIMITATIONS.md, not silently.

**`concat_List` — fast path + the same seeding slow path**: when the right operand has an arena
with ≥ `len(left)` slack at its watermark, claim `len(left)` slots in one CAS
(`low.CompareAndSwap(start, start-k)`) and copy only the left operand — O(len(left)). Otherwise
(nil arena, insufficient slack, lost CAS) it does **not** fall back to the round-1 "legacy copy":
it copies both operands into a **freshly seeded arena** with front slack ∝ the result length,
exactly like `Prepend`'s slow path. Empty left operand returns the right operand unchanged.

#### Arena bootstrap across the `++` recursion

Quorum objection 2 (gemini-3-1-pro) was correct against the round-1 design: a concat slow path
that returns a nil-arena list poisons the entire recursive chain — the base case (`[]`/literals)
has a nil arena by definition, so without seeding, *every* level of `[x] ++ recurse(...)` misses
the fast path and `std/list` stays O(n²). The fix is the seeding rule above. By construction:

1. **Base case**: `take(0, _)` returns `[]` — nil arena, as before.
2. **First level up**: `[x] ++ []` takes the slow path → seeds arena `A₀` (slack ∝ 1), returns a
   view with `start == A₀.low`. The nil arena never propagates: the first `++` above any base
   case *replaces* it with a seeded one.
3. **Each subsequent level**: the right operand is the value the recursion *just returned*, so
   its `start` equals the current watermark and the CAS `low: start → start−k` succeeds — an
   O(k) claim (k = len(left), i.e. 1 for `[x] ++ rest`).
4. **Slack exhaustion**: the claim fails for lack of room → re-seed with slack ∝ current length
   (geometric). A re-seed at length L copies L elements, and at least L elements' worth of slack
   was consumed since the previous seed ⇒ total copy work across all re-seeds is O(n): the
   classic doubling argument. Total cost of the chain: O(n) claims + O(n) re-seed copies =
   **amortized O(1) per element**, for any per-step left-operand size k (covers `flatMap`, where
   k = len(f(x))).
5. **Interference**: if a concurrent claimer wins the CAS between two levels (only possible when
   the same list value is shared across goroutines mid-recursion), the loser re-seeds — degrading
   that step to a copy, i.e. to today's behavior, never worse. Correctness never depends on
   winning.

**The front-slack asymmetry, stated plainly** (glossed in round 1): front slack accelerates
growth at the *front* — `x :: acc` and `small ++ growing` where the growing list is the **right**
operand. It does nothing for append-at-the-**end**: in `concat(growing, [x])` the growing list is
the *left* operand and the new element lands past the buffer's high end, where there is no slack,
whatever the arena state. By the V35 classification, every `std/list` builder except one is
right-recursion (`concat`, `zip`, `mergeBy`, `take`, `flatMap`, `mapE`, `filterE`, `flatMapE`) —
covered by the seeded claim path. The single left-append builder is `reverse`
(`concat(reverse(rest), [x])`), which the mechanism genuinely cannot help — it is fixed by
delegation to the existing iterative `_list_reverse` builtin instead (V19/V20). AC-9 (a `take`
benchmark with a measured failing base, V36) is the acceptance test that the bootstrap actually
holds; AC-5 covers `reverse` separately.

#### Immutability enforcement (mechanical)

Quorum objection 1 (gpt5-6-sol) stands against any grep: `Elements` is a raw exported `[]Value`,
and Go cannot make a slice read-only, so "nobody writes today" cannot carry Option B. Two layers,
both CI-gated, replace caller discipline:

1. **`listmut` analyzer** (`tools/linters/listmut/`, a standard `go/analysis` vetter; new make
   target wired into `ci` alongside `vet lint`, `make/ci.mk:11`):
   - **R1 — no derived index-writes**: flags any store `s[i] = v` (or `s[i] op= v`) where `s`
     derives intra-procedurally — through assignments, `:=` bindings, and slice expressions —
     from a `.Elements` selector on `eval.ListValue`. Catches both the direct form and the
     aliased form (`e := lv.Elements; e[i] = v`) that killed V5.
   - **R2 — no unaudited hand-offs**: flags a `.Elements`-derived slice passed as a `[]Value`
     argument or returned from a function, unless the callee/function is on an explicit,
     in-repo allowlist (audited read-only consumers; `len`, `range`, `copy`-as-source,
     `append(dst, s...)` spreads are recognized as reads). Bounds the intra-procedural limit of
     R1: a slice cannot silently cross a function boundary into unanalyzed writes.
   - **R3 — frozen escape set**: the set of functions returning a raw `Elements` slice is pinned
     to the three V34 sites; a new one fails the build until allowlisted in review.
   - **Seeded-violation fixtures**: a test package containing a direct write, an aliased write,
     and an interprocedural hand-off; a Go test asserts the analyzer reports all three
     (AC-10). The gate is proven to fire, not assumed to.
   - **Honest residual**: `unsafe`, reflection, and flows the intra-procedural taint cannot see
     remain possible in principle. The backstop is layer 2 plus the kill-switch (deleting the
     fast-path branch restores exact base behavior).
2. **Copy-on-escape at the boundaries** (V34): `builtins.SafeAsList` (zero non-test callers
   today), `embed.ToList` (the embedding API — code outside this repo's audit surface), and
   `testctx.GetList` return a defensive copy **iff the list is arena-backed** (via a small
   exported `eval` helper, e.g. `(*ListValue).SharedBacking() bool`). Nil-arena lists — every
   list that exists at HEAD — return the raw slice unchanged: zero cost, zero behavior change.
   Arena-backed lists pay one O(n) copy at the moment they leave the enforced surface, which is
   the price of making an external `xs[i] = v` provably unable to corrupt a second logical list
   version or race a reader.

**Stdlib hygiene**: `std/list.reverse` becomes a delegation to the existing iterative
`_list_reverse` builtin (V19/V20), same pattern as `map`→`_list_map` at `std/list.ail:56` (V8).

**Error-text honesty**: `eval_operations.go:58` stops advertising the nonexistent "enable tail
recursion" (V11); until Sprint 2 lands it should point at `--max-recursion-depth` and the
iterative `std/list` builtins. No test pins the hint text (V30).

### Implementation Plan

**Phase 1: Arena + Prepend (seeding included)** (~1 day)
- [ ] `internal/eval/list_arena.go`: `listArena`, `Prepend` (fast path + seeding slow path),
      memory-model comment (~150 LOC)
- [ ] `internal/eval/value.go`: additive fields + `SharedBacking()` helper + invariant doc comment (~20 LOC)
- [ ] `internal/builtins/list.go`: `listConsImpl` delegates to `Prepend` (~-8/+4 LOC)
- [ ] `internal/eval/list_arena_test.go`: unit tests (claim/exhaust/lost-CAS/nil-arena/seeding —
      including "slow path result is arena-backed"), property test (mixed cons/match/concat
      sequences vs a reference implementation), `-race` test over a Fork-shared module-level
      list (~250 LOC)

**Phase 2: Concat fast path + seeding + measurements** (~1 day)
- [ ] `listConcatImpl` small-left claim path + seeding slow path (same `eval` method surface)
- [ ] Bootstrap test: an AILANG-level `[x] ++ …` recursion (the V36 takebench shape) must show
      O(n) allocation — this is the direct regression test for quorum objection 2
- [ ] Reproduce AC-1…AC-5 + AC-9 measurements; record before/after in this doc's Implementation Report

**Phase 3: Immutability enforcement** (~1 day)
- [ ] `tools/linters/listmut/`: `go/analysis` analyzer, rules R1–R3 (~250 LOC + fixtures)
- [ ] Seeded-violation fixture package + Go test asserting all three violations are reported (AC-10)
- [ ] Make target (`vet-listmut`) added to `ci` prerequisites in `make/ci.mk`
- [ ] Copy-on-escape in `builtins.SafeAsList`, `embed.ToList`, `testctx.GetList` (arena-backed
      lists only) + a test that a write through an escaped slice cannot alter any list view

**Phase 4: Stdlib + message** (~0.5 day)
- [ ] `std/list.ail`: `reverse` delegates to `_list_reverse` (1 line; verify with `ailang check`
      + stdlib goldens)
- [ ] `internal/eval/eval_operations.go:58`: message text (1 line)

**Phase 5: Regression surface + docs** (~0.5–1 day)
- [ ] `make test`, `make verify-examples`, fixtures from [Programs that MUST still work](#programs-that-must-still-work)
- [ ] `CHANGELOG.md`; `docs/LIMITATIONS.md` gains the depth-cap + retention notes (V28: currently absent)
- [ ] Implementation Report in this doc

### Files to Modify/Create

**New files:**
- `internal/eval/list_arena.go` (~150 LOC) — arena type, claim protocol (fast paths + seeding slow
  paths), memory-model comment
- `internal/eval/list_arena_test.go` (~250 LOC) — unit + property + bootstrap + `-race` tests
- `tools/linters/listmut/` (~250 LOC + fixtures) — the R1–R3 analyzer + seeded-violation test

**Modified files:**
- `internal/eval/value.go` (+20 LOC) — additive hidden fields on `ListValue` + `SharedBacking()`
- `internal/builtins/list.go` (+10/−10 LOC) — `listConsImpl`/`listConcatImpl` delegate to the arena path
- `internal/builtins/safe_cast.go`, `internal/embed/convert.go`,
  `internal/effects/testctx/mock_context.go` (+~4 LOC each) — copy-on-escape for arena-backed lists
- `make/ci.mk`, `make/code-health.mk` (+~6 LOC) — `vet-listmut` target in the `ci` chain
- `internal/eval/eval_operations.go` (+1/−1 LOC) — RT_REC_003 hint text
- `std/list.ail` (+1/−5 LOC) — `reverse` delegation
- `docs/LIMITATIONS.md` (+~10 LOC) — depth cap, retention note
- `CHANGELOG.md` — entry under v0.34.0

## Conflict Surface

This change touches `internal/eval/` (value representation) — this section is mandatory. The
"position" being extended is not syntactic but representational: *who observes the concrete shape
of `eval.ListValue`?* Inventory: full-repo sweep by a read-only subagent, with every load-bearing
row spot-verified first-party (V15, V22–V26).

### What else lives here

| Consumer class | Sites (anchor) | Why unchanged / what must hold |
|---|---|---|
| **Direct `.Elements` reads** (len, index, range): equality, show, JSON, effects ABI, WASM bridge, XML (~35 sites), set builtins | ~230 non-test sites; e.g. `eval_patterns.go:218-244`, `builtins_json.go`, `effects/net.go:432-759` | `Elements` stays a fully-materialized `[]Value` — reads are bit-identical. This is the design's core conflict-avoidance move. |
| **Literal constructions** `&ListValue{Elements: …}` | 386 sites (V7) | Zero value of the new fields = nil arena = copy path = today's exact behavior. A construction site that never learns about the arena is *correct*, merely unoptimized. |
| **Pattern-match tail alias** | `eval_patterns.go:255` (V15) | Untouched. Tail views get nil arena (frozen decision), so a cons onto a matched tail copies — safe. The alias stays O(1). |
| **Concurrency: Fork-shared module-level lists** | `eval_evaluator.go:166-170`, `env.go:6-14` (V24) | The sharpest constraint. Naive mutable slack = data race. The CAS claim protocol confines writes to slots no existing view can read; racing claimers lose to the copy path. `-race` test mandatory. |
| **Raw-slice escapes** | `builtins/safe_cast.go:97` (V25/V34 — zero non-test callers), `embed/convert.go:346` (`ToList`, external embedding API), `effects/testctx/mock_context.go:354` (`GetList`) | All three copy-on-escape when arena-backed (nil-arena = raw, zero cost); capacity clip additionally defuses appends; the escape set itself is frozen by analyzer rule R3. An external `xs[i] = v` can then only touch a private copy. |
| **Subslice-aliasing constructions** | `eval_patterns.go:255`, `testing/shrink.go:212-217` | Construct plain `ListValue`s over shared backing — nil arena ⇒ correct. |
| **Capacity-hint sites** `make(…, len(x.Elements))` | `list_iterative.go:79,142`, `list_set.go`, `embed/convert.go` | Read `len` only; unaffected. |
| **Telemetry/trace** | `telemetry/effect_spans.go:75-76` (V26); `internal/trace` imports no `eval` | Length-only read; clean flank. |
| **Bytecode VM** | own repr (`bytecode.ListObj`); bridge deep-copies both ways (`bytecode_bridge.go:132-142,191-200`, V23) | Fully decoupled. VM `OpCons` stays O(n) (V22) — documented divergence, non-goal. |
| **Codegen / emit-go** | `internal/gen/` has zero `internal/eval` imports (V29) | No conflict. |
| **`ArrayValue` / `TupleValue`** | structurally similar, behaviorally separate; only bridge is `array.go:379-434` full copies | Untouched; their symmetric switch arms don't diverge because `ListValue`'s public shape is unchanged. |
| **Identity comparison of `*ListValue`** | none found (inventory) | Equality is structural everywhere; shared backing cannot change observable behavior. |

### Disambiguation strategy

There is no parser-level ambiguity; the "disambiguation" is the runtime invariant: **the fast
path may only write `buf[i]` after a successful CAS of the watermark onto `i`, and every view's
`start` is ≥ the watermark value at that view's creation.** Everything below the watermark is
unreachable from any live view; everything at-or-above is write-never — held today by V32/V33 and
held tomorrow by the `listmut` gate + copy-on-escape, not by convention. The invariant is
enforced in exactly one function (`Prepend` / the concat variant) and documented at the type.

### Programs that MUST still work

Regression fixtures (all verified to exist, V31; they exercise cons, list patterns, and
list-heavy recursion):

1. `examples/first_non_repeat.ail`
2. `examples/inline_tests_recursive.ail`
3. `examples/record_cons_pattern.ail`
4. `examples/record_list_extraction.ail`
5. `examples/pattern_matching_adt.ail`
6. `std/list.ail`'s own users via `make verify-examples` + the stdlib test surface (`go test ./internal/...`)

### What deliberately changes

- Memory/allocation behavior of `::` and `++` (the point of the change). No output of any
  program changes.
- The three raw-slice escape helpers (V34) return a private copy instead of an alias **for
  arena-backed lists only** — observable solely to a caller that mutates through the escaped
  slice, which is exactly the behavior being made impossible; nil-arena lists (everything at
  HEAD) are byte-identical.
- `std/list.reverse` becomes iterative: same results, ~n× faster, no longer consumes O(n) stack
  (its 10.2 s / n=2,000 base is AC-5).
- The RT_REC_003 hint text (V11, V30 — no test pins it).
- Anything else that changes observable behavior is a regression, not an intended change.

## Examples

### Example 1: the #676 repro (before/after)

**Before** (base, V10/V12): `ailang run --caps IO repro.ail` at n=12,800 → `RT_REC_003`; with
`--max-recursion-depth 200000` → prints `12800`, peak RSS 1,467.9 MB.

**After Sprint 1**: same commands; the flag-assisted run prints `12800` at ≤ 150 MB. (The
default-flags run still hits the depth cap — that is Sprint 2's exit criterion, not this one's.)

### Example 2: stdlib reverse

**Before** (V21): `reverse` of 2,000 elements = 10.2 s, 172.7 MB.
**After**: `std/list.ail` delegates — `export pure func reverse[a](xs: [a]) -> [a] = _list_reverse(xs)` —
< 1 s, ≤ 100 MB, same output. (Same delegation pattern as `map` at `std/list.ail:56`, V8.)

## Success Criteria (Acceptance)

Every AC names its command and the **measured value on unmodified `origin/dev` = `88631976e`**
(V12/V13/V21/V36/V37/V38), so each can fail; the two ACs whose base state is *green* (AC-6's
`-race` flag, AC-7) are explicitly labeled for what they are — regression gates whose improvement
content is the new artifacts' existence. RSS thresholds are absolute with headroom (machine: the
dev Mac Studio; run 3×, take the minimum).

- [ ] **AC-1** `/usr/bin/time -l ailang run --caps IO --max-recursion-depth 200000 repro12800.ail`
      prints `12800`, peak RSS ≤ 157,286,400 B (150 MiB). **Base: prints `12800`, 1,467,875,328 B — FAILS today.**
- [ ] **AC-2** `go tool pprof -top -sample_index=alloc_space` on that run: `listConsImpl` flat
      alloc < 50 MB. **Base: 1,289.63 MB (94.97%) — FAILS today.**
- [ ] **AC-3** repro at n=6,400, default flags: prints `6400`, peak RSS ≤ 104,857,600 B (100 MiB).
      **Base: prints `6400`, 428,654,592 B — FAILS today.**
- [ ] **AC-4** linearity: (RSS@12,800 − RSS@hello) / (RSS@6,400 − RSS@hello) ≤ 2.5.
      **Base: (1,467,875,328−46,432,256)/(428,654,592−46,432,256) = 3.72 — FAILS today.**
- [ ] **AC-5** revtest.ail (`reverse` of 2,000, snippet type-checked at base, V21): prints `2000`
      in < 1 s wall with peak RSS ≤ 104,857,600 B. **Base: 10.2 s, 172,687,360 B — FAILS today.**
- [ ] **AC-6** `go test -race ./internal/eval/ ./internal/builtins/` green, **including the new
      arena unit/property/bootstrap/Fork-race tests**. **Base: both packages `ok` under `-race`
      (V38) — the improvement claim is the new tests' existence; they FAIL today by absence, and
      each must be demonstrated to fail against a deliberately racy claim variant during review.**
- [ ] **AC-7** *(regression gate, not an improvement claim)* `make test` and `make verify-examples`
      green. **Base: passes by design at `88631976e` — this AC exists to fail against the change,
      not against the repo. (`make verify-examples` re-run first-party at base: exit 0,
      "192 modules checked, 0 drift", one pre-existing stale-manifest note
      (`lambda_expressions.ail` missing on disk) not introduced by this change.)**
- [ ] **AC-8** RT_REC_003 message no longer says "enable tail recursion"; existing
      `recursion_test.go` still green (it pins only the code substring, V30). **Base: message
      contains the phrase (V11) — FAILS today.**
- [ ] **AC-9** *(the `++` bootstrap, quorum objection 2)* takebench
      (`llen(take(4000, gen(8000, [])))`, V36) at default flags: prints `4000`, peak RSS
      ≤ 157,286,400 B (150 MiB, min of 3), and pprof `alloc_space` flat for `listConcatImpl`
      < 20 MB. **Base: prints `4000`, min-of-3 RSS 784,531,456 B, `listConcatImpl` 130.62 MB
      flat — FAILS today on both.**
- [ ] **AC-10** *(the aliased-write gate, quorum objection 1)* the `listmut` target runs in
      `make ci`, and its Go test proves the seeded-violation fixtures (direct write, aliased
      write, interprocedural hand-off) are all reported; removing any rule fails the test.
      **Base: no analyzer, no target, no fixtures exist (V37) — FAILS today by absence.**
- [ ] **AC-11** Documentation updated (CHANGELOG.md; LIMITATIONS.md gains depth-cap + retention
      notes — V28: absent at base, FAILS today).

## Testing Strategy

**Unit tests** (`internal/eval/list_arena_test.go`):
- Claim/exhaust/lost-CAS/nil-arena paths; capacity clip (an `append` to an escaped `Elements`
  must not alter any other view); **seeding: every `Prepend`/concat slow-path result is
  arena-backed** — the direct unit-level pin of the bootstrap rule.
- Bootstrap test (objection 2's regression test): an n-deep `[x] ++ …` chain from a nil-arena
  base performs O(n) total element copies (counted via the reference implementation), not O(n²).
- Copy-on-escape: writing through a slice obtained from `SafeAsList`/`ToList`/`GetList` on an
  arena-backed list alters no list view.
- Property test: random interleavings of cons / pattern-tail-style subslicing / concat vs a
  naive copy-only reference — structural equality after every step.

**Analyzer tests** (`tools/linters/listmut/`):
- Seeded-violation fixtures: direct `lv.Elements[i] = v`, aliased `e := lv.Elements; e[i] = v`,
  and an interprocedural hand-off to a writing callee — all three must be reported (AC-10).
- A clean-fixture control (read/range/len/subslice uses) must produce zero reports.

**Concurrency tests:**
- `-race`: N goroutines cons onto one shared list (module-level, via `Fork()`-style setup per
  V24); assert every resulting list is structurally correct.

**Regression-surface tests** (required — Conflict Surface is non-empty):
- One run per fixture in [Programs that MUST still work](#programs-that-must-still-work), output pinned.
- Full `make test` + `make verify-examples`.

**Manual:**
- AC-1…AC-5 measurement script (commands are in the Verification Log; keep in the sprint log).

## Deferred Decisions

Intentionally open for the implementer:

- Slack sizing/growth constants for BOTH seeding sites (`Prepend` and concat slow paths; must
  keep amortized O(1) and ≤ ~2× retained overhead; document the chosen constants) — agent.
- Exact method surface on `eval` (`Prepend` vs `Cons`; where the concat variant lives; the name
  of the `SharedBacking()` predicate) — agent.
- Small-left threshold for the `concat_List` fast path (any k with available slack is sound) — agent.
- `listmut` allowlist contents and maintenance policy (which audited read-only callees R2
  recognizes; kept in one file next to the analyzer) — agent.
- Optional `AILANG_LIST_ARENA=off` debug escape hatch (perf toggle, not a data-integrity fallback; if added, document in the dev-workflow debug-flag table) — agent.
- Benchmark harness location (ad-hoc script vs `tools/`) — agent.

## Non-Goals

- **Tail-call elimination (F5)** — Sprint 2, separate doc `m-eval-tail-calls` (see Sprint Split).
- **VM `OpCons`** — stays O(n) (V22); `ailang run` doesn't use the VM; parity deferred with the divergence documented.
- **`ArrayValue` / `TupleValue`** — untouched.
- **Rewriting the rest of `std/list` to Go builtins** — only `reverse` delegates (the sole
  left-append builder, which front slack cannot help, V35); the `[x] ++ rest` family becomes
  linear via the concat fast path **plus the seeding slow path** instead (bootstrap section, AC-9).
- **Persistent data structures (cons cells, RRB trees)** — Option A, rejected for blast radius; revisit only with new pressure (e.g. in-place mutation).
- **Arena propagation through pattern tails** — future work; Sprint 1 freezes it off.

## Timeline

**Day 1**: Phase 1 (arena, Prepend with seeding, unit + race tests).
**Day 2**: Phase 2 (concat fast path + seeding, bootstrap test; AC-1…AC-5 + AC-9 measured; fix
what the numbers say).
**Day 3**: Phase 3 (`listmut` analyzer, fixtures, make wiring, copy-on-escape).
**Day 4**: Phase 4 + Phase 5 start (stdlib reverse, message text, regression surface).
**Day 5**: buffer — full `make ci`, docs, Implementation Report.

**Total: 4–5 days.** (Sprint 2 estimated separately at 2–3 days in its own doc.)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Data race in the claim path | High | CAS-only claim; loser copies; write-once-below-all-views invariant in ONE function; Go memory-model comment; mandatory `-race` tests incl. Fork-shared list; kill-switch = delete the fast-path branch (copy path IS base behavior) |
| A future aliased write to a shared `Elements` slice (corrupts older list versions / races readers — worse than under copy-on-cons) | High | NOT left to discipline: `listmut` R1–R3 gate CI with proven-to-fire fixtures (AC-10); copy-on-escape at the three frozen boundary sites (V34); capacity clip stops the append form; residual (`unsafe`/reflection) documented + kill-switch |
| The `++` chain fails to bootstrap and `std/list` stays quadratic (quorum objection 2) | High | Seeding rule: no `::`/`++` result is ever bare-slice; unit-level seeding pin + O(n)-copies bootstrap test; AC-9 measures the actual `std/list.take` shape end-to-end against a failing base (V36) |
| Retention: small view pins a large arena | Med | Slack bounded at creation (≤ len+1), geometric growth ⇒ ≤ ~2× overhead (transient 2× also applies to one-shot big concats, which now seed); same aliasing class as base pattern-tails (V15); documented in LIMITATIONS.md |
| Per-op overhead of always-seeding tiny lists (arena struct + atomic per slow-path cons) | Low | One small allocation alongside the copy the slow path already does; AC-3/AC-7 wall/RSS gates catch a measurable regression; kill-switch unchanged |
| A construction site somewhere bypasses the arena | Low | Nil arena ⇒ copy path ⇒ correct; only a missed optimization. AC-1/AC-3/AC-9 catch it if it matters on the hot path |
| Stdlib `reverse` behavioral drift | Low | Golden/property tests: delegation output ≡ recursive output; `make verify-examples` |
| RSS ACs flaky across machines | Low | Absolute thresholds with ≥3× headroom over the predicted post-fix footprint; 3 runs, take min; pprof AC-2 is machine-independent |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; outputs byte-identical (fast path vs copy path is observationally equivalent; race loser copies) |
| A2: Replayability | 0 | Traces unchanged (V26: telemetry reads length only) |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Invariant confined to one function + one type comment; element-immutability is machine-checked by the `listmut` CI gate rather than socially assumed |
| A6: Safe Concurrency | 0 | Neutral by construction: CAS claim + loser-copies keeps the Fork-sharing contract (V24); `-race` gate mandatory |
| A7: Machines First | +1 | Models write canonical accumulator recursion; the language stops punishing the idiom it teaches |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | `::`'s cost becomes what its type and every FP tradition imply; removes a hidden quadratic that contradicted the visible cost model |
| A10: Composability | 0 | No composition changes |
| A11: Structured Failure | +1 | RT_REC_003 stops advising a nonexistent option (V11) |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism — the claim race affects allocation strategy only, never values
- [x] A3 (Effects): no hidden side effects — arena writes are invisible to every live view by invariant
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): optimizes FOR machine-written idiomatic code

## Quorum verification log (mission iteration 228)

Two rounds were run; Gate 2 allows one revision and one re-quorum, after which a still-rejected doc
parks for a human. Both artifacts are committed at `design_docs/planned/quorum/` (the `.ailang/state/` originals are gitignored).

| Round | Artifact | `gpt5-6-sol` | `gemini-3-1-pro` | Synthesis |
|---|---|---|---|---|
| 1 | `design_docs/planned/quorum/m-list-cons-quadratic-2026-08-19T08-24-23Z.json` | reject | reject | **BLOCKED** ($0.0970) |
| 2 | `design_docs/planned/quorum/m-list-cons-quadratic-2026-08-19T08-42-16Z.json` | reject | **pass** | **BLOCKED** ($0.1349) |

Both reviewers were PRESENT in both rounds — `absent_reviewers` is empty in both artifacts, so
neither verdict is an N−1 degrade. Round 2 raised `--max-cost-usd` to 0.35 because the doc grew
39.8 KB → 60 KB and `gpt5-6-sol` has previously dropped out on the default $0.10 cap at that size.

**Round-1 objections, both addressed (see the revision):**
- `gpt5-6-sol` — the doc's safety argument rested on greps that cannot see an aliased write
  (`e := lv.Elements; e[i] = v`). The controller measured this: the *current-state* half is
  **refuted** (2 `.Elements` slice bindings in non-test Go — one a `TupleType`, one in
  `internal/testing/shrink.go` which copies before every write — against a control of 1,883–2,423
  index-writes the matcher provably sees), so zero aliased writes exist at HEAD. The *design* half
  stands: a raw exported `[]Value` makes that a fact about today, not an invariant, and the arena
  raises the cost of a future write from one-list corruption to cross-version corruption. V5/V6 are
  retired; the doc now proposes a CI-enforced `listmut` analyzer plus copy-on-escape.
- `gemini-3-1-pro` — a concat slow path returning nil-arena lists poisons the whole `[x] ++ rest`
  chain. No defence was found; the design changed. Every list returned by `::` or `++` is now
  arena-backed, and the front-slack/left-append asymmetry is stated by command. **PASSED in round 2.**

**Round-2 blocking objection — WHY THIS PARKED, and it is a design-DIRECTION dispute, so the
narrow-refinement carve-out does not apply:**

> `gpt5-6-sol`: The design does not provide amortized O(1) prepend under normal persistent-list
> branching, despite making that its title and primary goal. After one prepend claims a tail's
> watermark, every later prepend of that same still-live tail loses the CAS and copies the entire
> tail. Repeating `x :: base` while retaining `base` therefore costs Θ(m·len(base)), and analogous
> branching affects `++`. The geometric-growth proof covers slack exhaustion along a single
> linear-use chain, not CAS failures caused by aliases, concurrency, or branching.

The controller judges this **correct**, and it is the intrinsic difference between a front-slack
arena and true cons cells: cons cells are O(1) under *every* sharing pattern, an arena only along a
linear use chain. It cannot be resolved by applying a reviewer-supplied fix — the only fix is a
different representation, which the doc rejected on blast radius (902 `.Elements` references, 386
constructions).

**The fact that decides it, and it is measured:** the reported defect *is* the linear-use case.
`gen(n - 1, "constant" :: acc)` never retains the old `acc`, so Option B fixes `#676` completely.
What it does not deliver is the general guarantee the doc's title claims.

**Decision required — `D-19`, one word:**
- **A** — accept Option B as a *linear-chain* optimization: retitle and rescope the doc to the
  guarantee it actually provides, ship the `#676` fix now, and record general persistent branching
  as a named residual.
- **B** — take true cons cells / structural sharing: O(1) under all sharing, correct by
  construction, and a substantially larger sprint that must be decomposed before it is planned.

## Related Documents

Neural search on "list cons quadratic" returned no doc above 0.29 (duplicate gate: proceed).
Closest, for context only:

- [m-effectful-list-combinators-sprint-plan](../implemented/v0_7_3/m-effectful-list-combinators-sprint-plan.md) (0.29) — added `mapE`/`filterE`, which are among the `[x] ++ rest` builders this sprint makes linear
- [m-bug-list-length-sprint-plan](../implemented/v0_6_1/m-bug-list-length-sprint-plan.md) (0.26)
- `M-ITERATIVE-LIST` (referenced at `std/list.ail:55`) — prior art for the delegate-to-Go-builtin pattern this doc extends to `reverse`

## References

- **Source issue**: [sunholo-data/ailang#676](https://github.com/sunholo-data/ailang/issues/676) — email-parse's minimal repro + the controller's full triage verdict (F1–F8) in the comments; blocks their M9 (replacing Python loaders with AILANG)
- **Routing**: [design_docs/PROGRAM.md](../PROGRAM.md) §4
- [Design Axioms](/docs/references/axioms)

## Future Work

- **Sprint 2 — `m-eval-tail-calls`** (follow-up doc): self-tail-call elimination in the
  tree-walking evaluator (prior art: 11 tail-call files in `internal/vm`/`internal/bytecode`,
  V4). Exit criterion: the #676 repro at n=12,800 with **default** flags prints `12800`
  (base: `RT_REC_003`, V10). Estimated 2–3 days.
- VM `OpCons` arena parity (V22) if/when the bytecode engine fronts `ailang run`.
- Arena propagation through pattern-matched tails (would accelerate destructure-and-recons loops).
- Revisit Option A (persistent cons cells) only if in-place mutation pressure ever appears.

---

**Document created**: 2026-08-19
**Last updated**: 2026-08-19 (revision round 2 — quorum objections addressed: V5/V6 retired for
V32/V33 + mechanical `listmut`/copy-on-escape enforcement; seeding rule added so the `++` chain
bootstraps; AC-9/AC-10 added with measured failing bases V36/V37)
