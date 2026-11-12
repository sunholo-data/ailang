### Fixed — security alerts: zod, go-jose, x/crypto, workflow token scope and action pins (2026-11-12)

Workstream 1a of the v1.0 codebase-health handover
(`design_docs/planned/HANDOVER-v1-codebase-health-2026-11-09.md` §2d).

- **zod (Dependabot #267, high, GHSA-mrhx-c7xc-m3w5).** `tools/pi-extensions/sandbox` now
  overrides the transitive `zod` to `^4.6.6` (locked at 4.6.7). `@anthropic-ai/sandbox-runtime`
  stays at exactly 0.0.71, because the maintenance fence pins that version, and every release of it
  still asks for zod `^3.25`, which has no patched version. On the canonical mission policy and 8
  valid/invalid probe configs, `SandboxRuntimeConfigSchema.safeParse` gives the same result and the
  same issue paths and codes under zod 3.25.76 and 4.6.7.
- **Go modules (Scorecard Vulnerabilities, high).** `github.com/go-jose/go-jose/v4` goes from
  v4.1.4 to v4.1.9 (GHSA-4w3g-jfj2-cw2q, GHSA-mgcc-cww5-fp2q, GHSA-vvm8-gf78-28g4).
  `golang.org/x/crypto` goes from v0.58.0 to v0.59.0 (GO-2026-5932). `go mod tidy` also moves
  x/sync to v0.25.0, x/term to v0.48.0 and x/text to v0.44.0. All of these are indirect except
  x/term and x/text.
- **docs (Scorecard Vulnerabilities).** `webpack-dev-middleware` goes from 7.4.5 to 7.4.6 in
  `docs/package-lock.json` (GHSA-g84c-rxfj-3j2c). It is a dev-server dependency, and the lockfile
  change is that one entry.
- **Token permissions (Scorecard TokenPermissions, high).** `dependabot-automerge.yml` has a
  read-only top-level token now. `contents: write` and `pull-requests: write` move onto the one job
  that runs `gh pr merge --auto`.
- **Pinned actions (Scorecard PinnedDependencies, medium).** Every `uses:` in
  `.github/workflows/` is now a commit SHA with a `# vX.Y.Z` comment, and Dependabot's
  github-actions ecosystem updates both. The exception is the SLSA reusable workflow, which must be
  referenced by tag. Triggers, job names and required check names are unchanged.
