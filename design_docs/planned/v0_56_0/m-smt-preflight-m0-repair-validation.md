# M-SMT-PREFLIGHT-M0-REPAIR implementation evidence

**Release target:** v0.56.0, UNRELEASED. **M0 accepted:** root round 3 PASS 95 at 4fb; actual 27 CI: 20 success / 7 skipped and actual Sonar 84.4% pass. Normal merge 0db7902 at 22:19:32 UTC. This is independent acceptance, not an executor self-score.


[Durable final and historical reports/raw identities](../../implemented/v0_56_0/evidence/m-smt-preflight-m0-repair/README.md)
include lossless 8.63 MB corpus, both FAIL 73 rounds, final PASS 95 and actual Sonar/CI.
The parent/plan/validation remain planned for the multiphase exception; M1–M4,
F2/F5/#1950/#1953 are deferred. The frozen review block/external BLOCKED records
are not changed. Earlier pending/source-60a7d7b observations below are historical;
accepted 4fb has unchanged production Go/all AIL plus a 69-line test-only repair.

The user approved the scoped design/evidence on 2026-10-29 (“approve the three designs”) and explicitly approved execution at observed 20:17:42UTC (“exeute all three sprints”). Both external quorum rounds remain BLOCKED. Their frozen packets/reviewer records were not rewritten or resent. M0-F1/F4/Z are ruled; the parent roadmap M1–M4, F2/F5, #1950/#1953, #1790, runner work and held telemetry remain excluded.

[Approved proposal](../m-smt-well-sorted-preflight.md) · [sprint plan](sprint-plan-m-smt-preflight-m0-repair.md) · [complete measured rows/identities](m-smt-preflight-m0-repair-validation.json)

## Exact source and proof identities

- Execution base: `6d0d3a9da2af2eadc504f7cb050aacde72cc3cca`.
- Historical local compiler/test source: `60a7d7b5fce9c137835774750cac1cca063e6b00`. The later accepted 4fb head preserves production Go/all AIL bytes but adds the 69-line SMT behavior regression; final CI is qualified to that accepted head. The original local evidence below remains pinned to this checkpoint.
- Worktree: `.claude/worktrees/smt-m0-repair-exec`, branch `fix/smt-m0-soundness`. Parked `p0/smt-unsound-encodings` and its dirty artifacts were not modified or built.
- Actual current baseline binary: `/tmp/ailang-backlog-20261029/m0-current-base`; fixed binary: `/tmp/ailang-backlog-20261029/m0-final-match-repaired`. Full binary SHA256, version/std tree, exact fixture SHA256 and environment qualification are in the companion JSON.
- Historical archived807 execution remains historical. Named-path equality and its nonempty cache/front-half differences are preserved; no whole-compiler equivalence claim replaces the new actual baseline/fixed builds.

All new `.ail` fixtures followed the actual baseline compiler's loaded **v0.16.11** prompt. They are pure verifier regressions with intentional true/false contracts; standalone inline tests/showcase effect annotations are not their oracle. Commands clear ambient Google/OTEL/OTLP tracing/export inputs, use matching stdlib and bounded local Z3. No provider review or external consumer/source-pin change occurred.

## Implementation checkpoints

| Checkpoint | Commit | Measured result |
|---|---|---|
| M1 | `82ddb7030` | Nine Int emitter shapes, signed runtime parity, nested and explicit `$ail_n`/`$ail_d` outer-binding collision controls |
| M2 | `fd6140dc0` | Typed UF actual-argument identity, finite guarded specifications, exact arity and unknown-requires fail-weak controls |
| M3 | `9eb2bb655` | Let/shadow/selected-If facts; unselected contradictory calls remain refutable |
| M4 | `a268d58da` | Pattern/guard reach, predicate-local substitution and fresh outer aliases for actual values/results |
| M5 | `ac82ae5a1`, repaired `d3666accf` | Paired helper parameter/environment parity and single declarations; nested bound freshness reopened and repaired after genuine RED |
| M6 | `50c1b77d5`, `427c49a5f`, `60a7d7b5f` | Actual CLI/runtime/mutations/corpus, ordinary unit-erasure preservation, first-match eligibility restoration |

