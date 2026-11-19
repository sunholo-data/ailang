# M-FLEET-ROLE-ARTIFACT-CONTRACT

**Status**: Planned — **revision 2 after quorum rc 3 BLOCKED; awaiting (a) Duplicate/Coverage gate ruling (UNRESOLVED, below), (b) ONE controller-run re-quorum, (c) explicit attended human design approval.** NOT approved; nothing proceeds to planning/implementation (terminal STOP below). Revised by the iteration-96 rotation designer (`pi:ollama/kimi-k3:cloud`; resolver returned `declared:provider-pin` — DECLARED/RESOLVED label, not observed metadata). Blocking record: `design_docs/fleet-mission-evidence/iteration96/quorum-r1.json` (3 present rejects, 1 absent/quota, $0.1508914) + `…/quorum-r1-controller-proofs.json`; r1 draft preserved at `…/production-designer-draft-r1.md`.
**Target**: Unversioned (loop-harness skill prose; stays in `design_docs/planned/` root)
**Priority**: P1 (implements ONLY evaluator-report item P1, recipe instruction; item P2 fully deferred below)
**Estimated**: ~1.5–2 h, one phase
**Dependencies**: None blocking. Source ticket `tool-hang:triage-judge:home-metadata-search`; disposition `design_docs/fleet-mission-evidence/iteration96/evaluator-report.md` §§4–6.
**Planner-Lane**: opus-required

Bare value required (`derive-planner-lane.sh` Step 2; r1's inline prose value failed closed as `planner-lane-field-invalid`, S-proof R12). Why opus: changed paths outside the default allowlist (`mission-control.sh:1439`) and `missions/fleet.toml` does not widen it (R12). Post-rewrite derive, inherited env, verbatim stdout rc 0: `codex:gpt-6.1-sol anthropic-fallback:declared:opus-required` — declared lane opus-required; the string is the driver's configured Anthropic-unavailable **fallback**, not a doc lane change. Widening the fleet allowlist is attended; not proposed.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms) (`docs/docs/references/axioms.mdx`)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | 0 | Recipe prose only; no runner verdict/threshold/retry/lane change. |
| A2 Replayability | 0 | No trace/NDJSON format or capture change. |
| A3 Effect Legibility | +1 | Read/produce surface becomes declared exact paths, not discovery. |
| A4 Explicit Authority | +1 | Narrows ambient license (identity research, `$HOME`/session sweeps); grants nothing. |
| A5 Bounded Verification | 0 | No new machinery; existing parity suite + bounded review. |
| A6 Safe Concurrency | 0 | — |
| A7 Machines First | +1 | Directives self-describing: named paths, worktree+SHA, provenance-labeled identity. |
| A8 Minimal Syntax | 0 | — |
| A9 Cost Visibility | 0 | — |
| A10 Composability | 0 | Existing sync and rc18 rule reused unmodified. |
| A11 Structured Failure | +1 | Missing named INPUT = typed report against its path; not empty-but-successful; distinct from expected-absent OUTPUT. |
| A12 System Boundary | 0 | Sandbox/fences/extensions untouched. |

**Net Score: +4** → **Move forward** (≥ +2). **Hard check:** A1 no implicit nondeterminism (no code paths); A3 no hidden side effects (writes only the two recipe files via existing sync); A4 removes ambient license, grants none; A7 contract is machine-consumable directive structure. No −1 on A1/A3/A4/A7.

## Problem Statement

An unattended pi **role** run lost a whole slot to a broad `$HOME` metadata sweep that every standing rule permitted (own reads this session, R1–R4):

