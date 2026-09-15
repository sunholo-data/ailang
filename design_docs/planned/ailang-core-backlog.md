# AILANG Core Backlog (triaged reports)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | `ailang lock` stores path dependencies as absolute paths, breaking committed locks across machines/users | bug | direct-fix | Lock portability doc (design_docs/implemented/v0_10_0/m-pkg-lock-portability.md) fixed registry/git deps but deliberately kept `path` deps absolute (resolver.go:127 `filepath.Abs()`), so the fix is a one-line change to store the manifest-relative path — no design decision left, and it caused a real outage (Daneel eparse, 2h no mail indexing). |