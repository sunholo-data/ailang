# M-VM-VAR-PATTERN-DEFAULT-ARM — variable patterns in match arms are evaluator-only on the strict VM (unbound variable), and misplaced/guarded variable arms give **silent wrong results** on the VM and in generated Go

**Status**: Superseded by / implemented in [m-vm-match-lowering.md](m-vm-match-lowering.md) (2026-10-02, v0.51.1) — one recursive lowering fixes this doc together with its three siblings. Original status: Planned — quorum attempted 2026-10-01: **both external reviewers unreachable** (gpt5-6-sol: no `OPENAI_API_KEY` on this runner; gemini-3-1-pro: Vertex 403 `aiplatform.endpoints.predict` on `ailang-dev`), so the quorum degraded to controller-only with the absences recorded by name — NOT quorum-cleared. The 27-row first-party Verification Log (every claim backed by a live binary run or a source read at cited lines) is the load-bearing evidence; re-run quorum when a reviewer route is available.
**Target**: v0.51.2 (bug fix; strict-VM soundness, clause-2 parity family)
**Priority**: P0 — two of the four confirmed variants are **silent wrong results** under `--strict-bytecode` and in `ailang compile` Go output (no error, no fallback, wrong value, exit 0); the reported variant degrades every constructor match with a named catch-all arm to evaluator-only, breaking `--strict-bytecode`.
**Estimated**: ~1.5 days (root cause fully localized; the correct binding mechanism already exists in the same file and is reused verbatim)
**Dependencies**: none. Sibling of [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) (planned; see Related Documents for the split-of-scope contract). Builds on [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) (landed: the if-chain half of this same bug family).

**Reported from**: sprint R1-M2-JOURNEY M2.3a (stapledons-godot `sim/core.ail`), where the workaround `_ => j` had to replace the natural `other => other` catch-all.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Restores evaluator/VM/Go-codegen agreement: the same program currently returns different values on `run` vs `run --strict-bytecode` (V3, V4) and vs generated Go (V7). |
| A2: Replayability | 0 | No trace or replay machinery touched. |
| A3: Effect Legibility | 0 | Pure pattern-matching lowering only; no effects involved. |
| A4: Explicit Authority | 0 | No capability or authority surface touched. |
| A5: Bounded Verification | +1 | Moves a whole construct class (named catch-all arms) from evaluator-only to strict-VM-decidable; the strict gate stops lying. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | The named catch-all arm is the idiom an AI writes first (`other => other`); today it demands a human-shaped workaround (`_ => j`) to run on the VM. |
| A8: Minimal Syntax | +1 | No new syntax — existing, documented, evaluator-correct syntax starts working everywhere. |
| A9: Cost Visibility | 0 | No cost accounting changes. |
| A10: Composability | 0 | Neutral. |
| A11: Structured Failure | +1 | Converts two silent-wrong-result shapes into loud errors (or into correct results); interim unsupported shapes fail loudly as EvalOnly instead of computing wrong values. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): removes implicit nondeterminism (route divergence), introduces none — the routing gate is a pure structural function of the arm list.
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): removes a human-only workaround requirement.

## Problem Statement

**Current State:**

A bare lowercase identifier as a match-arm pattern (a **variable pattern**, elaborated to
`core.VarPattern` at `internal/elaborate/patterns.go:125`) is a catch-all that **binds the
scrutinee** and always matches, in source order, like every other arm. The tree-walking
evaluator implements this correctly (`internal/eval/eval_patterns.go:109-112`). The lowering
used by the strict VM (`internal/gen/lower/match.go`) does not:

1. **Unbound default (reported bug).** `lowerConstructorMatch` captures a variable-pattern
   arm's body as the switch `Default` **without binding the variable** (match.go:113-125 —
   compare the if-chain path's `armBody`, which prepends `lowerPatternBindings`, match.go:303-312).
   If the body references the binding (`other => other`), the bytecode compiler reports
   `compiler: unbound variable "other"`, the function is tagged EvalOnly, and
   `--strict-bytecode` fails (V1, V2).
2. **Arm-order violation (silent wrong result).** `allConstructorPatterns`
   (match.go:77-88) accepts variable/wildcard arms in **any** position, but the switch path
   demotes them to `Default` — which in switch semantics runs only when **no** case matched.
   A variable arm that is not the final arm loses its first-match-wins position: the VM
   computes a different value than the evaluator, exit 0 (V3).
3. **Guard dropped (silent wrong result).** The default-collection pass ignores
   `arm.Guard` on variable/wildcard arms entirely (match.go:113-125); a guarded catch-all's
   guard is never emitted, so the arm fires unconditionally — again a silent wrong value
   on the VM (V4).
4. **Sibling gaps found in the same audit.** The if-chain path has no
   `ConstructorPattern` case in `lowerPatternBindings` (match.go:473-559; V6), the golang v1
   **value-switch** default emits an unbound variable (`return other`, V9), the golang v1
   **ADT switch** places `default:` at the arm's source position so Go switch semantics
   reproduce bug 2 in generated code (V7), and emitgo (v2 pipeline) has neither default
   binding (V10) nor unused-variable suppression (V11).

