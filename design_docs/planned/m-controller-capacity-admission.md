# M-CONTROLLER-CAPACITY-ADMISSION

**Status:** PARKED `needs-human-review` — HD-2a policy ruling required; no plan approval or implementation.
**Date:** 2026-10-02 · **Mission:** fleet iteration 14 · **Priority:** P0 (`blocking=all`)
**Target:** unversioned; current source v0.51.0. **Estimate after ruling:** 4–6h including bounded controller drills and independent evaluation.
**Author:** native Agent designer, `codex:gpt-6.1-sol`, FLAGGED rotation fallback after GLM, Kimi and Opus pins were rejected as Unknown model (controller-supplied transport record).
**Source audited:** `4460d91b91f9a4d78abadac5ae4026c823fb6a72`.
**Ticket:** `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash`; one occurrence, first/last seen `2026-10-02T05:52:38.021154Z`, canonical ID `inbox_1790920358021_18b9ddb0`. Ticket fetched with `ailang mission ticket open --json`, never read/acked through messaging.

## Problem and verified correction

The ticket combines two claims. The missing fallback ration check is **not reproduced at current HEAD**: `select_model` checks `_mc_is_over_ration "$fb"` before dispatch for every fallback rung (`tools/launchd/mission-control.sh:1260–1269`), `_mc_rung_bucket` maps `pi:openrouter/*` to `openrouter` (:1146–1155), and `_mc_probe_pi` checks admission again (:989–995). A hermetic drill of extracted current functions, with all four buckets blocked and probes stubbed, selects no controller (rc 1) and calls no inference probe. Do not duplicate the existing guard.

The historical admission premise is also unsupported by the cited first-party logs. Fleet's **03:14:57** blocked list is `codex ollama anthropic` (`/tmp/ailang-mission-fleet.log:35308`); Stapledon's **02:16:39** list is the same (`/tmp/ailang-mission-stapledon.log`, timestamp is the stable locator because the log remains append-active). OpenRouter is absent in both snapshots. Later blocked snapshots must not be substituted for these start-time snapshots. This does not prove the quota observation was accurate; it establishes that the ticket's quoted later list does not prove admission skipped a blocked OpenRouter bucket. Snapshot accuracy is unmeasured here, and a billing-reader change requires a separate concrete proposal.

The HTTP 402 misclassification **is confirmed**. At Fleet **03:15:31** the controller starts as `pi:openrouter/z-ai/glm-5.3`; its next provider error begins `402:` and says the request requires more credits or fewer max_tokens (770513 requested, 580808 affordable), followed by **04:18:20** `CRASHED at=gate-3 rc=1` (3840s). Stapledon starts **02:16:52**, receives the analogous 854293/617193 refusal, and ends **04:13:43** `CRASHED at=gate-3 rc=1` (7026s). Provider-response URLs are intentionally omitted from this doc. These are timestamp-anchored driver/provider emissions, not later agent prose quoting the ticket.

## Real controller path and systemic audit

`_mc_run_once` invokes text-mode `pi` directly, with the session ID, extension arguments and stdin EOF (:2495–2540). It does **not** invoke `scripts/mission_pi_run.sh`. The driver's post-exit loop (:2603–2655) tests the attempt-only log slice against `RUNTIME_QUOTA_SIG` before transient retry. That signature (:1332) recognizes three usage-limit phrases and anchored `^429:`; it does not match the observed `402:` emission. Generic nonzero verdicts then become CRASHED (:2668–2679), with generic timeout/crash notification (:2724–2740).

The role runner already parses the **last assistant error** from NDJSON and handles 402/429 as `provider_quota` rc19 (`scripts/mission_pi_run.sh:386–407`). That fix cannot certify this distinct text-mode controller path. Reuse the controller's existing demote/re-walk/PAUSED machinery rather than introducing the role runner, another retry loop, or a shared schema migration. `_mc_load_ration` (:1064–1097) reads and caches quota observations; its command-failure fallback blocks codex and ollama. Changing that fallback, refresh policy, unknown-bucket policy, thresholds or admission margins is a billing/admission policy change and is outside this minimal proposal.

## High-impact decisions and ruling request

