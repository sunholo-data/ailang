### Changed: checked pure signatures and derived interface purity (M1–M2)

- Breaking spelling change: `pure func` rejects declared labelled or open effect rows, including scoped Declassify rows. Remove `pure` for an effectful declaration, or use an empty row with a pure body. Existing absent and explicit empty rows remain valid.
- Interface JSON `pure` now derives from checked outer effect rows: labelled and declared-open rows report false; undeclared callback-shared tails remain latent. Non-function exports use a deep type walk. Serve-api/MCP use the interface signal, and compile cache v7 invalidates old always-true metadata.
- This partial sprint does not migrate the IFC source or teaching corpus or change SMT admission; M3–M4 remain pending. External packages such as `sunholo/mcp_oauth` using `pure` beside Declassify must migrate that spelling when the complete change ships.
