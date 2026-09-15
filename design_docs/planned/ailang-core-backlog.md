# AILANG Core Backlog (triaged reports)

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Effect checker resolves let-bound lambda to top-level function of same name | bug | direct-fix | The effect checker uses a different (non-lexical) symbol resolution than the type checker when attributing effects to calls, so shadowing a top-level function with a let-bound lambda falsely propagates the top-level function's effects — the fix is to share the type checker's resolver, no semantics decision involved (verified against v0.38.5-60-g0c6d9cec3 by reporter; no existing doc found: searched "effect checker", "shadowing", "lexical" in design_docs/). |
