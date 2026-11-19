# M-DX27: GitHub Repository Search Fallback for `ailang docs search`

**Status:** Planned (revised after quorum BLOCK — see Revision History)
**Target:** v0.36.0 (superseded from original v0.7.2; doc sat untouched since 2026-01-28 and is
only now being picked up at v0.35.0 — see Revision History)
**Priority:** P2 (Medium - nice-to-have, not blocking)
**Estimated:** 3-4 hours
**Dependencies:** None
**Created:** 2026-01-28
**Revised:** 2026-09-05

## Revision History

**2026-09-05 — quorum round 1: 3/3 reviewers BLOCK, all addressed below.**

- **gpt5-6-sol**: the "unauthenticated works at 10 req/min" success case had no verification
  log, and the doc's invalid-token handling ("fall back to unauthenticated") was a silent
  fallback. → Added a Verification Log (below) with live `curl` output. **The verification
  falsified the premise**: GitHub's code-search API now returns `401 Requires authentication`
  for unauthenticated requests — there is no unauthenticated tier to fall back to. The doc's
  Success Case, Rate Limit Case, and Configuration sections are rewritten around
  "authentication required" instead of "authentication optional." Silent-fallback language is
  removed project-wide from this doc; failures are now loud (Proposed Solution, Rate Limit
  Case, Success Criteria).
- **gemini-3-1-pro**: ~600 LOC of HTTP client / git-remote parsing / cache layer proposed to be
  wired directly into `internal/docsearch/search.go` and `cmd/ailang/docs_search.go`, violating
  PROGRAM.md's "default bias: extension, not core change." No Conflict-Surface section. →
  Redesigned as a `SearchBackend` interface (Implementation Overview, below); `Search()` in
  `internal/docsearch/search.go` is **not modified at all** — zero lines change in the existing
  hot path. Added a Conflict Surface section with grep-verified results.
- **oc-glm-5-2**: the `github://` prefix sentinel hardcoded into `Search()` was the same
  core-vs-extension violation from a different angle, plus a repeat of the silent-fallback
  objection. → Same `SearchBackend` redesign removes the sentinel entirely — backend selection
  now happens once, in the CLI layer, before `Search()` is ever called.

The feature's value proposition changes as a result of (a): this is no longer "docs search
works anywhere with no setup," it's "docs search works anywhere you have a `GITHUB_TOKEN` (or
`gh` CLI auth) configured." That's a materially smaller win than originally scoped — worth
re-confirming priority (P2) still holds before implementation. See Success Case below.

**2026-09-05 — quorum round 2: 3/3 reviewers BLOCK again, all narrow/verification-class, no
design-direction dispute — controller applied the fixes directly (docs-mission's first use of
the shared mission-control skill's narrow-refinement carve-out; sprint execution held pending
Mark's one-time OK per that carve-out's ratification requirement — see D-4 below).**

- **gpt5-6-sol**: Conflict Surface checked cache paths and env-var names but never checked
  whether equivalent HTTP-client / git-remote-parsing code already existed elsewhere in the
  repo before proposing to write it from scratch. → Grepped `cmd/ailang/`: it does —
  `getGitHubOwnerRepo()` at `coordinator_cloud_github.go:87` does exactly what Phase 2 proposed
  writing from scratch, and `createGitHubPR`/`addGitHubLabels` in the same file establish the
  Bearer-token request pattern Phase 1 needs. Added two Conflict Surface rows citing both by
  file:line. Phase 2 is revised from "write `detectGitHubRepo`/`parseGitRemote`" to "extract
  `getGitHubOwnerRepo` into a new shared `internal/gitutil` package and update both call sites"
  — reuse, not duplication.
- **gemini-3-1-pro**: independently named the same gap, specifically citing
  `coordinator_cloud_github.go`'s existing PR-creation machinery. Same fix as above; also
  confirmed the existing Bearer-token/header pattern (`Authorization: Bearer`, `Accept:
  application/vnd.github+json`, `X-GitHub-Api-Version: 2022-11-28`) is followed rather than a
  new convention invented for Phase 1's client.
