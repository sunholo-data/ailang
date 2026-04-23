# Known false-positive / safe rules

Standing decisions per SonarCloud rule for `sunholo-data_ailang`. The `Comment` column
is the **exact text to paste** when marking the rule via `mark_safe.sh` or `mark_fp.sh` —
it becomes the audit-trail entry on SonarCloud, so reviewers later understand why.

Add to this list whenever a new rule is triaged. Never bulk-mark a rule that isn't
listed here (or listed as `Review required`).

## Hotspots — bulk via `mark_safe.sh RULE_KEY "comment"`

| Rule | Scope | Verdict | Comment |
|------|-------|---------|---------|
| `go:S2245` | `internal/observatory/seed.go` (math/rand) | **Safe** | Deterministic seeding for reproducible benchmarks; crypto/rand would break reproducibility. |
| `go:S4036` | `cmd/ailang/` (exec from PATH) | **Safe** | Running known dev tools (git/go/ailang) from PATH is standard for a developer CLI. |
| `go:S1313` | `internal/apiserver/`, `internal/effects/stream_context.go` | **Safe** | Localhost / example IP constants, not secrets. |
| `typescript:S5852` | `ui/src/.../evolutionTreeUtils.ts`, `smartLabel.ts` | **Safe** | ReDoS bounded by internal event-label inputs; never receives user content. |
| `typescript:S2245` | `ui/src/.../useEventQueue.ts` | **Safe** | Math.random used for UI jitter/debounce, not security-sensitive. |
| `go:S4507` | `cmd/ailang/main_run.go:154` | **Safe** | Opt-in CLI flag for pprof; standard Go CLI debug feature pattern. |
| `go:S5445` | `internal/coordinator/worktree.go:49`, `cmd/ailang/eval_suite.go:535` | **Safe** | Intentional shared workspace paths; coordinator runs in single-tenant contexts. |
| `go:S2077` | `internal/observatory/`, `internal/messaging/inbox.go`, `internal/coordinator/store_sqlite.go` | **Safe (spot-checked)** | Dynamic SQL uses internal enum/dimension whitelists for column/table names; user inputs always go through `?` placeholders. Spot-checked store_sqlite.go:338 (OrderBy only set from hardcoded literals by all callers), inbox.go:755 (internally-built query templates), backend_controlplane_breakdowns.go:198 (whitelisted whereClause with parameterized args). |
| `go:S4790` | `internal/apiserver/mcp_schema.go:78`, `internal/docsearch/search.go:234` | **Safe** | Non-cryptographic use: sha1 truncates module names to fit 64-char MCP limit; md5 is a deterministic simhash for word-level text search. Neither is password/credential-related. |
| `go:S5443` | `internal/repl/repl.go:226` | **Safe** | Primary history path is `~/.ailang_history`; `os.TempDir()` is a defensive fallback only when `os.UserHomeDir()` fails. |
| `Web:S5725` | `internal/server/dist/index.html:10` | **Safe** | Google Fonts stylesheet from trusted CDN in generated UI bundle; SRI would require re-hashing on every Google Fonts API change; bundle is served only on localhost from the single-tenant Collaboration Hub daemon. |

## Issues — per-issue via `mark_fp.sh ISSUE_KEY "comment"`

| Rule | File:line | Verdict | Comment |
|------|-----------|---------|---------|
| `gosecurity:S6096` | `internal/builtins/tar.go:472,476,484` | **False Positive** | Guarded by isEntryPathTraversal + filepath.Rel containment check at lines 458-468; analyzer doesn't follow the guard. |
| `go:S5542` | `internal/builtins/crypto_rsa.go:89` | **False Positive** | PKCS1v15 used for signature verification only (required for RS256 JWT interop), not encryption. |

## Issues — bulk via `mark_wontfix.sh RULE_KEY "comment"`

Use `mark_wontfix.sh` when an entire rule's findings are not product concerns in
our context (as opposed to `mark_fp.sh` for per-issue analyzer false positives).

| Rule | Scope | Verdict | Comment |
|------|-------|---------|---------|
| `go:S3776` | all Go sources | **Won't Fix** | Cognitive Complexity threshold (15) is calibrated for CRUD apps. Lexer, parser, type-checker, and VM builtins inherently have high branching — refactoring would reduce readability without improving correctness. |
| `typescript:S1082` | `ui/.../EvolutionTree*.tsx` (a11y — clickable without keyboard listener) | **Won't Fix** | EvolutionTree visualization component; a11y keyboard-listener parity is not a product requirement for this internal observability view. |
| `typescript:S3923` | `ui/.../EvolutionTree.tsx` (duplicate branches) | **Won't Fix** | Duplicate branches in EvolutionTree render logic are intentional for readability; refactor deferred to a dedicated UI rework sprint. |

## Workflow note

For hotspots: `mark_safe.sh` paginates the first 500 TO_REVIEW hotspots and filters
client-side by rule, so if a rule has >500 pending hotspots rerun the script until
its count reaches zero.

For issues: `fetch_issues.sh` prints the issue key in the last column. Pass that
key to `mark_fp.sh`. The comment is added first (for audit trail), then the
transition is applied.

For bulk Won't Fix: `mark_wontfix.sh RULE_KEY "comment"` transitions every open
issue matching the rule in a single run — same pagination caveat as `mark_safe.sh`.
