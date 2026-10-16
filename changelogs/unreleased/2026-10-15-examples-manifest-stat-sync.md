### Fixed

- Added `make sync-examples-manifest` to regenerate example statistics and imported modules together, with the existing backfill target retained as an alias.
- Manifest validation now checks aspirational counts, reports the sync command on stale statistics, and rejects newly unlisted runnable examples. The 48 existing omissions are explicitly grandfathered pending disposition (#1835).
- Manifest gate self-tests use isolated fixtures and check diagnostic reasons without rewriting the repository manifest.
