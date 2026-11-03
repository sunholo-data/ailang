1. **Scope and delivery.** Base is clean at `007809894`, branch `fleet/i82-dfleet36-38-scope-arm`. This is a read-only plan; no files changed. Prepare one **DRAFT PR**, with two separable commits:
   - **C1:** D-FLEET-36 support arm, its tests, Authority addition, comment correction, and changelog section.
   - **C2:** D-FLEET-38 support arms, its tests, Authority addition, comment correction, and changelog section.
   Keep both decision rows OPEN. Merge only the commit(s) covered by Mark’s ruling. This PR ships the guard widening alone, before dependent edits, under `design_docs/fleet-mission.md:128–132`.
   Touch only `tools/launchd/githooks/pre-push`, `tools/launchd/test_mission_scope_guard.sh`, `design_docs/fleet-mission.md`, and the fragment specified below. No skill files or sync-script changes.

2. **Exact guard hunks.** The following unified diffs specify before/after text; all unshown arms remain byte-identical. C1 inserts before original `pre-push:31`; C2 inserts after original `pre-push:33`. Neither changes `_scope_is_harness` (`:14–23`) or `_scope_verdict` (`:50–63`).

   **C1:**
   ```diff
   @@ tools/launchd/githooks/pre-push:30
     case "$1" in
   +    .claude/skills/design-doc-creator/*|.agents/skills/design-doc-creator/*) return 0 ;;   # support, not harness (D-FLEET-36)
       design_docs/*|changelogs/*|CHANGELOG.md|docs/*|.ailang/state/sprints/*) return 0 ;;
   @@ tools/launchd/githooks/pre-push:12
   -# Harness paths: the fleet mission's write scope. Keep in step with
   +# Harness paths: fleet may change these; product loops may not. Keep in step with
   ```
   **C2:**
   ```diff
   @@ tools/launchd/githooks/pre-push:33
       tools/pi-extensions/sandbox/*) return 0 ;;   # the pi sandbox fail-closed fix (D-FLEET-6); the eval harness shares the rest of tools/pi-extensions
   +    # Exactly the 16 rename-only .agents skill mirrors; support, not harness (D-FLEET-38).
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
   @@ tools/launchd/githooks/pre-push:13
   -# design_docs/fleet-mission.md "Authority".
   +# design_docs/fleet-mission.md "Authority", including the separate support allowlist below.
   ```
   These are Bash 3.2 case patterns. The literal `/` before each `*` preserves the directory boundary.
   For **each** newly admitted path, `v1`, `docs`, `motoko`, and other product missions receive **allow** (empty verdict): their branch consults only `_scope_is_harness`. Fleet changes from **refuse** to **allow**.

3. **Exact test additions**, before the PASS message at `tools/launchd/test_mission_scope_guard.sh:53`. Retain existing tests. Each inline comment names the mutation killed by that specific assertion.

   **C1 block:**
   ```bash
   # D-FLEET-36: both mirrors, including nested files.
   for side in .claude .agents; do
     check fleet "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # delete this side's design-doc-creator pattern
     check v1 "$side/skills/design-doc-creator/scripts/create_planned_doc.sh" allow # move this side's pattern into _scope_is_harness
   done
   check fleet .agents/skills/model-manager/SKILL.md refuse # widen the .agents pattern to .agents/skills/*
   check fleet .claude/skills/use-ailang/SKILL.md refuse # widen the .claude pattern to .claude/skills/*
   check fleet .claude/skills/design-doc-creator-x/SKILL.md refuse # change .claude design-doc-creator/* to design-doc-creator*
   check fleet .agents/skills/design-doc-creator-x/SKILL.md refuse # change .agents design-doc-creator/* to design-doc-creator*
   ```
   **C2 block**, independently retained if C1 is dropped:
   ```bash
   # D-FLEET-38: exercise every enumerated arm, including use-ailang.
   for skill in benchmark-runner builtin-developer cloud-cascade-debug collaboration-hub \
     coordinator-helper eval-analyzer eval-gap-finder gemini-cli-helper github-issue-triage \
     local-model-onboarding local-ollama-eval parser-developer pr-monitor prompt-manager \
     trace-debugger use-ailang; do
     check fleet ".agents/skills/$skill/SKILL.md" allow # delete exactly this skill's support arm
     check v1 ".agents/skills/$skill/SKILL.md" allow # move exactly this skill's arm into _scope_is_harness
   done
   check fleet .agents/skills/model-manager/SKILL.md refuse # replace the enumerated arms with .agents/skills/*
   check fleet .claude/skills/use-ailang/SKILL.md refuse # additionally admit .claude/skills/use-ailang/*
   check fleet .agents/skills/use-ailang-extra/SKILL.md refuse # change use-ailang/* to use-ailang*
   ```
   In the loops, each expanded assertion targets its named side/skill; deleting another arm is not its claimed mutation.

