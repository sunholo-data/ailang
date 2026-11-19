VERDICT: PASS
SCORE: 90/100  (correctness/defect fix 38/40, tests+mutation 22/25, scope/policy 14/15, integration/guards 10/10, docs/changelog 6/10)

HEAD d2ee7dc03; merge-base with origin/dev 2f324920c.

Q1 REAL at base, gone at head. Real-git repro (/tmp/q1/run.ts: fake pi whose exec runs real git in a repo with mine.ts and theirs.go dirty; session edits mine.ts by absolute path, then `git add -A`):
 BASE (git show 2f324920c:.pi/extensions/unowned-dirty.ts): "may sweep 2 dirty file(s) ...: mine.ts, theirs.go" (own edit falsely reported).
 HEAD: "may sweep 1 dirty file(s) ...: theirs.go". Positive control: theirs.go is still reported at head, so the warning still fires for genuinely unowned files.

Q2 Runner. No CI/Makefile/script runs .pi/extensions tests (grep of Makefile, .github, scripts found nothing). Ran directly: `cd .pi/extensions && node --test .unowned-dirty.test.ts` (node on PATH) rc=0, tests 8, pass 8, fail 0. NON-BLOCKING: these tests are not gated by CI; they only pass because someone runs them.

Q3 Mutations (backup /tmp/ud.bak, restored each time; final run 8/8 green):
 M1 revert core (compare raw ownFiles, no relativeTo): KILLED by 2 handler tests (relative edit path; symlinked cwd). pass 6 fail 2.
 M2 realOrSelf returns p (no realpath): KILLED by 2 symlink tests.
 M3 drop `top.code !== 0` guard: KILLED by "outside a git repo" test.
 M4 relativeTo prefix without trailing "/": KILLED by 4 tests.
 M5 realOrSelf catch returns "": KILLED by 1 test.
 None survived. Caveat: tests use a faked pi.exec (not real git); my real-git repro above covers that.

Q4 cmp .pi/extensions/unowned-dirty.ts cmd/ailang/pi_assets/unowned-dirty.ts: rc=0 identical. Guard: `make verify-pi-assets` (diff -rq -x '.*' + cmp of 3 profile_assets files) rc=0. The 3 profile_assets files are not touched by this PR. The embed is `//go:embed all:pi_assets` in cmd/ailang/pi_setup.go: `go build ./cmd/ailang` ok, `go vet ./cmd/ailang` clean. Embedded copy only changes what `ailang pi setup` installs.

Q5 PR net diff vs merge-base: 8 files, exactly the 3 code files (test, ext, pi_assets copy), changelog, 4 evidence docs. The dev merge (5b22cfb16) added nothing to the PR's own paths beyond a clean merge. d2ee7dc03 touches only pi_assets/unowned-dirty.ts.

Q6 Scope. Everything is within ruling A plus the attended pi_assets sync it names. Evidence docs under design_docs/fleet-mission-evidence/iteration119 are normal fleet bookkeeping. Changelog fragment exists (changelogs/unreleased/2026-11-17-pi-unowned-dirty-relative-paths.md). NON-BLOCKING defect: it still says "Not yet synced ... Until then `ailang pi setup` installs the old extension", now false at this head.

Q7 Blast radius. Behaviour change is limited to the tool_call handler on sweeping git ops (add/stash/checkout/restore/reset): one extra `git rev-parse --show-toplevel` subprocess, and early return (no warning) when not in a repo. Per D-FLEET-33, pi --mode json binds a no-op UI context, so the warning never reaches the model (I did not re-verify the pi runner source; UNMEASURED by me). No change for non-sweeping commands. Mild residual risk: node:fs import added; imports of the extension elsewhere unaffected.

Strongest objection: the extension tests are not wired into any CI gate and use a mocked exec, so regression protection depends on manual runs. Also the changelog fragment is stale.

Blocking findings: none.
Non-blocking: stale changelog "Not yet synced" paragraph; tests not CI-gated; PR title says "needs attended sync", now done.

Worktree git status --porcelain: empty.
