# Sprint Plan: M-PKG-EXPLORER — Package Explorer Website

## Summary

Build a hybrid static+dynamic package explorer on the AILANG docs site (ailang.sunholo.com). Adds read-only API endpoints to the registry-validator, a build-time sync script that generates per-vendor/per-package MDX pages, and interactive React components for version timelines, provenance chains, dependency graphs, and ecosystem stats.

**Duration:** 5 milestones, ~11 days
**Dependencies:** Registry validator deployed (done), Docusaurus site deployed (done), 14+ packages published (done)
**Risk Level:** Medium (cross-stack: Go API + shell script + React + CI pipeline)

## Current Status Analysis

### Completed Recently
- M-PKG-AUTONOMOUS-UPDATES: ~1,200 LOC in 4 milestones (provenance, history, cascade)
- M-OBS-RETENTION: ~390 LOC in 2 milestones (observatory retention)
- M-DX-PKG-TOOLS: ~1,800 LOC (package tooling, check --package, module_prefix)
- Registry validator: 773 LOC (Cloud Run, /publish, /rebuild-index, /health, /version)

### Velocity
- Recent average: ~300-400 LOC/day (Go) based on last 14 days
- Frontend (React/JSX): BenchmarkDashboard is ~4,100 LOC total — built as a single feature
- Estimated capacity: ~400 LOC/day mixed Go+React

### Remaining from Design Doc
- Phase 1: Registry API endpoints (~500 LOC Go)
- Phase 2: Build-time sync + static pages (~320 LOC shell/mdx)
- Phase 3: PackageDetail + timeline + provenance (~770 LOC React)
- Phase 4: Explorer + graph + stats (~1,100 LOC React)
- Phase 5: Polish + deploy (~100 LOC mixed)
- **Total: ~2,790 LOC**

## Proposed Milestones

### M1: Registry Validator API Endpoints
**Goal:** Add read-only `/api/*` endpoints to the existing registry-validator Cloud Run service, with caching and CORS.
**Estimated:** 250 LOC implementation + 150 LOC tests = ~400 LOC
**Duration:** 2 days

**Tasks:**
- Create `cmd/registry-validator/cache.go` — in-memory cache with 5-min TTL for index.json and per-package metadata, invalidated on publish
- Create `cmd/registry-validator/handlers_api.go`:
  - `GET /api/packages` — return cached index.json
  - `GET /api/packages/:vendor/:name` — aggregate version metadata + history from GCS, compute dependents
  - `GET /api/packages/:vendor/:name/:version` — single version metadata + history
  - `GET /api/stats` — compute ecosystem stats from index (effect distribution, stability breakdown, publish timeline, top depended-on, agent vs human, etc.)
- Add CORS middleware allowing `ailang.sunholo.com` and `localhost:3000` origins
- Register new handlers in `main.go`
- Create `cmd/registry-validator/handlers_api_test.go` — mock GCS, test all endpoints
- Verify locally: `go run ./cmd/registry-validator/` + `curl localhost:8080/api/packages`

**Acceptance Criteria:**
- [ ] `GET /api/packages` returns valid JSON matching RegistryIndex schema
- [ ] `GET /api/packages/sunholo/auth` returns package detail with versions, metadata, dependents
- [ ] `GET /api/packages/sunholo/auth/0.1.0` returns single version metadata + history
- [ ] `GET /api/stats` returns ecosystem stats with all fields populated
- [ ] CORS headers present for ailang.sunholo.com origin
- [ ] Cache invalidation: publish triggers cache refresh
- [ ] All tests passing, linting clean

**Risks:**
- GCS read latency for aggregating metadata — Mitigation: cache aggressively, pre-aggregate on first request

---

### M2: Build-Time Registry Sync & Static Page Generation
**Goal:** Create a sync script that fetches registry data at build time and generates per-vendor/per-package MDX pages with correct Docusaurus sidebar integration.
**Estimated:** 200 LOC sync script + 70 LOC templates + 50 LOC sidebar/config = ~320 LOC
**Duration:** 2 days

**Tasks:**
- Create `docs/scripts/sync-registry.sh`:
  - Fetch index.json from registry-validator `/api/packages` (with fallback to direct GCS)
  - Fetch per-package detail via `/api/packages/:vendor/:name`
  - Write JSON snapshots to `docs/static/registry/`
  - Group packages by vendor, generate `docs/docs/packages/{vendor}/` directories
  - Generate `{vendor}/index.mdx` from vendor-index template
  - Generate `{vendor}/{name}.mdx` from package-page template
  - Generate `docs/src/data/packages-sidebar.json` with nested vendor categories
  - Graceful failure: if API unavailable, exit 0 with warning (build continues without registry pages)
