# Judge report: Phase 3a skill-resolution spike (fleet iteration 34)

Judge: sonnet (claude-sonnet-5-5), independent of the codex planner/executor and the opus controller.
Reviewed commit 3ec678757. Not committed.

**Score: 88/100. Verdict: PASS.** (70 = pass; no hard fail.)

| Category | Max | Score | Note |
|---|---|---|---|
| Controls and replication (adapts "Tests Pass") | 20 | 19 | All 8+ re-derived verdicts and all 3 replications agree; negatives fired. -1: pi negative arm returned empty text, not an explicit "not found" |
| Lint / instrument hygiene | 10 | 8 | Cleanup is ledger-bound and safe; repair 8 weakens the audit and is not covered by any gate |
| Acceptance criteria (measurement answered) | 30 | 27 | P1-P3 answered with discriminating tokens; gaps are named (keychain-only auth, codex per-fire CODEX_HOME) |
| Code quality / evidence honesty | 15 | 12 | See non-blocking N1-N5 |
| Documentation | 15 | 13 | Log rows, findings, probe doc present. D-FLEET-16 cited but not in the charter ledger at this commit |
| Design fidelity | 10 | 9 | Measurement-only, nothing selected |

## Blocking findings
None.

## Non-blocking findings
- **N1 (safety statement imprecise).** findings.md says "no audit delta names a run id". The CONCURRENT_CHANGE deltas in runs 4306, 4311 and 4447 DO name run ids: the sibling spike runs' own markers and World dirs (for example 4306's `audit/1.delta` lists a `...1791874311...-i-world` entry). The three live runs overlapped and wrote into the same user parents. This is benign because the names are unique per run, but the sentence should say "no delta names THIS run's id".
- **N2 (repair 8 weakens detection and is ungated).** After repair 8, a deletion of any pre-existing entry in a shared parent (a `<` delta line not naming the run) is classed CONCURRENT_CHANGE and exits 0 with `safety=OK`. The post-run leftover check covers leaks, not deletions. The only mutation gate (`mutation-gate.log`) exercises the scratch-cleanup skip, not repair 8. See the drill below. The cleanup itself cannot delete anything unowned, so this is a detection gap, not a hazard.
- **N3 (n is small, claims read general).** "User level wins a same-name collision" rests on haiku-5-5 only, 2 runs from the spike (plus my 3 calls, all U). "`$CODEX_HOME/skills` wins a collision" rests on ONE codex arm (K-collision, n=1, gpt-6.1-sol). Both are fine as "observed on this model/CLI version", and the log rows give versions, but they should not be read as a language of precedence. K-by-path is cited as evidence though its verdict is INSTRUMENT_FAILURE (one post control lost to capacity, one scorer rejection).
- **N4 (run 3918 is a weak repetition).** It ended `safety=SAFETY_FAILURE`, `controls_ok=33/51`, with many World/add-only/auth arms failed. findings.md says "valid arms only" and the cited arms (C-collision-1, C-v1-1, C-project) do have all their controls MATCH/ABSENT. Acceptable, but the SAFETY_FAILURE came from the pre-repair-8 /private/tmp delta and is not explained beside the citation.
- **N5 (small wording).** C-project (run 4306) is described as "mis-copied one hex digit"; the answer's resource token is 47 chars (one digit dropped), skill token exact. Same conclusion. `phase3b-decision.md` says "filed as charter decision D-FLEET-16", but `grep -rl D-FLEET-16 design_docs` hits only the decision file and the design doc: no ledger row exists in `design_docs/fleet-mission.md` at this commit. It also marks A "recommended" but "default if unanswered: C", which is internally inconsistent though still not a selection.
- N6: 416 tracked evidence files contain the local path `/Users/voightkampff/...`. Not a credential or address; noted only.

## (A) Evidence integrity: re-derived verdicts (token to location via runs/*/tokens.tsv)

| Arm (run) | Claimed | Final-answer tokens map to | Match |
|---|---|---|---|
| C-collision-1 (4306) | U/U | skill token U, resource token U | yes |
| C-collision-1 (3918) | U/U | U / U | yes |
| C-v1-1 (4306) | U/U | U / U | yes |
| C-v1-1 (3918) | U/U | U / U | yes |
| C-world-1 (4306) | U/U | U / U | yes |
| C-add-only (4306) | X/X | X / X | yes |
| C-synthetic (4306) | S/S | S / S | yes |
| C-settings (4306) | ABSENT | final text ABSENT, no tokens | yes |
| C-project (4306 / 3918) | INCONCLUSIVE / P/P | P skill + truncated resource / P / P | yes |
| K-claude (4447) | ABSENT | ABSENT; L+ pre/post MATCH L/L, negatives ABSENT | yes |
| K-project (4447) | A/A | A / A | yes |
| K-collision (4447) | H/H | H / H | yes |
| K-legacy-user (4447) | H/H | H / H | yes |
| K-agents-user (4447) | B/B (INSTRUMENT_FAILURE) | B / B, verdict correctly not promoted | yes |
| I-project, I-collision, I-world, I-no-skills (4311) | ABSENT | ABSENT, no tokens | yes |

