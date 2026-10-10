# Claude credit gateway: CLI compatibility evidence

**Reviewed:** 2026-10-10. **Status:** installed-client completion and tool-loop compatibility passed locally; production activation remains blocked by the deployment checklist. This records a client configuration and bounded API subset, not approval to forward an undocumented beta.

## Decision

For the guarded API-key lane, configure the reviewed Claude Code client with `CLAUDE_CODE_SIMULATE_PROXY_USAGE=1` alongside the existing experimental-feature restrictions. Keep gateway rejection of `claude-code-20250219` and unproved body fields. The client constructs a narrower request; the gateway must not strip beta headers or rewrite an otherwise admitted request.

Anthropic's [gateway compatibility guide](https://code.claude.com/docs/en/llm-gateway-protocol) recommends forwarding beta/version headers and body fields unchanged, and warns that header/body feature pairs must travel together. Its general compatibility recommendation to accept evolving fields is insufficient for a hard credit ceiling: this gateway intentionally admits only a reviewed billing contract. Features outside that contract fail before provider dispatch.

## Installed official client evidence

The inspected local native distribution identifies itself as Claude Code **2.1.295**:

- Artifact: `/Users/voightkampff/.local/share/claude/versions/2.1.295` (the local `claude` symlink target).
- SHA-256: `0116ee2e0a513900b633d9951367f18747686478e2b462805b8c31609f047f70`.
- Inspection read the binary and its embedded JavaScript only; it did not read authentication configuration or real provider keys.

The embedded JavaScript declares the switch as a Boolean environment option. Its seven UTF-8 literal occurrences are one interned string, one declaration and five executable uses. The query builder at byte offset **197216498** filters the request beta list down to an OAuth-specific beta when the switch is enabled; in the API-key path the resulting list is empty. Adjacent conditions disable specialized reuse, feature planning, selected beta-dependent cache/system/thinking-update construction and extra feature fields. The special system-cache marker path also checks the switch at byte offset **197305254**.

The switch is not an authentication or spend-control mechanism. The inspected uses do not change provider credential selection, base URL, project selection or credit limits. Generic extra-body fields and some independently conditioned request options can still enter the query builder. Ordinary `cache_control` remains possible. Strict gateway field validation, scoped authentication, reservations and authoritative settlement therefore remain mandatory.

`claude-code-20250219` is present in the artifact; the agentic-query construction can add it even when experimental betas are disabled. Literal `cache_edits` and `cache_reference` were absent from the inspected artifact. This absence does not prove server behavior or a billing contract. Inspection of the public [TypeScript beta SDK](https://github.com/anthropics/anthropic-sdk-typescript/blob/main/src/resources/beta/messages/messages.ts) and its [beta definitions](https://github.com/anthropics/anthropic-sdk-typescript/blob/main/src/resources/beta/beta.ts) did not establish a public contract for that internal header or those cache fields. Arbitrary beta-string typing is not a cost guarantee. Anthropic's [beta documentation](https://platform.claude.com/docs/en/api/beta-headers) explicitly allows beta pricing to differ.

The switch is undocumented in the reviewed public gateway guide. The artifact proves current client implementation, not a provider promise for future releases. Pin and record the cloud client version/artifact, rerun the fake-only protocol test before an upgrade, and fail closed if a new client ignores the setting or sends unreviewed fields.

## Provider-route environment boundary

The same installed artifact supports newer provider selectors (`CLAUDE_CODE_USE_ANTHROPIC_AWS`, `CLAUDE_CODE_USE_ANTHROPIC_GOOGLE_CLOUD`, `CLAUDE_CODE_USE_MANTLE`, `CLAUDE_CODE_USE_GATEWAY`) in addition to Vertex, Bedrock and Foundry. Host gateway/authentication descriptors, alternate API base URLs and `ANTHROPIC_UNIX_SOCKET` also influence routing. Removing only the three older selectors was insufficient.

For guarded jobs, inherited `ANTHROPIC_*`, `CLAUDE_*` and `_CLAUDE_*` controls are now denied except for the scoped task capability (`ANTHROPIC_API_KEY`) and harness telemetry (`CLAUDE_CODE_ENABLE_TELEMETRY`). The executor then installs the reviewed gateway/model/compatibility controls and obtains signed job identity headers. Regression tests include the observed host/provider/configuration/descriptor/socket inputs and unknown future provider/transport names. Ordinary executors retain their existing environment contract.

This boundary does not override managed policy stored on disk. An image or workspace requiring host-gateway authentication is incompatible with this lane and must fail startup; image review and the deployed canary must verify the effective settings. No enterprise-policy bypass is added.

## Documented system-message subset and cost bound

Anthropic documents ordinary mid-conversation `role: "system"` messages for Haiku 5.5 without a beta header. Admit only plain text/string content or text blocks with reviewed ordinary `cache_control`. Reject `clear_at`, per-message `output_config`, tool additions/removals and other extensions. Those are separately documented beta capabilities. Preserve admitted content and cache markers unchanged. See [mid-conversation system messages](https://platform.claude.com/docs/en/build-with-claude/mid-conversation-system-messages).

This subset adds input text to the same prompt/cache accounting, without admitting server tools or additional inference. The [Haiku 5.5 specifications](https://platform.claude.com/docs/en/models/haiku-5-5/overview) give a 1,000,000-token context and 128,000-token synchronous output ceiling. The [pricing contract](https://platform.claude.com/docs/en/about-claude/pricing) counts input, cache reads and cache writes together for the 100,000-token tier threshold. At the higher tier, the highest admitted input/cache rate is the one-hour write rate of $1 per million; output is $2.50 per million. Pessimistically reserving the entire input window plus maximum output therefore bounds one admitted request at **$1.32**:

`1,000,000 × $1 / 1,000,000 + 128,000 × $2.50 / 1,000,000 = $1.32`.

The reservation includes ordinary system text, tool-definition overhead, cached content and admitted thinking/output within the existing ceilings. It does not rely on an estimated prompt count. Pricing modifiers, server tools, unknown betas and unknown usage categories remain inadmissible; incomplete or impossible usage retains the reservation under the existing fail-closed rules.

## Current fake-only result

The initial discovery run in `/private/tmp/claude-credit-cli-proxy-discovery.log` observed an empty beta header, then exposed the missing system-role subset. After implementing only documented text system messages, the actual CLI **passed** `/private/tmp/claude-credit-cli-tool-contract.log` under the race detector: one ordinary completion and a two-request Bash `printf` tool loop, inside an empty temporary home/workspace. Each fixture checks empty upstream beta headers, per-request admission and exact settlement (15 micro-USD per fake usage record), with no remaining reservation. The second tool-loop request contains the executed tool result. Strict system-extension and all four beta-rejection regression cases remain in the gateway suite. No real provider key, live inference or billing reconciliation was involved. Deployed image/CLI verification and the reserved canary remain mandatory.
