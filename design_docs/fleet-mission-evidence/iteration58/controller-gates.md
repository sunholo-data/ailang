# Controller gate artifacts

Source candidate `0be2c1957a78b14aa281eb67f4c7e3efad82bb5a`; scratch CLI base42 (Go unchanged). Full launchd log remains in scratch; its decision-bearing tail and content hash are retained here, avoiding a 1,729-line bulk artifact. Dry-runs only probed lanes and did not write PID files or reload missions.

## fleet-i58-controller-launchd.log

rc=0; SHA256 `84073541c6745cd42210ff8d85a2291f7b18d3ec18b0eeb83b1a087a04da7de9`; 1729 lines.

```text
# discovery walk marker_count=2 node_ceiling=10672
ok 34 - arm cap terminates a hung command and reports it
ok 35 - arm cap kills a wrapper grandchild
# run_lane evidence ready=yes wrapper_pid=63482 child_pid=63489 cwd=/private/var/folders/kr/h0wr2sj94vd6ljtmsxv8jkt00000gn/T/test-motoko-connection-probe.FIix6A/run-lane-fixture-2861 pre_timeout_child_live=yes timeout=yes outer_cap_fired=no outer_cap_rc=0 survivors=0 cleanup=0 probe_rc=1 markers=yes real_lsof=/usr/sbin/lsof
ok 36 - production run_lane timeout kills wrapper grandchild
ok 37 - arm cap reports a hung arm by name with its captured output
ok 38 - arm cap override rejects invalid values
ok 39 - tree node ceiling rejects invalid values
ok 40 - tree discovery deadline rejects invalid values
ok 41 - descendant discovery refuses on the node-count ceiling
ok 42 - descendant discovery stub refusal carries its own message
# run_lane evidence ready=yes wrapper_pid=63795 child_pid=63800 cwd=/private/var/folders/kr/h0wr2sj94vd6ljtmsxv8jkt00000gn/T/test-motoko-connection-probe.FIix6A/run-lane-fixture-2863 pre_timeout_child_live=yes timeout=yes outer_cap_fired=no outer_cap_rc=0 survivors=0 cleanup=0 probe_rc=1 markers=yes real_lsof=/usr/sbin/lsof
ok 43 - production run_lane SIGKILL escalation kills a TERM-immune wrapper grandchild
ok 44 - REAL_LSOF containment refuses an ambient lsof ahead of getconf PATH
ok 45 - REAL_LSOF containment accepts a leading directory without an lsof
ok 46 - bound derivation responds to a slowed stimulus
ok 47 - bound derivation rejects a nonnumeric fork rate
ok 48 - bound derivation rejects a zero fork rate
ok 49 - bound derivation rejects an empty fork rate
ok 50 - bound floor disabled is loud, not silent
ok 51 - bound floor enforced refuses under the floor and accepts its boundary
ok 52 - bound floor flag rejects values other than zero or one
ok 53 - observed proxy spread refuses above 4.70 and accepts below it
ok 54 - bound drift classifier is loud only when the end rate needs a higher scale
# measurement failure evidence mode=exit1 rc=72 refusal_count=1 derived_count=0
ok 55 - bound measurement refuses a stimulus that exits nonzero
# measurement failure evidence mode=nonexec rc=71 refusal_count=1 derived_count=0
ok 56 - bound measurement refuses a non-executable stimulus
# measurement failure evidence mode=timer rc=73 refusal_count=1 derived_count=0
ok 57 - bound measurement refuses an unsuccessfully terminated timer
ok 58 - refusal-branch count still matches the set this suite covers (28)
ok 59 - wall-clock literal census preserves fixed timeout/node pins and scaled capacity bounds
# bound drift: end-of-suite fork rate 688/s scale_end=1 scale_used=1 drift=none
PASS: 59 probe self-test arms ran
launchd drivers: tests + bash 3.2 syntax OK
```

## fleet-i58-dryrun-available-healthy.log

rc=0; SHA256 `4b1ff1fd4c1dc208ef89898a1532af79d44345f953385c908379e218f1aa468f`; 7 lines.

