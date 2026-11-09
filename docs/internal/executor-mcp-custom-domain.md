# Executor MCP on a custom domain (v1.0 gate G4(b)): design and cutover runbook

**Status, 2026-11-09:** the Terraform change is written, tested and waiting:
[ailang-multivac#12](https://github.com/sunholo-data/ailang-multivac/pull/12) (open, mergeable,
`terraform test` 23/23 on `tests/executor_mcp_oauth.tftest.hcl`, re-run 2026-11-09). Nothing is
applied. **G4(b) is blocked on three manual steps that only Mark can take**: the GoDaddy CNAME,
the domain mapping, and the merge and promotion of #12. They are listed under "Decisions and
actions for Mark" below.

Gate text ([v1-mission.md](../../design_docs/v1-mission.md), "The v1.0 bar — v3", G4): *prod is
served on a stable custom domain (the `docparse.ailang.sunholo.com` precedent), not a raw
`*.run.app` URL. Done when the prod domain resolves with a valid cert and the MCP connects.*

Related: [gemini-enterprise-executor-mcp.md](gemini-enterprise-executor-mcp.md) (the service and
its OAuth layer); design `design_docs/planned/v0_55_0/m-ailang-executor-gemini-enterprise.md`.

## 1. Does a prod executor MCP exist?

**Yes, since 2026-11-05.** Only the public OAuth service is in prod. The IAM-only service used
by Gemini Enterprise stays dev-only, by design.

| | dev (`ailang-multivac-dev`) | prod (`ailang-multivac`) |
|---|---|---|
| IAM-only `*-executor-mcp` (GE path) | yes | no (deliberately; GE sends its ID token only to `*.run.app`) |
| OAuth `*-executor-mcp-oauth` | `https://ailang-dev-executor-mcp-oauth-812435936917.europe-west1.run.app` | `https://ailang-executor-mcp-oauth-1032908249524.europe-west1.run.app` |
| Image | `executor-mcp:latest` | `executor-mcp:v0.57.4` (pinned) |
| Sign-in | `sunholo.com` only | `*` (any verified Google account), 30 rpm / burst 30 per account |
| Custom domain | none (convention: dev and test stay on run.app) | **none yet** |

Evidence (read-only, 2026-11-09):

- `gcloud run services describe ailang-executor-mcp-oauth --project=ailang-multivac --region=europe-west1`:
  Ready since `2026-11-05T12:28`, created by `sa-cloudbuild@ailang-multivac-deploy`. Its args are
  `--oauth-issuer https://ailang-executor-mcp-oauth-1032908249524.europe-west1.run.app` with
  `--mcp-subject-rpm 30 --mcp-subject-burst 30`, and `OAUTH_ISSUER` is the same run.app URL.
- It was created by multivac PR #10 (merged), which is on both `origin/dev` and `origin/prod`
  (`5fd820c`), plus PR #11, which promoted the image.
- `curl …run.app/.well-known/oauth-authorization-server` returns `issuer` = the run.app URL.
  `POST /mcp/connect/` with no body returns 415, so the service is live.

So promotion to prod is already done. What G4(b) still needs is the domain.

## 2. How docparse's domain is mapped

The mapping is a **Cloud Run domain mapping**, not a load balancer and not Firebase Hosting.

- **Project:** `ailang-multivac` (prod). `gcloud beta run domain-mappings list --region=europe-west1 --project=ailang-multivac`
  lists billing, coord, dashboard, docparse and mcp `.ailang.sunholo.com`, all Ready. Dev and
  test have 0 mappings.
- **Created by hand:** `serving.knative.dev/creator: m@sunholo.com`, on 2026-04-02. Its
  conditions `Ready`, `CertificateProvisioned` and `DomainRoutable` are all True, and its
  resource record is `CNAME docparse.ailang → ghs.googlehosted.com.`
- **Certificate:** Google-managed and auto-renewed by Cloud Run. The cert actually served is from
  Let's Encrypt (`issuer=C=US, O=Let's Encrypt, CN=YR1`), `CN=docparse.ailang.sunholo.com`,
  valid until 2026-12-21.
- **DNS:** `sunholo.com` is at **GoDaddy**: NS `ns75/ns76.domaincontrol.com`, SOA
  `dns.jomax.net`. It is not Cloud DNS: `gcloud dns managed-zones list --project=ailang-multivac`
  returns 0 zones. A DNS change is therefore a manual GoDaddy edit.
- **No load balancer:** `gcloud compute url-maps list` and `gcloud compute ssl-certificates list`
  on `ailang-multivac` both return 0 items. The Compute API answers, so this is a real empty
  result.
- **Why the mapping is not in Terraform:** `google_cloud_run_domain_mapping` fails with
  *"Caller is not authorized to administer the domain"*. Domain ownership is verified per user
  in Search Console, and the Cloud Build SA is not a verified owner. `gcloud domains list-user-verified`
  as `m@sunholo.com` lists `sunholo.com`. The convention is recorded in multivac `CLAUDE.md`
  ("Custom Domain Mappings") and in the comment in `terraform/cloud_run_mcp.tf`.
- **OAuth:** docparse's issuer is `https://docparse.ailang.sunholo.com` (from `docparse_api_url`).
  Its RFC 9728 `resource` is `https://docparse.ailang.sunholo.com/mcp/connect/`. This shows the
  domain mapping passes `Host` and `X-Forwarded-Proto: https` through, and serve-api's
  `protocol.PublicBaseURL` needs both.

