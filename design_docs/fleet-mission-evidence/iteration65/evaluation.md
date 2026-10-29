# Evaluation: fleet iteration 65, planner-lane fence parsing (head c230c7555, base 58a41bad4)

Verdict: PASS 74 (marginal). Two fail-open regressions remain that the change introduced. Both are cheap to fix.

## 1. Reproduction (first-party)
- Routing suite at head, run under `env -i`: **115 passed, 0 failed**. The executor's number reproduces.
- I ran base (`git show 58a41bad4:...` into a scratch file) and head side by side on q..w:

| Fixture | Base | Head |
|---|---|---|
| q | opus path-not-in-allowlist (false deny) | codex ok |
| r | **codex ok (fail-open)** | opus path-not-in-allowlist |
| s | opus no-files-section | codex ok |
| t, u | opus path-not-in-allowlist | codex ok |
| v | opus path-not-in-allowlist | codex ok |
| w | codex ok | opus unparsable-path-entry |

- The fail-open fix (r) is confirmed.
- The 14 pre-existing fixtures a..p give byte-identical output at base and head (stdout+stderr compared).
- `make check-changelog` is green.
- Only awk available here is BSD 20200816. gawk and mawk are not installed, so cross-awk behaviour is untested. `match($0,/^ */)` with an empty match gives RSTART=1, RLENGTH=0 on BSD awk. POSIX requires the same, so gawk and mawk should agree. I could not confirm.

## 2. Fail-open hunt (about 27 shapes run at base and head)
Cases with the disallowed `internal/...` bullet placed after the fence. "Fail-open" means head routes to codex.

**New fail-open regressions (base was opus, head is codex):**
1. **CRLF line endings plus any fenced block (BLOCKING-ish).** `d2/a_crlf_plain.md`:
   ```
   ## Files\r\n- `tools/launchd/x.sh`\r\n```bash\r\necho hi\r\n```\r\n- `internal/parser/parser.go`\r\n
   ```
   - Base: `opus fail-closed:path-not-in-codex-allowlist`. Head: `codex:gpt-6.1-sol declared:codex-ok`.
   - Cause: the closer test `fence_tail ~ /^[ \t]*$/` does not accept the trailing `\r`. A closing fence is never recognised, so everything after it is swallowed.
   - The rest of the script tolerates CRLF: the Planner-Lane value is stripped with `[[:space:]]`, and the no-fence CRLF doc `b_crlf_nofence` is correctly denied. So CRLF docs do reach this code.
   - Fix: strip `\r` at the top of each awk record, or use `/^[ \t\r]*$/`.
2. **Unterminated fence after a valid bullet (design-intended, see fixture v and plan line 67).**
   - `- tools/launchd/x.sh` followed by ```` ``` ```` and then `- internal/p.go` with no closer: base opus, head **codex**.
   - Same for `~~~` (`d2/h_tilde_unterminated`).
   - It matches CommonMark, but a forgotten closer is the commonest authoring slip. It silently turns every later real bullet into "code" and grants codex.
   - The fail-closed alternative is a fence still open at EOF giving `opus fail-closed:unparsable-path-entry`. This is a one-line END block.
   - The repo's own principle is to fail closed, and the plan chose the opposite and encoded it in test v.

**Shapes that behaved correctly (head = opus or the intended result):**
- Closer with trailing text (```` ``` text ````) does not close, so the later bullet is checked and denied.
- Trailing-space closer closes.
- Backtick fence whose info string contains a backtick is not a fence (denied).
- Tilde fence whose info string contains a backtick is a fence.
- Adjacent fences, fence inside a blockquote, a fence indented 2 or 3 spaces, and an indent-4 or tab-indented fence with an indented body are all handled sanely.
- A fence opened before the Files heading and never closed: head `no-files-section` (base was codex, so this is an improvement).
- A fenced second `## Files` heading is not counted.
- A bare `---` or `# comment` inside a fence no longer terminates the section.

