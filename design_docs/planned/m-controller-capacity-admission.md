# M-CONTROLLER-CAPACITY-ADMISSION

**Status:** RULED **A** (D-FLEET-11, Mark, attended, 2026-10-02) — narrow scope authorized; Quorum round 1 BLOCKED → Revision 2 (designer); round 2 BLOCKED at N−1 on verbatim-fixable objections only → **Revision 3** (controller, Gate-2 narrow-refinement carve-out) applied them; routed to sprint plan (fleet iteration 15).
**Date:** 2026-10-02 · **Mission:** fleet iteration 15 (Rev 1: iteration 14) · **Priority:** P0 (`blocking=all`)
**Target:** unversioned; current source v0.52.0. **Estimate:** 4–6h including bounded controller drills and independent evaluation.
**Revision 2 author:** `claude:claude-opus-5-5` (designer role). **Revision 1 author:** `codex:gpt-6.1-sol`.
**Source audited (Rev 2):** `76a5aef65` (origin/dev), after the `f75b8e158` refactor that moved probes + the ration gate into `tools/launchd/lib/lane-probe.sh`. Every line reference below is re-derived at this SHA.
**Ticket:** `driver:controller-fallback-skips-openrouter-ration-and-402-reads-as-crash`; one occurrence, `2026-10-02T05:52:38.021154Z`, canonical ID `inbox_1790920358021_18b9ddb0`.

## Ruling (D-FLEET-11 = A)

> "authorize the narrow scope as proposed (anchored HTTP 402 classification through the existing bounded demotion/re-walk and PAUSED-NO-CAPACITY handling, pause-aware final notification, controller regression tests). Thresholds, lane order, billing reader and rc stay unchanged." — Mark, 2026-10-02

Options B (measured admission/billing redesign) and NO are closed. Nothing in this revision broadens scope: no threshold, lane order, `_mc_load_ration` reader, re-walk bound, or process rc change.

## Problem and verified correction

The ticket combines two claims. **The missing fallback ration check is not reproduced.** `select_model` (`tools/launchd/mission-control.sh:926`) checks `_mc_is_demoted` (:1006) then `_mc_is_over_ration "$fb"` (:1010) before every fallback rung; `_mc_rung_bucket` maps `pi:openrouter/*` → `openrouter` (`lib/lane-probe.sh:248–258`, :251); `_mc_probe_pi` re-checks ration (`lib/lane-probe.sh:128`, :130). Rev 1's hermetic drill (all four buckets blocked → rc1, no probe) was run at `4460d91b9`; the logic moved files but not behaviour (`f75b8e158`: driver side is removals plus one `. lib/lane-probe.sh` line at :873, +282/−269 over 8 files). The historical start-time blocked lists did not contain OpenRouter (Fleet 03:14:57 and Stapledon 02:16:39 both `codex ollama anthropic`), so the admission half of the ticket is not a defect at HEAD.

**The HTTP 402 misclassification is confirmed.** Fleet 03:15:31 starts `pi:openrouter/z-ai/glm-5.3`; pi emits a line beginning `402: {"message"…` (request wanted 770513 tokens, 580808 affordable), then 04:18:20 `CRASHED at=gate-3 rc=1` (3840s). Stapledon 02:16:52 → analogous 854293/617193 refusal → 04:13:43 `CRASHED at=gate-3 rc=1` (7026s). Re-counted at Rev 2 by prefix only: `grep -c '^402:'` = 2 (fleet log), 1 (stapledon log), each `402: {"message` (V6). This is the support for "`^402:` is the actual observed pi emitter". Provider URLs/keys are never quoted.

## Real controller path

`_mc_run_once` (:2238) launches text-mode `pi` directly (:2293) — not `scripts/mission_pi_run.sh`. The retry loop (:2364–2413) slices this attempt's log lines and tests `RUNTIME_QUOTA_SIG` (:1077; consumed at :2373) **before** the transient check. On a match it demotes the rung, re-walks within `RUNTIME_QUOTA_REWALKS` (:1081, default 4), and when nothing is left sets `MC_PAUSED=1` (:2387) and **announces the pause via `_mc_notify "Mission …: PAUSED — no capacity"` (:2389–2391, label `pause`)**. The slot verdict then reads `PAUSED-NO-CAPACITY` (:2438). The final rc block (:2470–2505) does not consult `MC_PAUSED`, so at HEAD a pause additionally falls into the generic branch (:2482–2499): "FAILED (rc=N) — timeout or crash" notice **and** an rcfail episode write. The signature has no `402` arm, so today a 402 never reaches any of this and is a plain CRASHED.

