### Fixed — five runnable examples export `main` again (dev red since b6c9f592e)

- `run --entry main` on a non-exported `main` now fails loudly (b6c9f592e), and five examples in
  `examples/runnable/` still declared `main` without `export`, so `verify-examples` failed on every dev
  push from 03e53668c on. `adt_list_fields`, `if_else_chain`, `poly_adt_result_type`, `record_in_match`
  and `record_width_subtyping` now export it; `make verify-examples` is 248 passed, 0 failed.