- **Triage-33 first judge attempt**: `tool_hang`, rc 18, 1119 s, 148 tool executions, worktree unchanged, no commits. Open tool (typed verdict, `$HOME` unexpanded): `printf 'HOME=%s\n' "$HOME"; ls -la "$HOME/.pi" …; find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin -120 …`.
- **What the original directive did/didn't do** (sanitized excerpts, controller-recorded hashes, R4): it **DID name inputs and output** (inputs: `…/judge-input/manifest.json, proposed-items.json, auxiliary-compact.json and charter.md`; output: "The ONLY write you may make is … JUDGE_REPORT.json"). **R1's absent-input premise is retracted.** The observed defect is narrower: (a) it **invited identity research** — "Return actual provider/model from your session metadata if available, not a guessed identity."; (b) nothing prohibited broad discovery outside the named worktree.
- **Recovery = N=1 correlation, not causal proof.** ONE same-lane retry, differently instructed on several variables at once (identity supplied verbatim, explicit ban on HOME scans/hidden-store/session lookup/environment inspection/global discovery, no-previous-artifacts), returned rc 0 in 274 s (75 tools, 2 files changed). A structurally shaped report exists (R3); **structure confers no substantive acceptance; rc 0 does not prove all items were reviewed.** No prevention claim rests on this.
- **Standing gap**: the recipe has the *reactive* rc18 clause (`gate-3-route.md:476-480`, R5) but no *preventive* clause (R10 scoped negative). The identity invitation broke no standing rule.
- **Class history (motivation, not frequency)**: hung-command slot losses in three missions; two controller-lane (already capped). **Role-lane recurrence: N=1.**

**Impact:** one full slot + recovery bookkeeping per occurrence; N=1 measured.

## Goals

**Primary goal:** standing preventive **instruction-level** pi-recipe clause — provenance-labeled identity disposition, explicit broad-discovery prohibition, named-artifact and missing-input rules — rc18 retry rule preserved verbatim. Recipe text only: **no** runtime read/write/discovery refusal, **no** composition linter or pre-spawn check of real directives, **no** prompt-obedience guarantee (scope honesty accepted from all three rc-3 reviewers).

**Success metrics (only what is checkable):**
1. Both recipe copies contain the exact contract block (element names, prohibition list, missing-input rule) and the new rc18 MUST-sentences; byte parity via the **existing** sync check + existing parity suite.
2. Every pre-existing rc18 sentence ("Retry ONCE on the same lane" etc.) verbatim — bounded manual inspection of exactly the changed lines.
3. `make test-launchd-drivers` green with **no new suite wired**.

**Not metrics:** real-directive conformance (unmeasured — no lint entry point exists and none is added, R11); prevention rates; model obedience.

## High-Impact Decisions — all **PROPOSED / UNAPPROVED** (human decides at attended approval)

| Decision | Why High Impact | Chosen By | Change Cost |
|----------|-----------------|-----------|-------------|
| Instruction-level guidance only; NO runtime enforcement, NO directive checker | Synthetic fixtures can't measure production compositions (unanimous rejection); no lint entry point (R11); developer doctrine: no tests mirroring a reversible low-impact edit | human | high |
| Contract block after `PI_EVALUATOR_SESSION_HANDSHAKE_END` (line 522) + one pointer in the pi real-run section (~355) | Single composition point; marker convention exists (R7); adjacency non-overlapping (R6) | human | med |
| Scope: pi role recipe only; codex adoption deferred | Incident/evidence pi-lane only | human | med |
| rc18 ONE-retry rule verbatim; only MUST-sentences appended | Settled policy, exercised and worked | human | low |
| **No new test suite, no fixtures, no `make/test.mk` wiring** | Accepted quorum objection + narrower scope | human | low |
| Identity **provenance-labeled** (DECLARED/RESOLVED vs OBSERVED vs unavailable), never researched | A pre-run pin can't prove backend identity; OBSERVED requires transport-captured response metadata with known NDJSON provenance; no weights attestation | human | med |
| P2 excluded — deferred (below) | Unbundle the unmeasured behavior half | human | high |

### Design Freeze — every row UNCHECKED, PROPOSED/UNAPPROVED; nothing here resolves a human choice

