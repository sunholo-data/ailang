# M-UI-VITE8-MIGRATION: Modernize the Dashboard UI toolchain to Vite 8

**Status**: PLANNED — surfaced 2026-07-11 while triaging Dependabot majors ([#248](https://github.com/sunholo-data/ailang/pull/248) `@vitejs/plugin-react` 4→6 could not merge: it requires Vite 8, we are on Vite 5).
**Target**: v0.29 – v0.31 (opportunistic; no product dependency)
**Priority**: P3 (tooling debt — no user-facing or eval impact; unblocks a parked Dependabot PR)
**Estimated**: 0.5–1.5 days (one focused sprint; risk is in the React-compiler peer, not the Vite bump itself)
**Dependencies**: none in AILANG core. Self-contained to `ui/`.
**Author**: Claude Opus 4.8 (requested by Mark, 2026-07-11)

---

## Motivation

Dependabot keeps proposing a `@vitejs/plugin-react` 4→6 major bump ([#248](https://github.com/sunholo-data/ailang/pull/248)) that **cannot be merged as a bump** — `@vitejs/plugin-react@6` declares `peer vite@^8`, plus two new peers (`@rolldown/plugin-babel`, `babel-plugin-react-compiler`). The `ui/` app is on `vite@^5`. Even `@vitejs/plugin-react@5` needs Vite ≥6, so *no* plugin-react upgrade is possible without moving Vite first. #248 is currently parked via `@dependabot ignore this major version`, pointing at this doc.

This is pure toolchain modernization: no `.ail` language surface, no eval impact, no PROGRAM.md routing question. It exists to keep the one hand-maintained React app on a supported build toolchain (Vite 5 is two majors behind; Vite 8 is current) and to stop Dependabot re-litigating the plugin-react major every release.

## Where Vite is used (inventory)

Three JS trees reference the Vite ecosystem. Only the first is in scope.

| Path | Role | Current | In scope? |
|---|---|---|---|
| **`ui/`** | Collaboration Hub / Dashboard SPA (the hand-maintained app) | `vite@^5`, `@vitejs/plugin-react@^4.2.0`, `vitest@^3.2.6`, `react@^18.2.0`, `typescript@^6`, flat-config `eslint@^9` | **YES — migration target** |
| `internal/apiserver/templates/web_app/ui/` | Scaffold **template** emitted for generated web apps | `vite@^6`, `@vitejs/plugin-react@^4.3.0`, `react@^19` | Adjacent — align *after* `ui/` proves the path (see Out of scope) |
| `internal/executor/opencode/plugins/` | opencode plugin **tests only** (`vitest`, no dev server) | `vitest@^4.1.7` | No — already ahead; no Vite dep |

### `ui/` toolchain detail (the target)

- **Config**: `vite.config.ts` (minimal — `react()` plugin + dev server on `:3000` proxying `/api` and `/ws` to `:1957`), `vitest.config.ts`, `eslint.config.js` (flat), `tsconfig.json` + `tsconfig.node.json`.
- **Scripts**: `dev` (vite), `build` (`lint:errors && vite build`), `preview`, `test` (`vitest run`).
- **Build/CI**: `.github/workflows/dashboard-ui-build.yml` → `docker/Dockerfile.dashboard`, `ui-builder` stage on `node:22-bookworm-slim`. **Node 22 already satisfies Vite 8's floor (Node ≥20.19 / ≥22.12)** — no runner bump needed.

## Migration plan (phased, single sprint)

The Vite core bump is low-risk (our `vite.config.ts` uses only stable, unchanged API — `defineConfig`, `plugins`, `server.proxy`). The real work is the plugin-react 6 peer chain.

- **M1 — Vite 5 → 8 core.** Bump `vite` to `^8`. Bump `vitest` `3 → 4` in lockstep (Vitest 4 is the Vite-8-compatible line; Vitest 3 peers Vite 5–6). Run `npm run build` + `npm test`. Fix any `vite.config`/`vitest.config` API drift (expected: none-to-minimal given the tiny config).
- **M2 — plugin-react 4 → 6.** Bump `@vitejs/plugin-react` to `^6` and add its required peers `@rolldown/plugin-babel` and `babel-plugin-react-compiler`. Decide React-compiler posture (see Risks): either enable it or install-only to satisfy the peer. This is what closes [#248](https://github.com/sunholo-data/ailang/pull/248).
- **M3 — React 18 → 19 (conditional).** plugin-react 6 + the React compiler are designed around React 19. Evaluate bumping `react`/`react-dom`/`@types/react` 18→19. If M2 is clean on React 18, keep 18 and defer 19; if the compiler peer forces it, fold 19 in here.
- **M4 — Regenerate `ui/package-lock.json`, green the Docker build.** Confirm `.github/workflows/dashboard-ui-build.yml` passes (strict `npm ci` in the Docker `ui-builder` stage — the check that blocked #248). Smoke-test `vite dev` proxy + `vite build` output.
- **M5 — Close the loop.** Merge, then `@dependabot unignore` the plugin-react major so future minor/patch tracking resumes.

## Risks & unknowns

- **React Compiler peer (highest risk).** `babel-plugin-react-compiler@^1` is a *required* peer of plugin-react 6 but need not be *enabled*. Lowest-risk path: install to satisfy the peer, leave the compiler off in `vite.config.ts`. Enabling it is a separate opt-in (needs React 19 + a lint/codemod pass) and should not gate this migration.
- **React 18 vs 19.** If plugin-react 6 hard-requires React 19 at runtime, M3 becomes mandatory and the diff grows (types, `react-dom/client` already in use is fine; watch `@types/react` 18→19 breaking type changes).
- **Rolldown.** Vite 8 defaults toward the Rolldown bundler for some paths; `@rolldown/plugin-babel` is pulled in by plugin-react 6. Verify `vite build` output is byte-serving-equivalent (the dashboard is served static from the Docker image).
- **Vitest 4 breaking changes.** Config/API changes between Vitest 3 and 4 — audit `vitest.config.ts` and any `vi.*` mock usage in `ui/src/**/*.test.*`.

## Acceptance criteria

1. `ui/` on `vite@^8`, `@vitejs/plugin-react@^6`, `vitest@^4`; `npm ci` resolves with **no `ERESOLVE`** (strict Docker build).
2. `npm run build` and `npm test` green; `vite dev` proxy to `:1957` works.
3. `.github/workflows/dashboard-ui-build.yml` passes on the merge commit.
4. [#248](https://github.com/sunholo-data/ailang/pull/248) merged (or closed as subsumed) and the plugin-react major un-ignored.
5. No regression in the served Dashboard (manual smoke: load Hub, approval queue, message center, WS live updates).

## Out of scope

- **`internal/apiserver/templates/web_app/ui/`** (the scaffold template, already Vite 6). Worth aligning to Vite 8 for consistency in a follow-up, but it targets *generated* apps and has an independent compatibility surface — do it only after `ui/` validates the path.
- **eslint 9 → 10** ([#256](https://github.com/sunholo-data/ailang/pull/256)) — separately parked; blocked by `eslint-plugin-react@7.37.5` capping its peer at `eslint ^9.7` and eslint 10 crashing the flat config. Independent of Vite; tracked on that PR, not here.
- Enabling the React Compiler as an optimization (separate opt-in, post-migration).
