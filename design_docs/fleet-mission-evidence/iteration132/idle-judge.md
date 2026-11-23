# Iteration 132 evaluator report

Judge: sonnet via the Agent tool (`MISSION-ROLE: evaluator`, foreground, read-only). This was the third spawn attempt; the first two ended in API 529 Overloaded before any output. 70,479 tok, 9 tool uses, 58 s. The text below is the judge's report verbatim.

**VERDICT: NONE**
**CONFIDENCE: 88%**

I could not refute the controller. All five claims hold, with two small discrepancies on claim 4 and the log wording (see Discrepancies).

## Claims

**1. CONFIRMED.**
- **Command:** `diff <(jq -S . iteration131/tickets.json) <(jq -S . iteration132/tickets.json)`.
- **Result:** the only differing ticket is `skill:idle-record-barred-on-degraded-no-pick-fire`.
  - `last_seen` moved to `2026-11-23T07:20:43Z`.
  - `slots_lost` went 10 to 12.
  - Two message ids were added: `inbox_1795418443750_1004eee2` and `inbox_1795404296951_40d30714`.
  - The latest evidence is World iteration 424, degraded (codex 39.0% used vs 32.9% allowed).
  - The slot verdict now reads PAUSED-NO-CAPACITY at 04:27 local, which is the 423 fire.
- **Live re-run:** `ailang mission ticket open --json | jq -S .` against the iteration132 file produced no diff (`LIVE_SAME`). There are 16 tickets.

**2. CONFIRMED.**
- `scripts/mission_decisions.sh --check` printed `decision ledger valid: 46 rows`.
- `open-decisions.tsv` lists exactly D-FLEET-44, 45 and 46 as OPEN.
- `git log --oneline 21a39fb90..origin/dev -- design_docs/fleet-mission.md` printed nothing.

**3. CONFIRMED.** The mapping below re-derives from the charter and sums to 16.

| Disposition | Count | Tickets (charter row) |
|---|---|---|
| Await D-FLEET-46 | 8 | `quorum:gpt6-1-sol-openai-api-credits-exhausted` (21), `pi:openrouter:uncapped-max-tokens-402` (50), `executor:codex-credential-collision-drops-automatic-planner-handoffs` and `executor:codex-credential-lease-collision-fails-concurrent-dispatch` (54), `coordinator:job-retry-success-dropped-after-failed-completion` (55), `executor:3h-timeout-from-queue-and-unsliced-sprints` and `executor:timeout-loses-work-and-may-never-start` (56), `quorum:claude-p-ration-usage-429` (68) |
| Out of scope | 5 | `reporting:daneel-reports-not-declared-triage` (24, prod config), `messaging:pubsub-notify-wrong-project` (18, rig quota project), `crash:sprint-executor:sigbus-ram-tmp-full-make-test` (27 / D-FLEET-23, executor volume), `coordinator:cloud-lane-capped-by-local-worktree-limit` (52, coordinator), `ci:distinct-build-test-failures` (44, D-FLEET-17 class) |
| Folded into ruled D-FLEET-18 | 1 | `eval:motoko-connection-probe-orphaned` (58) |
| Parked | 2 | `tool-hang:triage-judge:home-metadata-search` (61, D-FLEET-44); `skill:idle-record-barred-on-degraded-no-pick-fire` (62, D-FLEET-45) |

The total is 8 + 5 + 1 + 2 = 16.

**4. CONFIRMED, with one discrepancy.**
- **Evidence:** `world-idle-snapshots.txt` has 11 lines carrying `1ac80c01…` (head `95a8775`).
  - The first is World fire `world-1795274220` at **2026-11-21 16:17 local**. The controller says 20:16 local, which is the second snapshot; this does not change the conclusion.
  - The last is the 08:16 fire on 11-23 (iteration 424).
  - The first 10 of these fires ended PAUSED-NO-CAPACITY (pi controller `glm-5.3:cloud`, at gate-2 or `fired`). 424 is COMPLETED, 3258 s, controller opus.
