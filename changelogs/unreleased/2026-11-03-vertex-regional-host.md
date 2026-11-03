### Fixed — Vertex AI requests now go to the endpoint for their location (data residency) (2026-11-03)

The Gemini provider sent every Vertex AI request to the global host `aiplatform.googleapis.com`,
with the location only in the URL path. Google documents global endpoints as giving no regional
isolation or data-residency guarantee: the guarantee comes from the endpoint you call. So a
`GOOGLE_CLOUD_LOCATION=eu` deployment was not covered by the EU multi-region guarantee it was
configured for (docparse, 2026-11-03). The host now follows the location:

| Location | Host |
|---|---|
| `global` | `aiplatform.googleapis.com` (unchanged) |
| `eu`, `us` (multi-region) | `aiplatform.eu.rep.googleapis.com`, `aiplatform.us.rep.googleapis.com` |
| a region, e.g. `europe-west4` | `europe-west4-aiplatform.googleapis.com` |

This applies to `generateContent` and `streamGenerateContent`. A location that is not a plain
region name is refused as a provider error rather than spliced into a hostname. Checked live on
`gemini-3.5-flash-lite` in `eu`, both inline and `gs://` file input: a logging proxy saw only
`aiplatform.eu.rep.googleapis.com`. Regional deployments (`us-central1` and so on) now use the
regional host, which Google documents for regional processing. Unset location still means
`global`, as before.
