# TEST SMOKE — M-AILANG-LANE-EXECUTOR-SHADOW M7 (ailang-only lane)

**Task:** create `lane-reports/ailang-only/test-smoke.md` containing one line
`test smoke <today>: ailang-only-executor`, commit, open a PR to `dev`, report the PR URL.

**Status:** BLOCKED at the commit step by the agent policy's process allowlist — details below.

## Deliverable

`lane-reports/ailang-only/test-smoke.md` exists and contains exactly:

```
test smoke 2026-10-20: ailang-only-executor
```

(today's date was obtained by running `git log --max-count=1 --format=%cd` — allowed by the
policy — rather than `Clock`, which the policy does not admit.)

## What blocks commit + PR

The agent policy (`/home/ailang/.ailang/agent-policy/agent-policy.toml`) sets
`process_allow = ["git:status", "git:diff", "git:log"]`. Attempts to run
`git add …`, `git commit …` or any other write-side subcommand return:

```
Err(NotAllowed(git add))
Err(NotAllowed(git commit))
```

so the file cannot be committed or pushed from inside this lane. The
worktree branch is `coordinator/task-b0237a7a` (grafted shallow checkout of
`dev` @ `f0f7df2046a730dff350680fe558dcd4a288c69a`); the deliverable sits
untracked in it, ready for the operator/harness to stage, commit and open the
PR to `dev` from outside the sandbox. No PR URL is reported because no PR
could be opened from this lane.

## Notable std / tooling observations (the point of this lane)

1. **`Clock` effect is not admitted by the agent policy**, so there is no way
   to get "today's date" from inside a policy-restricted AILANG program; the
   workaround used `exec("git", ["log", …])`. A read-only wall-clock lookup
   (e.g. `date_utc() -> string ! {}`) would remove the need to shell out for
   what is trivially available in the host. **Category: stdlib function.**

2. **`exec` returns `Result[ProcessOutput, ProcessError]`, but there is no
   safe way to read the error payload** — the only accessors exported by
   `std/result` are `unwrap` (panics via a division-by-zero guard in
   `std/result.ail`) and `isOk`/`isErr`; pattern matching on `Err(e)` does
   work but the error's `show(e)` is ADT-shaped with no field accessors
   (`NotAllowed(cmd)` — the message string is not reachable without
   unwrapping a variant). A `unwrapOr(res, default)` / `errorOf(res)` helper
   would make error reporting in scripts far less footgunny.
   **Category: stdlib function.**

3. **Module-name↔file-path hygiene is hostile to scratch probes.** A file
   `probe.ail` at `lane-reports/ailang-only/` cannot declare `module
   lane_reports/ailang_only/probe` (MOD010 wants the path to match the file
   name minus directory) *nor* `module probe` (same error, mirrored); the
   only workaround is to keep the module name equal to the bare file name and
   run from that directory. Hyphenated file names (`tmp-status.ail`) cannot
   get a matching module name at all (`PAR_HYPHEN_IN_MODULE`). **Category:
   core language change** (either relax MOD010 for scratch dirs, or add a
   per-file `--relax-modules` escape that does not require an env var).

4. **`git` write operations are impossible under the policy allowlist**
   (`git:status`, `git:diff`, `git:log` only). For a lane whose whole job is
   "write a report file, commit it, open a PR", the policy needs either
   `git:add`, `git:commit`, `git:push` on an allowlist of its own, or a
   dedicated extension package that wraps commit-and-PR as a single
   capability-gated operation. **Category: extension package** (the default:
   most gaps are).