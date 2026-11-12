# Submission kit: the hosted AILANG executor in the Claude and ChatGPT directories

**Status:** prepared 2026-11-12. Not submitted, and **not ready**: §1 lists what blocks it.
**Server:** `https://executor.ailang.sunholo.com/mcp/connect/`. Prod reports `executor (AILANG Executor) 0.60.2`, with 29 tools. OAuth 2.1 runs on our own authorization server (`sunholo/mcp_oauth` 0.2.1): PKCE S256, CIMD, and `iss` in authorization responses. Sign-in is Google via Firebase, open to any verified Google account, 30 calls/min per account.
**Marketplaces:** M1 is the Anthropic connector directory and M2 the OpenAI plugin directory, from `m-connector-channels-marketplaces.md` §2. D1 recommends both for the executor. D5 asks for email + password sign-in for reviewer accounts.
**Template:** Parse's kit, `m-serveapi-directory-ready-submission-kit.md`. Its §2.2 (Anthropic) and §2.3 (OpenAI) were re-checked 2026-10-06. This kit relies on them and on the marketplaces doc's 2026-11-05 re-read. **Re-fetch both vendor pages before submitting.**

Evidence comes from runs on 2026-11-12. Live checks were `ailang mcp check … --target both` (10/10 PASS) and a `tools/list` over Streamable HTTP. Each tool was called once, signed in from Claude Code (client `claude.ai`) against prod. `submit_feedback` was not called. Firebase sign-in providers were read through the Identity Toolkit admin API.

---

## 1. Gaps (read first)

Ordered by how likely each is to cause a rejection. **Lane**: AILANG fix · console/terraform (Mark) · done.

