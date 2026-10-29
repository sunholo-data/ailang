# M-IFC-FUNCTION-VALUE-SINKS validation

Execution approved by Mark, 2026-10-29: “Execute sprint #1719”. Independent evaluation and parent merge remain separate. No self-score is assigned.

## Provenance and scope

Initial dev base `1b70d59f16b1c1ddde93aaaa9dec7e2d9c8a1b14`. Historical local branch `p0/ifc-function-value-sinks`, HEAD `37a8e8f4f`, contains committed plan/source/tests/examples/docs from 2026-10-26. Parent explicitly authorized recovery of committed work under the newly approved six-milestone sprint. Recovered compiler code/tests from `3dd7b89fd`, REPL test from `110093640`, examples from `96d32ff1a`, guide from `37a8e8f4f`. No historical dirty/untracked code was imported. The parked checkout remains untouched; its tracked patch and untracked `ifc_demand_scan.go` are banked separately under `/tmp/ailang-backlog-20261029/1719-parked-*`.

All planes/open PRs were checked at start: prod 1,030 tasks, dev 956, test 5,000 (limit reached, not exhaustive), plus local history/worktrees. No active #1719 implementation PR/task was found. The historical local work is the recovered source, not a competing carrier.

Compiler M1–M4 were recovered as one cohesive implementation and repaired in `3484c6789`; M5 CLI/REPL/carrier checks banked in `cadeb80ce`. Wrapper inference ships with callable propagation; no partial M1 release or issue closure is authorized. No HM/CoreTI/parser/codegen/VM/iface schema or #1958/#1963 production file was changed.

## Failing-first and cutoff audit

- Original dev fails the recovered R/C matrix and wrapper/HOF deeper-shape tests while direct/clean controls remain valid (`1719-matrix-red.log`).
- Historical committed code passed the original matrix but silently accepted result-probe depths 4/12/40 (`1719-probe-depth-red.log`) and lost a written function type through 40 aliases (`1719-m1-algebra-red.log`). These are repaired, not accepted as limits.
- Remove probe depth 3, replay depth 32 and the bounded summary-round exit. Aliases use visited identities; summaries reach equality; replay recognizes repeated owner/argument states. Depth-64 replay and a cycle with a reachable sink are pinned.
- Defensive depth 1,000/work 1,000,000 limits produce ordinary `ifc_analysis_limit` diagnostics. Typed recovery catches only that limit; unrelated programming panics rethrow. No partial summary grants permission to execute.
- Preserve the original non-silent concrete parameter environment and label/Check B walk. Symbolic summaries are built only when a conservative AST scan finds a written IFC label/refinement (unknown AST forms enable analysis). This avoids constructing irrelevant graphs in ordinary code such as std/fs. The initial unbounded std/fs experiment was stopped; the final raw corpus completes in about one second with identical diagnostics.

The callable domain represents joins as a union of alternatives. Forbidden labels are canonical sorted sets. Positive cover conjunction implements the ruled meet: nil is absent, an explicit bottom cover admits only bottom. Algebra, strictness, absent/bottom and direct diagnostic tests were repeated 20 times.

## Regression and mutation evidence

The design prose says 25 files but lists 24 named rows (R0–R14, C0–C2, N1–N6). Every row is retained, plus X1 multi-forbid wrapper and R11b guarded HOF parameter receiving an unrestricted callback: 26 CLI/unit cases. Direct messages and positions stay exactly `fv.ail:5:36` (sink) and `fv.ail:5:37` (positive cover), measured independently on the baseline.

Fresh-file CLI tests disable cache and use unique paths. New failures require the proper IFC message class; clean controls check successfully. REPL tests reject alias/HOF/wrapper-lambda leaks and accept clean alias/HOF flow. Both evaluator and bytecode refuse alias/HOF/wrapper programs before the `IFC1719_UNSAFE` print. Additional tests cover record updates, arrays, ADT pattern variables, block letrec, recursion/source order, no labelVar escape and clean deep callable chains.

Mutations use Go's `-overlay` with changed source copies under scratch, never tracked file checkout/revert:

| Mechanism removed | Regression failures |
|---|---|
| Concrete application demand check | R1, R3–R9, C1 all become wrongly accepted |
| Applied-at probe recording | R2, R10, C2 become wrongly accepted |
| Inferred demand recording | R12, R13, R11b become wrongly accepted |

Original R11 is also protected by applied-at because its actual callback is `gate`; R11b removes that redundancy and directly proves the written function-type requirement. Each mechanism subset fails for a missing IFC rejection, not a build/parser error. Restored focused/CLI/REPL and changed/core tests are green.

## Exact-source IFC corpus

Companion `m-ifc-function-value-sinks-corpus.json` records each root's repository SHA, file count, parser failures and SHA256 of the sorted path/content-hash index. Each arm analyzed the **same frozen bytes** through `CheckModuleIFC`, so unrelated HM errors cannot mask an IFC change. Source repositories with local edits are explicitly flagged; per-file hashes, not a clean-tree claim, establish identity.

| root | source | recorded HEAD | files | parser failures |
|---|---|---|---:|---:|
| 0 | examples | cd0f99024780 (plan-only over initial base) | 474 | 7 |
| 1 | std | cd0f99024780 | 51 | 0 |
| 2 | benchmarks | cd0f99024780 | 41 | 0 |
| 3 | ailang-packages | e998928e3dba | 163 | 0 |
| 4 | ailang-motoko | 91eb860a233f | 691 | 10 |
| 5 | ailang-demos | 0a93ddcb664e | 290 | 1 |
| 6 | ailang-parse | 0931946850e2 | 133 | 0 |
| 7 | ailang-world | df5a0671dadd | 39 | 1 |
| 8 | stapledons-godot | b07c976356b5 | 162 | 0 |
| 9 | daneel | e5d751e21caa | 80 | 0 |
| 10 | mk-20261027 | 835c738b8ca5 | 501 | 4 |

