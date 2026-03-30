# M-PKG-EXPLORER: Package Explorer Website

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 — humans need visibility into the package ecosystem; global stats inform language evolution
**Estimated**: 2 weeks (1 week API + data, 1 week UI)
**Dependencies**:
- Package registry (implemented v0.9.7) — GCS bucket, index.json, metadata.json, history.json
- Docusaurus docs site (implemented) — React 18, Recharts, MDX, deployed to GitHub Pages at ailang.sunholo.com
- Registry validator (implemented) — Cloud Run service with /publish, /health, /version

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Read-only UI over immutable registry data — no determinism impact |
| A2: Replayability | +1 | Surfaces version history and provenance chains that make updates replayable |
| A3: Effect Legibility | +1 | Effect ceilings and per-package effect usage visible at a glance |
| A4: Explicit Authority | +1 | Provenance view shows who approved each version, what triggered it |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | API endpoints serve structured JSON — website is a thin rendering layer over machine-readable data |
| A8: Minimal Syntax | 0 | No language syntax changes |
| A9: Cost Visibility | +1 | Global stats surface ecosystem growth, tarball sizes, validation pass rates |
| A10: Composability | +1 | Composes with existing registry, messaging, and autonomous update systems |
| A11: Structured Failure | 0 | No failure handling changes |
| A12: System Boundary | +1 | Registry is an explicit system boundary; website makes that boundary observable |

**Net Score: +7** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): Read-only over immutable data — no nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — pure read operations
- [x] A4 (Authority): No ambient access — public registry data, no auth needed for reads
- [x] A7 (Machines First): JSON API first, HTML rendering second

---

## Problem Statement

The AILANG package registry holds 14+ published packages with rich metadata (manifests, version history, provenance chains, dependency graphs, effect ceilings, validation results). All this data is currently accessible only via CLI (`ailang search`, `ailang pkg provenance`, `ailang pkg docs`) or raw GCS bucket inspection.

**Current State:**
- Humans cannot browse the package ecosystem without CLI access
- No visual dependency graph — hard to understand package relationships at scale
- No aggregated ecosystem stats (effect distribution, stability breakdown, publish frequency)
- Version history and provenance chains are buried in per-version JSON files
- No way to compare versions or track how a package evolved over time
- New contributors have no discovery surface beyond `ailang search`

**Impact:**
- **Developers**: Cannot evaluate packages before adopting them — need to install and inspect locally
- **Maintainers**: No dashboard to monitor ecosystem health across all 14+ packages
- **Language designers**: No aggregate data on effect usage patterns, dependency depth, or package growth trends to inform language evolution decisions

---

## Goals

**Primary Goal:** A web-based package explorer on the AILANG docs site that makes the registry browsable, inspectable, and analyzable by humans.

**Success Metrics:**
- All published packages browsable with manifests, exports, effects, and dependencies visible
- Version history timeline with provenance chain for each version
- Global dependency graph visualization showing package relationships
- Ecosystem stats dashboard: effect distribution, publish frequency, validation pass rates
- Page load under 2s for package list, under 3s for dependency graph

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Host on Docusaurus docs site vs Collaboration Hub | Determines deployment pipeline, component patterns, and URL structure | human | design | high |
| API proxy via registry-validator Cloud Run | Determines data flow, caching, CORS, and infrastructure cost | human | design | high |
| Dependency graph library (d3-force vs dagre vs custom) | Affects interactivity, performance, and maintenance | agent | compile | med |
| URL structure for deep-linking packages/versions | Affects SEO, shareability, and Docusaurus routing | agent | compile | low |

### Design Freeze

**RESOLVED** — both high-cost decisions made:

- [x] **Host on Docusaurus docs site** (`ailang.sunholo.com`) — integrates with existing docs navigation, GitHub Pages deployment, and Docusaurus React component system. No separate auth needed (public data). Leverages existing Recharts, lucide-react, and MDX infrastructure.
- [x] **API proxy via registry-validator** (`cmd/registry-validator/`) — the Cloud Run service already has GCS access and handles CORS. Add read-only `/api/*` endpoints alongside existing `/publish`, `/health`, `/version`.

---

## Solution Design

### Overview

**Hybrid static + dynamic approach.** At build time, a sync script fetches `index.json` from the registry and generates static MDX pages mirroring the `vendor/name` module path structure — e.g., `/docs/packages/sunholo/auth`. Each vendor gets an index page (`/docs/packages/sunholo/`) listing all its packages, enabling discovery by provider. At runtime, interactive components hydrate and fetch fresh data from the registry-validator API for version details, provenance, and live stats.

This mirrors how the existing site works: `generate_codebase_stats.sh` produces `static/codebase_stats.json` at build time, and `BenchmarkDashboard` fetches `benchmarks/latest.json` for charts. Here we do the same — a build script snapshots registry data into `static/registry/`, and React components can optionally fetch live data from the validator API for freshness.

### Data Flow

