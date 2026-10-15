### Stapledon workaround retirement

- Run all ten `std/map` operations natively under `--bytecode --strict-bytecode`, with immutable updates, canonical iteration, opaque VM values, and map result conversion. Refs #1447.
- Preserve lockfile generation metadata when dependency resolution is unchanged, including empty dependencies and toolchain upgrades. Existing malformed or unreadable locks now produce actionable errors. `ailang lock --check` remains the read-only CI drift gate.
- Include the applied function's name in application type errors, including swapped `std/map.lookup` arguments, while retaining anonymous-call, arity, and collision diagnostics.
- Document the downstream `readLineOpt` migration and keep the voice-list workaround pending the existing #1576/B4 replay-policy track. No downstream game migration is claimed here; no `doc` alias or `readLine` behavior change is introduced.