- Create `docs/src/templates/vendor-index.mdx.tmpl` — vendor index page template
- Create `docs/src/templates/package-page.mdx.tmpl` — package detail page template
- Create hand-written `docs/docs/packages/index.mdx` — top-level overview
- Update `docs/sidebars.js` to import generated `packages-sidebar.json`
- Add `sync-registry.sh` to `.github/workflows/docusaurus-deploy.yml`
- Add generated paths to `.gitignore`
- Implement `VendorIndex.jsx` — card grid component for vendor index pages
- Verify: run sync locally, then `cd docs && npm run build` succeeds

**Acceptance Criteria:**
- [ ] `sync-registry.sh` generates MDX pages for all 14+ packages grouped by vendor
- [ ] Vendor index page at `/docs/packages/sunholo/` lists all sunholo packages
- [ ] Package detail page at `/docs/packages/sunholo/auth` renders static manifest data
- [ ] Sidebar shows nested vendor categories with correct links
- [ ] `docusaurus build` succeeds with generated pages
- [ ] Script exits gracefully when registry unavailable
- [ ] Generated files are gitignored

**Risks:**
- Docusaurus sidebar dynamic import may need specific pattern — Mitigation: test with `require()` of JSON file
- Registry API may not be deployed yet — Mitigation: sync script can also use direct GCS `gsutil cp` or `curl` to public bucket URL

---

### M3: Package Detail Components (Static + Live Hydration)
**Goal:** Build React components that render on the generated static pages, showing manifest details immediately and hydrating with live version timeline and provenance data.
**Estimated:** 300 LOC PackageDetail + 200 LOC VersionTimeline + 150 LOC ProvenanceChain + 120 LOC hooks = ~770 LOC
**Duration:** 2.5 days

**Tasks:**
- Create `docs/src/hooks/useRegistryData.js` — generic hook: tries live API, falls back to `/registry/` static snapshot, localStorage SWR
- Create `docs/src/components/PackageExplorer/PackageDetail.jsx`:
  - Renders static props immediately (no spinner for basic info)
  - Fetches live data for version list, provenance, dependents
  - Shows manifest table, exports list, effect badges, dependency/dependent lists
- Create `VersionTimeline.jsx` — vertical timeline with expand/collapse per version
  - Shows version, date, author, summary, change class badge
  - "View Provenance" expand button
- Create `ProvenanceChain.jsx` — expanded provenance view:
  - Validation results (compiles, effects valid, contracts)
  - Three hashes (content, interface, tarball) + size
  - Trigger message, change class, approval info
  - Message trail with timestamps and status badges
- Create `docs/src/components/PackageExplorer/styles.module.css` — CSS modules for all components
- Wire components into generated MDX pages (update templates if needed)
- Test with local `docusaurus start` against live or mock data

**Acceptance Criteria:**
- [ ] PackageDetail renders static data without API call (no loading spinner for basic info)
- [ ] Version timeline shows all versions with correct dates and summaries
- [ ] Provenance chain displays validation, hashes, approval info, message trail
- [ ] Live data hydration works when API available
- [ ] Graceful fallback when API unavailable (static data shown, "data may be stale" banner)
- [ ] Dark/light theme compatible
- [ ] All styles use CSS modules

**Risks:**
- CORS may block live API calls from localhost during dev — Mitigation: validator CORS includes localhost:3000

---

### M4: Interactive Explorer, Dependency Graph & Stats
**Goal:** Build the full interactive explorer page with live search/filter, the dependency graph visualization, and ecosystem stats dashboard.
**Estimated:** 300 LOC explorer + 150 LOC PackageCard + 300 LOC DependencyGraph + 200 LOC EcosystemStats + 30 LOC MDX pages = ~980 LOC
**Duration:** 3 days

**Tasks:**
- Create `PackageExplorer.jsx` — full interactive explorer:
  - Search across name, ai_summary, tags
  - Filter by effects, stability, tags (multi-select dropdowns)
  - Sort by name, last updated, version count
  - Package cards with expand/collapse for inline detail
  - Uses live API via `usePackageIndex()` hook