- **oc-glm-5-2**: this doc's claims about `internal/docsearch/search.go`'s actual `Search()`
  signature and `SearchOptions`/`SearchResult`/`SearchStats` types were asserted, not verified —
  the Verification Log only covered the external GitHub API, not the codebase. → Grepped
  `internal/docsearch/search.go` directly: all four claims match verbatim (line-cited). Added a
  Verification note in the Conflict Surface section. The underlying claim was correct; the
  objection was a valid process gap (no log entry existed for it), not a wrong premise.

Per the shared skill's narrow-refinement carve-out (`.claude/skills/mission-control/SKILL.md`
Gate 2): all three round-2 objections carried a concrete, reviewer-named fix, and none disputed
the `SearchBackend` design direction — they asked for a reuse check the doc had skipped and a
verification-log gap. The controller applied both directly (bounded, single-command, no design
judgment) rather than spending a third ~$0.07 quorum round. This is docs-mission's FIRST use of
this carve-out (distinct from `docs-4`'s D-3 grant, which the ledger scoped to that item alone
and does not generalise here) — per the carve-out's own ratification rule, the SPRINT is held
pending Mark's one-time OK (see decision ledger D-4), not force-run on the controller's own
say-so.

## Problem Statement

`ailang docs search` fails when neither `design_docs/` nor `docs/` directories are available
locally. This makes the command unusable for users with installed binaries outside the AILANG
source tree.

**Current behavior (still reproduces at HEAD, v0.35.0-61-g087fbea63):**
```bash
$ cd /tmp
$ ailang docs search "contracts"
Error: no documentation directory found (tried design_docs/ and docs/)

Hint: Use --path flag to specify a documentation directory
Example: ailang docs search --path docs "query"
```

**Impact:**
- Medium - workaround exists (clone repo or use --path)
- Users expect documentation search to "just work" anywhere
- Reduces friction for developers exploring AILANG features
- Discovered during demo feedback review

## Verification Log

Required before this doc could make any claim about GitHub's code-search API. Run live against
the real endpoint, 2026-09-05, from this repo:

**Unauthenticated:**
```bash
$ curl -sD - "https://api.github.com/search/code?q=contracts+repo:sunholo-data/ailang" -o /dev/null
HTTP/2 401
x-ratelimit-limit: 60
x-ratelimit-remaining: 58
x-ratelimit-used: 2
x-ratelimit-resource: core
```
Body: `{"message":"Requires authentication","documentation_url":"https://docs.github.com/rest","status":"401"}`

The `60`/`core` rate-limit headers here are the *generic* unauthenticated REST budget, not a
code-search-specific one — they're returned on the 401 response but are irrelevant, because the
request is rejected before it reaches the search resource. **There is no unauthenticated code
search.** This falsifies the original doc's central claim ("Unauthenticated: 10 requests/minute").

**Authenticated** (`gh auth token`, a standard `repo`-scope PAT — no special search permission
requested or needed):
```bash
$ curl -sD - -H "Authorization: Bearer $(gh auth token)" \
    "https://api.github.com/search/code?q=contracts+repo:sunholo-data/ailang" -o /dev/null
HTTP/2 200
x-ratelimit-limit: 10
x-ratelimit-remaining: 9
x-ratelimit-used: 1
x-ratelimit-resource: code_search
```
Body: `{"total_count":452,"incomplete_results":false,"items":[...]}` — real results returned.

**Conclusions that change the design:**
1. **Authentication is required, full stop.** There is no "prefer local, fall back to
   unauthenticated GitHub" path — it would be "prefer local, fall back to GitHub *if a token is
   configured*, else tell the user why it can't help."
2. **The rate limit numbers in the original doc were backwards and wrong.** Authenticated code
   search is **10 requests/minute** (`x-ratelimit-resource: code_search`), not 30. There is no
   30 req/min tier for this endpoint at all — that number came from GitHub's general REST rate
   limit, not the code-search-specific one, and was never checked against the resource actually
   being called.
3. A `GITHUB_TOKEN` (or `gh auth token`) with plain `repo` read scope is sufficient — no special
   scope needed.

