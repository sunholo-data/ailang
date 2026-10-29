# Sprint Plan: M-EFFECT-EXPLICIT-EMPTY-CALLBACK-CONTRACT

**Status:** IMPLEMENTATION COMPLETE / INDEPENDENT EVALUATION PENDING — Mark explicitly authorized “Execute explicit-empty sprint” on 2026-10-29.
**Design:** `design_docs/planned/v0_56_0/m-effect-explicit-empty-callback-contract.md`.
**Progress:** `.ailang/state/sprints/sprint_M-EFFECT-EXPLICIT-EMPTY-CALLBACK-CONTRACT.json`.
**Target:** v0.56.0 breaking minor; no release/version bump/deployment.
**Measured dev:** `61716789ae8f5ebcea534b1a993ca00d1533ec04` (2026-10-29).
**Required #1958 checkpoint:** `74a3716867c72f008d3f94df9fa36360c71d7e25`, draft PR #2005. Its row guard is required for the new closed-empty rejection; do not execute on bare dev without that guard.
**Duration:** 2 engineering days including corpus/consumer review allowance; four 60–90 minute milestone runs plus bounded external gates.
**Estimate:** 600 authored LOC (mostly tests/docs); moved conversion code and generated prompt mirrors excluded.
**Risk:** high, bounded by source-presence distinctions, negative engine/cache tests and actual consumer proof.
**AILANG prompt version loaded:** v0.16.9; full prompt loaded during planning, active version confirmed with `prompt --version-active`.

## Approved decisions and execution gate

Mark explicitly approved **“Approve option A design”** on 2026-10-29. Copying design rulings, not reopening them:

- F1: implement option A; no parameter-position precision redesign.
- F2: a function-type `! {}` is closed empty; omission remains implicit/open; nonempty concrete and declared row-polymorphic contracts retain current behavior.
- F3: accept IO-to-explicit-empty-slot rejection in v0.56.0, with an explicit breaking migration note.
- F4: the exact four pure-field annotations in motoko FixtureOverrides may be delivered **after** the new compiler proves all 60 core files and 9 extension boots. No other consumer widening is authorized.

There are no remaining OPEN design-freeze human decisions. This concrete plan still requires user approval and an explicit **execute sprint** instruction before code changes. Do not send coordinator handoffs or trigger another executor automatically.

If execution uncovers a genuinely new compatibility class, needs a broader consumer patch, or would change omitted/nonempty/polymorphic semantics: **STOP: commit current work, report `BLOCKED: <new decision and evidence>`, and end the run**. Existing ruled F1–F4 are not new blockers. A failed four-line migration remains a failure, never a reason to widen rows silently.

## Baseline, ownership and coordination

The #1958 owner executes this amendment in the same isolated owned workstream (`effect-row-1958`, `fix/effect-row-open-closed-soundness`), keeping commits separate by milestone. First bank the approved plan/JSON, fetch and rebase the clean owned branch onto current dev; preserve milestone history and other worktrees. Existing #1958 code remains a dependency, not duplicated implementation. Main checkout benchmark JSON edits stay parked.

You are not alone in the codebase. Do not revert others' edits. The executor owns only this plan's conversion/printing/iface/tests/example/docs/prompt delta and the conditional isolated consumer migration.

- **#1963:** already merged. Keep its conservative per-parameter mask/union and depth/ownership invariants unchanged.
- **#1790:** operator-pending, open head `8236413d3ae7d217f3e447ca51d4c6bda380737b`. It owns outer pure declarations and derived iface purity. Do not implement or merge it. Recheck before touching iface/conversion/goldens; preserve its metadata if it lands. The second lander regenerates interfaces from the combined tree. An overlapping file alone is coordination, not a new permission gate.
- **#1719 / PR #2007:** head `8771ceae5` reserves prompt **v0.16.10**, inheriting .9 unchanged. M3's teaching revision uses the next unused version **after .10** (intended .11), based on the final .10 text once it lands; preserve .9/.10 byte-for-byte. Recheck the registry before creating a version. M1/M2 can proceed while that separate merge is pending; do not import unlanded IFC code into this PR to obtain a prompt.
- **Consumer:** fork `sunholo-voight-kampff/motoko_agent`, pinned branch `sunholo/main-dst-20261027`, currently merge `af1d56513d3ae314bd831fd149c7b6f2503a361c`. Use a new isolated worktree/feature branch. Never edit/switch live mk checkouts or upstream `arniwesth/motoko_agent`.
- **Independent evaluator:** root evaluates the final amendment; do not self-evaluate. #1958 M4 remains unpassed until final compatibility and independent acceptance are recorded.

## Planning snapshot and velocity

The four-arm evidence is committed beside the design. Base and #1958-only accept mixed callback storage hand-off; #1963-only and combined reject. Direct literals and all-nonempty-concrete controls pass; explicit empty annotations currently fail to distinguish pure fields. The failed exact four-line scratch patch is retained. At planning time, no implementation had begun. Current measured execution is recorded in the companion validation report and sprint JSON.

