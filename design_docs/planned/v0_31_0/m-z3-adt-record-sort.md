# M-Z3-ADT-RECORD-SORT: Declare Reachable ADTs Through Records and Make `ai-check` Honest

**Status**: Planned — **DECIDED by Mark 2026-07-28 (attended): OPTION (B)** — sprint `#510` separately and FIRST (small, self-contained, closes the standing-rule violation affecting every `ai-check` caller incl. Ailang World); then THIS doc routes to sprint-planner unchanged with A5 honestly compliant. [Was: PARKED `needs-human-review` after quorum round 2 (see "Quorum round 2 — PARKED" below); the design direction was never contested.]
**Target**: v0.31.0
**Priority**: P0
**Estimated**: 3–4 days
**Created**: 2026-07-28
**Source**: ailang#477; Ailang World `w-m1-ailang-hardening`
**Dependencies**: Existing SMT record discovery and mutual-datatype support

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | +1 | Declaration closure and ordering are computed from a finite type graph with stable ordering. |
| A2: Replayability | 0 | No trace behavior changes. |
| A3: Effect Legibility | 0 | No effect semantics change. |
| A4: Explicit Authority | 0 | No authority change. |
| A5: Bounded Verification | +1 | Finite declaration construction is followed by a mandatory Z3 `-T:<seconds>` soft timeout; zero, unset, and sub-second configurations floor to 5 seconds. A 1-second adversarial probe returned structured timeout state in 1.159 s. There is no independent hard wall-clock kill, recorded as a bounded residual risk below. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | `ai-check` no longer reports process success when its JSON reports a verifier error; unsupported shapes retain a structured rejection. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | 0 | No cost model change. |
| A10: Composability | +1 | Records may compose with the already-supported user ADTs and sequences without a hidden declaration boundary. |
| A11: Structured Failure | +1 | An unresolved sort becomes `UNENCODABLE_TYPE`/`skipped`, never malformed SMT handed to Z3. |
| A12: System Boundary | 0 | Changes stay within the SMT core, its CLI consumers, and the eval harness test surface. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: Declaration order and SCC membership must be deterministic.
- [x] A3: No hidden effects are added.
- [x] A4: No ambient authority is added.
- [x] A7: JSON status and process status are made consistent for verifier errors.

---

## Problem

The SMT encoder accepts a contracted function whose parameter is a named record containing a
user ADT, but omits the record's datatype declaration. It then emits a constant of the omitted
sort and sends malformed SMT-LIB to Z3.

This checked AILANG program is the minimal downstream shape:

```ailang
module adtsort
export type Evidence = CompilerOutput(string) | TestReport(string, bool)
export type Proposal = { name: string, evidence: list[Evidence] }
export func hasName(p: Proposal, n: string) -> bool ! {}
ensures { result == (p.name == n) }
{ p.name == n }
```

`./bin/ailang check /tmp/iter115repro/adtsort.ail` succeeds. At commit `f495885b1`,
`./bin/ailang ai-check /tmp/iter115repro/adtsort.ail` reports:

```text
verify.verified=0, verify.skipped=0, verify.errors=1
Z3 ... Invalid constant declaration: unknown sort 'Proposal'
process exit=0
```

The root cause is not declaration order. The emitted SMT-LIB contains no declaration for
`Proposal` at all:

```smt2
(declare-const $p_p Proposal)
(declare-const $p_n String)
```

`EncodeFunction` pre-registers record aliases in a fixpoint
([`internal/smt/codegen.go`](../../../internal/smt/codegen.go), Step 0). A field sort is admitted
only when primitive or already in `ctx.DeclaredTypes`; `(Seq Evidence)` therefore defers
`Proposal`. The loop reaches `!progress` and silently abandons the remaining alias. Later record
discovery cannot recover it because the function parameter is represented by the named sort,
not an inline `TRecord`. Parameter constants are emitted regardless.

This is a direct violation of CLAUDE.md principle 2, **No Silent Fallbacks**: the encoder drops
required work without either rejecting the function or returning an error.

### Impact

The impact has two bounded parts:

1. **Coverage:** a record field containing a user ADT—directly, under `list`, or through another
   record—can prevent the enclosing sort from being declared. Ailang World's
   `w-m1-ailang-hardening` sprint loses three of seven transition predicates because they take
   `Proposal` with `list[Evidence]`.
2. **Control signal:** `ai-check` writes JSON containing `verify.errors=1` and then exits zero.
   Its primary audience is an AI agent convergence loop, which is explicitly told that
   `ailang ai-check FILE` is the green signal.

The v1.0 headline KPI is **not** inflated by this exit-code bug. The eval harness parses the JSON;
`VerifyOk` and `isVerifiedSuccess` independently require zero verifier errors. Encoder errors
exclude a run rather than counting it as proved. The exposure is shell/agent exit-code consumers
and silently reduced contract coverage.

---

## Evidence / Verification Log

“Inherited” rows were controller-verified first-party at commit `f495885b1` with the freshly
built binary, Z3 4.16.0, and macOS. “Author” rows were rechecked in this worktree at the same
commit on 2026-07-28. Counts are stated only where the controller enumerated them or the command
below did.

