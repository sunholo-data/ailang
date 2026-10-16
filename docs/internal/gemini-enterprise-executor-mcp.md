# Gemini Enterprise ↔ hosted AILANG executor MCP: setup runbook

How the hosted AILANG executor MCP was wired into Gemini Enterprise (GE) on 2026-10-14, as
a rebuild checklist. Everything that can be code is code; the rest is listed step by step with
the values used. Design: [m-ailang-executor-gemini-enterprise](../../design_docs/planned/v0_55_0/m-ailang-executor-gemini-enterprise.md).

## The pieces

| Piece | Where | How it is managed |
|---|---|---|
| Executor tools (`ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_eval`, `ailang_builtins`) | `executor_tools/executor.ail` (ailang) | Code |
| Run policy (IO only, restricted, 60s, 64KB) | `docker/executor-policy.toml` (ailang) | Code |
| Image `executor-mcp` | `docker/Dockerfile.executor-mcp`, `cloudbuild-dev.yaml` step `build-executor-mcp` (ailang) | Code; built by `ailang-core-dev` on every dev push |
| Cloud Run `ailang-dev-executor-mcp` (ailang-multivac-dev, europe-west1), IAM-only | `terraform/cloud_run_executor_mcp.tf`, `enable_executor_mcp` + `executor_mcp_invokers` in `terraform/environments/dev/terraform.tfvars` (ailang-multivac) | Terraform, applied by the `ailang-multivac-config-dev` trigger on push |
| Roll on each image build | `executor-mcp:executor-mcp` in `CORE_SERVICES`, `scripts/cloudbuild-lib.sh` (ailang-multivac) | Code |
| GE project org policies + Org Policy API (sunholo-gemini-enterprise) | `terraform-gemini-enterprise/` (ailang-multivac) | Terraform, **laptop apply** as m@sunholo.com: `make ge-plan` / `make ge-apply` |
| Rebuild on `executor_tools/**`-only pushes | `included_files` in `terraform-triggers/triggers.tf` (ailang-multivac) | Terraform, laptop apply: `make triggers-apply` (**pending as of 2026-10-14**) |
| The GE custom MCP connector | GE console | **Manual**: no Terraform resource yet (steps below) |

## Identities

- GE app project: `sunholo-gemini-enterprise`, number `863281350736`.
- GE calls the MCP as `service-863281350736@gcp-sa-discoveryengine.iam.gserviceaccount.com`,
  sent as a Google-signed ID token in `X-Serverless-Authorization`. Confirmed live on 2026-10-14: GE's client (`google-adk/2.11.0`) got 200s from the IAM-only service. It is sent **only to default
  `*.run.app` URLs**, never to custom domains. That agent holds `roles/run.invoker` on the
  service; so does `user:m@sunholo.com` for hand testing. A call with no token gets 403.
- The service runs as the existing `mcp` service account (feedback publish + Firestore only).

## From scratch, in order

1. **ailang:** push to `dev`. `ailang-core-dev` builds `executor-mcp`.
2. **ailang-multivac:** push to `dev` with `enable_executor_mcp = true` and the invoker list.
   The config trigger applies Terraform and creates the service. If the image build in
   step 1 started *before* this push, its roll step will not know about the service yet. Run
   `gcloud builds triggers run ailang-core-dev --branch=dev --project=ailang-multivac-deploy --region=europe-west3`
   once to roll it (seen 2026-10-14).
3. **GE project policies:** `cd ailang-multivac && make ge-apply`. This enables the Org Policy
   API, overrides `discoveryengine.managed.disableCustomMcpServerConnector` (enforced
   org-wide, which blocks every custom MCP connector), and sets
   `discoveryengine.managed.allowedEgressFqdns` to the executor's **exact** host. Never
   `*.run.app`. Wait 2–5 minutes for the policies to take effect.
4. **GE console → Data stores / connectors → Custom MCP server:**
   - MCP Server URL: `https://ailang-dev-executor-mcp-812435936917.europe-west1.run.app/mcp/`
   - Authentication: **No authentication**. Cloud Run IAM authenticates GE's service agent;
     there is no user sign-in.
   - Sensitive data protection policy: blank.
   - Description: the text below.
   If a failed attempt left a half-created connector (`ailang-mcp-dev_…`), delete it first.
   After creating it, open the connector's **Actions** tab: every tool starts **Disabled**
   (GE chat says "Actions for this custom MCP server are disabled"). Select all, then **Enable actions**.
   `submit_feedback` is the only tool that writes (to the AILANG feedback inbox).
5. **Verify:** `MCP_TOKEN=$(gcloud auth print-identity-token) executor_tools/smoke.sh <url>/mcp/`
   should report 6/6, with the infinite-loop check killed at about 60s. Then call a tool from GE chat.