The velocity helper ran but ended after its commit listing without a usable summary. Recent #1958 work delivered roughly 600 lines across four incremental milestones, predominantly tests; this attended session is not a reliable calendar throughput sample. Use a conservative **300 authored LOC/day**, 600 LOC / 2 days, with separate review and consumer-run allowance. Do not count generated 2,700-line prompt mirrors or moved conversion helpers as new semantic code.

## Registry reuse

`ailang pkg search effect` returned eight application packages (logging, billing_entitlements, external_backend, email, gmail, decisions, world/core, crew_core). Their published capability summaries do not implement compiler function-type elaboration or callback ownership. This sprint adds no package-like runtime capability or registry dependency. Every milestone records `none`: compiler/source contract, repository validation, or exact source annotation migration cannot be replaced by an application package. No package publication or inbox dispatch is needed.

The skill's JSON helper implicitly imports GitHub messages; this session requires gh-only GitHub operations and no inbox/coordinator mutations. Create the fully populated JSON directly using the skill schema and run its validator; do not invoke the importer.

## M1: Preserve explicit-empty contracts through conversion (~170 LOC)

**Goal:** `! {}` becomes a closed concrete pure callback contract without changing omitted, nonempty or row-variable behavior.
**Estimate:** 40 production + 130 tests; 0.5 day.
**Dependencies:** approved execution, existing #1958 guard.

**Ownership:** `internal/elaborate/file_funcs.go`, companion `internal/elaborate/function_effect_empty_test.go`, `internal/types/typechecker.go` and companion tests. Do not grow large unrelated files or refactor the unifier.

- [x] Add failing-first four-state conversion tests: absent=nil, explicit empty=non-nil empty, concrete nonempty, declared row variable; cover nested aliases and multi-argument callback types.
- [x] Use existing parser presence; no new syntax, token or AST flag. Explicit empty yields closed empty row + Concrete=true/Implicit=false. Do not merely mark an open empty compatibility row concrete.
- [x] Preserve omitted-arrow L1-open, nonempty concrete compatibility tails, parameter/budget metadata, and row-polymorphic sharing exactly. Audit legacy AST conversion separately; do not silently change its existing omitted-arrow rule.
- [x] Unit/pipeline discriminator: explicit empty slot rejects an annotated IO value even when its caller already declares IO; matching pure callback accepts. Existing omitted hand-off still charges.
- [x] Run targeted conversion/types/pipeline tests, `make test-changed`, lint and sizes; commit M1 with evidence of red→green and preserved controls.

## M2: Preserve contracts in strings, interfaces, aliases and cache (~170 LOC)

**Goal:** imported and warm-cache programs have the same contract as source.
**Estimate:** 30 production + 140 tests; 0.5 day.
**Dependencies:** M1.

**Ownership:** `internal/ast/ast_type.go`, companion AST tests, `internal/iface` conversion helper/companion tests, pipeline cache integration tests. `iface/builder.go` is 788 lines: move its existing AST conversion helper to `internal/iface/type_conversion.go` before extending it; retain other conversion cases unchanged. Count moved lines separately.

- [x] Preserve explicit empty in FuncType.String and AST print/compact JSON simplify(FuncType), without changing Lambda/declaration printing; retain canonical formatter nil/empty behavior. Add omitted/empty/nonempty JSON discriminators.
- [x] Align iface alias/constructor conversion and restoreNestedEffectRow ownership with M1; preserve explicit-empty direct params/returns. Normalized iface text must distinguish empty contracts on exports/constructors/aliases; HashProjection/SignatureSet distinguish constructor-only and export-only changes while intentionally excluding alias bodies. Alias-only JSON plus internal aliasDigest and warm direct/transitive edits provide separate coverage, without a schema or hash-policy change. Coordinate #1790.
- [x] Verify existing TFunc2 substitution and JSON flag/row persistence with real conversion/import fixtures; no parallel contract map or new cache design.
- [x] Cold/warm/imported alias tests exercise direct, tuple/list/Option/record positions and actual multi-argument callbacks. Add BOTH cross-compiler old-accepted-cache→fixed rejection and SAME-binary dependency edits omission↔!{} (direct/transitive alias plus callback-bearing constructor/export). Compiler identity and dependency digests must invalidate separately. Narrow internal rendering/digest correction is authorized if needed; no new iface schema or unrelated purity change. Notify #1771 owner before reserving an existing cache epoch.
- [x] Generate/verify current interfaces using existing tools. Record exact changed modules and why; preserve newer #1790 changes if present. Run iface/format/AST/cache-focused tests plus changed/core gates, lint, sizes and boundaries; commit M2.

## M3: Both-engine regressions, example and teaching (~180 LOC)

**Goal:** demonstrate the approved distinction and prevent runtime caps from masking a static regression.
**Estimate:** 120 tests + 60 example/docs/teaching lines; 0.5 day, generated prompt mirrors excluded.
**Dependencies:** M2; prompt revision waits for #1719's .10 baseline to land or an explicit artifact handoff from root.

