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
- Action: **`mark_safe.sh go:S2077 "..."`** using the standing comment.
- Why: Spot-checked 2026-04-21 — dynamic column/table names come from internal
  enum/dimension whitelists; user input always flows through `?` placeholders.
  Key sites: store_sqlite.go:338 (OrderBy set from hardcoded literals only),
  inbox.go:755 (internally-built query templates),
  backend_controlplane_breakdowns.go:198 (whitelisted whereClause + parameterized args).

### Insecure hash (non-crypto uses)
- Rule: `go:S4790`
- Files: `internal/apiserver/mcp_schema.go:78` (sha1), `internal/docsearch/search.go:234` (md5)
- Action: **`mark_safe.sh go:S4790 "..."`**
- Why: sha1 is used to hash-truncate module names to the 64-char MCP limit; md5
  is a deterministic simhash over words for text search. Neither is credential-
  or password-related.

### Predictable temp fallback in REPL history
- Rule: `go:S5443`
- File: `internal/repl/repl.go:226`
- Action: **`mark_safe.sh go:S5443 "..."`**
- Why: Primary history path is `~/.ailang_history`; `os.TempDir()` is a defensive
  fallback only when `os.UserHomeDir()` errors.

### SRI on Google Fonts in served UI bundle
- Rule: `Web:S5725`
- File: `internal/server/dist/index.html:10`
- Action: **`mark_safe.sh Web:S5725 "..."`**
- Why: Generated UI bundle served on localhost from the Collaboration Hub daemon.
  SRI hashes would have to be regenerated on every Google Fonts API change.

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
