# Sprint Plan: M-BYTECODE-STRICT-256-REG-CEILING

Fix large list literals and wide-record else-if chains on the strict VM, and report residual evaluator-only functions before VM dispatch.

**Design:** [approved design](m-bytecode-strict-256-reg-ceiling.md)  
**Status:** Planned; implementation has not started. Coordinator approval of this plan is the executor handoff gate.  
**Target:** v0.53.1 · **Duration:** 4 engineering days (32 hours, including 8 hours contingency) · **Risk:** medium  
**Estimate:** 650 LOC: approximately 350 implementation, 250 tests, 50 documentation/census artifacts. Issue #1741 is linked by the design commit.

## Current state and estimate evidence

The checkout is clean at 49dfe86d on coordinator/task-a54ac096; std/VERSION is v0.53.0. Source inspection confirms all-element allocContig in compileListLit, per-level result allocation in compileIfExpr, and a shared multi-module CompileBytecodeFromResult entry point. The approved design audits every allocContig call site and related bridge/check/test paths, satisfying the systemic-analysis prerequisite.

The seven-day velocity script found only the design commit; this is a shallow repository and its changelog scan did not produce recent measurable implementation velocity. No historical LOC/day is claimed. Capacity is a planning assumption of 162.5 LOC/day, with 24 hours work and 8 hours contingency. The estimate exceeds the design's approximate 450 LOC because it explicitly budgets helper diagnostics, multi-format integration tests, the census and durable examples.

Go is absent in this environment. Installed ailang is v0.52.5, not repository v0.53.0. Planning validation uses source inspection; executor must obtain the repository-supported Go toolchain and build from its checkout before banking runtime acceptance or coverage. No coverage percentage was measured during planning; require no regression in touched packages and exercise each new behavior directly.

## Scope decisions

The detailed Solution Design, Success Criteria and Non-Goals take precedence over the introductory success metric that mentions tuple chunking. This sprint chunks lists only; tuple concatenation, spilling, wider operands, general allocator changes and record-update opcode substitutions are deferred. The strict gate checks every prototype in the compiled image, including imported and unreachable helpers; this intentionally can reject programs whose chosen entry never calls the unsupported function. Check --package integration is deferred unless needed to keep flag handling consistent; an unsupported combination must be rejected explicitly.

The first executor task is the pre-change corpus census. Counts are not invented here. Keep baseline and final reports at design_docs/planned/v0_53_1/m-bytecode-strict-256-reg-ceiling-census.md, with deterministic counts, exclusions, reasons and commands. Use existing compile/disasm facilities and corpus tooling first; add a test-local helper only if those cannot enumerate prototypes. Do not run effectful corpus programs merely to count coverage.

## Registry reuse audit

M1–M5 each have action **none**, package null: these are compiler codegen, Go diagnostic plumbing, CLI/testing integration and repository documentation. There is no package-like capability to import; registry search/info/docs are not applicable. All implementation reuses existing bytecode opcodes, the image compiler, diagnostic formatters and test harnesses. This is an AILANG compiler-fix lane, not a package or VM core-floor change.

## Milestones

### M1: Coverage census and regression baseline (~50 LOC)

**Effort:** 3 hours before contingency. **Dependencies:** None

