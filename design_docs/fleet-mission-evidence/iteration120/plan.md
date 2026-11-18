# Fleet iteration 120: re-land the D-FLEET-36 A + D-FLEET-38 A scope-guard arm

Planner output, 2026-11-18. Base `origin/dev` = `022f68f1a`. Worktree `/Users/voightkampff/.ailang-driver-pin/fleet-i120-wt`,
branch `fleet/i120-dfleet36-38-scope-arm`, left clean at `022f68f1a` after planning.
Rulings: D-FLEET-36 = A and D-FLEET-38 = A, Mark, attended 2026-11-16 (ledger rows at `design_docs/fleet-mission.md:185` and `:187`).

## 0. Verdict summary

- **`git cherry-pick bcf7f97ac 261fb3b64` does NOT apply cleanly.** `bcf7f97ac` conflicts in
  `design_docs/fleet-mission.md` and `tools/launchd/test_mission_scope_guard.sh`. `pre-push` and the
  changelog fragment auto-merge. Both conflicts are adjacent-insertion conflicts against #2382's D-FLEET-18 lines
  (`97e792147`), and a union resolution (HEAD side first, then the cherry-picked side) is correct. Once
  `bcf7f97ac` is resolved that way, `261fb3b64` applies with no conflict, and `test_mission_scope_guard.sh` passes.
  Tested in the worktree, then undone with `git reset --hard origin/dev` (worktree only).
- **Do not ship the cherry-pick as-is.** The executor applies the reviewed patch
  `/tmp/fleet_i120/final-022f68f1a.patch` (`git apply --check` is clean on `022f68f1a`). The patch is the
  cherry-pick plus five edits: the fragments are re-dated and say "ruled", not "PROPOSED"; the Authority lines are reordered; the pre-push comment
  says "rename-only or identical"; one duplicate `.claude/skills/use-ailang` row is relabelled; and a new
  exactly-16 refuse loop is added. The loop is the only behaviour change to the test.
- **The arm stays at 16.** The controller's reading holds. Section 2 has the evidence and one correction.
- `bash tools/launchd/test_mission_scope_guard.sh` passes with the patch. `make test-launchd-drivers` reports
  rc=0, 58/58 suites, in 386 s. 17 mutants were run: 17 killed by the final suite, 0 survived. Three of them survive the PR #2108 suite.

## 1. How to build it (executor steps)