All four consumers share one semantic source of truth — the evaluator — and today disagree
with it in different subsets of the same construct class.

**Verified symptom table** (binary v0.51.0 `b99dd25c`-dirty, all commands run in this
session; programs in Verification Log V1-V10):

| # | Shape | `ailang run` (evaluator) | `run --strict-bytecode` | `compile --emit-go` (v1) | `compile --emit-go-v2` |
|---|---|---|---|---|---|
| 1 | ctor match, **final unguarded** var default whose body references it (the reported repro) | correct | **error**: evaluator-only, `unbound variable "other"` (V1) | correct — binds `other := _adt` (V8) | **broken Go**: `default: return other`, unbound (V10) |
| 2 | var arm **before** a constructor arm | `c` | **`idle`** — silent wrong (V3) | same wrong answer as VM: `default:` first + `case JKindPlanned` (V7) | same as VM (V10 family) |
| 3 | **guarded** var arm | `a` | **`c`** — silent wrong (V4) | correct (guards route to if-else chain, V24) | same as VM |
| 4 | value (string) match, var default | correct | correct (if-chain path binds, V5) | **broken Go**: `default: return other`, unbound (V9) | correct |
| 5 | if-chain match with a **constructor arm** (`(Some(n), _) => n`, …) | `7` | **error**: `unbound variable "n"` (V6) | correct | broken (shared lowering) |
| 6 | final **wildcard** default (body does not reference a binding) | correct | correct (V22) | correct | correct |

**Impact:**

- Every AILANG program using the natural catch-all idiom in a constructor match
  (`Planned(_) => …, other => …`) is evaluator-only: `--strict-bytecode` refuses it
  (reported from an active mission sprint, which had to rewrite source to work around it).
- Two shapes produce confidently wrong values on the strict VM and in generated Go with
  exit 0 — the worst class for a determinism-guaranteed language, and a direct
  NO-SILENT-FALLBACKS + axiom A1 violation.
- `ailang compile` emits Go that does not build in two of these shapes, caught only when
  the `go build` verification gate is enabled.

## Goals

**Primary Goal:** A variable pattern in a match arm binds the scrutinee, honors its guard,
and keeps its position on the strict VM, in emitgo-v2 output, and in golang-v1 output —
byte-for-byte the same result as the evaluator.

**Success Metrics:**
1. The reported repro (varpat.ail) returns `committed` under `--strict-bytecode` (today: error).
2. The misplaced-arm and guarded-arm shapes (V3, V4) return the evaluator's value under `--strict-bytecode` (today: a different value, exit 0).
3. `ailang compile --emit-go` and `--emit-go-v2` on all repro shapes produce Go that builds (verification gate green), with the same result as `ailang run`.
4. Zero new EvalOnly functions across `std/` and `examples/` in a strict-VM sweep (the corpus contains no misplaced/guarded var arm today — V23).
5. All existing tests pass unmodified, including every fixture in the "Programs that MUST still work" list.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix at the **lowering** layer (`internal/gen/lower/match.go`), not in `SwitchStmt` IR or the bytecode compiler | The binding/order/guard information is lost at `SwitchStmt` (V16); the lower layer is the one point shared by VM + emitgo-v2 | agent | design | med |
| Gate the switch fast-path: a variable/wildcard arm is eligible as switch `Default` only if it is **final and unguarded**; otherwise route the whole match to the (already correct) if-chain | Restores first-match-wins and guard semantics without touching switch codegen; changes lowered IR shape for previously-silent-wrong programs only | agent | design | med |
| Bind the default arm by prepending `lowerPatternBindings` output as statements inside `Default` (no `SwitchStmt` struct change) | Reuses the landed, tested M-LOWER-FIX mechanism (V14, V17); avoids IR-format ripple to every emitter | agent | design | low |
| Add a **depth-1** `ConstructorPattern` case to `lowerPatternBindings` (if-chain path); Lit/nested args there fail **loudly** (panic → EvalOnly stub) instead of silently mis-binding | Needed so the gate can route misplaced/guarded var arms in constructor matches without regressing their arg bindings (V6); keeps the nested-pattern sibling doc's recursive rewrite as the general solution | agent | design | low |
| golang v1: bind var defaults in the value-switch path and route non-final var/wildcard arms to the if-else chain | v1 codegen is correct for the reported shape (V8) but wrong for V7/V9; mirrors the lower-layer gate | agent | design | low |
| emitgo: emit a `_ = <name>` unused-suppression line after every `VarDecl` | Bound-but-unused default vars would otherwise break `go build` of v2 output (V11); mirrors golang v1's `writeSuppressUnused` | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Fix site: lowering layer (decided above; alternatives considered and rejected in Solution Design)
- [x] Loud-error vs silent-wrong for the interim Lit/nested-arg shapes (loud, via the existing panic→EvalOnly channel — V21)
- [ ] Sprint-planner may re-target the version folder (v0.51.2 assumed; sibling docs target their landing patch)

