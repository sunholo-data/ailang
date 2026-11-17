# Iteration 119 plan: pi unowned-dirty compares absolute own paths with relative porcelain paths

Ruling D-FLEET-33 A: the fleet lands the `.pi/extensions/**` diff. The `cmd/ailang/pi_assets` sync goes to V1 or an attended session.
Ticket: `pi:unowned-dirty-absolute-vs-relative-paths`. Base: `022f68f1a` on branch `fleet/i119-unowned-dirty`.
Worktree `B=/Users/voightkampff/.ailang-driver-pin/.fleet-i119-build`. Use absolute paths in every command.

## Planner findings (measured 2026-11-17, node v26.13.0, pi 0.85.1)
- `.pi/extensions/unowned-dirty.ts` has not changed since `0e8351a89` (2026-08-28). Lines :42 and :78 are still absolute. The handler code the diff touches (:47-:52) matches the diff context.
- `/tmp/fleet_i119_proposed.diff` is byte-identical to the banked `design_docs/fleet-mission-evidence/iteration66/unowned-dirty-proposed.diff` (sha256 `8a6dbb3c…e2e9df`). If /tmp has been cleaned, use the banked copy.
- **Gap in the iteration-66 diff:** its two "wiring" tests only exercise the pure helpers. Reverting the fix at the handler call site (M3), dropping `.trim()` (M4) or inverting the early return (M5) leaves those tests 5/5 green. This plan adds a handler-level test (`/tmp/fleet_i119_handler_test.diff`, sha256 `d0eefb88…a99a8f`) that kills all three mutants.
- No CI job runs the `.pi/extensions` node tests. CI `test` runs `go test ./...` directly, not `make test`. `test-windows` runs no node. `make test-pi-extensions` (local only, a dependency of `make test`) runs them.
- `make verify-pi-assets` passes at HEAD and fails after this change. Measured on a scratch copy: `Files .pi/extensions/unowned-dirty.ts and cmd/ailang/pi_assets/unowned-dirty.ts differ`. Dotfile tests are excluded (`-x '.*'`), so only the `.ts` file drifts. `unowned-dirty.ts` is not one of the `internal/executor/pi/profile_assets` files.
- Eval claim CONFIRMED: pi 0.85.1 `print-mode.js` binds extensions with no uiContext, so `runner.js` uses `noOpUIContext`, where `notify: () => {}` is a no-op. Under `--mode json` the warning is invisible. The fix only adds one `git rev-parse` exec per sweeping git op.
- Claude Code port: `tools/claude-mods/unowned-dirty/hooks/register.ts:13` `relativeTo` has the same contract (prefix strip, `Set` result). It does NOT realpath, so the symlink part of the fix is pi-only. Out of scope.

## Step 1: Re-measure the defect at HEAD (record command and output in the report)
```bash
cd /Users/voightkampff/.ailang-driver-pin/.fleet-i119-build/.pi/extensions && node -e '
import("./unowned-dirty.ts").then(({parsePorcelain, unownedDirty}) => {
  const cwd = "/repo", p = "mine.ts";
  const own = new Set([p.startsWith("/") ? p : `${cwd}/${p}`]); // exactly :42/:78
  console.log("own=", [...own], "unowned=", JSON.stringify(unownedDirty(parsePorcelain(" M mine.ts\n M theirs.go\n"), own)));
})'
```
Expected (planner run): `own= [ '/repo/mine.ts' ] unowned= ["mine.ts","theirs.go"]`. `mine.ts` is the session's own edit and is wrongly reported.

