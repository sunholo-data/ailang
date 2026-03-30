# M-DX27: GitHub Repository Search Fallback for `ailang docs search`

**Status:** Planned
**Target:** v0.7.2
**Priority:** P2 (Medium - nice-to-have, not blocking)
**Estimated:** 3-4 hours
**Dependencies:** None
**Created:** 2026-01-28

## Problem Statement

`ailang docs search` fails when neither `design_docs/` nor `docs/` directories are available locally. This makes the command unusable for users with installed binaries outside the AILANG source tree.

**Current behavior:**
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

## Proposed Solution

Add GitHub repository search as a third fallback when local directories aren't available.

**Fallback hierarchy:**
1. **Local `design_docs/`** (fastest) - for developers in source tree
2. **Local `docs/`** (fast) - for users with docs directory
3. **GitHub code search API** (network) - for installed binaries outside source tree

### Implementation Overview

```go
// cmd/ailang/docs_search.go
func findDocsDir() (string, error) {
    // Try local directories first
    if localPath := tryLocalDocs(); localPath != "" {
        return localPath, nil
    }

    // Fall back to GitHub API search
    // This returns a special marker value that signals GitHub mode
    return "github://sunholo-data/ailang", nil
}

// internal/docsearch/search.go
func Search(opts SearchOptions) ([]SearchResult, SearchStats, error) {
    if strings.HasPrefix(opts.DocsPath, "github://") {
        // Use GitHub code search API
        return searchGitHub(opts)
    }

    // Existing local file search
    return searchLocal(opts)
}
```

## GitHub API Integration

### API Endpoint

GitHub Code Search API:
```
GET https://api.github.com/search/code
```

**Query parameters:**
- `q` - Search query (e.g., `"contracts" repo:sunholo-data/ailang path:design_docs`)
- `per_page` - Results per page (default: 30, max: 100)
- `page` - Page number (pagination)

**Rate limits:**
- Authenticated: 30 requests/minute
- Unauthenticated: 10 requests/minute
- Header: `X-RateLimit-Remaining` shows remaining requests

### Example Request/Response