Raw-stream checks (tokens only, nothing copied):
- **C-collision-1 raw stdout:** the final `result` event (success, not error) holds U skill + U resource tokens. The model called `Skill` for the name, then `Read` on the `~/.claude/skills/...` path. Tool results carried only U tokens, never P tokens. The final answer is the model's own result content, not an echoed prompt.
- **K-collision and K-project raw stdout:** `cat` outputs contained only H (respectively A) tokens, and the last `agent_message` item holds the matching pair, equal to final.txt. For K-collision the planted P-side (A) and B tokens never appeared in any tool output.
- **Auth (C-auth-isolated):** `init` system event `apiKeySource: none`, result `success`, output_tokens 3, final text `1`. C-auth-base is identical (`none`, 3 tokens). Claim confirmed.
- **Caveat:** runs/1791874306-*/metadata.txt lists `CLAUDE_CODE_OAUTH_TOKEN` among the env names (values not recorded). The caveat in findings.md is correct and appropriately scopes the auth claim.
- Summaries: 4306 `controls_ok=51/51 safety=OK`; 4447 `32/43 safety=OK`; 3918 `33/51 safety=SAFETY_FAILURE` (N4).

## (B) Instrument audit
- **Cleanup safety.** `own` runs `mkdir` (no `-p`) and appends to the ledger only after it succeeds, so a pre-existing path aborts under `set -eu` and is never ledgered. `rm -rf --` is applied only to ledger entries (cleanup trap and per-group marker removal, which re-checks ownership). `rm -rf` does not follow symlinks, and the pi `models.json`/`auth.json` links sit inside the owned scratch and are removed as links. Cleanup runs with `set +e`, so one failure cannot skip the rest. I found no path by which it deletes something it did not create.
- **Repair review.** 1 (unset CLAUDE_CONFIG_DIR), 2 (strip one fence), 3 (K:H path), 4 (symlinked models.json), 5 (`--add-dir` argument order), 7 (init event, <=5 tokens), 9 (last `agent_end`) are direction-neutral or correct. 6 (negative-control Skill-name allowance) only applies to `kind == negative` and only to the run's own `-n<k>` names; it makes a control easier to pass but cannot create a subject MATCH. 10 (CLAUDE.md/AGENTS.md reads) only stops those reads voiding a verdict; the files cannot carry run tokens, and it could only turn an INVALID_DISCOVERY into ABSENT for K-claude, where the answer ABSENT is what the model gave. 8 (CONCURRENT_CHANGE) cannot manufacture a subject verdict but does weaken the safety audit (N2). 11 (codex runs `--fence off`) is disclosed and widens codex's write surface only (sandbox read-only remains).
- **Mutation drill on repair 8** (copies in a scratch dir inside my worktree, `SPIKE_TEST_ROOT`, `--dry-run --reps 1 --only codex`):

| Variant | Result |
|---|---|
| v0: original audit + deliberate stray `zz-fleet-spike-<runid>-LEAK` dir in a test-root user parent | AUDIT status=SAFETY_FAILURE, rc=1, `safety=SAFETY_FAILURE` (caught) |
| v1: audit neutered (`if false; then result=SAFETY_FAILURE`), no leak | rc=0, `safety=OK` (nothing to catch) |
| v2: audit neutered + the same deliberate leak | AUDIT status=CONCURRENT_CHANGE, rc=0, `safety=OK` (**NOT caught**) |

  With repair 8's SAFETY_FAILURE branch removed, no other mechanism catches a leaked run-named dir in a shared parent: the leak is printed as CONCURRENT_CHANGE and the run exits 0. The cleanup trap's own `[ ! -e ]` check covers only ledgered paths. The existing mutation gate does not cover this branch. Copies and the three dry-run evidence dirs they created were deleted; `git status` is clean.

## (C) Independent replication (own method; fresh random 16-hex tokens; markers made with `mkdir`, removed with `rm -rf`, `find ... -name 'zz-judge-*'` empty afterwards)

| Claim | Setup | Result |
|---|---|---|
| **C1 claude collision: CONFIRMED** | Same name planted in `~/.claude/skills/<n>` (token U=b40d742f7669e691) and `<worktree>/.claude/skills/<n>` (token P=692f6e7877866203); haiku-5-5, cwd = worktree, 3 calls | All 3 calls returned `b40d742f7669e691` = **U (user level wins)**. Negative control (name planted nowhere): "Unknown skill", no token |
| **C2 codex: CONFIRMED** | gpt-6.1-sol, read-only, `-C` worktree. Skill only in `.claude/skills` (T=2554d3464c82996f); skill only in `.agents/skills` (T=56c9a04d4261b7b2) | `.claude/skills`-only: `ABSENT` (token not returned). `.agents/skills`-only: `56c9a04d4261b7b2` = **returned**. Negative: `ABSENT`. No capacity errors |
| **C3 pi: CONFIRMED (with a note)** | pi 0.85.1, kimi-k3:cloud, `--print --mode json --no-session --approve`. Skill dir in a fresh `/private/tmp` scratch; cwd was a fresh empty dir in that scratch because cwd = worktree made pi fail to start (project `.pi/extensions/ailang-exec.ts` tool-name conflict with the user-level extension, rc 1, empty output; this is a worktree-side artifact, not a skill result) | With `--skill <abs dir>` + `/skill:<name>`: returned `7d2460ee8c2e1636` = the planted token. Without `--skill`: no token (final text empty rather than an explicit "not found", so the negative is weaker than the probe doc's row 9) |

## (D) Scope and honesty
- Rows P1-P3 and findings.md are scoped to named CLI versions and one model each, state the keychain-only gap and the credential-copy gap, and explicitly say the spike did not re-measure v1's driver cwd. Over-reach is limited to N3 and N5.
- phase3b-decision.md selects nothing (header: "No mechanism has been selected or activated"); options A/B/C match the measurements (U beats project, H beats project, pi explicit only, S wins with env token). Its default-if-unanswered line (C) is a parked default, not a selection.
- Credential/address grep over `design_docs/fleet-mission-evidence/iteration34` (`mark@|@aitanalabs|@sunholo\.com|sk-ant-|sk-proj-|refresh_token|eyJhbGci`): **zero hits**.
