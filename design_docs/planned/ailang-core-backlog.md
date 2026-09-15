# AILANG Core Backlog

Triage rows from ailang-core reports. One row per report; Recommendation drives routing.

| Date | Title | Class | Recommend | Why |
|---|---|---|---|---|
| 2026-09-15 | Gemini Vertex client ignores region: global host accepted any location segment | bug | direct-fix | Verified: `buildURL` (AuthADC branch, internal/ai/gemini/generate.go:272) always uses the global `aiplatform.googleapis.com` host with location in path only, which Vertex answers without regional routing — data-residency via `GOOGLE_CLOUD_LOCATION` is illusory; fix is to use `https://{location}-aiplatform.googleapis.com` when location != "global" (report located the change; no design doc needed). Search terms "aiplatform", "GOOGLE_CLOUD_LOCATION": no planned doc covers regional host enforcement — nearest, `implemented/v0_30_0/m-gemini-exec-project-plumbing.md`, only plumbs the env var through. |
