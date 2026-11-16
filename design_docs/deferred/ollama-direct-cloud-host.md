# Direct `ollama.com` host and request signing (nice to have)

**Status:** Deferred — not planned. Recorded so the option is findable if a need appears.
**Created:** 2026-11-10 (Mark, attended: "I don't think we use it so we can move it to a nice to have
design doc note in case it comes up in the future")
**Origin:** PR #2206, which replaced the `github.com/ollama/ollama` Go module with local wire types
(`internal/ai/ollama/wire`). Roadmap: [planned/v1_0_0/ROADMAP.md](../planned/v1_0_0/ROADMAP.md) §7.

## What was dropped

The retired SDK could talk to Ollama Cloud **directly**, with no local daemon:

1. `OLLAMA_HOST=ollama.com` (or `https://ollama.com`) pointed the client at the hosted API.
2. Requests to that host were **signed** with the device key created by `ollama signin`.
3. `OLLAMA_AUTH` turned the same request signing on for a self-hosted server.

The local client does none of this. `wire.NewClient` refuses the host with an error that names
`OLLAMA_HOST` and says to use a local daemon; `OLLAMA_AUTH` is not read.

## What still works

Everything AILANG uses today. Cloud models are reached **through the local daemon**, which proxies
`-cloud` models to `ollama.com` on the device key — no API key and no signing in our code. That is the
path the provider factory, the mission quorum caller and the eval lanes take. The quota probe
(`internal/mission/ollama_quota.go`) and the web-search effect (`internal/effects/web.go`) call
`ollama.com` over their own plain HTTP and were never routed through this client.

Verified on the rig at the time of the swap: a chat through `glm-5.3-flash:cloud` via the daemon
returned normally on a binary built from the PR branch.

## When this would matter

- Running AILANG somewhere **without an ollama daemon** — a slim container, a Cloud Run job, a CI
  runner — and wanting Ollama Cloud models there. Today that needs the daemon installed and signed in,
  or another provider route for the same model (OpenRouter is the existing fallback).
- A self-hosted ollama server that requires signed requests.

Neither is a current use. Nobody has asked for it.

## If it is ever picked up

- **Scope:** request signing in `internal/ai/ollama/wire` (key discovery, the signature over method,
  path and timestamp, the `Authorization` header), lifting the `ollama.com` refusal, and honouring
  `OLLAMA_AUTH`. The exact signing scheme must be re-read from the ollama source at the time — it is
  not recorded here, on purpose, because it is theirs to change.
- **Reading a private key from the user's home directory** is new surface for the binary. It needs an
  explicit opt-in and a line in the security-model page; it must not happen merely because
  `OLLAMA_HOST` names the hosted API.
- **Any env var goes through the `internal/config` registry.**
- **Parity cannot be tested the way the swap was.** The SDK is gone, so there are no golden bytes to
  compare against; it needs a recorded exchange against the real service, or the SDK pinned as a
  test-only dependency (which would bring the vulnerable module back into `go.mod`).
- **Consider the alternative first:** if the need is "cloud model, no daemon", routing that model
  through OpenRouter or the provider's own API may be less code and no key handling.

## Size

Small — a few days — but only worth doing for a named user.
