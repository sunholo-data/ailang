### Added — fine-tune pilot-4 tooling (docs/docs/finetuning/pilot-4.md)

- `tools/finetune/pilot4_run.sh` + `pilot4.yaml`: round-2 on-policy pipeline. Pools round-1 and
  round-2 repairs, restarts from the pilot-2 adapter, gates on a 30-minute behaviour check, and
  traces every held-out arm.
- `tools/finetune/trace_dropped.py`: counts tool calls the model generated but the agent never
  received (a runaway or malformed `write` the server's parser dropped), and which runs ended on one.
- `tools/finetune/probe_after_read.py`: replays real "after reading the template" requests from a
  training trace and records the first action. Refuses held-out benchmarks.
- Attempt `history` in `onpolicy_pairs.py` (plus `--enrich-pairs` to add it to earlier repairs by
  code hash) and loop-break episodes in `synth_episodes.py`; build-up episodes (helpers + stub
  `main`, check, then one edit); reference lookups for `MOD010`, Option/Result mix-ups, and
  signature lookups for type mismatches in calls to stdlib functions.
- `build_mix2.py --max-write-chars` and `--build-per-bench`; `collect_onpolicy.sh` takes `OP`/`LANE`.
- `pi-gemma4-12b-ft4` lane in the model registry.

### Fixed — fine-tune thoughts quoted a generic fix for `MOD010`

The fix phrase matched the old "canonical path" wording; the compiler now says "doesn't match file path".
