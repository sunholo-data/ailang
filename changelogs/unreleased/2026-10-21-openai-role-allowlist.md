### Fixed — mission roles can spawn only gpt-6.1-sol among OpenAI models (D-FLEET-12): codex launches hide spawn_agent's model choice, mission codex-exec refuses other models, quorum seating refuses them, and a routing-evidence check flags them

Native child overrides are hidden on driver and skill launches; managed Codex execution validates the parent model before daemon contact.
