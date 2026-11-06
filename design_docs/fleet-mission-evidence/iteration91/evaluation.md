# Fleet iteration 91 — independent evaluation (sonnet, Agent tool, foreground)

The judge ran in its own detached worktree each round. It wrote nothing to `~/.ailang/state` and never modified the branch.

## Round 1: commit `635ea94da`. PASS 81/100, 0 blocking

| Criterion | Score |
|---|---|
| Fixes the World ticket | 26/30 |
| No false suppression | 17/25 |
| Blast radius and readers | 14/15 |
| Test quality | 15/20 |
| Docs, process, changelog | 9/10 |

Verified first-party:
- **Env at the driver block:** `MISSION_NAME`, `MISSION_REPO` and `MISSION_DOC` are exported at driver lines 74–76. `MISSION_WORKDIR` comes from `pin-root.sh:360` and from World's plist. The `IDLE_STATE_DIR` seam works through the real `_mc_bounded`.
- **Live parity on World:** read, `--record` and read again all gave `c099527d…`.
- **Exclusions:** they skip record-only merges `1835d8be` and `8712ffc5`.
- **Messaging store:** the helper pins it, so the reader and writer cannot split.
- **Blast radius:** with no record, the block costs one `[ -f ]` and no network.
- **Slot-verdict readers:** no reader breaks.

Findings (all non-blocking):
1. **Strongest objection.** `--record` fingerprinted the inputs at the END of the fire. A ruling that lands mid-fire would be absorbed, and the loop could then skip for up to 24 h.
2. The pre-check runs before the overlap guard. Its reads are harmless.
3. Charter edits that flip no ledger row are invisible.
4. A dry run consumes or honours a record.
5. An empty `MISSION_DIRECTIVE_AUTHORS` is accepted silently.
6. Test gaps: the check-runs tip-vs-head choice, inbox id identity and the 200-row trim were untested. Three of the judge's seven mutants survived.
7. A red or flaky tip after a record costs one extra fire. This fails safe.

## Controller fix: `33190af48`
- **Finding 1.** The driver now takes a `--snapshot` once per fire, after the overlap guard and the dry-run exit and before the controller starts. `--record` promotes the snapshot whose `fire=` matches `MISSION_FIRE_ID`, and never recomputes.
- **Finding 4.** The pre-check skips dry runs.
- **Finding 5.** A set-but-empty author allowlist is refused.
- **Finding 6.** New arms cover the tip-vs-head check-runs choice, inbox id identity and the 200-row trim. A mid-fire race arm and snapshot-block arms with a placement check were also added. The mutant count went from 8 to 12, all killed.
- **Finding 3.** Documented in the Gate 5 text as a limitation bounded by the 24 h expiry.

## Round 2: commit `33190af48`. PASS 86/100, 0 blocking

| Criterion | Score |
|---|---|
| Fixes the World ticket | 27/30 |
| No false suppression | 21/25 |
| Blast radius and readers | 14/15 |
| Test quality | 15/20 |
| Docs and process | 9/10 |

- **(a) The race is closed.** The snapshot runs once per slot, outside the transient-retry loop, so retries share one `MISSION_FIRE_ID`. Codex controllers receive it through `codex-env-args.sh`'s `MISSION_*` forwarding, and the driver has no `env -i`.
- **(b) No overlapping fire can clobber a pending snapshot.** The overlap guard yields and the dry run exits before the snapshot block. The guard's existing gap between its check and the pidfile write is not widened by this change.
- **(c) Mutants.** The suite's 12 are all killed. The judge wrote 11 more: 8 killed and 3 survived. The survivors are redundant defences: a loosened fire-prefix match, no `at=` check on promote, and no hash-shape check on promote.

Findings (all non-blocking):
1. **Residual.** A fire that changes its own inputs after its snapshot makes the next fire run in full. Examples are mail to its own inbox, a ledger row it files, or a non-record commit it pushes. This fails safe. World iterations 341–350 sent `[claim] … no pick` mail to `mission-world`; if that practice resumes, the feature never idles.
2. Charter edits that flip no ledger row are documented as not being inputs.
3. There is no end-to-end test through the real `_mc_bounded` and the pin.