The role runner has its own NDJSON last-assistant classifier (`scripts/mission_pi_run.sh:386`, rc19 `provider_quota` at :407). Different process seam (role runs, not the controller); not reused, not changed.

## Quorum round 1 objections → resolution

**1. claude-sonnet-5 — ClassifyFailure never audited.** `func ClassifyFailure` is `internal/coordinator/retry_chain.go:70`. It lower-cases an executor error string and returns `FailureTransport` on a substring hit in `transportSignatures` (:51–64: `status 429`, `rate_limit`, `status 5xx`, `eof`, `mid-generation (no output)`, …), else `FailureModel` ("do not spend another execution"). It has **zero non-test callers** in the tree (only `retry_chain_test.go:27,41`) — this is exactly the "ClassifyFailure unwired" in mission memory. It lives in the Go **cloud coordinator** package (Cloud Run task chain); the **bash launchd driver** that runs the controller never links or execs it, and no `ailang` subcommand exposes it, so the driver could only call it via a new CLI route (scope expansion, and against the S5 CLI reduction). It does **not** handle 402: a `402: {...credits...}` message matches no signature and classifies `FailureModel`, the opposite of the needed "advance the chain". Its loose `status 429`/`eof` substrings are also the matching style the driver comment (:1073–1076) forbids for a log full of mission prose. Conclusion: not the same seam, not a parallel mechanism on the same data — one classifies a coordinator executor's error string, the other an attempt-sliced controller log. The shell `^402:` arm extends the driver's existing anchored signature. Wiring ClassifyFailure (and teaching it 402) is a separate cloud-plane item, out of scope; flagged for the controller below.

**2. gemini-3-1-pro — `elif MC_PAUSED` pauses silently.** The pause is not silent: the existing PAUSE branch already calls `_mc_notify` at **:2389–2391** before `break`, and that call retries ×3, dedupes same-day, spools on failure and posts the GH issue (:269–335). Adding a second `_mc_notify` in the final block would double-announce. The new `elif` therefore only (a) logs, (b) suppresses the misleading generic "timeout or crash" notice. Drill V13 proves the generic notice is suppressed only when `MC_PAUSED=1`; acceptance requires the fake-pi integration test to observe exactly one pause notice.

**3. oc-glm-5-3 — corrupt hunk, churn mismatch, rcfail unanalysed, no `_mc_notify`.** (a) The diff below is real `git diff` output from editing the worktree at `76a5aef65`, then reverted; the fenced block was extracted to `/tmp/fleet15/doc_r2.diff` and `git apply --check` re-run (V12). (b) Churn is the diff's own: `mission-control.sh` **+7/−2**. (c) The `rm -f rcfail.episode` from Rev 1 is **dropped**; see the analysis below. (d) `_mc_notify`: see objection 2.

### rcfail.episode analysis

Readers/writers (`grep -n rcfail tools/launchd/*.sh tools/launchd/lib/*.sh`; nothing outside `mission-control.sh`): **:2476** `rm -f` on the landed-record branch; **:2489** defined in the generic branch, read (suppress when equal to `$RC`), written `printf "$RC"`; **:2503** `rm -f` on rc0. Semantics (Mark 2026-08-31, comment :2484–2488): consecutive identical-rc crashes post once; the marker ends on rc change, completion, or a landed record.

Decision: **a pause does not touch the file.** A pause is neither a crash (so it must not *start or extend* an episode) nor a completion (so it must not *end* one). Clearing it would re-arm the crash notice so a crash→pause→crash dry-out posts every crash — the noise the gate exists to stop. At HEAD a pause *writes* `RC` into the marker (V13, HEAD arm), so a pause followed by a genuine rc1 crash suppresses the real crash notice: a latent masking bug this change removes. The landed-record branch clears because it is evidence of success; a pause is not.


## Quorum round 2 objections → resolution (Revision 3, controller-applied verbatim fixes)

Round 2 (2026-10-02T21:49:44Z) BLOCKED at N−1 (`gpt6-1-sol` absent, unreachable). Every objection carried a concrete reviewer fix and none disputed the direction, so the controller applied them verbatim under the Gate-2 narrow-refinement carve-out:

