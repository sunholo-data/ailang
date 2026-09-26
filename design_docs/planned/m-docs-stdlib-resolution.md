# Docs stdlib resolution for release-tarball installs

**Status**: Planned
**Target**: v0.38.6
**Priority**: P1 (Medium)
**Estimated**: 2 days
**Dependencies**: None

**Type**: Deployment / UX ruling — this is not a language-semantics change. It rules on *where
`ailang docs` finds the stdlib sources* when the binary is installed outside the repo, and on
what the release tarball must ship so that capability is present by default.

## Problem Statement

GitHub issue #1131 (daneel, measured on the v0.36.0 darwin/arm64 release tarball, third
independent report of the same gap): `ailang docs --list` fails with
`stdlib directory not found` on a tarball install. The binary only finds the stdlib if run
from the repo root, or when `AILANG_STDLIB_PATH` is set manually. There is no
`~/.ailang/std`, and the release tarball ships no `std/` at all.

**Root cause — two divergent resolvers.** `cmd/ailang/docs.go:findStdlibDir()` is a
standalone, weaker resolver with only three candidates:

1. `AILANG_STDLIB_PATH` (single path, silently ignored if `io.ail` is missing there)
2. `./std`
3. `../std`

Meanwhile the compiler's loader (`internal/loader/stdlib_resolver.go`) resolves via a
6-tier search path: CLI flag → `./std` → **binary-relative** `../std` → `AILANG_STDLIB_PATH`
(multi-path) → **user data dir** (`~/Library/Application Support/ailang/std`,
`$XDG_DATA_HOME/ailang/std`, `%APPDATA%\ailang\std`) → system dirs
(`/usr/local/share/ailang/std`, `/usr/share/ailang/std`). So `ailang run` and
`ailang docs` disagree about where the stdlib lives; on a tarball install *both* fail, but
even the loader's richer order cannot help when the tarball ships no `std/` and nothing
populates the user data dir. The capability is absent, and the docs resolver additionally
can't see locations the loader can.

