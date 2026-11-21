### Registry check pacing and budget diagnostics

- Pace the registry package check with a Go memory target 64 MiB below its RSS cap, retaining the RSS watchdog and 200-second wall limit.
- Include cap source, runtime settings, sampled memory and elapsed time in refusals, plus image-specific disposable Docker reproduction when image identity is available.
- Bank optional check peak RSS and check wall milliseconds in publish metadata, excluding dependency preparation and verification.
- Document memory troubleshooting and the limits of local/Docker comparisons. Production rollout uses release tags per the recorded design ruling.

Design: [M-VALIDATOR-CHECK-BUDGET-PARITY](../../design_docs/planned/v0_56_1/m-validator-check-budget-parity.md).