- **gemini-3-1-pro / oc-kimi-k3** — `[ "$MC_PAUSED" -eq 1 ]` on an unset variable. Fix applied: `elif [ "${MC_PAUSED:-0}" -eq 1 ]`. V15 shows the variable is always bound (top-level `MC_PAUSED=0`), V17 shows zero stderr under `set -u` on all arms and that the guard is load-bearing under mutation. Acceptance item added (generic crash notice exactly once, zero stderr).
- **oc-glm-5-3** — deployment premise that `MISSION_RUNTIME_QUOTA_SIG` is unset. Fix applied: V15 setter search (repo, env files, plists, launchd domain) and V16 refactor stat; the suggested sentence is now in the Proposed-diff section.

## Proposed diff (reviewable, NOT applied)

Generated by `git diff` at `76a5aef65`, then reverted with `git checkout -- tools/launchd/mission-control.sh`. **Revision 3** (controller, Gate-2 narrow-refinement carve-out) changes only the `elif` guard to `"${MC_PAUSED:-0}"`, the verbatim fix from quorum round 2 (gemini-3-1-pro, oc-kimi-k3).

```diff
diff --git a/tools/launchd/mission-control.sh b/tools/launchd/mission-control.sh
index 637a99718..9240f255b 100755
--- a/tools/launchd/mission-control.sh
+++ b/tools/launchd/mission-control.sh
@@ -1070,11 +1070,12 @@ export STALL_CHILD_AGE
 # not congested. A same-model retry is guaranteed to fail, so the response is to
 # demote the rung and re-walk the chain (see MC_DEMOTED).
 #
-# Anchored to the four emitters we have actually observed, NOT to loose words like
+# Anchored to the five emitters we have actually observed, NOT to loose words like
 # "quota" or "rate limit". This log carries mission prose about quota routinely —
 # the codex probe's own output is written to it — so a loose pattern would demote
 # a healthy controller because the iteration happened to be writing about limits.
-RUNTIME_QUOTA_SIG="${MISSION_RUNTIME_QUOTA_SIG:-reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:}"
+# `^402:` is pi text-mode's OpenRouter credit refusal (`402: {"message":...`), fleet+stapledon 2026-10-02.
+RUNTIME_QUOTA_SIG="${MISSION_RUNTIME_QUOTA_SIG:-reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:|^402:}"
 # Bound the re-walks. The demote list already guarantees progress (each re-walk
 # removes one rung, so the chain is finite), but a bound keeps a pathological
 # chain from eating the slot.
@@ -2479,6 +2480,10 @@ if [ "$RC" -ne 0 ]; then
       --title "Mission iteration killed post-record (rc=$RC) — work landed" --from "$MSG_FROM" 2>/dev/null
     [ -n "${MISSION_GH_ISSUE:-}" ] && gh issue comment "$MISSION_GH_ISSUE" --repo "$MISSION_REPO" \
       --body "ℹ️ Mission iteration exited **rc=$RC after landing its record** at $(date '+%F %H:%M %Z') — the mission log gained an entry during this run, so this was a late watchdog kill of a lingering child, not a lost iteration. The queue advanced normally. Log on the rig: \`$LOG\`." 2>/dev/null
+  elif [ "${MC_PAUSED:-0}" -eq 1 ]; then
+    # Already announced by the PAUSE branch's _mc_notify. Not a crash: no generic notice,
+    # and the rcfail episode is left as-is (only rc change or completion ends it).
+    log "iteration paused for provider capacity (rc=$RC) — not a crash"
   else
     log "iteration exited rc=$RC"
     # EPISODE-GATED on the rc value (Mark 2026-08-31): consecutive identical crashes post ONCE,
```

No new rc, error code or verdict schema. The env override MISSION_RUNTIME_QUOTA_SIG keeps its meaning (an override replaces the default wholesale, as today); V15 confirms no plist, env, or script in the deployment sets it, so the five-emitter default is what the rig executes. Any future override must be re-based on this default or 402 admission silently no-ops. The landed-record branch keeps precedence over the pause branch (`if` before `elif`). Process exit stays the original nonzero `$RC` (:2506).

### Files to modify (sprint)

- `tools/launchd/mission-control.sh` (+7/−2): the diff above, exactly.
- `tools/launchd/test_controller_chain.sh` (~+50): it still extracts only `_mc_set_controller` and `select_model` (:22–23). After the refactor `_mc_is_over_ration`, `_mc_is_demoted` and `_mc_canon_id` are **undefined** in the suite — a run at HEAD logs 63 `command not found` (21 each) and still passes, i.e. those branches run vacuously (V9). Extract them (driver + `lib/lane-probe.sh`) and add blocked-OpenRouter/no-probe and all-blocked cases with an anti-vacuity "no `command not found` on stderr" assertion.
- `tools/launchd/test_controller_capacity.sh` (~120, new): hermetic fake-`pi` integration over the real retry loop + final rc block: attempt slicing, 402 demote/re-walk, pause, one pause notice, no generic notice, rcfail unchanged.
- `make/test.mk` (+1): wire the new suite into `test-launchd-drivers` (:70) next to `test_controller_chain.sh` (:84).

Excluded: `.pi/extensions/**`, `lib/lane-probe.sh` logic, billing reader, roles, `internal/coordinator`, language core, schemas.

## Design freeze

- [x] Mark rules A/B/NO — **A**, attended, 2026-10-02 (D-FLEET-11).
- [x] Independent quorum: round 1 BLOCKED (3/3), round 2 BLOCKED at N−1 (`gpt6-1-sol` unreachable) on verbatim-fixable objections only; Revision 3 applies those fixes (carve-out). No round-3 quorum. PASS is not plan approval.
- [ ] Approved sprint plan; independent evaluator separate from generator.

Delegated to the executor: helper/fixture layout and how outbound notices are stubbed, provided tests execute the actual driver control flow and spend zero inference. Not delegated: signature text beyond `^402:`, thresholds, lane lists, re-walk bound, rc, episode policy (frozen above).

## Verification log (executed at `76a5aef65` unless marked)

| ID | Command/source | Observed result |
|---|---|---|
| V1 | `git log -1`; `git status --porcelain` | `76a5aef65`; clean before edit |
| V2 | Ticket metadata | **Carried from Rev 1, not re-run** (designer must not touch messaging/tickets this pass) |
| V3 | `sed -n` / `grep -n` on `select_model`, `lib/lane-probe.sh` | fallback demote :1006, ration :1010; bucket map :248–258; pi probe ration :130 |
| V4 | All-blocked hermetic drill | **Carried from Rev 1 at `4460d91b9`, not re-run**; logic moved by `f75b8e158` without change. Re-run is acceptance item 1 |
| V5 | `RUNTIME_QUOTA_SIG` default (old vs patched) under `/usr/bin/grep -qE` on 6 probes | old: `402:`→1, `429:`→0, usage-limit→0. new: `402:`→**0**, `429:`→0, usage-limit→0. Both: `the provider returned 402: credits`→1, `  402: indented`→1, `HTTP 402 credits quota`→1 (prose stays unmatched) |
| V6 | `grep -c '^402:'` + `cut -c1-14` on `/tmp/ailang-mission-{fleet,stapledon}.log` | 2 and 1 hits, all `402: {"message`; Rev 1 timestamp anchors carried |
| V7 | Read :2238, :2293, :2364–2413, :2415–2451, :2453–2468, :2470–2506 | direct pi; quota-before-transient; PAUSE `_mc_notify` :2389; slot verdict :2438; final block ignores `MC_PAUSED` |
| V8 | `grep -n` `scripts/mission_pi_run.sh` | role-runner regex :386, rc19 :407 — separate seam |
| V9 | `bash tools/launchd/test_controller_chain.sh` with patch applied | rc0, last `PASS pi-rung-sets-provider-pi`; 63 `command not found` for the three unextracted helpers |
| V10 | `bash tools/launchd/test_mission_heartbeat.sh` with patch applied | rc0, `PASS: 25 heartbeat arms ran` |
| V11 | `grep -rn ClassifyFailure --include='*.go'` | def `retry_chain.go:70`; callers only `retry_chain_test.go:27,41`; no `402` in either file |
| V12 | Extract fenced diff → `/tmp/fleet15/doc_r2.diff`; `git apply --check` | **rc 0**; extracted file byte-identical (`cmp`) to the live `git diff` |
| V13 | Final rc block awk-extracted (`^if [ "$RC" -ne 0 ]` … `^exit`) and sourced with stubbed `ailang`/`gh`/`log`, RC=1, patched vs HEAD | patched, paused, no marker → 0 sends, marker absent; paused, marker=1 → 0 sends, marker=1. not paused → 1 send / suppressed as before. HEAD paused, no marker → **1 generic send, marker written `1`** |
| V14 | `bash -n` on patched driver; `git checkout -- …`; `git status --porcelain` | syntax ok; reverted; only this doc modified |
| V15 | `grep -n MC_PAUSED tools/launchd/mission-control.sh`; `grep -n '^set -' …`; `grep -rn MISSION_RUNTIME_QUOTA_SIG .` (excluding `design_docs/`); `grep -l` over `~/.config/ailang/*.env` and `~/Library/LaunchAgents/*.plist`; `launchctl getenv MISSION_RUNTIME_QUOTA_SIG` | `MC_PAUSED=0` at top level :894 (before any function), `=1` only at :2387, read at :2438; driver runs `set -uo pipefail` (:38). Override: single repo hit, the :1077 definition; 0 env/plist hits (rc1); `launchctl getenv` empty. Default is live |
| V16 | `git show --stat f75b8e158`; `git show f75b8e158 -- tools/launchd/mission-control.sh` | 8 files, +282/−269; driver now sources `lib/lane-probe.sh` once (`. "$MC_DRIVER_ROOT/tools/launchd/lib/lane-probe.sh"`, :873). Supports carrying V4: the logic moved without change |
| V17 | V13 re-run under `set -uo pipefail` (`/tmp/fleet15/drill_u.sh`, bash 3.2.57), stderr captured, arms paused=1 / 0 / unset | patched: 1→0 sends, pause log, **0 stderr bytes**; 0→1 send; unset→1 send, **0 stderr bytes**. HEAD unset→1 send. Mutation (bare `"$MC_PAUSED"`, unset) aborts the block under `set -u` with no output — the `:-0` guard is load-bearing |

## Acceptance, tests and done-gates (NOT executed)

- [ ] All-ration-blocked extracted selection (with the real ration/demote helpers) returns no controller and invokes zero probes; blocked OpenRouter followed by an admitted rung reaches that rung; no `command not found`.
- [ ] Fake pi emits `^402:` and exits nonzero: one re-walk per demoted rung within the bound; terminal verdict `PAUSED-NO-CAPACITY`; exactly one `pause` notice; no "Mission iteration FAILED" notice; rcfail marker byte-identical before/after.
- [ ] 429 / usage-limit behaviour unchanged; exhausted re-walk budget pauses; no same-rung retry; a fresh eligible rung completes normally.
- [ ] A prior attempt's 402 cannot poison the next attempt; prose with `402`/`credits` not at line start does not classify. Unrelated nonzero stays CRASHED with the episode gate intact; watchdog kills stay KILLED.
- [ ] A nonzero rc produced without entering the pause branch posts the generic crash notice exactly once with zero stderr output (including `MC_PAUSED` unset, under `set -u`).
- [ ] rc0 with an old quoted 402 stays normal; landed-record branch keeps precedence over pause.
- [ ] Healthy and degraded dry-runs with `AILANG_DRIVER_PINNED=<tested SHA>`; every wait ≤30 min.
- [ ] `make test-launchd-drivers` rc0 on actual Bash 3.2, including the new suite. Changelog fragment. Independent evaluation passes.
- [ ] Mission-loop-change pre-flight (executing surface, pinned-origin reach, drills, suite, no reload mid-iteration).
- [ ] Ticket resolved only once on origin/dev; disposition states the ration half was not reproduced (no guard added).

## Risks and limitations

`^402:` could still match controller-generated text that starts a line with `402:`; negative tests cover realistic prose, and slicing bounds it to the current attempt. A 402 may mean a per-request budget (max_tokens) rather than an empty account; demotion is for this fire only and the next fire re-probes. A re-walk starts a different controller session on partial work — existing 429 behaviour, now knowingly extended to 402 by ruling A. Wall time spent inside pi before the 402 is not reclaimed. Pause notices are not episode-gated (same-day title dedupe only); unchanged.

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | 0 | same bounded declared chain |
| A2 Replayability | +1 | anchored emitter evidence, real diff |
| A3 Effect legibility | +1 | one pause notice; no false crash notice |
| A4 Explicit authority | +1 | attended ruling A recorded |
| A5 Bounded verification | +1 | hermetic drills, no inference |
| A6 Safe concurrency | 0 | watchdogs/overlap unchanged |
| A7 Machines first | +1 | typed capacity verdict replaces crash |
| A8 Minimal syntax | 0 | no language syntax |
| A9 Cost visibility | +1 | credit refusal recorded truthfully |
| A10 Composability | 0 | existing machinery reused |
| A11 Structured failure | +1 | pause no longer poisons the crash episode |
| A12 System boundary | 0 | shell controller only |

**Net +7.** No hard violations. Compiler conflict surface: N/A.

## Related documents

- [Fleet charter, HD-2a](../fleet-mission.md) · [Quota rationing/routing](m-quota-rationing-routing.md) · [Role-runner provider quota sprint](sprint-plan-pi-runner-provider-quota.md) · [Mission-loop-change skill](../../.claude/skills/mission-loop-change/SKILL.md)