```
                          BUILD TIME (GitHub Actions)
                          ─────────────────────────────
GCS Bucket
  └── index.json ──────→ docs/scripts/sync-registry.sh
                              │
                              ├── docs/static/registry/index.json (snapshot)
                              ├── docs/static/registry/sunholo/auth/metadata.json (per-package)
                              │
                              ├── docs/docs/packages/sunholo/index.mdx      (vendor index)
                              ├── docs/docs/packages/sunholo/auth.mdx       (package detail)
                              ├── docs/docs/packages/sunholo/firestore.mdx  (package detail)
                              └── docs/docs/packages/sunholo/logging.mdx    (package detail)
                                    ↓
                              Docusaurus Build → static HTML per vendor + package
                                    ↓
                              GitHub Pages (ailang.sunholo.com)
                                /docs/packages/              → all packages overview
                                /docs/packages/sunholo/      → sunholo vendor index
                                /docs/packages/sunholo/auth  → package detail (static)
                                /docs/packages/sunholo/firestore → package detail
                                /docs/packages/explorer      → interactive explorer
                                /docs/packages/graph         → dependency graph

                          RUNTIME (browser)
                          ─────────────────────────────
Registry Validator (Cloud Run) — new read-only /api/* endpoints
  GET /api/packages                         → cached index.json (live)
  GET /api/packages/:vendor/:name           → package detail + all version metadata
  GET /api/packages/:vendor/:name/:version  → single version metadata + history
  GET /api/stats                            → aggregated ecosystem stats
        │
        ▼
  React components hydrate on static pages:
    - Version timeline fetches live version data
    - Provenance chain fetches on click
    - Stats charts fetch live aggregates
    - Explorer page uses live data for search/filter
    - Falls back to static snapshot if API unavailable
```

### Architecture

**Components:**

1. **Build-Time Registry Sync** (shell script, `docs/scripts/sync-registry.sh`)
   - Fetches `index.json` from registry-validator `/api/packages` endpoint (or direct GCS read)
   - Fetches per-package metadata for each package in the index
   - Writes snapshot to `docs/static/registry/` (JSON files, gitignored)
   - **Groups packages by vendor** and generates directory structure:
     - `docs/docs/packages/{vendor}/index.mdx` — vendor index page listing all packages for that vendor
     - `docs/docs/packages/{vendor}/{name}.mdx` — package detail page
   - Uses templates: `package-page.mdx.tmpl` and `vendor-index.mdx.tmpl`
   - Generates sidebar items partial (`docs/src/data/packages-sidebar.json`) with nested vendor categories
   - All generated pages are gitignored, regenerated each build
   - Added as a build step in GitHub Actions workflow (after WASM build, before `docusaurus build`)
   - **Trigger**: runs on every docs deploy, also triggerable by `ailang publish` webhook (future)

2. **Registry API Layer** (Go, in `cmd/registry-validator/`)
   - New read-only HTTP handlers serving registry data from GCS
   - In-memory cache with 5-minute TTL (invalidated on publish via existing `handlePublish`)
   - CORS headers allowing `ailang.sunholo.com` origin
   - No auth required — public registry data

3. **Static Package Pages** (generated MDX, in `docs/docs/packages/`)
   - **`index.mdx`** — Top-level overview listing all vendors and package counts
   - **`{vendor}/index.mdx`** (generated) — Vendor index page listing all packages for that vendor with summaries, effects, stability badges. E.g., `/docs/packages/sunholo/` shows all 14 sunholo packages.
   - **`{vendor}/{name}.mdx`** (generated) — Package detail page with manifest, exports, effects, summary. E.g., `/docs/packages/sunholo/auth`
   - Each generated page mounts `<PackageDetail packageName="sunholo/auth" />` which hydrates with live data
   - **`explorer.mdx`** — Interactive explorer with live search, filter, and full dynamic data
   - **`graph.mdx`** — Dependency graph visualization
   - **URL structure mirrors module paths**: `import pkg/sunholo/auth/keys` → browse at `/docs/packages/sunholo/auth`

4. **React Components** (in `docs/src/components/PackageExplorer/`)
   - **`PackageDetail.jsx`** — Renders on static package pages; shows static snapshot immediately, fetches live data for version timeline and provenance
   - **`PackageExplorer.jsx`** — Full interactive explorer with search/filter (uses live API)
   - **`DependencyGraph.jsx`** — Force-directed graph using d3-force or d3-hierarchy (already a dependency)
   - **`EcosystemStats.jsx`** — Charts dashboard using Recharts (already a dependency)
   - **`PackageCard.jsx`** — Individual package card with effect badges, stability, tags
   - **`VersionTimeline.jsx`** — Vertical timeline of versions with provenance links
   - **`ProvenanceChain.jsx`** — Provenance detail: trigger, approval, change class, message trail