| ID | Provenance | Claim | Method | Result |
|---|---|---|---|---|
| V1 | Inherited A1; author recheck | Minimal record→`list[ADT]` program type-checks, then `ai-check` reports one Z3 error and exits zero | `./bin/ailang check /tmp/iter115repro/adtsort.ail`; `./bin/ailang ai-check ...; echo $?` | **Confirmed:** check succeeds; JSON has `errors: 1`, `skipped: 0`; exit 0. |
| V2 | Inherited A2 | Seven-shape bisection: record→ADT (direct/list/single ctor/nested) fails; record→`list[string]` and direct ADT parameters (unused and pattern-matched) verify | Controller's same-binary matrix in iteration brief | Relied on as controller-verified; not re-counted by author. |
| V3 | Inherited A3 | Failing script omits `Proposal`; direct ADT and record→`list[string]` scripts contain their datatype declarations | Controller ran `ailang verify -verbose` on all three and supplied verbatim dumps | Relied on; author `ai-check` output independently confirms the omitted-sort consequence. |
| V4 | Inherited A4; author code read | First alias fixpoint silently abandons unresolved records | Read `internal/smt/codegen.go:213-260`, especially `allFieldsPrimitiveOrDeclared` and `if !progress { break }` | **Confirmed.** Required aliases remain in `remaining` with no error/result. |
| V5 | **Author contradiction to A4** | The second fixpoint is also a silent give-up | Read `internal/smt/codegen.go:267-363`; `git log -- internal/smt/codegen.go` | **CONTRADICTED at `f495885b1`:** unresolved pending ADT/inline-record declarations go through `findSCCs` and `DeclareDatatypesMutual`; they are not simply dropped. The first fixpoint is the live silent-drop site. |
| V6 | Author recheck of B1 | A real current `.ail` corpus instance of `ParsedDocument → Block → Record_blocks_kind → Block` exists | `rg -n "ParsedDocument|Record_blocks_kind|type Block|blocks: list\\[Block\\]" --glob '*.ail' --glob '*.go' --glob '*.md'` | **Not found in the `.ail` corpus.** The canonical shape exists in comments and unit fixtures. The comment's “current corpus” implication is unverified and must not justify exclusion. |
| V7 | Author recheck of B1 | Z3 4.16.0 accepts the alleged record/ADT cycle as one plural datatype group | Pipe a `Block ↔ Record_blocks_kind` `(declare-datatypes ...)` script to `z3 -in`; `z3 --version` | **Confirmed:** Z3 4.16.0 returns `sat`. |
| V8 | Author recheck of B1/B2 | Mutual-recursion graph/group machinery already exists and is tested | Read `internal/smt/codegen_mutual.go`; `go test ./internal/smt -run 'Test(DeclareDatatypesMutual\\|FindSCCs\\|SplitDeclareDatatype)' -count=1` | **Confirmed:** Tarjan SCC + plural emitter exist; focused tests pass. This extends A3 and makes reuse feasible. |
| V9 | Author inventory of B3, corrected in revision 1 | Which record field sorts enter the first silent-drop path | Re-read `MapType`/`MapRecordFields` in `internal/smt/types.go:36-90` and the Step-0 predicate | **Confirmed, with prior wording corrected:** any successfully mapped nonprimitive sort not already declared defers the alias: a direct bare ADT/other `TCon` name, `(Seq ADT)`, or an unresolved named record. “Arbitrary `TCon`” meant an arbitrary **bare constructor name**, not an arbitrary parameterized SMT sort. Type variables/functions/unit fail mapping earlier and are deleted, also silently. Non-list `TApp` is separately flattened to its bare constructor name. |
| V10 | Author inventory of B3 | Top-level `Seq<ADT>` uses the same record-alias drop | Read collection path and Step 0 | **No:** the defect is specifically the record-alias dependency path. A direct ADT parameter is controller-verified; a direct list parameter does not require a record alias. `Seq<ADT>` still needs regression coverage, but is not evidence of this drop. |
| V11 | Inherited A5; author code read | Structured unencodable rejection exists | Read `internal/smt/encodable.go:13-112`, `cmd/ailang/ai_check.go:349-363`, `cmd/ailang/verify.go:403-423` | **Confirmed:** `RejectUnencodable = "UNENCODABLE_TYPE"`; `ErrUnresolvableTypes` maps to a skipped result carrying that code. |
| V12 | Author extension | Existing declaration validation catches the dangling parameter sort | Read `validateDeclarations`, `internal/smt/codegen.go:668-728` | **No:** it checks `define-fun` signatures and singular datatype declarations, but explicitly treats other declarations—including `declare-const`—as fine. This is the final leak. |
| V13 | Inherited A6; author code read | `ai-check` ignores verifier errors in its exit condition | Read `cmd/ailang/ai_check.go:158-165` | **Confirmed:** only failed checking or `Counterexample > 0` exits 1; JSON is written first. |
| V14 | Inherited A7; author repo sweep | In-repo `ai-check` consumers | `rg -n "ai-check" --glob '*.go' --glob '*.sh' --glob '*.yml' --glob '*.yaml' --glob '*.py' --glob 'Makefile*'` plus code reads | **Confirmed:** no in-repo Makefile/CI/shell gate; programmatic consumer is `RunAICheck`; multiple agent prompts treat it as convergence signal. |
| V15 | Inherited A7; author code read; **CONTROLLER EMPIRICAL PROBE (quorum round 2)** | `RunAICheck` can parse JSON after a nonzero subprocess exit | Read `internal/eval_harness/verify.go:47-92`; then MEASURED — a fake `ailang` shell script printing a valid `ai-check` JSON body (`verify.errors: 1`) to stdout and then `exit 1`, driven through the real `RunAICheck` in a throwaway Go test (removed after the run; worktree left clean) | **CONFIRMED EMPIRICALLY — no longer an assertion:** `err=<nil>`, 152 bytes of stdout captured, parsed result read back `verify.errors=1 available=true verified=0`. The non-zero child exit did **not** short-circuit parsing, because `waitErr` is consulted only when stdout is empty. This closes `gemini-3-1-pro`'s round-2 objection **at design time**; M3 AC4 keeps the test as a permanent regression guard rather than as the first proof. |
| V16 | Inherited A7/A9; author code read | KPI protection and stale comment | Read `internal/eval_harness/verify.go:80,140-154` and `agent_verify_test.go:154-176` | **Confirmed:** comment at line 80 is inverted on both counterexamples and errors; result gating requires zero errors. |
| V17 | Inherited A8; author recheck | `verify` has a strict flag; `ai-check` does not | `rg -n -- "strict" cmd/ailang/{verify.go,main.go,help.go}`; read `ai_check.go` flags | **Confirmed:** `verify -strict` makes skipped or errored functions nonzero; `ai-check` has no strict equivalent. |
| V18 | Author duplicate/precedent search | Existing design already covers this exact change | `./bin/ailang docs search ...`; `rg` over planned/implemented docs; read `m-smt-callee-sort-gate.md` | **No duplicate found.** The nearest relevant implemented doc adds undeclared-sort defenses for callees/`define-fun`; this doc extends that invariant to record closure and parameter constants. |
| V19 | Author code read and empirical probe (revision 1) | Solver invocation is bounded by a mandatory configured timeout | Read `internal/smt/solver.go:58,64,136-175`; ran `Solve` on an unsatisfiable 100-pigeon/99-hole Boolean fixture with `SolverConfig{Timeout: 1*time.Second}` under Z3 4.16.0 | **Confirmed:** production always passes `-T:<integer seconds>`; values below 1 second floor to 5 seconds, never unbounded. The adversarial query returned `StatusUnknown`, raw `timeout`, and structured error `"solver timeout"` after **1,158,965,750 ns (1.159 s)**; end-to-end probe elapsed was about **1.36 s**, not a hang. No `context`, `CommandContext`, timer, or explicit `Process.Kill` supplies a hard wall-clock backstop if Z3 ignores `-T:`. |
| V20 | Author exhaustive encoder-sort enumeration (revision 1) | `(Seq X)` is the only parenthesized/parameterized SMT sort shape `MapType` can emit today | Re-derived every return branch in `internal/smt/types.go:36-90`: `TCon` through `mapTCon`, `TList`, `TRecord`, `TApp`, and error branches | **Confirmed:** successful results are primitives (`Int`, `Real`, `Bool`, `String`), `(Seq X)` for `TList` or one-argument `list` `TApp`, a bare name for other `TCon`/non-list `TApp`, or a record sort name. TVar, function, unit, non-TCon-constructor `TApp`, nil, and unsupported types error. There is no map, set, or other generic SMT sort grammar at this commit; widening traversal to hypothetical sorts is unjustified. |

