# Fleet iteration 128 — independent idle-screen judge

- **Role**: `evaluator`, read-only at `09e190679` (origin/dev). Inputs only; no controller conclusion was carried in.
- First-party checks run: `ailang mission ticket open --json` (16 tickets; `skill:idle-record-barred-on-degraded-no-pick-fire` slots_lost=7, last_seen 2026-11-20T09:42:34Z, latest message from World), the World driver log `/tmp/ailang-mission-world.log`, World's repo `~/dev/sunholo-data/ailang-world` (read-only), `tools/launchd/lib/idle-fingerprint.sh`, `tools/launchd/mission-control.sh:1765-1800`, charter Authority/Guardrails, gate-2 lines 49-67, gate-5 lines 11-31.

---

VERDICT: NONE (confidence 75%)

## Q1. All 16 tickets -> disposition (no change since iteration 127 except the 7th occurrence of #15)

| # | Signature | Disposition |
|---|---|---|
| 1 | `ci:distinct-build-test-failures` | Row 44; failing tests in `cmd/ailang`, `internal/daemon`, `internal/runner`: outside the Authority allowlist. |
| 2 | `coordinator:cloud-lane-capped-by-local-worktree-limit` | Row 52; coordinator code, out of scope. |
| 3 | `coordinator:job-retry-success-dropped-after-failed-completion` | Row 55, fix merged attended (#2348); bookkeeping, awaits D-FLEET-46. |
| 4 | `crash:sprint-executor:sigbus-ram-tmp-full-make-test` | Row 27 / D-FLEET-23 RULED; remainder attended/V1 + terraform, out of scope. |
| 5 | `eval:motoko-connection-probe-orphaned` | Row 58, folded into D-FLEET-18 (RULED A, landed). Resolving = bookkeeping, no diff; same class as D-FLEET-46, not covered by it. |
| 6 | `executor:3h-timeout-from-queue-and-unsliced-sprints` | Row 56, #2343 -> `ad2148b23`; D-FLEET-46. |
| 7 | `executor:codex-credential-collision-drops-automatic-planner-handoffs` | Row 54, #2348; D-FLEET-46. |
| 8 | `executor:codex-credential-lease-collision-fails-concurrent-dispatch` | Row 54; D-FLEET-46. |
| 9 | `executor:timeout-loses-work-and-may-never-start` | Row 56; D-FLEET-46. |
| 10 | `messaging:pubsub-notify-wrong-project` | Row 18 / D-FLEET-19; rig change, no repo commit for `resolve --sha`; excluded from D-FLEET-46. |
| 11 | `pi:openrouter:uncapped-max-tokens-402` | Row 50, #2349; D-FLEET-46. |
| 12 | `quorum:claude-p-ration-usage-429` | D-FLEET-28 A, #2344 -> `0222baeb9`; row 68; D-FLEET-46 addendum. |
| 13 | `quorum:gpt6-1-sol-openai-api-credits-exhausted` | Row 21, #2347; D-FLEET-46. |
| 14 | `reporting:daneel-reports-not-declared-triage` | Row 24 / D-FLEET-22; prod-config rollout, no PR, out of scope. |
| 15 | `skill:idle-record-barred-on-degraded-no-pick-fire` | Row 62, parked on D-FLEET-45 (OPEN). 7 slots lost. |
| 16 | `tool-hang:triage-judge:home-metadata-search` | Row 61, parked on D-FLEET-44 (OPEN, design quorum-blocked). |

Answer: no ticket is mechanical + in scope + unparked + not awaiting D-FLEET-46. The only in-scope candidate is #15 (Gate-5 skill resource is under `.claude/skills/mission-*/**`), and it is parked on an OPEN row with a twice-blocked design.

## Q2. Screen by step

- D-FLEET-44: recommended A = request corrected evidence/design (a document request, not an in-scope code step); B default. No.
- D-FLEET-45: recommended = B (no change), so no code step. The sole non-default option A begins with "one more bounded design revision" (docs under `design_docs/**`, in scope) plus a retro-analysis, then re-quorum, then plan/build/judge. Charter screen's examples are code steps "that must ship alone first" (a scope-guard arm). A's first step is a design revision to a doc that already failed two reviews. Does not qualify on the charter's intent; a purely literal reading ("an in-scope step that has no diff") is arguable, see the argument against NONE.
- D-FLEET-46: A = bookkeeping resolves of tickets whose fixes landed attended; no fleet diff; it is the permission the row asks for. No.

## Strongest argument against NONE

The degraded-fire bar is a fleet-controller choice (`33190af48`, iteration 91), not a ruling, and the Gate-5 file is in scope. World now has a measured consecutive identical-input pair (Q3) in which only the bar stopped the record, and 7 slots are lost. A draft PR of D-FLEET-45 A's design revision (+ 10-fire retro-analysis) is a docs-only, no-merge artifact that costs nothing to review. Why NONE still holds: (1) the row recommends B and the screen is for rows whose *recommendation* is code; building A pre-empts the ledger; (2) the design failed quorum twice, and both round-2 defects (proof block ignores ledger-parser exit status; no recorded World fire qualifies) are independent of the 418 evidence; (3) the ticket is ruling-gated by the loop's own earlier decision and Mark has not ruled. Second argument (resolving row 58 / `claude-p-ration` / D-FLEET-46 tickets as bookkeeping) is rejected: it is exactly the act D-FLEET-46 asks permission for.

## Q3. World 418's claim, first-party

Source: `grep "idle snapshot taken" /tmp/ailang-mission-world.log` (local time UTC+1).

- 417 (11-20 06:39:02 local = 05:39Z): `0355e1e27457a8d866647f30f4c831a2de38416194843062384ce5af5e2c9454 head=0e9e8d3 ledger=107 open=2 directives=0 inbox=0 red=0`
- 418 (11-20 10:39:38 local = 09:39Z): `0355e1e27457a8d866647f30f4c831a2de38416194843062384ce5af5e2c9454 head=0e9e8d3 ledger=107 open=2 directives=0 inbox=0 red=0`
- **Byte-identical: CONFIRMED.** (Same hash also at 11-19 02:39 local, 27 h earlier, past expiry; 6 differing snapshots between. So 417->418 is the first *consecutive* identical pair since the head pathspec fix `ff08818ae`.) Repeated-hash census over 82 snapshots: `40d76c3d` x11 (11-08 01:48 -> 11-09 17:49 local), `145eb5c6` x18 (red=1, not no-pick), `cba752ed` x2, `0355e1e2` x3.
- **Would a recorded 417 have let 418 exit idle?** Input check: `idle-fingerprint.sh` hashes {head outside record paths, ledger statuses, directives, unread inbox, red checks}; the 24 h expiry is `mission-control.sh:1773` (`_mi_age -gt 86400`). Record age at 418's pre-check (~10:38 local) vs 417's snapshot `at=` ~05:39Z: ~4 h, inside the window. 418's own log (`world-mission-log.md` entry 418) says its range `e7c49228..9e05da55` holds only 417's record, 3 commits, 0 paths outside `design_docs/`, 0 ledger rows changed; the identical hash confirms the fingerprint inputs did not move. So the fingerprint comparison at `mission-control.sh:1789` WOULD have matched: yes for the hash arm. **Not measured:** that 417 met the remaining Gate-5 record conditions (no PARKED-ON-LANE / clock wait). `grep PARKED-ON-LANE world-mission.md` finds 4 hits, all marked RESOLVED/LANDED rows; World 417's own text asserts "an undegraded fire would have recorded" (a claim, not a measurement). `~/.ailang/state` holds only `.pending` snapshots for world/stapledon, no record file: no product mission has ever promoted one.
- **Does this overturn iteration 127's "a recorded 416 would not have idled 417"?** No. 416->417 still differs (inbox 1 -> 0; hashes `01aae166` vs `0355e1e2`). Both statements hold: only the 417->418 pair would have skipped.
- **Counterfactual over the 9 World fires 11-19 02:39 -> 11-20 10:39 local** (hash sequence 0355e1e2, 15d7c16b, 5067a493, 56c2e062, f8eacc0f, 4d83abf8, 01aae166, 0355e1e2, 0355e1e2): even if every one recorded, exactly 1 (418) would have skipped. Before `ff08818ae` head moved with every record commit, so none could. Forward-looking, with head now stable, the only wake sources are inbox churn (fleet replies/acks, ~2 of 9) and the 24 h expiry.
- **Older evidence for the ticket's value:** `40d76c3d`, 11 identical snapshots over ~40 h (World 363-370 era, every one fell back). With 24 h expiry a recorded chain would need ~2 full fires, not 11.
- **Did 418's controller name only dev-side OPEN ledger rows as its gate?** Partly. Entry 418: `--open` -> D-WORLD-119, D-WORLD-120 (dev-side OPEN rows; row 243 V4 S3-S12 gated on them). But it also lists row 150 (waits on "the attended A1"), row 151 (needs row 188's N2d/N4a), rows 217-219 (attended track), rows 258/259/261/265 "under rule (d)", and row 263 (waits on the fleet). Rows 150/151 unblock by attended work/commits, not necessarily a ledger flip. Whether the narrowed rule's "only unblock event is a ledger flip" holds for World 418: UNMEASURED (appears likely only for the 119/120-gated rows).
- **Self-healing signal (World's own assumption, not measured):** driver log 10:38:22 `Codex quota window: state=over window=10080m used=17.0% allowed=14.9%`; 418's text says the gap may close in ~3.5 h if `allowed` grows linearly, so a later fire could run undegraded and record. This fleet fire is still over ration (per brief), so no confirmation.

## Q4. Change D-FLEET-45's recommendation (B -> A)? NO.

Reasoning: (a) The new evidence is one consecutive identical pair; it falsifies World 416's ticket-text "only reason" claim less than it confirms a narrow slice of the cost: 1 of 9 fires recoverable in the last day, because inbox churn and the 24 h expiry also bind. (b) It does not touch either round-2 defect that blocked the design twice, nor the UNMEASURED "only a ledger flip unblocks" test, nor the unmet requirement that a retro-analysis find a qualifying fire. B's stated premise (the rule would not engage for the filing mission) is now weaker (head is stable; 110-119/120 is a dev-side ledger gate), but weaker is not refuted. (c) The cost to flip is small though (a text edit), so the honest action is to keep B and add the 418 measurement, flagging that Mark may reasonably choose A: the forward case (head stable, 5 of 6 fires per 24 h potentially skippable absent inbox churn) is better than the backward 1-of-9 count. Suggested addendum to the row text: "Iteration 128: World 418's snapshot is byte-identical to 417's (`0355e1e2`, 4 h apart, inside the 24 h expiry); a recorded 417 would have let 418 idle (hash arm measured; other Gate-5 record conditions unmeasured). 416->417 still differed. Of the 9 World fires 11-19 02:39 -> 11-20 10:39 local, 1 would have skipped. 7 slots lost. Recommendation unchanged (B)."

## Q5. Pick

**NONE**, confidence 75%. Residual 25%: the literal-reading risk that D-FLEET-45 A's design revision counts as "an in-scope step with no diff". If the controller disagrees, the bounded alternative is a docs-only draft PR of the revised design (no merge, no Gate-5 skill edit), not a skill change.

## UNMEASURED

- Whether World 417 met every Gate-5 record condition (lane park / clock wait) beyond the degraded bar.
- Whether the narrowed rule's "only unblock event is a dev-side ledger flip" passes for World 410-418 candidates (rows 150/151/263 are non-ledger unblock paths).
- Whether the ration gap actually closes before the next World fire (World's linear-growth assumption).
- Whether merged fixes cover their tickets' repros at HEAD (D-FLEET-46 A's step).