5. **Data Hooks** (in `docs/src/hooks/`)
   - `useRegistryData(path)` — generic hook that tries live API, falls back to `/registry/` static snapshot
   - `usePackageIndex()` — fetches index (live or static)
   - `usePackageDetail(vendor, name)` — fetches all version metadata for a package
   - `useEcosystemStats()` — fetches aggregated stats
   - All hooks use `useEffect` + `useState` with localStorage caching for stale-while-revalidate

### Why Hybrid (Static + Dynamic)?

| Concern | Pure Dynamic (client-side only) | Pure Static (build-time only) | **Hybrid** |
|---------|-------------------------------|------------------------------|------------|
| SEO / discoverability | No — search engines see empty shell | Yes | **Yes** — static HTML for each package |
| Works without JS | No | Yes | **Yes** — static content renders without JS |
| Fresh data | Yes — always live | No — stale until next build | **Yes** — static shell + live hydration |
| Build complexity | None | High — must fetch all data at build | **Medium** — one sync script |
| Offline / API down | Broken | Works | **Graceful** — static fallback |
| Deep-linkable | Hash-based only | Real URLs | **Real URLs** — `/docs/packages/sunholo/auth` |
| Sidebar navigation | Manual | Auto-generated | **Auto-generated** — nested by vendor |
| Discovery by vendor | N/A | N/A | **Vendor index pages** — `/docs/packages/sunholo/` |

### Generated Page Templates

**Vendor Index Template** (`vendor-index.mdx.tmpl`):

Each vendor gets an index page generated from this template:

```mdx
---
title: "{{vendor}} Packages"
description: "All AILANG packages published by {{vendor}}"
sidebar_label: "{{vendor}}"
---

import VendorIndex from '@site/src/components/PackageExplorer/VendorIndex';

# {{vendor}} Packages

{{package_count}} packages published by **{{vendor}}**.

<VendorIndex
  vendor="{{vendor}}"
  packages={{{packages_json}}}
/>
```

The `<VendorIndex>` component renders a card grid of all packages for that vendor with effect badges, stability, latest version, and links to each package detail page.

**Package Detail Template** (`package-page.mdx.tmpl`):

Each package gets a page generated from this template:

```mdx
---
title: "{{name}}"
description: "{{ai_summary}}"
sidebar_label: "{{short_name}}"
---

import PackageDetail from '@site/src/components/PackageExplorer/PackageDetail';

# {{name}}

> {{ai_summary}}

| Field | Value |
|-------|-------|
| Latest | {{latest}} |
| Stability | {{stability}} |
| Effects | {{effects}} |
| Exports | {{exports_count}} modules |
| Last Updated | {{last_updated}} |

<PackageDetail
  packageName="{{name}}"
  staticData={{{static_json}}}
/>
```

The `<PackageDetail>` component renders the static data immediately (no loading spinner), then fetches live data from the API to update the version timeline and provenance sections.

### URL Structure

URLs mirror the `vendor/name` module path convention used in AILANG imports:

```
AILANG import path:       import pkg/sunholo/auth/keys
Package browse URL:       /docs/packages/sunholo/auth
Vendor browse URL:        /docs/packages/sunholo/
All packages:             /docs/packages/

# Future: if other vendors publish packages
                          /docs/packages/acme/
                          /docs/packages/acme/widgets
```

This makes the URL structure intuitive — if you use `import pkg/sunholo/firestore/client`, you can guess the docs are at `/docs/packages/sunholo/firestore`.

### Integration with Existing Docusaurus Structure

The docs site already has a sidebar category "Packages & Registry" (see `sidebars.js`). The static pages auto-populate this:

```javascript
// sidebars.js — addition
{
  type: 'category',
  label: 'Packages & Registry',
  items: [
    'guides/packages',             // existing overview doc
    'packages/explorer',           // NEW: interactive explorer
    'packages/graph',              // NEW: dependency graph
    // Auto-generated vendor categories from packages-sidebar.json:
    {
      type: 'category',
      label: 'sunholo',           // vendor name
      link: { type: 'doc', id: 'packages/sunholo/index' },  // vendor index page
      items: [
        'packages/sunholo/auth',
        'packages/sunholo/firestore',
        'packages/sunholo/logging',
        'packages/sunholo/config',
        'packages/sunholo/http-helpers',
        'packages/sunholo/billing_store',
        // ... all sunholo/* packages
      ],
    },
    // Future vendors get their own category automatically:
    // { type: 'category', label: 'acme', items: [...] },
  ],
}
```

The sidebar items are generated by the sync script (writes `docs/src/data/packages-sidebar.json` that `sidebars.js` imports). Each vendor becomes a collapsible sidebar category with its index page as the category link.

The navbar already links to Documentation which contains the sidebar. No navbar changes needed.

### Registry Validator API Endpoints

Added to `cmd/registry-validator/main.go` alongside existing handlers:

