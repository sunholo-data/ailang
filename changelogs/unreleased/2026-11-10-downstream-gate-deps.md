### Added — `make downstream-check` provisions each corpus's dependencies and checks the candidate's own `ailang lock` (2026-11-10)

The downstream gate used to read dependencies only from a copy of the local registry cache, so
66 entries of four blocking corpora stayed NOT_MEASURED (daneel 53, motoko-fork 11, email-parse 1,
ailang-packages 1) and the gate could never pass. Diagnosed per corpus: daneel commits no
`ailang.lock` anywhere; motoko's ABI-8 registry fixture commits a names-only lock stub its own
README says to re-lock, and its two extension dirs commit none; email-parse's root lock pins
`sunholo/eparse@1.8.1`, which the local cache did not hold; and two entries are rejected by the
loader before type checking (an import of a module the package does not export, and an invalid
manifest).

Before the arms run, the baseline binary now provisions every corpus (`tools/downstream/provision.go`):
a committed lock is honoured as committed; a package dir with dependencies and no parseable lock
gets `ailang lock` in a per-run copy of the snapshot (path dependencies resolve from the snapshot,
the snapshot itself is never written; a committed lock the gate cannot read is reported as a
finding); and with the explicit `FETCH_DEPS=1` a registry version the package root lacks, and the
registry dependencies of what was fetched, are fetched with `ailang install --no-bin` into the
run's own package-root copy. Only an HTTP 404 leaves a version UNPROVISIONED and named; any other
fetch failure is exit 3. `ailang lock` runs without network, every provisioning command has the
per-check deadline, and provisioning has a 30-minute budget.

Wherever the gate generates a lock, the candidate runs its own `ailang lock` too and the resolved
sets are compared: a candidate lock failure (`LOCK/failed`) or a different resolved set
(`LOCK/resolution-differs`) is a break, because corpora such as daneel lock with the deployed
binary in CI. Both arms' checks read the baseline's lock. The report lists every package each arm
read with its version, content hash and source, and the write audit covers the provisioned trees.

### Changed — downstream gate: 600 s default per-check timeout; loader rejections are NOT_MEASURED with their own reason (2026-11-10)

`TIMEOUT` defaults to 600 s (was 120 s, 240 s with the retry), as ruled: at 240 s motoko's root
package timed out in the baseline on a loaded machine and read as 22 false breaks. An entry whose
own manifest the loader rejects (`invalid_manifest`) or that imports a module a package does not
export (`unexported_import`) is never type-checked, so it is NOT_MEASURED with that reason rather
than counted as a missing dependency; a dependency's bad manifest stays `deps_unresolved`.