```text
[2026-10-22 23:39:10] quorum artifact directory: /Users/voightkampff/.ailang/state/mission-quorum
[2026-10-22 23:39:10] anthropic credential: CLAUDE_CODE_OAUTH_TOKEN (the loops' own token) for inference; the quota reader reads usage with the login credential first
[2026-10-22 23:39:10] driver pin: running committed origin/dev @ 0be2c1957a78b14aa281eb67f4c7e3efad82bb5a with mission work repository /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading (source clone /Users/voightkampff/dev/sunholo-data/ailang was 0 behind); pinned target 0 behind origin/dev (baseline 42d93e4b6317e7b068bf52cd32dc226c013dd117)
[2026-10-22 23:39:10] driver pin drift: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:39:10] driver pin age: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:39:14] ration gate: blocked buckets (over ration or unknown quota):codex anthropic
[2026-10-22 23:39:40] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading pidfile=/Users/voightkampff/.ailang/state/mission-stapledon.pid prefs=claude-opus-5-5,codex:gpt-6.1-sol timeout=21600s | roles: designer=pi:ollama/glm-5.3:cloud planner=pi:ollama/kimi-k3:cloud executor=pi:ollama/deepseek-v4.1-flash:cloud evaluator=pi:openrouter/minimax/minimax-m3 | lanes=ok | pin=pinned(0 behind) | skill-sync=current
```

## fleet-i58-dryrun-degraded.log

rc=0; SHA256 `3e3b18181ab27625324dc610deb541d0af97bed62a43489bda4b487c7631c1b2`; 15 lines.

```text
[2026-10-22 23:37:37] quorum artifact directory: /Users/voightkampff/.ailang/state/mission-quorum
[2026-10-22 23:37:37] anthropic credential: CLAUDE_CODE_OAUTH_TOKEN (the loops' own token) for inference; the quota reader reads usage with the login credential first
[2026-10-22 23:37:37] driver pin: running committed origin/dev @ 0be2c1957a78b14aa281eb67f4c7e3efad82bb5a with mission work repository /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading (source clone /Users/voightkampff/dev/sunholo-data/ailang was 0 behind); pinned target 0 behind origin/dev (baseline 42d93e4b6317e7b068bf52cd32dc226c013dd117)
[2026-10-22 23:37:37] driver pin drift: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:37:37] driver pin age: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:37:43] ration gate: blocked buckets (over ration or unknown quota):codex anthropic
[2026-10-22 23:37:43] codex:gpt-6.1-sol quota admission blocked; skipping inference probe
[2026-10-22 23:37:43] codex model 'gpt-6.1-sol' unusable: quota-limited
[2026-10-22 23:37:43] codex probe output: Codex quota admission blocked (over ration or observation unavailable)
[2026-10-22 23:37:43] codex designer lane -> falling back to 'pi:ollama/glm-5.3:cloud' for this fire (model 'gpt-6.1-sol')
[2026-10-22 23:37:43] codex planner lane -> falling back to 'pi:ollama/kimi-k3:cloud' for this fire (model 'gpt-6.1-sol')
[2026-10-22 23:37:50] pi model 'bogus/no-such-model' unusable: probe failed (rc=1)
[2026-10-22 23:37:50] pi probe output: Error: Model "bogus/no-such-model" not found. Use --list-models to see available models.
[2026-10-22 23:37:50] pi executor lane -> falling back to opus for this fire (model 'bogus/no-such-model')
[2026-10-22 23:38:09] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading pidfile=/Users/voightkampff/.ailang/state/mission-stapledon.pid prefs=claude-opus-5-5,codex:gpt-6.1-sol timeout=21600s | roles: designer=pi:ollama/glm-5.3:cloud planner=pi:ollama/kimi-k3:cloud executor=opus evaluator=pi:openrouter/minimax/minimax-m3 | lanes=DEGRADED(3) - `designer`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `pi:ollama/glm-5.3:cloud` - `planner`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `pi:ollama/kimi-k3:cloud` - `executor`: **pi** lane `bogus/no-such-model` unusable (probe rc=`1`) → handed to `opus` (end of chain) | pin=pinned(0 behind) | skill-sync=current
```

