# Token-cost eval task

You are working in the AILANG repo at `examples/lsp_xref_fixture/`.

## Task

Add a new exported pure function to `a.ail` named `subtract` with signature `(int, int) -> int` that returns `x - y`. Then update `b.ail` to call it from a new exported pure function named `use_subtract` (parallel to the existing `use_add`).

## Acceptance

- `a.ail` exports both `add` and `subtract`
- `b.ail` imports both, and exports `use_add` (unchanged) AND `use_subtract`
- All four functions must be syntactically and type-correctly valid AILANG (running `ailang check` on either file should report no errors)

Stop as soon as the change is complete and the files type-check. Do not refactor anything else, do not add tests, do not edit any file outside `examples/lsp_xref_fixture/`.
