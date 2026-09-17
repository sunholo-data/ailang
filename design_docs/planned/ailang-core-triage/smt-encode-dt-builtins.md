# SMT-encode pure _dt_* datetime builtins (addDays/startOfDay/weekday) for verify

- **Date**: 2026-09-15
- **Class**: feature
- **Recommend**: duplicate-of design_docs/planned/ailang-core-backlog.md
- **Searched**: SMT, encodab, addDays, weekday, startOfDay, datetime across design_docs/; `_dt_` across internal/ and std/
- **Estimate**: (inherited from existing row — direct-fix, encoding map extension in `internal/smt/encodable.go` plus callee resolver)

Already ruled on: `design_docs/planned/ailang-core-backlog.md` carries a row dated 2026-09-15
for this exact report — "Encode pure _dt_* datetime builtins (addDays/startOfDay/weekday) for SMT",
class `feature`, recommend `direct-fix` — citing the same UTC-ms encodings the reporter supplied
(`addDays` = `ts + n*86400000`, `startOfDay` = `ts - ts mod 86400000`, `weekday` =
`((ts div 86400000) + 4) mod 7`), the same consumer (Daneel calendar sprint 2), and the same
secondary item (zero-argument callees rejected as `UNENCODABLE_TYPE ()`; cf.
`firstUnencodableBuiltin` in `internal/smt/encodable.go`). No design decision remains that a
reviewer could disagree with, so re-triaging would just fork the ruling. Calendar-irregular
`addMonths`/`addYears` staying opaque is also already stated there.

The mechanism checks out: `_dt_add` lives in `internal/builtins/datetime.go` and the reject path
names the blocking builtin via `firstUnencodableBuiltin` (`internal/smt/encodable.go`), matching
the reporter's `'addDays' that is not SMT-encodable` message.

Recommendation: proceed straight to the direct fix under the existing backlog row; do not open a
design doc.