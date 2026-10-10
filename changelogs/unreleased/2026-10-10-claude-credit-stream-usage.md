### Fixed — Claude credit gateway usage contract

- Replace cumulative streaming input/cache totals when present; preserve previous
  observations on optional null/omitted fields. Price `output_tokens` once as the
  inclusive total, with its documented thinking breakdown used only for validation.
- Reject null mandatory billing counters, invalid TTL splits and unknown billing
  categories. Failed validation retains the full reservation and blocks admission.
- Emit bounded billing-only failure diagnostics with internal/provider request IDs,
  usage observations and completion state. Prompts, credentials and arbitrary
  provider data are excluded. Diagnostic observations never authorize settlement.
- Record the first failed dev pilot: admission disabled, $1.32 unresolved exposure
  preserved pending provider reconciliation; no regular rotation promotion.
- Design: `design_docs/planned/m-cloud-claude-api-credit-guard.md`.