## Solution Design

### Overview

One semantic rule, enforced at one place: **a variable/wildcard pattern arm is an ordinary
ordered arm that happens to be irrefutable** — it binds its variable, carries its guard, and
keeps its position. The switch fast-path is only allowed to represent it as `Default` when
that representation is provably equivalent: the arm is **final** and **unguarded** (and the
only var/wildcard arm). Every other shape routes to the if-chain, which already implements
the rule correctly (M-LOWER-FIX). The golang v1 generator gets the same two corrections in
its own idiom (it works on Core, not the lowered IR); emitgo needs no lowering change at
all — it inherits the fix — only an unused-variable suppression.

### Architecture

**Root cause** (single sentence): `lowerConstructorMatch` treats a `VarPattern`/
`WildcardPattern` arm as an unordered, unguarded, unbound `Default`, while the evaluator
treats it as an ordered arm that binds the scrutinee; the correct binding code exists in
the same file (`armBody`/`lowerPatternBindings`, match.go:303-312, 473) but is only wired into
the if-chain path.

**Components:**

1. **`internal/gen/lower/match.go` — the whole fix** (~70 LOC):
   - **Eligibility gate.** Replace `allConstructorPatterns` with a predicate requiring:
     every arm is a `ConstructorPattern`, plus **at most one** trailing
     `VarPattern`/`WildcardPattern` arm with **no guard**. Any other mix (var arm
     non-final, guarded, more than one, or non-constructor patterns) routes to
     `lowerIfChainMatch`.
   - **Default binding.** In `lowerConstructorMatch`'s default-collection pass, build the
     default body exactly like the if-chain's `armBody`: bindings from
     `lowerPatternBindings(scrutinee, arm.Pattern, cti)` (VarPattern →
     `VarDecl(name, scrutinee)`; Wildcard → nil) followed by the flattened body. Because
     Bug A.2's guard machinery duplicates `Default` into guard-failing `Else` branches, the
     prepended `VarDecl` rides along into each copy — a case body that falls through a
     failed literal guard still sees the binding in its duplicated default.
   - **If-chain constructor arms.** Add a `*core.ConstructorPattern` case to
     `lowerPatternBindings`: for arg index `j`, a `VarPattern` (name ≠ `_`) binds via
     `FieldAccess{scrutinee, "_j"}` (the exact mechanism the tuple case and the switch
     bindings already use — V17); `WildcardPattern` args skip; **LitPattern/nested args
     panic with a precise message** (e.g. `lower: constructor arm with literal/nested
     sub-patterns in an if-chain match is not yet lowerable; evaluator fallback`), which
     the existing recovery converts into an EvalOnly stub with that reason (V21). The
     cond side needs no change: `lowerPatternCond`'s `ConstructorPattern` case already
     emits the tag check (match.go:395-402).
   - Note the boundary: constructor-only matches with Lit/nested args **keep the switch
     path** and the landed Bug A.2 machinery — untouched. The new if-chain case is
     reachable only for matches the gate routed away (misplaced/guarded var arms) and for
     existing mixed matches (V6), both of which are broken today.

2. **`internal/gen/golang/codegen_match.go` — v1 codegen parity** (~15 LOC):
   - Value-switch default: for a `VarPattern` (name ≠ `_`) emit
     `<goVar> := _scrutinee` + `writeSuppressUnused` before `return <body>` — the exact
     shape `generateMatchArmADT` (codegen_match_arms.go:167-181) and
     `generateMatchIfElse` (codegen_match_ifelsechain.go:34-38) already use (V8, V19).
   - Routing: extend `patternsNeedIfElse` (codegen_match_patterns.go:275-312) to return
     true when a var/wildcard arm is **not the final arm** (guards already route). The
     if-else chain is order-correct and binds var arms, so V7's `default:`-first emission
     becomes unreachable for misplaced arms.

3. **`internal/gen/emitgo/funcs.go` — unused suppression** (~3 LOC):
   - `emitVarDecl` appends `_ = <name> // suppress unused` (legal in Go even when the name
     is used), mirroring golang v1's `writeSuppressUnused`. Without it, every
     bound-but-unused default var (e.g. `x => "constant"`) would fail `go build`
     verification of v2 output (V11 shows the same failure already exists on the if-chain
     path for `zz => "B"` — fixed by the same line).

4. **`internal/bytecode/` — no changes.** `compileSwitch` already compiles `VarDecl`
   inside the Default scope (switch.go:151-158 via stmt.go:28-29) and `FieldAccess "_j"`
   compiles to positional `OpGetField` (collections.go:221-244) — the if-chain proof (V5)
   exercises exactly this machinery on the strict VM today.

### Implementation Plan