```go
// Existing
http.HandleFunc("/publish", v.handlePublish)
http.HandleFunc("/rebuild-index", v.handleRebuildIndex)
http.HandleFunc("/health", handleHealth)
http.HandleFunc("/version", handleVersion)

// New read-only API (this feature)
http.HandleFunc("/api/packages", v.handleAPIPackages)       // GET: full index
http.HandleFunc("/api/packages/", v.handleAPIPackageDetail) // GET: /api/packages/vendor/name[/version]
http.HandleFunc("/api/stats", v.handleAPIStats)             // GET: ecosystem stats
```

**`GET /api/packages`** — Returns the full index.json from cache. Response shape matches `RegistryIndex` struct.

**`GET /api/packages/:vendor/:name`** — Aggregates from GCS:
```json
{
  "index": { /* IndexEntry */ },
  "versions": [
    {
      "version": "0.2.0",
      "metadata": { /* PackageMetadata */ },
      "history": { /* VersionHistory */ }
    }
  ],
  "dependents": ["sunholo/firestore", "sunholo/billing_store"]
}
```

**`GET /api/packages/:vendor/:name/:version`** — Returns single version metadata + history.

**`GET /api/stats`** — Computed from index.json:
```json
{
  "total_packages": 14,
  "total_versions": 28,
  "effect_distribution": { "IO": 5, "Net": 8, "FS": 3, "Env": 4 },
  "stability_breakdown": { "experimental": 10, "stable": 4 },
  "publish_timeline": [{ "date": "2026-W12", "count": 5 }],
  "dependency_depth_max": 7,
  "avg_exports_per_package": 2.3,
  "total_tarball_bytes": 57344,
  "validation_pass_rate": 1.0,
  "top_depended_on": [{ "name": "sunholo/auth", "dependent_count": 4 }],
  "pure_packages": 5,
  "agent_vs_human": { "agent": 8, "human": 6 }
}
```

### Data Model for UI

```typescript
// Mirrors Go structs in internal/pkg/registry_types.go
interface PackageListItem {
  name: string;           // "sunholo/auth"
  latest: string;         // "0.2.0"
  versions: string[];     // ["0.1.0", "0.2.0"]
  ai_summary: string;     // Human-readable description
  tags: string[];         // ["auth", "security"]
  effects: string[];      // ["IO", "Net"]
  stability: string;      // "experimental" | "stable" | "frozen"
  exports: string[];      // Module list
  dependencies: string[]; // Package names
  last_updated: string;   // ISO 8601
  updated_by: string;     // "human" | "agent"
}

interface PackageDetailResponse {
  index: PackageListItem;
  versions: VersionWithHistory[];
  dependents: string[];
}

interface VersionWithHistory {
  version: string;
  metadata: PackageMetadata;  // Full metadata.json
  history: VersionHistory;    // Full history.json
}

interface EcosystemStats {
  total_packages: number;
  total_versions: number;
  effect_distribution: Record<string, number>;
  stability_breakdown: Record<string, number>;
  publish_timeline: { date: string; count: number }[];
  dependency_depth_max: number;
  avg_exports_per_package: number;
  total_tarball_bytes: number;
  validation_pass_rate: number;
  top_depended_on: { name: string; dependent_count: number }[];
  pure_packages: number;
  agent_vs_human: { agent: number; human: number };
}
```

### UI Pages

#### 1. Package Explorer (`/docs/packages/explorer`)

