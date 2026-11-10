### Breaking — `run --bytecode --verify-contracts` is refused (CTR004) instead of checking nothing (#1956)

The bytecode VM never consulted the contract checker, so `ailang run --bytecode
--verify-contracts` (and `--strict-bytecode`, and either under `--batch`) executed with no
`requires`/`ensures` checked and exited 0, while the evaluator enforced the same contract.
The combination is now refused before the file is read, with exit code 2:

```
Error: CTR004: --verify-contracts is not enforced on the bytecode VM yet (runtime contracts run only on the evaluator). Drop --bytecode to check contracts, or drop --verify-contracts to run on the VM. Tracking: #1956.
```

It does not fall back to the evaluator: you asked for the VM. Migration: drop `--bytecode`
to check contracts, or drop `--verify-contracts` to run on the VM unchecked (what happened
before, silently). The refusal is lifted when the VM enforces contracts
(M-RUNTIME-CONTRACT-PARITY M6–M7), which is additive.

`run --help` now says what was only in the guide: `run` does not check contracts unless
`--verify-contracts` is given. That default is stated as Stable in `stability.md`.
