### Fixed — CodeQL and SonarCloud security and reliability findings, triaged batch (2026-11-12)

Triage of the 25 open CodeQL high alerts and the 194 open SonarCloud BUG/VULNERABILITY
issues (v1.0 codebase-health handover, workstreams 1b and 2). Real findings fixed:

- **`ailang examples download` zip extraction fails loudly on a traversal entry.** An entry
  naming a path outside the destination (`../x`, an absolute path) used to be skipped
  silently, leaving a quietly partial examples tree. It now fails the extraction. Symlink and
  other non-regular entries are refused, as `internal/pkg` refuses them in tarballs, and a
  failed `MkdirAll` is reported instead of ignored (CodeQL `go/zipslip`).
- **The coordinator task store checks `TaskFilter.OrderBy` against a list of columns.** The
  SQLite store wrote the column into `ORDER BY` unchecked. Every caller passes a constant
  today, but a name not on the list (`created_at`, `priority`, `started_at`, `completed_at`,
  `queued_at`) is now an error from both the SQLite and the Firestore store, never a default
  sort (Sonar `go:S2077`).
- **The GitHub issue sync refuses a repo that is not an exact `owner/repo`.** On the webhook
  path the repo comes from the payload and goes into a subprocess argv. A malformed one is
  now refused with a `GitHub sync REFUSED` log line (Sonar `gosecurity:S6350`).
- **serve-api's `/api/{module}/{func}` dispatch passes the loaded module's own path to the
  engine,** not the URL's copy. The two are equal after the exact-match lookup, but now no
  client-typed string reaches the module loader (CodeQL `go/path-injection`, 9 alerts).
- **`_stringToInt` parses at `strconv.IntSize`.** Behaviour is unchanged on 64-bit hosts.
  On a narrower host an out-of-range string is `None`, not a truncated number (CodeQL
  `go/incorrect-integer-conversion`).
- **The REPL's `Eq[Float]` spells its NaN test `math.IsNaN`** instead of `a != a`, with a
  test pinning NaN reflexivity (Sonar `go:S1764`, 4 of the 7 reliability bugs).
- **The secret-approval intake log quotes the caller-supplied ref,** so a newline in it
  cannot forge a log line (Sonar `gosecurity:S5145`).

The other findings are false positives, test-only code or accepted design. The PR lists the
proposed dismissals. No alert or issue was dismissed or transitioned here.