### Revision 1 quorum disposition

- **Objection 1 accepted as a documentation/evidence gap.** V19 now establishes the mandatory
  soft timeout statically and empirically; M1 AC7 adds the requested recursive-datatype timeout
  regression and named production red mutations. The Risks section explicitly preserves the
  unresolved absence of an independent hard wall-clock kill.
- **Objection 2 rejected as stated, with its underlying fail-open concern accepted.** V20 proves
  that map/set/other parameterized SMT sorts do not exist in the current encoder grammar, so this
  design does not invent traversal for them. D3 and M1 AC6 instead make the grammar handling total:
  unknown parenthesized forms fail closed, and any future parameterized form must extend traversal.
  V9 is corrected to remove the misleading implication that “arbitrary `TCon`” meant arbitrary
  parameterized SMT syntax.

### Quorum round 2 — PARKED `needs-human-review`

Round 2 returned **blocked** again, on two NEW objections. Neither disputes the design
**direction**; both were reproduced first-party by the controller before being acted on, and they
did **not** land in the same place.

- **`gemini-3-1-pro` — CLOSED AT DESIGN TIME.** It objected that V15's cross-boundary claim was
  self-admittedly "not yet empirical" and must not be deferred to implementation. It was right to
  refuse it, and the controller simply **ran the probe it asked for**: see V15 above —
  `err=<nil>`, `verify.errors=1` parsed correctly from a child that exited 1. The objection is
  satisfied by measurement, not by argument.

