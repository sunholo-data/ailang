# M-GIT-BINARY-RESOLUTION-SWEEP: One Absolute-Path Git Resolver for All 92 Exec Sites

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1
**Estimated**: 4 days (M1: 1.5d, M2: 1d, M3: 1d, M4: 0.5d)
**Dependencies**: None
**Planner-Lane**: codex-ok (M2–M4 are mechanical conversions; M1 fixes the contract and should get the most review attention)

---

## Problem Statement

The SonarCloud quality gate is red partly on `new_security_rating` = B (actual 2, needs 1),
driven by exactly **4 open issues, all rule `go:S4036`** ("Make sure the PATH variable only
contains fixed, unwriteable directories") — bare-name `exec.Command("git", ...)` calls that
resolve `git` through whatever `PATH` happens to contain at run time (Verification Log V1).

Those 4 sites are not special. They are merely the 4 members of an identical class that were
*touched inside the new-code period* (`previous_version = v0.33.2`, 2026-08-26). The full class
is **92 bare-name git exec sites** across `cmd/ailang` (43), `internal/coordinator` (43),
`internal/pkg` (5), and `internal/eval_harness` (1) — and **zero** Go files outside `cmd/` and
`internal/` contain any (V2, V9, V13). Fixing only the flagged 4 would turn the gate green while
leaving 88 identical sites in place: gate-satisfying, not a fix. Per the mission's standing rule
(3n(d)), a new-code gate hit is *evidence about the unswept class*, never a threshold to satisfy.

Meanwhile, the repo already contains the correct pattern — and uses it exactly once.
`cmd/ailang/help.go` defines `resolveGit(look)` + a `sync.Once`-cached `gitBinary()` that
require an **absolute** resolution and refuse anything else, with the rationale written in its
doc comment: the tool "must not execute whatever a relative or writable PATH entry happens to
call git" (V4). It lives in `package main`, so no `internal/` package can import it, and its
consumer count outside its own file is **zero** (V3). The knowledge exists; the plumbing to
share it does not.

**Current State:**
- 92 bare-name git exec sites, 0 of which route through the existing resolver (V2, V3)
- 4 of 92 visible to Sonar's new-code gate; 88 invisible because they are old (V1 vs V2)
- The one hardened resolver is trapped in `package main` (`cmd/ailang/help.go:173,185`) (V3, V4)
- No CI gate prevents new bare-name git exec sites from being added (no such check exists in
  `make/*.mk` or `scripts/` — V7 shows the boundary gate; nothing polices exec sites)

**Impact:**
- Security posture: every one of the 92 sites executes whatever `PATH` resolves `git` to,
  in daemons (coordinator), eval harnesses, and CLI paths. Severity is MINOR per Sonar, but
  the class is broad and includes long-running privileged-ish processes (worktree management,
  cloud task execution).
- Process: the Sonar gate stays red on `new_security_rating` until the flagged sites change,
  and *any* future PR touching one of the 88 old sites re-flags it — a slow drip of
  one-off "fix the flagged line" patches unless the class is closed once.

## Goals

**Primary Goal:** Every git execution in the repo goes through one shared absolute-path
resolver, and CI makes a new bare-name git exec site a build failure.

**Success Metrics:**
- Bare-name git exec sites (non-test): 92 → 0 by end of M4, with the residual count stated
  explicitly at each milestone (88 after M1, 46 after M2, 6 after M3, 0 after M4)
- `LookPath("git")` call sites repo-wide: exactly 1, inside the new shared package (V6 shows 1 today, in `help.go`; it moves)
- Sonar `go:S4036` open issues in the class: 4 → 0 flagged sites converted (the resolver's own
  single `LookPath` may attract one residual finding — see "Sonar interaction")
- CI gate `make check-git-exec` exists, runs in ci.yml, and fails loudly on an empty
  enumeration (anti-vacuity fixture)

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Failure contract: deferred error via `exec.Cmd.Err` (os/exec's own Go 1.19+ contract), **no** bare-`"git"` fallback, **no** panic | Dictates the shape of all 92 conversions and the UX for a user without git | agent (this doc) | design | high after M2 starts |
| Helper location: new stdlib-only leaf package `internal/gitexec` | Must be importable by `package main` AND `internal/{coordinator,pkg,eval_harness}` without violating layer rules | agent (this doc) | design | med |
| CI gate is a per-file baseline **ratchet** (fails on increase AND on untightened decrease) with a known-positive anti-vacuity fixture | A count-only or vacuously-green gate lets the class recur silently | agent (this doc) | design | low |
| Milestone split with explicit residuals (4 → then 42 → 40 → 6) | Prevents the doc being read as claiming full coverage at M1 | agent (this doc) | design | low |

No design-freeze items: every decision above is agent-resolvable, preserves today's observable
failure behavior for users, and touches no cost/KPI/banked-data semantics.

**Quorum trigger analysis** (per design-doc-creator checklist): trigger 1 (freeze items) — no.
Trigger 2 (overrides shared machinery) — no: it *creates* shared machinery and converts callers;
`scripts/check_boundaries.sh` is reused as a pattern, not modified. Trigger 3 (cost/KPI/schema)
— no. Trigger 4 (external-system premises) — **arguably yes**: the problem statement leans on
SonarCloud's leak-period semantics (`sinceLeakPeriod=true`, an API whose sibling parameter
`inNewCodePeriod` is silently ignored — V1's control). The design's *substance* (92 sites, the
resolver, the gate) is verifiable entirely in-repo. Recommendation: run `ailang design-quorum`
before planning, since this is an unattended-loop doc.

## Deferred Decisions

- Whether to extend the same treatment to other bare-name binaries (`gh`, `bash`, `ollama`,
  …). Out of scope here; the gate's enumerator is deliberately git-only so it cannot silently
  half-cover a class it was never designed for. A follow-up doc can generalize.
- Whether test files (`_test.go`) should also be swept. Deferred: tests run in CI/dev
  environments, several deliberately construct degenerate shapes, and sweeping them adds churn
  without changing the shipped attack surface. The gate excludes `_test.go` **explicitly and
  visibly** (the exclusion is printed in its output) so the choice stays auditable.

## Solution Design

### Overview

1. Create `internal/gitexec` — a small (~150 LOC), stdlib-only package holding the single
   sanctioned `exec.LookPath("git")` in the repo, with the absolute-path refusal semantics
   lifted from `cmd/ailang/help.go`.
2. Convert git exec sites to `gitexec.Command` / `gitexec.CommandContext` in four milestones,
   starting with the 4 Sonar-flagged sites, ending at 0 bare sites.
3. Add `make check-git-exec` (script + make target + ci.yml step, following the
   `check-boundaries` pattern — V7) with a per-file ratcheting baseline and an anti-vacuity
   fixture, so the class cannot recur and the gate cannot rot into a vacuous checkmark.

### Architecture

**Package: `internal/gitexec`** (new; stdlib-only: `os/exec`, `context`, `errors`, `fmt`,
`path/filepath`, `sync`)

```go
package gitexec

// ErrUnresolvable is returned (wrapped) whenever git cannot be resolved to an
// ABSOLUTE path. Requiring an absolute result is deliberate: callers must not
// execute whatever a relative or writable PATH entry happens to call "git".
var ErrUnresolvable = errors.New("git is not resolvable to an absolute path")

// Path returns the process-wide cached absolute path to git, or an error
// wrapping ErrUnresolvable. Resolution runs once per process (sync.Once),
// caching success AND failure.
func Path() (string, error)

// Command returns an *exec.Cmd for the resolved absolute git path.
// On resolution failure it returns a Cmd whose Err field carries the
// resolution error, so Run/Start/Output/CombinedOutput fail with it —
// exactly os/exec's own contract for a failed LookPath (Go >= 1.19; this
// repo is on go 1.26.6 and Cmd.Err exists — V10).
func Command(args ...string) *exec.Cmd

// CommandContext is Command with a context.
func CommandContext(ctx context.Context, args ...string) *exec.Cmd

// resolveWith is the test seam: identical logic, injectable lookup.
func resolveWith(look func() (string, error)) (string, error)
```

**The failure contract — decided, with alternatives rejected:**

- **Chosen: deferred error via `Cmd.Err`.** On resolution failure, `gitexec.Command` returns a
  `*exec.Cmd` with `Err` set to `fmt.Errorf("gitexec: %w: %v", ErrUnresolvable, cause)`;
  `Run`/`Output` return that error. This mirrors what `os/exec` itself already does when
  `exec.Command("git", ...)` cannot find git — so **every one of the 92 call sites keeps its
  existing error-handling control flow unchanged**. A user whose git is genuinely not on PATH
  sees the same failure point as today (the `Run`/`Output` error path), with a *sharper*
  message: today `exec: "git": executable file not found in $PATH`; after,
  `gitexec: git is not resolvable to an absolute path: exec: "git": executable file not found in $PATH`.
  No new failure mode, no behavioral cliff, and `errors.Is(err, gitexec.ErrUnresolvable)` is
  available to any site that wants to special-case it.
- **Rejected: `(cmd, err)` two-value return.** Correct but forces a second error check at all
  92 sites *before* the existing `Run`/`Output` check — roughly doubling the diff and creating
  92 opportunities for a careless `_ =` swallow. The deferred contract achieves the same
  loudness through the error path every site already has — **except at the sites that already
  discard it**, and there are at least five: `internal/coordinator/daemon_tasks_exec_run.go:461,463`,
  `cmd/ailang/coordinator_browse.go:87,240`, and `internal/coordinator/worktree.go:239`
  (`_ = cmd.Run()`, "Ignore errors"). Those keep today's silent-swallow behaviour after
  conversion — this design neither improves nor worsens them, and the claim of uniform
  improved diagnostics across "every one of the 92" does **not** hold for them. They are a
  pre-existing class, listed here rather than fixed, so no reader mistakes conversion for
  diagnosis.
- **Rejected: fallback to bare `"git"` on resolution failure.** This is a silent fallback in
  exactly the sense CLAUDE.md's "NO SILENT FALLBACKS" principle forbids: it would re-create
  the S4036 condition precisely in the case the resolver exists to prevent, and it would do so
  invisibly. The one environment where the fallback would "help" (git absent or only
  relatively resolvable) is the one where running an arbitrary `git` is least safe.
- **Rejected: fail-closed (panic / os.Exit).** The biggest consumer is the coordinator daemon
  (`internal/coordinator`, 43 sites). A missing git must fail the *task* that needed git, with
  a typed error the daemon can log and bank — not kill a long-running process that also serves
  non-git work.

**Special consumer — `cmd/ailang/help.go` (M1):** `resolveGit`/`gitBinary` are deleted and the
stale-check probes call `gitexec.Path()`, mapping error → `""` to preserve the existing
"undeterminable → show the warning" semantics (V4). `help_stale_test.go` references
`gitBinary()` at 6 call sites (V3) and is rewritten against the new seam per the repo's
testing policy (no backward-compat shims). The declared-unpinnable empty-`git` guards in
`gitHead`/`gitDirty` keep working unchanged since they take the path as a parameter.

**Placement justification (boundary rules):** `make check-boundaries` runs
`bash scripts/check_boundaries.sh` (V7), which enforces two rules over *fixed package-name
sets*: CORE = {parser, types, eval, core, elaborate, effects, builtins, lexer, ast, pipeline,
runtime, link, iface}, DASHBOARD = {server, coordinator, observatory, messaging}, and a
CORE_SURFACE deny-list for dashboard imports. `internal/gitexec` is in none of these sets and
imports only the standard library, so: `cmd/ailang` (package main) may import it, DASHBOARD
packages (`internal/coordinator`) may import it (rule 2 only denies the compiler surface), and
no CORE package will import it (it has no reason to; rule 1 unaffected). `internal/pkg` and
`internal/eval_harness` are in the unpoliced "tools" layer (V7). No existing package named
`gitexec`/`gitutil`/`osutil`/`executil` exists to collide with (V5), and no second
`LookPath("git")` resolver exists anywhere to consolidate besides help.go's (V6).

### CI gate: `make check-git-exec`

New `scripts/check_git_exec.sh` + target in `make/code-health.mk` (next to `check-boundaries`,
V7) + a ci.yml step after "Check architecture boundaries" (ci.yml:133, V7) + membership in the
`ci:` aggregate target (make/ci.mk:11).

**Mechanism, in order — each step fails loudly:**

1. **Anti-vacuity control first.** The script keeps a known-positive fixture (a checked-in
   text file, e.g. `scripts/testdata/git_exec_gate_positive.txt`, containing a literal
   `exec.Command("git", "status")` line — a `.txt`, not `.go`, so it never compiles or gets
   swept). The enumerator regex is run against the fixture **before** the tree; **0 matches on
   the fixture → exit 2 "INSTRUMENT BROKEN"**, never a checkmark. An empty enumeration over
   the tree is only believable after the instrument has matched a known positive.
2. **Enumerate** bare-name git exec sites over all `*.go` under the repo root, excluding
   `_test.go`, `vendor/`, and `internal/gitexec/` (the one sanctioned site), with the regex
   `exec\.Command(Context)?\([^)"]*"git"` (validated in this doc: narrow and wide variants
   agree at 92 with zero diff — V2; nothing exists outside cmd/ and internal/ — V13).
3. **Compare against a per-file baseline** (`scripts/git_exec_baseline.txt`, `path count`
   lines). Failure conditions, all loud: (a) a file with matches that is absent from the
   baseline; (b) a count above baseline for any file; (c) **a count below baseline** — the
   run fails with "tighten the baseline", so the ratchet can only move down by an explicit,
   reviewed edit and can never silently loosen later.
4. **Positive invariant on the resolver:** exactly one `exec.LookPath("git")` in
   `internal/gitexec/` and zero elsewhere. This catches both drift shapes: someone adding a
   second resolver, and someone gutting the sanctioned one.

**What the enumerator CANNOT see — stated so nobody mistakes the gate for full coverage:**

- **Variable-first-arg exec**: `exec.CommandContext(ctx, git, ...)` where `git` is a variable
  (this exact shape exists today at `cmd/ailang/help.go:204,222` — V11). After the sweep this
  is the *good* shape's residue inside gitexec-fed code, but a hostile/careless author could
  put a bare `"git"` in a variable one line up. The regex cannot follow dataflow.
- **Shell strings**: `exec.Command("bash", "-c", "... git ...")`. One `bash -c` site exists
  today (`internal/eval_harness/watchdog.go:57` — V12; its command is process-tree cleanup,
  not git, so it is out of class — verified by reading the site).
- **Multi-line calls — the blind spot the evaluator found, and the most likely of the three to
  bite.** `grep` is line-anchored by default (no `-z`), so the regex cannot match when the
  `"git"` literal sits on a different physical line from `exec.Command(`:

  ```go
  cmd := exec.Command(
      "git", "status",
  )
  ```

  Measured: that shape returns **0** with the single-line known-positive control returning
  **1** in the same call. This is not hypothetical — **`internal/eval_harness/watchdog.go:57`
  is already written this way today** (V12 records that site for its `bash -c` shell-string
  shape and missed that it is *also* a multi-line call, which is exactly how a blind spot
  hides: the site was looked at, through the wrong lens). `gofmt` produces this shape
  naturally as soon as an argument list grows, so a future `git` call is more likely to be
  invisible the longer it is. **Mitigation for the sprint:** the gate SHOULD join logical
  lines before matching (`gofmt`-normalise, or match over `go/ast` rather than text) — and if
  the executor keeps the textual form, this limitation must be restated in the gate's own
  output, not only here.
- `syscall.Exec`, execution of scripts under `tools/` or launchd plists that themselves call
  git, and any non-Go surface (shell scripts, Makefiles). These are different classes with
  different owners; the gate is honest about only policing direct Go `os/exec` use of `"git"`.

**Sonar interaction:** converting the 4 flagged sites resolves the 4 `go:S4036` issues
(the flagged lines stop invoking `exec.Command("git", ...)`). The resolver's own single
`LookPath("git")` may attract one residual S4036 finding; if it does, it is marked reviewed-safe
via the `sonarcloud-triage` flow with a justification pointing at this doc — a per-site
disposition on the one auditable site, **not** a repo-wide rule suppression.

### Implementation Plan

**M1 — helper + flagged sites + gate (1.5d).** Residual after M1: **88 bare sites remain.**
- Create `internal/gitexec` (resolver, `Path`, `Command`, `CommandContext`, test seam).
- Convert the 4 Sonar-flagged sites: `cmd/ailang/prompt_freeze_check_git.go:82` (`gitBytes`),
  `cmd/ailang/prompt_freeze_core.go:194` (`scanCorpus`), `internal/coordinator/worktree.go:308`
  (repo-dir probe), `cmd/ailang/coordinator_cloud.go:459` (ahead-log probe). All four already
  consume `Run`/`Output` errors, so the deferred contract drops in without control-flow edits
  (site bodies read at V8's companion reads; see V1 for the flagged list).
- Migrate `cmd/ailang/help.go` off its private `resolveGit`/`gitBinary` (delete both; rewrite
  `help_stale_test.go` accordingly).
- Land `scripts/check_git_exec.sh` + make target + ci.yml step, baseline seeded at the
  measured post-M1 per-file counts (sum = 88).
- Refusal-branch tests + neutering mutations (below).

**M2 — `internal/coordinator` sweep (1d).** Converts the remaining 42 coordinator sites
(`worktree.go` 15 more, `merge.go` 8, `approval_processor.go` 6, `artifact_discovery.go` 5,
`daemon_tasks_exec_run.go` 4, `daemon_tasks_worktrees.go` 3, `observatory_sync.go` 1 — V9).
Baseline ratchets 88 → **46 remain.**

**M3 — `cmd/ailang` sweep (1d).** Converts the remaining 40 cmd sites (`coordinator_cloud.go`
14 more, `coordinator_browse.go` 11, `chains_diff.go` 4, `coordinator_inspect.go` 4,
`messages_send.go` 3, `coordinator_utils.go` 3, `coordinator_cloud_github.go` 1 — V9).
Baseline ratchets 46 → **6 remain.**

**M4 — tools layer sweep (0.5d).** Converts `internal/pkg/gitcache.go` (5) and
`internal/eval_harness/gemini_evaluator_bridge.go` (1). Baseline ratchets 6 → **0**; the
baseline file then asserts emptiness and the anti-vacuity fixture is what keeps the gate
non-vacuous forever after.

**Concurrent-session non-overlap (UPDATED — that session has since LANDED as `4d8705699`, one
commit ahead of this doc's base `e38c0c493`; the technical conclusion below is unchanged and was
re-verified against the merged commit: it introduces no new `exec.Command` git sites and touches
none of the 18 sweep files):** at authoring time a live session held uncommitted edits to
`std/string.ail`, `cmd/ailang/docs.go`, `cmd/ailang/prompt.go`, `cmd/ailang/verify.go`,
`internal/prompt/loader.go`, `prompts/versions.json`. **None of those files contains a git
exec site** (V8: grep over exactly those Go files = 0 matches, control = 1 match on a known
positive), so no sweep milestone touches them and nothing needs deferring. Standing rule for
the executor: if new git exec sites appear in those files by execution time, defer those files
to a follow-up commit after that session lands, and say so in the sprint log.

### Files to Modify/Create

- `internal/gitexec/gitexec.go` — NEW (~120 LOC): resolver, ErrUnresolvable, Path/Command/CommandContext, test seam
- `internal/gitexec/gitexec_test.go` — NEW (~200 LOC): refusal-branch tests + mutation protocol comments
- `scripts/check_git_exec.sh` — NEW (~120 LOC): gate with anti-vacuity fixture + ratchet baseline
- `scripts/git_exec_baseline.txt` — NEW: per-file allowed counts (seeded at 88, ratchets to 0)
- `scripts/testdata/git_exec_gate_positive.txt` — NEW: known-positive fixture for the instrument check
- `make/code-health.mk` — MODIFY (+6 LOC): `check-git-exec` target beside `check-boundaries` (V7)
- `make/ci.mk` — MODIFY (+1 word): add `check-git-exec` to the `ci:` aggregate (line 11 — V7)
- `.github/workflows/ci.yml` — MODIFY (+2 LOC): step after `make check-boundaries` (line 133 — V7)
- `cmd/ailang/help.go` — MODIFY (M1): delete `resolveGit`/`gitBinary`, call `gitexec.Path()`
- `cmd/ailang/help_stale_test.go` — MODIFY (M1): rewrite against the new seam (6 references — V3)
- `cmd/ailang/prompt_freeze_check_git.go`, `cmd/ailang/prompt_freeze_core.go`, `internal/coordinator/worktree.go`, `cmd/ailang/coordinator_cloud.go` — MODIFY (M1): the 4 flagged sites
- 14 more files across M2–M4 per the V9 distribution table
- `CHANGELOG.md` / `changelogs/v0.32-current.md` — MODIFY: entry per milestone

## Conflict Surface

This design touches no parser/lexer/typechecker/codegen path, so the mandatory trigger does
not fire; the section is included because the sweep's blast radius is wide (18 files, 92
sites) and the honest answer is not "no conflicts".

### Positions touched
The process-spawn surface: every direct Go `os/exec` invocation of `"git"` in the repo, plus
the CI gate lane (`make/code-health.mk`, `make/ci.mk`, ci.yml).

### What else lives here
- **Non-git exec sites** (`gh`, `bash`, `ollama`, editors, …) share the same call shape. The
  gate's regex is git-literal-anchored so it can never fire on them; the sweep must not touch
  them (Non-Goals).
- **Variable-first-arg git calls** (`help.go:204,222` — V11) are the *output* shape of the old
  resolver; after M1 they take their path from `gitexec.Path()` and remain out of the
  enumerator's sight by design.
- **`cmd.Dir` vs `-C` conventions** differ across sites (e.g. `gitBytes` sets `cmd.Dir`;
  `worktree.go` passes `-C`). The sweep preserves each site's convention verbatim — only the
  binary resolution changes. This keeps every conversion a two-line mechanical diff.
- **The concurrent worktree session's six files** — zero overlap, verified (V8).

### Disambiguation strategy
Not applicable (no grammar change). The gate disambiguates sanctioned vs bare use purely by
path: `internal/gitexec/` is the only directory allowed to name `"git"` in an exec call or
`LookPath`.

### Programs/flows that MUST still work post-change
- `ailang --help` staleness warning on a dirty checkout (help.go probes; behavior-preserving
  migration in M1)
- `ailang prompt freeze --check` (prompt_freeze_check_git.go / prompt_freeze_core.go)
- Coordinator worktree lifecycle: create/validate/merge/cleanup (`internal/coordinator`)
- Cloud task execution clone/commit/push/PR flow (`coordinator_cloud.go`)
- `ailang messages send` git-context capture; `ailang chains diff`; package `gitcache`
  clone/fetch
- All of `make ci` — in particular `check-boundaries` must stay green with the new package
  (it will: `gitexec` is in no policed set — V7)

### What deliberately changes
- A git resolved only via a **relative** PATH entry is now refused even where Go's own
  `ErrDot` protection has been overridden (`GODEBUG=execerrdot=0`). Today's behavior in that
  corner executes the relative git; after, the site's error path fires with `ErrUnresolvable`.
  This is the point of the design.
- Error text at failing sites changes from `exec: "git": executable file not found in $PATH`
  to the wrapped `gitexec: ...` form. Anything parsing that string would break — no in-repo
  code matches on it (checked while reading the 4 flagged sites; none inspect error text).

## Testing Strategy

### Refusal-branch enumeration + mutation protocol

The helper has exactly four refusal/decision branches. Each gets one test AND one neutering
mutation of the form `if false && <cond>` — imports stay used, the mutant **builds**, so "the
mutant does not compile" can never masquerade as "the guard fired". A test is only accepted if
it fails under its paired mutant.

| # | Branch | Test | Neutering mutation that the test must kill |
|---|--------|------|--------------------------------------------|
| B1 | `look()` returns error → refuse | `resolveWith(fail)` returns err wrapping `ErrUnresolvable` | `if false && err != nil` in the error check |
| B2 | `look()` returns a **relative** path → refuse | `resolveWith(-> "git", nil)` and `-> "./git"` both refused | `if false && !filepath.IsAbs(p)` |
| B3 | `Command`/`CommandContext` after failed resolve → returned Cmd carries `Err`; `Run` fails with `errors.Is(_, ErrUnresolvable)` | run a Cmd built under an injected failing resolver; assert error identity from `Run()`, not just construction | `if false && resolveErr != nil` at the `Cmd.Err` assignment |
| B4 | Success path → `Cmd.Path` is the absolute resolved path; `Args[0]` sane; context honored for `CommandContext` | `resolveWith(-> "/abs/git", nil)`; assert `cmd.Path == "/abs/git"` and a cancelled context aborts `Run` | swap the assignment to the bare literal: `Path: "git"` (builds; B4's absolute-path assertion kills it) |

Plus one **caching** test (not a refusal branch): the `look` func is invoked exactly once
across two `Path()` calls, including when the first call *failed* — failure is cached too, and
the test asserts the second call returns the same error without re-invoking `look`.

### Gate self-tests
- Instrument test: run the script against a tree copy with the fixture removed from the
  testdata path → must exit 2, not 0 (an empty enumeration must FAIL LOUDLY).
- Ratchet test: add a synthetic bare site under a temp file → exit 1 naming the file; remove
  a real site without editing the baseline → exit 1 "tighten the baseline".

### Regression
- `make test` (all packages touched per milestone), `make check-boundaries`,
  `make check-file-sizes` (gitexec is ~120 LOC, far under limits), full `make ci` before each
  milestone's merge.

## Non-Goals

- Sweeping non-git binaries (`gh`, `bash`, `ollama`, …) — same class shape, separate doc.
- Sweeping `_test.go` files (excluded visibly; see Deferred Decisions).
- Policing shell scripts, `tools/`, launchd plists, or anything outside direct Go `os/exec`
  use of `"git"` (stated as enumerator blind spots above).
- Repo-wide suppression of Sonar rule `go:S4036` — per-site disposition on the single
  sanctioned resolver only.
- Vendoring or shelling to a pinned git version; PATH is still *read* once at resolve time —
  the change is that its result must be absolute and is resolved at one auditable site.

## Success Criteria

- [ ] M1: `internal/gitexec` exists; the 4 Sonar-flagged sites and `help.go` route through it;
      refusal-branch tests pass and each documented mutant is killed; `make check-git-exec`
      runs in CI with baseline sum = **88** — the doc and the baseline both state that
      **88 bare sites remain** (this criterion is unmeetable by a doc claiming full coverage)
- [ ] M2: coordinator sweep done; baseline sum = **46 remaining**
- [ ] M3: cmd/ailang sweep done; baseline sum = **6 remaining**
- [ ] M4: baseline sum = **0**; positive invariant (exactly 1 `LookPath("git")`, in
      `internal/gitexec/`) enforced by the gate
- [ ] Gate anti-vacuity verified: fixture-missing run exits 2 (loud), never green
- [ ] Sonar: 4 `go:S4036` new-code issues no longer attach to the converted lines; any
      residual finding on the resolver is dispositioned per-site with a justification
- [ ] All tests passing; CHANGELOG updated per milestone; no edits to the six
      concurrent-session files

## Verification Log

Every "the codebase currently does X" claim above is backed by a row here. Commands were run
at `e38c0c493` (origin/dev) from the repo root on 2026-08-27. Empty/negative results carry a
same-call, same-path known-positive control.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | Exactly 4 new-code `go:S4036` issues, at the 4 named sites; leak-period narrows (instrument works) | `curl -s "https://sonarcloud.io/api/issues/search?componentKeys=sunholo-data_ailang&sinceLeakPeriod=true&rules=go:S4036&ps=50"` piped to python; controls: same query without `rules` (`&ps=1`) and without `sinceLeakPeriod` | `total: 4` — `prompt_freeze_check_git.go:82`, `prompt_freeze_core.go:194`, `worktree.go:308`, `coordinator_cloud.go:459`, all MINOR/OPEN. Controls: `leak: 19`, `all: 2404` (the leak-period filter demonstrably narrows; `inNewCodePeriod` is the known-broken parameter, not used) |
| V2 | 92 bare-name git exec sites (non-test) in cmd+internal; regex not under-matching | narrow `grep -rn --include='*.go' -E 'exec\.Command(Context)?\((ctx, )?"git"' cmd internal \| grep -v '_test.go' \| wc -l`; wide `[^)"]*"git"` variant; `diff` of sorted outputs; control incl. tests | narrow = **92**, wide = **92**, diff = empty; control including `_test.go` = 108 (≥ 92, instrument sees positives); known-positive control `grep -n 'exec.Command' cmd/ailang/prompt_freeze_check_git.go` → line 82 |
| V3 | The existing resolver has zero consumers outside its own file (+ its tests) | `grep -rn --include='*.go' 'gitBinary(' cmd internal` | Hits only in `cmd/ailang/help.go` (:41, :185 def) and `cmd/ailang/help_stale_test.go` (6 refs) |
| V4 | `resolveGit`/`gitBinary` exist at help.go:173/185, require ABSOLUTE, `""` = show-warning semantics | `sed -n '160,220p' cmd/ailang/help.go` | Doc comment: "must not execute whatever a relative or writable PATH entry happens to call \"git\"… empty result makes both probes report undeterminable, which SHOWS the warning"; `sync.Once` cache confirmed |
| V5 | No existing package could host this (negative-existence) | `ls internal \| grep -i 'git\|osutil\|executil'` with control `ls internal \| grep -c 'coordinator'` | grep exit=1 (no match); control = 1 (the pipeline finds known packages) |
| V6 | Exactly one `LookPath("git")` in the repo (non-test) | `grep -rn --include='*.go' 'LookPath("git")' cmd internal \| grep -v '_test.go'` | Single hit: `cmd/ailang/help.go:186` (this is also the positive control for the pattern) |
| V7 | Boundary gate = fixed name-set rules; `check-boundaries` lives in make/code-health.mk:161, aggregate in make/ci.mk:11, CI step at ci.yml:133; `gitexec` in no policed set | `make -n check-boundaries`; `grep -rn 'check-boundaries' make/*.mk`; `sed -n '128,138p' .github/workflows/ci.yml`; `sed -n '1,80p' scripts/check_boundaries.sh` | `bash scripts/check_boundaries.sh`; `make/code-health.mk:161`, `make/ci.mk:11`; ci.yml step "Check architecture boundaries"; CORE_PKGS/DASHBOARD_PKGS/CORE_SURFACE_PKGS arrays contain fixed names, none is `gitexec` |
| V8 | The six concurrent-session files contain ZERO git exec sites | `grep -n -E 'exec\.Command(Context)?\([^)"]*"git"' cmd/ailang/docs.go cmd/ailang/prompt.go cmd/ailang/verify.go internal/prompt/loader.go` with control `grep -c <same pattern> cmd/ailang/prompt_freeze_check_git.go` | exit=1 (no matches); control = 1 (same regex, known positive). The two non-Go files (`std/string.ail`, `prompts/versions.json`) are outside `--include='*.go'` scope by construction |
| V9 | Per-file distribution: cmd 43, coordinator 43, pkg 5, eval_harness 1; 69 `Command` + 23 `CommandContext` | `grep -rc --include='*.go' -E '<wide pattern>' cmd internal \| grep -v ':0$' \| grep -v '_test.go'`; per-package `cut -d/ -f1-2 \| sort \| uniq -c`; split greps | 18 files, counts as listed in Implementation Plan; 43+43+5+1 = 92 ✓; 69+23 = 92 ✓ |
| V10 | Go 1.26.6; `exec.Cmd.Err` exists (deferred-error contract available) | `head -5 go.mod`; `go doc os/exec.Cmd.Err` | `go 1.26.6`; `Err error  // LookPath error, if any.` |
| V11 | Variable-first-arg git exec shapes exist (enumerator blind spot is real) | `grep -n 'exec.CommandContext(ctx, git,' cmd/ailang/help.go` | Hits at help.go:204 and :222 (first arg is the variable `git`) |
| V12 | A `bash -c` exec site exists (shell-string blind spot is real); it is out of the git class | `grep -rn --include='*.go' -E '"(sh\|bash)", "-c"' cmd internal \| grep -v '_test.go'`; then read the site | Single hit `internal/eval_harness/watchdog.go:57`; site body is process-tree cleanup, no git invocation |
| V13 | ZERO git exec sites outside cmd/ and internal/ (negative + control) | root grep with `--exclude-dir={cmd,internal,vendor,node_modules,.git}`; control: same root grep including cmd+internal | exit=1 (none outside); control = **92** exactly (instrument sees the known positives through the identical invocation) |

**Not reproduced from the commissioning brief:** the brief estimated "roughly 95" sites; the
measured count is **92** (V2, cross-checked three ways: narrow regex, wide regex, root-wide
control). All other commissioned facts reproduced exactly.

## Related Documents

Searched via `grep -rli 'S4036\|LookPath\|exec.Command' design_docs/planned design_docs/implemented`
(plus a title scan of `design_docs/planned/v0_35_0/`). No existing doc covers git binary
resolution or an exec-hardening sweep — nearest neighbours, each distinct:

- `design_docs/implemented/v0_30_0/m-arch-boundaries.md` — origin of the
  `check_boundaries.sh` gate pattern this doc's CI gate imitates (fixed-set enumeration, loud
  setup-error exit). Distinct: layers vs exec sites. (Note: `scripts/check_boundaries.sh`'s
  own header cites a stale `v0_7_0` path for this doc; verified 2026-08-27 via
  `find design_docs -name '*arch-boundaries*'` — the implemented copy is the real one.)
- `design_docs/implemented/v0_33_1/m-ci-flake-systemic-fix.md` — prior art for "audit the
  whole class before patching the flagged instance" in CI. Distinct: flakes vs security class.
- `design_docs/implemented/v0_8_1/m-process-exec.md` — AILANG-language process effects, not
  Go-side exec plumbing. Distinct surface entirely.

## Timeline

| Day | Work |
|-----|------|
| 1–1.5 | M1: gitexec package, 4 flagged sites, help.go migration, tests+mutants, gate landed (baseline 88) |
| 2.5 | M2: coordinator sweep (baseline → 46) |
| 3.5 | M3: cmd/ailang sweep (baseline → 6) |
| 4 | M4: pkg + eval_harness sweep (baseline → 0), positive invariant on, CHANGELOG |
