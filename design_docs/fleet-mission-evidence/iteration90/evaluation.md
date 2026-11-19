# Iteration 90 — diff judge (sonnet, Agent tool, foreground, own worktree)

## Round 1 at `b216285db`: PASS 90, 0 blocking (87,187 subagent tokens, 25 tool calls)
| Row | Points |
|---|---|
| Dry-run block correctness | 29/30 |
| Test quality | 21/25 |
| `make test-launchd-drivers` (54/54, its own run) | 10/10 |
| Evidence and claims | 18/20 |
| Docs and changelog | 12/15 |

- It confirmed: the change is one 51-line insertion; no existing line changed; the parent environment is isolated by a subshell; failure exit codes propagate; it is safe under bash 3.2 and `set -u`; no side jobs.
- It ran 12 mutants of its own; 10 were killed. **M2 survived** (dropping the marker/root `|| exit 78`), and so did **M5** (dropping the forward-rc exit).
- It reproduced the stdin mechanism independently with `bash -c`. It could not reproduce it from a script file.
- Non-blocking findings: (1) the forwarding exits had no killing test; (2) the preview probe had no `< /dev/null`; (3) the docs lacked the live evidence and the stdin caveat.
- Strongest objection: the preview turns a dry-run into a hard gate on a live provider call.

## Controller fix `913c6f9e1`
- Arms `dry-run-codex-forward-rc`, `dry-run-codex-forward-missing` and `dry-run-probe-stdin`, each with a killing mutant (8/8 killed).
- `esac < /dev/null` on the preview probe.
- The docs record the live sets.

## Round 2 at `913c6f9e1` (resumed by SendMessage): PASS 93, 0 blocking (97,695 subagent tokens, cumulative)
| Row | Points |
|---|---|
| Dry-run block correctness | 30/30 |
| Test quality | 24/25 |
| `make test-launchd-drivers` | 10/10 (relied on the controller's 54/54 for the final tree) |
| Evidence and claims | 16/20 |
| Docs and changelog | 13/15 |

- All three findings are closed. Its own M2, M5, a forward-rc `exit 0` variant, a `/dev/zero` redirect and a dropped redirect are all killed.
- The claude-degraded set-3 rc 2 was judged correctly attributed to the 10 s `MISSION_PROBE_TIMEOUT` against a 13 s opus probe. It also argued that set 1's claude-degraded failure is the same cap, not stdin.
- New non-blocking findings: (a) the docs over-claimed a stdin cause for claude-degraded; (b) the degraded recipe's 10 s cap makes an opus-pinned dry-run flaky; (c) "launchd stdin is /dev/null" is inferred, not measured.
- (a) is fixed in `26b713887` (doc-only). (b) is documented as a trap in the design doc and the PR; not fixed, because changing the done-gate recipe or the probe floor is beyond the M3 diff. (c) is labelled "inferred" in the docs and the PR.
