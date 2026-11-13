# Fleet iteration 111 — Gate 5 skill fix, independent judge

- **Change**: [#2336](https://github.com/sunholo-data/ailang/pull/2336), `.claude/skills/mission-control/resources/gate-5-retro.md` step 2b plus its `.agents` mirror and a changelog fragment. Squash-merged as `1a28768771dc42b2ce1e3ba80cfda5073be8b97f`.
- **Lane**: Gate 5 skill fix. Two recorded frictions on one gap: iteration 110 (index 1/20 against log headlines 7/20) and iteration 111 (index 1/20 against 6/20; the whole-file grep read 42).
- **Judge**: evaluator, Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground. It received the diff and the claimed defect, not the controller's verdict. 52,642 tok, 9 tool uses, 44 s.

## Verdict: PASS 86/100

Scores: correctness 37/40, prose 18/25, cross-mission safety 17/20, hygiene 14/15.

- **Defect reproduced.** The whole-file grep reads 42. The index is newest-first, so its last 20 iterations are its first 20 rows. Those rows hold 1 `[HARNESS]`, and 19 of 20 end in `...`. `indexLine` in `internal/mission/index.go` cuts titles over 150 runes. The old `tail -20` row count took the oldest rows.
- **New snippet** (`/bin/bash`):

  | Mission | Rows | HARNESS | Tagged |
  |---|---|---|---|
  | fleet | 20 | 6 | 20 |
  | v1 | 20 | 12 | 20 |
  | motoko | 20 | 12 | 12 |
  | triage | 20 | 14 | 20 |
  | docs | 17 | 2 | 2 |
  | nonexistent mission | 0 | 0 | 0 |

  The selected rows are in numeric order across the archive/live boundary, and BSD `sort -t' ' -k2,2n` orders 111 after 99.
- **Adversarial cases.**
  - A wrong CWD or a missing archive reads low rows. The prose's "rows below 20, check the log" covers both.
  - An entry number duplicated across archive and live log is double-counted. This is the one silent wrong answer, and it is unlikely because `rotate-log` moves entries losslessly. Noted, not fixed.
- **Mutation check.** On the same data the old form gives 42 (whole file) or 1 (last 20 rows), and the new form gives 6, which matches the log headlines.
- **Required change.** The prose said the index missed "three" HARNESS rows. It missed five: 93, 96, 97, 105 and 109; 95 was kept. The controller verified this first-party with a per-row grep (93:0 95:1 96:0 97:0 105:0 109:0). It corrected the prose in `ae4b99ee7`, which also notes the newest-first `tail -20` defect, and re-ran the mirror sync test (5/5) and `check_context_docs.sh`.

## Gate 3b

- PR head `ae4b99ee7e9f7132c807a21c3ee532e8157a0105`: 24 check-runs, 0 not green. Bounded poll: two 540 s chunks under `/bin/bash`, with the verdict naming its SHA.
- Merge commit `1a28768771dc42b2ce1e3ba80cfda5073be8b97f` on dev: 18 check-runs, 0 not green. Same poller.

## Reach

The running skill is the main checkout's working tree (`~/.claude/skills/mission-control` → `~/dev/sunholo-data/ailang`). At this fire it sat at `bb7194101`, so it does not carry the edit until that checkout next syncs to `origin/dev`. The pin worktrees get it on their next re-pin. The loop does not touch the shared main tree.
