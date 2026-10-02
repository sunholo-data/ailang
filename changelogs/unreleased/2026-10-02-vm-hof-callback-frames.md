### Changed — bytecode VM: HOF callbacks no longer allocate; `bytecode.Value` packed to 40 bytes (#1501, 2026-10-02)

This is Phase 3 of [m-foldl-cons-cost-model](../../design_docs/implemented/v0_51_1/m-foldl-cons-cost-model.md),
done profile-first. The profile and its tables are in the doc's Implementation Report.

- **Frame recycling.** The VM keeps a per-VM LIFO free list of frames. Every push (entry, `CallClosure`,
  `CALL`) reuses a frame from it, and `RETURN` gives the frame back with its registers cleared, so a
  pooled frame never keeps a value alive. A frame left behind by an error is never reused. HOF builtins
  (`map`, `filter`, `foldl`, `foldr`, `mapAccumL`, `any`, `findIndex`, `flatMap`, `sortBy`, the string
  folds and the XML fold) now pass one argument buffer for all their callbacks.
  `go test ./internal/vm/ -bench HOFFoldl`, per 1,000 callbacks: before, 3,000 allocations and 104.8 µs;
  after, 1 allocation and 33.2 µs.
- **CLI wall time on `--bytecode`, min of 5 runs.** `foldl(\acc x. acc + x)` over 5,000,000 elements went
  from 0.60 s to 0.34 s. `filter`/`map` with closures over 5,000,000 elements went from 1.28 s to 0.67 s.
  On these shapes the VM is now 20–30× faster than the evaluator.
- **`bytecode.Value` is 40 bytes, not 48.** `Bool` now sits next to `Tag`, and `TestValueSize` pins the
  size. On the copy-bound consrepro (`foldl(\acc x. x :: acc)`), the VM allocates 19% fewer bytes. At
  40,000 elements the VM's time went from 2.05× the evaluator's to 1.80×.
- **Fixed:** when a callback faulted, `CallClosure` left its frame on the VM stack, so every caught
  callback error used up one frame of `MaxStack`. The stack is now restored to its depth before the call.
- **Not fixed: the remaining consrepro gap.** The profile puts more than 99.9% of it on the bytes `::`
  copies. A VM value is 40 bytes, against the evaluator's 16-byte interface, so the VM runs more GC
  cycles. Removing that needs the cons representation change (D-19). `docs/LIMITATIONS.md` records it.
  Benchmarks: `bench/vm_hof_callbacks/run.sh`.
