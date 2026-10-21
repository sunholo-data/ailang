### Fixed — mission roles can no longer spawn retired OpenAI models (D-FLEET-12): only gpt-6.1-sol and gpt-6-luna are allowed; codex launches hide spawn_agent's model choice, mission codex-exec refuses other models, quorum seating refuses them, and a routing-evidence check flags them

Native child overrides are hidden on driver and skill launches; managed Codex execution validates the parent model before daemon contact. gpt-6-luna stays allowed because the triage charter (ratified 2026-10-15) routes its verifiers to it; D-FLEET-27 asks Mark to confirm.