**Phase 1: Lower layer + unit tests** (~0.5 day)
- [ ] Gate: eligibility predicate + routing; keep `TestLowerMatchStmt_ConstructorWithWildcard` green (wildcard default stays on the switch path, still no VarDecl).
- [ ] Default binding via prepended `lowerPatternBindings` output.
- [ ] `ConstructorPattern` case in `lowerPatternBindings` (Var/Wildcard args bind; others panic loudly).
- [ ] Unit tests in `internal/gen/lower/lower_match_test.go`: (a) final var default emits leading `VarDecl` in `Default` and in the Bug A.2 duplicated `Else`; (b) misplaced var arm lowers to an ordered `IfStmt` chain (not a `SwitchStmt`); (c) guarded var arm ANDs the guard; (d) constructor arm in an if-chain binds args via `FieldAccess "_j"`; (e) wildcard default emits no `VarDecl`.

**Phase 2: golang v1 + emitgo** (~0.25 day)
- [ ] Value-switch default binding + `patternsNeedIfElse` routing extension; codegen golden for both.
- [ ] emitgo `emitVarDecl` suppression; v2 goldens for default binding (`v6`/`v8` shapes from this doc).

**Phase 3: Strict-VM goldens, sweep, docs** (~0.5 day)
- [ ] `tests/golden/bytecode/` fixtures (alongside `pattern_arity.ail`): the reported repro and the V3/V4/V6 shapes, asserting VM output == evaluator output; plus V5 (string match var default) and one final-wildcard default as regression guards.
- [ ] Strict-VM sweep of `std/` + `examples/` (V23 baseline: zero misplaced/guarded var arms; expect zero behavioral change).
- [ ] Run `make test`, `make fmt`, `make lint`, `make check-boundaries`.
- [ ] Docs: remove nothing from `docs/LIMITATIONS.md` (this gap is not listed there today — V25); add a reference-page note that variable-pattern arms bind on all execution routes; add a `changelogs/` entry.

### Files to Modify/Create

**New files:**
- `tests/golden/bytecode/var_default.ail` (+ golden spec rows in `golden_test.go`) — repro + V3/V4/V6 shapes, ~40 LOC
- `tests/golden/codegen/var_default.go.golden` (v1) and `--emit-go-v2` golden — codegen regression pins

**Modified files:**
- `internal/gen/lower/match.go` — gate, default binding, if-chain constructor case (~70 LOC)
- `internal/gen/lower/lower_match_test.go` — new unit tests (~150 LOC)
- `internal/gen/golang/codegen_match.go` — default binding + routing (~15 LOC)
- `internal/gen/golang/codegen_match_patterns.go` — `patternsNeedIfElse` extension (~8 LOC)
- `internal/gen/emitgo/funcs.go` — unused suppression (~3 LOC)
- `tests/golden/bytecode/golden_test.go` — spec rows (~20 LOC)
- `docs/docs/reference/` match reference page + `changelogs/` entry (~15 LOC)

## Examples

### Example 1: The reported repro (varpat.ail) — before/after

**Before** (current behavior, verified V1-V2):

```
$ ailang run --quiet --entry main --args-json 1 varpat.ail
committed
$ ailang run --quiet --bytecode --strict-bytecode --entry main --args-json 1 varpat.ail
Error: vm: CALL: varpat.cancel is evaluator-only (compiler: unbound variable "other")
       but no interop bridge is wired
```

**After** (all three execution routes agree):

```
$ ailang run --quiet --bytecode --strict-bytecode --entry main --args-json 1 varpat.ail
committed
```

### Example 2: The silent-wrong-answer shapes — before/after

```
pure func cancel(j: J) -> J = match j { other => Committed(0), Planned(_) => Idle }   -- V3
pure func f(j: J) -> string = match j { Planned(_) => "p", other if isC(other) => "a", Committed(n) => "c" }  -- V4
```

**Before:** evaluator `c` / `a`, strict VM `idle` / `c` — exit 0, no warning. Generated v1
Go repeats the V3 wrong answer (V7).
**After:** `c` / `a` on every route; the misplaced and guarded arms lower to an ordered
if-chain with the guard ANDed in.

### Example 3: If-chain constructor arm (V6) — before/after

```
pure func f(p: (Opt, int)) -> int = match p { (Some(n), _) => n, (_, 0) => 0, other => -1 }
```

**Before:** strict VM `compiler: unbound variable "n"` (evaluator-only).
**After:** `7` for `f((Some(7), 7))` — the new depth-1 constructor binding in the if-chain.

## Success Criteria

- [ ] varpat.ail returns `committed` under `--strict-bytecode` (AC1, acceptance: golden row).
- [ ] V3 and V4 shapes return the evaluator's value under `--strict-bytecode` (AC2, acceptance: golden rows asserting exact values).
- [ ] V6 shape returns `7` under `--strict-bytecode` (AC3, acceptance: golden row).
- [ ] `ailang compile --emit-go` and `--emit-go-v2` on all repro shapes produce Go that passes the `go build` verification gate, with evaluator-equal results (AC3, acceptance: codegen goldens).
- [ ] `make test`, `make fmt`, `make lint`, `make check-boundaries` green (AC4/AC5).
- [ ] Strict-VM sweep of `std/` + `examples/`: zero new EvalOnly functions; std/json.ail wildcard defaults unchanged (AC6, acceptance: sweep script output diff).
- [ ] Documentation updated (reference page + changelog).

