### Fixed — serve-api: refused tool calls carry `error` and `error_description` for ChatGPT (2026-11-12)

When the lazy-auth Bearer gate on `/mcp/connect/` refuses a `tools/call`, the
`_meta["mcp/www_authenticate"]` value now includes `error` and `error_description`.
OpenAI's Apps SDK auth guide says the value must contain both, and that this payload "is what
actually triggers the tool-level OAuth UI" in ChatGPT. Until now the value copied the HTTP
header, which has no `error` when the request carried no token (live on AILANG Parse,
2026-11-12).

- Without a token: `error="insufficient_scope"` (OpenAI's own example for "no access token
  provided") and `error_description="Sign in to use this tool."`.
- With a rejected token, `error="invalid_token"` is kept and a description is added.
- The HTTP `WWW-Authenticate` header is unchanged. It still follows RFC 6750 §3.1, with no
  error code when there is no token, and Claude starts sign-in from it.
- `protocol.MetaChallenge`. Covered by `TestMetaChallenge`, and by the two existing challenge
  tests, which now also require both parameters (both fail without the fix).