- **`gpt5-6-sol` — CONFIRMED, AND IT IS THE PARK.** It objected that the design scores axiom A5
  compliant while knowingly leaving **production** solver execution unbounded. Verified
  first-party at `internal/smt/solver.go:147-148`:

  ```go
  cmd := exec.Command(z3Path, args...)
  output, err := cmd.CombinedOutput()
  ```

  No `CommandContext`, no deadline, no `Process.Kill`, no process group. The only bound is Z3's
  **cooperative** `-T:` flag, so a wedged or non-conforming solver hangs `verify`/`ai-check`
  indefinitely. The reviewer is correct that a cooperative flag plus a test-only watchdog does not
  discharge a bounded-waits axiom. **Filed independently as `ailang#510`** — it is a real defect
  in its own right, not merely a documentation gap, and it exists at HEAD regardless of what this
  doc does.

**Why this parks rather than taking the narrow-refinement carve-out.** The carve-out permits the
controller to apply reviewers' verbatim fixes only when every remaining objection is free of
design-direction disputes *and* of controller judgment. `gpt5-6-sol`'s fix is a **scope
expansion** — adding process-lifecycle management (context deadline, process-group kill, reap,
fake-solver tests) to a sprint already estimated at 3–4 days. Deciding whether that belongs here,
or ships as its own item, is precisely the sizing judgment the carve-out excludes. Same reasoning
as iteration 114.

**Recommended to Mark — option (B):**

- **(A)** Absorb `#510` into this sprint as a fourth milestone. Honest, but pushes the estimate
  past the sprint-sized bar and couples an encoder fix to a process-lifecycle fix.
- **(B) — recommended.** Ship this doc as-is, with **A5 scored NON-compliant** and pointing at
  `#510`, and sprint `#510` **separately and first** (it is small, self-contained, and closes a
  standing-rule violation that affects every `ai-check` caller — including the sibling Ailang
  World mission, which gates on `ai-check`). Then this doc's A5 becomes honestly compliant and it
  routes to sprint-planner unchanged.
- **(C)** Ship this doc as-is and accept the unbounded wait as a documented residual risk. Cheapest,
  but it leaves a standing-rule violation open and would be scoring an axiom compliant that
  demonstrably is not.

---

## Goals

1. Verify finite user ADTs reachable through named record fields, including `list[ADT]` and
   genuine record↔ADT cycles supported by Z3.
2. Ensure no parameter/result constant with an unresolved nonprimitive sort reaches Z3.
3. Make `ai-check` exit nonzero whenever its emitted JSON reports a verifier **error**, while a
   structured **skip** remains a normal zero-exit fragment-boundary result.
4. Preserve the eval harness's ability to parse complete JSON from nonzero `ai-check` exits.

---

## Design Decisions

### D1 — `ai-check` error exit policy

| Option | Semantics | Assessment |
|---|---|---|
| A. Status quo | Exit 1 for check failure/counterexample; exit 0 for verifier error | Reject: contradicts JSON and the agent convergence contract. |
| B. Add opt-in `-strict` | Default remains dishonest; strict mode exits for errors and possibly skips | Reject: agents use the default, and `ai-check` already exits 1 for semantic failure. |
| **C. Unconditional nonzero for `verify.errors > 0`** | Errors are process failure; counterexamples remain failure; skips remain zero | **Recommend.** Consistent, small, and preserves “unsupported but honestly skipped” as distinct from encoder failure. |
| D. Nonzero for errors **and skips** | Treat inability to prove as command failure | Reject for this sprint: changes the established fragment-boundary contract and duplicates `verify -strict` semantics without demonstrated demand. |

**Decision:** C. The condition becomes:

```go
if !checkSection.Passed || verifySection.Counterexample > 0 || verifySection.Errors > 0 {
    os.Exit(1)
}
```

The JSON remains complete because `outputAICheck(output)` stays before the exit. No new flag is
added. A skipped function exits zero unless another function errors/counterexamples or checking
fails.

### D2 — What to do when declaration closure cannot be formed

| Option | Semantics | Assessment |
|---|---|---|
| A. Continue and let Z3 diagnose | Current behavior for this leak | Reject: raw solver error and no-silent-fallback violation. |
| B. Convert every unresolved graph to `verify.errors` | Loud but reports an unsupported fragment as an encoder defect | Reject when the graph is genuinely unsupported. |
| **C. Return `ErrUnresolvableTypes` with sort/path context** | Existing drivers report `skipped` + `UNENCODABLE_TYPE` | **Recommend.** Reuses the established structured boundary. |

**Decision:** C. The error must name the undeclared sort and its consumer, for example
`parameter "p" uses undeclared sort "Proposal"`; the driver hint must no longer claim only
“cross-module” or “recursive ADT” causes.

### D3 — Encoder capability architecture