| Decision | Recommendation | Decider | Cost of reversal |
|---|---|---|---|
| D1: scope the ticket to the proved controller 402 defect | A: retain existing ration checks; add an anchored `^402:` emitter to existing runtime exhaustion handling | Mark under HD-2a | Low source churn; changed capacity verdicts and lane re-walk behavior |
| D2: all-ration-blocked admission return/notice | Retain current rc1/no-controller refusal; do not claim an rc75 redesign is required | Mark if changed | Recovery/notice consumers must be audited for rc75 before changing |
| D3: quota observation/billing guard | No speculative reader or guard patch; a requested audit needs start-time quota output/source evidence | Mark | Potential cost/routing consequences |

**One-answer ruling options:** **A (recommended)** authorizes only the narrow 402 classifier plus controller regression tests using existing runtime demotion/re-walk bounds and PAUSED-NO-CAPACITY behavior. No threshold, routing order, billing reader, or no-controller rc change. **B** requires a separate measured admission/billing design (including rc75 consumer audit) before any implementation. **NO** leaves this ticket parked without changes. **Default if unanswered:** `needs-human-review`; no sprint approval, execution, ticket resolution or production change. This document's ruling request must be entered by the controller in the fleet decision ledger; the designer owns only this file and has not assigned or edited a ledger ID.

Even the narrow change can cause a different declared lane to be selected after a runtime refusal. It therefore remains parked rather than self-certifying as mechanical. Its emitter classification is mechanically separable from the unproved admission/billing half; recommendation A is that exact limited scope, not authority to broaden policy.

## Proposed minimal diff (reviewable, NOT applied)

Apply only after ruling, fresh quorum and normal plan/execute gates. This patch preserves existing environment override semantics, re-walk limit, retry policy and lane order. `^402:` is the actual observed pi emitter; do not use loose `credits`, `quota` or unanchored `402` patterns against mission prose.

```diff
diff --git a/tools/launchd/mission-control.sh b/tools/launchd/mission-control.sh
--- a/tools/launchd/mission-control.sh
+++ b/tools/launchd/mission-control.sh
@@ -1326,8 +1326,9 @@
 # demote the rung and re-walk the chain (see MC_DEMOTED).
 #
-# Anchored to the four emitters we have actually observed, NOT to loose words like
+# Anchored to the emitters we have actually observed, NOT to loose words like
 # "quota" or "rate limit". This log carries mission prose about quota routinely —
 # the codex probe's own output is written to it — so a loose pattern would demote
 # a healthy controller because the iteration happened to be writing about limits.
-RUNTIME_QUOTA_SIG="${MISSION_RUNTIME_QUOTA_SIG:-reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:}"
+# pi text-mode OpenRouter credit refusal: 402: {"message":...,"code":402}.
+RUNTIME_QUOTA_SIG="${MISSION_RUNTIME_QUOTA_SIG:-reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:|^402:}"
 # Bound the re-walks. The demote list already guarantees progress (each re-walk
@@ -2723,3 +2724,6 @@
-  else
+  elif [ "$MC_PAUSED" -eq 1 ]; then
+    log "iteration paused for provider capacity (rc=$RC) — not a crash"
+    rm -f "$STATE_DIR/mission-${MISSION_NAME:-control}-rcfail.episode"
+  else
     log "iteration exited rc=$RC"
     # EPISODE-GATED on the rc value (Mark 2026-08-31): consecutive identical crashes post ONCE,
```

A reviewer can extract this fenced diff to a temporary file and run `git apply --check <file>`; it is a proposal, not a command to apply now. No new error code, rc or verdict schema is allocated. The existing branch demotes the rung, walks the admitted declared chain, and when none remains sets `MC_PAUSED=1`; the slot then reads `PAUSED-NO-CAPACITY` rather than CRASHED. Driver process rc remains the original nonzero rc. The existing generic rc-failure notice still says timeout/crash even after a pause; if A is ruled, the proposed companion adds a small `MC_PAUSED` branch before generic failure notification to log capacity pause and avoid emitting that misleading generic crash notice. Reuse the existing `_mc_notify` pause message; do not suppress actual crash notices. This notification companion is required for truthful end-to-end acceptance, not an optional polish task.

### Files to modify after approval