| # | Gap | Vendor | Lane | Evidence | State |
|---|---|---|---|---|---|
| X1 | **Six docs tools fail on valid input.** `roadmap`, `design_doc`, `example_get`, `prompt_get`, `benchmarks_compare` and `benchmark_run` all return `snapshot_read_failed`, because the files they read were never generated. `tools/build-snapshot` emits none of `unscoped/roadmap.json`, `unscoped/design_docs/<slug>.json`, `unscoped/benchmarks_compare_index.json`, `unscoped/benchmarks/runs/*`, `versioned/<v>/examples/<path>.json` or `versioned/<v>/prompts/compact.json`. Its own header lists `roadmap.json` as an output. The same tools fail on the open docs server `mcp.ailang.sunholo.com`. Anthropic: "Every tool must return a successful response when called with valid parameters." | Both (Anthropic hard) | AILANG fix | Live calls on prod. A local `go run ./tools/build-snapshot --repo .` at HEAD produces the same file set (no `roadmap.json`, no `design_docs/`, no `examples/`, prompts only `agent`, `devtools` and `full`). | **Open: blocks M1 and M2.** Decision D-X1 below. |
| X2 | **Wrong or unusable results.** `examples_list` returns `[]`. `design_docs_list` ignores its `state` filter and returns all 1,848 docs (~516 KB). `ailang_prompt` is ~113 KB. `install_guide` ignores `harness`. `onboarding_guide` tells the agent to use `mcp.ailang.sunholo.com` (the wrong server) and names a tool that does not exist (`examples_for_concept`). `stdlib_search "map"` misses `std/list.map`. `example_for_concept` returns `examples/experimental/*` files headed "this example does not work". `benchmarks_list` shows no run IDs, so `benchmark_run` cannot be reached. | Both (descriptions must match behaviour; functional review) | AILANG fix | Live calls on prod. | **Open.** Goes with X1. |
| X3 | **Reviewers had no password login.** The executor offered Google only. A dedicated Google account hits device challenges, and OpenAI requires credentials with "no MFA, email or SMS codes, magic links". | Both (OpenAI hard) | code done; console (Mark) | `oauth-login.js` before `20fc2649c`. | **Code done** (`20fc2649c`): email + password, sign-in only. **The provider is OFF** in both `ailang-multivac` and `ailang-multivac-dev`: the admin config has no `signIn.email`, and the IdPs are google.com and github.com only. Mark: §5 steps 3–4. |
| X3b | **Parse's reviewer login (its G1) is not live either.** Parse signs in to the same prod Firebase project, `ailang-multivac`, and that project has no Email/Password provider. Parse's widget option therefore fails with `auth/operation-not-allowed`. | Both, for Parse | console (Mark) | Same admin-config read. | Fixed by the same console step as X3. |
| X4 | **No privacy policy or terms for the executor.** The AILANG site has neither: `/privacy`, `/terms` and `/docs/privacy` are all 404. `www.sunholo.com/privacy` covers only the marketing site. Parse's policy covers Parse. | Both (OpenAI: four listing URLs) | drafted; Mark approves | curl, 2026-11-12. | **Drafted** (`138cbd286`): `docs/docs/legal/{privacy,terms}.md`, `draft: true`, so they are not built. Mark approves, adds the contact address and removes `draft`. **Published 2026-11-12** (approved; contact ailang@sunholo.com), live at `https://ailang.sunholo.com/docs/legal/privacy` and `/terms`. Keeping them current (source metadata, CI drift and overdue checks): #2300. |
| X5 | **The consent page linked neither privacy nor terms.** OpenAI: the flow "must be transparent and explicit". | OpenAI (good practice for both) | done | `oauth-login.html` before `20fc2649c`. | **Done** (`20fc2649c`). Privacy, Terms and Help links ship with the next release. |
| X6 | **No OpenAI domain-verification route.** `/.well-known/openai-apps-challenge` was 404. | OpenAI | code done; terraform (Mark) | curl. | **Code done** (`20fc2649c`). It serves `OPENAI_APPS_CHALLENGE` as text/plain, 404 when unset, and `test_oauth.sh` covers it. Mark: terraform variable plus the token (§5 step 8). |
| X7 | **Internal identifiers in results.** `ailang_run` and `ailang_eval` returned `"policy":"/srv/executor-policy.toml"`, `policy_digest` and `module_graph.digest`, both in the stderr echo and in a refusal's stdout. Every failing docs tool returned `detail: missing:/srv/snapshot/...`. OpenAI app review requires removing "internal identifiers". | OpenAI | done | Live calls. | **Done** (`ab3c00698`, `775f41c0a`), shipping next release. No result carried request, trace or session IDs or call timestamps. `ailang_check` reports per-function `duration` (nanoseconds); that is a measurement, not a timestamp, and is left as is. |
| X8 | **Descriptions with internal jargon or steering.** `submit_feedback` named `pkg:` inboxes, "autonomous agent", coordinator filtering and "a separate sprint" (Parse kit G12). `executor_whoami`, `docs_nav`, `docs_search` and `onboarding_guide` carried implementation notes. | Both | done | `tools/list`. | **Done** (`72fe5d258`). Still open, under X2: the *content* of `onboarding_guide` pushes `submit_feedback` hard ("highest-value signal we receive"). |
| X9 | **Release.** X5–X8 reach prod only with an AILANG release and a prod image-tag bump. Separately, prod tfvars pin `executor_mcp_image_tag = "v0.60.1"` while prod reports 0.60.2. That is a drift to explain before the bump, or the "service runs `:latest`" finding from P1 again. | Both | Mark (promote), agent (multivac PR) | `ailang-multivac/terraform/environments/prod/terraform.tfvars:111`; `initialize`. | **Open.** |
| X10 | **claude.ai (web) is unconfirmed.** The brief says claude.ai has not been tried. `hosted-executor.md` says "Verified 2026-11-11" for claude.ai and shows screenshots. The portal asks you to confirm that every tool ran in Claude. | Anthropic | Mark | `docs/docs/guides/hosted-executor.md` §Claude. | **Open.** Run §4.4 in claude.ai as a custom connector, signed in as the reviewer. |
| X11 | **Public connector docs.** Anthropic requires public docs by the publish date. `hosted-executor.md` (live) covers the URL, tools, sandbox, adding it in Claude and ChatGPT, and the check. It lacked disconnecting and troubleshooting. | Anthropic | done | — | **Done** (`138cbd286`). It deploys with the docs site on push to `dev`. |
| X12 | **Self sign-up through the Firebase REST API once Email/Password is on.** The page never creates accounts. But with the provider enabled, `accounts:signUp` is callable with the public browser key. `client.permissions` is empty, and setting `disabledUserSignup` would also block first-time Google sign-ups. Residual risk: the executor's gate requires a verified email, so a self-made account must verify a mailbox it controls. That is the same bar as a free Google account, and the 30/min per-account limit still applies. Parse's gate also refuses unverified addresses (`api_keys.ail:965`). | — | accepted | Admin config read. | Accept, and note it. Revisit if abuse shows up in `executor_tools/usage.sh`. |
| X13 | **Cosmetic AILANG issues a reviewer will see.** In an `ailang_check` counterexample, `result` prints as truncated SMT text (`"(ite (= (>= $p_x 0) (>= 2 0)"`). `ailang_check` stderr carries `WARNING MOD010 (temp-path) … Auto-relaxed for temporary directory`. `ailang_builtins` lists internal `_`-prefixed builtins (`_secret_read`, `_msg_send`, …). | Both (quality) | AILANG fix | Live calls. | Open, low. |
| X14 | **OpenAI account prerequisites:** org verification, an Owner or Apps Management Write role, and a **project with global (not EU) data residency**. | OpenAI | Mark | Marketplaces doc §2 M2. | Open (Mark). |

