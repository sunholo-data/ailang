---
title: Privacy policy (hosted AILANG executor)
description: What the hosted AILANG executor and docs server collect when you use them from Claude, ChatGPT or another AI assistant, why, for how long, and how to remove it.
reviewed: 2026-11-12
reviewBy: 2027-02-12
---

# Privacy policy: hosted AILANG executor

**Last updated:** 2026-11-12

This policy covers two hosted services:

- the **AILANG executor**, `https://executor.ailang.sunholo.com/mcp/connect/`: an MCP server that
  checks and runs AILANG code in a sandbox and answers questions about the AILANG documentation.
  You sign in to use it.
- the **AILANG docs server**, `https://mcp.ailang.sunholo.com/mcp/`: the same documentation tools,
  with no sign-in.

The services are operated by **Holosun ApS** (CVR 44324687), Denmark ("we"). Contact:
[ailang@sunholo.com](mailto:ailang@sunholo.com).

It does not cover the AI assistant you use the services from. Claude, ChatGPT, Gemini and other
assistants handle your conversation, and the tool results we return to them, under their own
privacy policies.

## 1. What we collect

| What | When | Why | How long |
|---|---|---|---|
| **Your account:** sign-in provider user ID, email address, whether the address is verified, display name and profile photo URL, and sign-in times. For an account we create for you (for example, a directory reviewer's), also the email address and a salted password hash. | The first time you sign in, with Google or with an email and password we gave you, through Google Firebase Authentication | To know who is calling, to apply the per-account rate limit, and to let you sign in again | Until you ask us to delete your account |
| **Connection records:** which app you connected (its client ID, for example `https://claude.ai/...` or `https://chatgpt.com/...`), where the sign-in returns to, and your Firebase user ID, stored with **SHA-256 digests** of the codes and tokens we issued, never the codes or tokens themselves | Each time you connect an app, and each time the app renews its access | To check each tool call's token, and to let the app renew access without you signing in again | Sign-in requests: 10 minutes. Authorization codes: usable for 60 seconds, their digest kept for an hour to detect reuse. Access tokens: 1 hour. Refresh tokens: 30 days, replaced each time they are used. Expired records are deleted automatically. |
| **The code and expressions you run or check** | Each `ailang_run`, `ailang_eval` or `ailang_check` call | To run or check them and return the result | Not stored. The code is written to a temporary directory for that one call, which is deleted when the call ends. It is never logged. |
| **Call logs:** the tool name, outcome, duration, size of the code, a short hash of the code (so repeated identical calls are visible), the first error line (up to 300 characters, which can quote part of your code), your Firebase user ID and the app's host name (for example `claude.ai`) | Each tool call | To operate the service, find faults, prevent abuse and count usage | 30 days (Google Cloud Logging) |
| **Request logs kept by our hosting provider:** IP address, user agent, time, path and response status | Each HTTP request, including to the docs server | Security and operation | 30 days |
| **Documentation queries:** the search terms, versions and names you ask the documentation tools for | Each documentation tool call | To answer them | Only in the logs above |
| **Feedback you choose to send** with the `submit_feedback` tool: title, description, category, AILANG version, an optional code snippet, an optional package name and an optional contact detail | Only when you or your assistant send feedback | To read it, reply if you left a contact, and fix what it reports | Until the report is resolved and then archived. Ask us to delete it at any time. |

We do **not** receive your conversation with the assistant, your chat history, or files you give
the assistant, only the arguments of the tool calls it makes. We do not sell personal data, use it
for advertising, or use it to train AI models.

## 2. Who processes it (sub-processors)

| Who | What for | Where |
|---|---|---|
| Google Cloud (Cloud Run, Firestore, Cloud Logging, Pub/Sub) | Hosting, connection records, logs, feedback delivery | EU (Belgium, `europe-west1`) |
| Google Firebase Authentication | Sign-in and account records | Google's global infrastructure (may include the US) |
| Google Vertex AI (Gemini) | Only for the executor's A2A agent (used from Gemini Enterprise), which writes code with Gemini. The MCP tools listed in the Claude and ChatGPT directories do not use it. | Google's global endpoint |
| Discord | A notification that a feedback report arrived, which can include its title | US |
| AI coding tools (for example Anthropic Claude or OpenAI Codex) | Our team may use them to triage and act on feedback reports | US |

Transfers outside the EU rely on the providers' Standard Contractual Clauses and, where
applicable, the EU–US Data Privacy Framework.

## 3. Legal basis (GDPR)

- Running the service you asked for (Art. 6(1)(b)): your account, connection records, running
  your code.
- Our legitimate interest in operating a secure, reliable service and preventing abuse
  (Art. 6(1)(f)): logs and rate limits.
- Your consent (Art. 6(1)(a)): feedback you choose to send.

## 4. Disconnecting and deleting

- **Disconnect an app:** remove the connector in the assistant (Claude: Settings → Connectors;
  ChatGPT: Settings → Plugins). Its access token stops working within an hour at most, and its
  refresh token after 30 days or the next time it would be used, whichever comes first.
- **Delete your account:** email us at the address above. We delete your sign-in account and every
  connection record within 30 days. Logs expire on their own after 30 days.
- **Your rights:** you can ask for a copy of your data, its correction or deletion, restriction of
  processing, or object to it, and you can complain to the Danish Data Protection Agency
  (Datatilsynet).

## 5. Cookies and browser storage

The sign-in page stores your Firebase sign-in state in your browser (IndexedDB) so that it can
recognise you on that page. It sets no tracking or advertising cookies, and the services use no
analytics.

## 6. Children

The services are not directed at children under 13, and we do not knowingly collect their data.

## 7. Changes

We will update the date at the top when this policy changes, and announce material changes in the
AILANG changelog.
