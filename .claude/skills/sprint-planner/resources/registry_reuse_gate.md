# Registry reuse gate (sprint-planner step 0)

Before decomposing implementation, run `ailang pkg search <keywords>` for every package-like capability. Inspect candidates with `ailang pkg info <vendor/name>` and `ailang pkg docs <vendor/name>`. Classify every implementable milestone as `depend` (reuse and pin the package), `contribute` (extend the existing package via its `pkg:<name>` inbox), or `none` (record searches and why fresh code is required). Persist populated decisions in the plan and `registry_reuse` sprint JSON; placeholders block handoff.

`scripts/create_sprint_json.sh` writes a placeholder `registry_reuse` row; `sprint-executor/scripts/validate_sprint_json.sh` rejects placeholders, so an unfilled audit cannot be handed off. Sprints created before this gate have no `registry_reuse` field and only get a warning.

Example row:

```json
{"milestone": "M1", "package": "sunholo/example", "action": "depend", "reason": "Capability already exists"}
```