**Pattern to copy:** a Cloud Run domain mapping in `ailang-multivac/europe-west1`, created with
gcloud as `m@sunholo.com`; a GoDaddy CNAME to `ghs.googlehosted.com.`; the Google-managed cert;
Terraform for everything except the mapping itself.

## 3. Domain name

Hosts in use under `ailang.sunholo.com`, from repo grep and live DNS: `mcp` (the **docs** MCP,
`ailang-mcp`, mapped), `docparse`, `dashboard`, `coord`, `billing`, `registry` (in the
`ailang-registry` project), and `ntfy` (referenced only). The apex `ailang.sunholo.com` is
GitHub Pages. There is no wildcard record: `exec.`, `executor.` and `run.` do not resolve.

- **Prod: `executor.ailang.sunholo.com`.** Mark approved this on 2026-11-05 (recorded in PR #12)
  and it is what #12 implements. `mcp.` is taken by the docs MCP. `executor` reads more clearly
  than `exec` and matches the service name `ailang-executor-mcp-oauth` and the `serverInfo` name
  `executor`. `exec.ailang.sunholo.com` is the shorter alternative, and switching to it means
  changing one tfvars value plus the two list entries.
- **Dev: none, recommended.** Multivac's convention is that dev and test use the run.app URL
  and only prod gets a custom domain (comment in `environments/dev/terraform.tfvars`), and G4(b)
  is about prod only. If a dev name is wanted later, use `executor-dev.ailang.sunholo.com`.
  The same variable exists in dev (`executor_mcp_oauth_custom_domain = ""`). The mapping would go
  in `ailang-multivac-dev`; domain verification is per user, so the same gcloud command works.

## 4. What changes

### 4a. Terraform (multivac PR #12, branch `feat/executor-mcp-custom-domain`)

The core of the diff (the full diff, the prod plan and the test list are in the PR):

```hcl
# terraform/cloud_run_executor_mcp.tf
 locals {
-  executor_mcp_oauth_name   = "${var.prefix}-executor-mcp-oauth"
-  executor_mcp_oauth_issuer = "https://${local.executor_mcp_oauth_name}-${data.google_project.current.number}.${var.region}.run.app"
+  executor_mcp_oauth_name        = "${var.prefix}-executor-mcp-oauth"
+  executor_mcp_oauth_run_app_url = "https://${local.executor_mcp_oauth_name}-${data.google_project.current.number}.${var.region}.run.app"
+  executor_mcp_oauth_issuer      = var.executor_mcp_oauth_custom_domain != "" ? "https://${var.executor_mcp_oauth_custom_domain}" : local.executor_mcp_oauth_run_app_url
 }
 # lifecycle of google_cloud_run_v2_service.executor_mcp_oauth: two new preconditions
+    precondition {   # issuer host must be a Firebase authorized domain
+      condition = var.firebase_authorized_domains == null || contains(coalesce(var.firebase_authorized_domains, []), trimprefix(local.executor_mcp_oauth_issuer, "https://"))
+    }
+    precondition {   # and an allowed referrer of the sign-in page's browser key
+      condition = length(var.allowed_referrers) == 0 || contains(var.allowed_referrers, "${local.executor_mcp_oauth_issuer}/*")
+    }
+output "executor_mcp_oauth_domain_mapping_command" { … }   # the exact gcloud create command

# terraform/variables.tf
+variable "executor_mcp_oauth_custom_domain" { default = "" }   # bare lowercase host; validated

# terraform/environments/prod/terraform.tfvars
 firebase_authorized_domains = [ …,
   "ailang-executor-mcp-oauth-1032908249524.europe-west1.run.app",
+  "executor.ailang.sunholo.com",
 ]
 allowed_referrers = [ …,
+  "https://executor.ailang.sunholo.com/*",
 ]
+executor_mcp_oauth_custom_domain = "executor.ailang.sunholo.com"

# terraform/environments/dev/terraform.tfvars
+executor_mcp_oauth_custom_domain = "" # run.app issuer
```

The local feeds both `--oauth-issuer` (serve-api's RFC 9728 `authorization_servers`) and
`OAUTH_ISSUER` (oauth.ail's RFC 8414 metadata, the same-origin sign-in page at
`<issuer>/oauth-login.html`, and the RFC 9207 `iss`), so one variable moves all of them together.

#12 also adds `--server-info name=executor,title=AILANG Executor,version=0.57.4`. This is
branding that rides along with the domain change, and it is passed only on release tags ≥ v0.57.4.

Prod plan, as recorded in the PR: **0 to add, 4 to change, 0 to destroy**. The four changes are
the executor service args and env, the Identity Platform `authorized_domains`, and the two
browser keys' `allowed_referrers`. Dev plans no change. Test does not enable the OAuth service.

There is no `google_cloud_run_domain_mapping` resource, because of the authorization failure
described in §2. To bring mappings into Terraform later, Mark would add
`sa-cloudbuild@ailang-multivac-deploy.iam.gserviceaccount.com` as an owner of `sunholo.com` in
Search Console. That would cover all six mappings, so it is a separate decision from this one.

### 4b. DNS (GoDaddy, manual)

| Name | Type | Value | TTL |
|---|---|---|---|
| `executor.ailang` (in zone `sunholo.com`) | CNAME | `ghs.googlehosted.com.` | default (600) |

That is the only DNS change. No TXT verification record is needed, because `sunholo.com` is
already verified for `m@sunholo.com`.

### 4c. OAuth configuration

| Item | Before | After | Where it comes from |
|---|---|---|---|
| Issuer (`iss`, RFC 8414 `issuer`, RFC 9207 `iss`) | run.app URL | `https://executor.ailang.sunholo.com` | `local.executor_mcp_oauth_issuer` → `--oauth-issuer` + `OAUTH_ISSUER` |
| `authorization_endpoint` / `token_endpoint` | `<run.app>/oauth/{authorize,token}` | `https://executor.ailang.sunholo.com/oauth/{authorize,token}` | derived from the issuer (`mcp_oauth` `asMetadata`) |
| Sign-in page | `<run.app>/oauth-login.html` | `https://executor.ailang.sunholo.com/oauth-login.html` | `loginUrlFor(issuer)` in `executor_oauth/oauth.ail` |
| Protected-resource metadata URL | `<host>/.well-known/oauth-protected-resource/mcp/connect/` | same path on whichever host the client used | request `Host` (`serveapi/protocol.PublicBaseURL`), no config |
| `resource` | the host the client used + `/mcp/connect/` | same | request `Host`, no config |
| Redirect URIs | from each client's CIMD document | unchanged | clients own them; nothing to register |
| Token audience | none: opaque `ax_` tokens stored as sha256 digests in Firestore `executor_oauth_records`, not bound to the issuer | unchanged | existing connections keep working |
| Firebase authorized domains | includes the run.app host | **+ `executor.ailang.sunholo.com`** | tfvars `firebase_authorized_domains`. `signInWithPopup` refuses an unlisted host |
| Browser key referrers | `*.ailang.sunholo.com` + the run.app host | + the explicit host (already matched by the wildcard, but the new precondition checks the exact entry) | tfvars `allowed_referrers` |

**Not docparse's issuer.** The new issuer is a separate origin with its own Firestore collection
(`executor_oauth_records`). It shares only the Firebase project `ailang-multivac` used for Google
sign-in, as dev already does. docparse's issuer, records and API-key semantics are untouched.

**After cutover, the run.app URL keeps working.** Its metadata names
`resource = <run.app>/mcp/connect/` and `authorization_servers = [https://executor.ailang.sunholo.com]`,
which RFC 9728 allows. A client that cached the *old* AS metadata may see an `iss` mismatch on its
next authorize, and removing and re-adding the connector clears that. Refresh tokens issued
earlier remain valid at either token endpoint, because both reach the same service and the same
Firestore records.

### 4d. Client configs that hard-code a run.app hostname

Grepped 2026-11-09 across this repo, multivac `origin/dev`, `~/.claude.json`,
`~/.claude/settings.json`, and `ailang-docs`, `ailang_bootstrap`, `ailang-packages`,
`ailang-world`, `ailang-parse`, `docparse`, `ailang-fleet` and `~/dev/daneel`.

| Location | Host | Action |
|---|---|---|
| multivac `terraform/environments/prod/terraform.tfvars` lines 47 and 73 | prod OAuth run.app | **keep through the cutover**. Remove them in a follow-up once nothing signs in there (optional; harmless if left) |
| multivac `terraform/environments/dev/terraform.tfvars` lines 84 and 105 | dev OAuth run.app | none (dev stays on run.app) |
| multivac `terraform-gemini-enterprise/variables.tf` line 13 | dev **IAM** run.app | none: GE sends its ID token only to `*.run.app`. If GE is ever pointed at the OAuth service instead, add `executor.ailang.sunholo.com` to `allowed_egress_fqdns` |
| `~/.claude.json` (MCP server `ailang-exec-oauth`) | dev OAuth run.app | none required. Optionally add prod: `claude mcp add --transport http ailang-exec https://executor.ailang.sunholo.com/mcp/connect/` |
| `docs/internal/gemini-enterprise-executor-mcp.md` lines 43, 98 and 128 | dev IAM and dev OAuth run.app | none (dev-specific instructions). This doc is linked from there |
| `design_docs/planned/integrations-review-internal-2026-11-05.md` | dev OAuth run.app | none (historical review) |

**Nothing names the prod run.app URL as a connect target**, so no published client instructions
change. Any new public connect instructions should use
`https://executor.ailang.sunholo.com/mcp/connect/`.

## Decisions and actions for Mark

1. **Confirm the name.** `executor.ailang.sunholo.com` was approved on 2026-11-05 per #12. Say so
   if `exec.` is preferred instead; that is a three-line tfvars change on #12.
2. **GoDaddy:** add `CNAME executor.ailang → ghs.googlehosted.com.` in the `sunholo.com` zone.
3. **Create the mapping** as `m@sunholo.com` (the domain's verified owner):
   ```bash
   gcloud beta run domain-mappings create --service=ailang-executor-mcp-oauth \
     --domain=executor.ailang.sunholo.com --region=europe-west1 --project=ailang-multivac
   ```
   An agent session whose gcloud account is `m@sunholo.com` can run this once Mark says go,
   because the command is the same whoever types it.
4. **Wait for the certificate**, usually 15–30 minutes after DNS resolves:
   ```bash
   gcloud beta run domain-mappings describe --domain=executor.ailang.sunholo.com \
     --region=europe-west1 --project=ailang-multivac --format='value(status.conditions)'
   # all of Ready / CertificateProvisioned / DomainRoutable must be True
   curl -s https://executor.ailang.sunholo.com/.well-known/oauth-protected-resource/mcp/connect/
   # serves now with resource = executor.ailang.sunholo.com and authorization_servers = the run.app issuer
   ```
5. **Merge multivac #12 into `dev`, then promote `dev → test → prod`** (branch pushes; the
   `ailang-multivac-config-*` triggers apply the change). Dev and test plan no change. Run a
   read-only `make plan ENV=prod` first and expect 4 in-place changes. **Never before step 4:**
   the issuer is where every client is sent to sign in, and sending them to a host without a
   certificate breaks all sign-ins.

## Verify (G4(b) done-when)

```bash
dig +short CNAME executor.ailang.sunholo.com              # ghs.googlehosted.com.
echo | openssl s_client -connect executor.ailang.sunholo.com:443 -servername executor.ailang.sunholo.com 2>/dev/null \
  | openssl x509 -noout -subject -enddate                  # CN=executor.ailang.sunholo.com, future notAfter
curl -s https://executor.ailang.sunholo.com/.well-known/oauth-authorization-server   # issuer = https://executor.ailang.sunholo.com
curl -s https://executor.ailang.sunholo.com/.well-known/oauth-protected-resource/mcp/connect/
                                                          # resource AND authorization_servers on executor.ailang.sunholo.com
ailang mcp check https://executor.ailang.sunholo.com/mcp/connect/   # directory checks (anthropic)
claude mcp add --transport http ailang-exec-prod https://executor.ailang.sunholo.com/mcp/connect/
                                                          # browser sign-in, then call executor_whoami and ailang_check
```

Then update the G4 row of `design_docs/v1-mission.md`: (b) → DONE with the date, and N drops by 1.

## Rollback

Set `executor_mcp_oauth_custom_domain = ""` in prod tfvars and promote it. The issuer returns to
the run.app URL. Leave the mapping and the CNAME in place, since they cost nothing. Tokens are not
bound to the issuer, so no user is signed out. Delete the mapping
(`gcloud beta run domain-mappings delete …`) only to retire the name.

## Considered and not chosen

- **A global HTTPS load balancer with a serverless NEG.** Google recommends it over domain
  mappings, which are still Preview, for production latency. It would be the only load balancer
  in the project and would add a forwarding-rule cost (about $18 a month) and a second pattern.
  The five existing prod mappings, docparse among them, already pass the Anthropic connector
  checks on a domain mapping. If a latency problem is measured, revisit this for every
  `*.ailang.sunholo.com` service together.
- **Firebase Hosting rewrite.** It adds a hop and a second cert path, and nothing in the project
  uses it.
- **Reusing `mcp.ailang.sunholo.com`.** That host is the public docs MCP, with 122 references in
  repo docs. Merging the two surfaces would change a stable URL's tool list and auth.