## Testing Strategy

**Unit tests (`internal/gen/lower/lower_match_test.go`):**
- Lowered-IR assertions per Phase 1 list — every pattern variable in every arm position receives a `VarDecl`; routed shapes assert `IfStmt` (not `SwitchStmt`); guard conds ANDed; Bug A.2 `Else` duplication carries the binding.

**Integration / goldens:**
- `tests/golden/bytecode/` VM rows for the whole symptom table (shapes 1-6), asserting value equality with the evaluator.
- `tests/golden/codegen/` goldens for v1 value-switch default binding, misplaced-arm if-else routing, and emitgo-v2 default binding + suppression.

**Regression-surface tests** (one per "Programs that MUST still work" entry, below):
- std/json.ail `get`/`asString` (final wildcard default on the switch path — must emit no VarDecl and keep SwitchStmt form); examples/match_hof_lambda.ail (value-match var default `n => n + 1` and `_ => "ok"` — if-chain path); examples/pattern_matching_adt.ail (Some/None, no default); the V5 string-match program; std/sem.ail:378 single-arm var match.

**Manual testing:**
- Re-run the full Verification Log command set (V1-V10) and confirm every cell in the symptom table reads "correct".
- Non-strict `--bytecode` continues to return the evaluator's value for every shape (bridge fallback unaffected — the routing only changes which functions compile).

## Conflict Surface

**Syntactic/semantic positions touched:**
- Lowering of `core.Match` arms whose pattern is `VarPattern`/`WildcardPattern` in constructor matches (`internal/gen/lower/match.go` — the gate at :77, default collection at :110-125).
- `lowerPatternBindings` (match.go:473) gains a `ConstructorPattern` case — the function that already serves the if-chain's Var/Tuple/List/Record cases (M-LOWER-FIX territory).
- golang v1: `codegen_match.go` value-switch default (~:144-166) and `patternsNeedIfElse` (codegen_match_patterns.go:275-312).
- emitgo: `emitVarDecl` output only (funcs.go:102).

No parser, lexer, AST, type, or effect changes — the surface syntax and the elaborated Core are untouched; the evaluator's semantics is the fixed point everything else converges to.

**What else lives in these positions (and why it is safe):**

