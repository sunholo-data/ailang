### Fixed — imported polymorphic functions no longer depend on unrelated code (#2157)

Exported functions over open records (e.g. `std/sem.is_newer`) were exported with their field type
and record row left free: the generalisation collector skipped open records. An importer's own
fresh type variables, numbered from the same per-module counter, then collided with them, so a
valid program was rejected after an unrelated edit, and two legal uses of one imported function at
different field types conflicted (`No instance for Num[string]`). Generalisation now walks open
records and record rows, record rows are quantified and instantiated as record rows, and every
exported scheme is closed (std + examples: 11 unclosed exports → 0). A second, older collision is
gone too: the row unifier and the inference context both minted `ρN` names from separate counters,
which rejected some valid programs that use open record patterns (`{field, ...}`) with
`kind mismatch: row variable ρ7 has kind Row Effect`. Design:
`design_docs/planned/v1_0_0/m-v1-tyvar-hygiene-and-digest-determinism.md`.

### Fixed — module interface digests are deterministic (#2076)

Type variables in stored interfaces are renamed canonically (`'t0`, `'r0`, `'e0`; a free variable
would be `'f0`) in order of first appearance, and the checker allocates fresh variables in sorted
order (record literals, record updates, record patterns, generalised type variables). `iface.json`
is byte-identical across cold runs (`std/sem`: 8 digests in 20 runs → 1) and no longer changes when
a private declaration is added. `interface_hash` (v1 and v2) and `.stdlib-golden/` were already
deterministic and are unchanged; the compile-cache key format is unchanged. Importers of `std/sem`
stop recompiling on cold caches.
