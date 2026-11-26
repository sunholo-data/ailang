# MDX-invalid site content redlined dev — fix the class, not just the line

**Status**: Planned
**Target**: v0.64.2
**Priority**: P1 (incident class is recurring; the specific red line is already green)
**Estimated**: 2 days
**Dependencies**: None (docs `package-lock.json` already pins every dependency the gate needs)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime/language semantics touched; pure CI tooling |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect surface changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Turns "is this page MDX-valid?" into a local, seconds-scale, exit-code check (the whole point of the design) |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Gate failure is machine-readable `file:line:column + message`, not a 33-minute webpack log an agent must mine |
| A8: Minimal Syntax | +1 | No language syntax; one script + two make targets + workflow lines |
| A9: Cost Visibility | 0 | CI minutes are visible per step; fail-fast pre-flight *saves* ~30 red minutes per bad docs push |
| A10: Composability | +1 | Gate reuses the site's own pinned MDX pipeline rather than a parallel reimplementation |
| A11: Structured Failure | +1 | Same structured VFile error fields CI already prints, surfaced at push time with a stable target name |
| A12: System Boundary | 0 | No boundary crossings |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

### The incident (verified)

Commit `aa8c801b9` ("docs: sync 1.0 reference/guides, SECURITY.md, and Bar G5 metrics fragment (Slice 3…)", Voight-Kampff bot, 2026-11-26T12:47Z) **added** `docs/docs/reference/security-model.md` with this line 110:

```markdown
- <https://github.com/sunholo-data/ailang/security/advisories/new>
```

Docusaurus compiles **every** site page — `.md` and `.mdx` alike — with `format: 'mdx'`
(`DEFAULT_MARKDOWN_CONFIG.format` is `'mdx'` since Docusaurus 3, with `detect` deferred to v5 —
verified in `docs/node_modules/@docusaurus/core/lib/server/configValidation.js:96-112`; this site
does not override `markdown.format`). Under the MDX grammar a `<` starts a JSX tag, so `<https:` is
read as element name `https` followed by a namespace `:` — and the `/` at column 10 is illegal:
`micromark-extension-mdx-jsx: Unexpected character / (U+002F) before local name…`.

That is byte-for-byte the CI failure:

```
Error: MDX compilation failed for file ".../docs/docs/reference/security-model.md"
Cause: Unexpected character `/` (U+002F) before local name, ...
  "line": 110, "column": 10, "offset": 9544
  (run 45856293827, job docs-build 140041472561, step "Build Docusaurus site", 2026-11-26T16:33:19Z)
```

The same content compiles **cleanly** under `format: 'md'` (CommonMark autolink) — an asymmetry that
makes hand-verification and naive tooling (`ailang check`, markdown linters, regex sweeps) miss the
hazard while the production build dies. Both directions of that claim were reproduced locally this
session (see Verification Log).

### The blast radius (verified)

- **Red required checks on dev.** `Deploy Documentation to GitHub Pages` runs on push to
  `dev`. Runs 45854700033 (16:00Z) and 45856293827 (16:10Z) on `09b77efcf06c` (Release v0.64.0):
  job `docs-build 140041472561` failed at "Build Docusaurus site", `docs-gate 140047669665`
  failed by adjudication ("docs-relevant paths changed but docs-build result=failure"), deploy skipped.
- **Blocked agent PRs.** `docs-build`/`docs-gate` run per-**PR tree** (`pull_request: branches:
  [dev, main]`), so the broken file rode into every open PR's run and redlined the required
  `docs-gate` context on PRs #2474 and #2475 — both stuck on a failure their own diffs did not
  introduce (the "inherited failure" the task was created from).
