# A/B: bash lane vs `ailang_only` lane — 2026-09-16

**Subject**: `pi-or-minimax-m3` (OpenRouter, metered), pi 0.85.1, 5 benchmarks, 1 trial each, both arms
in one session. Policy for the `ailang_only` arm: `allowed_caps = ["IO","FS"]`, `fs_sandbox` = the eval
workspace root. Raw `eval-paired` output: `m-agent-ailang-only-execution-ab-2026-09-16.json`.

Why OpenRouter and not the rig's local 27B: the rig run was in GPU contention (nightly + rotation
live, rig lock dir absent under a held acquire — see memory `project_rig_lock_not_exclusive_2026_09_16`),
so its numbers were not a measurement. Mark's call: "much faster proof using OpenRouter".

| arm | benchmark | result | error | s | $ | turns | tools | executor_version | tool_policy |
|---|---|---|---|---|---|---|---|---|---|
| or-bash | api_call_json | PASS | none | 32 | 0.0368 | 12 | 11 | pi@0.85.1 | ['<cli default>'] |
| or-bash | cli_args | PASS | none | 30 | 0.0446 | 12 | 11 | pi@0.85.1 | ['<cli default>'] |
| or-bash | fizzbuzz | PASS | none | 45 | 0.0292 | 8 | 7 | pi@0.85.1 | ['<cli default>'] |
| or-bash | fold_reduce | PASS | none | 27 | 0.0227 | 6 | 5 | pi@0.85.1 | ['<cli default>'] |
| or-bash | state_machine_vending | PASS | none | 115 | 0.0449 | 12 | 11 | pi@0.85.1 | ['<cli default>'] |
| or-aionly | api_call_json | PASS | none | 42 | 0.0235 | 6 | 5 | pi@0.85.1 | ['Read', 'Edit', 'Write', 'AilangCheck', 'AilangRun'] |
| or-aionly | cli_args | fail | logic_error | 2 | 0.012 | 2 | 1 | pi@0.85.1 | ['Read', 'Edit', 'Write', 'AilangCheck', 'AilangRun'] |
| or-aionly | fizzbuzz | PASS | none | 18 | 0.0211 | 6 | 5 | pi@0.85.1 | ['Read', 'Edit', 'Write', 'AilangCheck', 'AilangRun'] |
| or-aionly | fold_reduce | PASS | none | 39 | 0.0223 | 6 | 5 | pi@0.85.1 | ['Read', 'Edit', 'Write', 'AilangCheck', 'AilangRun'] |
| or-aionly | state_machine_vending | PASS | none | 40 | 0.0219 | 5 | 4 | pi@0.85.1 | ['Read', 'Edit', 'Write', 'AilangCheck', 'AilangRun'] |

**Paired**: on (`ailang_only`) 4/5, off (bash) 5/5; one discordant pair (`cli_args`, bash-only pass);
McNemar not reportable (b+c = 1, floor 10); control arm at 100% so headroom warns. **No evidence of
a systematic gap at this size.** The one miss is a MODEL STOP, not a gate refusal: 2 turns, one
`read`, 93 output tokens, then `stop` — the placeholder was never edited and `ailang_run` was never
called. The `ailang_only` arm was cheaper on 4 of 5 and never hit `policy_violation`.

Every row on both arms carries `executor_version` and `tool_policy` (the bash arm banks the
`<cli default>` sentinel), so the two lanes can never pool silently. The earlier attempt on
`pi-or-deepseek-v4-flash:floor` banked one `api_error` (idle 3 min, no output) — that lane's known
`stream_dead` shape — and exposed that failure rows dropped provenance, fixed in `173564f20`.
