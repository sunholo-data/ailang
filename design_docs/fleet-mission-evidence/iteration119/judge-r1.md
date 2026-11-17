# Judge evaluation, iteration 119 (commit 1feb66036)
Score 90/100, PASS.
Correctness 33/35, tests 20/25, scope 15/15, landing 13/15, docs 9/10.

## Findings
1. Defect real at base: ownFiles holds `${cwd}/${p}`; porcelain is repo-relative; unownedDirty compared them with a bare Set.has. The test's HEAD-defect assertion reproduces it (own paths reported unowned).
2. Fix verified end-to-end with real git (not fakes), 7 scenarios, tracked own file + other dirty files:
   root-relative path, root-absolute path, cwd in a subdirectory (porcelain is root-relative, so this works), `../sub/mine.ts`, `./sub/mine.ts` and symlinked cwd. All named only the genuinely foreign files. Non-repo cwd is silent.
   Own edit is correctly excluded in every case.
3. node --test .unowned-dirty.test.ts: pass 6, fail 0.
4. Scope: diff touches only .pi/extensions (2 files), the changelog fragment and iteration119 evidence. Nothing under cmd/. Conforms to D-FLEET-33 A.
5. CI: `.github/workflows/ci.yml:426` runs `make verify-pi-assets` in the `checks` job; Makefile:368-370 does `diff -rq -x '.*' .pi/extensions cmd/ailang/pi_assets`. I diffed the two unowned-dirty.ts files: they differ (the new import and helpers), so `checks` WILL go red on this branch until `make pi-assets` is run. The plan is accurate: do not push direct to dev, open a PR, attended half syncs. No CI job runs the .pi/extensions node tests (the grep found no `test-pi-extensions` call in the workflow). The plan says it runs via `make test`, which I did not verify.
6. Changelog fragment is accurate and discloses the unsynced embed and the `ailang pi setup` consequence.
7. Eval impact confirmed: pi 0.85.1 at /opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent; print-mode.js bindExtensions passes no uiContext, runner.js:270 falls to noOpUIContext whose notify is `() => {}` (line 92); internal/executor/pi/args.go:38 invokes `--mode json`. The warning is invisible in evals, so the only cost is one extra `git rev-parse` exec per sweeping git op.

## Mutants (my own, 15, run on scratch copies)
| id | change | result |
|---|---|---|
| m1 | drop slice(prefix.length) | killed (3 fail) |
| m2 | realOrSelf = identity | killed |
| m3 | handler passes ownFiles (HEAD defect) | killed |
| m4 | no trim of toplevel | killed |
| m5 | code !== 0 -> === 0 | killed |
| m6 | ignore trailing slash (always add /) | killed |
| m7 | handler does not realpath ownFiles | SURVIVED, real gap: symlink only tested at pure-helper level; handler test uses a real tmp cwd |
| m8 | handler does not realpath top | SURVIVED, near-equivalent: git prints a physical toplevel already |
| m9 | relativeTo no startsWith filter | killed |
| m10 | remove non-repo early return | SURVIVED, equivalent in output: git status fails with empty stdout, so no notify; only an extra exec. Non-repo path untested |
| m11/m12/m13 | absolute/relative own-path spelling in handler | killed |
| m14 | realOrSelf without try/catch | killed (nonexistent file) |
| m15 | no trailing-slash handling | killed |

## Non-blocking findings
- Untracked directories collapse in porcelain (`?? sub/`), so an own newly written file inside a wholly-untracked directory is still reported as `sub/` (pre-existing; not addressed, not claimed fixed). Worth a line in the plan's Risks.
- m7 gap: add a handler test with cwd reached via a symlink.
- Non-repo branch (top.code !== 0) has no test.
- The pre-existing `/^(internal\/ai\/)?ollama/` filter comment says "placeholder-free"; untouched, out of scope.
- Claude-port relativeTo does not realpath (acknowledged out of scope).

## Strongest objection
The symlink fix (the reason realOrSelf exists) is only proven by a pure-helper test; a handler regression that drops the realpath mapping of ownFiles (m7) passes all 6 tests. Not blocking: I confirmed the behaviour manually with a real symlinked cwd.

No blocking findings. Landing must be via PR with the attended `make pi-assets` sync before merge.

## Delta review r2 (27ba6e487 on 1feb66036)
- Source unchanged: `unowned-dirty.ts` is byte-identical to 1feb66036 (cmp). Delta is the test file (+43) and executor-r2.md only.
- `node --test`: tests 8, pass 8, fail 0 (my count 6 + 2).
- Re-run of my mutants on scratch copies: m7 (no realpath of ownFiles in handler) KILLED; m10 (remove non-repo early return) KILLED; m8 (no realpath of top) still survives, equivalent in practice since git prints a physical toplevel.
- Soundness: the symlink test sets cwd to a symlink and the fake git reports the physical toplevel; without realpath the own path `link/mine.ts` is not under the toplevel, so mine.ts would be reported too and the assertion on exactly `theirs.go` fails (that is exactly what m7 shows), so it is not vacuous. The non-repo test asserts `calls == ["rev-parse"]` and no notice; with the early return removed `git status` is called and theirs.go is reported, so it asserts something the handler guarantees (m10 kills it). Both use fake exec, so they test handler wiring, not real git; my earlier real-git run covers that.
- Verdict holds: PASS. Score 93/100 (tests 23/25). Remaining non-blocking: untracked-directory collapse in porcelain; m8 equivalent. Landing unchanged: PR plus attended `make pi-assets` before merge.