The M5 nested-bound fault was not accepted as complete: a local-let source equality falsely VERIFIED on the intermediate binary, while its true inequality refuted. Core and actual CLI RED records are retained. Reserving enclosing lexical rename values restores COUNTEREXAMPLE/VERIFIED respectively. A parameter-only probe did not trigger because root `$p_` renames protected it; that distinction is retained.

The M4 fact prepass initially visited a later unreachable opaque callee: baseline VERIFIED became SKIPPED. Final source truncates at the same structural first-match boundary as the value encoder and retains its primitive ite route. Core and actual CLI now preserve VERIFIED. The initial failed repair and both RED records remain in scratch; they are not final acceptance evidence.

## Acceptance measured on actual binaries

- **A1/A2:** all seven modules check on both actual compilers. Baseline parity/halfDown/gap falsely VERIFIED; fixed rows refute. Negative arithmetic models and nonnegative/toward-zero/remainder-sign controls are asserted. Twelve signed pairs per operator match interpreter and strict VM exactly. Nine Core producer forms and generated-name collisions pass.
- **A3:** same-call identity proves, distinct-call equality refutes; satisfying/violating requires are separated. Unencodable/recursive requires contribute no leaked ensures and terminate conservatively. Unit erasure preserves nullary contract callers. The callee's own bounded unit-recursion ERROR is inherited and explicitly retained, not repaired by widening unrolling.
- **A4:** actual fixed `partial`/`wrong` is COUNTEREXAMPLE; both runtime engines return 0 at the nonzero witness and opt-in runtime contracts fail the false ensures. Its independent bounded recursive row and aggregate exit1 remain visible.
- **A5:** lexical/shadow/capture, bounded quantifier scopes, match-bound arguments, body/requires/ensures short-circuit and unselected helper contradictions pass. Interleaved user-helper arguments and diamond declarations retain distinct explicit environments. Value/condition helpers use the exact same parameter list, including explicit/already-lifted Core environment parameters. Arbitrary front-end closure lifting is neither newly supported nor claimed.
- **A6:** `/0` and `%0` check and raise RT001 in both runtime engines. Zero-only/unselected-zero verification retains M0-Z totalization; no nonzero assumption, totality/runtime-error-freedom claim or new status rule was added.
- **A7:** all three clean final-source Go-overlay mutations build, falsely VERIFY their target, and fail the focused meaningful Go test (exit1). Full corpus comparison below accounts for all changed verdicts and inherited failures.
- **A8 local:** the justified final-source changed/core rerun completed with exit0, as did focused repairs, lint0, sizes, boundaries/changelog, runnable examples/manifest and stdlib interfaces. Required PR CI and independent acceptance remain pending.

## Corpus accounting

Selection is exactly **530** tracked execution-base examples/std `.ail` files plus **521** `.ail` files from package archive `8e6bbafa3b12335c06bb3dab8896fea8cd74886f`. Every selected file's bytes independently match its pinned Git blob. No exports-only package selection or private/test omission was substituted. Each arm has 1,051 file calls and 3,036 function rows.

| Result | Actual base | Final fixed |
|---|---:|---:|
| VERIFIED |306|307|
| COUNTEREXAMPLE |27|27|
| ERROR |40|39|
| SKIPPED |627|627|
| Uncontracted |2035|2035|
| UNKNOWN (solver timeout) |1|1|
| Exit0 / exit1 files |921 /130|922 /129|

The only verdict/exit change is `packages/discord/client.ail:validEmbed`: ERROR (callee postconditions referenced free `e`) → VERIFIED. The corrected encoding uses the actual Embed argument in its typed UF and scoped facts; the caller's body/postcondition share the appropriate argument-dependent terms. The full old/new SMT and rows are retained. No callee body/trust policy was promoted into a proof.

