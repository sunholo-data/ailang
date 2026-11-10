### Added — `make downstream-check` provisions each corpus's dependencies (2026-11-10)

The downstream gate used to read dependencies only from a copy of the local registry cache, so
66 entries of four blocking corpora stayed NOT_MEASURED (daneel 53, motoko-fork 11, email-parse 1,
ailang-packages 1) and the gate could never pass. Diagnosed per corpus: daneel commits no
`ailang.lock` anywhere; motoko's ABI-8 registry fixture commits a names-only lock stub its own
README says to re-lock, and its two extension dirs commit none; email-parse's root lock pins
`sunholo/eparse@1.8.1`, which the local cache did not hold; and two entries were not dependency
problems at all (an import of a module the package does not export, and an invalid manifest).

Before the arms run, the baseline binary now provisions every corpus (`tools/downstream/provision.go`):
a committed lock is honoured as committed; a package dir with dependencies and no parseable lock
gets `ailang lock` in a per-run copy of the snapshot (path dependencies resolve from the snapshot,
the snapshot itself is never written); and with the explicit `FETCH_DEPS=1` a registry version the
package root lacks is fetched with `ailang install --no-bin` into the run's own package-root copy.
A version the registry does not have stays UNPROVISIONED and named; any other fetch failure is
exit 3. The report gains a provisioning section and lists every package each arm read with its
version, content hash and source. The write audit now also covers the provisioned trees.

### Changed — downstream gate: 600 s default per-check timeout; manifest errors are verdicts (2026-11-10)

`TIMEOUT` defaults to 600 s (was 120 s, 240 s with the retry), as ruled: at 240 s motoko's root
package timed out in the baseline on a loaded machine and read as 22 false breaks. "is not
exported by package" and "invalid manifest" now normalise to `PKG/not-exported` and
`PKG/invalid-manifest` instead of `DEP/*`: both come from the corpus's own manifest, which both
arms read, so they are measured rather than counted as not measured.