```
┌─────────────────────────────────────────────────────────────┐
│  AILANG Package Registry            [search_________] [🔍]  │
├─────────────────────────────────────────────────────────────┤
│  ECOSYSTEM STATS (collapsible)                              │
│  14 packages · 28 versions · 100% pass rate                 │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │Effect    │ │Stability │ │Agent vs  │ │Top Deps  │       │
│  │Distrib.  │ │Breakdown │ │Human     │ │          │       │
│  │ [chart]  │ │ [chart]  │ │ [chart]  │ │ [chart]  │       │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘       │
├─────────────────────────────────────────────────────────────┤
│  Filters: [All Effects ▾] [All Stability ▾] [All Tags ▾]   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐    │
│  │ sunholo/auth                              v0.2.0    │    │
│  │ API key validation, HMAC signing, bearer tokens     │    │
│  │ [experimental] [Pure] [auth] [security]   2d ago    │    │
│  │                                        [▼ Expand]   │    │
│  ├─────────────────────────────────────────────────────┤    │
│  │ sunholo/firestore                         v0.1.0    │    │
│  │ Firestore REST client for AILANG                    │    │
│  │ [experimental] [Net,FS,Env] [database]    5d ago    │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  [Expand] reveals inline detail panel:                      │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ ▲ sunholo/auth                                      │    │
│  │ ┌──────────────┬──────────────────────────────┐     │    │
│  │ │ MANIFEST     │ DEPENDENCIES & DEPENDENTS    │     │    │
│  │ │ Edition: 1   │ Depends on: (none)           │     │    │
│  │ │ AILANG: >=.. │ Used by: firestore,          │     │    │
│  │ │ Effects: ()  │   billing_store, docparse    │     │    │
│  │ │              │                              │     │    │
│  │ │ EXPORTS      │ VERSION HISTORY              │     │    │
│  │ │ • auth/keys  │ ●─ v0.2.0 (Mar 20) agent    │     │    │
│  │ │ • auth/      │ │  "Fixed bearer edge case"  │     │    │
│  │ │   bearer     │ │  Class A · auto-approved   │     │    │
│  │ │              │ │  [View Provenance ▶]       │     │    │
│  │ │              │ ●─ v0.1.0 (Mar 15) agent     │     │    │
│  │ │              │    "Initial release"          │     │    │
│  │ └──────────────┴──────────────────────────────┘     │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

#### 2. Provenance Detail (expanded within version timeline)

```
┌─────────────────────────────────────────────────────────────┐
│  ◀ Back   sunholo/auth v0.2.0 — Provenance                 │
├──────────────────────┬──────────────────────────────────────┤
│  VALIDATION          │  HASHES                              │
│  ✓ Compiles          │  Content:   sha256:a1b2c3...         │
│  ✓ Effects valid     │  Interface: sha256:e5f6g7...         │
│  3/5 contracts       │  Tarball:   sha256:i9j0k1...         │
│  AILANG v0.9.12      │  Size: 4,096 bytes                   │
├──────────────────────┴──────────────────────────────────────┤
│  PROVENANCE CHAIN                                           │
│  Trigger: msg_abc123 (upgrade-available from core)          │
│  Change class: A (patch, interface unchanged)               │
│  Auto-approved: yes                                         │
│  Agent trace: trace_xyz789                                  │
│  Previous version: 0.1.0                                    │
├─────────────────────────────────────────────────────────────┤
│  MESSAGE TRAIL                                              │
│  10:00 [upgrade-available] from core                        │
│        "AILANG v0.9.12 released, check compatibility"       │
│  10:05 [task-started] from package-agent                    │
│        "Running compatibility check..."                     │
│  10:12 [publish-success] from validator                     │
│        "Published sunholo/auth@0.2.0"                       │
└─────────────────────────────────────────────────────────────┘
```

#### 3. Dependency Graph (`/docs/packages/graph`)

Interactive force-directed graph using d3-hierarchy (already in docs `package.json`) or d3-force:

```
┌─────────────────────────────────────────────────────────────┐
│  Package Dependency Network         [Filter by effect ▾]    │
│                                                             │
│         ┌─────────┐                                         │
│         │ logging │◄──────┐                                 │
│         └─────────┘       │                                 │
│              ▲         ┌──┴──────┐                          │
│              │         │firestore│                           │
│         ┌────┴────┐    └─────────┘                          │
│         │  auth   │       ▲                                 │
│         └─────────┘       │                                 │
│              ▲      ┌─────┴──────────┐                      │
│              └──────│billing_store   │                       │
│                     └────────────────┘                       │
│                            ▲                                │
│                     ┌──────┴─────────┐                      │
│                     │billing_service │                       │
│                     └────────────────┘                       │
│                                                             │
│  Legend: ○ Pure  ◐ IO only  ● Net+FS+Env                   │
│  Hover for details · Click to expand in explorer            │
└─────────────────────────────────────────────────────────────┘
```

Nodes colored by effect ceiling (pure=green, IO-only=blue, Net/FS=orange). Node size proportional to dependent count. Edges show dependency direction. Click navigates to explorer page with package expanded.

### Implementation Plan

**Phase 1: Registry Validator API Endpoints** (~2 days)

- [ ] Create `cmd/registry-validator/handlers_api.go` with read-only handlers:
  - `GET /api/packages` — serve cached index.json
  - `GET /api/packages/:vendor/:name` — aggregate all version metadata from GCS
  - `GET /api/packages/:vendor/:name/:version` — serve metadata.json + history.json
  - `GET /api/stats` — compute and cache ecosystem stats
- [ ] Create `cmd/registry-validator/cache.go` — in-memory cache with 5-min TTL, invalidated by `handlePublish`
- [ ] Add CORS middleware allowing `ailang.sunholo.com` origin
- [ ] Add tests for new handlers
- [ ] Deploy updated validator via existing Cloud Build trigger

**Phase 2: Build-Time Registry Sync & Static Pages** (~3 days)

- [ ] Create `docs/scripts/sync-registry.sh`:
  - Fetch index.json from registry-validator `/api/packages` (or direct GCS `gsutil cp`)
  - Fetch per-package metadata via `/api/packages/:vendor/:name`
  - Write JSON snapshots to `docs/static/registry/`
  - Group packages by vendor, create `docs/docs/packages/{vendor}/` directories
  - Generate `{vendor}/index.mdx` from `vendor-index.mdx.tmpl` for each vendor
  - Generate `{vendor}/{name}.mdx` from `package-page.mdx.tmpl` for each package
  - Generate `docs/src/data/packages-sidebar.json` with nested vendor categories
- [ ] Create `docs/src/templates/package-page.mdx.tmpl` — template for package detail pages
- [ ] Create `docs/src/templates/vendor-index.mdx.tmpl` — template for vendor index pages
- [ ] Add `sync-registry.sh` step to `.github/workflows/docusaurus-deploy.yml` (between "Generate codebase statistics" and "Build Docusaurus site")
- [ ] Add generated dirs to `.gitignore`: `docs/static/registry/`, `docs/docs/packages/*/`, `docs/src/data/packages-sidebar.json`
- [ ] Update `docs/sidebars.js` to import generated sidebar items with vendor nesting
- [ ] Create hand-written `docs/docs/packages/index.mdx` — top-level overview listing all vendors
- [ ] Implement `VendorIndex.jsx` component — card grid of packages for a vendor
- [ ] Verify `docusaurus build` succeeds with generated vendor/package pages

**Phase 3: PackageDetail Component (hydrates static pages)** (~2 days)

- [ ] Create `docs/src/components/PackageExplorer/PackageDetail.jsx` — renders static data immediately, fetches live data for version timeline + provenance
- [ ] Implement `VersionTimeline.jsx` — vertical timeline with expand/collapse per version
- [ ] Implement `ProvenanceChain.jsx` — validation results, hashes, message trail
- [ ] Create data hooks in `docs/src/hooks/useRegistryData.js` — tries live API, falls back to static snapshot
- [ ] Style with CSS matching existing Docusaurus theme (`docs/src/css/custom.css`)

**Phase 4: Interactive Explorer & Dependency Graph** (~3 days)

- [ ] Implement `PackageExplorer.jsx` — full interactive explorer with search, filter, sorting (live API)
- [ ] Implement `PackageCard.jsx` — summary card with effect badges, stability, tags
- [ ] Implement `DependencyGraph.jsx` using d3-hierarchy (already available) or d3-force
- [ ] Implement `EcosystemStats.jsx` using Recharts (already available)
- [ ] Create `docs/docs/packages/explorer.mdx` and `docs/docs/packages/graph.mdx`
- [ ] Node coloring by effect, sizing by dependent count
- [ ] Click-to-navigate from graph nodes to static package pages

**Phase 5: Polish & Deploy** (~1 day)

- [ ] Loading states and error boundaries
- [ ] localStorage caching for stale-while-revalidate UX
- [ ] Graceful fallback when API unavailable (static snapshot only, with "data may be stale" banner)
- [ ] Mobile-responsive layout
- [ ] Docusaurus dark/light theme compatibility
- [ ] Empty state handling (0 packages)
- [ ] Verify CORS works end-to-end with deployed validator
- [ ] Test full build pipeline: sync → generate → build → deploy

### Files to Modify/Create

**New files (registry-validator):**
- `cmd/registry-validator/handlers_api.go` — Read-only API handlers (~250 LOC)
- `cmd/registry-validator/handlers_api_test.go` — Handler tests (~150 LOC)
- `cmd/registry-validator/cache.go` — In-memory cache for registry data (~100 LOC)

**New files (build pipeline):**
- `docs/scripts/sync-registry.sh` — Fetch registry data + generate MDX pages per vendor/package (~200 LOC)
- `docs/src/templates/package-page.mdx.tmpl` — Template for generated package pages (~40 LOC)
- `docs/src/templates/vendor-index.mdx.tmpl` — Template for vendor index pages (~30 LOC)
- `docs/src/data/packages-sidebar.json` — Generated sidebar items with vendor categories (auto-generated, gitignored)

**New files (docs site — static pages):**
- `docs/docs/packages/index.mdx` — Hand-written overview page listing all vendors (~50 LOC)
- `docs/docs/packages/{vendor}/index.mdx` (generated, gitignored) — one per vendor
- `docs/docs/packages/{vendor}/{name}.mdx` (generated, gitignored) — one per package
- `docs/docs/packages/explorer.mdx` — Interactive explorer page (~30 LOC)
- `docs/docs/packages/graph.mdx` — Graph page (~30 LOC)

**New files (React components):**
- `docs/src/components/PackageExplorer/VendorIndex.jsx` — Vendor index card grid (~150 LOC)
- `docs/src/components/PackageExplorer/PackageDetail.jsx` — Static+live detail view (~300 LOC)
- `docs/src/components/PackageExplorer/PackageExplorer.jsx` — Full interactive explorer (~300 LOC)
- `docs/src/components/PackageExplorer/PackageCard.jsx` — Package card (~150 LOC)
- `docs/src/components/PackageExplorer/VersionTimeline.jsx` — Timeline component (~200 LOC)
- `docs/src/components/PackageExplorer/ProvenanceChain.jsx` — Provenance detail (~150 LOC)
- `docs/src/components/PackageExplorer/EcosystemStats.jsx` — Stats charts (~200 LOC)
- `docs/src/components/PackageExplorer/DependencyGraph.jsx` — Graph visualization (~300 LOC)
- `docs/src/components/PackageExplorer/styles.module.css` — Component styles (~200 LOC)
- `docs/src/hooks/useRegistryData.js` — Data fetching with static fallback (~120 LOC)

**Modified files:**
- `cmd/registry-validator/main.go` — Register new API routes (~5 LOC)
- `docs/sidebars.js` — Import generated package sidebar items (~10 LOC)
- `docs/docusaurus.config.js` — Add registry-validator URL to site config (~3 LOC)
- `.github/workflows/docusaurus-deploy.yml` — Add sync-registry step (~10 LOC)
- `.gitignore` — Add `docs/static/registry/`, `docs/docs/packages/*/` (generated vendor dirs), `docs/src/data/packages-sidebar.json`

---

## Examples

### Example 1: Browsing Packages

**Before (CLI only):**
```bash
$ ailang search auth
sunholo/auth v0.2.0 — API key validation, HMAC signing, bearer token extraction
  Tags: auth, security, api-key  Effects: (pure)  Stability: experimental

