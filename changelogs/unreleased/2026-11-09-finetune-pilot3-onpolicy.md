### Added — fine-tune pilot-3 on-policy repair pipeline

- `tools/finetune/collect_onpolicy.sh` runs the pilot-2 fine-tune on the training benchmarks through the
  trace proxy; it refuses held-out tasks and takes the rig through the cooperative handoff.
- `tools/finetune/onpolicy_pairs.py` replays the traced writes and edits, re-checks every file version
  with the current `ailang`, and emits the failing versions as repair items. When the model's own fix
  failed, the item also records the attempt it was fixing (`prev`).
- `stale_fix.py --prompt onpolicy --items`: verified repair of those items. The repairer gets the task,
  the expected output and the agent reference.
- `synth_episodes.py --onpolicy`: an `onpolicy` episode kind. The model's own mistakes stay as no-loss
  context, followed by a lookup of a teaching-prompt section matched to the error, then the verified fix.
- `train2.py` skips assistant messages with `weight: 0` when masking.
- `build_mix2.py --max-steps` drops long episodes.
- `thoughts.py` now quotes the specific error line rather than the module-loading wrapper, and adds fix
  phrases for PAR017, `->` match arms, lambdas, interpolation and `as` casts.
- New docs page `docs/docs/finetuning/pilot-3.md`, a step-by-step log kept as the work happens.
