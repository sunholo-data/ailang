# Sprint Plan: M-PROCESS-SUBCMD — Subcommand Allowlists for the Process Effect

## Summary

Extend `--process-allowlist` so an entry can name a **subcommand chain** (`git:status`,
`gh:pr:list`) and the runtime refuses everything else for that binary — through **every**
exec path, not just `exec`. Closes the consumer ask in #1137 (Daneel: `git` on the allowlist
grants `git push --force` against its own audit trail).

**Design doc:** [m-process-subcmd-allowlist.md](m-process-subcmd-allowlist.md)
**Duration:** 1 day (~6–8h; design doc said 4h — buffer is for the third exec path and the e2e test)
**Dependencies:** none (M-PROCESS-EXEC shipped v0.8.1; per-binary allowlist verified on HEAD 2026-09-11)
**Risk Level:** Low — additive parse of an existing flag, no type-system change, no new effect

## Current Status Analysis

### What exists (measured on HEAD, 2026-09-11)
- Per-binary allowlist, path-pinned at startup: `internal/effects/process.go:22` (`ResolveAllowlist`),
  checked in `processExec` (`process.go:101`). `-process-allowlist /bin/date` → `exec("/bin/echo")` = `NotAllowed(/bin/echo)`.
- Allowlisting a bash wrapper does **not** grant `/bin/bash` (reproduced: wrapper `Ok`, `exec("/bin/bash",…)` = `NotAllowed`).
- Per-subcommand: **absent** — `git` allowlisted ⇒ `git push --force` allowed.

### The systemic finding the design doc missed
The allowlist is checked in **three** places, each a hand-copied `if pc.HasAllowlist { … }` block:

| Path | Site | Today's error shape |
|------|------|---------------------|
| `exec` | `internal/effects/process.go:101` | `Err(NotAllowed(cmd))` ADT |
| `spawnProcess` | `internal/effects/process_spawn.go:149` (`resolveCommand`) | Go `error` "command not allowed" |
| `asyncExecProcess` | `internal/effects/stream_async_process.go:72` | Go `error` `_stream_async_exec_process: command not allowed` |

Adding the subcommand check to `processExec` alone (what the design doc's "Files to Modify" table
implies) leaves `spawnProcess("git", ["push"])` wide open. **M2 replaces all three with one
authorizer** — that is the fix, and the test that proves it is the sprint's most important one.

### Velocity
Recent 7d: feature commits landing at ~1/day alongside dependabot churn; last comparable
effect-runtime change (M-PROCESS-EXEC Phase 1) was ~600 LOC in a day. Capacity for this sprint:
~700 LOC total, comfortably one day.

### Scope decisions
- **`--process-denylist`: deferred**, not in this sprint. Daneel's stated need is "deny everything
  except named subcommands", which an allow-only list expresses. A deny-after-allow second mechanism
  is harder to audit (the whole point of the flag) and has no consumer. Revisit if one appears.
- **Flag-level filtering** (`git:commit` but not `--amend`): non-goal, per design doc and consumer.
- **Global options before the subcommand** (`git -C /x status`): **refused** under `git:status`.
  Matching is positional on `args[0..n]`; fail-closed is the correct default and is documented.
- **Phase 2 signature-level `Process[scope=…]`**: separate track (M-EFFECT-REFINEMENT P4/P5).

## Proposed Milestones

### Milestone 1: Parse `cmd:sub[:sub…]` into the ProcessContext
**Goal:** `ResolveAllowlist` understands colon entries; the data model can express "this binary,
only these subcommand chains".
**Estimated:** 110 impl + 120 tests = 230 LOC
**Duration:** 2h

