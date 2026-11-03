# Fleet iteration 80 — D-FLEET-21 option A proposed diff

Base: detached `29ddaec78` (`origin/dev`); initial worktree clean. Read the whole decision row at `design_docs/fleet-mission.md:170`: OPEN, unanswered default C. This is a proposal for a DRAFT PR only; no merge, ruling update, implementation, or executor handoff before Mark rules.
Scope: replace seven existing lines in EACH Gate 3b resource, one existing line in EACH SKILL.md, and add one changelog fragment (five proposed PR files). Registry reuse: none, documentation-only. This root plan is the planner handoff artifact.

Controller measurement supplied 2026-11-03 (not independently remeasured): last 40 completed CI runs p50 14 min, p90 18, max 30; recent fleet PR full check-set spans 14–15 min, slowest leg test-windows. Over ~200 runs (10-28..11-03), 23 exceeded 30 min: 10-29 had 18/90, max 99 min (backlog day); 11-02 had 2/47, max 37 min. Durations are updatedAt−createdAt, including queue wait. The row's older evidence is iterations 43–44 (10-16, 31–55 min). The median comment was reasonable; the 30-min cap misses the tail.

## Exact BEFORE → AFTER replacements

Apply every pair below identically to `.claude/skills/mission-control/resources/gate-3b-ci-green.md` AND `.agents/skills/mission-control/resources/gate-3b-ci-green.md`. Line numbers refer to base. Minus/plus prefixes are diff notation; all other characters, including indentation, are literal.

```diff
@@ line 16 @@
-The Bash foreground ceiling is **600 s**; retain the poll's **1800 s** cap (iteration 13's **4 h** wedge).
+The Bash foreground ceiling is **600 s**; retain the poll's **3600 s** cap (iteration 13's **4 h** wedge).
@@ line 19 @@
-Bash poll call: run_in_background: true, timeout: 2000000 (ms).
+Bash poll call: run_in_background: true, timeout: 3800000 (ms).
@@ line 20 @@
-These are **Bash tool arguments**, not shell variables or GNU timeout; keep timeout at least 2000000 ms (cap plus margin).
+These are **Bash tool arguments**, not shell variables or GNU timeout; keep timeout at least 3800000 ms (cap plus margin).
@@ line 47 @@
-deadline=$(( $(date +%s) + 1800 ))    # 30-min cap; CI is ~15-20m — never open-ended
+deadline=$(( $(date +%s) + 3600 ))    # 60-min cap; CI median ~15 min, tail past 30 min — never open-ended
@@ line 52 @@
-  [ "$(date +%s)" -ge "$deadline" ] && { echo "Gate 3b TIMEOUT for $target after 30m (status=$st) — PARK, do not hang" > "$verdict"; break; }
+  [ "$(date +%s)" -ge "$deadline" ] && { echo "Gate 3b TIMEOUT for $target after 60m (status=$st) — PARK, do not hang" > "$verdict"; break; }
@@ line 65 @@
-The poll's 30-minute deadline bounds this chain. Reject a mismatched full SHA; completion without a verdict or tool failure means report/PARK, never green or endless reads.
+The poll's 60-minute deadline bounds this chain. Reject a mismatched full SHA; completion without a verdict or tool failure means report/PARK, never green or endless reads.
@@ line 68 @@
-The poll runs background-only because a foreground call cannot hold the 1800 s cap and re-entering the snippet would restart the cap.
+The poll runs background-only because a foreground call cannot hold the 3600 s cap and re-entering the snippet would restart the cap.
```

Apply this single-line replacement identically to `.claude/skills/mission-control/SKILL.md` AND `.agents/skills/mission-control/SKILL.md`; do not reflow or add lines. Rule 6's next line, `   failure, not an error to retry in place.`, stays unchanged.

```diff
@@ line 323 @@
-   one unbounded wait burns the entire 6h slot. Default cap ≤30 min; treat expiry as a parkable
+   one unbounded wait burns the entire 6h slot. Default cap ≤30 min (exception: Gate 3b CI poll ≤60 min); treat expiry as a parkable
```

Each SKILL.md remains 628 lines (delta 0). `scripts/check_context_docs.sh` uses a 500-line default and the existing `.claude/skills/mission-control/SKILL.md 2781` exemption in `scripts/context_docs_baseline.txt`; 628 is within that ratchet. It checks `.claude` skills, not `.agents`; cmp covers the latter. Do not edit/bump the baseline or use its slack to add lines.

## Changelog fragment — full text

Path: `changelogs/unreleased/2026-11-03-gate-3b-ci-poll-cap.md`. Matches `changelogs/unreleased/README.md`; no root CHANGELOG or active release-note edit.

```markdown
### Changed — Gate 3b allows a bounded 60-minute CI poll (2026-11-03)

Raise the mission-control Gate 3b CI poll deadline to 3600 seconds and its background Bash tool timeout to 3800000 milliseconds in both skill mirrors. CI has a median around 15 minutes, with a tail beyond 30 minutes including queue wait. Standing rule 6 names Gate 3b as an exception to its 30-minute default; expiry still parks the item for review.

Decision: [D-FLEET-21 option A](../../design_docs/fleet-mission.md).
```

## Verification arms and mutations killed

Run from worktree root. Commands are observation-only except disposable gate scratch, which MUST stay inside this worktree via `TMPDIR="$PWD"`; never run sync without `--check`, changelog fold without `--check`, actual CI polling, or mutation writes. Mutation descriptions are hypothetical negative controls, not instructions to modify the checkout. Base readings below were obtained with cheap read-only commands only.