- Create `PackageCard.jsx` — reusable card with effect badges, stability badge, tags, version, relative time
- Create `DependencyGraph.jsx` — interactive graph:
  - Use d3-hierarchy tree layout or d3-force (agent decision)
  - Nodes colored by effect ceiling (green=pure, blue=IO, orange=Net+FS)
  - Node size proportional to dependent count
  - Edges show dependency direction
  - Hover tooltips with package summary
  - Click navigates to static package page
  - Effect filter dropdown
- Create `EcosystemStats.jsx` — charts using Recharts:
  - Effect distribution bar chart
  - Stability breakdown pie chart
  - Agent vs human updates bar chart
  - Top depended-on packages horizontal bar chart
  - Collapsible section at top of explorer page
- Create `docs/docs/packages/explorer.mdx` and `docs/docs/packages/graph.mdx`
- Update sidebar to include explorer and graph pages

**Acceptance Criteria:**
- [ ] Explorer search finds packages by name, summary, and tags
- [ ] Effect/stability/tag filters work correctly
- [ ] Dependency graph renders all 14+ packages with correct edges
- [ ] Nodes colored by effect, sized by dependent count
- [ ] Click on graph node navigates to `/docs/packages/sunholo/auth` etc.
- [ ] Ecosystem stats charts render with correct data
- [ ] Explorer and graph pages accessible from sidebar
- [ ] Page load under 2s for explorer, under 3s for graph

**Risks:**
- d3-force may need new npm dependency — Mitigation: try d3-hierarchy tree first (already installed)
- Graph performance with 50+ nodes — Mitigation: current ecosystem is 14 packages; optimize later if needed

---

### M5: Polish, CI Integration & Deploy
**Goal:** Final polish, CI pipeline integration, end-to-end testing, and deployment verification.
**Estimated:** ~120 LOC mixed
**Duration:** 1.5 days

**Tasks:**
- Add loading states and error boundaries to all components
- Implement localStorage SWR caching in hooks
- Add "data may be stale" banner when API unavailable
- Test mobile-responsive layout
- Test Docusaurus dark/light theme in all components
- Empty state handling (0 packages in registry)
- Verify `.github/workflows/docusaurus-deploy.yml` with sync step works end-to-end
- Test CORS from deployed ailang.sunholo.com to registry-validator
- Update CHANGELOG.md with feature description
- Run full pipeline: sync → generate → build → deploy → verify live

**Acceptance Criteria:**
- [ ] Full build pipeline works: sync → generate → build → deploy
- [ ] CORS works end-to-end from deployed site to registry-validator
- [ ] Dark/light theme looks correct on all pages
- [ ] Mobile layout is usable
- [ ] Empty state (no packages) shows helpful message
- [ ] API-down fallback works gracefully
- [ ] CHANGELOG.md updated
- [ ] All tests passing, linting clean

**Risks:**
- GitHub Actions environment may lack `curl`/`jq` for sync script — Mitigation: Ubuntu runners have both; verify in workflow

---

## Success Metrics
- All 14+ packages browsable at `/docs/packages/sunholo/*`
- Vendor index at `/docs/packages/sunholo/` lists all packages
- Interactive explorer with search, filter, sorting
- Dependency graph with effect-colored nodes
- Ecosystem stats with 4+ chart types
- Static pages work without JavaScript
- Live hydration adds version timeline + provenance
- Full CI pipeline: sync → build → deploy
- CHANGELOG.md updated

## Dependencies
- Registry validator must be deployed with new API endpoints (M1) before M2 sync script can use it
- M2 (static pages) must work before M3 (components that render on them)
- M3 (detail components) and M4 (explorer/graph) are mostly independent — can parallelize
- M5 (polish) depends on all prior milestones

## Open Questions
- Should we deploy the validator API update (M1) separately before starting M2, or can M2 sync directly from GCS bucket?
  - **Recommendation**: M2 sync script should support both — API as primary, GCS as fallback. Deploy M1 first if possible.

## Notes
- Total estimated LOC: ~2,790 (implementation + tests)
- At ~400 LOC/day velocity: ~7 working days core + 4 days integration/polish = 11 days
- This is a cross-stack sprint: Go (API), shell (sync), React/JSX (components), YAML (CI)
- Existing patterns to follow: BenchmarkDashboard (React components in Docusaurus), generate_codebase_stats.sh (build-time data sync)