- `tools/launchd/mission-control.sh` (~+6/-2): signature, emitter comment, and pause-aware final rc notification branch. Preserve nonzero exit and record-first handling.
- `tools/launchd/test_controller_chain.sh` (~+50): extract actual ration/demotion helpers; currently it extracts only set/select and stubs probes, so it does not itself prove ration helper behavior. Add blocked OpenRouter/no-probe and all-blocked cases with anti-vacuity assertions.
- `tools/launchd/test_controller_capacity.sh` (~120 new): hermetic direct-controller fake-pi integration, attempt slicing, re-walk/no-capacity and notification behavior.
- `make/test.mk` (+1): wire new controller capacity suite into `test-launchd-drivers` alongside the existing controller-chain suite (:82).

Production changes remain shell-only; `.pi/extensions/**`, billing implementations, roles, language core and schemas are excluded. The companion notification implementation is an agent-resolvable conditional with the behavior frozen above; no new policy decision is delegated.

## Design freeze and deferred decisions

- [ ] Mark rules A/B/NO and scope is recorded with attended provenance.
- [ ] Fresh independent quorum; triggers 1 (freeze), 3 (capacity/cost behavior) and 4 (provider emitter) apply. **NOT RUN by designer; controller owns quorum.** A quorum PASS is not a policy ruling or plan approval.
- [ ] Approved sprint plan and explicit execution authorization; independent evaluator remains separate from generator.

Agent may choose helper/test-fixture layout and how to stub outbound notices, provided tests execute actual current controller control flow and never spend inference tokens. No thresholds, model lists, re-walk limits, admission margins or guard behavior are delegated.

## Verification log (executed unless marked proposed)

| ID | Command/source | Observed result |
|---|---|---|
| V1 | `git rev-parse HEAD`; `git status --short` before design | `4460d91b91f9a4d78abadac5ae4026c823fb6a72`; initial tree clean |
| V2 | `ailang mission ticket open --json`, filtered by exact signature | rc0; one open blocking=all row, timestamp/ID above. CLI warns binary may be stale; no source-sensitive claim rests on binary behavior |
| V3 | `sed -n '1142,1180p;1260,1298p;989,1004p' tools/launchd/mission-control.sh` | OpenRouter canonical bucket, unconditional pre-dispatch fallback check, second pi probe check |
| V4 | Extract `_mc_rung_bucket`, `_mc_is_over_ration`, `_mc_canon_id`, `_mc_is_demoted`, `_mc_set_controller`, `select_model` with anchored function regex; execute `/bin/bash -c` with `_mc_load_ration(){ :; }`, blocked four-bucket list, preference opus, both pi fallback rungs, fail Anthropic/codex probes, pi stub emitting `PROBE` | `select_rc=1 controller=none`; bucket output `openrouter`; harness rc0; stderr empty (no PROBE) |
| V5 | Extract default `RUNTIME_QUOTA_SIG`; `/usr/bin/grep -qE "$sig"` with synthetic anchored 402, anchored 429 and usage-limit phrase | 402 rc1 (unmatched); 429 rc0; usage limit rc0 |
| V6 | Timestamp-anchored original driver lines in `/tmp/ailang-mission-{fleet,stapledon}.log` | Both start-time blocked lists exclude OpenRouter; selected pi OpenRouter; actual provider 402 followed by rc1 CRASHED, metrics above |
| V7 | Read :2495–2540, :2603–2655, :2668–2679, :2724–2740 | direct text pi path; attempt-sliced runtime classifier before transient; pause override exists; final generic notice is separate |
| V8 | Read `scripts/mission_pi_run.sh:386–407` | last-assistant NDJSON capacity check/rc19 exists in role runner, not controller dispatch |
| V9 | Read `tools/launchd/test_controller_chain.sh` and `make/test.mk:82` | existing suite extracts set/select, but not actual ration helpers; suite wired |
| V10 | `rg -n 'controller|ration|402' design_docs/fleet-mission-index.md`; `git log -4 --oneline -- tools/launchd/mission-control.sh` | index has no earlier controller-credit design; recent driver edits are extensions/reader timeout/notices/model pin, not proof of missing 402 fix |
| V11 | `bash .claude/skills/design-doc-creator/scripts/create_planned_doc.sh m-controller-capacity-admission` | rc0, scaffold at owned path; dual related-doc search completed |
| V12 | proposed signature and patch check | `git apply --check` on extracted fenced diff rc0; current regex 402 rc1, proposed regex 402 rc0; no live provider inference attempted |

