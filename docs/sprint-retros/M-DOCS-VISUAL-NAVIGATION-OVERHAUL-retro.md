# Sprint Retro: M-DOCS-VISUAL-NAVIGATION-OVERHAUL

**Date**: 2026-11-26
**Branch**: `docs/visual-navigation-overhaul`
**Design Doc**: `design_docs/planned/v1_0_0/m-docs-visual-navigation-overhaul.md`
**Sprint Plan**: `design_docs/planned/v1_0_0/sprint-plan-m-docs-visual-navigation-overhaul.md`

## Summary

Executed Milestones M1–M6 across 5 pairwise-disjoint parallel Wave 1 slices (`visual-start-here`, `core-guides`, `agent-guides`, `ref-foundations`, `operate-split`) plus Wave 2 integration (`ia-nav-llms-integrator`):

- **Zero merge conflicts**: All 5 Wave 1 branches merged cleanly into `docs/visual-navigation-overhaul`.
- **Navigation & IA (`D1=a`)**: Shrank top navbar `10 -> 6` items, compressed `sidebars.js` into 5 top-level sections (`Start Here`, `Learn AILANG`, `For AI Agents`, `Reference`, `Operate & Internals`, max depth `3`, `14` visible items on initial load, `0` orphaned user pages), fixed `editUrl` to `dev`, and configured `@docusaurus/plugin-client-redirects` for all merged/moved/unlisted routes (zero 404s).
- **Tier Banners & Runbook Split (`D3=a`)**: Relocated 10 internal operator/fleet runbooks to `docs/internal/runbooks/` and swizzled `docs/src/theme/DocItem/Content/index.js` (`<TierBanner />`) to render automatic Experimental and Internal/Operator callouts from frontmatter `tier:`.
- **Visuals & Prose Concision (`D2=a`, `D5=a`)**: Added 8 dark/light-safe brand SVGs in `docs/static/img/diagrams/`, replaced all 52 ASCII-art box diagrams with Mermaid, removed 23 superseded `docs/docs/prompts/v*.md` snapshots (`58,539L`), cut prose walls across the top 20 pages, and eliminated all overdue/missing `reviewBy` dates (`26 -> 0`).
- **Agent `llms.txt` / `llms-full.txt` Split (`D4=a`)**: Fixed all broken paths in `tools/generate-llms-txt.sh` and emitted a concise `<35 KB` `llmsstxt.org` routing index (`/llms.txt`) alongside `/llms-full.txt` (bundling `README.md`, `for-ai-agents.mdx`, `getting-started.mdx`, 9 core reference pages, and verbatim active prompt `v0.16.20`), updating `scripts/check_llms_txt.sh` and its 7-arm self-test.

## Notable Catch During Integration

- Static component xref in `/tmp/docs_audit.py` only checked `.md`/`.mdx` and `index.jsx` imports, flagging `BenchmarkDashboard`, `DataProvenance`, and `LocalCloudBadge` as unused. Cross-component `git grep` before deletion revealed all three are actively imported by `AgentEvalPerformance`, `BenchmarkExplorer`, `BenchmarkStandaloneGallery`, `EloLeaderboard`, `OSLocalLeaderboard`, and `ValueDashboard`. Only the 2 genuinely unreferenced entries (`ModelRadarComparison/` and `docs/src/components/ailang-repl.js`) were deleted.
- Placing `...packagesSidebar` directly under `Reference` (instead of nested under a sub-category) kept sidebar `max_depth` strictly `<= 3`.

## Next Step

- Hand off to `docs-verifier` (`M7`) for independent full-site verification (`TST-DOCOV-01..12`) and before/after audit scorecard.
