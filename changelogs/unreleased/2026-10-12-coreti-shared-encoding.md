### Fixed — large modules compile-cache again: CoreTypeInfo is DAG-encoded (#1778)

Since M-ALIAS-BODY-CLOSURE (#1706) published alias bodies fully expanded, the
plain-gob `coretypeinfo.gob` wrote every expansion again under each node that
mentions it. motoko's `src/core/session.ail` went from 29 MB to 188 MB, over the
64 MiB artifact ceiling, so it never cached and every motoko start paid a
70–130 s recompile.

- `coretypeinfo.gob` is now a hash-consed DAG (`internal/pipeline/cache_coreti_dag.go`):
  each distinct type subtree is written once and referenced by index. For
  session.ail it is 188 MB → 276 KB, and decoding takes 2.0 s → 1.5 ms.
- A decoded value equals a plain gob round trip of the same CoreTypeInfo. Tests
  pin this per variant and across a std-module corpus, with source-enumeration and
  field-count guards so a new type variant or field cannot be silently dropped.
- Guard: a two-module fixture with nested imported record aliases must write
  under 32 KiB (plain gob: 16.3 MB).
- `cacheKeyVersion` v6 → v7, so existing caches recompile once.
