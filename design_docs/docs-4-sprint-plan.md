# Sprint Plan: docs-4 — guides taxonomy and redundancy pass

## Overview

- Routing brief: `design_docs/docs-4-brief.md` (ratified, quorum-passed).
- Goal: apply the brief's Phase A sidebar/cruft pass, then remove the five enumerated redundant sections/pages in Phase B, in order.
- Duration: 1–1.5 executor days plus one evaluator round.
- Scope: exactly the files listed in the brief's **Files** section; all are under `docs/`.
- Structural-only sprint: no content rewrites, no surviving-page renames/moves on disk, and no CHANGELOG entry.
- The executor must follow Appendix A dispositions and Appendix B's target tree literally; placement decisions are not reopened.

## Current status and execution guardrails

The brief establishes the baseline: 62 files directly under `docs/docs/guides/` (including one tracked `.bak`), 61 published guide pages plus 11 evaluation pages, 63 guide IDs in the sidebar, and nine served-but-unlinked guide pages. The target is 59 direct files, 70 guide IDs, no orphan or dangling IDs, and no repository-internal links to the two deleted page IDs.

The only permitted implementation surface is:

- `docs/sidebars.js`
- `docs/docs/guides/**` (including the four listed guide edits/deletions)
- `docs/docs/intro.mdx`
- `docs/docs/examples.mdx`
- `docs/docs/architecture/index.md`
- `docs/docs/guides/debugging.md`

Do not edit `internal/`, `cmd/`, `docs/scripts/sync-registry.sh`, `make/docs.mk`, deferred stale facts, or any file not listed above. Do not stage, commit, push, or use git operations to discard work except for the four mandatory build-side-effect cleanup paths explicitly required by acceptance check 6.

## Milestone breakdown

### M1 — Phase A: remove cruft and rewrite the guide sidebar

**Duration:** 2–3 hours

**Goal:** complete all mechanical navigation work before any redundancy removal.

**Tasks:**

- Delete the tracked cruft file `docs/docs/guides/agent-integration.mdx.bak`.
- Rewrite only the guide entries in `docs/sidebars.js` to Appendix B: dissolve `Ecosystem` and `Prompts`; add `Coordinator & Messaging`, `AI Effect`, and `Memory & Search`; wire all nine orphans; move every file exactly as Appendix A specifies; keep non-guide entries unchanged.
- Do not edit frontmatter or page contents in this milestone.
- Verify the sidebar syntax and inspect the diff for the exact Phase A scope before starting M2.

**Files:** `docs/sidebars.js`, `docs/docs/guides/agent-integration.mdx.bak`.

**Acceptance criteria:** the `.bak` is absent; the sidebar has Appendix B's 70 guide IDs; all surviving pages retain their existing disk paths; no Phase B content changes are mixed into the milestone.

### M2 — Phase B/B1: consolidate cross-project messaging, then delete the page

**Duration:** 1.5–2 hours

**Depends on:** M1.

**Goal:** preserve the one missing CI example in the canonical messaging page, then delete the redundant page.

**Tasks:**

- In `docs/docs/guides/agent-messaging.md` § `Workflows`, append exactly the `Automated Feedback (Advanced)` fenced bash snippet from `cross-project-messaging.mdx`; carry over the snippet, not the surrounding duplicated prose.
- Remove `agent-messaging.md`'s Related link to `cross-project-messaging` because the page now owns the carried-over material.
- Delete `docs/docs/guides/cross-project-messaging.mdx` and remove its sidebar entry.
- Preserve the canonical cloud/local storage model in `agent-messaging.md`; do not copy the contradictory Storage section.

**Files:** `docs/docs/guides/agent-messaging.md`, `docs/docs/guides/cross-project-messaging.mdx`, `docs/sidebars.js`.

**Acceptance criteria:** the CI block is present once in `agent-messaging.md`; no `cross-project-messaging` link or sidebar ID remains; no unrelated messaging prose is rewritten.

### M3 — Phase B/B2: delete the obsolete development page and retarget inbound links

**Duration:** 1.5–2 hours

**Depends on:** M2.

**Tasks:**

- Delete `docs/docs/guides/development.md` and remove its sidebar entry.
- Retarget `docs/docs/intro.mdx`, `docs/docs/examples.mdx`, and the `docs/docs/architecture/index.md` “Contributing to the codebase” bullet to exactly `https://github.com/sunholo-data/ailang/blob/dev/CONTRIBUTING.md`.
- Retarget `docs/docs/guides/debugging.md`'s “Full development workflow” link to `/docs/guides/development-workflow`.
- Do not carry over the stale make-target tables, layer table, operator recipe, or policy sections.

**Files:** `docs/docs/guides/development.md`, `docs/docs/intro.mdx`, `docs/docs/examples.mdx`, `docs/docs/architecture/index.md`, `docs/docs/guides/debugging.md`, `docs/sidebars.js`.

**Acceptance criteria:** all four inbound links are retargeted as specified; no link matches the deleted `guides/development` ID; `development-workflow` remains intact and is still allowed to match the bounded acceptance grep.

### M4 — Phase B/B3: trim the duplicated agent onboarding section

**Duration:** 45–60 minutes

**Depends on:** M3.

**Tasks:**

- In `docs/docs/guides/getting-started.mdx`, replace exactly the section from `## <Icon name="bot" inline /> For AI Agents: CLI Integration` up to (excluding) `## <Icon name="user" inline /> For Human Developers: Manual Installation` with the brief's three-line pointer to `AI Agent Integration`.
- Preserve the preceding MCP Servers section unchanged because `agent-mcp.md` links into it.
- Ensure the removed inline `typescript`-fenced AILANG block and “Best model” claim do not reappear.