- **Mirror arm:** `cmp .claude/skills/mission-control/resources/gate-3b-ci-green.md .agents/skills/mission-control/resources/gate-3b-ci-green.md` AND `cmp .claude/skills/mission-control/SKILL.md .agents/skills/mission-control/SKILL.md`; require rc 0 for EACH. Kills an edit applied to only one mirror. Base: both rc 0.
- **Broader mirror arm:** `/bin/bash tools/launchd/sync-agents-skills.sh --check`, require rc 0. Kills other mission/sprint mirror drift; base rc 0 across all six covered skills. Existing `test_agents_skills_sync.sh` also has a drifted-copy negative control and frontmatter checks; do not change it.
- **Stale-value arm:** for EACH Gate 3b copy run `git grep -n -E '\+ 1800|after 30m|2000000|30-minute deadline' -- <file>`; require rc 1 (no matches), treat rc >1 as error. Extend with `git grep -n -E '1800|2000000|30-min|after 30m' -- <file>`, also rc 1. Kills stale values even when identically wrong in both mirrors, including lines 16/68 missed by the narrower pattern. Base: narrower grep rc 0 with five lines/file (19,20,47,52,65), extended rc 0 with seven (16,19,20,47,52,65,68): expected RED for proposed-state checks, not a base workaround.
- **Positive-value arm:** for EACH copy, `git grep -n -E '3600|3800000' -- <file>` must return rc 0 with five lines (16,19,20,47,68); individually assert both tokens present with separate `git grep -q -F` calls. Kills deletion of the cap or tool-timeout guidance; base combined grep rc 1, zero matches/file. The corrected comment's `past 30 min` is evidence, not a residual 30-min poll cap.
- **Exact-diff arm:** read-only `git diff --unified=0 29ddaec78 -- .claude/skills/mission-control .agents/skills/mission-control` must contain EXACTLY the eight replacement pairs above per mirror, no other hunks; inspect the untracked fragment separately against its full text above. `git diff --check` must return rc 0. Kills extra poll-structure edits, incorrect values such as 3601/3800001, changing the duration claim, losing the rule-6 exception, or relaxing other defaults; presence-only greps cannot prove these. Planner artifact is outside this implementation-diff check.
- **Context gate:** `TMPDIR="$PWD" make check-context-docs`, require rc 0; also `wc -l` both SKILL.md copies must report 628. Kills line growth above the planned zero delta; the make gate alone permits growth up to its grandfathered baseline, so the exact line count is necessary here. Base: size observation passes (628 ≤2781); full make rc UNMEASURED, deferred to controller because the planner may write only this plan and the gate creates scratch files. Do not claim a full-gate pass; report any inherited red without editing unrelated files.
- **Relevant driver subset (controller, outside planner sandbox):** `TMPDIR="$PWD" /bin/bash tools/launchd/lib/run-suites.sh 1 tools/launchd/test_mission_heartbeat.sh tools/launchd/test_agents_skills_sync.sh`, require rc 0. These are the relevant `LAUNCHD_SUITES` in `make/test.mk`, not the full `make test-launchd-drivers`. Heartbeat lines 217–228 read gate resources and require `stamp gate-3b` (8/8 stamps overall); removing it is the mutation killed. Sync suite kills one-sided edits and invalid frontmatter. No test found pins these poll numbers/phrases; these suites do not validate the new timeout values. Base: Gate 3b stamp present in both copies; suites NOT RUN (scratch/process spawning; heartbeat starts a background child). Socket-touching sandbox results would be UNINFORMATIVE UNDER SANDBOX.
- **Fragment arm:** `TMPDIR="$PWD" make check-changelog`, require rc 0; inspect that the new fragment opens with exactly the specified `### Changed — …` and body/link. Kills malformed fragment name/heading or misplaced root release notes. Deferred; no fragment exists on base.

Search basis: `git grep` across both gate resources, rule 6, make targets, shell/Go/Python/TypeScript tests, scripts and CI workflows; read the sync test/helper, heartbeat test, context gate/baseline and changelog convention. All seven known Gate 3b lines were found; no additional CI-poll 30-minute/1800 wording was found. Historical timeout war stories are unchanged. `make/code-health.mk:229` runs the context gate; `make/test.mk:130–149` dispatches driver suites.

## Boundaries, risks, and delivery

Do NOT change `.claude/skills/mission-control/resources/gate-3-route.md:160` or its `.agents` mirror: `2000000 ms for 1800 s` describes the codex EXECUTOR wrapper's independent 30-min cap, not Gate 3b. Preserve the 600 s foreground ceiling, background-only launch, `/bin/bash` requirement, full-SHA target/read and drift checks, selected run id, 30 s cadence, terminal verdict file, fresh paths, 45 s read deadline/5 s cadence/60000 ms read timeout, direct-confirmation rules and Gate 3b stamp. Lines 328–332, including `On timeout, do NOT keep waiting`, remain byte-for-byte unchanged: PARK/needs-human-review and report, never automatic re-poll or false green.
A 3600 s poll consumes up to 1/6 of a 6 h slot, 30 min more than before; it does not extend the slot/watchdog or guarantee green before expiry (measured max 99 min still exceeds it). The Bash background ceiling is 7200000 ms per the decision/task; 3800000 ms is below it and gives 200 s margin over the 3600 s deadline. Poll/API latency can overshoot the shell deadline; the outer tool timeout remains necessary. Keep standing rule 6's ≤30-min default for every other wait, its hard-ceiling requirement and park-on-expiry semantics.
Delivery: one small documentation-only draft PR with the five proposed files; keep D-FLEET-21 OPEN and do not merge until Mark rules. No new tests, scripts, runtime code, harness settings, slot budgets, baseline changes or unrelated sprint work.
