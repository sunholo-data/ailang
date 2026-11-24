### Added — `make verify-gap-map`: what `ailang verify` proves across the downstream corpora

`make verify-gap-map [AILANG=<bin>] [CORPORA=a,b] [ADD=name=dir] [JOBS=n] [NOBANK=1]` runs
`ailang verify --json` over every contracted file in the `tools/downstream/corpora.yaml` corpora
(local checkouts, read only). It reports counts by status, the largest normalised skip /
inconclusive / error reasons, and the callees, types and builtins skips are blamed on. It banks a
snapshot in `~/.ailang/state/verify-gaps/` (outside the repo: some corpora are private) and diffs it
against the previous one. Exit 2 means a proof was lost, or an error/unknown appeared, in a file whose
content did not change; a change in edited source is reported as drift.

`ailang verify --json` results gain `requires_callees`: the reachable callees whose `requires` the
verifier assumes rather than checks at the call. It changes no verdict; it measures what the
call-site-precondition decision (D1) could affect. First measurement: 20 results, 11 of them
verified; the four downstream ones discharge the callee's `requires` from their own.