## Proposed Solution

Add GitHub code search as a third fallback when local directories aren't available **and a
usable GitHub token is present**. When neither local docs nor a token are available, fail with a
clear, actionable message — do not silently degrade to a lower-functionality mode.

**Fallback hierarchy:**
1. **Local `design_docs/`** (fastest) - for developers in source tree
2. **Local `docs/`** (fast) - for users with docs directory
3. **GitHub code search API** (network, requires a token) - for installed binaries outside the
   source tree, when `GITHUB_TOKEN` is set or `gh auth token` succeeds
4. **Loud failure** - if none of the above apply, tell the user exactly why and what to do
   (existing `--path` hint, plus a hint to set `GITHUB_TOKEN`)

### Implementation Overview — extension-shaped, not core-shaped

The reviewers' shared objection (gemini-3-1-pro, oc-glm-5-2) was that the original plan wired
GitHub-specific logic into `Search()` itself via a `strings.HasPrefix(opts.DocsPath, "github://")`
check — the one function every local search call already goes through. That makes the shared hot
path depend on HTTP-client and cache code it will never use in the (overwhelmingly common)
local-docs case, and it's exactly the kind of core-surface growth PROGRAM.md's default bias
argues against.

Fix: introduce a narrow `SearchBackend` interface. Backend *selection* happens once, in the CLI
command, before any search runs. `internal/docsearch/search.go`'s existing `Search()` function
is **not modified** — it keeps its current signature and becomes the local backend's
implementation unchanged. This mirrors two existing interface-selection patterns already in the
codebase: `internal/observatory/backend.go`'s `Backend` and `internal/ai/provider.go`'s
`Provider` — this is not a new architectural idea for this repo.

```go
// internal/docsearch/backend.go (NEW — the entire core-facing surface)
package docsearch

import "context"

// SearchBackend is implemented by anything that can answer a documentation
// search. The CLI selects which implementation to use before Search is ever
// called; docsearch itself has no branching on backend identity.
type SearchBackend interface {
    Search(ctx context.Context, opts SearchOptions) ([]SearchResult, SearchStats, error)
}

// LocalBackend wraps the existing local-filesystem Search implementation
// unchanged. This is a zero-behavior-change wrapper.
type LocalBackend struct{}

func (LocalBackend) Search(ctx context.Context, opts SearchOptions) ([]SearchResult, SearchStats, error) {
    return Search(ctx, opts) // existing function, byte-for-byte unchanged
}
```

```go
// internal/docsearch/github/github.go (NEW package — isolated, optional unit)
package github

// Backend implements docsearch.SearchBackend against the GitHub code search
// API. It is only constructed when local docs are absent, so its HTTP client,
// cache, and git-remote-parsing code are never loaded on the common path.
type Backend struct {
    Repo  string // e.g. "sunholo-data/ailang"
    Token string // required — NewBackend returns an error if empty/invalid
}

func NewBackend(ctx context.Context) (*Backend, error) {
    // resolves token (GITHUB_TOKEN env, else `gh auth token`), verifies it
    // with a lightweight call, detects repo from git remote or defaults to
    // sunholo-data/ailang. Returns a loud error if no usable token exists —
    // see "No Silent Fallback" below.
}

func (b *Backend) Search(ctx context.Context, opts docsearch.SearchOptions) ([]docsearch.SearchResult, docsearch.SearchStats, error) {
    // ... calls GitHub code search, converts results to docsearch.SearchResult
}
```

