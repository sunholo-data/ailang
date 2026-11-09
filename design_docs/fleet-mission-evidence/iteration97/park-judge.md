VERDICT: PARK-UPHELD

Independent evaluator, fleet iteration 97. Judges the controller's disposition only. All commands run read-only; scratch work in /tmp/i97j.

## 1. Are the round-2 rejections real?

**(a) Sol: the block can print OK when mission_decisions.sh fails. REAL.**
Reproduction under /bin/bash with a stub that prints two ids then exits 3 (/tmp/i97j/stub.sh), run through the doc's exact `open=$(... | cut -f1)` and `for id` lines:
```
branch: dev open rows: 2
OK D-X-1
OK D-X-2
```
The pipeline `stub | cut` reports cut's status, so the parser's rc is lost. Nothing in the block checks it. D1(b) says a failed ledger read bars the record, and the block does not enforce that. V19 only tested a bad workdir (branch read fails), never a parser failure after output. Sol's proposed_fix (capture `rows=$(...)` with an `if !` guard, plus parser-failure tests) is concrete and mechanical.

**(b) Kimi: no recorded fire would have recorded. REAL in substance; its "n=2" detail is slightly harsh.**
- ~/.ailang/state has only the two `.pending` snapshots, no promoted file (ls output: world and stapledon `.pending` only).
- World log on origin/mission/world-iter370-record, `design_docs/world-mission-log.md` lines 1613 and 1656, both say "NOT recorded ... degraded". The ledger row 259 on that branch cites iterations 363-370 as eight consecutive ADMIN/no-pick fires, and log 366 as a third instance.
- Derivation under D1/D2: the iteration-370 STATUS says release rows 150/151 are "not loop-routable (row 188 N2d/N4a re-measured 0 since 2026-11-01)", "Upstream predicates 58/58 closed (controls fired)", and a "Weekly external-issue sweep". These are external predicates and a clock-like duty, none hashed by the fingerprint. D2 says an external predicate or clock beside a cited row means "no", and D1(d) bars clock waits. The doc's own Risks section concedes this fire "would still not record", and also that D-WORLD-113 exists only on the branch (V12), so it would not qualify as a cited row.
- Stapledon: the doc never read it (its "Not verified" list). I could not trace its no-pick fires either. Its slot log for 11-06..09 shows COMPLETED fires of 487-17237 s and three PAUSED-NO-CAPACITY, but nothing says which were no-pick. So "zero would record" is shown for World only, and stapledon is unknown. The doc offers no positive case at all.
- V11 numbers reproduce exactly (8 COMPLETED 11-07..11-09: 8931, 6080, 5035, 6050, 4739, 4769, 4104, 3605). But those with stamps=10-14 and a codex controller are not shown to be no-pick fires, so "each no-pick fire runs 1-2.5 h" is only established for 369/370.
- The observation criterion's second branch ("each report shows which D2 or D1 bar fired") is satisfied by today's behaviour, so Kimi's "unfalsifiable on a no-op" point is correct.

Other doc facts I checked and found right: 12 mutants in the loop (line 28); idle-fingerprint.sh lines 58-68 do mirror the block's branch resolution; retro lines 13-17 are the paragraph to replace; snapshot/pre-check ranges in mission-control.sh match.

## 2. Does the NARROW-REFINEMENT CARVE-OUT hold? NO.

Carve-out text (gate-2-pick.md:236+): EVERY remaining blocking objection must carry a concrete reviewer-authored `proposed_fix`, and none may dispute the design DIRECTION, and none may need controller judgement.
- Sol passes alone: concrete fix (guard `if ! rows=$(...)` and tests), completeness-class defect.
- Kimi fails all three limbs:
  - Direction: "A design whose entire measurable benefit is conjectural ... fails premise verification." That challenges whether the change is worth landing.
  - Fix concreteness: fix 1 is "Add a V-row that retro-applies D1/D2 ... If zero would have, re-scope the doc ... or do not land." That is an investigation with a conditional abort, not verbatim text.
  - Controller judgement: fix 3 says "cited canonical list ... if no such list exists, define it in this same edit", which is controller-invented content. Fix 2 needs the controller to pick a falsifiable threshold.
- Applying only Sol's fix would "override" Kimi's objection, which the carve-out forbids ("never overriding an objection"). The carve-out also needs first-use ratification by Mark, which has not happened (per the skill text, ratification was surfaced at iter-95), so it could not be applied silently anyway.

## 3. Judgment park vs PARKED-ON-LANE vs SPLIT

Standing rule 8 (SKILL.md ~555-600): a judgment park waits on a human call; PARKED-ON-LANE is for capacity (quota, outage) and resumes on a clock/probe rc=0. Here both round-1 and round-2 quorums ran with `absent_reviewers: []` (JSON), so no lane failed. The blocker is whether the change is worth landing, which is judgment. The gate-2 QUORUM-AT-PICK text itself mandates "still-rejected -> needs-human-review, park, next item". The rule-8 tell (a reset time, quota, or "no other route" in the sentence) is absent. No SPLIT is warranted: the Sol fix is tiny but moot if the human picks B. Classification correct. The one-word ask (A/B) fits DECISIONS-row format, and D-FLEET-44 (iteration 96) is the same pattern.

## 4. Policy class (HD-2a)?

Charter Authority: HD-2a covers "routing, lane order, quota or ration thresholds, or a billing guard". This change edits skill text and a launchd test only (both on the may-change allowlist: `.claude/skills/mission-*/**`, `tools/launchd/**`). It touches no lane walk, ration or guard. It is NOT HD-2a policy class. A park is still right, but for design-validity reasons, not policy reasons. The doc's D4 reasoning holds. One caveat: it loosens a safety bar for product missions (World/stapledon), so a human decision is sensible even outside HD-2a, which the park already provides.

## 5. Missed items

- Cheaper correct disposition: not really. The doc's own Risks section says the change would not engage for World, whose fires motivated the ticket. B plus a ticket reply is the minimal action; the real fix is on World's lane (record the attended step as a dev-side ledger row), which could be told to World via the ticket resolution.
- The A option as phrased ("authorize a bounded further revision applying the round-2 fixes then build") hides that Kimi's fix 1 may conclude "do not land". The DECISIONS ask should say A may end in no-build, and set B as the default if unanswered (matching D-FLEET-43/44 style).
- The prior iteration-95 carve-out ratification status was not checked by me beyond the text; the controller should not rely on it.
- The Sol block defect is also an independent small bug in the doc (fail-open), worth noting in the park row so A's reviser has it.
- The park text should not claim stapledon behaviour; the doc has none.

Strongest objection to the controller's disposition: the park spends a human decision slot on a design whose own Risks section says it would not engage for the one mission that filed the ticket, so a cheaper disposition is to resolve the ticket as "won't change; World to add a dev-side ledger row" with B as the default, rather than offering option A at all. The human-queue cost (rule 8) is borne for a branch with no demonstrated qualifying case.