**Ownership:** companion CLI/pipeline tests; `examples/runnable/explicit_empty_callback_contracts.ail` and its surgical manifest entry/statistics; changelog fragment; effects reference; new teaching prompt version after .10 via prompt-manager.

- [x] Evaluator and bytecode reject IO-to-explicit-empty-slot before marker output from direct/imported/nested variants, even with IO capability and an IO caller. Positive pure-slot calls execute; named IO callbacks remain storable through fully concrete records.
- [x] Preserve original #1958 and #1963 regressions: omitted nested store-only hand-offs reject; literals/length/all-concrete stores, row-poly callbacks, AI streaming wrappers and ceiling authority controls remain green.
- [x] Add a runnable example with explicit-empty pure `probe`, explicit-IO stored `render`, pure `build/answer`, and IO main printing 42. The render marker must not run during construction. Keep the manifest entry/modules and aggregate totals coherent.
- [x] Changelog opens `### ` and explains the v0.56.0 breaking callback-type change and intended migration. Reference docs distinguish omitted, empty, nonempty and row-polymorphic callback types.
- [x] Load prompt-manager; create next unused prompt version after .10, preserving .9/.10 identities and content. Teach the four-state distinction; do not append new semantics to a frozen existing version.
- [x] Run targeted engine/prompt tests, `make test-changed`, lint, sizes, examples/manifest, interface gates; commit M3.

**Example syntax checklist:**

| Item | Decision |
|---|---|
| Contracts | skip: higher-order stored callbacks and field invocation are outside the decidable SMT fragment; this example proves a static callback contract, not an arithmetic theorem |
| Effects | include: `probe: int -> int ! {}`, `render: int -> int ! {IO}`, pure build/answer, main `! {IO}` |
| Inline tests | include: pure `answer() -> int tests [((), 42)]`; marker absence and engine parity are asserted by Go integration tests |

## M4: Actual consumer proof, conditional delivery and final handoff (~80 LOC)

**Goal:** resolve the original measured compatibility block without broadening authority or losing evidence.
**Estimate:** 4 consumer annotation lines + about 76 documentation/provenance/evidence lines; 0.5 day including bounded gates.
**Dependencies:** M3.

- [x] Inventory existing explicit-empty **function types** across the corpus (AST-aware, distinguish declarations). Run base/fixed no-cache comparisons with matching stdlib and pinned roots; preserve intentional rejection cases and report every unexplained flip.
- [x] In scratch first, apply exactly four changes to FixtureOverrides in `src/core/test/ext_fixture.ail`: append `! {}` to describe, prompt, budget, policy arrow types. Keep all other callbacks, outer factory rows and Ports ABI unchanged.
- [x] With the actual final compiler plus #1958/#1963, prove all **60 core files** via fresh no-cache checks and all **9 extension boots** via `gmake check_core`/verify_extensions. Use known 180-second per-file bounds and concurrency two; classify timeouts honestly. If the exact patch does not restore gates, STOP under the new-decision rule; do not widen more code.
- [x] Only after proof, F4 authorizes delivery: recheck fork overlaps/auth, create isolated feature worktree from fresh pinned fork branch, commit/push exact four-line patch and open its fork-target PR via gh. Normal fork-only merge after relevant gates and review; never --admin or upstream writes. Inherited CI/compiler failures require measured base proof and disclosure, not a false all-green claim.
- [x] Pin the exact full merged consumer SHA in Docker and MOTOKO.md; distinguish source pin from deployment and keep live rig checkout unchanged. Preserve original width60→55, delivered width60→60, combined nested55/60 and failed four-line-before-compiler evidence alongside the new proven result.
- [x] Final gates: `make test-changed`, lint, sizes, boundaries/architecture closure, changelog, freeze/verify all interfaces, verify-examples. Do not run `make test`; exact-head CI supplies the full suite, subject to the evaluator's explicit gate policy.
- [ ] Mark only measured milestones passed; bank #1958 M4 implementation evidence after actual proof/delivery and required gates, then hand root exact code/consumer SHAs, plan/JSON and logs for independent evaluation. Final acceptance requires that independent PASS; do not self-score or archive docs before it.

## Schedule, validation and handoff

Day 1: M1 then M2, committing each. Day 2: M3, then M4 corpus/conditional consumer delivery and review. Tests are failing-first contract discriminators, not mirrors of implementation. Existing immutable JSON/milestone criteria remain intact; update completion only with evidence.

Native structural/focused gates run once per meaningful change. Exact-head CI owns broad tests. Repeat only failed/affected checks when a new delta requires it. Preserve historical failed evidence and prior evaluator verdicts; no exclusions, threshold changes or fabricated pass.

Execution is now explicitly approved. Invoke sprint-executor, bank plan/JSON, refresh base/overlap status and begin M1. Final independent sprint-evaluator is root. No coordinator sends/acks or automatic executor trigger is authorized by this planning task.
