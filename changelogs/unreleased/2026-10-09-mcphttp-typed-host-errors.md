## Added

- **serve-api: typed host errors over MCP** (#1602). Errors from `Invoker.Invoke`
  implementing `protocol.JSONRPCError` now return their exact nonzero code and
  nonempty message as a per-message JSON-RPC error over HTTP 200 SSE. Wrapped
  errors are supported, and supported batches preserve sibling results. Untyped
  and malformed hooks retain the frozen whole-POST `-32603` envelope; timeout,
  cancellation and capacity mappings are unchanged.

Delivery to ailang-world requires a published AILANG release containing this
change and a World dependency pin update. This sprint does not publish a release.
