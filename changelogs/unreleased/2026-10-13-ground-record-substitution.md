### Fixed — cold compiles 2–3× faster: substitution no longer re-walks ground alias records (#1787)

Since #1706 published nullary aliases as fully expanded records, every substitution walked
and re-allocated those variable-free bodies, and the solver and CoreTypeInfo apply
substitutions thousands of times per module. Motoko's `src/core/session.ail` went from 34 s
to 77 s cold. Each `TRecord` now caches whether it is ground (no type, row or effect
variables) and substitution returns it untouched; a record that does change copies its field
map only once a field actually differs. Records are never edited in place, so the cache
cannot go stale; Row and TFunc2, which are cloned and then edited, are deliberately not
cached.

| cold, empty cache | before | after |
|---|---|---|
| motoko `check src/core/session.ail` | 74 s / 443 MB | 23 s / 372 MB |
| `sunholo/crew_lab` `check --package .` | 34 s / 208 MB | 18 s / 189 MB |
| `sunholo/crew_core` `check --package .` | 15 s / 151 MB | 6 s / 142 MB |

Output is identical before and after on all three. Guard: `TestSubstitute_GroundRecordIsNotRewalked`
(fails with the check removed) plus cycle tests for the memo (`internal/types/ground.go`).
