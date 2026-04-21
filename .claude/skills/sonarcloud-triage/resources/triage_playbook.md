# Triage playbook

When you see SonarCloud finding X in file Y, do Z. Mirrors `known_fp_rules.md`
from the finding side — use this when starting from a specific Sonar alert
rather than batch-triaging a whole rule.

## Hotspots

### Weak crypto (`math/rand`) in `internal/observatory/seed.go`
- Rule: `go:S2245`
- Action: **`mark_safe.sh go:S2245 "..."`** using the standing comment.
- Why: benchmarks need deterministic, seedable randomness — `crypto/rand` would
  defeat reproducibility.

### Running external commands in `cmd/ailang/`
- Rule: `go:S4036`
- Action: **`mark_safe.sh go:S4036 "..."`** using the standing comment.
- Why: CLI tools shelling out to `git`, `go`, `ailang` via PATH is standard
  developer-tool behavior.

### Hardcoded IP in `internal/apiserver/`, `internal/effects/stream_context.go`
- Rule: `go:S1313`
- Action: **`mark_safe.sh go:S1313 "..."`**
- Why: Localhost / example constants, not credentials.

### ReDoS in UI utils (`evolutionTreeUtils.ts`, `smartLabel.ts`)
- Rule: `typescript:S5852`
- Action: **`mark_safe.sh typescript:S5852 "..."`**
- Why: inputs are internal event labels, not user content; bounded.

### `Math.random` in `useEventQueue.ts`
- Rule: `typescript:S2245`
- Action: **`mark_safe.sh typescript:S2245 "..."`**
- Why: UI jitter / debounce timing, not security-sensitive.

### Debug feature in `cmd/ailang/main_run.go:154`
- Rule: `go:S4507`
- Action: **`mark_safe.sh go:S4507 "..."`**
- Why: pprof is opt-in via CLI flag; standard in Go CLIs.

### Predictable temp path
- Rule: `go:S5445`
- Files: `internal/coordinator/worktree.go:49`, `cmd/ailang/eval_suite.go:535`
- Action: **`mark_safe.sh go:S5445 "..."`**
- Why: intentional shared workspace; coordinator runs single-tenant.
- Note: `internal/repl/repl.go:220` was fixed in the code (moved to `~/.ailang_history`),
  not marked Safe.

### SQL injection hotspots
- Rule: `go:S2077`
- Files: `internal/observatory/`, `internal/messaging/inbox.go`,
  `internal/coordinator/store_sqlite.go`
- Action: **spot-check first**, do not bulk-mark. Confirm 2–3 sites use whitelisted
  table/column names rather than raw user input before running `mark_safe.sh`.

## Issues (BLOCKER / CRITICAL)

### Tar slip in `internal/builtins/tar.go:472,476,484`
- Rule: `gosecurity:S6096` (3 issues)
- Action: **`mark_fp.sh ISSUE_KEY "..."`** for each using the standing comment.
- Why: `isEntryPathTraversal` + `filepath.Rel` containment guard at lines 458-468
  defuses the path traversal; Sonar's analyzer doesn't follow the guard through.

### PKCS1v15 in `internal/builtins/crypto_rsa.go:89`
- Rule: `go:S5542`
- Action: **`mark_fp.sh ISSUE_KEY "..."`** using the standing comment.
- Why: used only for signature verification (required for RS256 JWT interop),
  not encryption. PKCS1v15 verify is the standard path.

## Real fixes already landed

- `docs/src/components/ModelRadarComparison/index.jsx:53` — added `localeCompare`
  comparator (rule `javascript:S2871`). Commit `76d0e9a5`.
- `internal/repl/repl.go:220` — moved history file from `/tmp/.ailang_history`
  to `~/.ailang_history` (rule `go:S5445`). Commit `76d0e9a5`.

## Out of scope / configuration (not triaged via this skill)

- `sonar-project.properties` controls which paths are scanned. `tools/`, `benchmarks/`,
  `scripts/`, `.claude/`, `examples/` are excluded — any finding reported in those
  paths means the exclusion list is out of date, fix `sonar-project.properties`
  rather than mark Safe.
- `main_run.go` cognitive complexity 210 and `internal/trace/otel_emitter.go`
  complexity 123 are tracked as tech debt, not triaged here.