| Position | Existing valid form | Interaction with this change |
|---|---|---|
| Var/wildcard arm, **final, unguarded**, in a ctor match | std/json.ail `JObject(kvs) => …, _ => None` (V22) | Stays on the switch path; wildcard binds nothing, so no `VarDecl` is prepended and lowered IR is byte-identical (guarded by regression test d) |
| Var default whose body does **not** reference the binding | `x => "constant"` | New `VarDecl` is emitted but unused — harmless on the VM; emitgo suppression line added so v2 `go build` stays green (V11) |
| Constructor arm with **literal** sub-patterns (`Num(0) => …`) | Bug A.2 landed machinery (`_lit_<i>` temp + guard, default duplication) — extractBindingsAndGuards match.go:228-293, guard wrap :142-190 | Constructor-only matches keep the switch path and this machinery untouched; the prepended default `VarDecl` rides into the duplicated `Else` (unit test a) |
| Constructor arm with **nested** sub-patterns (`Some(Some(x))`) | `_pat_<i>` temps (broken today — the nested-pattern sibling doc's scope) | Unchanged on the switch path; on the if-chain path (routed shapes) they panic loudly → EvalOnly instead of silently mis-binding (see "What deliberately changes") |
| Guards on **constructor** arms | Combined into the case guard `if` (match.go:142-177) | Unchanged; the default body (now with binding) is duplicated into the failing-guard `Else` as before |
| Guarded **var** arm | Today: guard silently dropped (V4) | Routes to if-chain; guard ANDed (unit test c) — this is the fix, not a conflict |
| If-chain matches (Tuple/List/Record/Lit arms + Var defaults) | M-LOWER-FIX semantics (V5 correct today) | The shared `lowerPatternBindings` gains a case; existing cases untouched; V5 becomes a regression guard |
| golang v1 guarded matches | Any guard → if-else chain (V24) | Unchanged; the routing extension only adds the non-final-var-arm trigger |

**Disambiguation:** purely structural, on the elaborated Core arm list (pattern kind, position, guard presence). No token-level ambiguity is introduced anywhere.

**Programs that MUST still work (regression fixtures):**
1. `std/json.ail` — `get`, `asString` (final wildcard defaults; also exercised by `tests/json_accessors_test.ail`).
2. `examples/match_hof_lambda.ail` — `n => n + 1` and `_ => "ok"` value-match defaults (if-chain).
3. `examples/pattern_matching_adt.ail` — Some/None constructor matches, no defaults.
4. The V5 string-match program (`match s { "a" => "A", other => other }`) — strict VM correct today, must remain correct.
5. `std/sem.ail:378` — single-arm var match `json_str => decode_frame(json_str)` (if-chain single-arm path).

**What deliberately changes (intentional incompatibilities):**
- Matches with a **misplaced or guarded** var/wildcard arm lower to if-chains instead of switches: VM and emitgo-v2 output previously *wrong* for these (V3/V4) now computes the evaluator's answer; any consumer diffing lowered IR or disassembly for these shapes will see a different (correct) form.
- Matches with a **guarded/misplaced var arm AND constructor arms with Lit/nested sub-patterns** become loud EvalOnly (strict errors; non-strict still runs correctly via the bridge) where the VM previously produced a silently wrong value. This narrows as the nested-pattern sibling doc lands its recursive if-chain conds; if that doc lands first, this branch never exists.
- emitgo v2 output gains `_ = <name> // suppress unused` lines after `VarDecl`s (cosmetic diff to generated Go).

## Verification Log

All claims verified first-party in this session. Binary: `ailang v0.51.0` (`b99dd25c22dc7bf1bd5c3ace8dfab14a46aa3f7d-dirty`, `/usr/local/bin/ailang`); repo HEAD `35ae3261eb51f1bd4e517a8a96068266527e2965` (relevant files read at HEAD and consistent with observed behavior; this sandbox has no Go toolchain — all Go claims below are source reads, all behavior claims are binary runs). Repro files live under `/tmp/varpat/`.

| # | Claim | Instrument | Verdict |
|---|-------|-----------|---------|
| V1 | Reported repro: final unguarded var default references its binding → evaluator `committed`, strict VM error `evaluator-only (compiler: unbound variable "other")` | `ailang run --quiet --entry main --args-json 1 varpat.ail` → `committed`; `--bytecode --strict-bytecode` → `Error: … CALL: … cancel is evaluator-only (compiler: unbound variable "other") but no interop bridge is wired` | Confirmed |
| V2 | Non-strict `--bytecode` silently returns the evaluator's answer for V1 | `run --quiet --bytecode …` → `committed` (bridge fallback, `internal/runner/vm.go:204-207`) | Confirmed |
| V3 | Var arm **before** a constructor arm: evaluator `c`, strict VM `idle` (silent wrong, exit 0) | `/tmp/varpat/v1.ail`; eval run and strict run, outputs `c` vs `idle` | Confirmed |
| V4 | Guarded var arm: guard dropped; evaluator `a`, strict VM `c` (silent wrong) | `/tmp/varpat/v3.ail` (`other if isC(other) => "a", Committed(n) => "c"`); eval `a` vs strict `c` | Confirmed |
| V5 | Value (string) match with var default is correct on strict VM (if-chain binds) | `/tmp/varpat/v4.ail`; eval `b` = strict `b` | Confirmed |
| V6 | If-chain match with constructor arm: `unbound variable "n"` → evaluator-only | `/tmp/varpat/v5.ail` (`(Some(n), _) => n`); strict error | Confirmed |
| V7 | golang v1 emits `default:` at the var arm's source position → Go switch reproduces the V3 wrong answer | `ailang compile --emit-go v7.ail`; generated `switch _adt.Kind { default: … return NewJCommitted(int64(0)) case JKindPlanned: … }` | Confirmed |
| V8 | golang v1 ADT path binds var defaults correctly (`other := _adt`) — the parity reference for this fix | `ailang compile --emit-go v6.ail` → `default:\n other := _adt\n _ = other // suppress unused\n return other` | Confirmed |
| V9 | golang v1 **value-switch** default does NOT bind → unbound `other` in generated Go | `ailang compile --emit-go v4.ail` → `default:\n return other` (grep `/tmp/varpat/gen/v4/v4.go:11-13`; no declaration of `other` in output) | Confirmed |
| V10 | emitgo v2 (shared lowering) emits unbound `return other` for V1 | `ailang compile --emit-go-v2 --no-verify-go v6.ail` → `/tmp/varpat/gen6v2/v6/v6.go:9-10: default: return other` | Confirmed |
| V11 | Bound-but-unused vars: emitgo emits no unused-suppression (v2 `go build` would fail; pre-existing on if-chain path for `zz => "B"`) | `ailang compile --emit-go-v2 v8.ail` → `zz := s` unused; `internal/gen/emitgo/funcs.go:102-113` (no suppress); verify gate runs `go build ./...` (`cmd/ailang/compile.go:533-545`) — skipped in this sandbox only because `go` is not in PATH (`compile.go:506-509`) | Confirmed |
| V12 | Elaborator produces `core.VarPattern` for lowercase identifier patterns | read `internal/elaborate/patterns.go:125` | Confirmed |
| V13 | Evaluator binds VarPattern to the whole scrutinee and always matches | read `internal/eval/eval_patterns.go:109-112` | Confirmed |
| V14 | `lowerConstructorMatch`'s default pass builds the body WITHOUT bindings (and without the var arm's guard); the if-chain's `armBody` DOES prepend bindings (the landed M-LOWER-FIX pattern) | read `internal/gen/lower/match.go:113-125` vs `:303-312` | Confirmed |
| V15 | `allConstructorPatterns` accepts var/wildcard arms in any position; the default pass drops both the arm's position and its guard | read `internal/gen/lower/match.go:77-88` and `:113-125` | Confirmed |
| V16 | NEGATIVE: `SwitchStmt` retains no pattern/binding info for `Default` (binding must be injected as statements) | read `internal/gen/stmt/stmt.go:127-141` (`Default []Stmt`; `SwitchCase.Bindings` only, per case) | Confirmed |
| V17 | The binding mechanism (VarDecl in Default scope + `FieldAccess "_j"` → positional `OpGetField`) already exists and works on the strict VM | read `internal/bytecode/compiler/switch.go:151-158`, `stmt.go:28-29`, `collections.go:221-244`; V5 exercises it end-to-end (if-chain) | Confirmed |
| V18 | NEGATIVE: `lowerPatternBindings` has no `ConstructorPattern` case (default returns nil) | read `internal/gen/lower/match.go:473-559` (cases at :475 Var, :483 Tuple, :495 List, :538 Record); V6 empirically confirms | Confirmed |
| V19 | golang v1 if-else chain binds var defaults, order-correct (model for V9 fix) | read `internal/gen/golang/codegen_match_ifelsechain.go:27-50, 105-130` | Confirmed |
| V20 | Guards route v1 golang to the if-else chain (v1 is guard-correct; the lower path is not — V4) | read `codegen_match_patterns.go:275-288` (any guard → if-else); contrast match.go:113-125 | Confirmed |
| V21 | Loud-error channel exists: lower panic → EvalOnly stub with `LowerError`; compiler tags proto; strict runner errors, non-strict falls back | read `internal/gen/lower/program.go:146-215`, `internal/bytecode/compiler/compiler.go:105-118`, `internal/runner/vm.go:204-219`; also the live V1/V6 error format | Confirmed |
| V22 | Corpus: final wildcard defaults in constructor matches exist and work on the VM | read `std/json.ail:65-71, 98-101, 245-252` (JObject/JString/JNumber + `_ => …`); doc example `std/env.ail:41` | Confirmed |
| V23 | Corpus: no misplaced/guarded var arm and no named var default in a constructor match exists in `std/` + `examples/` (only `std/sem.ail:378`, a single-arm if-chain var match, which is correct; one-line multi-arm matches swept separately — all final wildcards: `std/json.ail:245,250`, doc comment `std/env.ail:41`) | two sweeps of `std/*.ail examples/*.ail`: awk arm-shape scan over multi-line matches + one-line `match … { … => …, … => … }` grep; positive control: both scans DO find the wildcard defaults of V22 | Confirmed |
| V24 | v1 codegen is correct for the reported shape (so only V7/V9-class shapes change there) | V8 generated output | Confirmed |
| V25 | NEGATIVE: this gap is not listed in `docs/LIMITATIONS.md` today (docs task adds a reference note, not a limitations row, unless an interim loud shape remains after the sibling doc lands) | read `docs/LIMITATIONS.md` (pattern entries at :23-45; none cover var arms) | Confirmed |
| V26 | Distinctness from the nested-pattern sibling doc: that doc's case 7 **preserves** today's switch-path Var/Wildcard/Lit binding behavior (i.e. leaves this bug in place); its case 6 covers the if-chain constructor case my Component 1 third bullet adds at depth 1 | read `design_docs/planned/v0_49_1/m-bytecode-nested-pattern-lowering.md:186-207, 226-227, 255-256` | Confirmed |
| V27 | No new error codes are introduced (panic strings flow into free-text `EvalReason`; no `MOD/TC/E` code allocation) | design uses only existing channels (V21); `grep` of proposed changes introduces none | Confirmed |