### Connector description (used by Gemini to decide when to call it)

```
AILANG code executor and documentation. AILANG is a deterministic, effect-typed functional programming language for AI-written code. Use this connector whenever the user asks to write, check, run, or explain AILANG code (.ail files), or asks about AILANG syntax, stdlib, or builtins.

Always call ailang_prompt first. It returns the syntax rules and stdlib reference, and AILANG written without it rarely type-checks. Then:
- ailang_check: type-check a module and verify its contracts; returns errors with fixes
- ailang_run: run a module whose entry point is main, in a sandbox (println only; no file, network, or process access; 60 second limit)
- ailang_eval: evaluate one expression
- ailang_builtins: search builtin functions
- docs_search and the other docs tools: look up AILANG documentation and examples
- submit_feedback: report bugs or confusing behaviour to the AILANG team

Write, check, fix, and re-check until clean, then run.
```

## Seeing how callers use it

Each tool call logs one JSON line (`jsonPayload.event = "executor_call"`). It records the
tool, outcome, duration, code size and a 12-character code hash, plus the first error with
the request directory stripped. The code itself is never logged. Outcomes:

- `ok`
- `check_failed` (ai-check)
- `typecheck_failed` or `policy_violation` (refused before the run)
- `runtime_error` (RTxxx)
- `timeout`
- `output_limit`
- `no_match` (builtins search)
- `exec_failed`, `code_too_large`

```
executor_tools/usage.sh        # last 24h: calls per tool, outcomes, pass rate, top errors
executor_tools/usage.sh 7d
```

What to look for: a repeated `code_sha` means the caller resubmitted unchanged code. Code
calls with no `ailang_prompt` call mean the connector description isn't steering it. The top
first errors show what to teach.

## Using it from Claude Code

```
claude mcp add --transport http ailang-exec https://ailang-dev-executor-mcp-812435936917.europe-west1.run.app/mcp/ \
  --header "Authorization: Bearer $(gcloud auth print-identity-token)"
```

The identity token lasts about an hour, so re-add the server when it expires. The caller must
be in `executor_mcp_invokers`. claude.ai and Claude Desktop connectors need OAuth: planned as design phase P1b (below).

## Traps hit on 2026-10-14

- **`gcloud org-policies` bills the API call to your default gcloud project.** With
  `aitana-multivac-dev` as the default, it fails with "Organization Policy API has not been
  used in project …". Pass `--billing-project=sunholo-gemini-enterprise`; the Terraform
  stack sets `billing_project` for this reason.
- **zsh mangles `$VAR:suffix`.** Building `--process-allowlist "$B:ai-check"` produced
  `ailangi-check`: zsh treats `:a` as a path modifier. Write `${B}:ai-check`.
- **serve-api `Process` had a fixed 30s exec timeout.** That capped any policy timeout above
  30s. It is fixed by `serve-api --process-timeout` (with `--process-allowlist`). The image passes
  90s, and the policy run limit is 60s.
- **`claude mcp add` drops the trailing slash** (it stores `/mcp`). That is harmless: `/mcp`
  answers too.

## Not yet done

- **User OAuth: deployed in dev (design phase P1b, D7), sign-in limited to `sunholo.com`.** Our
  own sign-in server is `executor_oauth/`, on `sunholo/mcp_oauth` 0.1.2 (docparse's issuer is not
  reused: its tokens are docparse API keys). Run `executor_oauth/test_oauth.sh` locally (21 checks).
  - a second Cloud Run service, `ailang-dev-executor-mcp-oauth`, from the same image, publicly
    reachable, run with serve-api `--mcp-connect-only`: no `/mcp/`, every tool on
    `/mcp/connect/` behind the Bearer verifier, HTTP serves only the OAuth routes and sign-in page;
  - connect Claude Code with
    `claude mcp add --transport http ailang-exec https://ailang-dev-executor-mcp-oauth-812435936917.europe-west1.run.app/mcp/connect/`
    (no header: it opens the browser sign-in);
  - this IAM-only service stays as it is for GE's service-agent path;
  - who may sign in: `sunholo.com` only in dev; everyone in prod, with a per-user rate limit
    shipped before the prod promote;
  - to use OAuth from GE instead, put the sign-in server's authorization and token URLs in the
    connector form, set the redirect to `https://vertexaisearch.cloud.google.com/oauth-redirect`,
    and add the OAuth service's host to `allowed_egress_fqdns` (multivac `terraform-gemini-enterprise`).
- A2A (long-running work on the motoko lane): design phase P2.
- Agent Gateway / Agent Registry route: deliberately deferred. See the design doc.
- Prod: the service exists in dev only.
