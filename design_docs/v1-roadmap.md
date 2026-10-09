# V1 roadmap: the groomed path to v1.0.0

**Groomed**: 2026-10-09, attended with Mark (Claude Opus 5.5). **Status**: proposal. The V1 loop is
paused. Nothing here changes what it picks until the charter points its queue at this file (the
"bar v3" charter refactor, a separate step).

**Source of truth for the bar**: [v1-mission.md](v1-mission.md) § "The v1.0 bar". This file is the
ordered work list against it: one line per item, nothing landed, no history. History stays in the
charter and its log.

## The claim

**AILANG orchestrates AI agents as typed, verified packages** (Mark, 2026-10-09: the package system
plus the message system *is* the orchestration framework):
- every agent acts only through type-checked programs admitted against your policy;
- packages publish proved interfaces;
- the messages between them are typed, budgeted effects;
- any widening of authority stops for a human.

The flagship is a two-package orchestration whose CI proves each clause with a mutant that must fail.
See [m-v1-orchestration-flagship](planned/v1_0_0/m-v1-orchestration-flagship.md). The exact bar
sentence is ratified per clause at the release gate, after that clause's prerequisite lands (verify,
then freeze).

## Distance: N = 22 design docs remaining (was 12)

Counted by the `D-51` unit: design docs serving an open clause, one count per doc.