**Note on the frequency claim:** the construct is absent from the in-repo corpus (V23) but
was hit immediately by an external consumer (stapledons-godot `sim/core.ail`, sprint
R1-M2-JOURNEY M2.3a, per the task report) whose maintainer had to replace the natural
`other => other` with `_ => j` — exactly the workaround suppression V23 suggests. No eval
pass-rate claim is made; this doc claims only the first-party table above.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Regression in the landed Bug A.2 literal-sub-pattern guards or M-LOWER-FIX if-chain bindings (this code has been patched incrementally before) | High | Every existing test in `lower_match_test.go` is a fixture that must pass unmodified; new tests assert binding completeness for every arm position — the A-family regressions were invisible precisely because no test asserted bindings |
| Routed shapes change lowered IR / disassembly (goldens, eval-only sweep diffs) | Med | Only shapes that are silent-wrong or unbound today change; golden sweep before/after in Phase 3; corpus is clean (V23) |
| Lit/nested-arg constructor arms in routed matches become loud EvalOnly (previously silent wrong) | Med | Intentional (no-silent-fallbacks), documented in Conflict Surface; disappears when the nested-pattern sibling doc lands; non-strict mode still runs correctly via the bridge |
| Merge collision with the nested-pattern sibling doc (both extend `lowerPatternBindings`/`lowerConstructorMatch`) | Med | This doc's additions are the depth-1 subset of that doc's case 6/7 recursive semantics (same `_j` field names, same cond shape); whichever lands second generalizes rather than rewrites — coordination noted in Related Documents |
| emitgo suppression line noise in all v2 output | Low | Cosmetic, legal Go, mirrors v1's existing `writeSuppressUnused` output |

