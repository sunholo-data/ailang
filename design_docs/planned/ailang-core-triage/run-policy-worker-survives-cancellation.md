# Policy-supervised worker survives successful parent cancellation

- **Date**: 2026-10-11
- **Class**: bug
- **Recommend**: design-doc
- **Source**: canonical message `inbox_1791741386377_b0b76b40`, from `stapledons_godot`; follows `inbox_1791630246888_28589e92`.
- **Searched**: `policy cancel`, `policy.*surviv`, `supervis.*cancel`, `detached`, `process group`, `control`, `reportDecision`, `Setpgid`.

The consumer's official v0.54.4 (`bc078dc1b41308542bbccd3417bf040283e817df`, Darwin arm64) retest confirms ordinary managed/async cancellation works, but policy-supervised cancellation still blocks its M1 cleanup gate. Its saved `explicit-cancel/policy-cancel.json` records `Ok(())`, host exit in 2–5 ms, and a surviving `--policy-worker` reparented to PID 1; `explicit-cancel/lifecycle.json` records the same result for two stream-source workers after a 48 ms quit. These are inspected reporter artifacts, not a new reproduction by this triage session; no provider calls were used.

Source inspection supports the reported mechanism: `cmd/ailang/run_policy_supervise.go` (`supervisePolicyRun`) creates its worker with `proctree.Configure`, which sets a separate process group. Killing the outer owned group therefore misses this runtime-created inner group. The supervisor's timeout uses a background context, and `cmd/ailang/run_policy.go` (`reportDecision`) closes the admission pipe after one message; that pipe currently supplies no continuing parent-lifetime signal.

Related designs are [worker lifecycle](../../implemented/v0_54_4/m-background-worker-lifecycle.md), [policy hardening](../../implemented/v0_41_0/m-executor-policy-hardening.md), and [policy result integrity](../../implemented/v0_53_0/m-run-policy-result-unforgeable.md). None found deciding this cross-group ownership boundary: the released lifecycle design explicitly covers descendants in the owned group and excludes detached descendants. A follow-up design must choose safe propagation of cancellation/parent death across AILANG-created policy groups, bounded stop/reap acknowledgement, and truthful success semantics while preserving confinement, admission-channel integrity, timeout/output limits, unrelated-worker isolation, and PID-reuse safety. Graceful signal forwarding alone cannot cover the existing force-kill path. Require exact supervisor and worker PID/group evidence for explicit managed/async cancellation and host teardown before rerunning the consumer gate. Provider-side inference cancellation remains separate.

The same message's stdin EOF observation and the separate generated-bin keyless `--offline`/`--help` report (`inbox_1791740982044_18b3491c`) are distinct concerns; they are not part of this cancellation triage artifact.

Follow-up implementation: [M-WORKER-FOLLOWUP](../../implemented/v0_54_5/m-worker-policy-and-lazy-ai.md), independently accepted 87/100. Released in [v0.54.5](https://github.com/sunholo-data/ailang/releases/tag/v0.54.5); [official-artifact cleanup retests](../../implemented/v0_54_5/validation/release/official-acceptance-summary.json) pass for marker-backed managed and two unfinished async policy workers, with every recorded positive PID absent. Original report and baseline triage evidence above are preserved.