2,625 source files; 23 recorded parser failures (not counted clean); seven pre-existing IFC diagnostics on each arm; **zero diagnostic/file changes**. Consequently there are zero new unannotated-module errors and no new corpus over-approximation requiring a ruling. Historical `.agent`/`.motoko`/`.claude` evidence and node_modules are excluded; tracked source modules are included. Raw logs/source snapshots remain in the local evidence bank.

Opt-in harness: registered `AILANG_IFC_CORPUS` path-list; unset explicitly skips, missing/empty roots fail. Run the exact test on each source tree using the same roots:

```sh
AILANG_IFC_CORPUS=/path/to/root0:/path/to/root1 go test ./internal/types -run '^TestIFCFunctionValueCorpusDelta$' -count=1 -v
```

## Required CLI consumer comparisons

18 old/new CLI checks: all ten pinned repo fixtures keep their expected exit codes (including expected-fail `leak_attempt`); gmail package checks 0→0. The seven leak_lab examples initially hit their existing shared module-name mismatch, so a second explicit `AILANG_RELAX_MODULES=1` run reaches the intended IFC gate on both arms: authority/redact remain accepted, direct/helper/record/relabel/url remain rejected. **Zero status flips and zero first-diagnostic changes.** Strict results are retained separately, not used as IFC evidence.

CLI arms disable cache, use matching stdlib roots and the same source paths. No source consumer was modified and no feedback/message was sent.

## Examples, teaching and counterpart handoff

Syntax reference loaded before edits: teaching prompt v0.16.8. Expected-fail example cannot run contracts/inline tests by definition; its three violations are pinned in `TestSecretExamples_IFC`. Accepted twin declares IO/Declassify and has pinned stdout; contracts/inline rows are skipped because the showcased entry performs those effects and integration tests verify behavior. Manifest explicitly records expected failure; the generic example runner skips it using the existing expected-fail convention, without marking it broken.

The guide removes the local unannotated-wrapper limitation and keeps imported sink/#1134 limits. The new immutable prompt version must inherit reserved #1963 v0.16.9 after it lands; previous version bytes remain untouched. Prompt number/hash and final combined head are recorded at finalization below.

#1771 follows this PR and must rebase its imported-name branch into the single `identVal`/callable-value lookup. Its v2 `IFCParam.Forbidden` is sorted `[]string`; reserve `applied` as `IFCLabel` per the existing F4 ruling/request. This sprint introduces no v3 or serialization change and makes no claim to close imported-HOF #1134.

## Gate status

Initial baseline types/repl/pipeline and lint pass. Final changed packages/core, focused IFC/CLI/REPL, lint (0 issues), file sizes (<800 production lines), examples/manifest, boundaries and changelog checks are banked under `/tmp/ailang-backlog-20261029/1719-*.log`. Final combined-tree/prompt status is added before completion. The executor does not duplicate a full make test; PR CI and the independent evaluator own that gate.

## Final integration and independent review repair

Integrated actual dev `87f846980c8b6fc601173be427b1364968559488` (#1963) in `c11206719`, preserving all latent-effect fixes and prompt9 bytes. Manifest conflict retained both PRs' additions:250 total/242 working.

Root's independent source review found `isCtorName` incorrectly treating unknown capitalized imported functions as constructors. Its full CLI `Apply` repro executed and returned TAINT. Repair `d81430352` removes the capitalization guess: only actual local constructor declarations receive storage semantics; unknown imported names use the ruled conservative F2 fallback. No iface/HM/pipeline API expansion. Failing-first raw test is banked; full two-module CLI checks and both engine selections now reject Apply/apply before returning taint, while imported Some(gate) storage remains accepted. This repair also passed the final combined corpus; no unexpected constructor over-approximation was observed.

Combined raw corpus:2,628 snapshots including both new IFC examples and the newly integrated latent example. Baseline7/fixed10 diagnostics; the sole changed file is the intentionally added expected-fail example with its three actual alias/HOF/wrapper leaks. All other2,627 files have identical IFC diagnostics;23 parser failures remain recorded. Companion JSON now records the final combined roots/hash indexes and classifies this intentional change. The repeated18CLI consumer checks remain zero status/first-diagnostic flips with all pinned expected exits satisfied.

Teaching prompt **v0.16.10**, SHA256 `a9435c79d78002c3c0b51e36bff06b5f918f7e3a90198dd2301e493c0ff9e001`, inherits v0.16.9 exactly except version header and one IFC sink sentence. v0.16.9 remains byte-identical, SHA256 `6197fe5f4bf1c112737c550e08d557eea5c92db9b85839efaac8d46accd28be1`. Source/embedded/versioned/current docs and three llms mirrors use existing generators; freeze check reports63 registry entries. Active CLI serves v0.16.10. No code block/optimization was changed; the new sentence is exercised by the complete negative/clean test matrix. Referenced historical prompt-accuracy/code-block scripts were absent (searched); no nonexistent gate is claimed.

Final gates PASS:make test-changed (changed packages + all core); focused IFC/CLI/REPL; prompt/freeze/registry tests in internal/prompt, internal/eval_harness and cmd/ailang; lint0issues; file sizes; verify-examples/manifest245 modules/0drift/0missing; boundaries; changelog; diff check. Raw combined corpus and18CLI consumer comparisons are banked. Full suite belongs to exact-head PR CI/independent evaluator; execution does not self-score or move docs to implemented. Completion archival remains the parent's separate post-PASS audit.
