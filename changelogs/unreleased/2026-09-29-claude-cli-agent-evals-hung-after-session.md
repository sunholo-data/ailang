### Fixed — every claude-CLI agent eval hung after its session finished (2026-09-29)

`claudehistory.SyncSession` opened a transaction and then queried through the pool. Since
`0fba88a82` capped every SQLite pool at one connection, that second query waited forever. The
benchmark sat idle with its claude session already gone. The audit that found it also found
`observatory.GetChainStages` querying spans and sessions inside an open result set. That hung
`ailang chains find` and the chains API whenever spans or sessions were requested. Both are
fixed, and both have tests that run on a one-connection pool and fail without the fix. The
importer's test DB now uses the production pool size.
