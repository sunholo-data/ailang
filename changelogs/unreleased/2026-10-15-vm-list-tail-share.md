### Fixed — bytecode VM: `x :: rest` list walks are O(n), not O(n²) (#1827)

`_list_tail` (every cons/rest pattern), `take` and `drop` on the VM copied the
suffix on each call, so recursion that walks a list with `x :: rest` allocated
O(n²). They now return a view sharing the source's storage (lists are
immutable; capacity is capped so an append can never write into the source),
matching the interpreter and Go codegen. Reporter's repro: 6.8 s → 1.05 s,
identical output. M-VM-LIST-TAIL-SHARE Phase 1; the `--strict-bytecode`
change (Phase 2) and `bench/vm_tail_walk` are still open.