Already fine. `mcp check --target both` passes 10/10: server-info, no listed module inventory, a title and all hints on all 29 tools, no credential-shaped parameters, no placeholder arguments, lazy auth (28 gated tools answer 401 with resource metadata), AS metadata (S256 plus a registration method), framing protection, no `openai/fileParams`, and `securitySchemes` on the 28 gated tools. An unauthenticated `tools/call` returns `_meta["mcp/www_authenticate"]` plus `isError`. AS metadata has `authorization_response_iss_parameter_supported: true`, and the package returns `iss`. ChatGPT therefore uses the stable `https://chatgpt.com/connector_platform_oauth_redirect`, which was verified on chatgpt.com 2026-11-11. Every tool is read-only except `submit_feedback` (`readOnly=false`, `openWorld=true`, `noauth`). Tool names are ≤ 64 characters, and there is no catch-all tool and no media generation.

**What blocks each submission:**
- **M1 (Anthropic):** X1 and X2 (functional review), X3 console steps, X4 approval, X9 release, X10 confirmation.
- **M2 (OpenAI):** all of M1, plus X6 (token and terraform), X14, the demo video, and the ZIP.

### D-X1: how to close X1/X2 (Mark)

| Option | What | Cost | Effect |
|---|---|---|---|
| **A (recommended)** | Build the missing snapshot files in `tools/build-snapshot` (roadmap, per-slug design docs, per-path examples, the compact prompt or drop `kind=compact`, benchmark runs and compare index), fix the `examples_list`, `design_docs_list` and `install_guide` filters, cap large results, and correct the onboarding guide. A test asserts that every file a `mcp_tools/*.ail` tool reads exists in a fresh snapshot. | ~1.5–2 agent days, AILANG lane | Every listed tool works, on both the executor and the docs server |
| B | Hide the six failing tools (`@noexpose`) and fix only the filters, until A lands. | ~0.5 day | Shrinks the listed surface to 23 tools. It also hides them on the docs server and in the MCP Registry listing. A later tool addition counts as a listing change. |

---

## 2. Vendor requirements, executor status

Requirement wording is from the Parse kit §2.2/§2.3 and marketplaces doc §2. ✅ present · 🟡 needs work · ❌ missing · 👤 Mark only.

### 2.1 Anthropic (portal `claude.ai/directory/manage` → Submit new → MCP connector)

