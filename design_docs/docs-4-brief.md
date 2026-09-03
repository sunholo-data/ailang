# docs-4 design brief — clause 5 taxonomy pass for `docs/docs/guides/` (ONE sprint)

**Status**: Planned · **Clause**: 5 (concision and anti-sprawl) — with a clause-3 component (orphaned
pages unreachable from the nav) · **Estimated**: 1–1.5 days executor + one evaluator round ·
**Dependencies**: none (both blockers named in the charter's docs-4 row are cleared: clauses 1–3 are
green, and D-2 dissolved the allowlist question — `docs/*` reaches nested paths).

**Planner-Lane**: codex-ok

(Every touched file is under `docs/**`, inside `MISSION_PLANNER_ALLOWLIST`'s `docs/*` pattern —
verified per D-2's control: `case` globs match `/`. This brief DOES carry design claims — a
disposition for every one of the 62 files — so unlike docs-1/2/3/5/10 it is a design doc and should
pass the mission's design quorum at its gate. Quorum-trigger check from the design-doc-creator
skill: (1) no design-freeze items — clause 5 already carries Mark's 2026-08-28 ruling that deletion
is in scope; (2) overrides no shared machinery; (3) touches no cost/KPI/banked-data surface; (4) no
external-system premises. All four false, so an attended session could skip it; the unattended
loop runs it regardless per the skill.)

**Quorum log.** Round 1 (`docs-4-brief-2026-09-03T10-54-51Z.json`): BLOCKED, both present
reviewers reject (`oc-glm-5-2` absent, degraded to N-1). Both objections were narrow verification
gaps, not direction disputes, each with a concrete `proposed_fix`: `gpt5-6-sol` — V6 probed only
8 of 9 orphan URLs; `gemini-3-1-pro` — no Verification Log row proved the B4/B5 section-cut
headings actually exist and are adjacent. The controller measured both directly (not re-routed
through the designer, since both are single-command verifications with no design judgment):
the 9th orphan (`evaluation/cost-and-speed-budgets`) also returns `200` (V6, corrected), and both
heading pairs are exactly as asserted (new row V29). Re-quorum pending.

---

## The decision this doc was asked to make: one sprint, not a decomposition

The controller asked for an explicit, evidence-backed call between (a) one sprint-sized doc with a
fully enumerated action list, or (b) a parent decomposition plus 2–4 sub-sprints ("consolidate camp
A / camp B"). **The answer is (a).** The premise behind (b) — that 62 accreted files hide several
camps of near-duplicate pages each needing a multi-day merge — was tested and is false:

- **Literal duplication across the 61 pages is close to zero.** A pairwise instrument that counts
  identical non-heading lines (>40 chars) between every pair of guides (1,830 pairs) finds only
  TWO pairs sharing ≥4 lines: `agent-workflows.mdx`↔`claude-code-integration.mdx` (5 lines) and
  `getting-started.mdx`↔`quick-start-examples.mdx` (4 lines). No pair shares more. (V7)
- **The redundancy that does exist is section-level, not page-level:** the same `ailang messages
  send/list/ack` command block appears in 4–5 guides, `make services-start` in 6, `ailang
  coordinator status` in 7 (V8). That is five sections in five files, each ≤45 lines — an
  afternoon, not a camp.
- **Six of the 2026-08-17 "audit pass" commits already did the page-level merging** (folded the
  comparison essays into `three-camps-comparison`, slimmed `agent-integration`, trimmed the
  bridge/state pages, removed superseded pages — V22). What is left is what those passes did not
  touch: the *nav* and the *cruft*.
- **The remaining work is enumerable at the file level** (Appendix A gives every one of the 62
  files a disposition; Appendix B gives the target sidebar tree) and is mostly a single edit to
  `docs/sidebars.js`, one `git rm`, two page deletions, three section trims and five link
  retargets.

So the charter's "40+ guides accreted over time" is accurate about the *nav* (nine pages are
reachable only by URL; the largest operational cluster is split across two top-level categories;
five pages sit in the wrong audience bucket) but not about *content* duplication. Decomposing a
sidebar edit into three sprints would front-load design for work that does not exist.

## Problem statement (measured at HEAD `55891002f`, 2026-09-03)

`docs/docs/guides/` holds **62 files**: 61 published pages (`.md`/`.mdx`) plus one `.bak`; the
`evaluation/` subdirectory holds 11 more pages (V1, V2). Against the charter's clause 5 ("the site is
organised and categorised, and says each thing once … the nav reflects a real taxonomy rather than
accretion order") and clause 3 ("no orphaned pages unreachable from the nav"), the live defects are:

1. **Cruft committed into a published directory.** `agent-integration.mdx.bak` has been tracked in
   git since `f8f0c0976` (2025-11-13). It is the pre-rewrite copy of `agent-integration.mdx` — it
   still tells agents to download per-platform tarballs and to load `prompts/v0.3.8.md`, both
   superseded by the 2026-08-17 rewrite that points at `ailang prompt`. Docusaurus does not
   publish non-`.md(x)` files (live URL → 404, with a 404 control and a 200 control), so it is
   repo cruft rather than a published page — but it is exactly "a page that exists only because
   nobody dared remove it". (V3)
2. **Nine orphaned pages** — built and served (all return HTTP 200 on the live site) but absent
   from `sidebars.js`, so unreachable from the nav: six top-level (`ai-stdlib-discovery`,
   `coordinator-workers`, `custom-ai-providers`, `notification-channels`, `notify-daemon`,
   `strict-fallbacks`) and three under `evaluation/` (`cost-and-speed-budgets`, `local-ollama`,
   `measurement-contract`). Five of the six top-level orphans also have **zero inbound links from
   any page or component**; `measurement-contract` likewise. A reader cannot get to them at all
   except by guessing the URL. (V4, V5, V6)
3. **The coordinator cluster is split across two top-level categories.** `coordinator.md` and
   `coordinator-setup.md` sit under *Internals & Vision → Contributors* (collapsed by default),
   while their companions `collaboration-hub`, `cloud-messaging-integration`, `workspaces` and
   `agent-messaging` sit under *For AI Agents → Workflows*. `coordinator-setup.md`'s own title is
   "Setting Up the Coordinator for External Projects" — a user-facing operations page, not a
   contributor page. (V11)