- **Red release.** Tag `v0.64.0` (`09b77efcf06c`) fired the Cloud Build trigger
  `ailang-core-release` (project `ailang-multivac-deploy`), which failed. Not the docs error: the
  release pipeline never compiles the site (`docker/Dockerfile.mcp` serves raw `.md` snapshots), and
  the GitHub `CI` workflow was green for the tagged SHA (run 45854698840), so its `ci-gate` passed.
  Evidence points at `build-agent-motoko` — the only image step that runs **without** `allowFailure`
  in the release pipeline while pinned to an external fork; the v0.64.1 release commit `f43d12e3`
  repins `MOTOKO_COMMIT` `809b…→abe9…` alongside the docs fix. That diagnosis is a hypothesis from
  this session (no Cloud Build / Logging read permission from the executor SA — both API calls
  refused); the release-manager lane owns confirming it from the build logs.
- **Latency of the signal.** The error surfaced only after the full pipeline ran — WASM build, npm
  install, sync steps, webpack server+client compile: **~33 minutes** from job start to the error
  line. Every future bad docs push pays that again.

### The class is recurring (measured, not asserted)

This is the **second** recorded instance of "MDX-invalid site content redlines dev":

| Instance | Introduced by | Hazard | Detection |
|---|---|---|---|
| 2026-10-08, PR #1698 (fixed) | #1670 (`f610b897`) | bare `{repo, ref, path}` in a planned-doc H1, copied verbatim by `sync-design-docs.sh` into the generated roadmap MDX → evaluated as a JS expression | after merge, by the dev docs build — redlining every PR that touched the site until fixed |
| 2026-11-26, this incident | `aa8c801b9` (direct bot push to dev, no PR) | angle autolink `<https://…>` at `security-model.md:110` → JSX start-tag | after push, by the dev docs build — blocking PRs #2474/#2475 |

Two different hazards (`{}` expression injection vs `<` JSX parsing), two different arrival routes
(sync-generated page vs direct bot push), one shared gap: **nothing MDX-compiles site content before
it lands on dev**. The existing doc gates check orthogonal invariants — `check-doc-review` checks
`reviewBy` dates, `check-doc-samples` runs `ailang check` over fenced `.ail` blocks only. Neither
touches the prose/markup grammar the Docusaurus build actually compiles.

**Current state (2026-11-26T18:30Z, verified this session):**
- The one-line fix **landed on dev**: `f43d12e3` ("Release v0.64.1") rewrote line 110 as
  `- [https://…](https://…)`. Verified: current `docs/docs/reference/security-model.md` compiles
  cleanly under the exact CI pipeline, and **all 244 site `.md`/`.mdx` files compile cleanly**.
- Run 45866690029 ("Deploy Documentation to GitHub Pages") on `f43d12e3`: **completed success**.
- PRs #2474/#2475: head SHAs `3d4de6dcd6cf` / `b96f1d0c60df` already carry the fixed line and
  their `docs-build` + `docs-gate` check runs are **green**. The reported blocker is resolved.

What remains is the missing gate — the fix for the *class*, so the third instance never reaches
dev's Pages build again.

## Goals

**Primary Goal:** Every site page that reaches `dev` compiles under the exact production MDX
pipeline, checked locally and in CI in seconds, with a `file:line:column` error — before push,
before PR merge, and fail-fast in the Pages workflow.

**Success Metrics:**
1. `make check-mdx` compiles all `docs/docs/**/*.{md,mdx}` pages with the site's pinned MDX
   pipeline and exits non-zero with `file:line:column` on the first invalid page (measured: full
   sweep of the current 244 pages runs in < 60s locally on the executor).