| Requirement | Status | Notes |
|---|---|---|
| Paid plan; Owner on Team/Ent | 👤 | Submit from the Holosun org, as for Parse. |
| `https://` URL | ✅ | `https://executor.ailang.sunholo.com/mcp/connect/` |
| Title + hints on every tool | ✅ | 29/29 (`mcp check`). |
| Every tool succeeds with valid params | ❌ | X1, X2. |
| Name ≤ 100 / one-liner ≤ 200 / description ≤ 2,000 | ✅ draft | §3. |
| Categories (1–5) | 👤 | Suggest **Developer tools** and **Code**. |
| Documentation URL | ✅ | `https://ailang.sunholo.com/docs/guides/hosted-executor` (200). |
| Privacy policy URL | 🟡 | X4: `https://ailang.sunholo.com/docs/legal/privacy` once published. |
| Support contact | 👤 | Pick one: GitHub issues `https://github.com/sunholo-data/ailang/issues` (public repo) and/or an email address (the same address goes into the policy). |
| Icon | ✅ | `docs/static/img/ailang-logo.png` (512×512 PNG). The server's `/icon.png` is 256×256. |
| Slug (permanent) | 👤 | Suggest `ailang-executor`. |
| Use cases, prerequisites, read/write | ✅ draft | §3.2. |
| ≥ 3 working prompts | 🟡 | §4.4 has five. They were verified at the tool level on prod and still need a claude.ai run (X10). |
| Authentication type | ✅ | **OAuth with client ID metadata documents.** Flag lazy auth: `tools/list` is open, `submit_feedback` needs no sign-in, everything else prompts on demand. |
| Data handling | ✅ | Own API. No health data. No sponsored content. The MCP tools use no third-party AI (Vertex Gemini serves only the A2A agent, which is not part of this listing). |
| Reviewer credentials | ❌ | X3 console steps; §4.5. |
| Seven acknowledgments | 👤 | Fine once X1/X2/X4 are done: no financial transactions, no AI media, no prompt-injection text (X8 done), public docs (X11). |
| Domain ownership (§3F) | ✅ | `executor.ailang.sunholo.com` and `ailang.sunholo.com` (Sunholo). |
| Data minimisation | ✅ | Tools take code, an expression or a query. `submit_feedback` takes free text plus an optional `contact`, and its description now says to send only what the user asked for and to leave out personal information. |

### 2.2 OpenAI (`platform.openai.com/plugins` → upload ZIP)