- **Eligibility to record:** none of the 10 paused fires was eligible. They never reached Gate 5, and the pi fallback also counts as a degraded fire.
- **Reading of 424:** it was not skippable, because no earlier fire had recorded `1ac80c01`. It is not a slot the degraded bar cost. It was degraded (probe rc 75) and so could not record.
- **Next fire:** the next identical fire will therefore run in full while codex stays over ration. This confirms your reading.
- **The 423 fire (04:16):** it wrote a full no-pick record and then hit an Ollama 429. Iteration 424 recovered the stranded record via PR #634. Even under D-FLEET-45 option A, the retained "unfinished-gate" bar would have barred it. Option A therefore would not have saved 424 either.
- **Before the head moved:** three fires exited IDLE on 11-21 (05:38Z, 09:38Z, 13:38Z). That shows recording works when undegraded.

**5. CONFIRMED.**
- **Fire died:** the fleet's 11-21 15:15 fire died with "You've hit your weekly limit", slot verdict `CRASHED at=gate-4`. The 16:17 fire ran as pi controller, hit an Ollama 429, and paused at gate-3.
- **Unpushed worktree:** `.wt-fleet-iter132-judge` exists. Its `idle-judge.md` reads NONE, 90%.
- **Remote check:** `git ls-remote origin 'refs/heads/fleet/i132*'` printed nothing.
- **Plan:** crediting it in the log without adopting its verdict is consistent with the rules.
- **Side note:** that judge file mentions "423, 424, 425, 426 idled". I did not check this and it does not bear on NONE.

## Screen of OPEN rows for a routable first step

- **D-FLEET-44 A:** the first step is requesting a corrected scope-limited design plus duplicate-gate reconciliation.
  - It is policy-adjacent: it rests on the role-artifact contract and standing-rule compliance.
  - Even afterwards, "a later explicit design approval is still required."
  - The default is B "immediately; no planning/execution/merge".
  - Not routable.
- **D-FLEET-45 A:** a further design revision plus a re-quorum.
  - The row's own recommendation is B, and the judge in iteration 131 weakened A further.
  - Not routable.
- **D-FLEET-46 A:** the first step is resolving tickets "one per fire, after re-checking at HEAD".
  - The re-check is mechanical, but the act it leads to, resolving tickets, exists only under a ruling.
  - The default is B.
  - Not routable.
- **Result:** no row begins with an in-scope step that needs no ruling.

## Strongest counter-argument

D-FLEET-44's text recommends A and names it an in-scope design revision, so the fleet could "begin" revising the design without a ruling. This does not survive:
- The row's own default is B until Mark answers.
- A revision is not an independent mechanical step. It feeds a request that still needs explicit design approval.
- Charter row 61 and iteration 96 already treated that design as PARKED after a blocked quorum.
- I found no new evidence this iteration that changes that.

A second, weaker counter: D-FLEET-46 is eight bookkeeping resolves with merged SHAs. It is still ruling-gated, and doing it would resolve the fleet's own pending decision on its own behalf, which the rules forbid.

## Discrepancies with the controller's numbers

- **First `1ac80c01` snapshot:** the first one is 11-21 16:17 local, not 20:16. The earlier time makes 10 pre-424 fires, not 9 or fewer, but it changes nothing.
- **Fire counts:** the log says "3 fires exited idle and 10 paused" between iterations 422 and 424. That matches the slot-verdicts file (3 IDLE, then 10 PAUSED/CRASH-type lines before the COMPLETED one).
- **Everything else matches:** 16 tickets, `slots_lost` 12, 46 valid rows, D-FLEET-44/45/46 open, no `design_docs/fleet-mission.md` commits since `21a39fb90`, and an empty `fleet/i132*` ref list.

## Recommended Next

When the next identical World fire runs in full under the codex over-ration condition, count it as a 13th slot on the D-FLEET-45 ticket. Do not change the D-FLEET-45 recommendation.
