# M-MCP-FILE-HANDOFF — a user's uploaded file reaches a remote MCP tool

**Status:** Planned (design, awaiting Mark's decisions D1–D4)
**Created:** 2026-10-07
**Program lane:** extension (a `sunholo/*` package + docparse adoption). No core change is expected; see the Conflict Surface.
**Research:** [m-serveapi-directory-ready-file-handoff-research.md](../v0_51_0/m-serveapi-directory-ready-file-handoff-research.md) (primary sources, fetched 2026-10-07)
**Related:** [m-serveapi-directory-ready.md](../v0_51_0/m-serveapi-directory-ready.md) · [m-mcp-oauth-package.md](../v0_52_0/m-mcp-oauth-package.md)

## Problem

AILANG Parse's listed connector (`/mcp/connect/`, OAuth 2.1) works in claude.ai: sign-in, a sample parse and a public-URL parse were all verified on prod on 2026-10-07. But the most natural request fails: **"here's a file I uploaded, parse it"**.

Trigger incident (claude.ai, 2026-10-07):
- The user uploaded `AGM.docx` (14 KB). Claude had it in its code sandbox, but `mcpParse` takes the file as base64 `content`.
- 19 KB of base64 hit Claude's output limit, so Claude **silently parsed the file itself** with a hand-written Python zip reader. That reader saw body text only (no headings, comments, footnotes or lists) and invented the style names.
- Prod logs show the tool call never reached the server.

**Root cause.** MCP tool arguments are model output. No shipped mechanism moves user-file bytes to a remote MCP server in Claude, and none exists in the MCP spec: SEP-2631 is an open draft, and Claude doesn't implement drafts. ChatGPT has one (`openai/fileParams`). Every other path must move bytes **with code, not with the model**.

## What the research established (primary sources)

| Fact | Consequence |
|---|---|
| ChatGPT: `_meta["openai/fileParams"]` makes the host pass `{download_url, file_id, mime_type?, file_name?}`; submission rejects any other shape | ChatGPT web gets a native path. Dev-mode support, URL expiry and size limits are not documented, so we test them |
| Codex: "Disallow custom MCPs from uploading files via fileParams" (source) | Codex uses the code path (local file → upload URL → `curl`) |
| claude.ai sandbox egress: off / package managers only (Team default) / specific domains / all domains. Enterprise default off. Free/Pro/Max: only a toggle is documented | The claude.ai code path works **only if** egress reaches our domain. We must detect that and say so clearly |
| MCP traffic bypasses the egress setting | Tool calls always work; only the byte upload depends on egress |
| MCP Apps: Claude renders widgets on web, Desktop and iOS; widgets may `fetch` declared `connectDomains`; no picker API; `<input type=file>` inside the host iframe is undocumented | Possibly a Claude path that doesn't depend on egress, but only a spike can tell |
| ChatGPT widgets: `window.openai.uploadFile` / `selectFiles` | A second ChatGPT path |
| SEP-2631 (draft, authored at OpenAI): `files/authorizeUpload` returns `{transport:"https", method:"POST", url, headers, multipart:{fileField, fields:{token}}, expiresAt}`, and the client passes a file URI to `tools/call` | **Shape our descriptor exactly like this** so that adopting the spec later is a rename |
| Claude limits: tool result about 150k characters, 240 s per call, 30 MB per uploaded file | Size caps and timeouts |

## Design

### The upload handoff (all clients that can run code with network access)
1. `createUpload(filename, mimeType, sizeBytes?)` (signed in, `@mcp_auth("oauth2")`) returns a **SEP-2631-shaped descriptor**:
   `{upload: {transport:"https", method:"POST", url:"https://docparse.ailang.sunholo.com/uploads", multipart:{fileField:"file", fields:{token:"<one-time>"}}, expiresAt}, fileRef:"dpfile:<id>", curl:"curl -F token=… -F file=@<path> <url>"}`.
   The ready-made `curl` line keeps the model's job to one copy-paste.
2. `POST /uploads` (multipart; serve-api already streams multipart uploads to a temp file):
   - checks the token: single use, valid 10 minutes, stored only as a digest, bound to the account and size cap;
   - stores the bytes in the existing temp bucket, which deletes them after 1 day; we also delete them after first use;
   - returns `{fileRef, sizeBytes, sha256}`.
3. `mcpParse`, `mcpConvert` and `editDocument` accept `fileRef`. The server checks it belongs to the caller's account, reads it and deletes it.
4. **Egress-blocked clients get a clear answer, never a silent fallback.** Tool descriptions say: *"If the upload fails with a network error, tell the user their assistant can't reach docparse.ailang.sunholo.com and how to allow it (Settings → Capabilities → network access), or ask for a link. Never parse the file yourself instead."*

### ChatGPT native
`mcpParse`, `mcpConvert` and `editDocument` gain an optional `file` parameter declared in `_meta["openai/fileParams"]`, with OpenAI's required 4-property schema. When it's present, the server fetches `download_url` immediately (one fetch under `Net[scope=public]`, size capped), then parses.

### Inline content
`content` stays for text (Markdown in for convert) and small binaries, capped at about 100 KB with an error that points to `createUpload`.

### Reuse: `sunholo/mcp_files` package (the D1 recommendation)
A pure core (token mint/verify, descriptor builder, `fileRef` codec, the fileParams schema JSON) plus hooks (`put/get/del` bytes, `account`), modelled on `sunholo/mcp_oauth`. docparse adopts it; any future AILANG service listed in the directories gets file input for free.

## Decisions for Mark

- **D1 — Package or docparse-only?** *Recommend:* package `sunholo/mcp_files`, plus docparse adoption, as we did for OAuth. It's about one extra day, and it's reusable.
- **D2 — The claude.ai egress dependency.** *Recommend:* accept it.
  - Document how to allow `docparse.ailang.sunholo.com`, and make the failure message explicit.
  - Raise it with Anthropic (the closed MCP issue #2240 asked for connector domains to be auto-allowlisted).
  - Ask in the directory submission how they expect file input to work.
- **D3 — MCP Apps upload widget.** *Recommend:* a **spike first** (half a day): does `<input type=file>` + `fetch` to a declared `connectDomain` work inside claude.ai's widget iframe? If yes, it becomes the primary claude.ai path, because it doesn't depend on egress.
- **D4 — Free-tier upload limits.** *Recommend:* use the plan's existing file-size limit (`maxFileSizeMb`). Uploads count as part of the request, not as a separate quota.

## Conflict Surface

- **Core change: none expected.** serve-api multipart (streamed to a temp file) and `@raw` routes cover `POST /uploads`.
- **Risk:** the binary body bytes must reach storage unchanged. Verify with a sha256 round trip in the e2e test.
- **The listed surface's tool schemas gain optional parameters.** That's backwards compatible for MCP clients, and `/mcp/` (the agent surface) is unchanged.
- **A @route-path collision is impossible:** `/uploads` is not a serve-api builtin path.

## Milestones

| # | What | Accept when |
|---|---|---|
| F0 | **Stop silent fallbacks** (docparse): new tool descriptions; `content` cap + error | Re-run the AGM upload in claude.ai: Claude uses the tools or tells the user what to do, never a local parse (transcript) |
| F1 | **Spike D3**: a minimal MCP App with a file input on dev | Recorded: does a file picker + fetch work in claude.ai web and Desktop (yes/no, with screenshots) |
| F2 | **`sunholo/mcp_files` 0.1.0**: core + hooks; Z3 on token expiry and single-use; tests | `pkg quality` shows no gates; descriptor JSON matches SEP-2631 field for field (golden test) |
| F3 | **docparse adoption**: `createUpload`, `POST /uploads`, `fileRef` on the three tools | e2e: create → curl upload → parse a 1.5 MB DOCX; sha256 round trip; replay refused; expired token refused; another account's `fileRef` refused |
| F4 | **ChatGPT `openai/fileParams`** on the three tools | ChatGPT web (dev mode) parses an uploaded DOCX, or an explicit "dev mode doesn't fill fileParams" finding, which then needs a listed-app test |
| F5 | **Cross-client matrix + docs**: AGM, a 1.5 MB DOCX, a PDF and an XLSX in claude.ai (egress on and off), Desktop, Claude Code, ChatGPT and Codex; update connect.html and the listing text | Every cell is "Parse output" or "clear instruction", with no silent fallbacks |

**Order:** F0 and F1 can run in parallel now; then F2 → F3 → F4 → F5. Estimate: 3–4 days.