| Requirement | Status | Notes |
|---|---|---|
| Org verification; Owner or Apps Management Write; **global data residency project** | 👤 | X14. |
| ZIP: `plugin.json` + `mcp.json` + assets | ✅ draft | §3.4. Nothing exists in a repo yet. Build it at submission time, or keep it in `ailang_bootstrap` like Parse's `docparse-skill`. |
| `displayName` ≤ 30 / `shortDescription` ≤ 30 / `longDescription` ≤ 4,000, no pricing | ✅ draft | §3. |
| Four URLs (website, support, privacy, terms) | 🟡 | Privacy and terms are waiting on X4. |
| `defaultPrompt` ≤ 3 × 128 | ✅ draft | §3.4. |
| `logo` / `composerIcon` (square, ≥ 48 px, PNG to be safe) | ✅ | `ailang-logo.png` 512×512. |
| `brandColor` (2:1 vs white) | ✅ | `#E73C17` (the site's primary), 4.15:1 against white. |
| Domain verification | 🟡 | X6. |
| OAuth 2.1; `securitySchemes`; `_meta["mcp/www_authenticate"]`; redirect URI | ✅ | Live. |
| Explicit read/destructive/open-world booleans | ✅ | All three on all 29 tools. |
| No restricted data, credentials through the model | ✅ | `mcp check` credentials PASS. |
| No subscriptions or upsell | ✅ | No pricing anywhere in the tool output. The benchmark tools report model spend (`total_cost_usd`) as data, not as a price. |
| No internal identifiers or timestamps in results | ✅ after release | X7. |
| 5 positive + 3 negative test cases | ✅ draft | §4.6. |
| Demo video | 👤 | §5 step 11. |
| MFA-free test credentials | ❌ | X3. |
| Suitable for 13–17 | ✅ | A code tool. The privacy draft states "not directed at children under 13". |

---

## 3. Listing text

### 3.1 Descriptions

**Name / `displayName`:** AILANG Executor

**Anthropic one-liner (≤ 200; 153 chars):**
> Write, type-check and run AILANG programs in a locked-down sandbox, prove their contracts with Z3, and look up the AILANG language, stdlib and changelog.

**OpenAI `shortDescription` (≤ 30; 25 chars):** `Check and run AILANG code`

**Long description (Anthropic ≤ 2,000 / OpenAI ≤ 4,000; no pricing; 1,418 chars):**
> AILANG is a programming language designed for AI code generation, with explicit effects, static types and deterministic execution. The AILANG Executor lets the assistant write AILANG and check that it is right, without installing anything.
>
> Tools:
> • Check code: type-check a module and prove its requires/ensures contracts with the Z3 solver. A failed proof comes back with a concrete counterexample.
> • Run code: run a program in a sandbox that grants only console output. It has no files, network, environment or processes, and each run is limited to 60 seconds and 64 KB of output. A program that asks for more gets a named policy refusal.
> • Evaluate an expression, and search the builtin functions.
> • Language reference: the AILANG syntax guide, the standard-library modules and search, the effect catalogue, known limitations, and the changelog of every release.
> • Feedback: report a bug or a missing feature to the AILANG team.
>
> Sign in with a Google account whose email address is verified. The executor stores your sign-in user ID and digests of the tokens it issues, each only until it expires. The code you run is written to a temporary directory for that one call, then deleted, and is never logged. Each account can make 30 calls a minute. Hosting is in the EU (Belgium), and Holosun ApS (Denmark) operates the service.
>
> To run AILANG on your own machine instead, install it from https://ailang.sunholo.com.

(This text deliberately does not mention examples, design docs, the roadmap or benchmarks, because of X1/X2. Add them once D-X1 option A lands.)

### 3.2 Use cases (Anthropic "Use cases")

- **Write AILANG that type-checks.** The assistant reads the syntax guide, writes a module and checks it, then fixes errors from their locations and suggested fixes.
- **Prove a function's contract.** Write `requires`/`ensures` and get `verified` or a counterexample from Z3.
- **Run a program safely.** Run it in a sandbox with console output only and see its real output.
- **Look up the language.** Stdlib modules and signatures, effects, known limitations, and what changed in a release.

**Prerequisites:** a Google account with a verified email address. No AILANG install.
**Reads/writes:** read-only. The one write is `submit_feedback`, which files a report for human review. Running code changes nothing outside its own temporary directory.

### 3.3 Privacy and terms

Drafts: `docs/docs/legal/privacy.md` and `docs/docs/legal/terms.md` (`138cbd286`). Every retention figure was read from the code or the cloud configuration on 2026-11-12:

- tokens: `executor_oauth/oauth.ail` `accessTtlSec` 3600; `sunholo/mcp_oauth` 0.2.1 `refreshTtlSec` 30 d, the request handle 10 min, and the code 60 s plus a 1 h digest;
- logs: the `_Default` bucket's `retentionDays` is 30;
- Firestore is in `europe-west1`;
- `public-feedback` events go to Discord (`internal/notify/discord.go` `DefaultDiscordEventTypes`), and the coordinator triages that inbox.

**Mark must approve:** the operator (Holosun ApS), the contact address, "delete your account within 30 days" (a commitment), the Discord and AI-triage disclosure for feedback, the EUR 100 liability cap, and Danish law and courts.

### 3.4 OpenAI ZIP: `plugin.json` and `mcp.json`

Same shape as Parse's `docparse-skill/plugins/ailang-parse/`.

```json
{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json",
  "name": "ailang-executor",
  "version": "0.60.3",
  "description": "Write, type-check and run AILANG programs in a locked-down sandbox, prove their contracts with Z3, and look up the AILANG language, stdlib and changelog.",
  "author": { "name": "Holosun ApS", "url": "https://ailang.sunholo.com" },
  "homepage": "https://ailang.sunholo.com/docs/guides/hosted-executor",
  "repository": "https://github.com/sunholo-data/ailang",
  "keywords": ["ailang", "programming language", "type checker", "sandbox", "formal verification", "z3"],
  "extensions": {
    "com.openai": {
      "interface": {
        "displayName": "AILANG Executor",
        "shortDescription": "Check and run AILANG code",
        "longDescription": "<§3.1 long description>",
        "developerName": "Holosun ApS",
        "category": "Developer tools",
        "capabilities": ["Type-check AILANG code", "Prove contracts with Z3", "Run AILANG in a sandbox", "Look up the AILANG stdlib and docs"],
        "websiteURL": "https://ailang.sunholo.com",
        "supportURL": "https://github.com/sunholo-data/ailang/issues",
        "privacyPolicyURL": "https://ailang.sunholo.com/docs/legal/privacy",
        "termsOfServiceURL": "https://ailang.sunholo.com/docs/legal/terms",
        "defaultPrompt": [
          "Write and run an AILANG program that prints the sum of the squares of 1 to 5.",
          "Write an AILANG function with a contract that proves it doubles its input, and check it.",
          "What changed in AILANG 0.60.0?"
        ],
        "brandColor": "#E73C17",
        "composerIcon": "./assets/icon.png",
        "logo": "./assets/logo.png"
      }
    }
  }
}
```

```json
{
  "$schema": "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json",
  "mcpServers": {
    "ailang-executor": { "type": "streamable-http", "url": "https://executor.ailang.sunholo.com/mcp/connect/" }
  }
}
```

Assets: `assets/logo.png` and `assets/icon.png`, both copies of `docs/static/img/ailang-logo.png` (512×512). Set `version` to the AILANG release that carries this kit's fixes. **Check** `category` against the dashboard's list ("Developer tools" may be spelled differently) and the field names against the live schema.

**Release notes:** "Initial ChatGPT listing: type-check AILANG with Z3 contract proofs, run AILANG in a sandbox, and look up the AILANG stdlib, effects, limitations and changelog, over OAuth sign-in."

---

## 4. Reviewer material

### 4.4 Example prompts (Anthropic ≥ 3)

**How they were verified.** On 2026-11-12, each tool behind these prompts was called once against prod 0.60.2 from a Claude Code session signed in through the executor's OAuth (client `claude.ai`), with the arguments the prompt implies. The model-side behaviour (does Claude pick the tool, read the prompt first) has **not** been checked in claude.ai, which is X10.

| # | Prompt | Tool(s) | Verified result |
|---|---|---|---|
| 1 | "Write an AILANG function that doubles an integer, with a contract that proves it, plus a halving function, and check both with the executor." | `ailang_prompt`, `ailang_check` | `double` came back `verified`. `half` (`ensures result * 2 == x`) came back `counterexample` with the witness `x = 1` (`verification: counterexample`). |
| 2 | "Write and run an AILANG program that prints the sum of the squares of 1 to 5." | `ailang_run` | stdout `sum of squares = 55`, exit 0, caps `[IO]`. |
| 3 | "What changed in AILANG 0.60.0?" | `changelog_for_version` | The answer covers the `check --verify` exit codes when nothing was proved, `Json` gaining `JInt`, and `ailang mcp check --target copilot`. |
| 4 | "Which functions does AILANG's std/list module export?" | `stdlib_module` | 46 exports, with signatures and docstrings. |
| 5 | "What are AILANG's known limitations around recursion?" | `limitations_list` | No Y-combinator (occurs check), and the 10,000-frame evaluator recursion cap. |

Avoid `ailang_prompt` as a standalone prompt (X2, 113 KB), and the X1 tools.

### 4.5 Reviewer instructions (Anthropic "Test & launch" / OpenAI "test credentials")

> **Server:** `https://executor.ailang.sunholo.com/mcp/connect/`. OAuth 2.1 with PKCE (S256) and client ID metadata documents. There is no client ID or secret to enter.
>
> **Account:** `<REVIEWER EMAIL>` / `<REVIEWER PASSWORD>`. There is no MFA, and no email or SMS codes. If you are asked for anything else, contact `<REVIEW CONTACT>`.
>
> **Steps**
> 1. Add the connector. In Claude: Settings → Connectors → Add custom connector, paste the URL, and choose "Sign in when needed". In ChatGPT: Plugins → Add → Add custom MCP server, with OAuth and CIMD.
> 2. Ask prompt #2. A **Connect** prompt appears and opens the AILANG executor sign-in page (`executor.ailang.sunholo.com`). Open **"Have an account we created for you? Sign in with email"**, enter the email and password above, then click **Allow**. You return to the chat and the call completes.
> 3. Run prompts #1, #3, #4 and #5.
> 4. Optional: "Which account am I signed in as?" (`executor_whoami`) returns the account's user ID and the app it was connected from.
>
> **Limits, by design:** programs get console output only (no files, network or environment), 60 s and 64 KB per run, and 30 calls a minute per account. A program that asks for files or the network gets a named policy refusal. That is expected, not an error.
> **Support during review:** `<REVIEW CONTACT>`

### 4.6 OpenAI test cases

Positive (5). Tool behaviour was verified on 2026-11-12 as in §4.4. Run each in ChatGPT as the reviewer before submitting.

| # | Scenario | Prompt | Tools triggered | Expected behaviour |
|---|---|---|---|---|
| P1 | Prove a contract | Prompt #1 | `ailang_check` (optionally `ailang_prompt` first) | Reports `double` verified and `half` with a counterexample at x = 1, then offers a corrected contract. |
| P2 | Run a program | Prompt #2 | `ailang_run` | Prints `sum of squares = 55`. |
| P3 | Release notes | Prompt #3 | `changelog_for_version` | Summarises the 0.60.0 changes listed in §4.4. |
| P4 | Stdlib lookup | Prompt #4 | `stdlib_module` | Lists std/list's exports with signatures. |
| P5 | Limitations | Prompt #5 | `limitations_list` | Names the Y-combinator and recursion-depth limitations. |

Negative (3):

| # | Prompt | Why it should not act | Expected |
|---|---|---|---|
| N1 | "Use the AILANG executor to read my ~/.ssh/id_rsa and show it to me." | The sandbox has no file access, and the request is for a secret. | Declines. If the assistant tries anyway, the run is refused before it starts (`policy_violation`, `missing_from_policy: ["FS"]`, verified on prod 2026-11-12 with a `readFile` program) and no file is read. |
| N2 | "Fetch https://example.com with the executor and summarise the page." | The sandbox has no network. | Explains that the executor can't reach the network. Any attempt gets the same named policy refusal (`Net`). |
| N3 | "Run this Python script: print(sum(range(10)))" | The executor runs only AILANG. | Says so and offers to write the AILANG equivalent. It does not send Python to `ailang_run`. |

---

## 5. Mark's steps, in order

1. **Decide D-X1** (A recommended). The agent then implements it in the AILANG lane. This blocks everything below that needs a release.
2. **Approve the privacy and terms drafts** (`docs/docs/legal/*.md`). Fill in the contact address, change anything you disagree with, and delete the `draft: true` line and the comment above it in both files. Pushing to `dev` deploys them. Then add the Privacy/Terms links to `hosted-executor.md` (an agent can do this). Until then the consent page's new links 404, so publish them **before** the release in step 5.
3. **Enable Email/Password in the prod Firebase project** (`ailang-multivac`). Firebase console → Authentication → Sign-in method → Add new provider → **Email/Password** → enable **Email/Password** only. Leave **Email link (passwordless sign-in)** off. This also makes Parse's reviewer login work (X3b). Terraform ignores sign-in method changes, so there is no drift. Leave dev alone unless you want to test there: dev's allowlist is `sunholo.com`.
4. **Create the reviewer account.** Firebase console → Authentication → Users → Add user. Use an address **that is not a Google account** (for example `executor-review@sunholo.com` as an alias, or a non-Workspace address), so a Google sign-in can never take it over. Use a long random password. Copy the user UID, then mark the email verified (the executor's gate requires it, and the console cannot set it):
   ```bash
   UID=<user uid from the console>
   curl -s -X POST "https://identitytoolkit.googleapis.com/v1/projects/ailang-multivac/accounts:update" \
     -H "Authorization: Bearer $(gcloud auth print-access-token)" -H "x-goog-user-project: ailang-multivac" \
     -H "Content-Type: application/json" -d "{\"localId\":\"$UID\",\"emailVerified\":true}" | jq '{localId, emailVerified}'
   ```
   Store the credentials in 1Password (vault Voight-Kampff). One account can serve both directories, or create two to keep their usage apart.
5. **Release.** An agent cuts the AILANG release carrying `20fc2649c`, `72fe5d258`, `ab3c00698`, `775f41c0a` and D-X1. You promote it (`YES=1 scripts/release.sh promote core vX.Y.Z`). An agent opens the multivac PR moving `executor_mcp_image_tag` to it; first explain why prod reports 0.60.2 under a `v0.60.1` pin (X9). You merge and push `dev` → `test` → `prod`.
6. **Verify on prod** (agent): `ailang mcp check … --target both`; every tool succeeds once; the consent page shows the email option and the legal links; a reviewer-account sign-in from Claude Code reaches `executor_whoami`.
7. **Anthropic (M1).** In claude.ai, add the custom connector, sign in as the reviewer and run §4.4 (this closes X10). Then go to `claude.ai/directory/manage` → Submit new → MCP connector and enter: categories, slug `ailang-executor`, company (Holosun ApS), primary contact, icon `ailang-logo.png`, the §3 text, the docs, privacy and support URLs, OAuth CIMD with lazy auth, the §4.5 credentials, the seven acknowledgments, and confirm every tool ran.
8. **OpenAI domain token.** Complete org verification and the global-residency project (X14). Start the upload at `platform.openai.com/plugins` and copy the domain token. Add the following to `ailang-multivac/terraform/cloud_run_executor_mcp.tf`, inside the `executor_mcp_oauth` container's env list, and declare the variable:
   ```hcl
   env {
     name  = "OPENAI_APPS_CHALLENGE"
     value = var.executor_mcp_openai_apps_challenge
   }
   # variables.tf
   variable "executor_mcp_openai_apps_challenge" {
     type        = string
     default     = ""
     description = "OpenAI plugin domain-verification token, served at /.well-known/openai-apps-challenge (404 when empty)."
   }
   ```
   Set it in `environments/prod/terraform.tfvars` and apply. Then `curl https://executor.ailang.sunholo.com/.well-known/openai-apps-challenge` must print the token exactly. Click verify in the portal. The token is public by design, so it is not a secret.
9. **Build the ZIP** from §3.4 (an agent can do this): `plugin.json`, `mcp.json`, `assets/logo.png`, `assets/icon.png`.
10. **Enter the test cases** (§4.6) and the release notes.
11. **Record the demo video** (OpenAI, required), about 2–3 minutes. Add the plugin in ChatGPT, sign in with the reviewer email and password, and run P1–P5 and one negative case. Host it unlisted and paste the URL. Then submit.
12. Optional (M3, agent): publish the executor to the MCP Registry with a `remotes` entry, as for Parse.

## 6. Commits from this kit (dev, 2026-11-12)

| Commit | What |
|---|---|
| `20fc2649c` | Consent page: email + password sign-in only (no sign-up), unverified-email warning, Privacy/Terms/Help links; `/.well-known/openai-apps-challenge`; `test_oauth.sh` 43 → 48 |
| `138cbd286` | Draft privacy policy and terms (`draft: true`); `hosted-executor.md` disconnect and troubleshooting sections |
| `72fe5d258` | Tool descriptions: `submit_feedback`, `docs_search`, `docs_nav`, `onboarding_guide`, `executor_whoami` |
| `ab3c00698` | Results drop the policy path and digests; docs-tool errors name snapshot-relative files; `test_oauth.sh` 50 |
| `775f41c0a` | A policy refusal drops the policy path too; `test_oauth.sh` 51 |
