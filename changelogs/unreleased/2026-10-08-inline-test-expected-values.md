### Fixed — inline test expected values (Refs #495) (2026-10-08)

- Inline rows now evaluate list, tuple, record and local ADT expected values through
  the existing module-scoped harness evaluator, including nested combinations.
- Check, ai-check and LSP diagnostics reject unsupported row syntax on both arms
  with TST001/TST002. The runner validates before evaluation, and harness conversion
  returns errors instead of panicking on unsupported AST nodes.
- Row arithmetic remains excluded; use a named test block for elaborated expressions.
  Finding 3's composed-contract evidence remains bounded: the original 213-line
  sketch was not supplied; no SMT behavior is changed by this fix.