**Tasks:**
- `internal/effects/process_context.go`: add
  `Subcommands map[string][][]string` — resolved-command-key → list of allowed arg-prefix chains.
  `nil`/absent = every subcommand (today's behaviour). Keyed by the same key as `Allowlist`
  (the entry as written: name or absolute path), so `/usr/bin/git:status` and `git:status` both work.
- `internal/effects/process.go` `ResolveAllowlist`: split each entry on `:` **after** the path
  part. Rule: for an entry beginning with `/`, the command is everything up to the first `:`;
  otherwise the command is the text before the first `:`. `git:*` ≡ bare `git`. A bare `git` and a
  `git:status` in the same list ⇒ bare wins (broadest grant, logged nowhere — it is what the
  operator wrote). Empty subcommand segment (`git:`, `git::status`) ⇒ flag error at startup, not a
  silent allow (CLAUDE.md §2).
- `process_wasm.go`: keep the no-op in step (signature unchanged).
- Tests (`process_test.go`, table-driven):
  - `git:status` → `Subcommands["git"] == [["status"]]`, `Allowlist["git"]` resolved
  - `gh:pr:list,gh:pr:view` → two chains under `gh`
  - `git,git:status` → no `Subcommands["git"]` entry (bare wins)
  - `git:*` → same as bare
  - `/usr/bin/git:status` → key is `/usr/bin/git`
  - `git:` and `git::status` → error
  - `echo,date` → `Subcommands` empty, existing tests untouched

**Acceptance Criteria:**
- [ ] Every entry shape above parses to the documented structure (table test green)
- [ ] Malformed entries fail at startup with the offending entry in the message
- [ ] Existing `TestProcessContext_ResolveAllowlist` / `_AbsolutePath` unchanged and green
- [ ] `go vet`, `gofmt -l` clean

**Risks:**
- Windows drive-letter paths (`C:\git.exe:status`) collide with `:` — Mitigation: Process exec
  is not a supported Windows surface today (`process_wasm.go`/`_js.go` are the only non-unix
  variants); document the limitation in the flag help rather than parse around it.

### Milestone 2: One authorizer, three exec paths
**Goal:** A single `(*ProcessContext).Authorize(cmdName string, args []string)` decides
allow/deny + resolved path, and `exec`, `spawnProcess`, `asyncExecProcess` all call it. A
subcommand allowlist cannot be bypassed by choosing a different std/process entry point.
**Estimated:** 90 impl + 160 tests = 250 LOC
**Duration:** 3h

**Tasks:**
- `internal/effects/process_authorize.go` (new, ~90 LOC):
  ```go
  type ProcessDenial struct{ Ctor, Detail string } // Ctor: NotAllowed | NotFound
  func (pc *ProcessContext) Authorize(cmdName string, args []string) (resolvedPath string, denial *ProcessDenial)
  ```
  Order: (1) `pc == nil` or no allowlist → `LookPath`; (2) command not in allowlist → `NotAllowed(cmd)`;
  (3) unresolved at startup → `NotFound(cmd)`; (4) `Subcommands[cmd]` present → some chain must be
  a prefix of `args`, else `NotAllowed("git push")` (command + the args that were tried, up to the
  longest chain length, joined by space; `NotAllowed("git")` when `args` is empty).
- Replace the three inline blocks:
  - `process.go:99-115` → `Authorize`; map denial to `makeProcessResultErr(denial.Ctor, denial.Detail)`
  - `process_spawn.go` `resolveCommand` → thin wrapper over `Authorize` (keeps its `error` return, message now carries the subcommand)
  - `stream_async_process.go:63-88` → `Authorize`
- Tests:
  - `TestProcessExec_Subcmd_*`: `git:status` allows `["status"]`, `["status","--short"]`; refuses `["push"]`, `["push","--force"]`, `[]`, `["-C","/x","status"]`
  - `TestProcessExec_Subcmd_MultiLevel`: `gh:pr:list` allows `["pr","list"]`, refuses `["pr","merge"]`, `["pr"]`
  - `TestProcessExec_Subcmd_MixedList`: `echo,git:status` — `echo anything` Ok, `git status` Ok, `git push` NotAllowed
  - **`TestProcessAuthorize_AllThreePathsAgree`**: for each of `exec`, `spawnProcess`, `asyncExecProcess` with `git:status` (or `echo:hello` if git absent on CI), `push` is refused — the bypass-closed proof. Uses a fake binary via `t.TempDir()` script where needed so it is hermetic.
  - Existing allowlist tests in `process_test.go`, `process_managed_test.go`, `stream_async_ops_test.go` stay green (behaviour for bare entries unchanged).

**Acceptance Criteria:**
- [ ] `grep -n "pc.Allowlist\[" internal/effects/*.go` (non-test) hits exactly one site: `Authorize`
- [ ] All-three-paths test green; each path's refusal names the subcommand (`git push`)
- [ ] `go test ./internal/effects/...` green; no test deleted except by replacement
- [ ] Error text for the existing bare case is unchanged (`NotAllowed(/bin/echo)` etc.)

**Risks:**
- `git` not present on a CI runner — Mitigation: tests use `echo:hello` / a `t.TempDir()` script for the hermetic cases and skip git-specific ones with a logged reason.
- `asyncExecProcess` returns Go errors, not the ADT — Mitigation: keep that contract (it is a Stream-side builtin); only the message changes.

### Milestone 3: CLI surface, docs, example, consumer close-out
**Goal:** An agent that greps help finds the syntax; the reference doc explains the exact
matching rule; #1137 and Daneel's three messages are answered with a verified command line.
**Estimated:** 40 impl + 80 tests + docs = ~200 LOC
**Duration:** 2h

**Tasks:**
- `cmd/ailang/main_run.go:82` flag help: `Allowed commands (comma-separated, path-pinned at startup). cmd:sub restricts a command to a subcommand chain: git:status,gh:pr:list; git:* = any`. Same text via the `--help` line in `help.go` (added 2026-09-11).
- `cmd/ailang/process_subcmd_e2e_test.go`: build-and-run the real binary (pattern: `ai_check_exit_test.go`) with `--caps IO,Process --process-allowlist echo:hello` on a temp `.ail`: `exec("echo",["hello"])` prints Ok, `exec("echo",["bye"])` prints `Err(NotAllowed(echo bye))`. This is the test that guards the flag string → runtime plumbing (`setupProcessHandler`).
- `examples/runnable/process_subcmd_allowlist.ail` + `examples/manifest.json` entry (tags `process, security, allowlist`; type-checked by `verify-examples`, which runs without `Process`). Header comment gives the exact invocation.
- `docs/docs/reference/effects.md`: extend the Process flag table row with the `cmd:sub` rule, the prefix-match semantics, the `-C` fail-closed note, and the "flags after the subcommand are allowed" caveat. Remove "(per-subcommand narrowing is unbuilt)".
- `changelogs/v0.32-current.md` Unreleased entry.
- Move design doc → `design_docs/implemented/v0_38_0/`, Status → Implemented; update link in `m-effect-refinement.md`, `m-git-guardrails.md`, `docs/docs/design-docs.md`.
- Close-out: comment on #1137 with the verified command line (`Fixes #1137` in the final commit), and `ailang messages send daneel … --type feedback` under the messaging env prefix; ack the three inbox messages (`inbox_1789049068346_981d871c`, `inbox_1789049189112_68ccbe3a`, `inbox_1789049365811_6809da5e`) individually — never `ack --all`.

**Acceptance Criteria:**
- [ ] `ailang run --help | grep process-allowlist` shows the `cmd:sub` syntax
- [ ] e2e test green against the built binary
- [ ] `make verify-examples` green with the new example in the manifest
- [ ] Design doc in `implemented/`, links resolve, CHANGELOG entry present
- [ ] #1137 closed by the final commit; Daneel messaged and acked per id

## Success Metrics
- `internal/effects` coverage on the new file ≥ 90% (small, table-driven)
- Zero remaining inline `pc.Allowlist[` checks outside `Authorize`
- Examples: 1 new, type-checked in CI
- Docs: `reference/effects.md`, CHANGELOG, design doc moved
- `make ci` green

## Dependencies
- None blocking. `git`/`gh` presence on CI is handled by hermetic fixtures.

## Open Questions
1. **Denylist** — deferred as above. Say so if you want it in this sprint (+~80 LOC, +1h).
2. **Refusal detail** — plan says `NotAllowed("git push")` (command + tried args up to the
   longest chain). Alternative is `NotAllowed("git")` + separate field. Going with the design
   doc's string form unless told otherwise; it is what the consumer's error handling already matches on.
