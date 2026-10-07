# Sprint plan: M-MCP-FILE-HANDOFF

Design: [m-mcp-file-handoff.md](m-mcp-file-handoff.md). D1 decided (reusable split); D2–D4 as recommended.

**Estimate:** 4–5 days.

**Registry reuse:**
- `sunholo/mcp_oauth`: pattern only (hooks + pure core).
- `sunholo/auth` / `sunholo/firestore`: no change. docparse adopts as before.
- **New package: `sunholo/mcp_files`.** `pkg search` found no upload or file-handoff package.

## Milestones

| ID | Lane | What | Acceptance (test or command) | Deps |
|---|---|---|---|---|
| F0 | docparse | Stop silent fallbacks: tool descriptions for the uploaded-file path + an inline `content` cap of about 100 KB | e2e: 150 KB `content` → an error naming `createUpload`. Lint: the descriptions forbid local parsing. claude.ai AGM re-test transcript: no local parse | — |
| F1 | spike | A throwaway MCP App with a file input + fetch to a declared `connectDomain`, deployed to the dev project and tested in claude.ai web + Desktop | Recorded yes/no per host, with screenshots; the service is deleted afterwards | — |
| F1b | AILANG serve-api | `@mcp_file("param")` → `_meta["openai/fileParams"]` + the 4-property schema + binding; `securitySchemes` on gated tools; `_meta["mcp/www_authenticate"]` on 401 tool results | A golden tools/list test against OpenAI's documented shape; an e2e that binds a file object; `mcp_check_e2e_test` unchanged; AILANG patch release | — |
| F2 | package | `sunholo/mcp_files` 0.1.0: tokens (Z3: expiry, single use), the SEP-2631 descriptor, the `fileRef` codec, `fetchFileParam` under `Net[scope=public]`, storage hooks | `pkg quality` shows no gates; a descriptor golden test matches SEP-2631 field for field; mutants killed | — |
| F3 | docparse | `createUpload`, `POST /uploads`, `fileRef` on mcpParse/mcpConvert/editDocument, temp-bucket hooks, a privacy line | e2e: create → curl → parse a 1.5 MB DOCX (sha256 round trip); replayed, expired and cross-account tokens refused | F2 |
| F4 | docparse | `@mcp_file` on the three tools | ChatGPT web (dev mode) parses an uploaded DOCX, or a recorded "dev mode doesn't fill fileParams" | F1b released |
| F5 | all | Cross-client matrix + connect.html + listing text | Every cell is "Parse output" or "clear instruction"; no silent fallbacks | F3, F4 |