**Files and examples:** Existing corpus/disasm tools, internal/bytecode/compiler/*_test.go; new census Markdown. Example fixtures: cmd/ailang/testdata/bytecode_pressure/{large_list,record_chain,unsupported_helper}.ail (created during execution).

**Acceptance criteria:**

- [ ] Before changing codegen, census std/, examples/, and goldens/: record files attempted, compile failures, prototype totals, EvalOnly names/reasons/positions, and counts grouped by cause.
- [ ] Classify list-size and chain-pressure failures separately; retain unsupported corpus files as explicit exclusions, never as successful coverage.
- [ ] Capture 601/1000-element list and 15/16/60-arm, 16-field record baselines; audit stderr assertions and large-literal disassembly goldens.

### M2: Structured strict-bytecode coverage reporting (~240 LOC)

**Effort:** 9 hours before contingency. **Dependencies:** M1

**Files and examples:** internal/bytecode/image.go, internal/bytecode/compiler/compiler.go, internal/errors/codes.go, internal/runner/vm.go, internal/testing/bytecode_engine.go, cmd/ailang/{commands_language,check,help}.go and adjacent tests. Inspect existing help routing before editing help.go. Example: unsupported_helper.ail.

**Acceptance criteria:**

- [ ] check --strict-bytecode compiles the complete imported image and returns nonzero with one positioned BC001 per EvalOnly prototype, including unreachable helpers; plain check stays unchanged.
- [ ] Text, --json, and --format agent expose canonical function name, source file/line, and original reason with deterministic ordering.
- [ ] Strict run rejects the image before VM entry dispatch; strict test reports EvalOnly helpers before executing test bodies.
- [ ] Non-strict bytecode run emits stderr warnings with an exact total and a bounded listing; stdout and evaluator bridge results remain unchanged.
- [ ] Existing Rand-mode, entry-failure, test-engine, and bridge parity tests remain passing with intentional diagnostic expectations updated.

### M3: Bounded list-literal chunk lowering (~160 LOC)

**Effort:** 5 hours before contingency. **Dependencies:** M2

**Files and examples:** internal/bytecode/compiler/collections.go and collections_test.go; new CLI pressure parity test; examples/bytecode_large_list.ail.

**Acceptance criteria:**

- [ ] K=64 by default; empty and <=K literal instruction sequences remain unchanged.
- [ ] K-1, K, K+1, 2K, 255, 256, 300, 601, and 1000-element lists compile and match evaluator values and element order under strict VM.
- [ ] Free each chunk block before the next; retain only accumulator/current chunk temporaries; allocator highWater never exceeds 256 on the isolated large-list fixtures.
- [ ] Effectful element-order parity and nested-expression/literal tests pass; no opcode or operand-encoding changes.
- [ ] Large tuples and >255-field records remain outside lowering scope and produce coverage diagnostics when compilation marks them EvalOnly.

### M4: Else-if expression result-register reuse (~110 LOC)

**Effort:** 4 hours before contingency. **Dependencies:** M2

**Files and examples:** internal/bytecode/compiler/control_flow.go and control_flow_test.go; new CLI pressure parity test; examples/bytecode_record_chain.ail.

**Acceptance criteria:**

- [ ] compileIfExprInto reuses the outer destination only through literal else-position IfExpr nodes, preserving register ownership and jump patching.
- [ ] 15, 16, and 60-arm chains over 16-field records compile and match the evaluator for first, middle, final, and default paths.
- [ ] Nested then-branch conditionals, pinned parameter results, and ordinary if expressions retain correct results; statement-form if lowering is unchanged.
- [ ] High-water measurements demonstrate bounded chain pressure as arm count grows; record rebuild and sorted-field layout remain unchanged.

### M5: Corpus closure, examples, documentation and validation (~90 LOC)

**Effort:** 3 hours before contingency. **Dependencies:** M3, M4

**Files and examples:** Both new examples, the census Markdown, docs/docs/reference/cli.md, docs/docs/reference/errors/index.md, cmd/ailang/help.go where applicable, CHANGELOG.md and affected parity assertions.

**Acceptance criteria:**

- [ ] Repeat the M1 census using the same files and methodology: observed failures attributable to the two targeted causes disappear; every residual is named and triaged with a reason.
- [ ] New large-list and record-chain examples pass check --strict-bytecode and evaluator/strict-VM parity; the unsupported fixture demonstrates BC001 and non-strict warnings.
- [ ] Update CLI check help and docs/docs/reference/cli.md, error-code documentation, and CHANGELOG.md to explain the flag, early rejection, warning, limits, and workarounds.
- [ ] make fmt, make lint, make test, and make check-boundaries pass; targeted compiler, CLI, runner and testing parity suites pass with a newly built binary.

## Day-by-day execution

| Day | Work | Exit evidence |
|---|---|---|
| 1 | Toolchain/build preflight; M1 census first (3h); M2 report plumbing and formatter tests (5h) | Baseline counts and failing fixtures banked; report names/positions stable |
| 2 | Complete M2 check/run/test wiring (4h); begin M3 lowering and boundary tests (4h) | Early BC001 gate and stderr-only warnings tested; chunk boundaries correct |
| 3 | Finish M3 (1h); M4 result reuse (4h); M5 census/docs/examples (3h) | Both reported shapes pass strict VM; residual failures enumerated |
| 4 | Eight-hour contingency for integration, effect-order/jump regressions and complete required checks | Final census, parity, lint/test/boundary evidence ready for sprint-evaluator |

M2 lands before either lowering fix. M3 and M4 have independent code paths but run sequentially by default. If lowering overruns, Phase A can be reviewed independently; do not mark the whole sprint complete without both repro classes passing.

## AILANG syntax and example gate

AILANG prompt version loaded: v0.16.6 (ailang prompt and `ailang prompt --version-active` were invoked). This is the installed CLI's active prompt label, separate from binary and repo versions. Reload the checkout binary's prompt during execution before writing .ail fixtures.

| Module | Contracts | Effects | Inline tests |
|---|---|---|---|
| examples/bytecode_large_list.ail | skip: stress fixture correctness is established by Go-driven evaluator/VM parity rather than solver expansion of a 601-element literal | include: pure LUT helper returning [float]; main() -> () ! {IO} prints the measured length | skip: integration harness asserts 601/1000-element lengths and complete values |
| examples/bytecode_record_chain.ail | skip: enumerated branch parity is the relevant register-pressure proof | include: pure applyOverride(record, int) -> record; main() -> () ! {IO} prints selected fields | skip: integration harness covers first/middle/final/default branches |
| cmd/ailang/testdata/bytecode_pressure/unsupported_helper.ail | skip: intentionally unsupported bytecode fixture tests diagnostics | include: pure oversized helper; main() -> () ! {IO} is a dispatch sentinel | skip: Go tests assert BC001 and that strict dispatch never starts |

All generated .ail fixtures follow the same syntax gate. Use ailang check before running, and verify showcase examples with the newly built binary. The effect-order regression fixture declares ! {IO} explicitly and records output order across chunk boundaries; contracts and inline tests are skipped because the CLI harness compares output directly.

## Validation, risks and handoff

Run targeted go tests for internal/bytecode/compiler, internal/runner, internal/testing, internal/errors and cmd/ailang while developing, then make fmt, make lint, make test and make check-boundaries once at closure. Check supported coverage tooling before recording a package coverage comparison. Preserve small-literal goldens; only intentional large-literal disassembly and diagnostic expectations change. Existing strict Rand-mode and entry-failure assertions may need BC001 expectations, while their runtime semantics must remain intact.

Keep accumulator/current chunk registers alive until CONCAT consumes them, and never free the reused if destination in a child branch. Chunk folds copy accumulated lists and can cost O(n²/K); no performance improvement is promised. Large caller live sets may still exhaust registers, in which case the strict gate must report the original reason rather than mask it. Explicitly test operand counts around 255/256 so uint8 truncation is never treated as successful compilation; any independent encoding bug discovered goes to a named follow-up or scope review.

Prefer additive prototype source-line metadata and a shared, sorted report; compiler metadata must not import runner/CLI packages. Warnings list at most ten names and always include the exact total. The corpus's imported modules must be deduplicated when counting prototypes; record policy and denominator in both reports. No new VM opcode, parser/typechecker or stdlib semantics change is authorized.

Sprint JSON is not_started with every passes field null. The coordinator consumes the artifact markers and presents this concrete plan for approval; merging/approving the plan triggers sprint-executor through the existing coordinator pipeline. No separate agent dispatch is needed from this planning stage. Planning validation: JSON syntax, milestone IDs, dependencies, reuse rows, LOC sum and duration passed equivalent Python checks; the repository shell validator could not run because jq is absent (its error labels this as invalid JSON). git diff --check passed. Executor then hands the completed implementation to sprint-evaluator.
