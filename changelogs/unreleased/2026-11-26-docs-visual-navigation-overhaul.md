### Documentation

- **Website visual & navigation overhaul (`M-DOCS-VISUAL-NAVIGATION-OVERHAUL`)**:
  - Shrank top navbar (`10 -> 6` items) and compressed `sidebars.js` into a 5-section progressive-disclosure hierarchy (`Start Here`, `Learn AILANG`, `For AI Agents`, `Reference`, `Operate & Internals`, max depth `3`, `14` visible items on load, `0` orphaned user pages).
  - Installed `@docusaurus/plugin-client-redirects` covering all merged guides (`three-camps-self-audit`, `recipes/ai-tool-loop`), 24 historical/versioned `/docs/prompts/v*` routes, and 10 relocated fleet runbooks (`docs/internal/runbooks/`) with zero 404s.
  - Added `<TierBanner />` theme wrapper (`docs/src/theme/DocItem/Content/index.js`) rendering automatic Experimental and Internal/Operator banners from `tier:` frontmatter.
  - Authored 8 brand-palette vector SVGs (`docs/static/img/diagrams/*.svg`), converted all 52 ASCII-art box diagrams to Mermaid, cut prose walls across the top 20 guides, refreshed all overdue `reviewBy` dates (`26 -> 0`), and taught every `v0.60`–`v0.64.1` construct (`|>`, destructuring, `_?`, `deriving (Json)`, `expr?`, `ailang inspect`, `check --fix`, Z3 `int64`/tuples, IFC, `--bytecode`, atomic `ailang.lock`).
  - Split `/llms.txt` (`<35 KB` `llmsstxt.org` routing index with 12-row Task->CLI->MCP table) and `/llms-full.txt` (complete reference corpus + verbatim active prompt) and updated `make check-llms-txt` / `make test-check-llms-txt`.