| Option | Semantics | Assessment |
|---|---|---|
| A. Predeclare every ADT before every record | Fixes acyclic cases but fails record↔ADT cycles and over-declares unused types. | Reject. |
| B. Special-case record fields whose sort matches `adtTypes` | Fixes the reporter shape but fragments dependency handling; nested/list/cyclic cases recur. | Reject. |
| **C. One needed-sort declaration graph for record aliases, inline records, and ADTs** | Emit acyclic SCCs in dependency order and cyclic SCCs with existing plural form. | **Recommend.** Systemic and reuses current machinery. |

**Decision:** C. Keep demand-driven filtering in the drivers. Inside `EncodeFunction`, convert
needed record aliases to pending datatype declarations and combine them with
`ExtraDeclarations` and `adtTypes` before dependency resolution. The graph must see sort
references recursively under `(Seq ...)`. Acyclic dependencies emit dependency-first;
multi-node SCCs emit one deterministic `declare-datatypes`; self-recursive singletons may retain
the singular form accepted by Z3.

The sort-reference collector is total over the encoder's current output grammar: primitives and
bare names are leaves; `(Seq X)` recursively visits `X`; any other parenthesized form returns
wrapped `ErrUnresolvableTypes`. A future `MapType` addition that emits another parameterized form
must extend this collector in the same change. It must never silently classify an unknown form as
resolved.

`activeRecordTypes` and `activeFieldSetToSort` must be registered when a record alias enters the
pending graph, not only after emission, so record access/construction encoding can resolve the
named sort.

### D4 — Defense-in-depth invariant

| Option | Semantics | Assessment |
|---|---|---|
| A. Check only the original `Proposal` parameter | Minimal patch | Reject: another declaration consumer can leak later. |
| **B. Validate every emitted declaration's sort uses before solver invocation** | `declare-const`, `define-const` result sort, `define-fun`, singular/plural datatypes | **Recommend**, with parser scope bounded to declaration headers. |
| C. Parse all generated SMT-LIB into a new full AST | Strongest, but not sprint-sized. | Defer. |

**Decision:** B. Extend the existing validation pass rather than add a second guard. It must
recognize plural datatype groups as declaring all member sorts atomically and must reject any
nonprimitive sort used by `declare-const`/typed `define-const`/`define-fun` unless declared
earlier or in the same mutual group. This is the enforceable form of “never emit a constant whose
sort is unresolved”; direct primitive and `(Seq primitive)` constants remain valid.

### Design Freeze

- [x] `ai-check` errors are unconditionally nonzero; skips remain zero.
- [x] Unsupported declaration graphs return `ErrUnresolvableTypes` and become
  `UNENCODABLE_TYPE` skips.
- [x] Record aliases join the existing dependency graph/SCC mechanism; no ADT-name special case.
- [x] The declaration validator covers constant sort consumers and plural datatype groups.

---

## Solution Design

### Declaration closure

For each function, the existing driver computes a demand-driven needed-sort set. The encoder then
builds a declaration graph:

```text
needed record aliases ─┐
inline record decls ───┼─> declaration nodes ─> sort-reference graph ─> SCCs ─> SMT-LIB
needed user ADTs ──────┘
```

Each node contains sort name, declaration body, kind, and its referenced nonprimitive sorts.
Dependencies nested under `Seq` count normally (`Proposal → Evidence`). Existing deterministic
field ordering is retained; node/SCC/member output must be sorted to avoid Go-map iteration
changing SMT-LIB.

For the repro, the graph is acyclic:

```text
Proposal ──field evidence: Seq Evidence──> Evidence
```

so output declares `Evidence`, then `Proposal`, then `$p_p`.

For the canonical fixture:

```text
Block ──> Record_blocks_kind ──Seq──> Block
```

both sorts form one SCC and are emitted through `DeclareDatatypesMutual`.

### Fail-loud boundary

After declarations and parameter/result constants are assembled—but before expressions reach
Z3—`validateDeclarations` walks declaration headers in order. It maintains a primitive/declaration
environment, with plural datatype member sorts installed atomically. An unresolved consumer
returns wrapped `ErrUnresolvableTypes` with the declaration kind, consumer name, and sort.

Both `verify` and `ai-check` already translate that sentinel into a skipped result. Their current
reason/hint says “cross-module types” even when the defect is local; this sprint replaces it with
neutral wording such as “SMT declaration closure could not resolve required sort” and includes the
wrapped detail.

### Exit-code interaction

These lanes are deliberately orthogonal:

| JSON outcome | Exit after this sprint | Meaning |
|---|---:|---|
| check failed | 1 | Source is invalid |
| counterexample > 0 | 1 | Contract is false |
| errors > 0 | 1 | Verifier/solver failed |
| skipped > 0, with no above condition | 0 | Source is valid but obligation is outside the supported fragment |
| verified > 0 only | 0 | Obligation proved |

Once M1 lands, the reporter repro should be **verified**, not skipped. M2 is the backstop: if a
future or unsupported shape cannot form declaration closure, it becomes a structured skip and
therefore does not exercise the new error exit. To test M3's error exit independently, its CLI
test must inject or use a genuine `verify.errors > 0` result rather than relying on the repaired
record shape.

