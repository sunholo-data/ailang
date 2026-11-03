### Changed — Fleet scope guard: design-doc-creator as a support path (PROPOSED, D-FLEET-36 A)

PROPOSED, pending Mark's ruling on D-FLEET-36. `tools/launchd/githooks/pre-push` lets the fleet mission push `.claude/skills/design-doc-creator/**` and `.agents/skills/design-doc-creator/**` as support paths, not harness paths, so product loops keep pushing them unchanged. Guard arm only; the skill's eight grep sites change in a later fleet fire, after this lands (iteration-16 rule). `tools/launchd/test_mission_scope_guard.sh` adds allow, product-loop and prefix-trap arms.
