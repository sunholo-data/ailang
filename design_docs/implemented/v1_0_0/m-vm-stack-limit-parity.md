# M-VM-STACK-LIMIT-PARITY — the bytecode VM's 1000-frame cap diverges from the evaluator's 10,000 recursion limit: evaluator-legal deep recursion silently truncates service handlers under `--bytecode`

Refs [#1576](https://github.com/sunholo-data/ailang/issues/1576).

**Status**: Implemented (PR #1729); independent evaluation passed 96/100 (round 2, 2026-10-09). See [implementation verification](m-vm-stack-limit-parity-verification.md).
**Target**: v1.0.0 (clause-2 soundness residue; sibling of [m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) Lane B, same carve-out as [m-bytecode-pattern-arity-fix.md](../../planned/v1_0_0/m-bytecode-pattern-arity-fix.md))
**Priority**: **P1** (per the issue's `priority:P1` label) — silent wrong result: truncated stdout, exit 0, stderr silent under default flags.
**Estimated**: ~0.5 day (root cause settled below; the production fix is two lines plus wiring plus regression tests — not investigation).
**Dependencies**: none. Explicitly does NOT depend on the parent doc's parked A2 second design round — that question is about the *parity harness's* classification scheme, not program correctness (the same carve-out that unblocked m-bytecode-pattern-arity-fix.md). The **residual** defect this issue exposes — the VM→evaluator fallback silently re-running a program after committed IO — is the parent doc's **B4** and stays parked there; this doc fixes the *limit divergence* that makes evaluator-legal programs fall into it.
**Filed bug**: [#1576](https://github.com/sunholo-data/ailang/issues/1576) (P1, `area:bytecode`, `from:stapledons_godot`), reported against v0.52.0, triage-verified live on origin/dev `658ff76a3` 2026-10-08.

## Problem Statement

Under `ailang run --bytecode`, a service handler that builds a list of ~1170 elements by **non-tail recursion** prints its pre-recursion line and then **silently drops the handler's result and the rest of the block**: the service loop continues reading, exits 0, and nothing appears on stderr. Without `--bytecode` the interpreter answers correctly. `n=984` works; `n ≥ ~1100` fails (the reporter's measured threshold; the code-level boundary measured first-party below is **~997**, i.e. MaxStack 1000 minus the handful of live frames).

The reporter's instrumentation (stubLoop's Ask arm): `println ASK; let s = handle(c,l,r); println HANDLED` prints `ASK`, never `HANDLED`; the loop then continues reading and exits 0. A standalone module with `rep(5000)` in an effectful recursive loop **does not reproduce the silent drop** — that shape shows a different (milder) symptom, measured below.

This is a **silent wrong result** — a direct NO-SILENT-FALLBACKS violation and an A1 (determinism) violation: the two backends disagree on evaluator-legal input, and the wrong one wins silently with exit 0.

## Root cause (verified live in this worktree, 2026-10-08)

Two compounding defects. **Neither is a frame/stack-unwinding bug** — the leading hypothesis in the tasking is refuted below with evidence.

### Defect α — the VM's frame-depth cap diverges from the evaluator's recursion limit

- VM: `DefaultMaxStack = 1000` (`internal/vm/vm.go:14`). The constant's own comment claims it "Matches the evaluator's recursion limit so divergence behavior is consistent (§3.6)" — **it never did**: the evaluator's default is **10,000** (`internal/eval/eval_evaluator.go:167`, and the `--max-recursion-depth` flag default is 10000, `cmd/ailang/main_run.go`). The comment is stale-or-aspirational, and nothing ever enforced it.
- Every non-tail `OpCall` pushes a frame; at `len(vm.Stack) >= vm.MaxStack` the VM returns `ErrStackOverflow` (`internal/vm/vm.go:303`, and `:125` for `CallClosure`). A handler at non-tail depth ~1170 therefore aborts the **VM run** — at a depth where the **interpreter completes** (10,000 limit).

### Defect β — the VM→evaluator fallback re-runs the program from the beginning after committed IO, silently

- On the VM error, non-strict `--bytecode` falls through to the evaluator (`internal/runner/entrypoint.go:150-156`, `tryRunEntryViaVM` returned `ranOnVM=false`): the evaluator **re-executes the entry from the start**. This is the parent doc's **B4 unsafe effect replay**, P0, parked pending the A2 design round — #1576 is it in the wild, in a worse shape.
- The re-run cannot reproduce the first run's work: the VM run had already **printed `ASK`** and **consumed both stdin lines** (`readLine` is an EvalOnly builtin bridged to the evaluator's real stdin reader, `internal/runner/vm.go` bridge). The evaluator's re-run reads EOF immediately, its loop returns unit, and the process exits **0**.
- **Nothing on stderr**: the fallback warning (`bytecode path unavailable (...); falling back to evaluator`) is guarded by `if !params.Quiet` — and **`ailang run` has been quiet by default since v0.49.0** (`changelogs/v0.32-current.md`, "Changed — `ailang run` is quiet by default"; `cmd/ailang/main_run.go:26` `fs.Bool("quiet", true, ...)`). Every default `ailang run --bytecode` invocation silently swallows the fallback warning. (This also corrects a premise in the parent doc's A2 — see the addendum there.)

### The measured mechanism (minimal repro, first-party)

`repro/service.ail` (minimal repro, kept as scratch under `/tmp`; the committed fixture shape is AC1's test file below): a tail-recursive `loop()` over `readLine()` that prints `ASK`, calls `handle(l)` (non-tail `rep(1170, l)` building a list), then prints `HANDLED-AFTER`. stdin = `hello\nvoice\n`.

| run | stdout | rc | stderr (signal lines) |
|---|---|---|---|
| interpreter (`ailang run --caps IO --entry main`) | `HELLO` / `ASK` / `HANDLED len=1170` / `HANDLED-AFTER` | 0 | — |
| VM (`--bytecode`, default flags) | `HELLO` / `ASK` **— truncated** | **0** | **silent** |
| VM (`--bytecode --verbose`) | `HELLO` / `ASK` | 0 | `⚠ bytecode path unavailable (vm: vm: stack overflow); falling back to evaluator` |
| VM (`--bytecode --max-recursion-depth 20000`) | `HELLO` / `ASK` | 0 | silent — **the flag does not reach the VM** |
| standalone `rep(5000)` (no stdin), VM | `START` / **`START`** / `DONE len=5000` | 0 | silent fallback |

Depth boundary sweep (same repro, `rep(n)`): n ≤ 995 completes on the VM; n ≥ 997 truncates. The standalone row is the **unsafe-replay duplication** signature (`START` twice — the committed println re-performed by the fallback re-run), and explains why the reporter's standalone repro "did not reproduce": without a consumed-input dependency the re-run *completes*, so the visible symptom is a duplicated prefix line rather than a silent drop.

### Refuted: frame/stack unwinding after deep recursion

The tasking's hypothesis ("frame or stack unwinding after deep recursion") is **not the mechanism**:
- `--strict-bytecode` on the same repro fails **loudly** with `vm: ... stack overflow` (rc=1) — the frame chain unwinds with the error preserved; nothing is dropped silently inside the VM. (In strict mode the trap surfaces even earlier, on the EvalOnly `readLine` stub, because the bridge is intentionally unwired — the service shape can only run non-strict today.)
- The faulted-callback unwind path was already fixed in v0.52.x: `CallClosure` restores the stack to its pre-call depth on error (`internal/vm/vm.go:135-143`; changelog v0.52, #1501).
- The "rest of block skipped" symptom is **not a lost continuation**: it is the *evaluator re-run* of the whole entry seeing EOF stdin. The VM's own continuation after the aborted call never existed — the run ended.

### Systemic view — the divergence was known and fixed everywhere EXCEPT the production path

The same limit-divergence was already measured and worked around twice, without ever fixing the production default:

| surface | VM frame cap | `--max-recursion-depth` wired to the VM? | evidence |
|---|---|---|---|
| `ailang test` named-test harness (`internal/testing/bytecode_engine.go`) | **10000** (`defaultVMMaxStack`, :44) | **yes** (:161-163, `machine.MaxStack = e.maxRecursionDepth`) | `TestEngineParity_RecursionLimitReachesVM` (`internal/testing/engine_parity_test.go:178`) asserts the exact contract this doc proposes — "bounds the VM's frame stack as it bounds the evaluator" |
| **`ailang run` production path (`internal/runner/vm.go:202`)** | **1000** (`vm.NewVM` default) | **no** (`MaxRecursionDepth` reaches only the evaluator: `entrypoint.go:43-44`, `run.go:183-184`, `batch.go:109`) | this issue |
| evaluator (both paths) | n/a — 10000 default | n/a | `eval_evaluator.go:167` |

And at v0.52.0 the stdlib itself was rewritten **iteratively** because of this same VM cap (M-ITERATIVE-LIST-REMAINING: `maximumInt`/`foldr`/… "failed `RT_REC_003` on the interpreter and `vm: stack overflow` on `--strict-bytecode` at 50,001 elements (reported by stapledons_godot)" — `changelogs/v0.32-current.md`). That was the per-case workaround (rewrite each stdlib helper); #1576 is the **user-code recurrence** the workaround could not cover. The unified fix is the one the test harness already chose: make the production VM's cap and flag wiring match the evaluator.

## Verification Log (first-party, this worktree, 2026-10-08; prebuilt `ailang` v0.52.5 + code reads)

| Claim | How verified |
|---|---|
| VM truncates the service repro: stdout `HELLO`/`ASK` only, rc 0, **stderr silent under default flags** | direct `ailang run --caps IO --bytecode --entry main` on the minimal repro, stdin `hello\nvoice\n`, stderr captured separately |
| The fallback fires with `vm: stack overflow` as its embedded cause | same run with `--verbose`: stderr `⚠ bytecode path unavailable (vm: vm: stack overflow); falling back to evaluator` |
| Interpreter answers the same repro (full output, rc 0) | direct `ailang run --caps IO --entry main` |
| Depth boundary ≈ 997 = MaxStack 1000 minus live frames | `rep(n)` sweep n ∈ {990…1010, 1170}: n ≤ 995 completes, n ≥ 997 truncates |
| `--max-recursion-depth` does **not** reach the VM leg | `--bytecode --max-recursion-depth 20000` on the repro: identical truncation (the flag raised only the *evaluator's* ceiling) |
| Standalone deep recursion shows **unsafe replay duplication**, not silent drop | standalone `rep(5000)` under `--bytecode`: stdout `START` twice then correct `DONE len=5000`, rc 0 |
| `DefaultMaxStack = 1000`; comment claims evaluator parity | read `internal/vm/vm.go:12-14` |
| Evaluator default limit 10,000; `--max-recursion-depth` default 10000 | read `internal/eval/eval_evaluator.go:167,179`; `cmd/ailang/main_run.go` (`fs.Int("max-recursion-depth", 10000, ...)`) |
| `MaxRecursionDepth` reaches only the evaluator in the production path | grep `internal/runner/`: `entrypoint.go:43-44`, `run.go:183-184`, `batch.go:109` — all `SetMaxRecursionDepth` on an evaluator; no `machine.MaxStack` assignment anywhere in `internal/runner/` |
| Exactly ONE production (`ailang run`) site constructs the VM with the default cap | grep `NewVM(` repo-wide (excluding `_test.go`): `internal/runner/vm.go:202` (default cap) and `internal/testing/bytecode_engine.go:160` (the non-test `ailang test` harness, which overrides `MaxStack` immediately after); `vm.NewVM` sets `MaxStack: DefaultMaxStack` |
| The test harness overrides cap + wires the flag; and already asserts the contract | read `internal/testing/bytecode_engine.go:42-44,158-163`; `internal/testing/engine_parity_test.go:178-…` (`TestEngineParity_RecursionLimitReachesVM`) |
| `--quiet` is the default for `ailang run` (fallback warning suppressed in every default invocation) | read `cmd/ailang/main_run.go:21-29,149-150` (`fs.Bool("quiet", true, …)`; `--verbose` clears it) and `internal/runner/entrypoint.go:152-155` (warning guarded by `!params.Quiet`); changelog v0.49.0 section "Changed — `ailang run` is quiet by default" |
| A stdin-capable CLI test helper already exists (no testutil addition needed) | read `runWithStdin` (`cmd/ailang/tail_call_parity_test.go:17-36`, used by `ctor_alias_parity_test.go`, `div_zero_parity_test.go`, `import_collision_test.go`) |
| No existing test pins `DefaultMaxStack = 1000` as a default (all VM tests set explicit small caps) | grep `MaxStack` in `internal/vm/*_test.go`, `internal/bytecode/compiler/call_test.go`: every hit is an explicit per-test override (5, 3, 8, 50) |
| The VM has no other runtime cap that could diverge similarly | grep `limit|budget|max` in `internal/vm/vm.go`, `frame.go`: MaxStack is the only one |
| The frame pool is lazily capped at MaxStack (no preallocation; 10000 frames is bounded heap) | read `internal/vm/frame.go:76-81` (`if len(vm.framePool) < vm.MaxStack`) |
| Frame-unwinding hypothesis refuted: strict mode fails loudly, `CallClosure` restores depth on error | strict run above; read `internal/vm/vm.go:125-143` + changelog v0.52 (#1501 callback-fault frame restore) |
| `readLine`/`println` are EvalOnly (Phase 2E) — bridged in non-strict, hard error in strict | strict run of the repro errors `readLine is evaluator-only ... no interop bridge`; non-strict runs print via the bridge |
| Prior art: stdlib rewritten iteratively for this same VM cap at v0.52.0 | read `changelogs/v0.32-current.md` M-ITERATIVE-LIST-REMAINING section (incl. the `vm: stack overflow` row reported by stapledons_godot) |
| **No design rationale for the 1000 cap exists in the VM master design doc** (`m-bytecode-vm.md` mentions MaxStack only as the MaxStack=5 TCO acceptance line; the constant was a Phase 2B default whose comment claims a parity that was never enforced) | grep `MaxStack\|stack overflow` over `design_docs/implemented/v0_11_0/m-bytecode-vm.md` — single hit, line 725 |
| **No test depends on the old flag-no-op** (a test asserting `--max-recursion-depth` is ignored by the `--bytecode` leg of `ailang run` would break under F2) | grep `max-recursion-depth` across `cmd/`+`internal/` tests: every hit exercises the evaluator leg (`deep_recursion_test.go`), the already-wired `ailang test` harness (`engine_parity_test.go`, `max_recursion_depth_test.go`), or pkg run-flags — none asserts VM-leg indifference on `ailang run` |

## Solution Design

Three changes, all in the layer the test harness already proved out:

1. **F1 — production cap parity** (`internal/vm/vm.go`): `DefaultMaxStack = 10000`, and fix the stale comment to state the actual contract: *matches the evaluator's default recursion limit (`internal/eval/eval_evaluator.go`, 10,000) so a body that recurses deeply fails — or passes — at the same depth on both engines* (same wording the test harness uses, `internal/testing/bytecode_engine.go:42-44`). The VM's frames are heap objects (`[]*Frame` + pooled register slabs, `frame.go`) and `vm.run` is a dispatch loop, not Go recursion — a 10,000-frame VM run is strictly lighter than the 10,000-deep evaluator run that is already legal.
2. **F2 — wire `--max-recursion-depth` to the VM** (`internal/runner/vm.go`, after `machine := vm.NewVM(img)`): `if params.MaxRecursionDepth > 0 { machine.MaxStack = params.MaxRecursionDepth }` — mirroring `internal/testing/bytecode_engine.go:161-163` one-for-one. `tryRunEntryViaVM` already receives `params` (`ModuleExecParams.MaxRecursionDepth`, `entrypoint.go:26`); the evaluator leg is wired in the same layer (`entrypoint.go:43-44`). This makes RT_REC_003's own user-facing advice ("raise the ceiling with `--max-recursion-depth`") true under `--bytecode`; today it is **false advertising** for the VM leg.
3. **F3 — document the VM depth contract** (`docs/docs/reference/limitations.md`): the depth section currently documents evaluator semantics only (`--max-recursion-depth N`, #1317 raised ceilings). Add that the same flag and default bound the VM's frame stack under `--bytecode`.

**Scope boundary (deliberate):** at depth *above* the (now shared) limit, a service handler **still** silently truncates — VM abort → fallback re-run → consumed stdin → exit 0 — because the fallback mechanism itself is unsound. That is the parent doc's **B4** policy decision (abort-loudly vs pre-effect-only fallback), parked pending the A2 design round; #1576 is filed there as new field evidence (see the parent doc's addendum). This doc's fix shrinks B4's trigger window from *any evaluator-legal depth > 1000* to *depths where the interpreter itself refuses (RT_REC_003)* — restoring parity: both engines fail, and (post-B4) both must fail loudly.

## Milestones & Acceptance Criteria

Single milestone, ~0.5 day (F1+F2+F3+tests). Every AC names a file that can fail it and is red on HEAD.

### Milestone M1 — limit parity + wiring (~0.5d)

- **AC1 (#1576 regression, the silent-drop shape)**: a CLI test in `cmd/ailang/` (new `stack_limit_parity_test.go`, using the existing `runWithStdin` helper, `tail_call_parity_test.go:17`): a service-loop program (tail-recursive `loop()` over `readLine()`, handler printing `ASK`, deep non-tail `rep(1170, …)`, then `HANDLED-AFTER`), stdin `hello\nvoice\n`, run as `run --bytecode --verbose --caps IO --entry main`. Assert stdout is **exactly** `HELLO`/`ASK`/`HANDLED len=1170`/`HANDLED-AFTER`, rc 0, and stderr does **not** contain `falling back to evaluator`. **Red on HEAD** (stdout truncated at `ASK`; the fallback warning is present under `--verbose`).
- **AC2 (committed-print-exactly-once, the replay-duplication shape)**: standalone program: `println("START")`, non-tail `rep(5000, …)`, `println("DONE len=…")`, run `--bytecode --verbose`: stdout is **exactly** `START` once + `DONE len=5000`, no fallback line on stderr. **Red on HEAD** (`START` printed twice — the fallback re-run re-performs the committed println).
- **AC3 (flag wiring)**: pure non-tail program at depth 200 (IO-free — the entry returns the computed value, because strict mode hard-errors on EvalOnly stubs like `_io_println`), run `--bytecode --strict-bytecode --max-recursion-depth 50`: rc ≠ 0 with `stack overflow` on stderr. **Red on HEAD** (rc 0 — the flag never reached the VM). Strict mode is the loud leg by design (no fallback to interpret); the service shape of AC1 cannot use strict because of the Phase 2E `readLine` bridge gap (Non-Goals).
- **AC4 (default-boundary cross-engine parity)**: pure non-tail recursion at depth 9000: interpreter and `--bytecode --strict-bytecode` both complete with identical stdout; at depth 11001: interpreter fails with `RT_REC_003` (rc ≠ 0) and the VM strict leg fails with `stack overflow` (rc ≠ 0) — **both loud, neither silent**. **Red on HEAD** (the VM leg fails already at 9000). Do not ride the exact 10000 boundary in the test (entry/caller frames shift it by a handful) — 9000/11001 are safely inside/outside.
- **AC5 (no regression)**: `go test ./internal/vm/... ./internal/testing/... ./cmd/ailang/...` green, including `TestEngineParity_RecursionLimitReachesVM` (unaffected: the harness overrides MaxStack itself), `frame_reuse_test.go` (explicit caps), `vm_fib_test.go` (tail-call 10,000 iterations under MaxStack=5 — TCO unchanged), `run_bytecode_test.go` (fallback-warning text pinned), `tail_call_parity_test.go` (200,000 tail iterations, constant depth).
- **AC6 (docs)**: `docs/docs/reference/limitations.md` states that `--max-recursion-depth` (default 10,000) bounds the VM's frame stack under `--bytecode` the same way it bounds the evaluator.

### Sprint-exit gate

- **AC7 (parity harness reconciliation)**: run the full bytecode parity harness (`scripts/verify_bytecode_parity.go`) before/after. Any file whose bucket moves because deep recursion now *runs on the VM* instead of silently falling back (a former fake-MATCH becoming DIVERGE) must be reconciled **by name** in the implementation report as a newly-surfaced VM bug — never absorbed silently, and never a reason to revert F1 (parent doc AC5/AC11 discipline).

## Conflict Surface (mandatory — touches `internal/vm`, `internal/runner`)

**Positions this changes, and what else already lives there:**

1. `DefaultMaxStack` (`internal/vm/vm.go:14`) — consumed by `NewVM`, whose only `ailang run` caller is `internal/runner/vm.go:202`; the other non-test caller, the named-test harness (`internal/testing/bytecode_engine.go:160`), **overrides** it (`:161-164`), so `ailang test` behavior is unchanged. No test anywhere pins the default value (verified).
2. `vm.MaxStack` semantics — the overflow guard sites (`vm.go:125` CallClosure, `vm.go:303` OpCall) and the framePool cap (`frame.go:81`). Raising the cap does not preallocate (lazy pool) and does not change Go-stack behavior (`vm.run` is iterative; HOF `CallClosure` re-entry nests `run` once per callback chain, unchanged).
3. `--max-recursion-depth` flag plumbing (`internal/runner/entrypoint.go:26,43-44`; `run.go:183-184`; `batch.go:109`) — currently evaluator-only. F2 adds the VM leg at the same layer. **Intentional incompatibility**: `--bytecode --max-recursion-depth N` with N < ~1000 now actually bounds the VM (previously the flag was a no-op for the VM leg — silently, which is exactly the NO-SILENT-FALLBACKS sin). No test depends on the old no-op (verified: `grep --max-recursion-depth` in tests exercises evaluator legs and the already-wired test harness).
4. Tail vs non-tail accounting — tail calls do not grow the frame stack (`OpTailCall` reuses the frame) and do not count toward the evaluator's limit either (#1486, `tail_call_parity_test.go` header). Depth-parity therefore holds for both shapes; `TestTailCallParityWithVM` (200,000 tail iterations) is the regression fixture.
5. Raised ceilings (#1317) — the evaluator honors `--max-recursion-depth` up to millions with fresh goroutines; post-F2 the VM honors it with heap frames (bounded by memory, lighter per level than an evaluator level). The limitations doc's contract ("depth is bounded by N and memory; exceeding N is always RT_REC_003") becomes true for the VM too — with the loud-leg caveat that a *non-strict* VM overrun still falls back silently until B4 lands (scope boundary above).

**Programs that MUST still work post-change (verified to exist):** `cmd/ailang/testdata/tailcall/shapes.ail` + `stdin_loop.ail` (tail-call parity suite, constant depth), `examples/runnable/recursion_quicksort.ail` (VM-native recursion at quicksort depth, far under either cap), `examples/runnable/block_recursion.ail`, `tests/golden/bytecode/*`, and `cmd/ailang/run_bytecode_test.go`'s existing assertions.

## Testing Strategy

- **CLI integration (primary)**: AC1–AC4 as `cmd/ailang/stack_limit_parity_test.go`, using `runWithStdin` (exists, `tail_call_parity_test.go:17`) and `buildAilang(t)`; the fixtures are the minimal repros above (service shape + standalone shape + pure-depth shape), written to `t.TempDir()` with `--relax-modules` like their siblings.
- **Contract pin**: AC4's both-loud-at-overrun assertions are the cross-engine depth contract — the same discipline `TestEngineParity_RecursionLimitReachesVM` applies at the `ailang test` layer, now at the `ailang run` layer.
- **Whole-corpus**: AC7 parity harness before/after (reconciliation-by-name rule).

## Non-Goals

- **B4's fallback policy** (abort loudly vs pre-effect-only restart) — owned by the parent doc, parked pending the A2 design round. This doc only shrinks its trigger window to evaluator-illegal depths.
- **Lowering the evaluator limit to 1000** instead — rejected: it breaks the documented #1317 depth contract and the tail-call parity tests (200,000 iterations), and shrinks what the language can do rather than fixing the divergence.
- **Closing the Phase 2E bridge gaps** (`readLine`/`println` EvalOnly) — why AC1 must run non-strict; the bridge gap is the parent doc's documented backlog.
- **VM memory optimization** for 10,000+ frames (lighter than the evaluator per level; not needed).
- **Changing `--strict-bytecode` semantics** (strict stays: no bridge, loud failure — used by AC3/AC4 as the loud leg).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Deep-recursion example files (>1000 frames) surface latent VM bugs now that they actually run on the VM instead of silently falling back | Medium | AC7 reconciliation-by-name; newly-DIVERGE rows are new VM bugs surfaced, filed, not reverted (parent AC5 discipline) |
| Memory blowup at 10,000 default frames | Low | Heap frames + lazy pool, lighter per level than the evaluator's Go-stack levels that are already legal at 10,000; measured in sprint (AC timing budget) |
| `--max-recursion-depth` accounting differs subtly between engines (evaluator counts builtin-callback re-entry; VM counts frames) | Low | 1:1 for non-tail calls, constant for tail calls on both (verified reasoning in Conflict Surface #4); AC4 pins the default-boundary behavior empirically |
| Frame pool retention at very high caps: `releaseFrame` pools up to `MaxStack` frames (`internal/vm/frame.go:81`), so with `--max-recursion-depth 2000000` a single deep run can leave up to 2M pooled frames (and their register slabs) live until the VM is dropped | Low (opt-in flag; one VM per run) | Accepted for the `ailang run` lifetime; if a long-lived VM ever honours the flag, cap the pool independently of `MaxStack` (e.g. min(MaxStack, a fixed pool ceiling)) |
| B4 lands later and re-litigates this cap | Low | Explicit scope boundary above: F1/F2 are correct under EITHER B4 policy branch (a shared limit is prerequisite-independent of the fallback policy) |

## Review notes 2026-10-08

- **Independently reproduced and fix confirmed.** The reviewer reproduced #1576 on origin/dev and confirmed the fix: with the VM's `DefaultMaxStack` at 10000 the service repro runs fully on the VM; at depth 11001 both engines fail loudly (interpreter `RT_REC_003`, VM strict `stack overflow`).
- **Line refs corrected** (amended in place): `TestEngineParity_RecursionLimitReachesVM` is at `internal/testing/engine_parity_test.go:178`; "exactly one production site" is qualified, because the non-test `internal/testing/bytecode_engine.go:160` also constructs a VM (and overrides its cap); the fallback-warning guard is `internal/runner/entrypoint.go:152-155` (`if !params.Quiet`), not `cmd/ailang/run_helpers.go` (corrected in the parent doc's A2 item 2).
- **Follow-up (cheap, recommended in this sprint or the next):** above the shared limit, a non-strict `--bytecode` run still falls back **silently** under the default `--quiet` until B4 lands. Print the fallback warning even in quiet mode — it is a correctness signal, not chatter — by dropping the `!params.Quiet` guard at `entrypoint.go:152` (check `cmd/ailang/run_bytecode_test.go`, which pins the warning text and its absence in non-fallback runs). This also lets the parent doc's A2 sniffer work without `--verbose`.
- **Risk added**: frame-pool retention at a raised `--max-recursion-depth` (see Risks & Mitigations).

## Related Documents

- [m-bytecode-vm-parity-bugs.md](../../planned/v1_0_0/m-bytecode-vm-parity-bugs.md) — parent doc; owns **B4** (the unsafe-replay fallback this issue exposes in the wild) and the A2 second design round; addendum records #1576
- [m-bytecode-pattern-arity-fix.md](../../planned/v1_0_0/m-bytecode-pattern-arity-fix.md) — the sibling carve-out precedent (#505, ready-to-sprint VM soundness fix independent of the parked A2 round)
- [design_docs/implemented/v0_11_0/m-bytecode-vm.md](../v0_11_0/m-bytecode-vm.md) — VM master design (no design rationale for the 1000 cap exists there; it was a Phase 2B constant whose comment claims parity it never had)
- [design_docs/implemented/v0_52_0/m-iterative-list-remaining.md](../v0_52_0/m-iterative-list-remaining.md) — the v0.52.0 stdlib rewrite that worked around this same VM cap per-helper (the per-case fix this doc replaces at the root)
- `internal/testing/engine_parity_test.go` / `internal/testing/bytecode_engine.go` — the already-shipped, already-tested statement of the contract F1+F2 productionize
- [#1317](https://github.com/sunholo-data/ailang/issues/1317) raised ceilings (`--max-recursion-depth` up to 2M, `docs/docs/reference/limitations.md`); [#1501](https://github.com/sunholo-data/ailang/issues/1501) `CallClosure` faulted-callback frame restore; [#1486](https://github.com/sunholo-data/ailang/issues/1486) tail calls in constant depth

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Evaluator-legal recursion depth no longer diverges between engines; the divergence window (1000, 10000] closes |
| A7: Machines First | +1 | `--max-recursion-depth` becomes a true lever for both engines (was silently a no-op for the VM leg) |
| A11: Structured Failure | +1 | The silent-truncation path for evaluator-legal depth is eliminated; at illegal depth both engines fail loudly (with B4 owning the last silent leg) |
| Others | 0 | No language-surface, effect, or authority changes |

**Net Score: +3** → **Proceed.** Hard violations: none.

## Sizing

~0.5 day: F1 (1 line + comment), F2 (3 lines), F3 (one paragraph), AC1–AC4 test file (~150 LOC), AC7 harness run + reconciliation. Fits the standard sprint box; pair with [m-bytecode-pattern-arity-fix.md](../../planned/v1_0_0/m-bytecode-pattern-arity-fix.md)'s sprint if the planner wants one bytecode-soundness pass.

---

**Document created**: 2026-10-08
**Author**: design-doc-creator (coordinator task `task-d03d7256`), root cause verified first-party in this worktree against the prebuilt `ailang` v0.52.5 and the code at HEAD `1dfd5615`.