| Change | Δ | Reason |
|---|---|---|
| Slim prompt (`m-eval-slim-prompt-self-discovery`) | −1 | RULED OUT 2026-09-02 (82% → 65%) |
| `m-run-selector-enumeration-floor` → fleet | −1 | Test-harness hygiene (proposal; Mark rules) |
| Open P0s never counted | +7 | The charter says "zero P0s ✅"; 10 are open. 7 docs cover the 8 language/lane P0s (#981 and #1597 excluded) |
| #1720, confined `ailang lock` still clones | +1 | A fourth `run --policy` escape, filed 2026-10-09; no doc yet |
| `m-trace-label-aware` | +1 | Replay reads traces; traces must not hold secrets |
| `m-typed-message-plane` | +1 | Typed edges (new 2026-10-09). Owns M-CLI-MSG-HANDLER as its M2, so that doc is not counted separately |
| `m-json-codecs` | +1 | Typed codecs for model output and messages (new 2026-10-09) |
| `m-ifc-ai-and-io-sinks` | +1 | Secrets must not reach prompts or output (to write) |

**12 − 2 + 7 + 1 + 1 + 1 + 1 + 1 = 22.** The count rises because the plan now covers what the claim
needs, not because progress went backwards. Most clause-2 items already have a design and a sprint
plan, so **the bottleneck is approval and execution**.

## Ordered work list (the loop picks top-down; ⏸ = waiting on Mark)

| # | Clause | Item | State | Next action |
|---|---|---|---|---|
| 1 | 2 | P0 #1326 + #573: latent effects through function values ([m-effect-latent-function-values](planned/v0_48_0/m-effect-latent-function-values.md)) | Design ✅, plan ✅, **executor PR #1708 open** | ⏸ approve #1708, then judge and land |
| 2 | 2 | P0 #616: effect-row variable unification ([doc](planned/v1_0_0/m-effect-row-var-unification.md)) | Design ✅, **plan PR #1678 open** | ⏸ approve #1678, then execute |
| 3 | 2 / 4 | P0 #1548: a `run --policy` program can forge `policy-result:` | Triage note only (`ailang-core-triage/run-policy-result-line-forgeable.md`) | Design doc, then sprint |
| 4 | 2 / 4 | P0 #1569: `fs_deny_write` bypassed by renaming a parent directory ([doc](planned/v0_52_6/m-fs-deny-write-dir-rename.md)) | Design ✅, plan ✅ | Execute |
| 5 | 2 / 4 | P0 #1607: `policy-tool` `pkg-docs` writes outside policy ([doc](planned/v0_52_6/m-pkg-registry-confinement.md)) | Design ✅, plan ✅ | Execute |
| 6 | 2 | P0 #1443: `pure func` accepts a declared effect row | Triage note only (`ailang-core-triage/pure-keyword-vs-declared-row-and-iface-purity.md`) | Design doc (may fold into #1) |
| 7 | 2 | P0 #752: `Declassify` is whole-body authority ([m-ifc-authority-scoping](planned/m-ifc-authority-scoping.md)) | Design ✅, plan ✅ (v0_53_0) | Execute |
| 8 | 2 | P0 #1134: IFC labels lost across modules ([m-ifc-cross-module-labels](planned/m-ifc-cross-module-labels.md)) | Design ✅ | Plan, then execute |
| 9 | 2 | [m-bytecode-vm-parity-bugs](planned/v1_0_0/m-bytecode-vm-parity-bugs.md) | Design ✅ | Plan, then execute |
| 10 | 4 | [m-json-codecs](planned/v1_0_0/m-json-codecs.md) | **New 2026-10-09**, quorum r2 dispositioned | ⏸ F1–F3, then plan; small and unblocks 11 and 12 |
| 11 | 4 | [m-typed-message-plane](planned/v1_0_0/m-typed-message-plane.md) (includes M-CLI-MSG-HANDLER as M2) | **New 2026-10-09**, quorum r2 dispositioned | ⏸ F1–F4; M0 (shadow field) first; nothing enforced without a clean window |
| 12 | 4 | [m-v1-orchestration-flagship](planned/v1_0_0/m-v1-orchestration-flagship.md) | **New 2026-10-09** (r6), quorum r2 dispositioned | ⏸ F1–F4; M1–M3 can start now; M4 needs P1–P5 |
| 13 | 2 / 4 | `m-ifc-ai-and-io-sinks`: prompts and output as secret sinks | **To write** | Design doc (a language-semantics change) |
| 14 | 4 | [m-trace-label-aware](planned/v0_36_0/m-trace-label-aware.md) | Design ✅ | Plan; must land before flagship M1 (replay) |
| 15 | 4 | [m-effect-clock-net-fs-modes](planned/v1_0_0/m-effect-clock-net-fs-modes.md) (effect sprint 3) | Design ✅ | Plan, then execute |
| 15a | 4 | [m-agent-step-cancellation](planned/v0_29_0/m-agent-step-cancellation.md) (#231) | Design ✅ | Revise to absorb `ailang-core-triage/ai-cancellable-provider-context.md` |
| 15b | 4 | [m-serve-api-live-tool-registry](planned/v0_29_0/m-serve-api-live-tool-registry.md) | Conflicts with the no-`listChanged` Cloud Run decision | Re-scope to opt-in/local-only, or ⏸ rule OUT |
| 15c | 2 / 4 | #1720 confined `ailang lock` clones | Issue only | Design doc, then sprint |
| 16 | 3 | [m-teaching-prompt-v1-cut](planned/v1_0_0/m-teaching-prompt-v1-cut.md) (R1.2) | **New 2026-10-09** | ⏸ F1–F3, then inventory and A/A floor |
| 17 | 5 | [m-contract-verification-coverage](planned/m-contract-verification-coverage.md) | Design ✅, idle since 08-23 | Plan |
| 18 | 5 | [m-verify-bounded-unrolling-false-counterexample](planned/m-verify-bounded-unrolling-false-counterexample.md) | Design ✅, idle since 08-23 | Plan |
| 19 | 5 | [m-cohort-manifest-build-provenance](planned/m-cohort-manifest-build-provenance.md) | Design, **quorum round 4 BLOCKED** | ⏸ rule: re-scope, or OUT of the bar |

Ordering rationale: rows 1–2 are already built or planned and wait only on approval, which is the
cheapest progress available. Rows 3–5 are the gate's own escapes, so the flagship claim cannot ship
while they are open. The flagship's two small enablers come before its large milestones.

## OUT of the v1 bar (normal road or another owner)

| Item | Where it goes | Why |
|---|---|---|
| `m-motoko-ailang-only-lane` | Motoko mission track | Not flagship-ready (no prod runs, no licence, ties pi at higher cost); cited as production use once its lane is in prod |
| P0 #1597 serve-api `--static` security headers | Normal road, fix soon | Product security for the docparse deployment, not a language claim |
| P0 #981 gate-0 directive watermark | Fleet | Mission harness |
| `m-run-selector-enumeration-floor` | Fleet | Test-harness hygiene |
| `m-effect-scope-params` | v1.1 | `D-27` |
| Typed inter-agent messaging (`std/agent`) | v1.1 | M-AGENT-ORCHESTRATION; the message plane is deployment infrastructure |

## Focus rules for the V1 loop (proposed for bar v3)

1. **Product work only.** A harness defect becomes a fleet ticket and never a V1 iteration.
2. **Execute before designing.** No new design doc for a clause while one of its rows has an
   approved design or plan that has not been executed.
3. **Every iteration reports `N` and the row it moved.** An iteration that moves no row says why.
4. **A `v1` label and milestone tag each row's issues and PRs**, so a weekly count of merged PRs
   that advanced a row can show drift.
