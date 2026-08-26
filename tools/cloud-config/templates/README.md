# Cloud coordinator prompt templates

Source of truth for the templates the cloud coordinator's package agents load at
dispatch time from `gs://<project>-ailang-config/templates/` (mounted read-only
at `/etc/ailang-config/templates/` via gcsfuse; referenced by each agent's
`invoke.template_file` in the cloud config).

These existed ONLY in the bucket until 2026-08-26 — unversioned, unreviewed, and
drifting: `pkg-feedback.md` was last touched 2026-04-28 and still described every
package as living in the ailang-packages monorepo, which became false the day the
`ailang_parse` agent was pointed at its own repo. Audited during the
mission-loop/coordinator reconciliation (see
design_docs/planned/v0_35_0/m-pipeline-reconciliation.md).

Edit HERE, then sync:

```bash
gsutil cp tools/cloud-config/templates/pkg-update.md tools/cloud-config/templates/pkg-feedback.md \
  gs://ailang-multivac-ailang-config/templates/
```

Note the main pipeline agents (design-doc-creator, sprint-planner,
sprint-executor) do NOT use templates — they are `invoke: type: skill` and load
the repo's `.claude/skills/`, which is the preferred shape. A template is only
warranted where no skill exists (today: the cascade-repair directive, which is
filled with wrapper-computed variables no skill receives).