$ ailang pkg provenance sunholo/auth@0.2.0
Published: 2026-03-20T12:00:00Z by sunholo-voight-kampff
Change class: A (auto-approved)
Trigger: msg_abc123
Previous: 0.1.0
```

**After (web + CLI):**
- Navigate to `https://ailang.sunholo.com/docs/packages/sunholo/` → see all sunholo packages
- Click `auth` → `/docs/packages/sunholo/auth` (static page, SEO-indexed)
- See manifest, exports, effect ceiling, dependents — all rendered as static HTML
- Version timeline hydrates with live data from registry-validator API
- Click "View Provenance" on v0.2.0 → see chain, message trail, hashes (fetched on demand)
- Or use `/docs/packages/explorer` for interactive search across all packages
- CLI commands still work identically

### Example 2: Ecosystem Health Check

**Before:** Manual inspection of each package's metadata.json in GCS bucket.

**After:** Visit the explorer page, see stats banner:
- 14 packages, 28 versions, 100% validation pass rate
- Bar chart: 5 pure packages, 3 with Net effect, 2 with FS
- 60% agent-updated, 40% human-updated
- Deepest dependency chain: 7 levels

### Example 3: Understanding Package Relationships

**Before:** `ailang tree` in each package, mentally assembling the graph.

**After:** Navigate to `/docs/packages/graph`:
- See all 14 packages as an interactive network
- Green nodes = pure, orange nodes = Net+FS effects
- Large nodes = many dependents (auth, logging are hubs)
- Click `billing_store` → see its dependencies highlighted
- Hover any node → tooltip with summary, version, effects

