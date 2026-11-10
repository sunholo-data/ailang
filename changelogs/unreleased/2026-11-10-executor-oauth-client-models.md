### Added
- **The executor's sign-in service can serve Copilot, Gemini Enterprise and Google Connected Apps
  clients** (`executor_oauth/oauth.ail`, on `sunholo/mcp_oauth` 0.2.0's entry points). Three env
  switches, all off unless set to `1`, so the same image keeps 0.1.2's CIMD-only behaviour in prod:
  - `OAUTH_CONFIDENTIAL_CLIENTS`: operator-created `mco_` clients (Gemini Enterprise; Copilot "OAuth
    with static registration"). Mint one with
    `OAUTH_CONFIDENTIAL_CLIENTS=1 OPERATOR_REDIRECTS=<uri> ailang run --caps IO,FS,Env,Net,Clock,Rand,SharedMem --entry createClientCli executor_oauth/oauth.ail`;
    the secret prints once. The command is `@noexpose @nomcp` with no route.
  - `OAUTH_DCR` + `OAUTH_DCR_REDIRECT_HOSTS`: RFC 7591 `/oauth/register` (Copilot "dynamic", Copilot
    Studio, Connected Apps). It **fails closed** without a host allowlist, because the package reads
    an empty list as "any https host".
  - `OAUTH_REVOCATION`: RFC 7009 `/oauth/revoke`.
  The metadata advertises only what is on. The consent page names a registered client by its
  redirect host and marks a self-registered (DCR) one as unverified. `executor_oauth/test_oauth.sh`:
  43 checks (27 → 43), covering switches off (no new metadata, 404s, operator command unreachable)
  and on (DCR host allowlist, a Copilot-style DCR sign-in, revocation, a Gemini Enterprise-style
  client with Basic auth and a wrong secret, no secret in the log, DCR off without hosts). The
  fail-closed check fails when the guard is removed.
