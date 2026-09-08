# Sprint M-DX27: GitHub repository search fallback

Design: `design_docs/planned/v0_29_0/m-dx27-docs-search-github-fallback.md`  
Target: v0.36.0  
Plan: 2 working days, approximately 11 hours

## Outcome and sequencing

Add an authenticated GitHub code-search backend for `ailang docs search`, selected only at the
CLI boundary after local `design_docs/` and `docs/` discovery fails. The first milestone is the
lowest-risk pure refactor: extract the existing remote parser into `internal/gitutil`, then make
the coordinator use it. `internal/docsearch/search.go` is a hard zero-diff file throughout.

The design estimates 3–4 hours for implementation. This plan budgets additional time for the
required tests, independent milestone gates, documentation, and final repository checks.

## Milestones

### Day 1 — M1: extract shared GitHub remote parsing (2 hours)

Files: `internal/gitutil/remote.go`, its test companion `internal/gitutil/remote_test.go`, and
`cmd/ailang/coordinator_cloud_github.go`.

Move the existing `getGitHubOwnerRepo` logic without changing behavior: `git remote get-url
origin`, `.git` trimming, HTTPS/SSH GitHub parsing, and the current error/non-GitHub results.
Delete the private function and update its call site. Test the pure helper through temporary git
repositories or controlled git commands. Finish with `go build ./...` and `go test ./...`.

### Day 1 — M2: backend seam and GitHub API implementation (3.5 hours)

Files: `internal/docsearch/backend.go`, `internal/docsearch/github/github.go`, and
`internal/docsearch/github/github_test.go`.

Add `SearchBackend` and `LocalBackend`; `LocalBackend` delegates to the existing `Search` and
does not edit `internal/docsearch/search.go`. Implement authenticated token resolution,
repository detection through `internal/gitutil`, request construction, response conversion,
pagination, ten-second timeouts, and loud errors for missing/invalid credentials, rate limits,
404s, malformed responses, and network failures. Use `httptest` rather than live credentials.
Finish with `go build ./...` and `go test ./...`.

### Day 2 — M3: cache (2 hours)

Files: `internal/docsearch/github/cache.go` and the backend test file.

Implement per-entry JSON caching under `~/.ailang/cache/docsearch/github/`, keyed by SHA-256 and
valid for one hour. Keep tokens out of cache data, handle corrupt/expired entries explicitly, and
test hit/miss, expiry, key isolation, atomic publication, and credential non-persistence. Finish
with `go build ./...` and `go test ./...`.

### Day 2 — M4: CLI wiring and user-facing documentation (2.5 hours)

Files: `cmd/ailang/docs_search.go`, `docs/docs/guides/cli.md`, and `CHANGELOG.md`.

Make backend choice once in the CLI: explicit `--path` and discovered local directories use
`LocalBackend`; only the no-local-docs path may construct the GitHub backend. Add the documented
disable controls (`AILANG_NO_GITHUB_SEARCH` and `--no-github`), preserve existing local behavior,
and provide actionable errors. Extend CLI tests for local preference, disabled/no-token paths,
authenticated fallback, and secret-safe errors. Document that authentication is required.
Finish with `go build ./...` and `go test ./...`.

### Day 2 — M5: integration and hard zero-diff certification (1 hour)

Verification-only target: `internal/docsearch/search.go` must remain unchanged. Run the complete
Go build/test gates plus `make fmt-check`, `make lint`, `make check-boundaries`, and
`make check-file-sizes`. Run an outside-source-tree integration test using a controlled HTTP
server/client seam; no live token is needed. Confirm the final changed-file list is limited to the
approved table and the explicit gitutil test companion. Finish with `git diff --exit-code --
internal/docsearch/search.go`, `go build ./...`, and `go test ./...`.

## Acceptance checklist

- Local `design_docs/` and `docs/` are always preferred and do not issue GitHub requests.
- GitHub fallback requires a usable `GITHUB_TOKEN` or `gh auth token`; missing and rejected tokens
  fail loudly, with no unauthenticated retry or silent degradation.
- Rate-limit, network, repository, and malformed-response errors are actionable and test-covered.
- Results are cached for one hour under the feature-specific cache directory, without credentials.
- `internal/docsearch/search.go` has zero diff lines.
- Every milestone independently passes `go build ./...` and `go test ./...`; final format, lint,
  boundary, and file-size gates also pass.

## Verification Log

All load-bearing repository claims below were checked in this worktree before planning. The
verification IDs are referenced by the machine-readable sprint JSON.

| ID | Command | Observed output |
|---|---|---|
| V1 | `nl -ba cmd/ailang/coordinator_cloud_github.go \| sed -n '85,105p'` | `getGitHubOwnerRepo` exists at lines 87–105 and parses `git remote get-url origin`, `.git`, and HTTPS/SSH GitHub prefixes. |
| V2 | `test ! -d internal/gitutil && echo 'internal/gitutil: absent'` | `internal/gitutil: absent`. |
| V3 | `rg -n 'getGitHubOwnerRepo\|GITHUB_TOKEN' cmd/ailang/coordinator_cloud_github.go` | One private-parser call site at line 46; existing `GITHUB_TOKEN` read at line 39. |
| V4 | `rg --files cmd/ailang \| rg 'coordinator_cloud_github.*_test.go$' \|\| true` | No dedicated coordinator GitHub test file; plan adds `internal/gitutil/remote_test.go`. |
| V5 | `nl -ba internal/docsearch/search.go \| sed -n '17,65p'` | `SearchOptions` line 18, `SearchResult` line 31, `SearchStats` line 38, and `Search` signature line 62 match the design. |
| V6 | `rg -n 'type Backend interface\|type Provider interface' internal/observatory/backend.go internal/ai/provider.go` | Interface precedents at `internal/observatory/backend.go:11` and `internal/ai/provider.go:442`. |
| V7 | `nl -ba cmd/ailang/docs_search.go \| sed -n '67,150p'` | CLI currently discovers local docs and directly calls `docsearch.Search` at line 138. |
| V8 | `rg -n 'Authorization: Bearer\|application/vnd.github\\+json\|X-GitHub-Api-Version\|http.NewRequestWithContext' cmd/ailang/coordinator_cloud_github.go` | Existing GitHub REST code establishes the required request/header precedent. |
| V9 | `rg -n 'cache/(registry\|git\|compile)' internal/pkg/registry.go internal/pkg/gitcache.go internal/pipeline/cache_store.go` | Existing caches use feature subdirectories `registry`, `git`, and `compile`. |
| V10 | `rg -n 'cache-info\|cleanup\|rebuild' cmd/ailang/docs_search.go` | Docs search already owns cache-management flags that wiring must preserve. |
| V11 | `sed -n '1,40p' cmd/ailang/docs_search_guard_test.go` | Existing regression test asserts SimHash search over `design_docs`. |
| V12 | `test -f docs/docs/guides/cli.md && echo present; test -f CHANGELOG.md && echo present` | Both documentation targets are present. |
| V13 | `make help \| rg 'build\|test\|fmt-check\|lint\|check-boundaries\|check-file-sizes'` | All required build/test/quality targets are exposed. |
| V14 | `git status --porcelain=v1; git diff --name-only` | Empty output; worktree clean before planning. |
| V15 | `test -x /Users/voightkampff/.agents/skills/sprint-planner/scripts/analyze_velocity.sh || echo absent` | The skill-documented velocity helper is absent; no numeric velocity was inferred. |

## Handoff

This plan is ready for user/controller approval. After approval, hand it to sprint-executor; the
planner does not stage, commit, or implement any milestone.