---

## Milestones

All three milestones are **in scope** for this 3–4 day sprint.

### M1 — Unified record/ADT declaration closure (1.5–2 days, in scope)

- Fold needed named record aliases into the existing pending datatype dependency graph.
- Reuse and harden SCC/plural-datatype emission; sort all graph inputs and outputs.
- Register record metadata before expression encoding.
- Cover direct ADT, `Seq<ADT>`, nested record→record→ADT, acyclic record→ADT→record, and genuine
  cyclic record↔ADT graphs.

#### Acceptance Criteria

1. `adtsort.ail` reports `verified=1`, `errors=0`, `skipped=0`; verbose SMT declares `Evidence`
   before `Proposal`.
   **Red mutation:** in production `codegen.go`, remove record aliases from the pending graph (or
   restore the ADT-excluding Step-0 predicate). The integration assertion must fail with an
   omitted `Proposal`/non-verified result.
2. A record with a direct single-constructor ADT field and a record with `list[ADT]` both verify.
   **Red mutation:** in the production sort-reference collector, stop descending through
   `(Seq ...)`. The list case must fail while the direct case localizes the regression.
3. Record→nested-record→ADT verifies with every datatype emitted dependency-first.
   **Red mutation:** in production graph construction, omit edges from record fields whose sort
   is another record alias. The nested integration test must fail.
4. The `Block ↔ Record_blocks_kind` fixture emits one plural `declare-datatypes` group accepted
   by Z3.
   **Red mutation:** in production SCC dispatch, emit each member as a separate singular
   declaration. The test must either fail declaration-shape assertions or Z3 acceptance.
5. SMT output for a fixed mixed graph is byte-identical across repeated encodes.
   **Red mutation:** in production emission, iterate the declaration-node map directly instead
   of sorting nodes/SCC members. Run enough repeated encodes to make the test fail reliably, or
   expose/test the deterministic ordering function directly.
6. Sort-reference extraction accepts every shape in V20, recursively extracts `(Seq X)`, and
   returns `ErrUnresolvableTypes` for an unrecognized parenthesized form such as `(Future X)`;
   adding any future parameterized output to `MapType` requires a matching collector case.
   **Red mutation:** in the production collector's default/else branch, treat an unrecognized
   parenthesized form as a leaf/resolved sort. The `(Future X)` negative test must fail.
7. A mutual/recursive datatype verification query that is deliberately expensive is bounded by
   the configured solver timeout and returns structured `unknown`/`"solver timeout"` state rather
   than hanging. The test uses a watchdog only to detect a production regression, with headroom
   above Z3's soft timeout.
   **Red mutation:** in production `solver.go`, drop the `-T:` argument (or remove the `< 1` floor
   so a zero configuration becomes unbounded). The adversarial test must exceed its watchdog and
   fail; it must clean up the child in test teardown.

### M2 — Declaration leak gate and structured skip (0.5–1 day, in scope)

- Extend `validateDeclarations` to `declare-const`, typed `define-const`, `define-fun`, singular
  datatypes, and plural groups.
- Return contextual `ErrUnresolvableTypes`.
- Make `verify` and `ai-check` skip reasons type-neutral and actionable.

#### Acceptance Criteria

1. A hand-built declaration list containing `(declare-const $p_p Missing)` returns
   `ErrUnresolvableTypes` naming `$p_p` and `Missing`; the solver is not invoked.
   **Red mutation:** in production `validateDeclarations`, restore the early-continue for
   `declare-const`. The unit test must fail.
2. A valid `(declare-const xs (Seq Evidence))` passes after `Evidence` is declared, while the
   same constant before declaration fails.
   **Red mutation:** in the production recursive sort parser, treat `Seq` as sufficient without
   validating its element sort. The negative ordering case must fail to be rejected.
3. A plural group may reference its own member sorts atomically, but a member reference outside
   the group fails.
   **Red mutation:** in production plural-group handling, register only the first member. The
   mutual fixture must fail.
4. An intentionally unresolved parameter-sort integration fixture reports
   `status:"skipped"` with rejection code `UNENCODABLE_TYPE`, and contains no Z3 error text.
   **Red mutation:** in the production `EncodeFunction` error return, suppress
   `ErrUnresolvableTypes` and continue assembling SMT. The integration test must observe
   `status:"error"`/solver text and fail.
5. Both CLI drivers use neutral closure wording rather than the current false
   cross-module-only hint.
   **Red mutation:** in either production driver, restore the literal
   `Uses cross-module types not yet supported...`; a table-driven test over both drivers must
   fail that case.

### M3 — Honest `ai-check` exit and harness compatibility (0.5–1 day, in scope)

- Add `verifySection.Errors > 0` to the unconditional exit-1 condition.
- Keep JSON emission before exit; do not make skips nonzero.
- Fix the inverted comment in `internal/eval_harness/verify.go`.
- Add an empirical subprocess test of `RunAICheck` parsing nonempty JSON from a nonzero child.

