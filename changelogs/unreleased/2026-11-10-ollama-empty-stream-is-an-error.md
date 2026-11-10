### Fixed — an empty Ollama streaming reply is an error, not an empty answer (2026-11-10)

A `/api/generate` or `/api/chat` reply with no lines at all used to return success with empty text
and zero tokens, whatever the HTTP status. An HTTP 500 with an empty body was therefore banked as
"the model said nothing". It is now an error at every status: a `StatusError` carrying the status
for 4xx/5xx (`ollama error: 500 Internal Server Error: empty response body`), an
`AuthorizationError` for 401, and `wire.ErrEmptyStream` for a 2xx reply, which is a truncated
stream, since both endpoints always end with a done chunk.

This is the one deliberate departure from the retired SDK's behaviour; the wire-parity golden for
that scenario (`err_500_empty_body`) is updated and marked. Callers that treated an empty result as
a model failure will now see a provider error instead, classified `Internal` and retried like any
other 5xx.