```go
// cmd/ailang/docs_search.go — the ONLY place backend selection happens.
// findDocsDir() is UNCHANGED: it still only knows about local directories,
// still returns a plain error (never a "github://" marker) when neither
// design_docs/ nor docs/ exists.
var backend docsearch.SearchBackend
if *pathFlag != "" {
    // explicit --path always means local, as today
    backend = docsearch.LocalBackend{}
} else if docsPath, err := findDocsDir(); err == nil {
    backend = docsearch.LocalBackend{}
    opts.DocsPath = docsPath
} else if os.Getenv("AILANG_NO_GITHUB_SEARCH") == "" {
    gh, ghErr := github.NewBackend(ctx)
    if ghErr != nil {
        fmt.Fprintf(os.Stderr, "%s: no local docs found, and GitHub fallback unavailable: %v\n", red("Error"), ghErr)
        fmt.Fprintln(os.Stderr, "\nHint: Use --path flag, or set GITHUB_TOKEN (see `ailang docs search --help`)")
        os.Exit(1)
    }
    backend = gh
} else {
    // AILANG_NO_GITHUB_SEARCH set — surface the original local-only error unchanged
    fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
    os.Exit(1)
}
results, stats, err := backend.Search(ctx, opts)
```

This makes the dispatch a single, narrow, reviewable seam in the CLI command — not a string
sniff buried in the shared library function every caller goes through.

## GitHub API Integration

### API Endpoint

```
GET https://api.github.com/search/code
```

**Query parameters:**
- `q` - Search query (e.g., `"contracts" repo:sunholo-data/ailang path:design_docs`)
- `per_page` - Results per page (default: 30, max: 100)
- `page` - Page number (pagination)

**Rate limits (verified live, see Verification Log):**
- Authentication is **required**. Unauthenticated requests get `401 Requires authentication`.
- Authenticated: **10 requests/minute** (`x-ratelimit-resource: code_search`) — a plain `repo`
  read-scope token is sufficient, no special permission needed.
- Header `X-RateLimit-Remaining` on successful (200) responses shows remaining requests in the
  current window; `X-RateLimit-Reset` gives the reset time as a Unix timestamp.

### Example Request/Response

**Request:**
```bash
curl -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  "https://api.github.com/search/code?q=contracts+repo:sunholo-data/ailang+path:design_docs"
```

**Response:**
```json
{
  "total_count": 42,
  "incomplete_results": false,
  "items": [
    {
      "name": "m-verify-requires-ensures.md",
      "path": "design_docs/implemented/v0_7_0/m-verify-requires-ensures.md",
      "sha": "abc123...",
      "html_url": "https://github.com/sunholo-data/ailang/blob/dev/design_docs/...",
      "score": 0.95,
      "repository": {
        "name": "ailang",
        "full_name": "sunholo-data/ailang",
        "default_branch": "dev"
      }
    }
  ]
}
```

## No Silent Fallback

Per CLAUDE.md principle 2 ("no silent fallbacks — fail loudly"), this feature must never
downgrade behavior without telling the user. Concretely:

| Condition | Behavior |
|-----------|----------|
| No `GITHUB_TOKEN` env var, `gh auth token` also fails | Loud error naming both: `"no GitHub token found (GITHUB_TOKEN unset, gh auth token failed); use --path or set GITHUB_TOKEN"`. No search attempted. |
| `GITHUB_TOKEN` set but GitHub rejects it (401 on the verification call) | Loud, specific message: `"GITHUB_TOKEN set but rejected by GitHub (401) — check the token is valid and has repo read access; not falling back further"`. **Does not** silently retry unauthenticated — the Verification Log shows there is nothing to fall back to. |
| Token valid, rate limit hit (429/403 with `x-ratelimit-remaining: 0`) | Loud message with reset time (Rate Limit Case, below). No silent retry loop. |
| `AILANG_NO_GITHUB_SEARCH=1` set | Loud message that the flag suppressed the fallback, same as today's local-only error. |

There is no scenario in which GitHub search silently degrades to a different mode — every
non-success path prints a specific, actionable reason and exits.

## User Experience

### Success Case
```bash
$ cd /tmp
$ export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
$ ailang docs search "contracts"
🔍 GitHub search: "contracts" (repo: sunholo-data/ailang, authenticated)
   Found: 12 files (API rate limit: 9/10 remaining, resets in 47s)

1. design_docs/implemented/v0_7_0/m-verify-requires-ensures.md (0.95)
   M-VERIFY: Requires/Ensures Contract Validation
   https://github.com/sunholo-data/ailang/blob/dev/design_docs/...

2. design_docs/planned/v0_7_2/m-dx26-property-test-empty-program.md (0.85)
   M-DX26: Property Test "Empty Program" Bug
   https://github.com/sunholo-data/ailang/blob/dev/design_docs/...
```

