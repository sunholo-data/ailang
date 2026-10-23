# Sprint Plan: M-MAINTENANCE-EVAL-ADAPTER

**Status:** Partially implemented; protected admission blocked. Mark authorized execution on 2026-10-23. Seven milestones pass; five remain incomplete or partially delivered.
**Goal:** Connect the real maintenance taskpacks to existing executors, preserving owned artifacts and emitting native evidence, with a qualified isolation boundary before live admission.
**Duration:** 6–8 engineering days including qualification/review buffer; planning envelope, not measured productive-day velocity.
**Scope:** 12 bounded milestones, ~3,130 implementation/test/config lines. Medium/high risk: tool containment and truthful leaf/accounting completeness.
**Repositories:** core sunholo-data/ailang (target dev) owns orchestration/executor integration; sunholo-data/ailang-packages (main) owns task contracts, grading adapters and analysis consumer.
**Inputs:** [Taskpack design](m-maintenance-eval-taskpack.md), [ledger design](m-maintenance-bench-evidence-ledger.md), completed [foundation sprint](m-maintenance-eval-foundation-sprint-plan.md). [Foundation PR160](https://github.com/sunholo-data/ailang-packages/pull/160) merged at `a4163559a2e20dddc1a1bb0869a06e42372994a1`; 185 native tests/500 generated cases, independent 90/100 and final CI passed.

## Decisions and boundaries

- Keep the existing executor/model/provider parsers and finite agent calls. The new code orchestrates exactly the declared maintenance stages; it is not a new agent loop, generic workflow engine, dashboard or benchmark rotation.
- First local containment implementation proposed: **pi**, reusing its existing allowlist/embedded-extension path and bash sandbox. Motoko currently advertises streaming/local-workspace/structured-output, but its toolpolicy.go refuses restrictions it cannot honor; it is not silently treated as protected. Backend choice and any later change stay visible in the manifest.
- Fresh sessions at every stage; carry owned source/locks/public app files/model notes, not old model conversations, reference solutions, oracle outputs or a gold reset. Competent starters make this maintenance-only; initial generation is a later distinct cohort.
- Oracle visibility applies to both the model's tools and the candidate program when grading. A separate directory is not proof of a boundary. Actual containment canaries and source-root inventory precede live admission.
- All ordinary development/CI runs are no-model. This sprint does not authorize an inference canary, registry 0.2 upload, core release, rig occupation or the 24-episode model campaign. A later launch names models, prices and actual enforceable bounds. Public source-known tasks stay development evidence.
- Prefer no new global env vars. If trusted per-task sandbox paths need a carrier, make it an optional task-owned contract derived by the existing environment builder; it must not be an unrestricted agent_env override.

## Verified inventory and systemic risks

Core source observed at cd33d050b2276633c82624453309ebf7fcea9e2f; installed AILANG v0.54.18. The original core checkout has unrelated dirty docs/state and is left parked. Execution uses a clean isolated core clone/worktree and explicit package source revision.

| Existing surface | Verified implication |
|---|---|
| eval_suite.go flags/discovery/dry-run/canary path | New mode needs an early route, not ordinary discovery of saturated eval tasks |
| agent_runner_multi.go workspace creation and deferred RemoveAll | Cannot call it repeatedly and assume source retention; extract task construction, own stage/checkpoint lifecycle explicitly |
| executor.Task/Result | Existing config, effective tools, pricing/provenance and aggregate counters are reusable; optional flags can be requests rather than guarantees |
| pi/events.go | Per-message usage exists and raw events are retained, but aggregate integer fields/default zeros do not prove original field presence or leaf coverage |
| motoko/toolpolicy.go + Capabilities | Restricted tools/program-policy requests are refused; native catalog remains broader than a protected lane |
| tools/pi-extensions/README.md + sandbox/ | Bash OS wrapper and write fencing exist; native read is explicitly unfenced, so mission containment is not oracle protection |
| managed_agents remote sandbox | Filesystem separation exists, but current fenced-text/patch export is not a qualified three-stage artifact bridge; do not choose it solely for a capability flag |
| File sizes | eval_suite.go 796, agent_runner_multi.go 781, motoko.go 770, parser.go 613; cohesive companion extractions are required |
| Native maintenance_bench 0.2 | Strict v2 accounting includes fresh+generated; scale 6 money refuses unsupported precision rather than rounding; source is merged but registry still 0.1 |

No Docker CLI was present in this attended environment. That inventory is not proof a new container platform is required. Reuse/qualify existing OS/extension primitives first; a failed boundary becomes an explicit implementation blocker, never a weaker default. This is a scoped admission claim over declared run/source roots, not a universal exfiltration/host-security certification.

Actual raw usage/billing precision and provider cache presence may prevent some economic comparisons. Preserve that availability signal. A later exact-decimal/generalized-money design can use reproduced source receipts; it is not smuggled into this runner or claimed a stdlib absence before audit.

## Registry reuse and velocity

On 2026-10-23 ran `ailang pkg search maintenance`, `sandbox`, `executor`, then `pkg info`/`pkg docs sunholo/maintenance_bench`. Maintenance found 0.1.0; sandbox search returned an unrelated simulation package; executor found none. The native 0.2 source from merged a416355 is required explicitly, never a silent fallback to 0.1.

- M1/M5/M10/M11/M12: **contribute** to existing maintenance_bench/taskpack source where package assets or analysis are affected.
- M2/M3/M4/M6/M7/M8/M9: **none** for a new registry dependency; these extend existing internal eval/executor/extension seams, not a new reusable runner package. Existing sandbox dependency is reused with its actual resolved lock/version qualified at implementation.

Foundation actual net code 4,653 physical .ail/.py/.sh lines versus 3,700 planned; reference snapshots/oracles contributed to the difference. Attended-agent elapsed time is not human engineering days, and productive LOC/day remains unavailable. The new estimate budgets six working days plus two for containment/accounting qualification and independent review; it is not derived from an invented speedup.

## Milestones

Core paths and package paths are distinguished below. Each coding slice should fit 60–90 minutes plus verification; split further if actual integration needs it. Do not exceed 800-line source files or use a whole-repo test run as an executor gate.

### M1: Frozen execution bindings (~320 LOC)

**ID:** `M1_EXECUTION_MANIFEST`. **Dependencies:** none. **Estimate:** 200 source/config +120 tests.

**Files/examples:** Core internal/eval_harness/maintenance_manifest{,_test}.go; package taskpacks/v1/execution/manifest.schema.json and development.example.json.

Add a versioned maintenance.execution/1 envelope referencing the merged ledger trajectory plan and digest-bound native starters, stage requests, public checks, oracle drivers, source roots and toolchain identities. Keep controller-only paths out of the agent payload.

- [x] Validate the finite 8/24 pilot and general bounded schedule, unique arms/IDs, stage requirements/oracles, native counterpart responsibilities, exact hashes and explicit model/harness/settings identities.
- [x] Reject traversal/absolute carried-artifact paths, symlinks/device files, duplicate JSON keys and conflicting ordinary-suite selection. Unmatched or altered stage contracts cannot silently compare.
- [x] Pin package source a4163559a2e20dddc1a1bb0869a06e42372994a1 and ledger v2. Registry0.1 lacks these APIs; never substitute it for unpublished 0.2.

### M2: Opt-in CLI routing and zero-dispatch dry run (~220 LOC)

**ID:** `M2_CLI_MODE`. **Dependencies:** `M1_EXECUTION_MANIFEST`. **Estimate:** 120 source/config +100 tests.

**Files/examples:** Core cmd/ailang/eval_suite_maintenance{,_test}.go, minimal eval_suite.go integration and help.go.

Implement the already proposed --maintenance-manifest flag through a small companion, routing before benchmark discovery, suite/canary/queue work. Use fs.Visit to distinguish explicit conflicting flags from defaults.

- [x] --agent is required. --dry-run prints the complete resolved finite schedule, source/analysis identities and budget/isolation availability without creating executors, requesting credentials or calling a model/canary.
- [x] Reject ordinary benchmark/tier/lang/trial/condition/queue/browser/cohort flags when they would override the manifest; document the exact allowed common flags. Existing invocations preserve behavior.
- [x] Keep eval_suite.go below 800 lines by extracting one cohesive existing helper, not shaving comments. CLI help describes only implemented behavior and an executable dry-run example.

### M3: Owned checkpoints and controlled workspace lifecycle (~300 LOC)

**ID:** `M3_CHECKPOINTS`. **Dependencies:** `M1_EXECUTION_MANIFEST`. **Estimate:** 180 source/config +120 tests.

**Files/examples:** Core internal/eval_harness/maintenance_workspace{,_test}.go.

Materialize native starters once and new stage workspaces from each trajectory's own source/locks/public app files/handover. Hash and retain failed snapshots. Keep controller evidence, future prompts, references, transcripts and hidden expected values in separate roots.

- [ ] A broken S1 is the exact source basis for S2; no reference/gold restoration. Missing required app files remain a model artifact failure, not a silently healed starter.
- [ ] Carry only declared regular files; explicit tests cover symlink traversal, rename/delete, excluded logs/caches and create-only immutable output. Record actual copied manifests/hashes.
- [ ] Reap child processes before grading/checkpoint finalization and on all error exits; preserve result artifacts and spent work while isolating build/runtime outputs per source identity.

### M4: Three-stage orchestration over executor interface (~300 LOC)

**ID:** `M4_FINITE_STAGES`. **Dependencies:** `M3_CHECKPOINTS`. **Estimate:** 170 source/config +130 tests.

**Files/examples:** Core internal/eval_harness/maintenance_runner{,_test}.go.

Run a finite declared stage list with injected executor.Executor for tests and the existing factory for admitted live backends. Every stage starts a fresh session; retention is files/notes, never ResumeSessionID or old conversations.

- [x] Scripted execution accounts for every scheduled stage after failure, timeout, cancellation, invalid setup, budget stop and never-started dispatch. Stage schedule is fixed and failed spend never disappears.
- [x] Only the current requirement/public diagnostics reach the model. Controller oracle verdicts are withheld during the primary trajectory; future-stage source is not seeded early.
- [x] Write native-ledger-compatible rows with prior/candidate/check/oracle/attempt identities; blocked continuation keeps the last actual owned checkpoint. Do not add a generic workflow engine.

### M5: Grade arbitrary retained candidates with trusted task drivers (~250 LOC)

**ID:** `M5_ORACLE_BRIDGE`. **Dependencies:** `M3_CHECKPOINTS`. **Estimate:** 150 source/config +100 tests.

**Files/examples:** Package taskpacks/v1/{resource-ownership,text-fidelity}/grade_candidate.py and shared protocol; core maintenance_grade{,_test}.go.

Factor existing oracle/candidate launch logic so a controller can grade an arbitrary frozen candidate. The oracle stays host-owned; candidate compilation/execution uses the qualified confinement policy too. Existing reference/mutation qualifiers remain valid controls.

- [ ] Reference candidates pass and compiled semantic mutants fail on specific obligations. Deliberately uncompilable model source is an artifact/check failure; missing trusted tools/fixtures are invalid measurement setup, not model errors.
- [ ] Trusted check/oracle source hashes cannot be weakened by agent edits. Reject mismatched input/output/source hashes, skipped assertions, stale builds, malformed typed output and overwritten evidence.
- [ ] Grade in a clean runtime against all retained obligations. Candidate code cannot read or rewrite oracle expected values/controller data during execution; test that boundary rather than assuming the host oracle being outside the workspace is sufficient.

### M6: Reuse task construction and existing dispatch controls (~270 LOC)

**ID:** `M6_SHARED_TASK_SEAM`. **Dependencies:** `M2_CLI_MODE`, `M4_FINITE_STAGES`. **Estimate:** 160 source/config +110 tests.

**Files/examples:** Core internal/eval_harness/agent_task{,_test}.go, agent_runner_multi.go and maintenance_runner.go.

Extract the reusable task assembly below workspace creation/cleanup; ordinary one-shot execution remains its caller. Reuse model resolution, pricing/preflight, capability validation, rig admission, timeouts, effective tool policy and parser/event handling.

- [x] Task construction parity tests pin ordinary prompts/metadata/model/timeouts/pricing/env/tool behavior; no DEBUG_AGENT environment hack to preserve workspaces.
- [x] New stages use explicit isolated ambient-context/extensions requests and frozen prompt/profile/settings digests. Unsupported executor capabilities fail before model dispatch.
- [x] Do not grow agent_runner_multi.go(781) or motoko.go(770) past800 lines or change Motoko core. Register protected-mode support only on an actually qualified backend.

### M7: Trusted per-eval native tool fence (~220 LOC)

**ID:** `M7_NATIVE_READ_FENCE`. **Dependencies:** `M3_CHECKPOINTS`, `M6_SHARED_TASK_SEAM`. **Estimate:** 120 source/config +100 tests.

**Files/examples:** Core tools/pi-extensions/maintenance-fence.ts and tests, corresponding embedded profile asset and drift gate.

Propose pi as the first local backend to qualify because existing tool filtering and bash sandbox integration are reusable. Add a narrowly scoped trusted read/write/search/list guard; mission write fencing alone is not oracle isolation.

- [x] No-model tests cover real native read/write/edit/grep/find/ls surfaces, relative cwd != root, prefix confusion, symlink ancestors, unknown path shapes and protected developer files. Allowed native app work remains possible for both languages.
- [x] Only the declared canonical tool set and trusted extensions load; no ambient/global extension or context-file discovery. Protect policy/extension assets and controller/source-bank roots, including shared Git object stores and other trajectory workspaces.
- [x] First-backend selection is a proposal for this sprint, not a claim existing pi or Motoko is already qualified. Record effective tool/policy digest and refuse unmeasured boundaries.

### M8: Reuse bash sandbox and qualify the actual boundary (~260 LOC)

**ID:** `M8_OS_CONTAINMENT`. **Dependencies:** `M7_NATIVE_READ_FENCE`, `M5_ORACLE_BRIDGE`. **Estimate:** 140 source/config +120 tests.

**Files/examples:** Core internal/executor/eval_containment.go, pi/{containment,args,profile_assets}*.go, existing sandbox/ wrapper and no-model integration probes.

Add an optional explicit eval-containment task contract, leaving nil/default payloads unchanged. Reuse the existing sandbox runtime for shell/compiler/app execution and pin its resolved dependency identity; do not build a VM/container platform or edit the installed personal/global pi setup.

- [ ] An actual no-model process canary must fail to read protected oracle/reference/controller roots via cat, Python open, AILANG FS, symlink traversal and native tools; it must also refuse modifying developer/oracle evidence while ordinary app builds run.
- [ ] Tool subprocess network cannot fetch reference solutions; inference stays in the existing executor. Qualify exact platform/tool versions and policy, fail closed if sandbox initialization/support is absent, and never infer universal host/holdout protection from a few checks.
- [ ] Declare the protected-root threat boundary and require all known source/reference copies and Git stores in it. If existing primitives cannot meet it, keep live admission closed and raise a scoped extension design rather than launching weaker comparative trials. Ordinary executor paths remain unchanged.

### M9: Existing-parser leaf receipts and honest presence (~300 LOC)

**ID:** `M9_USAGE_EVIDENCE`. **Dependencies:** `M6_SHARED_TASK_SEAM`. **Estimate:** 180 source/config +120 tests.

**Files/examples:** Core internal/executor/usage_evidence.go and pi/events usage helper/tests; maintenance_receipts{,_test}.go.

Capture typed usage evidence at the existing parser, preserving request/import identity, original field presence, source version/event digest, cumulative-versus-leaf semantics, retries/compaction coverage and cost provenance. Export neutral receipts accepted by the AILANG package.

- [x] Fixtures cover absent versus explicit zero, inclusive/exclusive input, combined reasoning, incomplete streams, retries, compaction and summary/leaf reconciliation; no second provider-log parser in Go adapter or AILANG package.
- [x] Aggregate Result integers alone do not establish reported cache/reasoning zero or per-request completeness. Untraceable calls/overlap keep affected economics unavailable; retain original reported totals as source evidence.
- [x] Preserve raw decimal cost text when available. Never round float CostUSD to six decimals to fabricate exact money; unsupported precision remains explicit under current0.2 rules. Do not relabel registry estimates/subscription equivalents as metered invoices.

### M10: Use native package analysis for runner outputs (~210 LOC)

**ID:** `M10_NATIVE_ANALYSIS`. **Dependencies:** `M4_FINITE_STAGES`, `M9_USAGE_EVIDENCE`. **Estimate:** 120 source/config +90 tests.

**Files/examples:** Core maintenance_analysis{,_test}.go; package explicit source consumer/example and Makefile target.

Write the existing trajectory/runs/usage v2 files and invoke the pinned native maintenance-ledger analysis. Do not port cohort, monetary or paired metrics into the Go runner.

- [x] A clean source-path consumer uses merged0.2 code explicitly until registry publication is authorized. Verify actual compiler/source/lock/tool identities and store external provenance; reportId remains declared, not self-attesting.
- [x] Independent known fixture reproduces185-test foundation expectations:8/24 coverage, cost delta-22500micros, fresh-input diagnostic-550, fresh+generated headline-540; all are labelled instrument controls.
- [x] Unknown/missing usage and invalid outcomes produce unavailable economic claims while valid correctness remains visible. Emit reference-linked reproduction cards/draft investigation material without approving roadmap code or claiming a language win.

### M11: Bounded no-model integration and failure matrix (~280 LOC)

**ID:** `M11_END_TO_END`. **Dependencies:** `M5_ORACLE_BRIDGE`, `M8_OS_CONTAINMENT`, `M10_NATIVE_ANALYSIS`. **Estimate:** 160 source/config +120 tests.

**Files/examples:** Core maintenance_integration_test.go; package taskpacks/v1/execution/ integration fixtures and replay instructions.

Run the finite 8/24 schedule through a scripted executor and real native candidate grading, then no-model containment probes for the selected backend. Keep fake accounting and measured native verdicts separate from real model outcomes.

- [ ] Plant broken early source, public-green semantic regressions, altered oracle, changed stage/provider/prompt identity, dropped costly retry, missing receipt, stale build, timeout and cancellation; each has an independent expected classification.
- [ ] All 24 rows and retained snapshots survive partial failure, and the model receives no private oracle feedback. Pipeline/source/receipt digests bind each actual artifact.
- [ ] No paid canary/provider/model call is allowed in normal tests or dry-run. If platform containment is unavailable, report the admission gap explicitly and do not mark its criterion passed with a skip.

### M12: Focused gates, help/docs and independent evaluation (~200 LOC)

**ID:** `M12_DELIVERY`. **Dependencies:** `M11_END_TO_END`. **Estimate:** 140 source/config +60 tests.

**Files/examples:** Core docs guide/help, changelogs/unreleased fragment and focused CI/asset checks; package docs/example update; sprint state and retrospective.

Deliver separate scoped core and package source PRs with pinned cross-repo identities, exact verification, proposed launch template and independent sprint evaluation. Do not release a core binary, publish a registry package or start the campaign in this sprint.

- [ ] Run focused Go/executor/CLI tests, make test-changed plus test-core, lint/file-size/boundary and pi-asset gates; TS guard tests and actual no-model OS checks are separate recorded outcomes. No full Go suite executor gate.
- [ ] Native package tests/properties/examples/quality/dry-run remain green without new warning allowlists or formatter exclusions; original frozen studies are unchanged.
- [ ] Publish reviewable source PRs only after concrete checks, then independent sprint-evaluator review; complete all state/plan criteria in the same trees. Exact model/provider/rates/enforceable bounds and actual launch authorization remain a later explicit decision.

## Schedule, acceptance and launch handoff

| Day | Deliverable |
|---|---|
|1|M1/M2 frozen bindings and CLI dry-run; no executor constructed|
|2|M3/M4 owned snapshots and scripted fixed stages|
|3|M5/M6 arbitrary candidate grading and shared task seam parity|
|4|M7/M8 native-tool/OS confinement and actual no-model canaries|
|5|M9/M10 source-qualified usage evidence and native analysis bridge|
|6|M11 full 8/24 scripted/native failure matrix and source identities|
|7–8|M12 focused CI, documentation, independent review and buffer|

Success is a real callable adapter with measured qualification and honest unavailable states, not an AILANG win. If current primitives cannot meet the declared worker/candidate boundary, stop live admission and report the failed canary/proposed extension; do not tick M8/M11 or call the whole sprint complete on a mock or skip.

No actual model is selected here. The later launch template freezes model/provider/harness/tool/prompt/source/seed identities, coverage policy, canary/preparation overhead, exact prices and currency basis, max requests/turns, the accounted-token measure, stage/campaign wall bounds and failure continuation. Existing budget limits may stop after banked/in-flight spend; do not call them hard pre-request meters. Required enforcement gaps must be resolved or explicitly priced/approved before calls.

Proposed dry-run invocation after implementation (currently unsupported):

```sh
ailang eval-suite --agent --maintenance-manifest /absolute/path/maintenance.execution.json --dry-run
```

It must not discover standing benchmarks or call a model. The two task families / 8 trajectories / 24 changes remain a feasibility sample; population/average-working-day claims and causal type-feedback claims need their own corpus/intervention.

## Verification and example syntax gate

AILANG prompt version loaded: v0.16.7; SHA256: 214ca761650bf8ba9f05a8e037a791a65496313ae8e846ff389decfd85786ab6, identical to the previously read teaching prompt. Reload before .ail edits. Core changes are Go/TypeScript and do not change compiler/evaluator/effect semantics.

| Showcase | Contracts | Effects | Inline tests/properties |
|---|---|---|---|
| Native source-path analysis consumer/example | include in existing pure ledger owners: partitions, missingness, source joins; skip wrapper: no new pure rules | include main IO/FS as needed; no AI/Net/Process for analysis | include owner laws and hand-counted fixture; wrapper skip with reason, execute example separately |
| Core CLI dry-run | skip: Go CLI is not an AILANG module | skip: Go inventory; it must cause no model/provider/credential/rig effect | include focused flag/discovery/zero-dispatch tests and actual dry-run example |
| Native candidate drivers | skip host Python orchestration SMT; existing AILANG cap/text predicates retain meaningful contracts | explicit native IO/FS/Env only as frozen task requires, no hidden access | run native public checks, exact external cases, malicious candidate and weakened-oracle controls |

Use make test-changed plus test-core and targeted Go tests for touched packages, make check-file-sizes/check-boundaries/verify-pi-assets and relevant lint/help gates. TS hooks need tests that drive actual installed tool behavior; shell confinement needs real OS probes in a supported CI lane, with unknown support not accepted as qualification. Guard Windows path assertions/optional binaries; pure test fixtures remain portable even when protected live mode is unsupported.

Native package gates remain its Makefile verify/integration/ledger-integration/qualify-taskpacks, exact quality and dry-run evidence. Re-run/broaden only after new changes/failures justify it. Keep old studies frozen. A compiler/runtime change discovered during work requires separate design/conflict analysis and approval.

## Approval and progress state

State schema is populated at `.ailang/state/sprints/sprint_M-MAINTENANCE-EVAL-ADAPTER.json` in this planning checkout, status in_progress and execution_authorized=true. Every milestone has concrete criteria, estimates, dependencies and reuse decision. The core active sprint is untouched. Standard creator imports/reads unrelated messages; the documented schema is written directly, as for the foundation sprint, and validated with the existing validator.

The foundation merge was explicitly authorized and completed. This concrete new cross-repo adapter/containment plan is ready for review under AGENTS.md's `sprint-planner → user says "execute sprint" → sprint-executor → sprint-evaluator` gate. No new implementation, remote dispatch or paid call starts merely because this plan exists. After execution approval, use clean isolated source checkouts, load sprint-executor and complete code/checks before requesting any merge/publication decision.

## Execution checkpoint — 2026-10-23

Seven milestones pass (M1/M2/M4/M6/M7/M9/M10). M3/M5 are partially implemented but process-lifetime/output bounds are unqualified; M8 is blocked by the measured macOS nested-sandbox failure; protected end-to-end M11 and full-delivery M12 therefore remain incomplete. Unchecked boxes above are deliberate. Status stays in_progress with a named blocking condition, not completed or user-paused.

Actual instrument controls passed: native 8 trajectories/24 stages, both failed S1 sources retained into S2,6 successful/2 known-failed trajectories and1 visible-green semantic regression;24 synthetic usage leaves/24 unchecked summaries keep economics unavailable. Package195 tests/500 property cases,25 grading controls, four normal/four malicious protected native cases;14 actual native-tool/OS tests with no skips. One OS test asserts admission CLOSED after nested sandbox_apply failure. These controls are not provider/model/language-quality measurements.

Independent review corrected unsuccessful-executor grading, observed model/harness drift, language-inferred diagnostic phases, transitive oracle identity, frozen executable/version checks, missing counterpart prompts and controller path confinement. No real provider usage, paid call, registry0.2 publication or core release occurred. Strict package quality remains PUB011-only (75 uncontracted exports);3/20 SMT verified,17 unsupported.

Current confinement evidence is the create-only r2 record under core design_docs/validation/maintenance-containment. The original record is historical and does not attest the current wrapper. Follow-up [broker/lifetime design](https://github.com/sunholo-data/ailang/blob/sprint/maintenance-eval-adapter/design_docs/planned/m-maintenance-eval-broker.md) is proposed, not implemented. The approved M8 contingency requires this design rather than silently substituting a new architecture. User design approval and a concrete follow-up sprint are required before that implementation; a priced model campaign remains a separate launch decision.