## Related Documents

**Implemented (may inform design):**
- [m-lower-fix.md](../implemented/v0_11_0/m-lower-fix.md) — landed the if-chain half of this bug family (the `armBody` binding pattern this fix reuses); see also the "unbound variable t" regression test at `internal/gen/lower/lower_match_test.go:163`.
- [m-bytecode-vm.md](../implemented/v0_11_0/m-bytecode-vm.md) — VM architecture and the EvalOnly/bridge mechanism.
- [m-dx20-wildcard-pattern-inference.md](../implemented/v0_6_1/m-dx20-wildcard-pattern-inference.md) — `_` wildcard vs named binding semantics at the elaborator (the elaboration this fix builds on).

**Planned (check for overlap):**
- [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) — SIBLING, same file, disjoint variant class: nested/literal sub-patterns. Its case 7 explicitly preserves today's switch-path default-arm behavior (V26), so this doc does not duplicate it; its case 6 (if-chain constructor conds/bindings, recursive) generalizes this doc's depth-1 if-chain addition — coordinate sprint order.
- [m-bytecode-pattern-arity-fix.md](../v1_0_0/m-bytecode-pattern-arity-fix.md) — SIBLING: fixed-length list pattern `len == n` (landed logic present at match.go's list cond); same clause-2 residue family.
- [m-bytecode-vm-parity-bugs.md](../v1_0_0/m-bytecode-vm-parity-bugs.md) — the parent parity-bugs lane this family spun out of.

**Related docs search note:** `ailang docs search` (neural, fallback-simhash mode) surfaced no doc on this topic above 0.45 relevance on the natural-language query; the siblings above were found by targeted `grep -rln "evaluator-only\|strict-bytecode" design_docs/` — the coverage-gate reading was done on the full text, not on search snippets.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- Evaluator pattern semantics: `internal/eval/eval_patterns.go` (`matchPattern`)
- Lowering: `internal/gen/lower/match.go`; consumers: `internal/bytecode/compiler/switch.go`, `internal/gen/emitgo/funcs.go`, `internal/gen/golang/codegen_match*.go`
- Task report: sprint R1-M2-JOURNEY M2.3a (stapledons-godot `sim/core.ail` workaround `_ => j`)

## Future Work

- The nested-pattern sibling doc's recursive `patternCond`/`patternBindings` rewrite subsumes this doc's depth-1 if-chain constructor case; after it lands, the loud EvalOnly boundary introduced here shrinks to nothing.
- Runtime tag identity for cross-ADT switches (separate, documented limitation) would remove the `inferADTFromCases` ambiguity that switch-lowered matches still inherit.

## Non-Goals

**Not attempted in this feature:**
- Nested/literal sub-pattern conds and bindings in if-chains — [m-bytecode-nested-pattern-lowering.md](../v0_49_1/m-bytecode-nested-pattern-lowering.md) scope.
- Any parser/elaborator/typechecker change — the surface syntax and Core are already correct; this is a parity fix in lowering + codegen.
- Match-arm exhaustiveness semantics — a var arm already makes a match exhaustive at elaborate time; unchanged.
- Performance work on the switch fast-path (the gate may route some matches to if-chains; the corpus shows this is rare and correctness dominates).

## Deferred Decisions

The following are intentionally left open for the implementer:
- Exact panic message wording for the interim loud EvalOnly shapes — agent may choose (must name the arm shape and point at this doc).
- Whether the `ConstructorPattern` case in `lowerPatternBindings` shares a helper with `extractBindingsAndGuards` or stays self-contained — agent may choose (keep the M-LOWER-FIX test assertions passing either way).
- Whether golang v1 routing lands as an extension of `patternsNeedIfElse` or a separate `patternsSafeForSwitch` predicate — agent may choose; behavior must be identical.

## Timeline

**Week 1** (~12 hours):
- Phase 1: lower gate + default binding + if-chain constructor case + unit tests (6h)
- Phase 2: golang v1 + emitgo + codegen goldens (3h)
- Phase 3: strict-VM goldens, sweep, docs, full `make test` (3h)

**Total: ~1.5 days across 1 week** (2× the raw estimate, per skill guidance)

---

**Document created**: 2026-10-01
**Last updated**: 2026-10-01