- [ ] **PROPOSED/UNAPPROVED**: block placement after line 522 + pointers; mirror via existing sync only.
- [ ] **PROPOSED/UNAPPROVED**: pi-recipe-only scope.
- [ ] **PROPOSED/UNAPPROVED**: worked example uses ONLY the sanitized evidence shape; `$HOME` literal (no expanded value exists in evidence; none may be introduced).
- [ ] **PROPOSED/UNAPPROVED**: P2 absent from milestones, AC, and changed paths.
- [ ] **PROPOSED/UNAPPROVED**: no `.pi/extensions/**` (D-FLEET-3) or `tools/pi-extensions/**` outside `sandbox/**` (D-FLEET-6) changes.

## Solution Design

### Overview
One marked **Role Artifact Contract** block in the shared pi routing recipe + two MUST-sentences on the rc18 clause. Standing proactive guidance — not enforcement, not a composition gate, not an obedience guarantee. Justification: the standing instruction gap plus one measured slot loss — not a proven causal fix.

### The contract block (normative content)
1. **Named INPUT artifacts** — every expected read named by exact path.
2. **Named OUTPUT artifacts** — every required produce named by exact path; an absent output path is *expected*, never a missing input.
3. **Working-tree context** — worktree root path + immutable base SHA under review.
4. **Identity disposition (provenance-labeled)** — directive supplies exactly one of: **DECLARED/RESOLVED** (configured/resolved lane value, labeled as such; a pre-run pin proves no backend identity); **OBSERVED** (actual returned message metadata captured by the transport, cited with its known NDJSON path; still no weights attestation); or `unavailable`. The role NEVER derives identity from session stores, `$HOME`, metadata trees, or env values; token/cost capture is transport-side from the named run NDJSON, not a role task; directives carry **no ambient env values or secrets**.
5. **Broad-discovery prohibition, precisely defined.** *Prohibited:* `find`/`ls -R`/`grep -r`/glob walks rooted at `$HOME` (any quoting), `/`, session stores (`~/.pi`, `~/.claude`, `~/.ailang`, equivalents), or metadata trees; `env`/`printenv`-style dumps aimed at identity/credentials; reading a previous attempt's artifacts unless named. *Remains permitted:* exact-path reads; bounded in-worktree `rg`/`git grep`/`git log/show`/`ls`; named sibling evidence paths.
6. **Missing-input rule** — a required INPUT absent at its supplied path is **reported against that path** (name it, state what was checked, proceed on the rest or stop per the role's gate) — never license to search elsewhere; a missing report is not empty-but-successful.

### rc18 extension (`gate-3-route.md:476-480`, edit in place)
Keep every existing sentence verbatim (R5) and append: the retry directive **MUST name the literal `hung_tool` command it forbids** (verbatim where shareable, sanitized *shape* otherwise — worked example: `find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin <N>`; **never expand `$HOME`**) and **MUST restate the contract's identity disposition** when the hang's context shows identity research was the trigger.

### Adjacency to the evaluator handshake (verified non-overlap)
Lines 494–522 read in full this session: protocol preamble only (read CLAUDE.md, classify unattended, `session_protocol_ack`, stop as transport failure) plus mission-stage env values — **0 identity mentions, 0 metadata mentions** (`grep -ci`, R6; matches controller full-content proof, S). The new block goes AFTER the END marker; the required preamble, inbox posture, and protocol are untouched.

## Implementation Plan (gated on approval — nothing runs before the terminal STOP clears)

**M1 (~1.5–2 h):** draft the marked block (~40–55 lines) after line 522; one pointer sentence at ~line 355; append rc18 MUST-sentences + worked example (lines 476-480 grow, no sentence removed); run `tools/launchd/sync-agents-skills.sh` (unedited) and verify `--check` rc 0; `make test-launchd-drivers` green; Gate-4 evidence row; controller commits.

## Files to Modify/Create

- `.claude/skills/mission-control/resources/gate-3-route.md` — (+ ~45–65 lines est.) marked contract block, one pointer sentence, rc18 MUST-sentences (existing sentences verbatim).
- `.agents/skills/mission-control/resources/gate-3-route.md` — byte-identical mirror via the EXISTING sync; parity enforced by the EXISTING `test_agents_skills_sync.sh` (R9).

No other paths; both inside the fleet Authority allowlist (R14). R1's scope (new ~150-line test, self-reconstruction fixtures, `make/test.mk` +1) is **removed entirely**.

## Examples

**Before** (triage-33 original, sha256 `90536f…6dd7`, verbatim): `Return actual provider/model from your session metadata if available, not a guessed identity.` — invited research into session/HOME/metadata trees; conformant with every standing rule; result rc 18 / 1119 s / no report.

**After** (contract-conformant, pi role lane):
```
Identity for this run (DECLARED/RESOLVED — configured/resolved lane, not observed response
metadata): provider=ollama, model=minimax-m3:cloud. Record as supplied; do not research your
identity; if unusable, record "unavailable". Transport captures tokens/cost from this run's
NDJSON at its named path. Absolutely no HOME scans, hidden-store/session lookup, environment
inspection, global file discovery, or access outside this isolated worktree.
```
Historical note: the actual retry phrasing ("The transport has independently verified your actual message metadata…") is permitted only as **OBSERVED** with its capturing NDJSON named; pre-run resolution alone must read DECLARED/RESOLVED.

## Success Criteria

- [ ] `sync-agents-skills.sh --check` rc 0; copies byte-identical; `make test-launchd-drivers` green (existing suites only).
- [ ] Bounded manual inspection: contract element names + prohibition list + missing-input rule present in both copies; pre-existing rc18 sentences verbatim; new MUST-sentences and literal-`$HOME` example present; no recipe line expands a real home path into it.
- [ ] `git diff --name-only` limited to the two recipe paths (plus this doc's Status line if the controller lands it; controller commits).

## Testing Strategy — no new tests (accepted rc-3 objection)

(1) A checker's self-assembled synthetic fixtures cannot measure production controller-composed directives — the unanimous rejection ground. (2) No production consumer: composition is controller-side at runtime; the role runner has no lint entry point (R11 scoped negative + control). (3) Developer doctrine: no tests mirroring a reversible low-impact instruction edit. Verification = existing sync/parity machinery + bounded manual inspection. Real-composition linting is Future Work (attended).

## Duplicate / Coverage Gate — **UNRESOLVED / BLOCKED** (named explicit gate)

Skill mechanics: ≥0.75 neural on a planned doc = REJECT; no author self-exception. Exact returned results (controller-captured NDJSON, `iteration96/production-designer-r1-search-capture.json`; **no fresh search/reindex/repair/model-pull performed**): neural on planned docs — `m-coordinator-child-env-opencode-retry-storm-recovery-plan.md` **(1.00)**, `m-entropy-budgets.md` **(0.95)**, `m-ifc-cross-module-labels-sprint-plan.md` **(0.90)**. R1 dismissed these as "keyword noise" — not skill compliance; retracted as a self-resolution.

Scope reads this session (R15): the 1.00 match is a sprint-recovery bookkeeping plan (coordinator child-env/opencode retry storm; status `blocked_pending_fresh_quorum`); 0.95 is a language-design doc (semantic entropy budgets); 0.90 is an IFC cross-module labels sprint plan. None addresses pi role directive content. **But the author cannot reconcile these scores against the skill threshold on their own authority.** The gate stands **UNRESOLVED / BLOCKED** and blocks planning even if the second quorum passes; scope + exact evidence are documented here for the operator/reviewer ruling.

## Deferred Decisions (agent latitude only after all gates clear)

Codex-recipe adoption (human); marker names/exact sentence wording within the pinned clause set; pointers for other pi roles (pointer-sized only).

## Deferred proposal — P2 (OPTIONAL, SEPARATE, attended decision required; NOT part of this design)

> Evaluator item P2 (per-command bash cap on role runs via unmodified `controller-bash-cap.ts`) stays **fully deferred**: it changes behavior on the `(cap, stall]` band — a periodically-progressing long command can live past stall today, so an unchanged stall setting does not prove policy neutrality — and controller-lane vs role-lane semantics differ (wording, resume flags, watchdog rhythm). It needs its own attended choice and future design. **No cap value, threshold, env knob, extension load, runtime wiring, lane, or billing change is chosen or introduced here.**

## Non-Goals

- Any runtime refusal/hook/guard/sandbox-rule/timeout/extension change; any pre-spawn lint of composed directives.
- Any real-directive conformance measurement, prevention-rate, or prompt-obedience claim — this pins recipe text only.
- Any threshold/stall/wall/retry-count/lane-order/fallback/quota/ration/billing change; ONE-retry and 600 s/1800 s bounds untouched.
- `.pi/extensions/**` (D-FLEET-3), `tools/pi-extensions/**` outside `sandbox/**` (D-FLEET-6), codex-role or new-instrumentation expansion.
- Closing `tool-hang:triage-judge:home-metadata-search` — controller resolves post-landing with `--sha`.
- **No AGENTS.md friction-twice fast-track** — the extension carve-out does not apply; only the terminal STOP gates do.

## Related Documents

- **[m-fleet-slot-kill-reaps-descendants-sprint-plan.md](m-fleet-slot-kill-reaps-descendants-sprint-plan.md)** — **overlap correction (r1 was wrong):** its Files section DOES include `make/test.mk` (read this session, R13). R1's "no overlap in files" is corrected, not rewritten away (r1 draft preserved as evidence). The former r1 `make/test.mk` overlap is **removed by this revision's narrower scope** (no test/make changes); the remaining two paths overlap the sibling nowhere, so no merge-order note is needed.
- **[evaluator-report.md §§4–6](../fleet-mission-evidence/iteration96/evaluator-report.md)** — independent disposition; P1 implemented, P2 deferred; not inherited: any implication that synthetic tests pin real conformance.
- **D-FLEET-34 / [fleet-mission.md](../fleet-mission.md)** — `stall:watchdog:slow-on-hung-tool` concerned *controller* thresholds; this doc changes none.
- **[m-harness-mission-loop.md](m-harness-mission-loop.md)** — fleet charter; this is one of its loop-harness fix items.

## Verification Log

Claim → bounded command → observed. **(O)** own observation this session; **(S)** controller-supplied proof. All read-only, ≤30 s, from the worktree root; no HOME/session/metadata sweep, env dump, or credential probe.

| # | Claim | Command | Observed | Src |
|---|-------|---------|----------|-----|
| R1 | First attempt / retry facts | `python3` json-reads of `iteration95/new-ticket-first-attempt.json`, `…/new-ticket-retry.json` | attempt: `tool_hang`, rc 18, 1119 s, 148 tools, `worktree_changed_since_start false`, 0 commits, `hung_tool` = literal `$HOME` sweep (unexpanded); retry: `ok`, rc 0, 274 s, 75 tools, 2 files changed | O |
| R3 | Retry report SHAPE only | `python3` shape-read `/tmp/triage-i33/judge-retry-worktree/JUDGE_REPORT.json` | keys incl. `items`,`auxiliary`,`actual_judge_provider/model`; items=5, aux=6 — structure, not substance | O |
| R4 | Original named inputs+output AND invited identity research; retry banned discovery | read `iteration96/original-directive-excerpts.json` | quoted lines (Problem Statement); sha256 pair as controller-recorded (not recomputed) | O (hashes S) |
| R5 | rc18 clause wording, 476-480 | read `gate-3-route.md:476-480` | "…NOT a lane failure at all… Retry ONCE on the same lane… FLAG the command as a harness ticket." | O |
| R6/R7 | Handshake 494-522: 0 identity/metadata mentions; markers | `sed -n '494,522p' … \| grep -ci 'identity\|metadata'`; `grep -n PI_EVALUATOR_SESSION_HANDSHAKE_ gate-3-route.md` | count `0` (protocol preamble only); START 494, END 522 | O; matches S full-content proof |
| R8 | Recipe copies identical | `diff -q` both `gate-3-route.md` copies | no diff | O |
| R9 | Sync/parity machinery exists, wired | `ls tools/launchd/sync-agents-skills.sh tools/launchd/test_agents_skills_sync.sh`; grep `make/test.mk` | both exist; `LAUNCHD_SUITES` `make/test.mk:75`; target `test-launchd-drivers:132` | O |
| R10 | No standing identity rule in recipe (negative) | `grep -ni identity gate-3-route.md` | one hit (:224, byte-identity checksum context) — no directive identity rule | O |
| R11 | No directive lint entry point in role runner (negative) | `grep -n "lint\|contract" scripts/mission_pi_run.sh`; control `grep -c directive` | one comment (:213); control = 6 | O |
| R12 | Planner-Lane mechanics; fleet allowlist unwidened; derive outputs | read `derive-planner-lane.sh` Steps 0/2/3 + `mission-control.sh:1439`; `grep -n PLANNER_ALLOWLIST missions/fleet.toml`; run derive on this doc | field = bare `codex-ok`/`opus-required`; default allowlist excludes this doc's paths; fleet.toml grep **empty** (rc 1); post-rewrite derive verbatim rc 0: `codex:gpt-6.1-sol anthropic-fallback:declared:opus-required` (declared=opus-required; emitted=driver fallback) | O; S: r1 inline → `…:fail-closed:planner-lane-field-invalid`; standalone scratch → same `…:declared:opus-required` |
| R13 | Slot-kill sibling Files INCLUDES `make/test.mk` (corrects r1) | read sibling lines 11-18 | "`- make/test.mk: invoke the new test with /bin/bash in test-launchd-drivers.`" | O |
| R14 | Both paths in Authority allowlist | read `fleet-mission.md` § Authority | `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**` covered | O |
| R15 | Duplicate-gate capture + scope reads | read `…/production-designer-r1-search-capture.json`; `head -12` of the 3 named docs | neural 1.00/0.95/0.90 as listed; scopes quoted in gate section — all distinct | capture S; reads O |
| R16 | Quorum rc 3 BLOCKED, 3 present rejects, 1 absent (quota), $0.1508914 | read `…/iteration96/quorum-r1.json` | verdicts/cost/absence exactly as stated; all objections reject synthetic-fixture population claims | artifact O (run S) |

**Named-artifact availability:** every named artifact existed at its supplied path and was read directly; no absent-input condition occurred.

## Terminal STOP — ALL gates must clear before any planning or implementation

1. **Duplicate/Coverage gate UNRESOLVED/BLOCKED** — operator/human ruling required; blocks planning even if the quorum passes.
2. **ONE controller-run re-quorum must PASS.** A quorum PASS is **not** design approval.
3. **Explicit attended human design approval** must ratify every PROPOSED/UNAPPROVED row.
Only then: sprint-planner → "execute sprint" → sprint-executor → sprint-evaluator, normal gates. Nothing in this revision skips any step.

## Future Work

- P2 design doc (role per-command cap), only via attended ruling; owns the `(cap, stall]` measurement, controller-wording, and policy-class questions.
- Codex-recipe adoption of the clause (deferred decision).
- Gate-3 composition lint for real directives (controller-workflow change, attended) — explicitly not built here.

---

**Document created**: 2026-11-08 (fleet iteration 96; scaffold via `create_planned_doc.sh`)
**Last updated**: 2026-11-08 — revision 2 by the rotation designer (`pi:ollama/kimi-k3:cloud`, DECLARED/RESOLVED label), addressing every rc-3 objection and the controller note