2. The hazard class is pinned by a self-test fixture: a page containing the historical
   `security-model.md:110` autolink **must** fail the gate (and a `{...}` expression fixture must
   fail it too — the #1698 arm).
3. `ci.yml` `checks` job runs the gate on every push (including bot pushes to dev) and PR;
   `TestGateTargetsAreWiredIntoC1` sees it wired (`internal/cihygiene/gate_wiring_test.go`).
4. `docusaurus-deploy.yml` `docs-build` runs it as a pre-flight before the WASM/webpack work, so a
   red docs push fails in ~2 minutes, not ~33.
5. The docs-sync automation that produced `aa8c801b9` runs the gate **before** pushing to dev
   (the only route that truly prevents a red dev).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Gate = exact production pipeline (Node `@docusaurus/mdx-loader`), not a regex/lint approximation | A regex that catches `<https` misses the `{}`-expression arm (#1698) and rots as MDX evolves; the exact pipeline can never diverge from what Pages builds | human (design) | design | low |
| Scope of `check-mdx` = `docs/docs/**/*.{md,mdx}` only | That is exactly the set the docs plugin compiles (`createMDXLoaderRule` `test: /\.mdx?$/i` over the docs content dir). Repo-root `docs/*.md` (LIMITATIONS.md etc.) is raw text for agents/LLMs, never MDX-compiled — verified — so it must not fail this gate | human (design) | design | low |
| Wire into `ci.yml` `checks` job **and** `docusaurus-deploy.yml` pre-flight **and** the docs-sync bot pre-push | Each wiring covers a different arrival route: PRs (checks), pushes (checks + fail-fast), direct bot pushes (only pre-push prevents the red dev) | human (design) | design | med |
| `checks` job gains a Node.js setup + `npm ci --prefix docs` (npm cache on `docs/package-lock.json`) | The gate needs the site's node_modules (~25-40s cached, vs 45-min job budget); alternative (vendoring a tiny parser) forks the pipeline — rejected | agent | design | low |
| `check-mdx` + `test-check-mdx` join the `ci:` aggregate in `make/ci.mk` | Every wired gate is a `make ci` member (convention; `check-doc-samples test-check-doc-samples` already there) — local `make ci` requires Node, which docs development already does | agent | design | low |
| Do NOT switch the site to `markdown.format: 'detect'`/`'md'` | `'md'` would silently accept angle autolinks (the incident hazard!) and `rehype-raw` differences change rendering of every HTML-bearing page; format changes are a site-wide rendering decision out of this incident's scope | human (design, ruling recorded) | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Gate mechanism: exact `@docusaurus/mdx-loader` pipeline (decided above)
- [x] Scope: `docs/docs/**` site content only (decided above)
- [x] Wiring: `ci.yml` checks + deploy pre-flight + docs-sync bot pre-push (decided above)
- [x] `format: 'mdx'` retained — no site-wide format change (decided above)

## Solution Design

### Overview

Add one script, two make targets, three wirings, and one rule line. The script reproduces the
production compile (`createProcessors` from `@docusaurus/mdx-loader` with the plugin-content-docs
option set and `DEFAULT_MARKDOWN_CONFIG` merged over the site's `markdown: {mermaid: true}` —
including `format: 'mdx'`, `mdx1Compat: {comments, admonitions, headingIds: true}`, `emoji: true`,
`admonitions: true`, `preprocessContent`, and `getFormat`), iterates every site page, and prints the
first failure with the structured `line/column/offset` fields exactly as the Pages CI does.

### Architecture

**Components:**

1. **`docs/scripts/check-mdx.mjs`** (~120 LOC) — the checker:
   - Mirrors `@docusaurus/mdx-loader/lib/utils.js compileToJSX`: `parseFrontMatter` →
     `validateMDXFrontMatter(frontMatter.mdx)` → `preprocessContent` (admonition/heading-id MDX1
     compat escapes) → `getFormat` → `getProcessor` → `process()`.
   - Enumerates pages with the same shape the plugin does: `docs/docs/**/*.{md,mdx}`, skipping
     `_`-prefixed partials (isMDXPartial) — they are embedded, not routes, but still compiled when
     imported, so: compile them too (partials fail loudly rather than silently skip; empty glob
     fails as anti-vacuity, following `make test-stdlib-ail`'s floor pattern).
   - `--self-test` mode: fixtures under `docs/scripts/testdata/mdx/` — `bad-autolink.md` (the
     historical line 110), `bad-expression.md` (a bare `{repo, ref, path}` H1, the #1698 arm), and
     `good.md` (the fixed line 110) — asserts bad fixtures fail with the expected message class and
     the good fixture passes; exits 0/2 accordingly.
2. **`make/docs.mk` targets**:
   - `check-mdx: docs-install-guard` — compiles every site page; used by CI and `make ci`.
   - `test-check-mdx` — self-test with fixtures (same name/shape as `test-check-doc-samples`).
   - `docs-install-guard` (internal): if `docs/node_modules/@docusaurus/mdx-loader` is missing,
     fail with "run make docs-install" rather than `npm install` silently inside a gate (no silent
     fallbacks; CI installs explicitly).
3. **`make/ci.mk`**: append `check-mdx test-check-mdx` to the `ci:` prerequisite list.
4. **`.github/workflows/ci.yml`** (`checks` job):
   - "Set up Node.js" (`actions/setup-node` pinned like the other actions, `node-version-file:
     docs/.nvmrc`, `cache: npm`, `cache-dependency-path: docs/package-lock.json`).
   - "Install docs dependencies" (`npm ci --prefix docs --no-audit --no-fund`).
   - "Check site pages compile as MDX" (`make check-mdx`) and "MDX gate self-test"
     (`make test-check-mdx`).
5. **`.github/workflows/docusaurus-deploy.yml`** (`docs-build` job): new step "MDX pre-flight —
   fail fast before the WASM/webpack build" **after** "Install dependencies" (`make docs-install`)
   running `make check-mdx`. The heavy steps (WASM, sync, webpack) are unchanged; a red page now
   dies at ~2 min instead of ~33.
6. **Docs-sync bot guardrail**: the automation lane that pushed `aa8c801b9` ("Slice 3" docs sync)
   must run `make check-mdx` before its push to dev. Skill doc line (see Files) — the bot is the
   only route that bypasses PR gates entirely.
7. **`.claude/rules/docs-review.md`**: add one rule to the site-page conventions: "No bare
   angle autolinks (`<https://…>`) or un-escaped `{…}` outside code spans on site pages — MDX parses
   them as JSX/expression starts (incidents: #1698, aa8c801b9). Use `[text](url)`; verify with
   `make check-mdx`."

### Implementation Plan

**Phase 1: The checker** (~4 hours)
- [ ] Write `docs/scripts/check-mdx.mjs` (pipeline mirror + sweep + `--self-test`)
- [ ] Add fixtures `docs/scripts/testdata/mdx/{bad-autolink.md,bad-expression.md,good.md}`
- [ ] Add `check-mdx`, `test-check-mdx` to `make/docs.mk`; append both to `ci:` in `make/ci.mk`
- [ ] Local verification: full sweep green on HEAD; bad fixtures red with file:line:column

**Phase 2: Wiring** (~3 hours)
- [ ] `ci.yml` `checks`: Node setup + `npm ci --prefix docs` + gate + self-test
- [ ] `docusaurus-deploy.yml` `docs-build`: MDX pre-flight step after "Install dependencies"
- [ ] `make test-check-gates`-style sanity: `go test ./internal/cihygiene/` green (gate wiring
      test sees the new targets invoked in the workflows)

**Phase 3: Guardrails & docs** (~2 hours)
- [ ] `.claude/rules/docs-review.md` MDX-hazard rule
- [ ] docs-sync skill: pre-push `make check-mdx` step
- [ ] `make simplicity-audit` (no command-surface regression), `make fmt`, `make lint`
- [ ] Changelog entry in `changelogs/v0.32-current.md` (CI-gates theme)

### Files to Modify/Create

**New files:**
- `docs/scripts/check-mdx.mjs` — the checker + self-test entry (~120 LOC)
- `docs/scripts/testdata/mdx/bad-autolink.md` — fixture, the historical `security-model.md:110` line (~8 LOC)
- `docs/scripts/testdata/mdx/bad-expression.md` — fixture, the #1698 `{repo, ref, path}` H1 (~6 LOC)
- `docs/scripts/testdata/mdx/good.md` — fixture, the fixed line 110 (~10 LOC)

**Modified files:**
- `make/docs.mk` — `check-mdx`, `test-check-mdx`, `docs-install-guard` (~15 LOC)
- `make/ci.mk` — append two prerequisites to `ci:` (~1 LOC)
- `.github/workflows/ci.yml` — `checks` job: Node setup, npm ci, gate, self-test (~25 LOC)
- `.github/workflows/docusaurus-deploy.yml` — `docs-build` pre-flight step (~10 LOC)
- `.claude/rules/docs-review.md` — MDX-hazard rule (~5 LOC)
- `docs-sync` skill doc — pre-push gate step (~5 LOC)
- `changelogs/v0.32-current.md` — entry (~6 LOC)

## Examples

### Example 1: The incident line, before and after

**Before** (what `aa8c801b9` pushed — MDX error):
```markdown
## Reporting Vulnerabilities

Please do **not** open public GitHub issues for security vulnerabilities. Report privately via **GitHub Security Advisories**:

- <https://github.com/sunholo-data/ailang/security/advisories/new>
```

**After** (what `f43d12e3` shipped — compiles):
```markdown
## Reporting Vulnerabilities

Please do **not** open public GitHub issues for security vulnerabilities. Report privately via **GitHub Security Advisories**:

- [https://github.com/sunholo-data/ailang/security/advisories/new](https://github.com/sunholo-data/ailang/security/advisories/new)
```

### Example 2: Gate output (new)

```
$ make check-mdx
Checking 244 site pages (docs/docs/**/*.{md,mdx}) with the pinned MDX pipeline…
✗ docs/docs/reference/security-model.md
  MDX ERROR (mdx): Unexpected character `/` (U+002F) before local name, expected a
  character that can start a name, such as a letter, `$`, or `_` (note: to create a
  link in MDX, use `[text](url)`)
  place: {"line":110,"column":10,"offset":9544}
make: *** [make/docs.mk:check-mdx] Error 1
```

…at minute 2 of a red push, with the exact file/line/column — instead of minute 33 of a webpack
log, after the Pages artifact was already doomed.

## Success Criteria

- [ ] `make check-mdx` on HEAD exits 0 (all 244 pages compile — verified true today)
- [ ] `make check-mdx` on a tree reverting `security-model.md:110` to the angle autolink exits non-zero naming `file:line:column` (acceptance: seed the reversion locally, run, revert)
- [ ] `make test-check-mdx` exits 0: the `bad-autolink` and `bad-expression` fixtures fail the gate, `good` passes (anti-vacuity: the self-test must fail if the checker is neutered)
- [ ] `ci.yml` `checks` job runs both targets; `go test ./internal/cihygiene/` passes (wiring canonical)
- [ ] `docusaurus-deploy.yml` `docs-build` runs the pre-flight after "Install dependencies" and before "Build WASM binary"
- [ ] All tests passing (`make test`, `make lint`, `make fmt`, `make simplicity-audit` clean)
- [ ] Documentation updated (rules file, changelog)

## Testing Strategy

**Unit tests:**
- `check-mdx.mjs --self-test`: fixture-driven (bad-autolink, bad-expression, good); proves both
  historical hazard arms are detected and the fixed form passes.

**Integration tests:**
- `internal/cihygiene` gate-wiring test sees `check-mdx`/`test-check-mdx` invoked in a workflow.
- Manual: temporarily revert line 110 in a scratch worktree → `make check-mdx` fails at
  `110:10`; restore → passes.

**Manual testing:**
- Confirm the checker's options match production by diffing its structured error against the CI
  log of run 45856293827 (same message, line, column, offset for the same input — done this
  session for the reproduction harness; the shipped script reuses the identical code path).

## Verification Log

Every load-bearing claim above was checked this session (2026-11-26, workspace `task-709360f9`,
snapshot `f43d12e3ced8` = origin/dev HEAD):

| # | Claim | How verified | Result |
|---|---|---|---|
| 1 | CI failed on `security-model.md` MDX at 110:10 | Downloaded job 140041472561 logs via GitHub API | `Error: MDX compilation failed … Unexpected character / (U+002F) before local name … line 110, column 10, offset 9544`; step "Build Docusaurus site" exit 2 at 16:33:19Z |
| 2 | The failing content is the angle autolink | Fetched `security-model.md` at `09b77efcf06c` (= `aa8c801b9`); read line 110 | `- <https://github.com/sunholo-data/ailang/security/advisories/new>`; offset 9544 maps to that line in the CI-reported position |
| 3 | `aa8c801b9` added the file (no prior bad line) | Commit API: file list shows `docs/docs/reference/security-model.md` `added +112 -0`; parent `1341a3704888` returns 404 for that path | Confirmed |
| 4 | The commit bypassed PR review | `commits/…/pulls` returns no PR for `aa8c801b9` | Direct push to dev; PR-only gates never saw it |
| 5 | Docusaurus compiles `.md` as `format: 'mdx'` | Read `@docusaurus/core/lib/server/configValidation.js` `DEFAULT_MARKDOWN_CONFIG.format = 'mdx'`; site config sets only `markdown: {mermaid: true}`; traced `mdx-loader/lib/loader.js compileToJSX → getFormat` | Confirmed |
| 6 | The old line fails under `'mdx'`, passes under `'md'` | Reproduction harness (`@docusaurus/mdx-loader` `createProcessors`, full default options): old line with `mdxProcessor` → the exact CI error at 110:10; with `mdProcessor` → clean | Confirmed both directions |
| 7 | Current dev HEAD is clean | Same harness over all 244 `docs/docs/**/*.{md,mdx}` files | 0 failures; HEAD `security-model.md` clean |
| 8 | Dev docs workflow green post-fix | Runs API: run 45866690029 ("Deploy Documentation to GitHub Pages") on `f43d12e3` | completed / **success** |
| 9 | PRs #2474/#2475 unblocked | Runs/check-runs API for head SHAs `3d4de6dcd6cf`, `b96f1d0c60df`; their line 110 fetched | Both carry the fixed line; `docs-build` + `docs-gate` check runs green |
| 10 | Release failure ≠ docs error | Read `cloudbuild-release.yaml`: steps never run npm/docusaurus; `docker/Dockerfile.mcp` has no Node; `CI` run 45854698840 green for the tagged SHA (ci-gate passes) | Confirmed (release failure cause is elsewhere; `build-agent-motoko` is a hypothesis — `f43d12e3` repins `MOTOKO_COMMIT` `809b…→abe9…`) |
| 11 | No fast MDX gate exists today (negative existence) | `grep -rn "mdx"` over `Makefile`, `make/*.mk`, `scripts/`, `tools/` (excluding `docs/node_modules`): only `tools/docsamples` (AILANG code blocks) and workflow path filters | Confirmed — `check-doc-samples` runs `ailang check` on fenced blocks only; nothing MDX-compiles page prose |
| 12 | Prior instance #1698 same class | PR API: title/body — bare `{repo, ref, path}` in an H1 copied by `sync-design-docs.sh` into the roadmap MDX; "fails the required docs-gate on every PR" | Confirmed |
| 13 | `TestGateTargetsAreWiredIntoC1` convention applies | Read `internal/cihygiene/gate_wiring_test.go` (make targets enumerated via `make -pn`; every gate-shaped target must be invoked in a workflow or carry a documented exemption) | Confirmed — new targets must be wired and are (plan above) |

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact Node version strategy in `ci.yml` (`node-version-file: docs/.nvmrc` vs pinning like the
  deploy workflow's `22.23.4`) — agent may choose; `.nvmrc` currently says `20` while the deploy
  workflow pins `22.23.4`, so follow whichever the repo prefers at execution time.
- Whether `check-mdx` prints all failures or stops at the first (first-failure recommended to
  match CI's single-error UX; agent may choose all-failures if it stays under one screen).
- Whether the pre-flight step in `docusaurus-deploy.yml` also runs on the `schedule`/`main` paths —
  it runs wherever `docs-build` runs; no extra conditionality needed (agent to confirm).

## Non-Goals

**Not attempted in this feature:**
- Diagnosing/fixing the `ailang-core-release` Cloud Build failure — owned by the release-manager
  lane from the build logs (this session had no Cloud Build/Logging read permission); the
  v0.64.1 release path (tag → trigger → smoke gates) is already unblocked by `f43d12e3` once CI
  is green for it.
- Changing the site's `markdown.format` — rejected above (would silently accept the incident
  hazard and alters HTML rendering pipeline-wide).
- Linting repo-root markdown (`docs/LIMITATIONS.md`, `CLAUDE.md`, …) as MDX — none of it is
  compiled by the site (verified: site content is `docs/docs/**` only); raw agent-facing text stays
  out of the gate.
- A `baseline`/shrink-only ratchet file for known-bad pages — unnecessary: the sweep is at zero
  failures today and the gate is pure (any failure is a new regression).

## Timeline

**Total: ~9 hours across 1-2 working days** (single-agent; no swarm contract needed)

- Phase 1 — checker + fixtures + make targets: ~4h
- Phase 2 — workflow wiring + cihygiene green: ~3h
- Phase 3 — rules/skill docs + audits + changelog: ~2h

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Checker options drift from production pipeline | High (gate would pass what CI fails) | The script imports `@docusaurus/mdx-loader`'s own exported functions (`preprocessContent`, `getFormat`, `validateMDXFrontMatter`, `getProcessor`) rather than reimplementing them; pinned by `docs/package-lock.json`; the self-test pins the historical case |
| `npm ci --prefix docs` flakiness in `checks` job | Low | Same npm-cache pattern as `docusaurus-deploy.yml` (already proven at 24s cold / seconds cached); failures fail loudly, no silent skip |
| The `checks` job grows by ~1 min | Low | 45-minute budget; pre-flight saves ~30 min per red docs push on the Pages side |
| Docusaurus major bump changes the option shape | Med | `engines`/version pinned in `package-lock.json`; a bump PR updates the loader invocation and the self-test catches drift (the fixtures encode expected refusals) |

## Related Documents

**Incident records:**
- PR [#1698](https://github.com/sunholo-data/ailang/pull/1698) — the 2026-10 instance of the same class (bare `{repo, ref, path}` → MDX expression)
- Runs [45856293827](https://github.com/sunholo-data/ailang/actions/runs/45856293827) / [45854700033](https://github.com/sunholo-data/ailang/actions/runs/45854700033) — the red Pages runs this design's pre-flight short-circuits
- Commit `aa8c801b9` (introduced the hazard), commit `f43d12e3` (one-line fix, on dev)

**Design lineage:**
- [m-v1-docs-for-1-0](../v1_0_0/m-v1-docs-for-1-0.md) — the doc-sample gate this design sits beside (`.ail` blocks)
- [m-v1-release-gates](../v1_0_0/m-v1-release-gates.md) — the release ci-gate whose green status ruled the docs error out as the release-build cause
- [PROGRAM](../PROGRAM.md) — routing: this is an AILANG-fix-lane change (docs tooling), no core/motoko split

## References

- [Design Axioms](/docs/references/axioms)
- MDX JSX grammar: `micromark-extension-mdx-jsx` ("Unexpected character at expected" — the exact rule the autolink trips)
- Docusaurus `DEFAULT_MARKDOWN_CONFIG` (`format: 'mdx'` default, `detect` deferred to v5): `docs/node_modules/@docusaurus/core/lib/server/configValidation.js`

## Future Work

- Generalize `check-mdx` to a `docusaurus build --fast` smoke (route/sidebar validation without the
  webpack pass) if further pre-push depth is wanted — separate design, not this one.
- The `ailang-core-release` failure diagnosis and any `build-agent-motoko` hardening — release lane,
  tracked separately by the release-manager skill.

---

**Document created**: 2026-11-26
**Last updated**: 2026-11-26