**Impact:** A release binary — the artifact we hand to AI agents and external users —
cannot answer "what's in the stdlib" without a manual env var. Agents consuming the binary
read stdlib docs as *absent* (issue #1131 is the third such report). This is a deployment
packaging gap plus a resolver unification bug, not a docs-feature gap.

## Ruling 1 — Resolution order

`ailang docs` adopts one resolution order, shared conceptually with the loader, with a new
final tier:

1. `AILANG_STDLIB_PATH` (and `--stdlib-path` where applicable) — user override, stays first.
2. Repo-adjacent: `./std`, `../std` — unchanged, keeps the dev workflow zero-config.
3. Loader-compatible locations: binary-relative `../std`, user data dir, system dirs —
   docs should find the stdlib anywhere the compiler would.
4. **Embedded-in-binary stdlib** (new final tier): `std/` is `//go:embed`-ed into the
   binary at build time and docs reads from the embedded copy as a fallback.

**Why embedded-in-binary (tier 4) is the pick:**

- **Size is a non-issue.** `std/` is 46 `.ail` files, ~312 KB. The binary already embeds
  editor assets, prompts, and pi assets via `//go:embed`. We are not the first to pay this cost.
- **Version-correct by construction.** The embedded stdlib is the one from the tagged tree
  the binary was built from, matching `BinaryVersion` exactly — the same guarantee
  `checkStdlibVersion` enforces for on-disk copies. No cache-invalidation, no
  "docs showed v0.36 signatures for a v0.38 binary" drift.
- **Deterministic and offline.** AILANG is a deterministic substrate; a docs lookup should
  not depend on network state or a download cache. **Cached download is rejected**: it adds
  a network dependency, a cache directory to manage, version-pinning logic, and a failure
  mode (offline agents) — all to avoid embedding 312 KB that never changes after build.
- **Fails never, for docs specifically.** Docs is read-only over the stdlib; an embedded
  copy is always sufficient for it (see Ruling 3).

**Implementation shape (for the eventual sprint, not here):** docs keeps tiers 1–3 for
source-of-truth fidelity during development (live `std/` edits appear immediately), and
falls back to an embedded FS view for tier 4. Any filesystem-based resolver in
`docs.go` should be unified with — or delegated to — the loader's resolver rather than
remaining a private copy; the divergence is the reason this bug survived three reports.

## Ruling 2 — Should the release tarball ship `std/`?

**Yes — ship `std/` in the release tarball.** Size vs capability is not a real trade at
312 KB (the binary alone is tens of MB). Reasons:

- The tarball install then works for `ailang run`/`check` on programs importing std
  modules without any env var (the loader already checks binary-relative `../std` and
  user-data dirs; a tarball layout of `bin/ailang` + `std/` satisfies tier 3).
- Embedded stdlib (Ruling 1) covers docs, but `run`/`check` *execute* stdlib modules by
  resolving their source files; embedded sources would require a loader-side embedded
  fallback, which is out of scope for this docs ruling. Shipping `std/` fixes both with
  the packaging change only.
- Release-manager prereqs already pin `std/VERSION` == tag; the tarball contents inherit
  that guarantee for free.

The tarball layout should place `std/` such that the loader's binary-relative tier
(`<bindir>/../std`) resolves — i.e. keep the existing relative layout, don't invent a new
install convention in this ruling.

## Ruling 3 — Full sources vs signatures for docs

Docs needs **full sources**, not only signatures. Evidence in `cmd/ailang/docs*.go`:

- `showModuleDocs` prints module headers, descriptions, exported **types** (a caller needs
  the shape of `ProcessOutput` to use `exec`), and examples (`--examples`).
- `--all-functions` renders one grep-able line per export: `module.func: sig -- doc` —
  it requires the doc comments, which only exist in source.
- Signatures are parsed from the AST (`parseExportSignatures`), deliberately replacing the
  old regex that truncated effect rows at `{`.

A signatures-only artifact would silently degrade the feature. Since `std/` is 312 KB,
embedding the full sources (Ruling 1) and shipping them (Ruling 2) cost nothing over a
derived signature file — and avoid a new generated artifact to keep in sync at release time.

## Ruling 4 — Honest failure mode

The current error already fails loud (CLAUDE.md principle 2) — **keep it, don't soften it,
fix the resolution instead.** With Rulings 1–2, genuine absence means the binary is broken
or tampered, which deserves a hard error. Concretely:

- Keep the non-zero exit and `stdlib directory not found` as the terminal failure.
- Improve the *diagnostic* text to name what was tried (mirror the loader's
  `errWithSearchTrace` pattern): list the paths searched, mention that the release tarball
  should include `std/`, and point at `AILANG_STDLIB_PATH` as the override.
- No silent fallback to a degraded (signature-only, stale-cached, or empty) view. If docs
  cannot find the stdlib it reports absence loudly, so the next report says "broken install"
  instead of "capability absent".

## Relation to the docs embedder and pkg registry

- **Docs embedder (`docs embed-warmup`, `docs_search.go`)** is a *different* system: it
  embeds design docs/changelogs for `docs search` (SimHash + optional neural). This ruling
  does not touch it. The only shared surface is that both must keep working on a tarball
  install; the stdlib embed (Ruling 1) should live near the docs command's own data, not
  be conflated with the search embedding cache.
- **Pkg registry distribution** stays out of scope. The registry (`internal/pkg`,
  `ailang publish`/`install`) distributes *user packages* with dependency resolution and
  cascade bumping; the stdlib is versioned with the binary, not a registry package.
  Routing stdlib through the registry would create a versioning cycle (registry validator
  itself needs the binary) and reuse the download-cache model already rejected in Ruling 1.
  If a future "AILANG SDK install" wants registry-based stdlib delivery, that is a separate
  design doc.

## Goals

**Primary Goal:** On any release-tarball install, `ailang docs --list` and
`ailang docs <module>` work with zero configuration, and genuine stdlib absence fails
loudly with a diagnostic that names what was searched.

**Success Metrics:**
- Fresh extraction of the release tarball on darwin/arm64 and linux/amd64: `ailang docs
  --list` succeeds with no env vars, no repo present.
- A bare binary (no `std/` anywhere, embedded tier removed by test build) fails with a
  trace of searched paths, non-zero exit.
- `docs` and the loader agree on every stdlib location: no private resolver divergence.

## Non-Goals

- No loader-side embedded-stdlib fallback for `run`/`check` (tarball ships `std/` instead).
- No new install conventions, `ailang install` of the stdlib, or registry-based stdlib.
- No changes to `docs search` embeddings.
- No language or effect-semantics changes.

## Open Questions (for approval)

1. Embed `std/` via `//go:embed all:std` from the repo root (build reproducibility: the
   embed picks up the tagged tree automatically — confirm the release build already builds
   from the tagged source, which cloudbuild-release asserts).
2. Should the unified resolver be exported from `internal/loader` for docs to reuse, or
   duplicated minimally with a shared test? (Preferred: reuse — the divergence caused this.)