## Step 2: Apply the changes (mechanical; no hand edits)
```bash
B=/Users/voightkampff/.ailang-driver-pin/.fleet-i119-build
git -C $B apply --check /tmp/fleet_i119_proposed.diff && git -C $B apply /tmp/fleet_i119_proposed.diff
git -C $B apply --check /tmp/fleet_i119_handler_test.diff && git -C $B apply /tmp/fleet_i119_handler_test.diff
git -C $B status --short   # expect exactly: M .pi/extensions/.unowned-dirty.test.ts, M .pi/extensions/unowned-dirty.ts
```
What the diffs do:
- `unowned-dirty.ts`: imports `realpathSync`. Adds exported `relativeTo(top, paths)`, which keeps the paths under `top/` and strips that prefix, and `realOrSelf(p)`, which is realpath with a fallback to the input. In the `tool_call` handler it runs `git rev-parse --show-toplevel` before `git status`, returns early if `top.code !== 0`, and compares against `relativeTo(realOrSelf(top.trim()), [...ownFiles].map(realOrSelf))`. Lines :42 and :78 stay absolute by design: `ownFiles` stays absolute and is relativized at comparison time.
- `.unowned-dirty.test.ts`: adds 2 pure wiring tests (iteration 66) and 1 handler test (new). The handler test uses a fake `pi` (`on`, `exec`) and a fake `ctx` (`cwd`, `ui.notify`). It sends an `edit` with relative path `mine.ts`, then `git add -A`, and asserts one notify naming only `theirs.go`.

If `/tmp/fleet_i119_handler_test.diff` is missing, make two edits to `.unowned-dirty.test.ts` instead. First, change the import line to `import unownedDirtyExtension, { parsePorcelain, unownedDirty, isSweepingGitOp, relativeTo, realOrSelf } from "./unowned-dirty.ts";`. Second, append:
```ts
test("handler: a relative edit/write path is not reported unowned on a sweeping git op", async () => {
	const top = realOrSelf(mkdtempSync(join(tmpdir(), "ud-h-")));
	const handlers: Record<string, Function> = {};
	const notes: string[] = [];
	const fakePi = {
		on: (name: string, fn: Function) => { handlers[name] = fn; },
		exec: async (_cmd: string, args: string[]) =>
			args[0] === "rev-parse"
				? { code: 0, stdout: `${top}\n` } // git prints a trailing newline
				: { code: 0, stdout: " M mine.ts\n M theirs.go\n" },
	};
	const ctx = { cwd: top, ui: { notify: async (msg: string) => { notes.push(msg); } } };
	unownedDirtyExtension(fakePi as never);
	await handlers.tool_call({ toolName: "edit", input: { path: "mine.ts" } }, ctx);
	await handlers.tool_call({ toolName: "bash", input: { command: "git add -A" } }, ctx);
	assert.equal(notes.length, 1);
	assert.match(notes[0], /may sweep 1 dirty file\(s\)[^:]*: theirs\.go\./);
});
```

