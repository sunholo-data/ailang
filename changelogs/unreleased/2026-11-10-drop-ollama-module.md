### Changed — the Ollama client uses local wire types; `github.com/ollama/ollama` is no longer a dependency (2026-11-10)

`internal/ai/ollama/wire` replaces `github.com/ollama/ollama/api` (v0.40.15) for the native
`/api/generate`, `/api/chat`, `/api/embed` and `/api/tags` calls made by the Ollama provider,
`std/ai`'s `_ollama_embed` / `_ollama_embed_media` builtins and the messaging embedder. The SDK
linked seven ollama packages into the binary and carried eight reachable, unfixed govulncheck
advisories plus one module-level one; all nine allowlist entries are deleted.

Parity is pinned by goldens captured from the SDK before the swap (69 scenarios across
`internal/ai/ollama/testdata/wire`, `internal/builtins/testdata`, `internal/messaging/testdata`):
request paths, headers and body bytes, streamed-reply decoding and every error text callers classify
are byte-identical, including the SDK's lossy tool-schema normalisation, document-order tool
arguments and `OLLAMA_HOST` resolution. One behaviour is dropped: the SDK signed requests to a
direct `ollama.com` host with the `ollama signin` device key. The client now refuses that host with
an error naming `OLLAMA_HOST`; reach Ollama Cloud through the local daemon, as AILANG already does.
`OLLAMA_AUTH` (SDK request signing for a self-hosted server) is no longer honoured.
