# Iteration 87 — independent evaluation (sonnet, Agent tool, foreground)

The judge wrote a 4-line summary to this path and returned the full verdict as its final message; the controller saved that message here verbatim in substance (tables and findings unchanged).

**Verdict: PASS, 92/100** on `b4651a0d5` (base `f6a95d75d`).

| Category | Score |
|---|---|
| Correctness | 33/35 |
| Tests / mutation strength | 23/25 |
| Scope / safety | 19/20 |
| Docs / changelog | 9/10 |
| Portability | 8/10 |
| **Total** | **92/100** (33+23+19+9+8, checked) |

**Checked in the diff:** anchoring (`re.match` + `\A`, whitespace-free fire id, ASCII digits, literal `.`); self-author filter on the transcript branch; crash/transcript mutual exclusion (`elif`); windowing `>` same as crash (a notice exactly at the watermark is excluded); `other` = self − crash − transcript; rc 0 and rc 1 verdicts both carry the transcript suffix, transcript-only stays rc 0; F8 requires both the crash and the current transcript literal; privacy (only at/hits/url/fire + counts; failure labels print expected metadata, not bodies); bash 3.2 (nothing bash-4; fire id last in `read -r`, emitted via printf/echo, no glob/word-split risk).

**Blocking findings:** none.

**Non-blocking:**
1. Step 6a prose: accurate, leaks nothing, a little wordy. The rc 0/rc 1 meanings in the paragraph above it are still crash-only, so a reader could miss that a transcript notice does not change them (the new paragraph's last sentence does say so).
2. The suite header's arm count was dropped rather than updated.
3. `transcript_arms` prints a space-joined string relying on unquoted word-splitting — works, fragile.
4. `match` → `search` is an equivalent mutant (`\A` already anchors); not a test gap.

**Judge's own mutants** (scratch copies via `G0_SCRIPT_UT`, full suite, cmp-differs confirmed):

| Mutant | Killed by |
|---|---|
| `match` → `search` | SURVIVED (equivalent) |
| window `>` → `>=` (transcript) | `transcript-window` |
| `other` drops transcript term | `transcript-current`, `transcript-ascii`, `transcript-window`, `transcript-other` |
| drop self filter on transcript branch | `transcript-foreign`, `transcript-other` |
| `[0-9]+` → `[0-9]*` | `transcript-ascii` |
| in-window count increments out of window | `transcript-window` |
| F8 transcript-literal check disabled | `transcript-f8-current-missing` |
| `\S+` → `.+?` fire id | `transcript-header-shape` |
| drop literal `.` after `hit(s)` | `transcript-header-shape` |
| drop rc-1 transcript suffix | `report B`, `transcript-verdict-crash` |

An eleventh mutant failed to apply (search string mismatch) and tested nothing. The judge did not separately spot-check the executor's named mutants (asked for 2); its ten overlap the executor's areas.

**Strongest objection:** regex tightness is covered only by executor-written fixtures; no test replays a verbatim live comment body. The live repro covers it (head on #2039 → 1 notice, fire `fleet-1793845378-59113`) but is not a regression test.

**Commands (rc):** suite under /bin/bash 168/0 (rc not captured — PIPESTATUS slip); `bash -n` ×3 0; `git diff --check` 0; `sync-agents-skills.sh --check` 0; live read-only #2039/#1945 rc 0; 10 mutant runs; judge worktree removed rc 0. Not rerun by the judge: shellcheck, check-changelog/check-file-sizes, the Go quorum shell test (the controller ran those).