Reproduce V4 using extracted functions, not duplicated selection logic. For historical V6 use start/end timestamp anchors (`03:14:57`, `03:15:31`, `04:18:20` Fleet; `02:16:39`, `02:16:52`, `04:13:43` Stapledon); line numbers can move as logs are written. Read only provider prefix/message/token numbers, omitting key/workspace URLs. No assertion is made about provider API contract beyond the observed actual pi emission.

## Acceptance, tests and done-gates (NOT executed)

- [ ] All-ration-blocked extracted current selection returns no controller and invokes zero inference probes; blocked OpenRouter followed by admitted declared rung reaches that rung. Unknown/stale reasons retain current distinctions.
- [ ] Actual fake pi controller emits anchored 402 and exits nonzero: existing chain re-walk occurs once per demoted rung within the existing bound; capacity terminal state is PAUSED-NO-CAPACITY, not CRASHED; pause notice appears, generic crash notice does not.
- [ ] Existing 429/usage-limit behavior unchanged; exhausted re-walk budget pauses; no duplicate same-rung retry. Fresh eligible rung can complete normally.
- [ ] Prior-attempt 402 cannot poison next attempt; prose containing 402/credits does not classify; current anchored 402 does. True unrelated nonzero remains CRASHED and watchdog kills remain KILLED.
- [ ] rc0 with an old quoted 402 remains normal; capacity classifier runs only on actual nonzero controller completion. Successful record behavior retains precedence.
- [ ] Run healthy and degraded dry-runs with exact `AILANG_DRIVER_PINNED=<tested SHA>` so pinning cannot substitute another source; every wait has an explicit ≤30-minute deadline.
- [ ] `make test-launchd-drivers` rc0 including new suite, actual Bash 3.2 verified. Documentation updated; independent sprint evaluation passes.
- [ ] Mission-loop-change five-line pre-flight: actual executing surface/copies identified, pinned-origin reach confirmed after authorized landing, healthy/degraded drills green, driver suite green, no reload during an iteration. No reload is needed for this shell-only change; verify installed/pinned state instead.
- [ ] Resolve ticket only after judged change is on origin/dev. Ration premise disposition must state corrected evidence, not falsely claim a guard was added.

## Risks and limitations

Anchored `402:` within controller-produced prose at line start could still false-match text-mode logs; test realistic negative prose and preserve attempt boundaries. Structured terminal session extraction would reduce ambiguity but expands scope and is not proposed here. A 402 can represent key/request budget capacity rather than a globally empty account; demotion is per rung **for this fire**, never a permanent account verdict. Re-walking can start a different controller/session with partial prior work; that is already existing quota behavior, and this ruling must knowingly extend it to 402. The provider spent significant wall time inside pi before its terminal error; post-exit classification cannot reclaim those minutes. No retry-timing or request max_tokens change is proposed.

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | 0 | same bounded declared chain; no new randomness |
| A2 Replayability | +1 | timestamped attempt and exact emitter evidence |
| A3 Effect legibility | 0 | existing provider/notice effects |
| A4 Explicit authority | +1 | HD-2a ruling required, no self-approval |
| A5 Bounded verification | +1 | hermetic fake-provider/drill acceptance |
| A6 Safe concurrency | 0 | watchdogs and overlap controls unchanged |
| A7 Machines first | +1 | existing typed capacity verdict replaces ambiguous crash |
| A8 Minimal syntax | 0 | no language syntax |
| A9 Cost visibility | +1 | capacity refusal recorded truthfully |
| A10 Composability | 0 | existing demote/re-walk machinery reused |
| A11 Structured failure | +1 | capacity differentiated from crash |
| A12 System boundary | 0 | shell controller only |

**Net +6.** No hard violations on A1/A3/A4/A7. Score qualifies the draft, never bypasses policy/quorum/plan gates. Compiler Conflict Surface is not applicable: no compiler/runtime language paths touched; controller conflict surface is captured in negative tests and unchanged watchdog/record precedence above.

## Related documents

- [Fleet charter, HD-2a](../fleet-mission.md): authority and done-gate.
- [Quota rationing/routing](m-quota-rationing-routing.md): existing bounded runtime exhaustion policy.
- [Role-runner provider quota sprint](sprint-plan-pi-runner-provider-quota.md): distinct NDJSON controller/role seam.
- [Mission-loop-change skill](../../.claude/skills/mission-loop-change/SKILL.md): actual-running surface and five-line pre-flight.

No production code, plan, ledger, ticket state or messages changed by this designer. This document remains uncommitted for controller review.