All 68 identities with ERROR/COUNTEREXAMPLE/UNKNOWN on either arm are enumerated in the companion JSON. `packages/explainer/words.ail:kept` times out per function on both arms at the fixed 1s budget. There are no whole-file process timeouts. The 157 diagnostic-only changes retain status/reason_code: 130 existing unsupported-encoding rows and 27 inherited verifier errors. Context wrappers, nondeterministic first-callee/declaration order and existing Float/transcendental faults account for these differences; the full reason pairs remain explicit. They are not repaired or relabelled as passing.

Initially-empty separate per-arm compile caches reuse imported Core only; SMT runs for every function. Baseline rows are reused unchanged for the justified final fixed-arm rerun after the real first-match repair. Initial absolute-path MOD010 probes were invalid harness runs, not mutation kills. A null-results summary formatting error was corrected by postprocessing complete raw rows without re-running compiler calls.

## Mutation and local gate records

Clean overlays preserve source checkouts and are independently hashed. Euclidean mutation makes `parity` VERIFIED; shared-result mutation makes `gap` VERIFIED; universal-axiom mutation makes returning `wrong` VERIFIED. Focused tests respectively fail `TestIntDivModEncoding_MatchesGoTruncation`, `TestFiniteFacts_ActualCallIdentity`, and `TestContractCallIdentityAndGuard`. Initial ambiguous/invalid probes remain historical and do not count as these kills.

- Runnable oracle: 265 pass, 0 fail, 10 existing documented skips (275 selected); manifest validation passes.
- All 49 frozen stdlib interfaces remain stable; no goldens changed.
- Existing #602 solver fixture warm-up, effect/explicit-empty/alias/IFCv2-v8 source, prompt bytes, runtime and parser/type policy remain preserved. No full executor `make test` was run.

Raw directory: `/tmp/ailang-backlog-20261029/m0-execution`. Load-bearing files: `final-cli-{identity,raw}.json`, `corpus-source-manifest.json`, `corpus-final-{raw,summary}.json`, `mutation-final60-results.json`, `test-changed-final60.log`, `structure-final60.log`, `verify-examples.log`, `manifest.log`, `verify-stdlib.log`. The companion JSON binds stable raw artifacts by bytes and SHA256. Final CI and independent evaluation must actually pass before acceptance/merge.

## Test-only coverage follow-up after the cb5 CI gate

The actual cb5 Sonar gate was FAIL solely for new-code coverage77.7% against
unchanged 80% policy:385 new lines,86 uncovered. Exact-PR source-line hit data
identified uncovered contract Match/Forall substitution and local result-binding
paths. CI run40957956051 exposed no coverage artifact, so the actual Sonar
per-line response was retained instead of inventing an artifact measurement.

`codegen_contract_behavior_coverage_test.go` adds three behavior tests using real
Z3 VERIFIED/COUNTEREXAMPLE pairs: pattern-bound parameter/result shadowing,
bounded quantified postconditions with actual arguments and result binders,
synthesized unit erasure, unknown postconditions and local result bindings.
These tests assert logical entailment/non-entailment, not emitted text. The first
new fixture used an unsupported bare primitive-variable Core match and produced
an honest Z3 datatype-match error. The retained correction uses the existing
literal-arm primitive route; no production encoding or outcome was changed.

Focused full SMT package PASS11.031s (72.6% whole-package statement coverage),
targeted lint0. The local profile hits26 previously uncovered Sonar new-code
lines, projecting84.42% new coverage. That is an explicitly labelled projection;
only the actual new-head Sonar run can establish its gate. Raw profile, original
failure, actual Sonar line data and projection are retained under
`/tmp/ailang-backlog-20261029/m0-coverage-repair/`.

This follow-up changes only tests and this validation note. All production Go
and all .ail bytes are identical to cb5/source60, so prior exact compiler corpus,
mutation and CLI/core evidence remain applicable without duplicate execution.
No production/gate/threshold/exclusion/workflow or semantic scope was changed.
Historical independent FAIL and CI-retry records remain intact; root owns the
formal verdict and actual gate acceptance.