**Residual and unchanged:**
- An indent-4 or tab fence whose body sits at column 0 is not recognised as a fence. The `# c` line then terminates the section and `internal/p.go` goes unchecked, giving codex at both base and head. Under CommonMark that doc is itself ill-formed (the `# c` is a real heading), so I count it as non-blocking.

## 3. Mutants
I mutated the script in place in the judge worktree and restored it with cmp after each run. All mutations are applied to both awk scans.

| Mutant | Result |
|---|---|
| M5: closer length check removed | KILLED (u) |
| M1: closer trailing-text check removed | **SURVIVES** (115/0) |
| M2: indent<=3 limit removed | **SURVIVES** |
| M3: tilde/backtick info-string rule removed | **SURVIVES** |
| M4: min fence length 3 lowered to 2 | **SURVIVES** |

- The executor's three drills are plausible, and M5 corroborates the closer-length one.
- Four fence rules have no test: the trailing-text closer, the indent limit, the backtick info string, and the minimum run length.
- There is no CRLF fixture and no fixture for the unterminated-fence direction beyond the v/w pair.

## 4. AC9 and changelog
- The AC9 edit is faithful to `cd76499c5`'s message: sentinel `SENTINEL-FROM-FILE`, the three-sided check, and the "vacuous" diagnosis.
- It switches `env -u MISSION_PLANNER_MODEL` to `env -i ...`. That is a defensible equivalent. It also adds a "do not execute" note and a dated annotation.
- Nothing else in that file changed (diff is one hunk, +12/-5).
- The changelog fragment name and form satisfy the README (`### Fixed — ... (date)`), and `make check-changelog` passes.
- The fragment claims "unterminated fences" are handled without disclosing the fail-open direction.

## 5. Rubric (approximate, /100)
- Correctness of the targeted fail-open fix and the false-deny and false-count fixes: 28/30.
- Regression safety: 12/20. Two new fail-open shapes (CRLF, unterminated), one undisclosed.
- Tests and mutation resistance: 12/20. 7 fixtures, but 4 of 5 of my mutants survive and CRLF is absent.
- Scope and discipline: 10/10. Only the intended files were touched, and a..p are byte-identical.
- Docs, changelog and AC9 fidelity: 8/10 (the fail-open caveat is not disclosed).
- Portability (bash 3.2, POSIX ERE, no gawk-isms): 6/10. It is plausible, but there is no cross-awk run.
- Total: about 74.

## Findings
BLOCKING (the verdict stays PASS only because the original fail-open is fixed and these are narrower; please fix before relying on it):
- `tools/launchd/derive-planner-lane.sh` closer test `fence_tail ~ /^[ \t]*$/` (both awk scans, about lines 118 and 157) rejects CRLF closers. Result: a new fail-open for CRLF docs containing any fenced block.

NON-BLOCKING:
- Unterminated fence at EOF fails open when it follows a valid bullet (test v, plan line 67 enshrines it). Recommend fail-closed.
- Missing tests and surviving mutants M1-M4. Add CRLF, trailing-text-closer, info-string-backtick and indent-4 fixtures.
- Cross-awk (gawk/mawk) run not performed.
- The two scans duplicate about 20 lines of fence logic (about 20 lines). Drift risk, and the mutants were only killable by editing both.

## Strongest objection
The fix makes the scanner trust Markdown's fence grammar, but the safe direction is the opposite one. Any input where the scanner and the author disagree about where a fence ends (a CRLF closer, a forgotten closer) now silently hides real core-path bullets and routes to the cheap lane. Before this change the same inputs failed closed to opus. The change closed one fail-open (r) and opened two others.

EVALUATION_VERDICT: PASS 74

## Round 2 (head c50fe663b)

Verdict: PASS 90. Both round-1 findings are fixed, and I found no new fail-open.

**Reproduced first-party**
- Routing suite under `env -i`: **122 passed, 0 failed**.
- Round-1 shapes replayed against the new head (r1 is the old head c230c7555):

