# CLI guide

## Documentation search

`ailang docs search "query"` searches local `design_docs/` or `docs/` directories first. When
run outside a source checkout, it can fall back to GitHub code search. GitHub fallback requires
authentication through `GITHUB_TOKEN` or `gh auth login`; unauthenticated search is not attempted.

Use `--path <dir>` to force a local corpus. Disable the network fallback with `--no-github` or
`AILANG_NO_GITHUB_SEARCH=1`. GitHub results are cached per query for one hour under
`~/.ailang/cache/docsearch/github/`; cache entries contain results and metadata only, never tokens.
