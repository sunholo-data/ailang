# Iteration 64 evaluation: controlplane-assert-peek (bf3672af3 + 92a5420e2)

Judge: independent evaluator, detached worktree at 92a5420e2 (clean, 0 porcelain lines before and after; executor worktree untouched; no ailang messages command run).

## Claims
1. --peek exists and skips MarkInboxMessageRead on every path: VERIFIED (cmd/ailang/messages_crud.go runMessagesRead: all-unread L307, latest L332, single-id L368 each guard on !*peek). The flag pre-dates the sprint; no Go change was needed or made.
2. Flag-order trap: VERIFIED. fs.Parse stops at the first positional. /tmp Go program with flag.NewFlagSet: [--peek --json id] gives peek=true json=true args=[id]; [id --peek] gives peek=false json=false args=[id --peek]. The same file's runMessagesForward carries the same warning in a comment.
3. .agents mirror: VERIFIED. sync-agents-skills.sh --check rc=0; diff shows identical hunk in both trees.
4. New arm kills both mutations: VERIFIED for the stated two (see M2b, M3). But see Finding 1: the arm is weaker than claimed for other mutations.

--json: the store's InboxMessage serialises `payload` (json:"payload"), so the body is in --json output. Prose says WHY flags go first (Go parser stops at first positional, trailing --peek silently ignored, message marked read) and why peek (Daneel hourly unread triage). Clear.

## Gates (judge tree)
- bash -n test_agents_skills_sync.sh: rc=0
- /bin/bash sync-agents-skills.sh --check: rc=0
- /bin/bash test_agents_skills_sync.sh: rc=0 (4 passed, 0 failed)
- make check-changelog: rc=0 (5 fragments valid)
- bash scripts/check_context_docs.sh: rc=0

## Mutations (each run with /bin/bash test_agents_skills_sync.sh, restored via cp from saved copy + cmp; tree clean after)
Note: any edit to the .claude retro also reddens the mirror-equality arm; I attribute only the new "Gate-5 rule 3(c)" arm below.
- M2b retro (c) command replaced by prose (no --peek), intact test: rc=1, new arm FAIL with its diagnostic. Killed.
- M3 `read <id> --peek --json` (trailing): rc=1, new arm FAIL. Killed.
- M5 `read --json --peek <id>` (valid CLI, different spelling): rc=1, new arm FAIL. Spelling lock, as stated in plan; acceptable.
- M6 `read -peek -json <id>`: rc=1, new arm FAIL (also spelling lock).
- M2 delete `count == 1` from the test AND remove the command from (c): new arm PASSES (vacuous), only mirror arm red. So count==1 is load-bearing for the zero-command case. Good.
- M1 add a SECOND inline `ailang messages read <id>` in clause (c), on the line after the first (wrapped line): new arm PASSES (rc=1 only from mirror arm). SURVIVED. Also survived with a second *valid* spelling (M1b). See Finding 1.
- M4 stray earlier `**(c)** stray` line at top of file: new arm still passes (first marker absorbs everything up to the real (d)); not vacuous, not spurious here.

## Findings
Non-blocking 1 (real defect in the guard, rig-awk specific): BSD awk (/usr/bin/awk, version 20200816) split(s, a, "`") ALSO splits on newline: `split("a`b\nc`d",s,"`")` returns n=4 not 3. The arm's "even index = inline code" parity therefore inverts after every line break inside the clause. Verified by printing the spans on the real mutated file: the 2nd command landed at index 5 (odd) and was never inspected. Effect: a second, mark-read `ailang messages read <id>` inside (c) goes undetected unless it sits on the same physical line as the first command; the `count == 1` intent is only honoured for line 1. A reword that moves the command onto a later line could also fail or pass by parity luck. Fix: don't rely on backtick parity; e.g. replace newlines with spaces (gsub(/\n/," ",clause)) before split, or use grep -o '`ailang messages read[^`]*`' on the joined clause. Not blocking: the ticket's actual regression (unflagged read, trailing --peek) is caught, and the plan states the guard locks a spelling, not semantics.
Non-blocking 2: guard anchors on the FIRST `**(c)**` in the whole file, not on rule 3. Today the file has exactly one `**(c)**`/`**(d)**` pair (lines 74/78), so it is correct; a future earlier (c) marker would be absorbed (M4) or mis-scope silently. Stated limitation.
Non-blocking 3 (observation, same class, mission-control only): the other `messages read` mentions (gate-0-preflight.md:38, gate-2-pick.md:62) are prohibitions on reading mission-fleet, not instructions; SKILL.md:622 uses `list --unread`. No remaining instruction in mission-control tells the controller to mark-read something. Clean.
Observation: `--peek` pre-existed, so the sprint is docs + guard only; the work is proportionate.

## Score (100)
- Acceptance criteria / correctness of the fix: 28/30
- Tests/gates and mutation strength: 17/25 (arm kills the ticket's mutations; second-command mutation survives due to awk parity bug)
- Code/prose quality and clarity of WHY: 18/20
- Artifacts (plan, sprint JSON, changelog fragment, mirror sync): 15/15
- Scope discipline / risk: 8/10
Total: 86/100 (pass threshold 70). No hard-fail triggers.

Judge worktree removed with `git worktree remove --force` (see below).

EVALUATION_SCORE: 86/100
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: /Users/voightkampff/.ailang-driver-pin/fleet-i64-evaluation.md
FEEDBACK_SUMMARY: Fix correct and verified; awk guard misses a second inline read after a line break (BSD awk splits on newline in split()), non-blocking.
EVALUATION_VERDICT: PASS score=86

## Round 2 (commit 3f320eac4, fresh detached worktree, clean: 0 porcelain lines)
Diff vs 92a5420e2: only `gsub(/\n/, " ", clause)` + comment in tools/launchd/test_agents_skills_sync.sh. Mutations edit the retro (also reddens the mirror arm); attribution below is to the Gate-5 arm only. All restored with cp+cmp. No ailang messages command run.
- Clean tree: suite rc=0 (4 passed).
- S1 (my survivor) second bare `ailang messages read <id>` on the wrapped line: rc=1, Gate-5 arm FAIL. Now killed.
- S2 (my survivor) second valid-spelling `read --peek --json <id>` on the wrapped line: rc=1, Gate-5 arm FAIL. Now killed (count==1 now enforced beyond line 1).
- K1 no `--peek`: rc=1, Gate-5 arm FAIL. Still killed.
- K2 trailing `--peek`: rc=1, Gate-5 arm FAIL. Still killed.
- W the real command wrapped inside its backticks with the 5-space continuation indent: rc=1, Gate-5 arm FAIL (fold leaves multiple spaces, so exact match fails). Fail-closed: a loud spurious failure on a reflow, never a silent pass. Acceptable; the diagnostic names the expected command.
- W2 wrapped with no indent (single-space fold): Gate-5 arm passes (only mirror arm red). Correct behaviour.
- bash -n rc=0; sync-agents-skills.sh --check rc=0; test_agents_skills_sync.sh rc=0 on the restored tree; tree clean afterwards.
No regression found. Finding 1 is CLOSED. Findings 2 (first-marker anchoring) and 3 unchanged, non-blocking. Judge2 worktree removed.

Round 2 score: 91/100 (tests/mutation strength 17 -> 22).
EVALUATION_SCORE: 91/100
EVALUATION_ROUND: 2
EVALUATION_REPORT_PATH: /Users/voightkampff/.ailang-driver-pin/fleet-i64-evaluation.md
FEEDBACK_SUMMARY: Finding 1 closed; both survivors now killed, no regression.
EVALUATION_VERDICT: PASS score=91
