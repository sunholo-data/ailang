### Fixed — vulnerability gate green: x/net v0.62.0, Ollama v0.40.13, allowlist re-verified

- `golang.org/x/net` v0.61.0 → v0.62.0 fixes GO-2026-6860 (quadratic text-node coalescing in `html.Parse`, reachable through the `std/html` parse builtins) and GO-2026-6866.
- `github.com/ollama/ollama` v0.40.9 → v0.40.13, the newest published release.
- The nine Ollama allowlist entries expired on 2026-10-29. Re-verified with the same three checks as on 2026-07-31: these are exactly the reachable set, none has an upstream fix, and Ollama is on its newest release. Re-armed until 2027-01-27, with the evidence recorded in `.govulncheck-allow.yml`.
