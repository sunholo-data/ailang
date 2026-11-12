### Fixed — registry validator: bounded tarball extraction, query-string API keys refused (2026-11-12)

Found during the 2026-11-12 CodeQL/Sonar triage (#2253); neither was on an alert list.

- **Decompression bomb.** `pkg.ExtractTarball`, shared by the registry validator, `ailang install`,
  `ailang pkg docs` and the dependency resolver, read every tar entry with an unbounded
  `io.ReadAll`, and the validator's request body was bounded only by `ParseMultipartForm`'s
  in-memory threshold. A small gzip of zeros could fill the validator's memory-backed `/tmp` and
  kill the instance. Extraction now refuses, with a `*pkg.TarballLimitError` that names the limit,
  any tarball over 32 MiB compressed (`MaxTarballBytes`), 8 MiB per entry
  (`MaxTarballEntryBytes`), 32 MiB extracted in total (`MaxTarballTotalBytes`) or 1000 entries
  (`MaxTarballEntries`). The decompressed gzip stream also has its own cap, which covers headers and
  the bodies of skipped entry types. Entries are streamed to disk instead of buffered. The validator
  caps the request body before parsing it and answers 413 with the limit's name. The limits are in
  `internal/pkg/tarball_limits.go`. They are 7–46x above the largest of the 450 tarballs published
  as of 2026-11-12, and all 450 still extract.
- **API key in the query string.** The validator accepted `?api_key=`, which puts keys in the
  Cloud Run request log. It now answers 400 and names the `X-API-Key` header. Every released
  `ailang publish`, `unpublish` and `pkg keys` has sent the header since auth was added in v0.10.0,
  and the request log showed no query-string keys in the 30 days up to the change.

The validator ships through its own release trigger (`ailang-registry-validator-release`). It
needs a release before either fix takes effect.