### No Token Case (replaces the original "unauthenticated success" case)
```bash
$ cd /tmp
$ ailang docs search "contracts"
Error: no local documentation directory found, and no GitHub token available
  (GITHUB_TOKEN is unset; `gh auth token` also failed)

Hint: To search documentation, try one of these:
  1. Clone the repo: git clone https://github.com/sunholo-data/ailang
  2. Use local docs: ailang docs search --path ~/ailang/design_docs "parser"
  3. Set GITHUB_TOKEN (any token with 'repo' read scope) to search via GitHub
  4. Run `gh auth login` if you use the GitHub CLI
```

### Rate Limit Case
```bash
$ ailang docs search "parser"
Error: GitHub code search rate limit exceeded (0/10 remaining, resets in 23s)

Hint: The code-search rate limit is 10 requests/minute regardless of token —
GitHub does not offer a higher tier for this endpoint. Try:
  1. Clone the repo: git clone https://github.com/sunholo-data/ailang
  2. Use local docs: ailang docs search --path ~/ailang/design_docs "parser"
  3. Wait 23 seconds for rate limit to reset
```

### Network Error Case
```bash
$ ailang docs search "types" (offline)
Error: GitHub API unavailable (network error)

Hint: Documentation search requires either:
  - Local documentation directory (design_docs/ or docs/)
  - Internet connection + GITHUB_TOKEN for GitHub fallback

To use local docs: ailang docs search --path /path/to/docs "types"
```

## Configuration

**Environment variables:**
```bash
# Required for GitHub fallback (no unauthenticated tier exists — see Verification Log)
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx
# If unset, `gh auth token` is tried as a fallback source for the same token.

# Optional - disable GitHub fallback entirely (use local only, fail loudly if absent)
export AILANG_NO_GITHUB_SEARCH=1
```

**Cache location:**

Existing cache conventions under `~/.ailang/cache/` use a subdirectory per feature
(`registry/`, `git/`, `compile/` — see Conflict Surface below), not a single flat JSON file at
the top level. Follow that convention:
```
~/.ailang/cache/docsearch/github/<sha256(query+repo+subdir+limit)>.json
```

**Cache format (per-entry file, not one shared file):**
```json
{
  "query": "contracts",
  "repo": "sunholo-data/ailang",
  "results": [...],
  "cached_at": "2026-09-05T15:30:00Z",
  "ttl_seconds": 3600
}
```

## Conflict Surface

Checked by grep against the current tree (2026-09-05) before proposing any new file or env var,
per gemini-3-1-pro's request:

