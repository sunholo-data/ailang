## Fixed

- Raise the bytecode VM's default frame limit from 1,000 to 10,000 and apply positive `--max-recursion-depth` overrides to `ailang run --bytecode`, matching the evaluator's default ceiling. Deep service handlers within that ceiling now finish without evaluator replay dropping consumed-input results or duplicating earlier output. Non-strict fallback after an above-limit error can still replay effects; that policy remains separate work. Refs #1576.