4. **Exact Authority additions.** Both are insertion-only hunks against distinct original lines, with independent row citations. Existing support paths and surrounding policy stay unchanged.

   **C1, after `design_docs/fleet-mission.md:86`:**
   ```diff
      `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
   +  `.claude/skills/design-doc-creator/**`, `.agents/skills/design-doc-creator/**` (D-FLEET-36),
      `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
   ```
   **C2, after `design_docs/fleet-mission.md:85`:**
   ```diff
      `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
   +  `.agents/skills/benchmark-runner/**`, `.agents/skills/builtin-developer/**`,
   +  `.agents/skills/cloud-cascade-debug/**`, `.agents/skills/collaboration-hub/**`,
   +  `.agents/skills/coordinator-helper/**`, `.agents/skills/eval-analyzer/**`,
   +  `.agents/skills/eval-gap-finder/**`, `.agents/skills/gemini-cli-helper/**`,
   +  `.agents/skills/github-issue-triage/**`, `.agents/skills/local-model-onboarding/**`,
   +  `.agents/skills/local-ollama-eval/**`, `.agents/skills/parser-developer/**`,
   +  `.agents/skills/pr-monitor/**`, `.agents/skills/prompt-manager/**`,
   +  `.agents/skills/trace-debugger/**`, `.agents/skills/use-ailang/**` (D-FLEET-38),
   ```
   `design_docs/fleet-mission*` is already a harness pattern at `pre-push:20`, so fleet may push these charter edits under the pinned base guard.

5. **Mirror audit.** Searched `_scope_is_fleet_support|tools/pi-extensions/sandbox|D-FLEET-6` across all requested roots, including both mission-skill trees. No additional authority mirror needs editing.
   - **Change:** `tools/launchd/githooks/pre-push:29,33,56`; `tools/launchd/test_mission_scope_guard.sh:50,52`, as specified above.
   - **Unchanged runtime/fixture references:** `tools/launchd/test_mission_pi_run_sandbox.sh:87`; `tools/launchd/test_pi_ext_args.sh:57`; `scripts/mission_pi_run.sh:148,176–177`; `make/test.mk:41`.
   - **Unchanged instructions:** `.claude/skills/mission-control/resources/gate-3-route.md:363,376,381` and the same `.agents` file/sites. These describe sandbox loading, not fleet authority.
   - **Zero results, with positive grep controls:** `internal/mission` → `^package mission` matches `internal/mission/ticket.go:1`; `cmd/ailang/mission*` → `^package main` matches `cmd/ailang/mission_ticket.go:1`; `docs/` → `^# The AILANG message plane` matches `docs/internal/message-plane-topology.md:1`.
   - `tools/launchd/sync-agents-skills.sh:21` and `test_agents_skills_sync.sh:72` separately enumerate six existing mirrored skills. **Do not change or run the copying sync here.** Their extension belongs to D-FLEET-38’s post-ruling follow-on.

6. **Changelog**, following `changelogs/unreleased/README.md`. Exact combined content of new `changelogs/unreleased/2026-11-03-fleet-scope-arm-dfleet36-38.md:1`:
   ```markdown
   ### Changed — Proposed fleet support scope for design-doc-creator (2026-11-03)

   PROPOSED / pending Mark's ruling: commit 1 implements only D-FLEET-36 A's guard arm, allowing both design-doc-creator skill trees as fleet support paths while preserving product-loop access. No skill content changes. See [Authority](../../design_docs/fleet-mission.md#authority-the-write-scope--enforced-by-toolslaunchdgithookspre-push).

   ### Changed — Proposed fleet support scope for 16 agents skill mirrors (2026-11-03)

   PROPOSED / pending Mark's ruling: commit 2 implements only D-FLEET-38 A's guard arms for the 16 explicitly enumerated rename-only .agents skill trees while preserving product-loop access. No skill copies or sync-list changes. See [D-FLEET-38](../../design_docs/fleet-mission.md).
   ```
   C1 creates the first section; C2 appends the second. If either commit is dropped, retain only its counterpart’s section; a C2-only rebase must recreate this file with the second section. Guard, tests, and Authority have no logical dependency between commits.

7. **Gates**, narrowest first, at base, C1, and combined head; also at C2-only head if C1 is dropped:
   ```bash
   bash -n tools/launchd/githooks/pre-push
   bash -n tools/launchd/test_mission_scope_guard.sh
   git diff --check
   bash scripts/mission_decisions.sh --check --file design_docs/fleet-mission.md
   bash tools/launchd/test_mission_scope_guard.sh
   make check-changelog
   make check-context-docs
   make check-no-personal-email
   make test-launchd-drivers
   ```
   Also check committed patch whitespace with `git diff --check 007809894 HEAD`; plain `git diff --check` covers working-tree edits.
   `make/test.mk:139` syntax-checks launchd shell files; `:115,130–148` includes the scope suite. There is **no ShellCheck gate for these files**: `make/code-health.mk:34–41` scopes `shellcheck-autopush` to four `scripts/hooks/` files.
   The scope suite writes temporary Git repositories and performs local filesystem pushes (`:56–77`), without a network socket. `test-launchd-drivers` writes fixtures and includes a live loopback socket arm at `tools/eval/test_motoko_connection_probe.sh:612–646`. `check-context-docs` creates temporary files; `check-no-personal-email` scans the charter through its tracked mission-doc scope.
   Base observations: both Bash syntax checks, whitespace check, and the 40-row ledger check completed successfully. **`make check-changelog`: UNINFORMATIVE UNDER SANDBOX**, because make attempted denied `/tmp/xcrun_db-*` writes despite returning zero. Fixture suites were not run. Any socket/write-restricted attempt must receive that same label, never pass/fail.

8. **Authority and risks.** This is purely an allowlist widening in implementation, and **is an authority widening**, therefore ruling-gated. It changes no routing, lane order, quota, ration threshold, billing guard, sync behavior, or skill content.
   Main risks: accidentally placing an arm in `_scope_is_harness` and refusing product loops; wildcard/prefix overreach; omitting one of the 16 names; admitting `.claude` counterparts unintentionally; and dropping a commit while retaining its documentation claims. The positive/product/negative assertions and per-row hunks address these risks. After the applicable ruling and guard-only landing, dependent work waits for a subsequent fire using the updated pinned `origin/dev` guard.