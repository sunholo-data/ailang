### Changed — motoko GLM-5.3-Flash rows cap output at 32000 tokens (was 65536)

`motoko-or-glm-5-3-flash` and `motoko-lane-or-glm-5-3-flash` now declare `max_output_tokens: 32000`,
pi's wire budget. In the 2026-10-02 `ailang_only` lane A/B (23 benchmarks x 3 trials) motoko and pi
tied 55/69, but motoko cost $6.86 against pi's $4.20: on hard benchmarks GLM-Flash reasoned to the
whole output cap (gauntlet_10: 3 of 4 steps, 197k output tokens), and at 65536 each such step cost
twice pi's. The empty-stop guard's two "you were cut off" nudges worked as designed; the model kept
reasoning, as it did under pi. motoko's wire name resolves to the lane row, so the cap reaches the
request.

The rig's `motoko` shim now runs `~/dev/mk-20261002` (`sunholo/main-dst-20261002`, `de68fddf`), the
commit the cloud image pins.
