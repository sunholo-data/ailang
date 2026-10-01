# docs-14 / iteration 17: one short Three Camps section

Authorized scope: attended charter ruling in design_docs/docs-mission.md:712–718, scheduled planner stage. One small milestone; no product edits performed by planner. Parent owns executor handoff. Product generation is gated on independent judge transport becoming available: controller reports sonnet spawn rejected, Ollama over ration and pi sandbox runtime dependency absent; exact evaluator preflight confirmed rc15 sandbox_unavailable. Park before product edits while this gate is unresolved. User directive for one milestone and /tmp artifacts overrides skill template's two-milestone/state-directory defaults.

## Verified baseline
- docs/docs/guides/three-camps-comparison.md: 213 lines; self-audit.md: 276; total 489, not 489 each.
- docs/docs/intro.mdx:40–43 contains existing position callout. Frontmatter slug `/` means canonical target `/docs/#three-camps`, not `/docs/intro`.
- Surviving inbound content links: 3 occurrences in 3 files: intro.mdx:42, guides/ifc-labels.mdx:32 (old Guardians fragment), references/index.md:180. Sidebar adds 2 occurrences; the deleted pages contain 4 mutual links. Design-doc links and historical result path are not public-page inbound links and remain unchanged.
- Current measured tutorialSidebar: top=6, categories=22, entries=117, depth=4. For AI Agents: 7 direct docs + 2 categories; 18 descendant doc entries. After: 5 direct docs + 2 categories; 16 descendant doc entries. Entire sidebar expected entries=115; all other structure unchanged. sidebar source 289→287 lines. Navbar unchanged (docs-22 is separate).
- No redirect plugin/config exists. staticDirectories=['static'], baseUrl='/', trailingSlash=false. Reuse static asset copying rather than add plugin/dependency/build machinery.
- Recent intro/sidebar history includes generated CLI docs, config reference and prior concise docs-mission edits; no implementation velocity inference from unrelated Go LOC. Budget 2–3 hours including build checks, single-day milestone.

## M1 — consolidate and preserve public routes
Exact edit surface (8 paths):
1. docs/docs/intro.mdx: replace the existing three-line callout with ONE section headed `## Three camps {#three-camps}`. Section length <=60 physical source lines, heading through next same-level heading; aim 15–25. Define syntactic, verification, orchestration briefly; describe AILANG contracts/effect rows and typed AI/tooling using existing local guide links. A link to original survey may retain provenance. No competitor table, scoreboard, language-count claim, numerical superiority claim, or new runnable code.
2. Delete docs/docs/guides/three-camps-comparison.md.
3. Delete docs/docs/guides/three-camps-self-audit.md.
4. docs/sidebars.js: remove exactly the two guide IDs; do not add a replacement navigation entry.
5. docs/docs/guides/ifc-labels.mdx: remove motivating-argument/deep-dive sentences at lines30–32. Its existing Guardians primary-source and design-doc links already preserve the relevant IFC motivation; no misleading link to the short camps section.
6. docs/docs/references/index.md: remove line180 bullet conflating camps with eval-only/Motoko/Claude harnesses. No replacement taxonomy claim needed.
7. Create TWO files: docs/static/docs/guides/three-camps-comparison.html and docs/static/docs/guides/three-camps-self-audit.html. Each small self-contained HTML redirect has canonical link, meta refresh and visible fallback anchor to `/docs/#three-camps`; JavaScript uses `window.location.replace('/docs/#three-camps')` with a fixed hash. Do not preserve obsolete fragment hashes. Each <=12 lines; no new config/dependency/lockfile. Public extensionless route serving must be proved against the production preview, not assumed from file existence.

## Measurement and acceptance
- Section <=60 lines; deleted narrative=489 lines. New section S replaces three-line callout: primary content delta=S-492, at least 432 lines removed. Removing IFC3 and references1 yields total content reduction >=436. With two redirects <=24 lines and sidebar2 removed, all scoped source reduction >=414 (executor reports actual git diff --numstat).
- Published docs pages 2 fewer; sidebar 117→115 and AI Agents docs18→16. Compare actual checkout baseline on executor start if generated package sidebar differs; always prove exactly two removed guide entries.
- Surviving links to either retired route: zero in authored docs/docs and sidebar after deletions. Redirect targets are intentional in static files. Historical design-doc/research/result-path references remain out of scope.
- Intro generated HTML has id=three-camps. Build creates both docs/build/docs/guides/three-camps-*.html, carrying fixed target and no deleted body/scoreboard.
- Serve built production site using existing npm run serve; request BOTH extensionless old URLs and check a real browser lands at `/docs/#three-camps`. Also test comparison URL with `#deep-dive-guardians-prompt-injection-as-a-type-system-problem` and self-audit URL with a historical/arbitrary fragment. Fixed target must replace old hash; destination heading is visible. Check redirects don't appear in sidebar, sitemap/search as full comparison articles. Test fallback anchor/meta without JS as available.

## Single-day sequence and validation
A. Executor records current baseline line/nav/link measurements, clean state and existing failures in isolated authorized checkout; do not change/discard unrelated work.
B. Capture make docs-build and make verify-examples pre-change logs and exit status (both can generate tracked outputs); maximum 30min each, poll <=60s and report progress. Baseline status: UNMEASURED. Controller confirms neither gate has run; planner did not run these because planning repo is read-only. Parent may supply prior baseline artifacts; cite exact logs instead of asserting green. make docs-build builds WASM/copies assets and npm sync-all writes generated docs; make verify-examples builds Go, writes examples_report.json/examples_status.md and validates manifest. Keep incidental/generated diffs outside scoped implementation unless required and reviewed.
C. Perform scoped edits, inspect section/link/source delta, run make docs-build and production redirect checks. Run make verify-examples after once, comparing failures against baseline. No .ail expected to change, no AILANG syntax edits, no new example files; existing examples gate is regression evidence. No evals, GPU work, inbox, harness changes, deployment or unrelated fixes.
D. Record actual counts, logs and before/after failures for evaluator. A pre-existing failing gate remains explicitly failing, never described as success; stop expansion into unrelated fixes. Implementation-caused failures must be resolved within scoped docs work. If extensionless static redirects do not serve through existing preview/hosting conventions, report the concrete failure to controller before broadening infrastructure; do not claim route preservation from build files alone.

Registry reuse: M1 action=none, package=null. Editorial consolidation and two static redirects are not an AILANG package-like capability; registry search not applicable. Existing Docusaurus static infrastructure is reused; no fresh executable framework.

Risks: extensionless hosting behavior, old fragment inheritance, incidental generated outputs and baseline build/example failures. The prescribed runtime checks settle these. No competitor/research fact refresh needed because there are no new superiority/factual survey claims.

Controller record: evaluator preflight failed before provider launch; this plan is held, not an executed sprint. The Agent tool requirement was followed for Sol designer/planner and Luna readiness. Sonnet judge rejected by tool; declared MiniMax fallback attempted through canonical runner and failed rc15.