4. **Pages in the wrong audience bucket.** `development.md` (make targets, Go package layout,
   "Adding a Binary Operator") is under *Learn AILANG*; `ai-effect`, `ai-routing` (and the
   orphaned `custom-ai-providers`) — language/runtime features for calling LLMs *from* AILANG —
   are under *For AI Agents → Prompts*, a bucket for agents that *write* AILANG;
   `secret-approvals.md` (Terraform deploy + coordinator/executor config + phone setup) is under
   *Learn AILANG*; `benchmarking.md` (`ailang eval` single-shot + custom benchmark YAML) is under
   *Build & Operate → Develop* rather than beside the `evaluation/` pages it belongs with; the
   *Ecosystem* group is a grab-bag of three semantic-memory language features, one agent CLI
   (`examples-search`) and one packages feature (`autonomous-package-updates`). (V9, V10, V17)
5. **Section-level redundancy — the "third copy" the north star forbids.** (V8, V12–V16)
   - `cross-project-messaging.mdx` (294 lines) says nothing `agent-messaging.md` does not already
     say (send with `--from/--type/--github`, list/read/ack, GitHub config, semantic search) —
     AND its *Storage* section asserts messages live only in local SQLite, which contradicts
     `agent-messaging.md`'s "one canonical cloud store, plus a private local one" (the model
     `CLAUDE.md` also mandates). Its only inbound link is from `agent-messaging.md` itself.
   - `development.md`: every section is either duplicated on the site (`architecture/index.md`
     layer table; `architecture/adding-operators.md`'s 8-step operator guide vs its 7-step
     list), stale-false (`internal/channels`, `internal/session`, `internal/typeclass` "TODO"
     directories do not exist; `ailang parse` is not a command), or repo-internal policy that
     belongs in `CONTRIBUTING.md`/`.claude/rules` (Testing Policy, Documentation Requirements).
   - `getting-started.mdx` § *For AI Agents: CLI Integration* is the third copy of the agent
     onboarding steps (`agent-integration.mdx` is the canonical one; `start-here/for-ai-agents`
     is the router). It also embeds an inline AILANG snippet in a ```` ```typescript ```` fence
     (a clause-2 defect in its own right) and quotes "Best model: Claude Sonnet 4.5".
   - `hooks-setup.mdx` § *Message System* repeats the `ailang messages` quick reference.
   - `semantic-caching-how-to.mdx` § *Two-Tier Search Architecture* repeats `semantic-search.md`'s
     subject (tiers, `_simhash`, `_ollama_embed`, hybrid pattern — `semantic-search.md` has its own
     *Hybrid Search Pattern* section at line 358).

## Scope rules (the executor follows these; the evaluator scores against them)

- **URL-stable.** No page is renamed or moved on disk. Categorisation changes are `sidebars.js`
  edits only. Reason: the site has no redirects plugin (`@docusaurus/plugin-client-redirects` is
  not in `docs/package.json`, V18), so a rename is a 404 for every external link.
- **Deletion criterion (clause 5 authorises deletion; this makes it checkable).** A page is deleted
  only when (i) every section is either already present on another live page, stale-false, or
  repo-internal policy, AND (ii) every inbound link is retargeted in the same commit. Both delete
  candidates satisfy (i) and (ii) — the evidence is in the Verification Log, not asserted.
- **Trim, don't annotate.** Where a section duplicates another page, the section is *replaced by a
  one-line pointer*, not prefixed with "see also". The charter is explicit that "add a clarifying
  note" is not an acceptable substitute for removal.
- **No content rewrites.** Stale *facts* inside pages that survive (e.g. `benchmarking.md`'s
  five-benchmark table) are clause-1 work and are listed under *Deferred* below — this sprint
  changes structure and removes duplicates, nothing else. Mixing the two is what the charter's
  original deferral warned against.
- **Blast radius**: `docs/sidebars.js`, `docs/docs/guides/**`, plus the four non-guide pages that
  link `guides/development` (`docs/docs/intro.mdx`, `docs/docs/examples.mdx`,
  `docs/docs/architecture/index.md`, `docs/docs/guides/debugging.md`). Nothing outside `docs/`.

## Phase A — mechanical (nav + cruft)

A1. `git rm docs/docs/guides/agent-integration.mdx.bak`.

A2. Rewrite the guide entries of `docs/sidebars.js` to the tree in **Appendix B**. Net effect on
    guide ids: 63 today → **70** (−2 deleted, +9 orphans wired). The `Ecosystem` and `Prompts`
    sub-categories disappear; a `Coordinator & Messaging` sub-category appears under *Build &
    Operate*; `AI Effect` and `Memory & Search` sub-categories appear under *Learn AILANG*. Every
    move is listed per-file in **Appendix A** — the executor does not decide any placement.

A3. Nothing else in Phase A. No frontmatter edits (`sidebar_position` values in the files are
    ignored because the sidebar is explicit, not autogenerated).

## Phase B — enumerated redundancy removal (five items, in this order)

B1. **Delete `docs/docs/guides/cross-project-messaging.mdx`.** Carry over exactly ONE thing first:
    its *Automated Feedback (Advanced)* CI-pipeline snippet (the ~15-line bash block under that
    heading) becomes a short subsection at the end of `agent-messaging.md` § *Workflows*, since
    `agent-messaging.md` has no CI example (V13). Then retarget `agent-messaging.md`'s "Related"
    link (currently line 887) — drop it, the page now IS the target. Remove the sidebar entry.
    Everything else on the page is already in `agent-messaging.md` (V12) or false (Storage §, V13).

B2. **Delete `docs/docs/guides/development.md`.** Retarget its four inbound links (V14) to the
    canonical contributor entry the site already uses for this purpose —
    `https://github.com/sunholo-data/ailang/blob/dev/CONTRIBUTING.md` (precedent:
    `docs/docs/feedback.mdx` line 220 links it exactly this way) — except `architecture/index.md`
    line 113, whose "Contributing to the codebase" bullet should point at the same
    `CONTRIBUTING.md` link, and `guides/debugging.md` line 848, which should point at
    `/docs/guides/development-workflow` (the page that actually describes the workflow). Remove
    the sidebar entry. Nothing is carried over: the make-target tables drift from `make help`
    (already do — `ailang parse` does not exist, V15), the layer table lives in
    `architecture/index.md`, the operator recipe lives in `architecture/adding-operators.md`.

B3. **Trim `getting-started.mdx` § *For AI Agents: CLI Integration*** (from that H2 up to, not
    including, `## … For Human Developers: Manual Installation`) to a three-line pointer: "If you
    installed manually, the agent onboarding path is [AI Agent Integration](/docs/guides/agent-integration):
    load `ailang prompt`, write, `ailang check`, `ailang run`." This also removes the inline
    ```` ```typescript ```` AILANG block and the "Best model" claim. Keep the preceding *MCP
    Servers* section untouched — `agent-mcp.md` links *into* it for the comparison table (V16).

B4. **Trim `hooks-setup.mdx` § *Message System*** (its three H3s: Checking Inbox / Sending
    Messages / Message Storage) to a one-line pointer at `/docs/guides/agent-messaging`. The
    surrounding *SessionStart Hook Behavior* section stays.

B5. **Trim `semantic-caching-how-to.mdx` § *Two-Tier Search Architecture*** (from that H2 up to,
    not including, `## Embeddings Doctrine`) to a one-line pointer at
    `/docs/guides/semantic-search` — which owns tiers, setup, and the hybrid pattern (V16).

**Not** in Phase B — considered and rejected with evidence, so the next iteration does not
rediscover them:

| Candidate | Why rejected |
|---|---|
| Delete/merge `agent-workflows.mdx` into `coordinator.md` | Not a subset: `chains diagnose`, `chains diff`, `hooks.log` troubleshooting appear in it and not in `coordinator.md` (V19). Its 20-line Quick Reference is the only redundant part; left in place because the page is a "worked workflows" page whose value is being self-contained. |
| Merge `notify-daemon.md` + `notification-channels.md` | Different audiences: the daemon page is operator install/config; the channels page is the Go `Channel`/`Registry` interface for adding a transport (V20). Both keep; they go in different categories (Operate vs Contributors). |
| Merge `semantic-search.md` + `semantic-caching-how-to.mdx` | Only one section overlaps (B5). The remaining 450+ lines each are distinct (Ollama setup/config/performance vs. patterns/determinism/embeddings doctrine). |
| Merge `coordinator.md` + `collaboration-hub.md` | Share only command names; hub page covers dashboard, API endpoints, DB schema — none of which `coordinator.md` has (V21). |
| Merge `three-camps-*` pair, `traces` + `telemetry`, `state-system-workflow` into the bridge page | Already done or already trimmed by the 2026-08-17 audit passes 5–6 (V22); different subjects for traces (program execution) vs telemetry (OTEL). |
| Delete `benchmarking.md` as superseded by `evaluation/` | Not superseded: no `evaluation/` page documents the custom-benchmark YAML spec or `--mock`/`expected_stdout` (V10). It is miscategorised and stale, not redundant → moved (A2), staleness deferred. |

## Acceptance

1. `find docs/docs/guides -maxdepth 1 -type f | wc -l` → **59** (62 − `.bak` − 2 pages).
2. The orphan check returns nothing:
   `cd docs && for f in $(find docs/guides -type f \( -name '*.md' -o -name '*.mdx' \) | sed -E 's|^docs/||; s|\.mdx?$||'); do grep -q "'$f'" sidebars.js || echo "ORPHAN: $f"; done`
   (note `sed -E` — BSD `sed` does not accept `\?` in basic regex; the plain-`sed` form of this
   loop reports every file as an orphan and was the first thing this brief got wrong).
3. `grep -o "'guides/[^']*'" docs/sidebars.js | sort -u | wc -l` → **70**, and no dangling ids
   (`for id in …; do [ -f docs/docs/$id.md ] || [ -f docs/docs/$id.mdx ] || echo DANGLING; done`
   prints nothing).
4. `grep -rlE "guides/(cross-project-messaging|development)([^a-z0-9_-]|$)" docs/docs docs/src docs/sidebars.js`
   → empty (all inbound links retargeted; `development-workflow` must still match its own name,
   hence the boundary in the regex).
5. Redundancy counts drop: `grep -l -- 'ailang messages ack --all' docs/docs/guides/*.md*` → 2
   files (was 4: `hooks-setup` and `cross-project-messaging` gone); `grep -l "_ollama_embed"
   docs/docs/guides/*.md*` no longer lists `semantic-caching-how-to.mdx`.
6. `bash docs/scripts/sync-registry.sh && make docs-build` green — with `onBrokenLinks: 'throw'`
   (V18) this is what proves every retargeted link resolves. Run it exactly this way: the deploy
   workflow runs `sync-registry.sh` before `make docs-build`, and without it the build fails at
   HEAD on gitignored `packages/sunholo/*` sidebar ids regardless of this sprint (V23).
7. `git diff --stat` touches only the files listed under *Files*.
8. CHANGELOG entry not required (site structure + duplicate removal; same precedent as docs-1/-3).

## Files

- `docs/sidebars.js` — guide entries rewritten to Appendix B (the bulk of the diff)
- `docs/docs/guides/agent-integration.mdx.bak` — deleted (A1)
- `docs/docs/guides/cross-project-messaging.mdx` — deleted (B1)
- `docs/docs/guides/development.md` — deleted (B2)
- `docs/docs/guides/agent-messaging.md` — gains the CI snippet subsection; drops one Related link (B1)
- `docs/docs/guides/getting-started.mdx` — one section trimmed to a pointer (B3)
- `docs/docs/guides/hooks-setup.mdx` — one section trimmed to a pointer (B4)
- `docs/docs/guides/semantic-caching-how-to.mdx` — one section trimmed to a pointer (B5)
- `docs/docs/intro.mdx`, `docs/docs/examples.mdx`, `docs/docs/architecture/index.md`,
  `docs/docs/guides/debugging.md` — one link each retargeted (B2)

## Deferred, not in this sprint (recorded so it is not rediscovered as new)

- **Clause 1 — stale facts inside surviving pages** (found while reading for this brief, V10,
  V15, V24): `benchmarking.md` lists 5 benchmarks (there are 93 `benchmarks/*.yml`) and a "Future
  Extensions" roadmap for v0.3.0–v0.5.0 at v0.34.0; `agent-workflows.mdx` carries a "Last updated:
  March 2026 (v0.9.0)" footer; `go-interop.md` opens with an "ABI Stability Notice (v0.5.x) …
  until v0.6.0"; `module_execution.mdx` is headed "AILANG v0.5.11"; `ai-prompt-guide.mdx` and
  `examples-search.mdx` were last touched 2025-12/2026-01 (`examples-search` says "97 working
  examples"). One clause-1 row per page, or one sweep item.
- **`tools/generate-llms-txt.sh` silently ships one guide instead of four** (V25). Its keep-list
  names `getting-started.md`, `ai-prompt-guide.md`, `module_execution.md` — all three are `.mdx`
  on disk — so the `[ -f "$file" ]` guard skips them and `llms.txt` contains a single
  `# Guide:` block (`wasm-integration.md`). Its comment also excludes `agent-integration` as
  "internal dev docs", which is the opposite of the nav's classification. A Critical-Principle-2
  shape (silent fallback in a published artifact) in `tools/**`, inside this mission's allowlist.
  Not folded in here because the fix is not mechanical: `cat`-ing MDX with JSX imports into
  `llms.txt` is probably wrong and the keep-list itself needs a decision. Queue as its own row.
- **`make docs-build` is red on a fresh checkout at HEAD** unless `docs/scripts/sync-registry.sh`
  runs first (V23); the workflow does this with `continue-on-error: true`, so a registry outage
  would fail the deploy build the same way. The charter calls `make docs-build` "the real gate"
  and this brief's acceptance uses the workflow's two-step form. The fix is in `make/docs.mk`
  (outside every allowlist pattern — flag, do not touch here).
- **Docusaurus deprecation**: `siteConfig.onBrokenMarkdownLinks` warns on every build (V23) —
  trivial config move, clause 3, separate row.

---

## Appendix A — disposition of every file in `docs/docs/guides/` (62 + 11)

Sidebar path notation: *L* = Learn AILANG · *A* = For AI Agents · *R* = Reference · *B* = Build &
Operate · *I* = Internals & Vision. "unchanged" = same category as today.

| # | File | Lines | Today | Action |
|---|------|------:|-------|--------|
| 1 | `agent-integration.mdx` | 117 | A (top) | keep, unchanged |
| 2 | `agent-integration.mdx.bak` | 402 | (not in nav; not served) | **DELETE** (A1) |
| 3 | `agent-mcp.md` | 139 | A › Harness | keep, unchanged |
| 4 | `agent-messaging.md` | 889 | A › Workflows | **move** → B › Coordinator & Messaging; **edit** (B1: +CI subsection, −Related link) |
| 5 | `agent-workflows.mdx` | 382 | A › Workflows | keep, unchanged |
| 6 | `ai-effect.mdx` | 323 | A › Prompts | **move** → L › AI Effect |
| 7 | `ai-prompt-guide.mdx` | 165 | A › Prompts | **move** → A (top level; *Prompts* sub-category dissolved) |
| 8 | `ai-routing.md` | 333 | A › Prompts | **move** → L › AI Effect |
| 9 | `ai-stdlib-discovery.md` | 72 | ORPHAN | **wire** → A › Harness |
| 10 | `ailang-vs-agents.mdx` | 311 | A (top) | keep, unchanged |
| 11 | `autonomous-package-updates.md` | 240 | B › Ecosystem | **move** → R › Stdlib & Packages |
| 12 | `benchmarking.md` | 351 | B › Develop | **move** → B › Evaluate (last item) |
| 13 | `brain-cache.md` | 286 | A › Harness | keep, unchanged |
| 14 | `build-a-motoko-extension.md` | 299 | R › Stdlib & Packages | keep, unchanged |
| 15 | `claude-code-integration.mdx` | 105 | A › Harness | keep, unchanged |
| 16 | `cloud-messaging-integration.md` | 1834 | A › Workflows | **move** → B › Coordinator & Messaging |
| 17 | `collaboration-hub.md` | 652 | A › Workflows | **move** → B › Coordinator & Messaging |
| 18 | `contracts.mdx` | 796 | L | keep, unchanged |
| 19 | `coordinator-setup.md` | 408 | I › Contributors | **move** → B › Coordinator & Messaging |
| 20 | `coordinator-workers.md` | 279 | ORPHAN | **wire** → B › Coordinator & Messaging |
| 21 | `coordinator.md` | 1495 | I › Contributors | **move** → B › Coordinator & Messaging (first item) |
| 22 | `cross-project-messaging.mdx` | 294 | A › Workflows | **DELETE** (B1) |
| 23 | `custom-ai-providers.md` | 205 | ORPHAN | **wire** → L › AI Effect |
| 24 | `database-architecture.md` | 168 | I › Contributors | keep, unchanged |
| 25 | `debugging.md` | 849 | B › Develop | keep; **edit** one link (B2) |
| 26 | `development-workflow.md` | 395 | B › Develop | **move** → I › Contributors |
| 27 | `development.md` | 242 | L | **DELETE** (B2) |
| 28 | `editor-setup.md` | 236 | L | keep, unchanged |
| 29 | `examples-search.mdx` | 241 | B › Ecosystem | **move** → A › Harness |
| 30 | `extension-packages.md` | 328 | R › Stdlib & Packages | keep, unchanged |
| 31 | `getting-started.mdx` | 481 | L | keep; **edit** (B3) |
| 32 | `go-interop.md` | 908 | B › Deploy & Embed | keep, unchanged |
| 33 | `hooks-setup.mdx` | 244 | A › Harness | keep; **edit** (B4) |
| 34 | `ifc-labels.mdx` | 397 | L | keep, unchanged |
| 35 | `lsp.md` | 107 | L | keep, unchanged |
| 36 | `microrag.md` | 319 | A › Harness | keep, unchanged |
| 37 | `mission-bootstrap.md` | 330 | A › Workflows | keep, unchanged |
| 38 | `mission-model-fleet.md` | 158 | A › Workflows | keep, unchanged |
| 39 | `module_execution.mdx` | 509 | L | keep, unchanged |
| 40 | `motoko-extension-development.md` | 239 | R › Stdlib & Packages | keep, unchanged |
| 41 | `notification-channels.md` | 151 | ORPHAN | **wire** → I › Contributors |
| 42 | `notify-daemon.md` | 175 | ORPHAN | **wire** → B › Coordinator & Messaging |
| 43 | `package-publishing.md` | 198 | R › Stdlib & Packages | keep, unchanged |
| 44 | `packages.md` | 500 | R › Stdlib & Packages | keep, unchanged |
| 45 | `parameterised-effects.md` | 282 | R › Language | keep, unchanged |
| 46 | `quick-start-examples.mdx` | 359 | L | keep, unchanged |
| 47 | `secret-approvals.md` | 167 | L | **move** → B › Coordinator & Messaging |
| 48 | `semantic-caching-how-to.mdx` | 502 | B › Ecosystem | **move** → L › Memory & Search; **edit** (B5) |
| 49 | `semantic-caching-vs-vectordb.md` | 177 | B › Ecosystem | **move** → L › Memory & Search |
| 50 | `semantic-search.md` | 550 | B › Ecosystem | **move** → L › Memory & Search (first item) |
| 51 | `serve-api.md` | 1442 | B › Deploy & Embed | keep, unchanged |
| 52 | `state-system-workflow.mdx` | 372 | A › Workflows | keep, unchanged |
| 53 | `streaming.md` | 422 | L | keep, unchanged |
| 54 | `strict-fallbacks.md` | 111 | ORPHAN | **wire** → L (after `ifc-labels`) |
| 55 | `telemetry.md` | 916 | B › Develop | keep, unchanged |
| 56 | `testing.md` | 605 | L | keep, unchanged |
| 57 | `three-camps-comparison.md` | 213 | A (top) | keep, unchanged |
| 58 | `three-camps-self-audit.md` | 276 | A (top) | keep, unchanged |
| 59 | `traces.md` | 344 | B › Develop | keep, unchanged |
| 60 | `wasm-ai-step-byo-key.md` | 241 | B › Deploy & Embed | keep, unchanged |
| 61 | `wasm-integration.md` | 1060 | B › Deploy & Embed | keep, unchanged |
| 62 | `workspaces.md` | 235 | A › Workflows | **move** → B › Coordinator & Messaging |

`evaluation/` (11 files, all stay in B › Evaluate): `README.mdx`, `architecture.md`,
`harness-setup.md`, `model-configuration.md`, `eval-loop.md`, `model-capability-threshold.md`,
`browser-sessions.md`, `browser-auth-profiles.md` — unchanged; `measurement-contract.md`,
`cost-and-speed-budgets.md`, `local-ollama.md` — **wire** (orphans today).

Tally: 47 keep-unchanged (incl. 8 eval) · 9 orphans wired · 16 sidebar moves · 6 page edits ·
3 deletions (1 cruft + 2 pages). Every `guides/` id in Appendix B corresponds to a row above.

## Appendix B — target sidebar tree (guide entries only; non-guide ids unchanged)

```
Learn AILANG
  guides/getting-started · guides/quick-start-examples · guides/editor-setup · guides/lsp
  guides/module_execution · guides/testing · guides/streaming · guides/contracts
  guides/ifc-labels · guides/strict-fallbacks
  ▸ AI Effect:        guides/ai-effect · guides/ai-routing · guides/custom-ai-providers
  ▸ Memory & Search:  guides/semantic-search · guides/semantic-caching-how-to
                      guides/semantic-caching-vs-vectordb
For AI Agents
  guides/ailang-vs-agents · guides/three-camps-comparison · guides/three-camps-self-audit
  guides/agent-integration · guides/ai-prompt-guide
  ▸ Harness:   guides/claude-code-integration · guides/agent-mcp · guides/hooks-setup
               guides/brain-cache · guides/microrag · guides/ai-stdlib-discovery
               guides/examples-search
  ▸ Workflows: guides/agent-workflows · guides/state-system-workflow
               guides/mission-bootstrap · guides/mission-model-fleet
Reference
  ▸ Language:          … guides/parameterised-effects … (unchanged)
  ▸ Stdlib & Packages: reference/stdlib · guides/packages · guides/extension-packages
                       guides/build-a-motoko-extension · guides/motoko-extension-development
                       guides/package-publishing · guides/autonomous-package-updates
                       ▸ Browse Packages (unchanged)
Build & Operate
  ▸ Develop:                  guides/debugging · guides/traces · guides/telemetry
  ▸ Deploy & Embed:           guides/go-interop · guides/wasm-integration
                              guides/wasm-ai-step-byo-key · guides/serve-api
  ▸ Coordinator & Messaging:  guides/coordinator · guides/coordinator-setup
                              guides/coordinator-workers · guides/collaboration-hub
                              guides/agent-messaging · guides/cloud-messaging-integration
                              guides/workspaces · guides/secret-approvals · guides/notify-daemon
  ▸ Evaluate:  guides/evaluation/README · architecture · harness-setup · model-configuration
               eval-loop · measurement-contract · cost-and-speed-budgets
               model-capability-threshold · local-ollama · browser-sessions
               browser-auth-profiles · guides/benchmarking
  ▸ Benchmarks Dashboard (unchanged)
  (Ecosystem — removed; all five members re-homed above)
Internals & Vision
  ▸ Contributors: guides/development-workflow · guides/database-architecture
                  guides/notification-channels
```

Guide-id count: L 10+3+3 · A 5+7+4 · R 1+6 · B 3+4+9+12 · I 3 = **70**.

---

## Verification Log

Every claim above about current file/content state carries a row here. Commands were run at
HEAD `55891002f` in `/Users/voightkampff/.ailang-driver-pin/docs` on 2026-09-03; `ailang` is
`v0.34.0-346-g327db37cd-dirty` (binary older than HEAD — only used for command-existence checks,
which do not depend on the 52-commit delta). One claim came back **false** during authoring and
is recorded as such (V4).

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | 62 files directly under `guides/` | `find docs/docs/guides -maxdepth 1 -type f \| wc -l` | `62` |
| V2 | 61 pages + 1 `.bak`; one subdir with 11 pages | `find docs/docs/guides -maxdepth 1 -type f -exec wc -l {} +`; `ls docs/docs/guides/evaluation` | 61 `.md/.mdx` + `agent-integration.mdx.bak` (402 lines); `evaluation/` = README.mdx, architecture, browser-auth-profiles, browser-sessions, cost-and-speed-budgets, eval-loop, harness-setup, local-ollama, measurement-contract, model-capability-threshold, model-configuration |
| V3 | `.bak` is tracked cruft, superseded, not served | `git ls-files docs/docs/guides/agent-integration.mdx.bak`; `git log --format="%h %as %s" -- <file>`; `diff <bak> <mdx> \| head -60`; `curl -s -o /dev/null -w "%{http_code}" https://ailang.sunholo.com/docs/guides/agent-integration.mdx.bak` | tracked; `f8f0c0976 2025-11-13 Fix website deployment: Update prompt references to v0.4.4` (only commit); diff shows `.bak` has per-platform tarball install + `prompts/v0.3.8.md`, `.mdx` has `install.sh` + `ailang prompt` + `ACTIVE_PROMPT` import; live URL `404` (controls: `guides/getting-started` → `200`, `guides/does-not-exist-control` → `404`) |
| V4 | 6 top-level + 3 `evaluation/` pages absent from `sidebars.js`; sidebar holds 63 guide ids | `cd docs && for f in $(find docs/guides -maxdepth 1 -type f \( -name '*.md' -o -name '*.mdx' \) \| sed -E 's\|^docs/\|\|; s\|\.mdx?$\|\|'); do grep -q "'$f'" sidebars.js \|\| echo "ORPHAN: $f"; done` (and `-mindepth 2` variant); `grep -o "'guides/[^']*'" sidebars.js \| sort -u \| wc -l` | ORPHAN: ai-stdlib-discovery, coordinator-workers, custom-ai-providers, notification-channels, notify-daemon, strict-fallbacks; evaluation/cost-and-speed-budgets, evaluation/local-ollama, evaluation/measurement-contract; `63`. **False first result recorded:** the same loop with `sed 's\|\.mdx\?$\|\|'` (BSD basic-regex `\?`) reported all 61 as orphans — extension never stripped. Re-run with `-E` before any conclusion was drawn. |
| V5 | Five orphans have zero inbound links; `custom-ai-providers` has 2 (recipes); `measurement-contract` zero | per-guide `grep -rl --include='*.md' --include='*.mdx' --include='*.js' --include='*.jsx' -E "guides/$f([^a-z0-9_-]\|$)" docs src sidebars.js` excluding self | `0` for ai-stdlib-discovery, coordinator-workers, notification-channels, notify-daemon, strict-fallbacks; `custom-ai-providers` ← `recipes/ai-tool-loop.md`, `recipes/ai-token-streaming.md`; `measurement-contract` ← none (`cost-and-speed-budgets` ← design-docs.md, evaluation/README.mdx, benchmarks/value.md; `local-ollama` ← design-docs.md, coordinator-workers.md, benchmarks/os-model-leaderboard.md) |
| V6 | Orphans are built and served | `curl -s -o /dev/null -w "%{http_code}" https://ailang.sunholo.com/docs/guides/<id>` for all 9 orphans (round-1 quorum objection `gpt5-6-sol`: the first pass probed only 8, omitting `evaluation/cost-and-speed-budgets`) | all `200`, now including the 9th: notify-daemon, notification-channels, coordinator-workers, strict-fallbacks, ai-stdlib-discovery, custom-ai-providers, evaluation/measurement-contract, evaluation/local-ollama, evaluation/cost-and-speed-budgets |
| V7 | Literal page-level duplication ≈ 0 | for each page: strip blank/heading/table/list/fence/import/frontmatter lines, keep lines >40 chars, `sort -u`; then `comm -12` over all pairs, report ≥4 (run under `bash -c` — zsh does not word-split `$files`, the first run errored "file name too long") | Only `5 agent-workflows.mdx claude-code-integration.mdx` and `4 getting-started.mdx quick-start-examples.mdx`; every other pair <4. Largest unique-line pools: cloud-messaging-integration 392, serve-api 265, coordinator 258 |
| V8 | Same command blocks recur across 4–7 guides | `grep -l -- '<string>' docs/docs/guides/*.md*` for each string | `ailang messages send user` → agent-messaging, agent-workflows, claude-code-integration, cross-project-messaging, hooks-setup (5); `ailang messages ack --all` → agent-messaging, hooks-setup, agent-workflows, claude-code-integration (4); `make services-start` → 6 files; `ailang coordinator status` → 7; `trigger_on_complete` → 6; `collaboration.db` → 8 |
| V9 | `development.md` is contributor content, stale on structure, and classified "internal dev docs" by the llms.txt tool | read whole file; `ls -d internal/effects internal/channels internal/session internal/typeclass`; `sed -n 60,80p tools/generate-llms-txt.sh` | H2s: Development Workflow (make targets), Project Structure, Adding a New Language Feature, Adding a Binary Operator, Adding a Built-in Function, Testing Guidelines, Code Style, Error Handling, Debug Commands, Performance, Documentation Requirements, Testing Policy; `internal/channels`, `internal/session`, `internal/typeclass`: `No such file or directory` (listed as "(TODO)" in the page); tool comment: `# EXCLUDED: development.md, agent-integration.md, benchmarking.md (internal dev docs)` |
| V10 | `benchmarking.md` documents a live command, is not covered by `evaluation/`, and is stale on scale | `ailang eval`; `ls benchmarks/*.yml \| wc -l`; `grep -rnE "expected_stdout\|--mock\|difficulty:" docs/docs/guides/evaluation/*.md*`; read page | `Error: --benchmark flag is required / Usage: ailang eval --benchmark <id> …` (command exists); `93` yml files vs the page's 5-row *Available Benchmarks* table; no `evaluation/` page mentions the YAML spec, `--mock` or `expected_stdout` (only `eval-loop.md`/`model-configuration.md` show `ailang eval --benchmark` invocations); page ends with "Future Extensions: v0.3.0 / v0.4.0 / v0.5.0" |
| V11 | Coordinator cluster split across two categories; `coordinator-setup` is user-facing | `grep -n "label:\|guides/" docs/sidebars.js`; `head -3 docs/docs/guides/coordinator-setup.md` | `guides/coordinator`, `guides/coordinator-setup` at lines 263–264 under `label: 'Contributors'` (line 261, inside `Internals & Vision`, `collapsed: true`); `guides/collaboration-hub`, `guides/cloud-messaging-integration`, `guides/workspaces`, `guides/agent-messaging` at lines 99–103 under `label: 'Workflows'` inside `For AI Agents`; title "Setting Up the Coordinator for External Projects" |
| V12 | `cross-project-messaging.mdx` content already in `agent-messaging.md` | read both; `grep -nE -- "--github\|--type bug\|--from \|external project" docs/docs/guides/agent-messaging.md` | agent-messaging: description "send and receive messages between AILANG core and external projects" (line 4/9); `--from` (28), `--type bug --github` (31, 226, 278), `--github` flag section (221), `list --from` (244), semantic-search fields section, GitHub workflow section. cross-project's H2s: Quick Start, Message Types, Full Message Format, Examples, GitHub Configuration, How Messages Are Processed, Checking Messages, Storage, Best Practices, Automated Feedback, Semantic Search, Related — each mapped to an agent-messaging section except the CI snippet |
| V13 | cross-project's Storage § contradicts the canonical model; CI snippet absent from agent-messaging | `sed -n 203,218p cross-project-messaging.mdx`; `sed -n '/^## Storage Backend/,/^## /p' agent-messaging.md`; `grep -nE "CI \|ci pipeline" agent-messaging.md` | cross-project: "Messages are stored in a SQLite database … `~/.ailang/state/collaboration.db` … shared between … All AILANG instances on the machine"; agent-messaging: "one canonical store (prod Firestore, `ailang-multivac`) plus a private local one … by default `ailang messages` reads only the local one"; no CI/pipeline example in agent-messaging (only hit is the Related link at line 887) |
| V14 | Inbound links to delete candidates | `grep -rlE "guides/<name>([^a-z0-9_-]\|$)" docs/docs docs/src docs/sidebars.js`; `grep -n "guides/development\b" …` | `cross-project-messaging` ← `guides/agent-messaging.md` (line 887), `sidebars.js`; `development` ← `intro.mdx:146` ("Development Setup"), `examples.mdx:605` ("Read the guide"), `architecture/index.md:113` ("Contributing to the codebase"), `guides/debugging.md:848` ("Full development workflow"), `sidebars.js` |
| V15 | `development.md`'s Debug Commands cite a non-existent command; site precedent for linking CONTRIBUTING.md | `ailang parse /dev/null`; `ls -la CONTRIBUTING.md`; `grep -rn CONTRIBUTING docs/docs` | `Error: unknown command 'parse'`; `CONTRIBUTING.md` exists (7440 B, H2s: How Your Contribution Becomes Code, Ways to Contribute …); `docs/docs/feedback.mdx:220` links `https://github.com/sunholo-data/ailang/blob/dev/CONTRIBUTING.md` |
| V16 | getting-started § is a third copy with an inline `typescript`-fenced snippet; `agent-mcp` links into getting-started's MCP table; semantic-search owns the hybrid pattern | `awk '/CLI Integration/,/For Human Developers/' getting-started.mdx`; `grep -n "getting-started#" agent-mcp.md`; `grep -in hybrid semantic-search.md` | section = Step 2 (`ailang prompt`), Step 3 (```` ```typescript ```` block containing `module benchmark/solution …`), Step 4 (`ailang run --caps IO,FS …`), "AI Success Metrics — Best model: Claude Sonnet 4.5"; agent-mcp line 24 → `/docs/guides/getting-started#-mcp-servers`; semantic-search line 358 `## Hybrid Search Pattern` |
| V17 | `ai-effect`/`ai-routing`/`custom-ai-providers` are language features; `secret-approvals` is ops; `examples-search` is agent CLI | first 15 lines of each | "AI Effect: Calling LLMs from AILANG"; "AI Provider Routing (OpenRouter) … AILANG ships an OpenRouter adapter"; "Add a new AI provider as a package … `[[ai_provider]]` block in your `ailang.toml`"; secret-approvals H2s: Status, Architecture, Prerequisites, Deploy (dev first), Coordinator and executor configuration, Phone setup, Test it; examples-search: "`ailang examples search` … especially useful for AI models learning AILANG patterns" |
| V18 | No redirects plugin; broken internal links fail the build | `grep -nE '"@docusaurus/(core\|plugin-client-redirects)"' docs/package.json`; `grep -n onBrokenLinks docs/docusaurus.config.js` | only `"@docusaurus/core": "3.10.2"` (no redirects plugin); `onBrokenLinks: 'throw'` (line 33), `onBrokenMarkdownLinks: 'warn'` (34) |
| V19 | `agent-workflows.mdx` is not a subset of `coordinator.md` | `grep -c` of each agent-workflows command in coordinator.md / claude-code-integration.mdx / agent-messaging.md / telemetry.md | `chains diagnose` 0/0/0/0, `chains diff` 0/1/0/0, `hooks.log` 0/1/0/0 (unique to agent-workflows + the hub); `env_from_payload` 4/0/0/0, `coordinator logs` 4/0/0/0, `github_sync` 3/0/0/1 (covered) |
| V20 | `notify-daemon` vs `notification-channels` are different audiences | read both in full | daemon: `make install-notify-daemon`, subcommands table, dedup, `daemon.yaml`, dual-subscribe, launchd troubleshooting; channels: `internal/notify` `Channel`/`Registry` Go interfaces, "Adding a new channel (~1–2h)" with Go code, dead-lettering via `gcloud pubsub` |
| V21 | `collaboration-hub` ≠ `coordinator` | `grep -n "^## " …` on both | hub H2s include Database Schema, ID Relationships Across Databases, API Endpoints; coordinator H2s include Agent Configuration (lines 32–413), Workflow Pipelines, Cloud Mode, Feedback Cost & Abuse Gate — disjoint except Quick Start/Troubleshooting |
| V22 | Page-level merges were already done 2026-08-17 | `git log --format="%h %as %s" --grep="audit pass" -- docs/` | 6 commits: `0002c9b0b` fold comparison essays (pass 6), `d51306eef` trim bridge/state pages (pass 5), `56c200c84` rewrite agent-integration slim (pass 4), `23793bc25` compress semantic-caching-vs-vectordb (pass 3), `a705f4954` remove superseded pages (pass 2), `03a251785` stale versions (pass 1) |
| V23 | Baseline build: `make docs-build` alone is red at HEAD; the workflow form is the gate | `make docs-build` (bare); `grep -n "sync-registry\|make docs-build" .github/workflows/docusaurus-deploy.yml`; `grep -n packages docs/.gitignore`; then `bash docs/scripts/sync-registry.sh && make docs-build` | bare: WASM ok, sync-all ok, then `[ERROR] Invalid sidebar file at "sidebars.js". These sidebar document ids do not exist: packages/sunholo/a2ui …` → `make: *** [docs-build] Error 1` (9.8 s); workflow lines 290–295: `run: bash docs/scripts/sync-registry.sh` / `continue-on-error: true` / `run: make docs-build`; `.gitignore` lines 40–41: `/docs/packages/*/`, `/src/data/packages-sidebar.json`; workflow-form result: **see V23b** |
| V23b | Workflow-form baseline build is green at HEAD | `bash docs/scripts/sync-registry.sh; make docs-build`; then `git status --short` | `sync-registry exit=0`; `[SUCCESS] Generated static files in "build".`; `docs-build exit=0`. Side-effect: the build's sync scripts modify four TRACKED files (`docs/docs/design-docs.md`, `docs/docs/prompts/current.md`, `docs/docs/roadmap/index.md`, `docs/src/data/packages-sidebar.json`) — restored with `git checkout --` before committing this brief. Nuance for the deferred `make/docs.mk` item: `packages-sidebar.json` is tracked despite the `.gitignore` rule, so the committed sidebar data points at the gitignored `packages/sunholo/*` pages — that is the exact mechanism of the bare-build red. The executor must NOT commit those four files either (acceptance 7). |
| V24 | Stale markers in surviving pages (deferred list) | `grep -nE "^\*\*(Last updated\|Status\|Version)" guides/*.md*`; `git log -1 --format=%as -- <file>`; `cat std/VERSION` | `agent-workflows.mdx:382 **Last updated:** March 2026 (v0.9.0)`; `module_execution.mdx:507 **Version**: v0.5.11`; go-interop opens "ABI Stability Notice (v0.5.x) … until v0.6.0"; last commits: ai-prompt-guide 2025-12-17, examples-search 2026-01-02, development 2026-01-16, agent-workflows/hooks-setup/cross-project-messaging 2026-03-05; `std/VERSION` = `v0.34.0` |
| V25 | `tools/generate-llms-txt.sh` ships one guide, not four | `sed -n 60,80p tools/generate-llms-txt.sh`; `for g in getting-started.md ai-prompt-guide.md module_execution.md wasm-integration.md; do [ -f docs/docs/guides/$g ] && echo present \|\| echo MISSING; done`; `grep -n "^# Guide:" llms.txt` | keep-list = those four names guarded by `if [ -f "$file" ]`; `MISSING MISSING MISSING present`; `llms.txt` line 197 `# Guide: wasm-integration.md` is the only guide block |
| V26 | Planner-lane mechanics; every touched path is inside the allowlist | `grep -n "Planner-Lane\|codex-ok\|opus-required" tools/launchd/derive-planner-lane.sh`; `grep -n MISSION_PLANNER_ALLOWLIST ~/.config/ailang/mission-docs.env` | field read as `^\*\*Planner-Lane\*\*:` with values `codex-ok\|opus-required` (lines 84–91); mission allowlist includes `docs/*` (env line 130); all *Files* above are `docs/…` |
| V27 | Coverage/duplicate gate: no existing design doc for this work; no clause-5 rows already queued | `grep -rli "taxonomy pass\|guides taxonomy\|consolidate.*guides\|sidebar.*restructur" design_docs/planned design_docs/implemented`; `grep -nE "clause 5\|taxonomy\|orphan\|duplicate" docs/docs-sync-findings.md` | both empty except findings line 55 stating no clause-5 findings were produced by docs-2 |
| V28 | Sidebar tree today (for Appendix B's "unchanged" claims) | `grep -n "label:\|guides/" docs/sidebars.js` | full listing captured at authoring (63 guide ids across Learn AILANG 11, For AI Agents 4+3+5+9, Reference 1+5, Build & Operate 5+4+8+5, Contributors 3); `git log -1 -- docs/sidebars.js` → `6780f72c6 2026-08-26` |
| V29 | B4/B5 cut boundaries exist and are adjacent in the asserted order (round-1 quorum objection `gemini-3-1-pro`) | `grep -nE '^##+ ' docs/docs/guides/hooks-setup.mdx docs/docs/guides/semantic-caching-how-to.mdx` | hooks-setup.mdx: `124:## Message System` immediately followed by its three named H3s (`126 Checking Inbox`, `146 Sending Messages`, `158 Message Storage`), then `162:## SessionStart Hook Behavior` — B4's boundary is exact. semantic-caching-how-to.mdx: `200:## Two-Tier Search Architecture` (with `204`/`217`/`230` H3s) immediately followed by `246:## Embeddings Doctrine` — B5's boundary is exact. |