#### Acceptance Criteria

1. `ai-check` exits 1 after emitting valid, complete JSON when `verify.errors > 0`.
   **Red mutation:** remove `verifySection.Errors > 0` from production `ai_check.go`; the CLI
   subprocess test must observe exit 0 and fail.
2. A counterexample still exits 1; a pure structured skip with no errors exits 0; a verified
   result exits 0.
   **Red mutation:** in production exit logic, replace `Errors > 0` with `Skipped > 0`. The skip
   case must turn red; removing `Counterexample > 0` must turn the counterexample case red.
3. JSON is parseable and contains its result on every exit-1 path.
   **Red mutation:** move production `outputAICheck(output)` below the `os.Exit(1)` branch. The
   subprocess test must receive empty/non-JSON stdout and fail.
4. `internal/eval_harness.RunAICheck` returns a parsed result with `Verify.Errors > 0` and no
   wrapper error when the real child exits 1 after writing JSON.
   **Red mutation:** in production `RunAICheck`, change the guard from
   `rawOutput == "" && g.waitErr != nil` to `g.waitErr != nil`. The empirical harness test must
   fail. This is the required re-verification of V15.
5. The stale comment accurately states that counterexamples and verifier errors are nonzero,
   while nonempty JSON is still parsed.
   **Red mutation:** restore the production comment's old claim and enforce it with the
   repository's comment/behavior assertion adjacent to the subprocess test (the behavioral ACs,
   not comment text alone, remain the release gate).

---

## Files Expected to Change

| File | Purpose | Estimate |
|---|---|---:|
| `internal/smt/codegen.go` | unified pending declaration graph; fail-loud validation | +100/−60 |
| `internal/smt/codegen_mutual.go` | deterministic SCC/member ordering or structured declaration helpers | +30/−10 |
| `internal/smt/codegen_*_test.go` | dependency, cycle, determinism, leak-gate unit tests | +250 |
| `cmd/ailang/ai_check.go` | error exit condition; neutral skip reason | +15/−10 |
| `cmd/ailang/verify.go` | neutral skip reason | +5/−5 |
| `cmd/ailang/*_test.go` | end-to-end JSON/status/exit tests | +150 |
| `internal/eval_harness/verify.go` | correct stale comment only; behavior should remain | +2/−2 |
| `internal/eval_harness/*_test.go` | empirical nonzero-child JSON parsing test | +60 |
| `CHANGELOG.md` | user-visible verifier coverage and exit behavior | +10 |

Exact test-file placement is implementer discretion; production ownership and behavioral
boundaries above are fixed.

---

## Conflict Surface

### Semantic positions affected

This changes SMT sort declaration and CLI status semantics, not AILANG parsing or typechecking.
The affected type positions are:

- named record aliases needed by a contracted function's parameters, result, body, or contracts;
- record fields containing primitives, user ADTs, sequences, or other records;
- ADT constructor fields containing records/sequences/other ADTs;
- parameter/result/callee declarations consuming those sorts;
- `ai-check` process status after its JSON result is complete.

### Existing constructs sharing those positions

- Primitive record fields and `list[string]` already verify.
- Direct user ADT parameters, including pattern-matched parameters, already verify.
- Nested records and inline records already use dependency-first declaration.
- Cross-module record aliases and user ADTs are demand-filtered by
  `buildNeededSortSet`/`filterExtraDeclarationsForFunction`.
- Recursive/mutually recursive ADT graphs already use singular/plural datatype support.
- Parametric non-list ADTs flatten type arguments in `MapType`; they are **not** silently
  promoted to supported by this design.
- Cross-function `define-fun` signature leaks already use `ErrUnresolvableTypes` defenses from
  M-SMT-CALLEE-SORT-GATE.

### Programs and tests that must remain valid

The controller-verified controls and the two existing test surfaces below are mandatory
regressions because they isolate the changed sort positions:

1. `/tmp/iter115repro/v3_list_string.ail` — record with `list[string]`.
2. `/tmp/iter115repro/v4_adt_param.ail` — direct ADT parameter.
3. `/tmp/iter115repro/v4b_adt_param_used.ail` — direct, pattern-matched ADT parameter.
4. Existing `internal/smt/codegen_nested_records_test.go` nested-record fixtures.
5. Existing `internal/smt/codegen_mutual_test.go` mutual/SCC fixtures.

The first three are controller fixtures outside the repository and must be copied into stable Go
integration fixtures during implementation rather than referenced by `/tmp` in CI.

### Intentional behavior changes

- The failing `Proposal` shape changes from Z3 `error` to `verified`.
- Any future dangling nonprimitive constant sort changes from Z3 `error` to structured
  `skipped/UNENCODABLE_TYPE`.
- An actual `ai-check` verifier error changes process exit from 0 to 1 after identical JSON is
  emitted.
- Skips remain exit 0; KPI calculation remains JSON-based and unchanged.

### What else could break, and why

- **Declaration ordering:** combining formerly separate passes could forward-reference an
  acyclic sort. Dependency-order and byte-determinism tests guard it.
