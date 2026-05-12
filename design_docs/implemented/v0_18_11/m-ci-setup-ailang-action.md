---
name: M-CI-SETUP-AILANG-ACTION — official setup-ailang GitHub Action + install discoverability
description: Ship a marketplace-listed `sunholo-data/setup-ailang@v1` action with caching + platform auto-detect, and close the discoverability gap on the existing install.sh one-liner so CI users stop rolling their own boilerplate.
type: design-doc
---

# M-CI-SETUP-AILANG-ACTION: official setup-ailang GitHub Action + install discoverability

**Status**: Implemented (proposal B shipped 2026-05-12 as `sunholo-data/setup-ailang@v1.0.0` — proposal A was already shipped via [docs/static/install.sh](../../../docs/static/install.sh), proposal C is N/A because the registry is GCS-backed not GitHub-backed)
**Target**: v0.19.0 (independent surface — no language/runtime change). Action lives at [sunholo-data/setup-ailang](https://github.com/sunholo-data/setup-ailang).
**Priority**: P2 — paper-cut that compounds with every new repo adopting AILANG in CI
**Estimated**: ~150 LOC action (TypeScript) + ~80 LOC install.sh hardening + docs/marketplace listing
**Dependencies**: none (uses existing release artefacts: `linux.x64.ailang.tar.gz`, `darwin.{arm64,x64}.ailang.tar.gz`, `.sha256`, `.sig`)
**Author**: Claude Opus 4.7 + Mark
**Created**: 2026-05-12
**Source**: agent message [`43553198`](../../../.ailang/messages/) from `demos/linkedin` —
"[ci] Install ergonomics — setup-ailang action + install.sh + auth fallback".
The reporter has **three copies** of the same install boilerplate in one
production workflow (`sunholo-data/ailang-demos/.github/workflows/deploy-invoice-processor.yml`)
and hit the GitHub anonymous-API rate limit (60/hr → curl exit 22) on a
real deploy this week.

---

## Problem statement

Every consumer that wants AILANG in CI today writes ~15 lines of bash that:

1. Calls `https://api.github.com/repos/sunholo-data/ailang/releases/latest`
   to resolve the latest tag.
2. Hardcodes the platform (`linux.x64`, `darwin.arm64`, …) — no auto-detect.
3. Downloads the tarball, extracts, moves to `/usr/local/bin`.
4. Often includes a `find -name ailang` fallback because the tarball
   internal layout has varied across releases.

The pattern is duplicated across the test job, the WASM-fetch job, and
the deploy job in the reference workflow above. Two hazards bite real users:

- **Anonymous GitHub API rate limit (60/hr)**: a busy CI pipeline that
  fans out across matrix jobs (or a noisy fork) will hit `403 Forbidden`
  with no body, surfacing as `curl: (22)`. Users must manually add
  `-H "Authorization: Bearer $GITHUB_TOKEN"` and learn this the hard way.
- **Platform-hardcoding drift**: Linux runners hardcode `linux.x64`,
  Apple-Silicon devs hardcode `darwin.arm64`, every new repo gets to
  re-derive the platform tag mapping.

Note that the **install.sh one-liner already exists** at
[`docs/static/install.sh`](../../../docs/static/install.sh), published as
`https://ailang.sunholo.com/install.sh`. It auto-detects platform,
verifies sha256, and supports `VERSION=…` pinning. The reporter
appears unaware of it — i.e. the gap is partly **discoverability**, not
implementation. Their 15-line bash boilerplate could already be a
1-line `curl … | sh -s -- --version v0.18.11`.

What does **not** yet exist:

- An official **`sunholo-data/setup-ailang` GitHub Action** with binary
  caching (the well-known `actions/setup-{node,go,python}` pattern).
  Caching is the real CI win — once the binary is in
  `actions/cache`, install drops from ~25s (download + extract) to ~5s
  (cache restore).

The reporter also proposes a third item (`ailang lock` should respect
`$GITHUB_TOKEN`). This is **out of scope** because the AILANG package
registry is not GitHub-backed — `RegistryClient.BaseURL` is
`https://storage.googleapis.com/ailang-registry`
([`internal/pkg/registry.go:14`](../../../internal/pkg/registry.go#L14)).
GCS public reads have no per-user quota worth worrying about; no auth
fallback is needed there. (We should respond to the reporter to clarify
this — the rate limit hits the binary install only, not `ailang lock`.)

## Goals

1. **Ship `sunholo-data/setup-ailang@v1`**, listed on the GitHub
   Marketplace, with:
   - `version: latest | <semver>` (default `latest`)
   - `cache: true | false` (default `true`)
   - Platform auto-detect across `ubuntu-*`, `macos-*` (arm64 + x64),
     `windows-*` (graceful skip with a clear error until we ship
     Windows binaries — currently gated by M-EXT-PORTABILITY-GATE).
   - SHA256 verification on every download (we already publish `.sha256`).
   - GitHub-token-aware downloads (uses the action's automatic
     `GITHUB_TOKEN` to dodge the 60/hr anonymous limit).
   - Adds `ailang` to `PATH` and exposes the resolved version as a step
     output (`steps.setup.outputs.version`) for downstream pinning.
2. **Promote `install.sh` in the docs** so the discoverability gap closes:
   - Front-page README install snippet shows the one-liner first, the
     manual tarball pattern second.
   - New `docs/docs/guides/ci-cd.md` page with a copy-paste matrix:
     "GitHub Actions → use setup-ailang", "Other CI → use install.sh",
     "Local dev → use install.sh or `brew` (future)".
3. **Harden `install.sh`** for CI use:
   - Honour `$GITHUB_TOKEN` / `$GH_TOKEN` if set (avoids the 60/hr limit
     when invoked from CI without the action).
   - Add `--prefix <dir>` (default `/usr/local/bin`, falls back to
     `~/.local/bin` if not writable) so CI without sudo Just Works.
   - Add `--quiet` for noise-free CI logs.
4. **Deprecate the bash-blob pattern in user-facing docs.** The current
   docs do not show a 15-line boilerplate, but existing user repos do.
   Provide a one-line migration snippet in the new ci-cd.md page so
   the demos repo (and any others) can swap in.

## Non-goals

- **Windows tarballs.** Out of scope — gated by M-EXT-PORTABILITY-GATE
  follow-up. The action will emit a clear "Windows not supported yet"
  error rather than silently skipping.
- **Homebrew tap / apt repo / Scoop manifest.** Defer to a later
  milestone — install.sh covers the single-binary case adequately.
- **`ailang lock` GitHub-token fallback.** N/A: registry is GCS, not
  GitHub. Reply to the reporter to clarify.
- **Composite action vs JavaScript action choice.** This doc picks
  JavaScript (Node 20) because it gives access to `@actions/cache` for
  the binary cache; a composite action would require a separate
  cache step and lose the one-step ergonomics.
- **Reusable workflows.** Different abstraction; orthogonal to this work.

## Conflict surface

Touches:

- **New repo: `sunholo-data/setup-ailang`** — separate repo so the
  marketplace listing has clean history and doesn't ship bundled
  `node_modules` to the language repo. Mark as the canonical home;
  link from this repo's README and from `docs/docs/guides/ci-cd.md`.
- **`docs/static/install.sh`** — backwards-compatible additions
  (`--prefix`, `--quiet`, optional auth header). Existing one-liner
  invocations keep working.
- **`README.md`** — install section reorders to put the one-liner first.
- **`docs/docs/guides/ci-cd.md`** (NEW) — copy-paste recipes for the
  three CI shapes (GitHub Actions / generic CI / local).
- **`.github/workflows/release.yml` (in this repo)** — no change needed;
  release artefacts are already what the action consumes.
- **`.github/workflows/setup-ailang-test.yml` (in `setup-ailang` repo,
  NEW)** — matrix test running the action on
  `{ubuntu-latest, macos-latest, macos-13}` × `{latest, v0.18.11, pinned}`
  × `{cache: true, cache: false}` to catch regressions.

Risk: **none for this repo's runtime** — the action consumes published
artefacts; nothing in `internal/` changes. Risk for the action repo is
the usual marketplace-action churn (Node version EOL, `@actions/*` SDK
upgrades) — accept it as ongoing maintenance cost (~1h/quarter).

## Design — `setup-ailang` action

### Inputs

```yaml
inputs:
  version:
    description: "AILANG version (semver) or 'latest'"
    required: false
    default: "latest"
  cache:
    description: "Cache the binary across runs"
    required: false
    default: "true"
  github-token:
    description: "Token used to dodge the 60/hr anonymous GitHub API limit. Defaults to GITHUB_TOKEN."
    required: false
    default: ${{ github.token }}
```

### Outputs

```yaml
outputs:
  version:
    description: "Resolved AILANG version (e.g. 'v0.18.11'). Useful for downstream pinning."
  cache-hit:
    description: "Whether the binary was restored from cache."
```

### Algorithm

1. Resolve `version`:
   - If `latest`, hit `https://api.github.com/repos/sunholo-data/ailang/releases/latest`
     **with** `Authorization: Bearer ${github-token}` to dodge the
     anonymous limit. Read `tag_name`.
   - Else use input verbatim (normalise leading `v`).
2. Detect platform:
   - `process.platform` + `process.arch` → `{linux,darwin}.{x64,arm64}`.
   - Windows → `core.setFailed('Windows not supported yet — track …')`.
3. Compute cache key: `ailang-${version}-${platform}-${arch}`.
4. If `cache: true`, `restoreCache([toolDir], cacheKey)`. If hit, jump to
   step 7.
5. Download:
   - `tc.downloadTool('https://github.com/sunholo-data/ailang/releases/download/${version}/${platform}.${arch}.ailang.tar.gz', undefined, 'Bearer ${github-token}')`.
   - `tc.downloadTool` for the matching `.sha256` file. Verify.
6. Extract with `tc.extractTar(...)`. If `cache: true`, `saveCache([toolDir], cacheKey)`.
7. `core.addPath(toolDir)`; `core.setOutput('version', resolvedVersion)`;
   `core.setOutput('cache-hit', String(restored))`.
8. Smoke-test: spawn `ailang --version`, fail loudly if it doesn't match.

### Usage example (what users get to copy-paste)

```yaml
- uses: sunholo-data/setup-ailang@v1
  with:
    version: v0.18.11      # or 'latest'
    cache: true
- run: ailang --version
- run: ailang lock && ailang run main.ail
```

vs. today's 15-line boilerplate the reporter has three copies of.

### Why a Node action, not composite

Composite actions can't call `@actions/cache` directly (they need a
nested `actions/cache` step with explicit key management), which loses
the one-line ergonomics. Node also gives us `@actions/tool-cache` which
does the download+extract+verify+addPath dance correctly across the
runner-image quirks (especially on macOS arm64 where tar invocation
differs).

## Design — install.sh hardening

The existing script is solid; minor additions:

```sh
# 1. Token-aware (auto-picks up CI tokens, no behavior change for humans):
AUTH_HEADER=""
if [ -n "${GITHUB_TOKEN:-}" ]; then
    AUTH_HEADER="-H \"Authorization: Bearer $GITHUB_TOKEN\""
elif [ -n "${GH_TOKEN:-}" ]; then
    AUTH_HEADER="-H \"Authorization: Bearer $GH_TOKEN\""
fi

# 2. --prefix flag for sudo-less CI installs:
PREFIX="${PREFIX:-/usr/local/bin}"
# parse --prefix=<dir> and --prefix <dir>; fall back to ~/.local/bin
# if PREFIX is not writable.

# 3. --quiet for clean CI logs (suppress info() calls).
```

All three additions are backward-compatible with the current
`curl … | bash` invocation.

## Acceptance

### Action (in `sunholo-data/setup-ailang`)

- [ ] Repo created with MIT license, README, `action.yml`, `dist/`
  built artefact committed.
- [ ] Matrix CI on the action repo: `{ubuntu-latest, macos-latest,
  macos-13}` × `{latest, v0.18.11}` × `{cache: true, cache: false}`
  passes — proves install + cache + restore.
- [ ] `cache-hit` output is `true` on the second run of the same matrix
  cell.
- [ ] Anonymous-limit regression test: a job that disables cache and
  installs 5 times in a row succeeds (proves the token is being used).
- [ ] Marketplace listing live at
  `https://github.com/marketplace/actions/setup-ailang`.
- [ ] `v1` tag + `v1.0.0` tag both point to the same SHA, per
  marketplace conventions.

### install.sh

- [ ] `GITHUB_TOKEN=$X curl … | bash` succeeds where unauthenticated
  curl is rate-limited (mock by spamming from one runner IP).
- [ ] `curl … | bash -s -- --prefix ~/bin` installs without sudo.
- [ ] `--quiet` suppresses info-level output but keeps errors.
- [ ] No regression: existing `curl … | bash` and
  `VERSION=v0.18.11 curl … | bash` still work.

### Docs

- [ ] `docs/docs/guides/ci-cd.md` (NEW) ships with three copy-paste
  recipes (GH Actions, generic CI, local dev) and a "migrating from
  hand-rolled boilerplate" snippet.
- [ ] `README.md` install section leads with the install.sh one-liner.
- [ ] CHANGELOG entry under v0.19.0 announcing both items.
- [ ] Reply to agent message `43553198` clarifying:
  - install.sh already exists (give the URL)
  - `setup-ailang@v1` shipped (give the marketplace URL)
  - `ailang lock` GH-token fallback is N/A (registry is GCS)

## Validation plan

1. **Bootstrap the action against this very repo.** Add a workflow
   `.github/workflows/dogfood-setup-ailang.yml` that uses
   `sunholo-data/setup-ailang@v1` to install and run `ailang --version`
   + a trivial example. Catches future-self regressions.
2. **Migrate the reporter's repo** (`sunholo-data/ailang-demos`) as
   the canonical migration test. Three copies of boilerplate → three
   one-line action steps. Confirm CI is green and install time
   drops on second run (cache hit).
3. **Time the difference.** Record before/after install duration on
   `ubuntu-latest` for the three reference workflows. Expect ~25s →
   ~5s on cache hit, parity (~25s) on first miss. Publish the
   numbers in the marketplace README.

## Why this matters for AI-author workflows

Every new AILANG-using repo today re-derives the install pattern,
re-discovers the rate limit, and re-learns the platform tag mapping.
For an autonomous agent generating a new project + CI scaffold, that's
~15 lines of bash it has to get right or the deploy fails on first run.
With `setup-ailang@v1` the agent emits **one well-known step** that any
human reviewer instantly recognises (mirrors `actions/setup-go`,
`setup-node`, `setup-python`). This:

1. Removes a class of CI failures the agent currently has to learn to
   debug (rate-limit 22, wrong platform tag, missing `find` fallback).
2. Compounds across thousands of CI runs as the cache hit rate grows.
3. Signals "AILANG is a serious citizen of the GH Actions ecosystem" —
   a small but real adoption signal.

## Refs

- Source message: `ailang messages read 43553198-21c8-45e2-a577-55ab752dca6a`
- Existing install.sh: [`docs/static/install.sh`](../../../docs/static/install.sh)
- Registry client (proves C is N/A): [`internal/pkg/registry.go:14`](../../../internal/pkg/registry.go#L14)
- Reference workflow with the boilerplate to consolidate:
  `sunholo-data/ailang-demos/.github/workflows/deploy-invoice-processor.yml`
- Marketplace prior art: [`actions/setup-go`](https://github.com/actions/setup-go),
  [`actions/setup-node`](https://github.com/actions/setup-node)