## fleet-i58-dryrun-healthy.log

rc=0; SHA256 `2bd5bbc61d5742f4969b2d7c8dbd5260cb4822b8f3309a3de78809429bc76e77`; 15 lines.

```text
[2026-10-22 23:36:58] quorum artifact directory: /Users/voightkampff/.ailang/state/mission-quorum
[2026-10-22 23:36:58] anthropic credential: CLAUDE_CODE_OAUTH_TOKEN (the loops' own token) for inference; the quota reader reads usage with the login credential first
[2026-10-22 23:36:58] driver pin: running committed origin/dev @ 0be2c1957a78b14aa281eb67f4c7e3efad82bb5a with mission work repository /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading (source clone /Users/voightkampff/dev/sunholo-data/ailang was 0 behind); pinned target 0 behind origin/dev (baseline 42d93e4b6317e7b068bf52cd32dc226c013dd117)
[2026-10-22 23:36:58] driver pin drift: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:36:58] driver pin age: 0 below warning threshold 25; notice re-armed
[2026-10-22 23:37:04] ration gate: blocked buckets (over ration or unknown quota):codex anthropic
[2026-10-22 23:37:04] codex:gpt-6.1-sol quota admission blocked; skipping inference probe
[2026-10-22 23:37:04] codex model 'gpt-6.1-sol' unusable: quota-limited
[2026-10-22 23:37:04] codex probe output: Codex quota admission blocked (over ration or observation unavailable)
[2026-10-22 23:37:04] codex designer lane -> falling back to 'pi:ollama/glm-5.3:cloud' for this fire (model 'gpt-6.1-sol')
[2026-10-22 23:37:04] codex planner lane -> falling back to 'pi:ollama/kimi-k3:cloud' for this fire (model 'gpt-6.1-sol')
[2026-10-22 23:37:04] anthropic:claude-haiku-5-5 quota is UNREADABLE (not measured) — probing the lane instead of refusing it; an unreadable quota is not an exhausted one
[2026-10-22 23:37:11] codex executor lane: claude:claude-haiku-5-5 rung skipped (anthropic probe failed or over ration) — walking the chain
[2026-10-22 23:37:11] codex executor lane -> falling back to 'pi:ollama/deepseek-v4.1-flash:cloud' for this fire (model 'gpt-6.1-sol')
[2026-10-22 23:37:37] DRY RUN ok: mission=stapledon repo-slug=sunholo-data/stapledons-godot doc=design_docs/stapledon-mission.md workdir=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading pidfile=/Users/voightkampff/.ailang/state/mission-stapledon.pid prefs=claude-opus-5-5,codex:gpt-6.1-sol timeout=21600s | roles: designer=pi:ollama/glm-5.3:cloud planner=pi:ollama/kimi-k3:cloud executor=pi:ollama/deepseek-v4.1-flash:cloud evaluator=pi:openrouter/minimax/minimax-m3 | lanes=DEGRADED(3) - `designer`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `pi:ollama/glm-5.3:cloud` - `planner`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `pi:ollama/kimi-k3:cloud` - `executor`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `pi:ollama/deepseek-v4.1-flash:cloud` | pin=pinned(0 behind) | skill-sync=current
```

## Record index provenance

Both index commands printed this record worktree’s paths. The fresh checkout causes the CLI’s source-mtime staleness warning; `git diff --exit-code base42 merge2de -- internal cmd go.mod go.sum` is clean, so the stamped scratch CLI has identical Go input contents. No shared binary was installed. STATUS55’s full six-line block moved into the archive; new STATUS58 adds six lines; the charter remains345 lines before factual pointer/queue edits. STATUS57/56 and all29 decision rows are preserved.

Excerpt formatting trims line-end whitespace; hashes above refer to the unchanged original scratch logs. The corpus TSV uses quoted fields/LF to satisfy whitespace gates; parsed cells are identical to the original snapshot.
