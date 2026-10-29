# Iteration 66 — `pi:unowned-dirty-absolute-vs-relative-paths`: measured, parked per D-FLEET-3

Measured at `c0458fdfd` (2026-10-28) by the controller, first-party.

## The defect is real at HEAD

`.pi/extensions/unowned-dirty.ts:42` and `:78` record the session's own files as ABSOLUTE paths
(`p.startsWith("/") ? p : \`${ctx.cwd}/${p}\``). `unownedDirty()` (`:24`) compares them against
`parsePorcelain(git status --porcelain)`, whose paths are repo-RELATIVE. No absolute path equals a
relative one, so the session's own edits are always reported as unowned.

Reproduction (node v26.12.1, `--experimental-strip-types`, importing the HEAD pure functions; the
wiring's `add()` copied verbatim):

```
own = {"/repo/mine.ts"}; porcelain = " M mine.ts\n M theirs.go\n"
HEAD wiring  : ["mine.ts","theirs.go"]     <- own edit reported as "did not write"
relativized  : ["theirs.go"]
```

The existing test (`.pi/extensions/.unowned-dirty.test.ts`) feeds relative own-paths
(`new Set(["mine.ts"])`) to the pure function, so the wiring bug is untested.

## Proposed diff (not applied): [`unowned-dirty-proposed.diff`](unowned-dirty-proposed.diff)

- Add `relativeTo(top, paths)`, the same contract as the Claude Code port's
  `tools/claude-mods/unowned-dirty/hooks/register.ts:13`.
- Before `git status`, run `git rev-parse --show-toplevel`; if it fails, return (no repo, nothing to sweep).
- Compare `unownedDirty(all, relativeTo(top, ownFiles))`.
- Resolve symlinks on both sides (`realOrSelf`, `fs.realpathSync` with a fall-back to the
  spelling for a deleted file) before comparing. Added after the independent judge reproduced that
  `git rev-parse --show-toplevel` returns the PHYSICAL path while a cwd under a symlink
  (macOS `/var` → `/private/var`) spells own paths through the link, so `startsWith` fails and
  every own file is silently dropped — the original symptom, back with no failing test.
- Two wiring tests: absolute own paths are reported unowned without `relativeTo` (the HEAD
  defect) and excluded with it; an own path spelled through a symlinked directory matches the
  physical toplevel only after `realOrSelf`.

Controller run of the patched copy (the diff applied with `patch -p1` to a copy of HEAD's two
files): `node --test` 5/5 pass. Mutants: drop the prefix strip in `relativeTo` → 3/1 (round-1
diff); make `realOrSelf` the identity → 4/1. Each is killed by one new test. Still untested: the
`git rev-parse` call inside the handler itself (the judge's note; the handler needs a pi runtime).

**A second out-of-scope step is needed:** `cmd/ailang/pi_assets/unowned-dirty.ts` is a synced copy
(`make pi-assets`; `cmp` identical at HEAD) that `ailang pi install` embeds. `cmd/ailang/**` outside
`mission*` is refused by the fleet scope guard. After the `.pi/extensions` fix, the sync has to be
run from an attended session or by V1.

## Expected eval impact: none on scores

- Every pi eval and mission run uses `pi --mode json` (`internal/executor/pi/pi.go:180`,
  `scripts/mission_pi_run.sh:302`).
- In pi 0.85.1, print/json mode binds extensions with no UI context
  (`dist/modes/print-mode.js:53`), so `ctx.ui` is `noOpUIContext`, whose `notify` is `() => { }`
  (`dist/core/extensions/runner.js:88-92`). The warning this extension emits never reaches the
  model or the transcript in an eval.
- The only eval-visible side effect is one extra `git rev-parse --show-toplevel` subprocess
  (10 s cap) before the existing `git status --porcelain` on a sweeping git command. That command
  runs only when the model issues `git add|stash|checkout|restore|reset`.
- Interactive pi sessions are the only consumer of the fix: the warning stops naming the
  session's own files.

## Why it parks

D-FLEET-3: any change to `.pi/extensions/**` parks for Mark with the diff and expected eval impact,
because the pi eval harness loads the same extensions. The asset sync step above is also outside
fleet authority.

## Independent judge (sonnet, iteration 66)

Verdict PARTIAL: claims (i) defect, (ii) diff applies and its test kills a mutant, (iii) no eval
score impact (evals run `pi --mode json`, `args.go:38`, `mission_pi_run.sh:302`; the extension is
installed globally under `~/.pi/agent/extensions`, so it does load in evals, and its only effect
there is the extra subprocess) and (iv) the `pi_assets` copy, all CONFIRMED. Its strongest
objection was the symlink/realpath gap, which the diff above now covers.
