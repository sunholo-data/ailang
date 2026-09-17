# Gemini Vertex client cannot enforce region: global host accepts any location segment

- **Date**: 2026-09-15
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `aiplatform`, `GOOGLE_CLOUD_LOCATION`, `vertex|regional|europe-west|data residency` across design_docs/. Nearest coverage is `implemented/v0_30_0/m-gemini-exec-project-plumbing.md`, which only plumbs the env var through and explicitly pins location to `"global"` (`internal/executor/managed_agents/client.go`, `defaultLocation`). No planned or implemented doc rules on regional host enforcement — this is a genuine gap.
- **Estimate**: ~2 lines in `internal/ai/gemini/generate.go` (both `buildURL` and `buildStreamURL` AuthADC branches: when `c.location != "" && c.location != "global"`, use `https://{location}-aiplatform.googleapis.com/v1` instead of the global `vertexAIBaseURL`; the constant lives in `internal/ai/gemini/client.go`)

The Vertex global host (`aiplatform.googleapis.com`) accepts any location path segment and
answers without routing to that region — measured 11 Sept with ADC on aitana-multivac-dev:
`locations/nowhere-9`, `locations/europe-west4`, and `locations/global` all returned HTTP 200
from the same global host, while the regional host `europe-west4-aiplatform.googleapis.com`
also answers and is the only one that enforces residency. So `GOOGLE_CLOUD_LOCATION=europe-west4`
with `--ai gemini-2-5-flash` *looks* like EU processing but is not; data residency via that env
var is illusory for anyone relying on it today. Consumer impact: Daneel (Mark's EU-processing
requirement) is currently forced to bypass the AI effect and call the regional host via Net
domain-allowlisting; the interpreter cannot pin regions until the client does. Reporter located
the change; it is a one-line base-URL swap in the AuthADC branch (streaming variant included),
with no semantics decision — regional host is exactly what Vertex documents for non-global
locations. The edit itself is small (≤2 lines), but the Estimate names two files — `generate.go` carries the change and `client.go` holds the base-URL constant — which rubric row 5 puts over `DIRECT_FIX_MAX_FILES`, so this lands as design-doc.