---

## Success Criteria

- [ ] All published packages visible in web UI with correct manifest data
- [ ] Version timeline shows all versions with publish date and summary
- [ ] Provenance chain fully rendered for versions with provenance data
- [ ] Dependency graph renders all 14+ packages with correct edges
- [ ] Effect ceiling visible per-package and color-coded in graph
- [ ] Ecosystem stats dashboard shows all chart types (effect distribution, stability, agent/human, top deps)
- [ ] Search/filter works across name, tags, effects, stability
- [ ] Each package has its own static page at `/docs/packages/sunholo/auth` etc. (mirrors module path)
- [ ] Each vendor has an index page at `/docs/packages/sunholo/` listing all its packages
- [ ] Static pages render without JavaScript (manifest, exports, summary visible)
- [ ] Deep-links work: sharing `/docs/packages/sunholo/auth` loads the correct package page
- [ ] Sidebar shows nested vendor categories with package links
- [ ] Page load under 2s for package list, under 3s for dependency graph
- [ ] All tests passing
- [ ] Docusaurus builds and deploys cleanly via existing GitHub Actions workflow
- [ ] Documentation updated (sidebar, any cross-references)

---

## Testing Strategy

**Unit tests:**
- Registry API handlers: mock GCS responses, verify JSON shape and caching
- Stats computation: verify aggregation logic against known index data
- React components: basic render tests with mock data (Docusaurus doesn't have a test runner by default — consider inline vitest or manual testing)

**Integration tests:**
- API endpoints against real GCS bucket (staging)
- Full page load with real registry data in local `docusaurus start`

**Manual testing:**
- Navigate all pages on local Docusaurus dev server
- Verify dependency graph interactivity (hover, click, zoom)
- Test with 0 packages (empty state), 1 package, 14+ packages
- Verify mobile layout and Docusaurus dark/light theme compatibility
- Test CORS from localhost and deployed ailang.sunholo.com

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Graph layout algorithm (d3-force vs dagre vs d3-hierarchy tree) — agent may choose based on performance with 14-50 nodes
- Chart color palette — agent may choose, should harmonize with Docusaurus Sunholo theme (see `docs/src/css/custom.css`)
- Whether stats are inline on explorer page or separate page — agent may choose
- Pagination strategy for package list (none needed at 14 packages, but design for 100+) — agent may choose
- Cache invalidation details in registry-validator (TTL vs publish-hook) — agent may choose
- Whether to add d3-force as a new dependency or use existing d3-hierarchy for graph — agent may choose
- JSX vs TSX for components (existing docs components use `.jsx`) — agent should match existing convention (`.jsx`)

---

## Non-Goals

**Not attempted in this feature:**
- **Write operations** — No publishing, editing, or deleting from the website. CLI-only for writes.
- **User accounts / auth** — Package data is public. No login required to browse.
- **Private registry browsing** — Private registries are Phase 3; this is public registry only.
- **Source code browsing** — Not embedding a code viewer for .ail files. Use `ailang pkg docs` or GitHub for source.
- **Real-time updates** — No WebSocket push for new publishes. SWR with localStorage cache is sufficient at current scale.
- **Package comparison** — Side-by-side version diff is out of scope. Version timeline is sufficient.
- **Build-triggered redeploy on publish** — Publish does not auto-trigger a docs rebuild (future work). Static pages update on the next docs deploy. Live API data fills the gap between builds.

---

## Timeline

**Week 1** (~5 days):
- Phase 1: Registry validator API endpoints + caching + CORS + deploy
- Phase 2: Docusaurus pages, package list, search, filters

**Week 2** (~5 days):
- Phase 3: Detail panel, version timeline, provenance chain
- Phase 4: Dependency graph + ecosystem stats
- Phase 5: Polish, deploy, verify

**Total: ~10 days across 2 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| GCS read latency for package detail (aggregating multiple metadata.json) | Med | Server-side caching with 5-min TTL; detail endpoint pre-aggregates on first request |
| CORS between GitHub Pages and Cloud Run | Low | Existing serve-api handles CORS (commit 8fb51252); reuse pattern. Registry-validator already serves to external clients |
| Build-time sync fails (registry unavailable during GitHub Actions build) | Med | `sync-registry.sh` exits gracefully if API unreachable — build proceeds with stale/empty data. Static pages show "data unavailable" placeholder. Not a blocking failure. |
| Static pages stale between builds (new package published, site not redeployed) | Low | Live API hydration fills the gap. Explorer page always uses live data. Static pages show "last synced" timestamp. Future: webhook triggers docs rebuild on publish. |
| Docusaurus client-side React hydration with large interactive components | Low | BenchmarkDashboard already proves this pattern works. Lazy-load graph component |
| Registry validator uptime affects live features | Med | Static pages work without API. Interactive features fall back to static snapshot with "data may be stale" banner |
| d3-force bundle size bloating Docusaurus build | Low | Dynamic import (`React.lazy`) for graph page; d3-hierarchy already bundled |
| Dark/light theme compatibility in Docusaurus | Low | Use CSS custom properties from existing theme; test both modes |

---

## Related Documents

**Implemented (informs design):**
- [m-pkg-package-system.md](../../implemented/v0_9_5/m-pkg-package-system.md) — Phase 1 package system (manifests, lock files)
- [m-pkg-registry.md](../../implemented/v0_9_7/m-pkg-registry.md) — Registry architecture (GCS, validator, CLI)
- [m-pkg-msg-package-messaging-graph.md](../../implemented/v0_9_9/m-pkg-msg-package-messaging-graph.md) — Package messaging system
- [m-dx-app-package-adoption.md](../../implemented/v0_9_11/m-dx-app-package-adoption.md) — Application package adoption

**Planned (check for overlap):**
- [m-pkg-autonomous-updates.md](m-pkg-autonomous-updates.md) — Autonomous package update pipeline (provenance data source)
- [m-pkg-ecosystem-status.md](m-pkg-ecosystem-status.md) — Post-publication audit (known limitations)
- [m-pkg-transitive-lock-fix.md](m-pkg-transitive-lock-fix.md) — Lock file dependency resolution fixes

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Registry types](../../internal/pkg/registry_types.go) — Go structs for index, metadata, history, provenance
- [Registry validator](../../cmd/registry-validator/main.go) — Existing Cloud Run service
- [Docusaurus config](../../docs/docusaurus.config.js) — Site configuration
- [Existing components](../../docs/src/components/) — BenchmarkDashboard, AilangRepl patterns
- [Custom CSS](../../docs/src/css/custom.css) — Sunholo theme variables

---

## Future Work

- **Publish-triggered docs rebuild**: Registry validator calls GitHub Actions `workflow_dispatch` on successful publish, so static pages update within minutes
- **Source code viewer**: Inline `.ail` file browsing with syntax highlighting
- **Package comparison**: Side-by-side diff between versions
- **Real-time publish feed**: WebSocket notifications for new publishes
- **Private registry support**: Auth-gated browsing for private packages
- **API documentation generation**: Auto-generate API docs from exported module interfaces
- **Download stats**: Track install counts per version (requires registry-side logging)
- **Badge generation**: SVG badges for stability, effect ceiling, version (embed in READMEs)

---

**Document created**: 2026-03-24
**Last updated**: 2026-03-24
