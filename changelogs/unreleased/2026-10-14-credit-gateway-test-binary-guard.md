### Fixed — a test run inside a credit-lane job could end that job's credit lease

Cloud agents on the Claude API-credit lane carry their live gateway capability in the
environment, and the evaluator runs `go test` there. On 2026-10-12 `go test ./cmd/ailang/...`
reached `coordinatorExecuteJob`'s deferred lease release with that capability; the gateway
released the running task and the evaluator's next request was refused with HTTP 402
(task-f257f24d, failed after 11 minutes). `CloudCreditIdentityToken` now refuses inside a
test binary (`ErrCreditGatewayFromTest`), so neither admission nor release can reach a real
gateway from a test; tests keep passing a stub token.