**Request:**
```bash
curl -H "Accept: application/vnd.github+json" \
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

## Implementation Plan

### Phase 1: GitHub API Client (1.5 hours)

**File: `internal/docsearch/github.go`** (NEW)
- [ ] `searchGitHub(opts SearchOptions)` - Main entry point
- [ ] `buildGitHubQuery(query, repo, subdir)` - Construct API query
- [ ] `fetchGitHubResults(query, limit)` - HTTP request with pagination
- [ ] `parseGitHubResponse(body)` - Parse JSON response
- [ ] `convertToSearchResults(items)` - Convert GitHub items to SearchResult

**Error handling:**
- Rate limit exceeded → show clear message with retry time
- Network errors → suggest using local docs or --path flag
- 404 repository → fall back to error message

**Authentication:**
- Check `GITHUB_TOKEN` env var for authenticated requests
- If missing, use unauthenticated (lower rate limit)
- Don't fail if token is invalid, fall back to unauthenticated

### Phase 2: Repository Detection (0.5 hours)

**File: `internal/docsearch/repo.go`** (NEW)
- [ ] `detectGitHubRepo()` - Get repo from git remote
- [ ] `parseGitRemote(url)` - Extract org/repo from git URL
- [ ] Default to `sunholo-data/ailang` if not in git repo

**Git remote detection:**
```bash
$ git remote get-url origin
https://github.com/sunholo-data/ailang.git
# OR
git@github.com:sunholo-data/ailang.git
```

### Phase 3: Integration & Caching (1 hour)

**File: `cmd/ailang/docs_search.go`**
- [ ] Update `findDocsDir()` to return special marker for GitHub mode
- [ ] Add `--no-github` flag to disable GitHub fallback
- [ ] Update help text to document GitHub fallback

**File: `internal/docsearch/search.go`**
- [ ] Detect `github://` prefix in DocsPath
- [ ] Route to `searchGitHub()` or `searchLocal()`
- [ ] Simple cache: `~/.ailang/cache/github_search_cache.json`
  - TTL: 1 hour (GitHub content doesn't change that often)
  - Cache key: `query + repo + subdir + limit`

### Phase 4: Testing & Documentation (1 hour)

- [ ] Test with `GITHUB_TOKEN` set (authenticated)
- [ ] Test without `GITHUB_TOKEN` (unauthenticated)
- [ ] Test rate limit handling (trigger 403 response)
- [ ] Test outside source tree (cd /tmp && ailang docs search)
- [ ] Test inside source tree (should still prefer local files)
- [ ] Update `docs/docs/guides/cli.md` with GitHub fallback docs
- [ ] Update CHANGELOG.md

## User Experience

### Success Case
```bash
$ cd /tmp
$ ailang docs search "contracts"
🔍 GitHub search: "contracts" (repo: sunholo-data/ailang)
   Found: 12 files (API rate limit: 29/30 remaining)

1. design_docs/implemented/v0_7_0/m-verify-requires-ensures.md (0.95)
   M-VERIFY: Requires/Ensures Contract Validation
   https://github.com/sunholo-data/ailang/blob/dev/design_docs/...

2. design_docs/planned/v0_7_2/m-dx26-property-test-empty-program.md (0.85)
   M-DX26: Property Test "Empty Program" Bug
   https://github.com/sunholo-data/ailang/blob/dev/design_docs/...
```

### Rate Limit Case
```bash
$ ailang docs search "parser"
Error: GitHub API rate limit exceeded (resets in 23 minutes)

Hint: To search documentation, try one of these:
  1. Clone the repo: git clone https://github.com/sunholo-data/ailang
  2. Use local docs: ailang docs search --path ~/ailang/design_docs "parser"
  3. Set GITHUB_TOKEN for higher rate limit (30 req/min vs 10 req/min)
  4. Wait 23 minutes for rate limit to reset
```

### Network Error Case
```bash
$ ailang docs search "types" (offline)
Error: GitHub API unavailable (network error)

Hint: Documentation search requires either:
  - Local documentation directory (design_docs/ or docs/)
  - Internet connection for GitHub fallback

To use local docs: ailang docs search --path /path/to/docs "types"
```

## Configuration

**Environment variables:**
```bash
# Optional - increases rate limit from 10/min to 30/min
export GITHUB_TOKEN=ghp_xxxxxxxxxxxxx

# Optional - disable GitHub fallback (use local only)
export AILANG_NO_GITHUB_SEARCH=1
```

**Cache location:**
```
~/.ailang/cache/github_search_cache.json
```

**Cache format:**
```json
{
  "version": 1,
  "entries": {
    "sha256(query+repo+subdir+limit)": {
      "query": "contracts",
      "repo": "sunholo-data/ailang",
      "results": [...],
      "cached_at": "2026-01-28T15:30:00Z",
      "ttl_seconds": 3600
    }
  }
}
```

## Success Criteria

- [ ] `ailang docs search` works outside source tree using GitHub API
- [ ] Local files always preferred (no unnecessary API calls)
- [ ] Rate limit errors show clear message with retry time
- [ ] Authenticated requests use `GITHUB_TOKEN` if available
- [ ] Results cached for 1 hour to reduce API calls
- [ ] Network errors provide helpful fallback instructions
- [ ] Documentation updated with GitHub fallback behavior

## Files to Modify/Create

| File | Type | Changes | LOC |
|------|------|---------|-----|
| `internal/docsearch/github.go` | NEW | GitHub API client | ~200 |
| `internal/docsearch/repo.go` | NEW | Git remote detection | ~60 |
| `cmd/ailang/docs_search.go` | MODIFY | Add --no-github flag, update help | ~30 |
| `internal/docsearch/search.go` | MODIFY | Route to GitHub or local | ~40 |
| `internal/docsearch/cache.go` | NEW | GitHub search cache | ~100 |
| `docs/docs/guides/cli.md` | MODIFY | Document GitHub fallback | ~20 |
| `CHANGELOG.md` | MODIFY | Document feature | ~10 |
| `internal/docsearch/github_test.go` | NEW | Unit tests | ~150 |
| **Total** | | | ~610 |

## Security Considerations

### GitHub Token Handling
- Never log or print `GITHUB_TOKEN` value
- Don't fail if token is invalid, fall back to unauthenticated
- Document that token is optional (not required)

### API Request Safety
- Validate repo name before making API calls (no injection)
- Use HTTPS for all API requests
- Timeout HTTP requests after 10 seconds
- No credentials stored in cache files

### Cache Security
- Cache only public search results (no sensitive data)
- Use SHA256 for cache keys (no plaintext queries in filenames)
- TTL prevents stale data

## Alternatives Considered

### Alternative 1: Embed All Design Docs in Binary

Use `go:embed` to include all markdown files in the binary.

**Pros:**
- Zero network latency
- Works offline
- No rate limits

**Cons:**
- **Rejected by user** - "not keen on adding all design docs to the ailang CLI"
- Binary size bloat (~2-5 MB of markdown)
- Docs get stale between releases
- Can't search latest docs without upgrading binary

**Decision:** Rejected per user feedback.

### Alternative 2: Host Documentation on Website

Add search API to docs.ailang.dev.

**Pros:**
- Full control over search quality
- Can add AI-powered semantic search
- No GitHub rate limits

**Cons:**
- Requires hosting infrastructure
- Website maintenance burden
- Another dependency for CLI
- Costs money (hosting, compute)

**Decision:** Deferred - could add later if GitHub API proves insufficient.

### Alternative 3: Download Docs on Demand

Download design_docs.tar.gz from GitHub releases on first search.

**Pros:**
- One-time download
- Works offline after first use
- No rate limits after initial download

**Cons:**
- Slow first search (~5-10 seconds)
- Disk space (~5 MB)
- Cache invalidation complexity
- Doesn't get latest docs

**Decision:** Keep as possible future enhancement.

## Future Enhancements

1. **Smart caching** - Cache individual file contents, not just search results
2. **Incremental updates** - Use GitHub API `since` parameter to fetch only changed files
3. **Private repo support** - Handle private repos with authentication
4. **Multi-repo search** - Search across related repos (e.g., ailang-examples, ailang-plugins)
5. **Content preview** - Show snippet of matching content in results
6. **Local clone detection** - If ~/.ailang/repos/ailang exists, use that instead of API

## Migration Notes

**For users:**
- No migration needed - feature is additive
- GitHub search happens automatically when local docs unavailable
- Disable with `AILANG_NO_GITHUB_SEARCH=1` if needed

**For developers:**
- No code changes needed
- Local files still preferred (no behavior change in source tree)
- Set `GITHUB_TOKEN` for testing to avoid rate limits

## Related Documents

- Demo feedback: Message a7dd508c (docs search failure outside source tree)
- Current implementation: `cmd/ailang/docs_search.go`
- Existing search: `internal/docsearch/search.go`

## Notes

- GitHub Code Search API is preview feature but stable since 2022
- Rate limits are per-user (IP for unauthenticated, token for authenticated)
- Search quality may differ from SimHash (GitHub uses their own algorithm)
- Cache helps reduce API calls but doesn't solve rate limit issues completely
- Consider adding `--verbose` flag to show API request details for debugging
