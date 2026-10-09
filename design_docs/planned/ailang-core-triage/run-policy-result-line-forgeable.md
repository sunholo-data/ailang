# `run --policy`: a confined program can forge the supervisor's `policy-result:` line (#1548)

- **Date**: 2026-10-03
- **Class**: bug (security, low)
- **Recommend**: design-doc
- **Searched**: `policy-result`, `limitEnvelope`, `--policy-worker`, `ExtraFiles`, `result-fd`, `nonce` across `design_docs/` and `docs/`; `git log origin/dev --grep 1548`; open PRs. Nearest coverage: `implemented/v0_41_0/m-executor-policy-hardening.md` (built the supervisor and the fd-3 control pipe, but treats the `policy-result:` envelope as trusted output) and `docs/docs/guides/agent-tool-policy.md:18,200` (documents the envelope as the host-facing contract). The 2026-10-03 policy fixes (`96b6e5174`, `c66f33e4d`, `1099e49f6`, `ed1df2d3f`, release v0.52.1) closed #1551-#1554/#1558/#1559 but not this one.
- **Estimate**: omitted (design-doc)

Still present at origin/dev `2d7a9a174` (v0.52.1).

**Mechanism.** The supervisor (`cmd/ailang/run_policy_supervise.go`) gives the worker an authenticated control channel already: an `os.Pipe` passed as fd 3 (`cmd.ExtraFiles`, line 71) carries the admission decision, and the parent re-emits it "from the control pipe only". The *limit* verdicts are different. The parent writes them as `policy-result: {...}` lines on its own stderr (lines 166/170/180/185, `limitEnvelope` at `run_policy.go:299`). The worker's stderr is copied byte-for-byte onto that same stream (`copyCapped(os.Stderr, stderr)`, line 126), and a worker exit code is passed straight through (`return exitErr.ExitCode()`, line 178). So a program can print `policy-result: {"reason":"timeout",...}` and `exit(3)`, and the host sees exactly what a real supervisor kill looks like. The worker's own self-timeout path (`run_policy.go:196`) also writes the envelope from inside the worker, so today a host could not even filter by origin. The `policy: {admission}` line (line 159) can be forged the same way.

Impact is as the reporter says: low. Confinement is unaffected and the lie is about the same run. But AILANG World records `limit` in its effect result, and any host that retries, bills or reports on "supervisor killed it" is trusting program-controlled text.

**Options.**

1. **Out-of-band result channel (host opt-in).** `--policy-result-fd N` or `--policy-result-file PATH` (the path must be outside `fs_sandbox`). The supervisor writes the envelope (and the admission line) only there. The program has no fd or path to reach it. This mirrors the fd-3 pattern the worker already uses. It is a new CLI surface, and hosts must adopt it.
2. **Make stderr unforgeable in place.** The supervisor reads worker stderr line by line and escapes any worker line that begins with `policy-result:` or `policy:` (for example, prefixing `worker: `). It also reserves rc 3 for itself, remapping a worker exit 3 to 1 (or a new documented code). The worker's self-timeout must then report over fd 3 instead of printing the envelope itself. There is no new surface, and every existing host becomes correct. The cost is a line-buffering copier where there is now a raw 32 KiB chunk copier, which must keep the output cap exact.
3. **Per-run nonce.** The host passes a nonce that only the supervisor sees (not via the worker env, which is allowlisted anyway) and it is echoed in the envelope. This works, but it is the weakest ergonomically and still leaves rc 3 forgeable.

**Recommendation.** Do (2) as the default fix, since it repairs the documented contract for every host without new flags. Add (1) only if a host asks for a machine channel separate from human stderr. The decisions to rule on are: the remapped exit code for a worker that exits 3 (this is a contract change for programs that deliberately `exit(3)`), and whether escaped lines are rewritten or dropped. Pin both with a test where the program prints a forged envelope and exits 3, and the host-visible result must differ from a real `timeout_ms` kill.

Neighbour, not part of this doc: **#1547** (`run --policy` ignores `AILANG_CACHE_DIR`) is a direct fix, now specced in [run-policy-worker-cache-dir.md](run-policy-worker-cache-dir.md). `workerEnvAllow` (`run_policy_supervise.go:44`) omits `AILANG_CACHE_DIR`, so the worker's compile cache always lands in `<fs_sandbox>/.ailang/cache`.

Issue: https://github.com/sunholo-data/ailang/issues/1548
