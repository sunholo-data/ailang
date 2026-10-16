### Fixed — long list literals and else-if chains fell off the strict bytecode VM (#1736) (2026-10-16)

Two code shapes made a function evaluator-only, which `--strict-bytecode` reported only at run
time (`register allocator: contiguous block of N would exceed 256`):

- **List literals longer than about 256 elements** (a 601-entry LUT, a crew-names table).
  A literal now compiles as 64-element `MAKE_LIST` chunks joined with `_concat_List`, so
  register pressure no longer grows with its length.
- **Long `if … else if …` chains over a wide record** (16+ fields, about 15+ arms). The
  lowering binds each level's condition, and in value position its result, to compiler temps
  whose scope encloses the rest of the chain, so every level pinned registers inside the
  previous arm's freed block until a record rebuild found no contiguous run. The compiler now
  tests the condition directly and writes each level's result into the enclosing result
  register, so a chain of any length uses constant registers.

Both shapes match the interpreter on `--bytecode --strict-bytecode` (tests up to 601 elements
and 80 arms; 2000 elements and 120 arms checked by hand). Stapledon's workarounds (chunking
literals to 100, `applyOverride2`) can come out. Still open: Phase A of
`m-bytecode-strict-256-reg-ceiling` (an `ailang check --strict-bytecode` gate that reports any
remaining evaluator-only function before run time).