- **Accessor/constructor lookup:** delaying emission without early metadata registration could
  make record expressions unencodable. Record construction/access tests must accompany graph
  tests.
- **Demand filtering:** accidentally adding all module aliases could reintroduce cross-module
  cascade failures described in `verify.go`. The graph input remains the driver's filtered maps.
- **Mutual declaration validation:** a validator unaware of atomic plural groups could reject
  valid cycles. M2 AC3 guards this exact conflict.
- **Harness subprocess handling:** the child now exits 1 for a JSON-level error. V15 shows the
  intended code path, but M3 requires an empirical test before release.
- **External shell consumers:** none exist in-repo by V14; out-of-repo callers that treated an
  encoder error as success will now fail, which is intentional and must be called out in the
  changelog.

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| One unified declaration graph is larger than expected | Medium | Medium | Reuse existing declaration strings, Tarjan SCC, and plural emitter; do not introduce a full SMT AST. |
| Regex-based sort extraction misclassifies constructor/accessor names | Medium | High | Represent dependencies from mapped field sorts where available; keep raw declaration parsing behind focused tests and fail closed. |
| Determinism regresses through map/SCC iteration | Medium | Medium | Sort nodes, edges, SCCs, and group members; byte-identical repeated-encode AC. |
| Capability accidentally admits parametric ADTs without monomorphization | Medium | High | Scope graph nodes to concrete existing declarations; unresolved constructor sorts fail through M2. |
| Fail-loud gate converts a supported case to skip | Low | Medium | Controls for primitive, sequence, direct ADT, nested record, and mutual groups. |
| Exit-code change surprises an external caller | Medium | Low | JSON schema/output order unchanged; changelog explicitly documents the correction. |
| Full test suite binds loopback under sandbox | Medium | Low | Label such failures **UNINFORMATIVE UNDER SANDBOX**; rely on focused non-network suites plus unsandboxed CI for socket tests. |
| Z3 ignores or fails to honor its soft/per-query `-T:` timeout | Low | High | **Residual risk:** production uses `exec.Command`/`CombinedOutput` with no context deadline, timer, or hard wall-clock process kill. The mandatory `-T:` and timeout regression AC bound normal supported Z3 behavior, but a wedged/nonconforming child can still hang. Queue a hard-kill backstop separately rather than claiming one exists. |

---

## Out of Scope

- General parametric-ADT monomorphization (`Option[T]`, `Result[E,T]`, or arbitrary non-list
  `TApp`). In `internal/smt/types.go:56-67`, the non-list `TApp` branch returns `con.Name` and
  silently discards `ty.Args`, so `Box[int]` and `Box[string]` collapse to the same SMT sort
  `Box`. This distinct correctness issue requires its own backlog item and is not fixed here.
- Recursive contract reasoning or bounded recursion changes.
- Changes to Z3 version, solver logic, hard-kill policy, or model extraction; this sprint only
  regression-tests the existing mandatory soft timeout.
- Making `ai-check` skips nonzero or adding an `ai-check -strict` flag.
- A full SMT-LIB parser/typed intermediate representation.
- Changing eval KPI definitions, benchmark scores, or historical results.
- Re-arguing demand: ailang#477 and the Ailang World 7→4 predicate reduction establish it.

---

## Deferred Decisions

- Internal declaration-node struct and helper names — implementer may choose.
- Whether deterministic SCC ordering lives in `codegen.go` or `codegen_mutual.go` — implementer
  may choose, provided M1 AC5 holds.
- Stable integration-fixture location — implementer may choose under existing SMT/CLI test
  conventions.
- Exact neutral skip prose — implementer may choose, but it must name the unresolved
  declaration/sort and must not assert a cross-module-only cause.
- Parametric non-list ADT encoding/monomorphization for
  `internal/smt/types.go:56-67` — queue separately because type arguments currently collapse.
- Whether to add an OS-level hard wall-clock kill around Z3 — queue as a separate bounded-
  verification hardening item; current production has only Z3's mandatory soft `-T:` timeout.

---

## Related Documents

- [M-SMT-CALLEE-SORT-GATE](../../implemented/v0_30_0/m-smt-callee-sort-gate.md) — established
  `ErrUnresolvableTypes` as a defense against undeclared callee sorts. This design extends the
  same invariant to record declaration closure and constant declarations; it is not duplicate
  work.
- [M-SMT-CROSS-MODULE-TYPES](../../implemented/v0_14_3/m-smt-cross-module-types.md) — introduced
  the demand-driven alias/ADT and mutual-declaration context this design must preserve.

---

## Sprint Exit

The sprint is complete only when M1, M2, and M3 acceptance criteria pass, focused
`go test ./internal/smt ./cmd/ailang ./internal/eval_harness` is green, `go build ./...` is green,
and the changelog documents both the expanded record/ADT verification fragment and
`ai-check`'s corrected error exit. Any loopback bind failure under this sandbox is recorded as
**UNINFORMATIVE UNDER SANDBOX**, not as a pass or regression.
