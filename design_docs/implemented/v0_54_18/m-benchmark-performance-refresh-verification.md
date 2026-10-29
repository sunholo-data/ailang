# Benchmark performance refresh: verification and publication record

Implemented 2026-10-23 following the attended **execute sprint** instruction.
Independent evaluation: **PASS 95/100**, no remaining implementation blockers.
This is a local implementation verdict; full Go CI and production publication
have not run for this change.

## Review build

[Private performance page](https://voights-mac-studio.tail97eda0.ts.net:8451/docs/benchmarks/performance)
(both devices require Tailscale). HTTPS page and original logo returned 200.
The production Docusaurus build is served from an isolated preview directory;
its data-fetch origin alone points locally to the enriched saved JSON. Production
source continues to use the established live-dashboard-first/static-fallback
helper. Other Tailscale routes were preserved.

## Evidence preservation

- Added 39 model/mode/harness evidence rows from saved validated results only.
- Existing `latest.json` fields and all 61 history entries compare exactly after
  removing the new `benchmarkEvidence` block; original formatting also retained.
- Argon direct: 49/49 final, 48/49 first attempt, 23/23 current Core IDs;
  measurements still report v0.54.8 / 2026-10-13.
- Astra direct: 3/3, zero Core IDs, visibly provisional; its agent observations
  remain a separate configuration. Argon agent: 2/2 smoke results.
- Billing categories and timing boundaries remain explicit. Missing token
  telemetry is unavailable or a partial subtotal, rather than zero usage.
- No raw bank changes, paid probes, model calls or evaluation reruns.

## Gates

- `make test-changed` passed (changed analysis package plus language core).
- Full `go test ./internal/eval_analysis -count=1` passed after review fixes.
- `npm --prefix docs run test:benchmarks`: 11 tests passed; also wired into the
  docs build workflow. Render fixtures include the docs compiler's loose spread
  transform, which caught a browser-only Set conversion regression.
- `make lint`: zero issues after the final Go changes.
- File-size, changelog and architecture-boundary checks passed.
- Isolated `npm run build` passed with existing prompt/design/stdlib sync steps.
  Registry pages/sidebar were refreshed only in the isolated copy. A local
  node_modules symlink caused duplicated Docusaurus context during SSG; cloning
  dependencies into that copy resolved it. Existing deprecation/icon/update-check
  warnings did not fail the build.
- `git diff --check` passed.

Actual Chrome rendering exercised desktop 1440px and mobile 390px, light/dark,
both plots, shared history inspector, Core/all scopes, model detail, mode switch,
cost ordering, empty search, missing Python, and the real static fallback path.
Nine chart paths rendered; no page-wide mobile overflow or runtime exceptions.
Local screenshots and repeatable browser check are retained under
`design_docs/mockups/benchmark-redesign/verification/`; browser report and gate
logs are also available in `/private/tmp/benchmark-*` on this Studio.

Independent review found and resolved two data defects before passing: legacy
gallery agent coverage leaking into the direct Astra row, and missing token
telemetry displaying zero. No compiler/runtime semantics were changed.

## Publication handoff

Publication remains the final user decision, as specified in the approved plan.
The source, static artifact and review build are ready. When authorized:

1. Verify the active GitHub account and publish the reviewed commit through the
   established workflow, then observe full Go CI and the docs deployment.
2. Publish this same enriched `latest.json` through the existing benchmark data
   hosting workflow. The live dashboard wins over the static copy: deploying
   HTML alone would leave the live endpoint on its legacy schema. Preserve its
   saved history and any newer banked observations; refresh offline if needed.
3. Verify the rendered public performance route, Argon/Astra, original logo,
   plots and live endpoint schema. Confirm no local-only fetch-origin change
   entered the production source.

No new model evaluations are needed for any of those publication steps.