| New surface | Grep result | Verdict |
|---|---|---|
| `~/.ailang/cache/github_search_cache.json` (original doc's path) | No hits anywhere in the repo | No name collision, but flat single-file cache doesn't match existing convention — changed to `~/.ailang/cache/docsearch/github/` (see Configuration) to match `~/.ailang/cache/{registry,git,compile}/` (`internal/pkg/registry.go:190`, `internal/pkg/gitcache.go:14`, `internal/pipeline/cache_store.go:24`) |
| `AILANG_NO_GITHUB_SEARCH` | No hits anywhere in the repo | Clear, no collision |
| `GITHUB_TOKEN` | **Already read** by `cmd/ailang/coordinator_cloud.go:263` and `cmd/ailang/coordinator_cloud_github.go:39`, for git-credential configuration and PR creation in the cloud coordinator | Not a naming collision — both existing uses are also read-only lookups of an ambient token, same variable, compatible semantics (a user's GitHub PAT). This feature becomes the third consumer. No write/mutation conflict. Document this in the code comment where the token is read, so a future fourth consumer checks here first. |
| `~/.ailang/config.yaml` | This feature adds no config.yaml keys — token comes from env/`gh auth token` only | No conflict; kept out of config.yaml deliberately since a token is a secret, not a config value the existing config file otherwise stores |
| Git-remote → owner/repo parsing (Phase 2's original plan) | **Already exists**: `getGitHubOwnerRepo(ctx, workDir)` at `cmd/ailang/coordinator_cloud_github.go:87` does exactly this (`git remote get-url origin`, strips `.git`, matches `https://github.com/`/`git@github.com:` prefixes, splits `owner/repo`) | Reuse, don't duplicate — but it is unexported in `package main` (`cmd/ailang`), so `internal/docsearch/github` cannot import it directly. Phase 2 below is revised to EXTRACT it into a new shared `internal/gitutil` package instead of writing a second implementation from scratch. |
| Authenticated GitHub REST call pattern (Phase 1's HTTP client) | **Already established** in the same file: `createGitHubPR`/`addGitHubLabels` both use `http.NewRequestWithContext` + `Authorization: Bearer <token>` + `Accept: application/vnd.github+json` + `X-GitHub-Api-Version: 2022-11-28` headers | Not extracted into a shared client (each caller's request/response shape differs enough — PR creation vs. label POST vs. code search GET — that a generic wrapper would just be a thin `http.NewRequestWithContext` + header-setting helper). Phase 1 follows the same header set for consistency rather than inventing a different convention, and notes the precedent inline in code comments. |

**Verification note (oc-glm-5-2's objection — the codebase claims elsewhere in this doc were
asserted, not verified):** confirmed by direct grep against `internal/docsearch/search.go`,
2026-09-05 — `type SearchOptions struct` (line 18), `type SearchResult struct` (line 31),
`type SearchStats struct` (line 38), and `func Search(ctx context.Context, opts SearchOptions)
([]SearchResult, SearchStats, error)` (line 62) all match this doc's claims about the existing
function this design wraps, verbatim.

## Implementation Plan

### Phase 1: GitHub Backend (1.5 hours)

**File: `internal/docsearch/backend.go`** (NEW, ~15 LOC)
- [ ] `SearchBackend` interface
- [ ] `LocalBackend` — thin wrapper around existing `Search()`

**File: `internal/docsearch/github/github.go`** (NEW package, ~150 LOC)
- [ ] `NewBackend(ctx)` — resolves token (`GITHUB_TOKEN` env, else `gh auth token`), verifies it
      live with a minimal request, returns a loud `error` (not a degraded backend) if no usable
      token is found
- [ ] `(*Backend).Search(ctx, opts)` — implements `docsearch.SearchBackend`
- [ ] `buildQuery`, `fetchResults` (handles the verified 10 req/min limit + 429/403 responses),
      `parseResponse`, `convertResults`

**Error handling:**
- Rate limit exceeded → loud message with retry time (verified header names: `x-ratelimit-remaining`, `x-ratelimit-reset`)
- Invalid/rejected token → loud message naming the 401, no retry (see No Silent Fallback)
- Network errors → loud message suggesting local docs or --path flag
- 404 repository → loud error message

### Phase 2: Repository Detection — EXTRACT, don't rewrite (0.25 hours, revised per
gpt5-6-sol/gemini-3-1-pro's Conflict Surface objection)

**File: `internal/gitutil/remote.go`** (NEW shared package, ~20 LOC — moved, not duplicated)
- [ ] Move `getGitHubOwnerRepo(ctx, workDir)`'s logic here as an exported
      `GitHubOwnerRepo(ctx context.Context, workDir string) (owner, repo string, err error)`,
      byte-identical behavior to the existing `cmd/ailang/coordinator_cloud_github.go:87`
      implementation
- [ ] Update `coordinator_cloud_github.go` to call `gitutil.GitHubOwnerRepo` instead of its own
      private copy — this sprint deletes the original, it does not fork it
- [ ] Default to `sunholo-data/ailang` if not in a git repo (unchanged behavior)

No new git-remote-parsing logic is written. `internal/docsearch/github` imports `internal/gitutil`,
same as `cmd/ailang` now does.

### Phase 3: Cache (0.5 hours)

**File: `internal/docsearch/github/cache.go`** (NEW, ~80 LOC)
- [ ] Per-entry file cache under `~/.ailang/cache/docsearch/github/` (see Configuration)
- [ ] TTL: 1 hour

### Phase 4: CLI Wiring (0.5 hours)

**File: `cmd/ailang/docs_search.go`** (MODIFY, ~25 LOC)
- [ ] Backend selection logic shown in Implementation Overview — `findDocsDir()` itself is
      unchanged
- [ ] Add `--no-github` flag (mirrors `AILANG_NO_GITHUB_SEARCH`)
- [ ] Update help text

**File: `internal/docsearch/search.go`**
- [ ] **No changes.** This is the point of the redesign.

### Phase 5: Testing & Documentation (1 hour)

- [ ] Test with valid `GITHUB_TOKEN` (authenticated success)
- [ ] Test with invalid `GITHUB_TOKEN` (expect loud 401 message, no fallback attempt)
- [ ] Test with no token and no `gh auth token` (expect loud "no token" message)
- [ ] Test rate limit handling (mock or trigger a 429/403 response)
- [ ] Test outside source tree with valid token (cd /tmp && ailang docs search)
- [ ] Test inside source tree (should still prefer local files, zero GitHub calls)
- [ ] Update `docs/docs/guides/cli.md` with GitHub fallback docs, stating the token requirement plainly
- [ ] Update CHANGELOG.md

## Success Criteria

- [ ] `ailang docs search` works outside source tree via GitHub API **when a valid token is available**
- [ ] Local files always preferred (no unnecessary API calls, zero GitHub code touched on that path)
- [ ] No local docs + no token → loud, specific error naming both missing conditions (not the old generic message alone)
- [ ] Invalid token → loud 401-specific message, no silent fallback to any other mode
- [ ] Rate limit errors show clear message with retry time, correctly citing the verified 10 req/min code-search limit
- [ ] Results cached for 1 hour to reduce API calls
- [ ] Network errors provide helpful fallback instructions
- [ ] `internal/docsearch/search.go`'s `Search()` function has zero diff lines
- [ ] Documentation updated with GitHub fallback behavior, stating the token requirement plainly (not "optional, for higher limits")

## Files to Modify/Create

| File | Type | Changes | LOC |
|------|------|---------|-----|
| `internal/docsearch/backend.go` | NEW | `SearchBackend` interface + `LocalBackend` | ~15 |
| `internal/docsearch/github/github.go` | NEW | GitHub API client + backend impl | ~150 |
| `internal/gitutil/remote.go` | NEW (extracted, not duplicated — see Conflict Surface) | `GitHubOwnerRepo()`, moved from `coordinator_cloud_github.go` | ~20 |
| `cmd/ailang/coordinator_cloud_github.go` | MODIFY | Delete private `getGitHubOwnerRepo`, call `gitutil.GitHubOwnerRepo` | ~-15/+2 |
| `internal/docsearch/github/cache.go` | NEW | Per-entry file cache | ~80 |
| `cmd/ailang/docs_search.go` | MODIFY | Backend selection at CLI layer, `--no-github` flag | ~25 |
| `internal/docsearch/search.go` | **UNCHANGED** | — | 0 |
| `docs/docs/guides/cli.md` | MODIFY | Document GitHub fallback, token requirement | ~20 |
| `CHANGELOG.md` | MODIFY | Document feature | ~10 |
| `internal/docsearch/github/github_test.go` | NEW | Unit tests | ~150 |
| **Total** | | | ~455 (down from ~510 — the extraction REMOVES 60 LOC of planned duplication and adds back 20+2 for the shared package and its call-site update) |

## Security Considerations

### GitHub Token Handling
- Never log or print `GITHUB_TOKEN` value
- Verify the token with a live call at backend-construction time; if rejected, fail loudly and
  do not proceed with a degraded mode (see No Silent Fallback — this replaces the original
  "don't fail if token is invalid" language, which was the exact silent-fallback pattern
  gpt5-6-sol and oc-glm-5-2 both flagged)
- Document that a token is **required** for this feature (not optional)

### API Request Safety
- Validate repo name before making API calls (no injection)
- Use HTTPS for all API requests
- Timeout HTTP requests after 10 seconds
- No credentials stored in cache files

### Cache Security
- Cache only public search results (no sensitive data)
- Use SHA256 for cache keys/filenames (no plaintext queries in filenames)
- TTL prevents stale data

## Alternatives Considered

### Alternative 1: Embed All Design Docs in Binary

Use `go:embed` to include all markdown files in the binary.

**Pros:**
- Zero network latency
- Works offline
- No rate limits, no token requirement

**Cons:**
- **Rejected by user** - "not keen on adding all design docs to the ailang CLI"
- Binary size bloat (~2-5 MB of markdown)
- Docs get stale between releases
- Can't search latest docs without upgrading binary

**Decision:** Rejected per user feedback. Worth re-raising given Alternative below now that
GitHub fallback requires a token — embedding removes the token requirement entirely for the
common case, at the cost of staleness.

### Alternative 2: Host Documentation on Website

Add search API to docs.ailang.dev.

**Pros:**
- Full control over search quality, no GitHub token requirement for end users
- Can add AI-powered semantic search
- No GitHub rate limits

**Cons:**
- Requires hosting infrastructure
- Website maintenance burden
- Another dependency for CLI
- Costs money (hosting, compute)

**Decision:** Deferred - could add later if GitHub API's now-confirmed auth requirement makes
the CLI-side fallback too high-friction for casual users.

### Alternative 3: Download Docs on Demand

Download design_docs.tar.gz from GitHub releases on first search.

**Pros:**
- One-time download, no auth required (GitHub Releases assets are public, unlike code search)
- Works offline after first use
- No rate limits after initial download

**Cons:**
- Slow first search (~5-10 seconds)
- Disk space (~5 MB)
- Cache invalidation complexity
- Doesn't get latest docs

**Decision:** Given the Verification Log finding that code search requires auth, this
alternative's "no token needed" property is now a meaningfully stronger advantage than it was
at doc creation time. Keep as a strong candidate for re-evaluation, not just a future
enhancement — but out of scope for this revision, which is about answering the quorum block on
the GitHub-code-search design as proposed.

## Future Enhancements

1. **Smart caching** - Cache individual file contents, not just search results
2. **Incremental updates** - Use GitHub API `since` parameter to fetch only changed files
3. **Private repo support** - Handle private repos with authentication (already required for public repos too, per Verification Log)
4. **Multi-repo search** - Search across related repos (e.g., ailang-examples, ailang-plugins)
5. **Content preview** - Show snippet of matching content in results
6. **Local clone detection** - If ~/.ailang/repos/ailang exists, use that instead of API

## Migration Notes

**For users:**
- No migration needed - feature is additive
- GitHub search happens automatically when local docs unavailable **and `GITHUB_TOKEN` (or `gh auth token`) is available** — this is a change from the original plan, which assumed no token was needed
- Disable with `AILANG_NO_GITHUB_SEARCH=1` if needed

**For developers:**
- No code changes needed
- Local files still preferred (no behavior change in source tree)
- Set `GITHUB_TOKEN` for testing to avoid depending on `gh auth token` state

## Related Documents

- Demo feedback: Message a7dd508c (docs search failure outside source tree)
- Current implementation: `cmd/ailang/docs_search.go`
- Existing search: `internal/docsearch/search.go`
- Quorum interface patterns referenced: `internal/observatory/backend.go`, `internal/ai/provider.go`

## Notes

- GitHub Code Search API requires authentication as of this verification (2026-09-05) — the
  "preview, unauthenticated-friendly" framing from 2026-01-28 no longer holds; verify again
  before implementation if significant time has passed, since GitHub has changed this API's
  auth requirements before
- Rate limits are per-token (confirmed: `x-ratelimit-resource: code_search`, limit 10/min)
- Search quality may differ from SimHash (GitHub uses their own algorithm)
- Cache helps reduce API calls but doesn't solve rate limit issues completely
- Consider adding `--verbose` flag to show API request details for debugging
