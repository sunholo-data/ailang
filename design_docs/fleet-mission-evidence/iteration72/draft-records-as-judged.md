# Fleet iteration 72 — draft records handed to the independent judge

Base: `origin/dev` = `490fb730b2f809f846c2c9b437b2aefcdfc12933` (Gate-1 read 2026-10-30T12:04:58Z).
Controller: `claude:claude-opus-5-5`. No code is proposed. Both records are bookkeeping:
one ghost resolution and one decision row.

## Record 1 — resolve `skills:showcase-feature-discoverability` as a ghost of #1691

Ticket (attended, filed 2026-10-03T09:00Z, message `inbox_1791018014090_80cc2f04`) restates
GitHub #476 with three asks: (1) wire the docs MCP by default in mission bootstrap; (2) make
sprint-executor load AILANG syntax (use-ailang) as a mandatory first step; (3) add a CI
feature-usage gate for showcases. Its evidence line: "Checked 790169359: sprint skills still have
no mandatory syntax step."

Claims:
- C1. Ask 2 is built. `03263525c` (#1691, 2026-10-09, 6 days after the ticket) added Core
  Principle 11 to `.claude/skills/sprint-executor/SKILL.md:104` ("Load AILANG syntax before
  writing `.ail` — no exceptions": run `ailang prompt` before the first `.ail` edit each session,
  record the version, discharge the contracts/effects/inline-tests checklist) and the AILANG Syntax
  Gate to `.claude/skills/sprint-planner/SKILL.md:251-254`. Details are in
  `.claude/skills/sprint-executor/resources/milestone_checklist.md:122-133`. `03263525c` is an
  ancestor of `origin/dev`. Both `.agents` mirrors `cmp` identical.
- C2. Asks 1 and 3 were declined by the approved design, not left unbuilt by accident.
  `design_docs/implemented/v0_53_1/m-skill-feature-discoverability-gate.md` § Non-Goals
  (`:281-291`): MCP / mission-bootstrap registration is "issue ask (1)… owned by the mission
  charter/bootstrap template", and "A CI hard gate — issue ask (3)… Deferred". The doc's Status is
  "Implemented in v0.53.1 (03263525c; closes #476)".
- C3. The source issue is closed with that verdict. `gh issue view 476`: CLOSED
  2026-10-26T13:48:43Z, one comment (same second) from an attended triage: "Addressed by
  03263525c (v0.53.1, #1691)… The design ruled MCP wiring and a CI gate as non-goals. verified
  against origin/dev 40fbb90d5".
- C4. Asks 1 and 3 are outside fleet authority anyway: ask 3 is CI workflow wiring in the showcase
  repo (ailang-world), and CI workflows are refused by the fleet scope guard.
- Proposed resolution text: "Ghost of #1691 (`03263525c`, 2026-10-09): sprint-executor Core
  Principle 11 and the sprint-planner AILANG Syntax Gate make `ailang prompt` mandatory before any
  `.ail` write (ask 2). Asks 1 (MCP in bootstrap) and 3 (CI feature gate) are Non-Goals of the
  approved design M-SKILL-FEATURE-DISCOVERABILITY-GATE; #476 closed 2026-10-26 by attended triage
  with that verdict." `--sha 03263525c<full>`.

## Record 2 — park `pi:openrouter:uncapped-max-tokens-402` as decision D-FLEET-35

Ticket (stapledon, filed 2026-10-07T18:58Z, message `inbox_1791399484944_a083ef1e`). Two asks:
(a) whatever provisions pi models should write a capped entry for every routed OpenRouter model;
(b) the 402 "fewer max_tokens" text should classify as config, not capacity. Workaround already
applied (attended 2026-10-07): `maxTokens 32000` entries in `~/.pi/agent/models.json` for the
three models; backup `models.json.bak-20261007-orcaps` exists.

Claims:
- C5. Ask (a) is real at HEAD, at repo level. The repo's canonical pi config,
  `tools/pi-extensions/models.mission.json`, has no `openrouter` entry for `z-ai/glm-5.3`,
  `moonshotai/kimi-k3` or `minimax/minimax-m3` (its openrouter models: gemma-4-26b, glm-5.3-flash,
  three deepseek-v4-flash-0731/v4.1-flash spellings, v4.1-flash-1017). The installed
  `~/.pi/agent/models.json` does carry all three at 32000. So the caps live only on this rig.
- C6. The three models are still routed. This fire's driver env:
  `MISSION_DESIGNER_FALLBACK` ends `…,pi:openrouter/z-ai/glm-5.3,pi:openrouter/moonshotai/kimi-k3`,
  `MISSION_PLANNER_FALLBACK=…,pi:openrouter/moonshotai/kimi-k3`, `MISSION_EVALUATOR_FALLBACK=pi:openrouter/minimax/minimax-m3,…`.
  `~/.config/ailang/mission-*.env` also names `pi:openrouter/z-ai/glm-5.3` (designer) and
  `pi:openrouter/minimax/minimax-m3` (evaluator). The controller rung for OpenRouter GLM was
  retired by Mark on 2026-10-07 (option B, `tools/launchd/mission-control.sh:929-936`), so these are
  role fallbacks only.
- C7. The fix for ask (a) is outside fleet authority. The file is `tools/pi-extensions/models.mission.json`
  (only `tools/pi-extensions/sandbox/**` is in scope, D-FLEET-6), and it is held equal to
  `internal/modelreg/models.yml` by `TestPiModelsConfigMatchesRegistry`
  (`internal/eval_harness/pi_models_config_test.go:63`), also outside scope. Nothing in fleet scope
  writes `~/.pi/agent/models.json`: `git grep models.json` over `tools/launchd`, `internal/mission`,
  `cmd/ailang/mission*`, `missions`, `scripts/mission_*` finds only comments.
- C8. Ask (b) is policy. `RUNTIME_QUOTA_SIG` (`mission-control.sh:1152`) carries `^402:` as a
  capacity signal under D-FLEET-11 = A and D-FLEET-13 = A. Reclassifying a 402 subtype changes
  runtime classification, which is HD-2a policy (the D-FLEET-26 pattern).
- C9. No recurrence in retained logs. `grep -c 'can only afford'` = 0 in every
  `/tmp/ailang-mission-*.log` (retained since 2026-10-27 08:59). The one `^402:` hit
  (`ailang-mission-triage.log:16093`) is a line-number prefix inside quoted skill text, not a 402.
  Positive control: `grep -c 'quorum artifact' /tmp/ailang-mission-world.log` = 48.
- Proposed D-FLEET-35 (OPEN): "`pi:openrouter:uncapped-max-tokens-402` (stapledon, 10-07): the caps
  for the three routed OpenRouter models live only in the rig's installed pi config. Who adds them
  to the repo, and should a 402 'can only afford' read as config?" Recommendation **A**: the owner of
  `tools/pi-extensions/` + `internal/modelreg/models.yml` (V1/attended) adds registry rows and
  `models.mission.json` entries for `z-ai/glm-5.3`, `moonshotai/kimi-k3` and `minimax/minimax-m3`
  with `max_output_tokens` 32000 (the installed value), so the drift test holds them; keep `^402:` as
  capacity (no reclassification; with the caps in the repo the oversized request cannot be formed,
  and no 402 has occurred since 10-27). Then the fleet resolves the ticket. Alternative **B**: also
  reclassify "can only afford" as config. Default until answered: ticket stays open, nothing changes.
