### Fixed — runtime contract violations name the function

`ailang run --verify-contracts` reported `contract violation: ensures failed in  at k1.ail:5:11`
with an empty function name, and trace `contract_check` events carried the same empty name:
the name was in scope where contracts are attached to a function value and was dropped there
(`ContractContext.SetFunction` had no production caller). The message now reads
`contract violation: ensures failed in half at k1.ail:5:11: (result >= 0)`, and the trace event
carries `half`. The contracts guide's example message is corrected to the real one.