**Files:** `docs/docs/guides/getting-started.mdx`.

**Acceptance criteria:** the exact B3 boundary is replaced by a pointer; the surrounding sections and inbound MCP anchor remain unchanged; no broad prose rewrite occurs.

### M5 — Phase B/B4: trim the hooks message-system duplicate

**Duration:** 30–45 minutes

**Depends on:** M4.

**Tasks:**

- In `docs/docs/guides/hooks-setup.mdx`, replace the `Message System` section and its three H3s (Checking Inbox, Sending Messages, Message Storage) with one line pointing to `/docs/guides/agent-messaging`.
- Preserve the surrounding `SessionStart Hook Behavior` section and all other hook material.

**Files:** `docs/docs/guides/hooks-setup.mdx`.

**Acceptance criteria:** the three duplicate H3s are gone, exactly one pointer remains, and the SessionStart section is untouched.

### M6 — Phase B/B5: trim semantic-cache duplication and run all final gates

**Duration:** 2–3 hours including build and evidence capture

**Depends on:** M5.

**Tasks:**

- In `docs/docs/guides/semantic-caching-how-to.mdx`, replace the section from `## Two-Tier Search Architecture` up to (excluding) `## Embeddings Doctrine` with one line pointing to `/docs/guides/semantic-search`.
- Preserve `Embeddings Doctrine` and the remaining caching how-to content.
- Run every acceptance command below in numerical order and record its result. Check 6's cleanup is a separate command step immediately after the build and before check 7.
- Confirm the final diff contains only the brief's Files list. Do not add a CHANGELOG entry.

**Files:** `docs/docs/guides/semantic-caching-how-to.mdx` plus all files in the brief's Files section for final scope verification.

**Acceptance criteria:** B5 is a trim-to-pointer at the exact boundary; all eight final acceptance checks pass, with check 6's four-path cleanup performed before check 7.

## Day-by-day execution order

### Day 1

1. Read the brief and preserve a clean status baseline.
2. Complete M1 and verify Appendix B navigation, including all nine orphan insertions.
3. Complete M2/B1, including the exact CI snippet carry-over and deletion.
4. Complete M3/B2, retargeting all four inbound links and deleting `development.md`.
5. Run focused link/content checks before proceeding.

### Day 2 (half-day contingency / evaluator handoff)

1. Complete M4/B3, M5/B4, and M6/B5 in order.
2. Run the eight final acceptance checks exactly as written below.
3. If the build mutates the four known paths, perform the mandatory cleanup step before inspecting `git diff --stat`.
4. Hand off command output and the final scope check to the evaluator; do not commit or stage.

## Final acceptance criteria — literal commands

The executor must run these in order. The expected result follows each check.

1. File count is 59:

   ```bash
   find docs/docs/guides -maxdepth 1 -type f | wc -l
   ```

   Expected: `59`.

2. The orphan check prints nothing:

   ```bash
   cd docs && for f in $(find docs/guides -type f \( -name '*.md' -o -name '*.mdx' \) | sed -E 's|^docs/||; s|\.mdx?$||'); do grep -q "'$f'" sidebars.js || echo "ORPHAN: $f"; done
   ```

3. There are 70 unique guide IDs and no dangling IDs:

   ```bash
   grep -o "'guides/[^']*'" docs/sidebars.js | sort -u | wc -l
   for id in $(grep -o "'guides/[^']*'" docs/sidebars.js | sort -u | sed "s/'//g"); do [ -f docs/docs/$id.md ] || [ -f docs/docs/$id.mdx ] || echo DANGLING; done
   ```

   Expected: first command prints `70`; second prints nothing.

4. No deleted-page inbound links remain:

   ```bash
   grep -rlE "guides/(cross-project-messaging|development)([^a-z0-9_-]|$)" docs/docs docs/src docs/sidebars.js
   ```

   Expected: empty output.

5. Redundancy counts match the brief:

   ```bash
   grep -l -- 'ailang messages ack --all' docs/docs/guides/*.md*
   grep -l "_ollama_embed" docs/docs/guides/*.md*
   ```

   Expected: first command lists exactly 2 files; second command does not list `semantic-caching-how-to.mdx`.

6. Registry sync and docs build are green, then perform the mandatory cleanup as its own step:

   ```bash
   bash docs/scripts/sync-registry.sh && make docs-build
   ```

   Immediately after that command, before touching git for any other purpose:

   ```bash
   git checkout -- docs/docs/design-docs.md docs/docs/prompts/current.md docs/docs/roadmap/index.md docs/src/data/packages-sidebar.json
   git status --short
   ```

   Expected: build succeeds; `git status --short` shows none of those four paths.

7. Only the brief's Files list is touched (run after check 6 cleanup):

   ```bash
   git diff --stat
   ```

   Expected: every listed path is in the brief's Files section; no other path appears.

8. CHANGELOG entry is not required. Confirm no CHANGELOG edit was introduced:

   ```bash
   git diff --name-only -- CHANGELOG.md
   ```

   Expected: empty output.

## Success metrics and handoff

- 59 direct guide files, 70 non-dangling guide IDs, and no orphan output.
- B1–B5 completed in order with deletion criterion and same-commit inbound-link retargeting satisfied for both deleted pages.
- Surviving page URLs remain stable; categorisation is sidebar-only.
- The docs build passes after registry sync, and the four build side-effect files are cleaned before final scope inspection.
- No files outside the brief's Files list are changed, no content rewrite is introduced, and no CHANGELOG entry is made.