## Step 3: Changelog fragment `changelogs/unreleased/2026-11-17-pi-unowned-dirty-relative-paths.md`
```markdown
### Fixed — pi `unowned-dirty` reported the session's own edits as unowned (2026-11-17)

The extension recorded edit/write paths as absolute (`${ctx.cwd}/${p}`) but compared them with the
repo-relative paths `git status --porcelain` prints, so every own edit was named in the "may sweep N
dirty file(s) this session did not write" warning. The handler now resolves `git rev-parse
--show-toplevel` and relativizes own paths against it (`relativeTo`), resolving symlinks first
(`realOrSelf`, for a cwd under macOS `/var` → `/private/var`). Adds a handler-level test. Same
contract as the Claude Code port's `relativeTo` (tools/claude-mods/unowned-dirty).
Ticket `pi:unowned-dirty-absolute-vs-relative-paths`, ruling D-FLEET-33 A.

**Not yet synced:** the embedded copy `cmd/ailang/pi_assets/unowned-dirty.ts` is NOT updated in this
commit: the fleet's push scope excludes `cmd/ailang/pi_assets/**`. An attended session must run
`make pi-assets` and commit the result before merge, or `make verify-pi-assets` (CI `checks`) fails.
Until then `ailang pi setup` installs the old extension.
```

## Step 4: Gates (run each unpiped, record the exact output)
| # | Command | Expected |
|---|---------|----------|
| G1 | `cd $B/.pi/extensions && node --test .unowned-dirty.test.ts` | `ℹ pass 6`, `ℹ fail 0` (was 3/3 at HEAD) |
| G2 | `make -C $B test-pi-extensions` | rc=0. Every file `fail 0`; unowned-dirty row now `pass 6` (HEAD baseline: 17 files, all fail 0) |
| G3 | `git -C $B apply --check -R --include='.pi/extensions/unowned-dirty.ts' /tmp/fleet_i119_proposed.diff && git -C $B apply --check -R /tmp/fleet_i119_handler_test.diff` | both rc=0 (the source is exactly the reviewed diff). A bare `-R` of the full proposed diff FAILS by design, because the handler patch rewrites its import line |
| G4 | `make -C $B check-changelog` | rc=0 (fragment accepted) |
| G5 | `make -C $B verify-pi-assets; echo rc=$?` | **EXPECTED FAIL**: `DRIFT: cmd/ailang/pi_assets is stale…`, rc=2. Record it. This is the attended half, not an executor defect. Do NOT run `make pi-assets`. |
| G6 | `git -C $B diff --name-only` | exactly the 2 `.pi/extensions` files and the changelog fragment (plus this plan). Nothing under `cmd/` |

## Step 5: Mutation drill (scratch copies only; never `git checkout <file>` in $B)
Run `/bin/bash /tmp/fleet_i119_mutants.sh $B` after Step 2. It copies the two files to `/tmp/i119exec/<mutant>/`, mutates with perl, and runs node. Each mutant must show `fail ≥1` with the named test red. If a mutant reports `MUTATION DID NOT APPLY` or `fail 0`, that is a gate failure.
| Mutant | Change | Must go red (planner-measured) |
|---|---|---|
| M1 (iter 66) | `relativeTo`: drop `.slice(prefix.length)` | both wiring tests + handler (fail 3) |
| M2 (iter 66) | `realOrSelf` returns `p` (identity) | wiring: symlinked cwd (fail 1) |
| M3 (derived) | handler passes `ownFiles` instead of `relativeTo(...)`, i.e. the HEAD defect | handler (fail 1). Survives the iter-66 tests alone |
| M4 (derived) | `top.stdout` not `.trim()`ed | handler (fail 1). Survives the iter-66 tests alone |
| M5 (derived) | `top.code !== 0` → `=== 0` | handler (fail 1). Survives the iter-66 tests alone |
| M6 (derived) | prefix always `${top}/` (ignore trailing slash) | wiring: absolute own paths (`/repo/` row) (fail 1) |

## Step 6: Landing (facts for the controller; the controller decides)
- Do NOT push this to `dev` directly. With the fleet half alone, the required `checks` job goes red at `make verify-pi-assets`, and direct pushes skip PR CI.
- Recommended sequence:
  1. The executor commits on `fleet/i119-unowned-dirty`: the 2 `.pi/extensions` files, the fragment, and this plan.
  2. The fleet pushes the branch and opens a PR against `dev` titled "fix(pi): unowned-dirty compares own paths repo-relative (D-FLEET-33 A)". The body says `checks` is EXPECTED red on verify-pi-assets until the attended half lands. Do not merge.
  3. An attended session (or V1) checks out that branch, runs `make pi-assets` and confirms `git diff --name-only` shows only `cmd/ailang/pi_assets/unowned-dirty.ts`. It commits, pushes to the same branch, waits for `checks` to go green, then merges.
- Alternative: the attended session does both halves in one commit on dev. That breaks the ruling's split; only choose it if a PR is unwanted.
- The controller reports: G1 to G6 results with the G5 red recorded as expected, the M1 to M6 table, the PR URL, and an explicit handoff line: "attended: run `make pi-assets` on <branch>, push, merge".

## Risks
- Windows CI: irrelevant. No CI job runs these node tests, including `test-windows`. `realpathSync` on Windows is untested.
- macOS symlinked tmpdir (`/var` → `/private/var`): the reason `realOrSelf` exists. The tests realpath their own tmp base so they are stable on Linux and macOS.
- Deleted own files: `realOrSelf` falls back to the spelled path. Under a symlinked cwd, a deleted own file can still be reported unowned. This is a residual false positive; warnings only, never a block.
- Pre-existing and not fixed here: renames (`R  a -> b` parses as the path `a -> b`), quoted paths with escapes, and bash-written files (unknowable by design). The extra `git rev-parse` adds one exec per sweeping git op. A non-repo cwd now returns silently instead of running `git status`.
- Install drift: until the attended sync lands, `ailang pi setup` users get the old (buggy) embedded copy. Repo-local pi sessions load `.pi/extensions` and get the fix.
