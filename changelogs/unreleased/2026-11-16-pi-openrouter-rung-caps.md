### Fixed — pi: the driver's routed OpenRouter rungs have caps in the repo (2026-11-16)

D-FLEET-35 A (Mark, attended). The mission driver routes three OpenRouter models through pi
as role fallbacks: `z-ai/glm-5.3` (designer), `moonshotai/kimi-k3` (designer and planner)
and `minimax/minimax-m3` (evaluator). Their output caps existed only in the rig's installed
`~/.pi/agent/models.json` (the attended 10-07 workaround for the uncapped-request 402). A
model missing from that file makes pi fall back to its built-in catalogue and send an
uncapped request.

- `tools/pi-extensions/models.mission.json` gains `openrouter` entries for all three at
  `maxTokens` 32000, transcribed from the installed config. `thinkingLevelMap.off` is `null`,
  as `TestPiModelsConfigMatchesRegistry` requires. The installed kimi-k3 entry maps
  `off: "none"`, and minimax-m3 has no map.
- New registry row `pi-or-glm-5-3` (32000). `pi-or-kimi-k3` and `pi-or-minimax-m3` already
  existed at 32000. `z-ai/glm-5.3` joins `unequalWireBudgets` beside its two siblings, because
  `or-glm-5-3` declares 65536.
- Wire check on pi 0.85.1 (`scripts/check_pi_wire_budget.sh`): kimi-k3 and minimax-m3 both
  sent `max_completion_tokens` 32000. PASS.
- New guards:
  - `TestDriverPiOpenRouterRungsAreHeld` requires every `pi:openrouter/*` rung in the driver's
    fallback exports to have a pi registry row and a canonical entry with one cap.
  - `TestPreflight_HoldsDFleet35Rungs` passes the canonical config and refuses an installed
    file without the model.
- `^402:` stays classified as capacity (`RUNTIME_QUOTA_SIG` unchanged).