```bash
cd /Users/voightkampff/.ailang-driver-pin/fleet-i120-wt
git status --short                                    # must be empty, HEAD 022f68f1a
git apply /tmp/fleet_i120/final-022f68f1a.patch
git add -A
git commit  -m "fix(fleet): scope guard admits design-doc-creator and 16 .agents skill mirrors (D-FLEET-36 A, D-FLEET-38 A, ruled 2026-11-16)"
```
If the patch is missing, use the alternative. Run `git cherry-pick bcf7f97ac`, resolve the two conflicts by keeping
BOTH hunks (HEAD's D-FLEET-18 lines first, then the D-FLEET-36 lines), then run `git cherry-pick 261fb3b64`. After that,
hand-apply the deltas listed in section 3. Two commits are fine, and so is one. The commit message must carry the
attribution trailer (`Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`).

Pinned-guard check (the iteration-16 rule): every file this PR touches is ALLOWED for `MISSION_NAME=fleet` by the
CURRENT `origin/dev` guard. It was verified by sourcing `git show origin/dev:tools/launchd/githooks/pre-push`:
the 2 changelog fragments (support), `design_docs/fleet-mission.md` (harness), and `tools/launchd/githooks/pre-push`
and `tools/launchd/test_mission_scope_guard.sh` (harness). The push is therefore not blocked by the guard it changes.

## 2. Re-measurement of the controller's claims (independent)

Script: `/tmp/fleet_i120/remeasure_planner.sh`, output `/tmp/fleet_i120/planner-remeasure.txt`. It uses the same normaliser as
`/tmp/fleet_i120/norm.sh` and classifies every `.claude/skills` dir, with a separate IDENTICAL class (cmp-equal on all files).

| Claim | Result |
|---|---|
| 14 of the 16 named skills are rename-only at HEAD | **Confirmed.** benchmark-runner builtin-developer cloud-cascade-debug coordinator-helper eval-analyzer eval-gap-finder gemini-cli-helper github-issue-triage local-model-onboarding local-ollama-eval parser-developer pr-monitor prompt-manager use-ailang. None has a one-sided file. |
| collaboration-hub and trace-debugger are byte-identical | **Confirmed**, with a **correction on cause**: both became identical at **`a6868eecb`** (2026-11-10, "remove the web UI"). That commit edited both copies of both skills; `a6868eecb^` differs and `a6868eecb` is identical for each. `f9ffb99c8` (collab `rest_api_reference.md`) and `87f77e13e` (trace-debugger) came later and kept them identical by editing both sides. They did not create the identity. |
| totals | paired=41: identical=17, rename-only=14, content=10 (content: agent-inbox ailang-packages benchmark-manager cli-doc-maintainer design-doc-creator docs-sync microrag model-manager motoko-analyzer post-release). `.claude`-only: perf-sweep, simplicity-audit. `.agents`-only: none. |
| 8 `grep -E "^\d+\."` sites still at HEAD | **Confirmed**: `.claude` create_planned_doc.sh:128/137/154/163, `.agents` :125/134/153/162. |
| dev CI red on `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` | Not re-measured (outside fleet scope). The PR touches non-docs paths, so the `changes` job sets code=true and the full matrix runs. The PR will inherit that red. It also sets shell=true (`tools/launchd/`), so `launchd drivers (bash 3.2)` runs. |

**Should the arm still carry exactly 16? Yes.** Reasons:
1. The ruling text is explicit. It says "`_scope_is_fleet_support` gains exactly those 16 `.agents/skills/<name>/` paths" and names
   collaboration-hub and trace-debugger. Changing the count would mean departing from an attended ruling, and the fleet cannot do that.
2. The ruling's criterion is "a verbatim `rsync --delete` from `.claude` loses nothing". `sync-agents-skills.sh` is verbatim
   (rsync), not a rename. An identical pair meets that criterion trivially, because the sync is a no-op today.
3. The arm is needed for identical pairs specifically. After the follow-up adds them to the `SKILLS` list, `--check`
   (run by `test_agents_skills_sync.sh` inside `make test-launchd-drivers`) goes red as soon as an attended commit edits only
   the `.claude` copy. The fleet then has to push the regenerated `.agents` copy. Without the arm it could not.
   Identical now does not mean identical later. `a6868eecb`, `f9ffb99c8` and `87f77e13e` show attended work editing these files.
4. Mutant M2 (delete the collaboration-hub arm) is killed, so the test pins the identical pairs too.

**Related finding (not in this PR; for attended):** `release-manager` has been in `sync-agents-skills.sh`'s `SKILLS` list since
`95061c37f`. The guard refuses `.agents/skills/release-manager/**` AND `.claude/skills/release-manager/**` for
the fleet, which was checked by sourcing the guard. The fleet cannot push a release-manager re-sync even though the sync
script names it as fleet scope. This needs a ruling. The new exactly-16 loop deliberately keeps it refused, because no ruling admits it.

## 3. The exact final diff, file by file

Full patch: `/tmp/fleet_i120/final-022f68f1a.patch` (129 lines, +80/-0, 5 files). It is reproduced verbatim in section 8.

1. **`tools/launchd/githooks/pre-push`** (+19). In `_scope_is_fleet_support`, this is `bcf7f97ac`'s line as the FIRST case arm:
   `.claude/skills/design-doc-creator/*|.agents/skills/design-doc-creator/*) return 0 ;;   # support, not harness (D-FLEET-36)`.
   It also adds `261fb3b64`'s 16 one-per-line arms `.agents/skills/<name>/*) return 0 ;;` after the D-FLEET-6 sandbox arm.
   **Delta vs 261fb3b64:** the comment above the 16 changes from "Exactly the 16 rename-only .agents skill mirrors" to
   "Exactly the 16 .agents skill mirrors D-FLEET-38 names (rename-only or identical to .claude); support, not harness, so
   product loops keep them. No .agents/skills/* wildcard." `_scope_is_harness` is NOT touched.
2. **`tools/launchd/test_mission_scope_guard.sh`** (+31). The new rows go after #2382's D-FLEET-18 block and before the
   `PASS scope verdicts` echo:
   - D-FLEET-36 block (from bcf7f97ac): for `.claude` and `.agents`, `fleet` allow and `v1` allow on
     `design-doc-creator/scripts/create_planned_doc.sh`; `design-doc-creator-x` refuse on both sides; `.claude/skills/use-ailang` refuse
     (comment relabelled "widen to .claude/skills/* (the D-FLEET-36 arm names one skill)").
   - D-FLEET-38 block (from 261fb3b64), with the skill list moved into a space-padded var `D38=' … '`. For each name: `fleet` allow,
     `v1` allow, `<name>-x` refuse. There are also the `model-manager` refuse, `use-ailang-extra` refuse and `.claude/skills/use-ailang` refuse rows.
   - **NEW (not in #2108):** an exactly-16 loop. It iterates `"$HERE"/../../.agents/skills/*/`, skips `mission-*|sprint-*|design-doc-creator`
     and the D38 names, and asserts `fleet` REFUSE on every other skill. At 022f68f1a that is 18 dirs, including `_shared`, release-manager
     and the identical non-ruled skills. A floor `[ "$n" -ge 10 ] || fail …` stops it passing vacuously. Why: the
     #2108 suite let "admit a 17th skill" survive for every skill except model-manager (M12, M14 and the extra arm in section 5).
     Bash 3.2 safe: it uses case globs with `[[:space:]]` and no arrays. Run under `/bin/bash` 3.2.57.
3. **`design_docs/fleet-mission.md`** — **only the Authority "May change" bullet changes** (+9). It inserts after the
   D-FLEET-18 line (`tools/eval/motoko_connection_probe.sh`, …) and before `tools/pi-extensions/sandbox/**` (D-FLEET-6):
   one line `.claude/skills/design-doc-creator/**`, `.agents/skills/design-doc-creator/**` (D-FLEET-36), then 8 lines holding the
   16 `.agents/skills/<name>/**` paths ending "(D-FLEET-38)". **Delta vs 261fb3b64:** #2108 put the D-FLEET-38 block between `changelogs/**`
   and `CHANGELOG.md`, which split the original support list. The new order is chronological by ruling, after D-FLEET-18.
   **Do not touch the ledger rows, the log or the record stack.** origin/dev's charter is behind the record stack: records
   95–119 live on PR #2412's branch (`fleet/i119-record`). Touching anything but the Authority bullet invites a conflict with that PR.
   The Authority bullet is ~100 lines from where record PRs edit, and `git apply --check` is clean.
4. **`changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet36.md`** (new, +8) and **`…-dfleet38.md`** (new, +13). These are re-dated from
   `2026-11-03-*` and reworded to follow the D-FLEET-18 fragment's format: "ruled A 2026-11-16", no "PROPOSED/pending". The 38 fragment says
   14 rename-only + 2 identical since 2026-11-10. Each names the follow-up as a separate change (iteration-16 rule).
   `scripts/changelog_fold.sh --check` and `scripts/check_changelog.sh` both pass.

## 4. Acceptance commands (all run by the planner on the patched tree; executor re-runs them)

| Command | Source | Planner result |
|---|---|---|
| `/bin/bash tools/launchd/test_mission_scope_guard.sh` | the suite | rc=0, 3 PASS lines |
| `make test-launchd-drivers` | `make/test.mk:136`. **This is exactly what CI's `launchd-drivers` job runs** (`.github/workflows/ci.yml:933-934`, step "Run launchd driver tests", macos-latest, gated by `needs.changes.outputs.shell == 'true'`), after a `/bin/bash` 3.x assertion | rc=0, `run-suites: all 58 suites passed`, 386 s (log `/tmp/fleet_i120/test-launchd-drivers.log`). Run it UNPIPED, because a pipe reports the pipe's rc. |
| CI's other step in that job: `for f in scripts/hooks/*.sh; do /bin/bash -n "$f"; done` | ci.yml:942-949 | Not affected (no `scripts/hooks` change) |
| `bash scripts/mission_decisions.sh --check --file design_docs/fleet-mission.md` | charter ledger validator | `decision ledger valid: 42 rows` |
| `bash tools/launchd/sync-agents-skills.sh --check` | sync gate (also inside test_agents_skills_sync.sh) | rc=0, unchanged |
| `make check-changelog` (= `scripts/check_changelog.sh`) and `bash scripts/changelog_fold.sh --check` | fragment gate | both ✓ (33 fragments) |

**Is there a charter-vs-allowlist consistency test? No.** `git grep` for `_scope_is_fleet_support`, `Authority` and `fleet-mission.md`
across `tools/ scripts/ internal/ cmd/ make/ .github/` finds only the `pre-push` header comment "Keep in step with design_docs/fleet-mission.md
'Authority'". The pre-push refusal text also cites it. `test_controller_capacity.sh` and `rulings_parse_test.go` mention the file for unrelated reasons.
Nothing compares the two texts, so charter drift is enforced only by review. This is listed as an optional follow-up, not in this PR.

## 5. Mutation table (each mutant was applied to `tools/launchd/githooks/pre-push` in the worktree, the suite run, and the change reverted)

Harness: `/tmp/fleet_i120/mut/run_mutants.py`, results `/tmp/fleet_i120/mut/results-final-suite.txt`, every failing row per mutant
in `/tmp/fleet_i120/mut/allfails.txt`. These came from a non-fatal copy of the verdict section, so each named row was CONFIRMED to fail under its mutant, not only the
first failing row. The real suite exits at the first FAIL, and that row is shown in the "first FAIL" column.

| # | Mutant | Named killing row | Final suite | First FAIL in real suite |
|---|---|---|---|---|
| M1 | delete one D-FLEET-38 arm (pr-monitor) | `check fleet .agents/skills/pr-monitor/SKILL.md allow` (D38 loop) | KILLED | same row |
| M2 | delete the collaboration-hub arm (identical pair) | `check fleet .agents/skills/collaboration-hub/SKILL.md allow` | KILLED | same row |
| M3 | move the design-doc-creator arm into `_scope_is_harness` | `check v1 $side/skills/design-doc-creator/scripts/create_planned_doc.sh allow` (both sides fail) | KILLED | v1 .claude row |
| M4 | widen to `.agents/skills/*` (replacing the first 38 arm) | `check fleet .agents/skills/model-manager/SKILL.md refuse`; also the exactly-16 loop (18 rows), `use-ailang-extra`, all 16 `<name>-x`, `design-doc-creator-x` | KILLED (37 rows) | `.agents/skills/design-doc-creator-x` refuse |
| M5 | drop the `/` boundary: `use-ailang*` | `check fleet .agents/skills/use-ailang-x/SKILL.md refuse` and `…/use-ailang-extra/SKILL.md refuse` | KILLED | use-ailang-x row |
| M6 | delete only the `.claude` design-doc-creator side | `check fleet .claude/skills/design-doc-creator/scripts/create_planned_doc.sh allow` | KILLED | same row |
| M7 | delete only the `.agents` design-doc-creator side | `check fleet .agents/skills/design-doc-creator/scripts/create_planned_doc.sh allow` | KILLED | same row |
| M8 | `design-doc-creator/*` → `design-doc-creator*` (both sides) | `check fleet {.claude,.agents}/skills/design-doc-creator-x/SKILL.md refuse` | KILLED | .claude -x row |
| M9 | move one 38 arm (trace-debugger) into `_scope_is_harness` | `check v1 .agents/skills/trace-debugger/SKILL.md allow` | KILLED | same row |
| M10 | widen the 36 arm to `.claude/skills/*` | `check fleet .claude/skills/use-ailang/SKILL.md refuse` (+ `.claude/skills/design-doc-creator-x`) | KILLED | .claude -x row |
| M11 | also admit `.claude/skills/use-ailang/*` | `check fleet .claude/skills/use-ailang/SKILL.md refuse` | KILLED | same row |
| M12 | admit a 17th skill (agent-inbox) | exactly-16 loop row `.agents/skills/agent-inbox` refuse | KILLED (**SURVIVES the #2108 suite**) | same row |
| M13 | admit a 17th skill (model-manager) | `check fleet .agents/skills/model-manager/SKILL.md refuse` (+ loop) | KILLED | same row |
| M14 | admit `.agents/skills/release-manager/*` | exactly-16 loop row release-manager refuse | KILLED (**SURVIVES the #2108 suite**) | same row |
| M15 | delete all 16 D-FLEET-38 arms | D38 loop `fleet … allow` (16 rows) | KILLED | benchmark-runner row |
| M16 | delete the whole D-FLEET-36 line | `check fleet $side/…/create_planned_doc.sh allow` (2 rows) | KILLED | .claude row |
| M17 | (test-side) the exactly-16 enumeration goes vacuous (glob path broken) | floor `[ "$n" -ge 10 ] || fail` | KILLED ("only 1 non-admitted .agents skills enumerated") | floor |

Extra control: with the **#2108 test file** (`261fb3b64`'s version) and the final hook, admitting `agent-inbox`, `release-manager` or
`codebase-organizer` each **SURVIVED**. That suite pinned "exactly 16" only against model-manager, which is why the loop is added.
After every mutant the unmutated suite was re-run, rc=0.

## 6. Risks / what could break

- The exactly-16 loop reads the live `.agents/skills/` tree. If an attended commit ADDS a new `.agents/skills/<x>/`, the loop asserts refuse,
  and the guard already refuses it, so it stays green. If a later ruling admits another skill, that PR must add it to `D38` (intended). If
  `.agents/skills` were moved or emptied, the floor fails loudly instead of passing.
- The CI `changes` filter does not include `.agents/skills/`, so a `.agents`-only PR never runs this suite. That is harmless, because a pure tree change
  can only add refuse rows, which the guard satisfies.
- The PR inherits the dev-red Go test (`TestCloudAgents_RegistryMatchesTheDeletedRoutingTable`). Merge per the attended handoff. Do not "fix" it here.
- `bcf7f97ac`/`261fb3b64` on draft PR #2108 (still OPEN, draft) become superseded. Close #2108 with a pointer once the new PR merges. Do NOT
  merge both.

## 7. Follow-ups (NOT in this PR; each starts only after this arm is on origin/dev, the iteration-16 rule)

1. **D-FLEET-36 step 2:** replace the 8 `grep -E "^\d+\."` sites with `[0-9]` (`.claude/skills/design-doc-creator/scripts/create_planned_doc.sh:128/137/154/163`,
   `.agents/…:125/134/153/162`). Keep each copy's other content as-is, because design-doc-creator is a CONTENT pair (residual 109) and must not be rsynced.
2. **D-FLEET-38 step 2:** extend `SKILLS` in `tools/launchd/sync-agents-skills.sh` with the 16 names, run it (verbatim rsync), and extend
   `tools/launchd/test_agents_skills_sync.sh` (its hard-coded list at :72 and the label at :15). Pre-check: `diff -rq` must show no `Only in .agents` for any of the 16. It showed none at 022f68f1a.
3. **Ticket resolution:** resolve `skill:design-doc-creator-grep-portability` and `skills:agents-copies-stale-non-mission` only after BOTH the
   arm and the dependent edit for that ruling are on origin/dev. Record it in the ledger/log through the record PR, not this one.
4. Optional, needs a ruling: a consistency test that greps the charter Authority bullet for every `_scope_is_fleet_support` / `_scope_is_harness`
   pattern (no such test exists today). Also the `release-manager` sync-vs-guard gap in section 2.

## 8. Final patch (verbatim, `/tmp/fleet_i120/final-022f68f1a.patch`)

```diff
diff --git a/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet36.md b/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet36.md
new file mode 100644
index 000000000..c51d0e0a7
--- /dev/null
+++ b/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet36.md
@@ -0,0 +1,8 @@
+### Changed — Fleet scope guard admits the design-doc-creator skill (D-FLEET-36, ruled A 2026-11-16)
+
+The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
+`.claude/skills/design-doc-creator/**` and `.agents/skills/design-doc-creator/**`, per Mark's attended
+ruling D-FLEET-36 = A (2026-11-16). They are support paths, not harness, so product missions keep
+pushing them; every other `.claude/skills` path stays refused for the fleet. Guard arm, Authority line
+and regression arms only; under the iteration-16 rule the skill's eight `grep -E "^\d+\."` sites change
+in a separate fleet change once this lands on the pinned `origin/dev` guard.
diff --git a/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet38.md b/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet38.md
new file mode 100644
index 000000000..9d2cfb585
--- /dev/null
+++ b/changelogs/unreleased/2026-11-18-fleet-scope-arm-dfleet38.md
@@ -0,0 +1,13 @@
+### Changed — Fleet scope guard admits 16 named `.agents` skill mirrors (D-FLEET-38, ruled A 2026-11-16)
+
+The fleet support allowlist (`tools/launchd/githooks/pre-push`, `_scope_is_fleet_support`) now admits
+exactly 16 enumerated `.agents/skills/<name>/**` trees, per Mark's attended ruling D-FLEET-38 = A
+(2026-11-16): benchmark-runner, builtin-developer, cloud-cascade-debug, collaboration-hub,
+coordinator-helper, eval-analyzer, eval-gap-finder, gemini-cli-helper, github-issue-triage,
+local-model-onboarding, local-ollama-eval, parser-developer, pr-monitor, prompt-manager,
+trace-debugger and use-ailang. Each differs from its `.claude/skills` source only by the 2026-07-28
+claude-to-Codex renames, or (collaboration-hub, trace-debugger, since 2026-11-10) not at all. They
+are support paths, not harness, so product missions are unaffected; there is no `.agents/skills/*`
+wildcard and no `.claude/skills` path. Guard arm, Authority lines and regression arms only; extending
+`tools/launchd/sync-agents-skills.sh` to these skills is a separate fleet change once this lands on
+the pinned `origin/dev` guard (iteration-16 rule).
diff --git a/design_docs/fleet-mission.md b/design_docs/fleet-mission.md
index 7907255a1..4511ea1bb 100644
--- a/design_docs/fleet-mission.md
+++ b/design_docs/fleet-mission.md
@@ -85,6 +85,15 @@ Clause map: **1** UNMEASURED, goal unmoved; **2** UNMET (nothing resolved; media
   `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
   `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
   `tools/eval/motoko_connection_probe.sh`, `tools/eval/test_motoko_connection_probe.sh` (D-FLEET-18),
+  `.claude/skills/design-doc-creator/**`, `.agents/skills/design-doc-creator/**` (D-FLEET-36),
+  `.agents/skills/benchmark-runner/**`, `.agents/skills/builtin-developer/**`,
+  `.agents/skills/cloud-cascade-debug/**`, `.agents/skills/collaboration-hub/**`,
+  `.agents/skills/coordinator-helper/**`, `.agents/skills/eval-analyzer/**`,
+  `.agents/skills/eval-gap-finder/**`, `.agents/skills/gemini-cli-helper/**`,
+  `.agents/skills/github-issue-triage/**`, `.agents/skills/local-model-onboarding/**`,
+  `.agents/skills/local-ollama-eval/**`, `.agents/skills/parser-developer/**`,
+  `.agents/skills/pr-monitor/**`, `.agents/skills/prompt-manager/**`,
+  `.agents/skills/trace-debugger/**`, `.agents/skills/use-ailang/**` (D-FLEET-38),
   `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
   **Anything else is refused**, including CI workflows, `go.mod`, the server, the UI and non-mission
   commands. A fix that genuinely needs one of those parks for Mark.
diff --git a/tools/launchd/githooks/pre-push b/tools/launchd/githooks/pre-push
index 806d056fe..c2256bf2d 100755
--- a/tools/launchd/githooks/pre-push
+++ b/tools/launchd/githooks/pre-push
@@ -28,10 +28,29 @@ _scope_is_harness() {
 # server, the UI, other commands) is refused rather than silently allowed.
 _scope_is_fleet_support() {
   case "$1" in
+    .claude/skills/design-doc-creator/*|.agents/skills/design-doc-creator/*) return 0 ;;   # support, not harness (D-FLEET-36)
     design_docs/*|changelogs/*|CHANGELOG.md|docs/*|.ailang/state/sprints/*) return 0 ;;
     internal/config/mission.go|make/test.mk) return 0 ;;
     tools/eval/motoko_connection_probe.sh|tools/eval/test_motoko_connection_probe.sh) return 0 ;;   # support, not harness (D-FLEET-18)
     tools/pi-extensions/sandbox/*) return 0 ;;   # the pi sandbox fail-closed fix (D-FLEET-6); the eval harness shares the rest of tools/pi-extensions
+    # Exactly the 16 .agents skill mirrors D-FLEET-38 names (rename-only or identical to .claude);
+    # support, not harness, so product loops keep them. No .agents/skills/* wildcard.
+    .agents/skills/benchmark-runner/*) return 0 ;;
+    .agents/skills/builtin-developer/*) return 0 ;;
+    .agents/skills/cloud-cascade-debug/*) return 0 ;;
+    .agents/skills/collaboration-hub/*) return 0 ;;
+    .agents/skills/coordinator-helper/*) return 0 ;;
+    .agents/skills/eval-analyzer/*) return 0 ;;
+    .agents/skills/eval-gap-finder/*) return 0 ;;
+    .agents/skills/gemini-cli-helper/*) return 0 ;;
+    .agents/skills/github-issue-triage/*) return 0 ;;
+    .agents/skills/local-model-onboarding/*) return 0 ;;
+    .agents/skills/local-ollama-eval/*) return 0 ;;
+    .agents/skills/parser-developer/*) return 0 ;;
+    .agents/skills/pr-monitor/*) return 0 ;;
+    .agents/skills/prompt-manager/*) return 0 ;;
+    .agents/skills/trace-debugger/*) return 0 ;;
+    .agents/skills/use-ailang/*) return 0 ;;
   esac
   return 1
 }
diff --git a/tools/launchd/test_mission_scope_guard.sh b/tools/launchd/test_mission_scope_guard.sh
index 358dc4749..2a9282966 100755
--- a/tools/launchd/test_mission_scope_guard.sh
+++ b/tools/launchd/test_mission_scope_guard.sh
@@ -64,6 +64,37 @@ check fleet tools/eval/xmotoko_connection_probe.sh refuse # kills: path 1 basena
 check fleet tools/eval/xtest_motoko_connection_probe.sh refuse # kills: path 2 basename-prefix glob
 check fleet scripts/motoko_connection_probe.sh refuse # kills: path 1 directory wildcard
 check fleet scripts/test_motoko_connection_probe.sh refuse # kills: path 2 directory wildcard
+# D-FLEET-36: both mirrors, including nested files.
+for side in .claude .agents; do
+  check fleet "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # kills: delete this side's design-doc-creator pattern
+  check v1 "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # kills: move the pattern into _scope_is_harness
+done
+check fleet .claude/skills/design-doc-creator-x/SKILL.md refuse # kills: design-doc-creator/* -> design-doc-creator*
+check fleet .agents/skills/design-doc-creator-x/SKILL.md refuse # kills: design-doc-creator/* -> design-doc-creator*
+check fleet .claude/skills/use-ailang/SKILL.md refuse # kills: widen to .claude/skills/* (the D-FLEET-36 arm names one skill)
+# D-FLEET-38: every enumerated arm, product loops unaffected, no over-widening.
+D38=' benchmark-runner builtin-developer cloud-cascade-debug collaboration-hub coordinator-helper
+  eval-analyzer eval-gap-finder gemini-cli-helper github-issue-triage local-model-onboarding
+  local-ollama-eval parser-developer pr-monitor prompt-manager trace-debugger use-ailang '
+for skill in $D38; do
+  check fleet ".agents/skills/$skill/SKILL.md" allow # kills: delete this skill's arm
+  check v1 ".agents/skills/$skill/SKILL.md" allow # kills: move this skill's arm into _scope_is_harness
+  check fleet ".agents/skills/$skill-x/SKILL.md" refuse # kills: drop this arm's / boundary (<name>* admits siblings)
+done
+check fleet .agents/skills/model-manager/SKILL.md refuse # kills: replace the arms with .agents/skills/*
+check fleet .agents/skills/use-ailang-extra/SKILL.md refuse # kills: use-ailang/* -> use-ailang*
+check fleet .claude/skills/use-ailang/SKILL.md refuse # kills: also admit .claude/skills/use-ailang/*
+# Exactly 16: every other .agents skill in the tree stays refused (mission-*/sprint-* are
+# harness, design-doc-creator is D-FLEET-36). The count floor keeps the loop from passing empty.
+n=0
+for d in "$HERE"/../../.agents/skills/*/; do
+  skill=$(basename "$d")
+  case "$skill" in mission-*|sprint-*|design-doc-creator) continue ;; esac
+  case "$D38" in *[[:space:]]"$skill"[[:space:]]*) continue ;; esac
+  check fleet ".agents/skills/$skill/SKILL.md" refuse # kills: admit a 17th .agents skill
+  n=$((n+1))
+done
+[ "$n" -ge 10 ] || fail "only $n non-admitted .agents skills enumerated; the exactly-16 rows ran (nearly) empty"
 echo 'PASS scope verdicts: product loops refused on harness paths (incl. the fleet charter), fleet held to its allowlist, attended unaffected'
 
 # End to end: git must actually run the hook when core.hooksPath comes ONLY from env.
```