| Shape | r1 head | New head |
|---|---|---|
| CRLF + fenced block (`d2/a_crlf_plain`, `c/c_crlf`) | codex (fail-open) | `opus path-not-in-codex-allowlist` |
| Unterminated ``` or ~~~ after a valid bullet (`c/k`, `d2/h`) | codex | `opus unparsable-path-entry` |

- CRLF-unterminated, a fenced second Files heading, CRLF `---`, and CRLF with only allowed paths (still codex, correct) all behave correctly.
- Pre-existing fixtures a..p are byte-identical to base under both a codex pin and a `pi:` pin (0 diffs).
- Extra control, not in your list: I ran base vs new head over the first 600 tracked `design_docs/*.md` files with a codex pin. There were **0 diffs** (no real-doc regression).
- Ill-formed shapes where the fence body sits at column 0 under an indent-4 or tab fence still route to codex (`c/f`, `c/g`, `c/j`). This is unchanged from base and from round 1, and CommonMark itself treats the `# c` as a real heading, so it is non-blocking.

**Specific probes you asked for**
- **`exit` vs the END block:** `exit` fires only on a heading or `---` line that `fence_skip` returned 0 for. So `fenced` is 0 at that point and the sentinel cannot misfire. A fence left open after the Files section ended is never scanned, because `exit` has already happened. Probe `r2/a`: unterminated fence in a later `## Notes` section still gives codex, which is correct.
- **Sentinel for a fence opened OUTSIDE the Files section:**
  - A fence opened before the Files heading and closed inside it hides the heading, giving `no-files-section` (`r2/d`).
  - A fence opened before Files and never closed hides the heading too (`c/a`, `c/b`).
  - The END sentinel requires `in_files && fenced`. If `fenced` is true at END with `in_files` false, the Files heading was hidden, and `section_count` has already failed closed before the path scan.
  - That is why mutant M9 (drop the `in_files` condition) is equivalent. It survives, harmlessly: unreachable code.
- **Awk portability:** the shared `fence_skip` with local params (`raw, indent, ...`), concatenated ahead of each program by `awk "$FENCE_AWK"'...'`, is plain POSIX user-function syntax. Locals as extra params work in BSD, gawk and mawk. `sub(/\r$/,...)` with `\r` works on BSD awk 20200816 and is standard in gawk and mawk. `fenced` and `fence_char` are intentionally global and uninitialised, which is fine. gawk and mawk are still not installed here, so this is reasoning plus a BSD run, not a cross-awk measurement.
- **Unchanged from base, not a regression:** `{2,4}` interval braces in the heading regex. Very old mawk lacks interval support. This already existed and the project's CI environment presumably handles it.

**Mutants (against the new head, restored by cp+cmp each time)**

| Mutant | Result |
|---|---|
| M1 trailing-text closer rule removed | KILLED (121/1) |
| M2 indent<=3 limit removed | KILLED |
| M3 info-string backtick rule removed | KILLED |
| M4 min fence run 3 -> 2 | KILLED |
| M7 CR strip removed | KILLED (2 fail) |
| M8 END sentinel removed | KILLED |
| M10 closer char match removed | KILLED |
| M9 END drops `in_files` | survives (equivalent, unreachable) |

The round-1 survivors M1-M4 are all now killed.

**Residual NON-BLOCKING**
- No gawk or mawk run.
- The CRLF fixtures are generated at run time (correct, given `.gitattributes` `text=auto`). I did not check that the run-time copy actually contains CR bytes beyond trusting the suite's assertion. The M7 kill supports that it does.
- M9's `in_files` guard is dead defence-in-depth.

Rubric: correctness 29/30, regression safety 19/20 (the two fail-open regressions are fixed and a 600-doc corpus is unchanged), tests 18/20, scope 10/10, docs 8/10 (changelog and plan state the exceptions), portability 6/10 (reasoned only). Total about 90.

EVALUATION_VERDICT: PASS 90
