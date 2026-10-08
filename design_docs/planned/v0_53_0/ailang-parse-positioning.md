# AILANG Parse positioning as a Claude / ChatGPT connector

**Status:** Research and recommendation. Not a sprint plan. Nothing here is committed.
**Date:** 2026-10-07. Vendor pages were read today. Some OpenAI help pages could only be read through a headless browser.
**Question (Mark):** ChatGPT now parses uploads itself. Claude reads documents and writes its own parsers. Anthropic and OpenAI both ship Office add-ins. What is still defensible about AILANG Parse as a connector?
**Inputs:**
- Public repo `ailang-parse` @ `0bdd89e`, package v0.50.0.
- Private API `docparse` @ `a5f3313`.
- Submission kit `design_docs/planned/v0_51_0/m-serveapi-directory-ready-submission-kit.md` (on `origin/dev`, `1e908346e`).
- File handoff design `design_docs/planned/v0_53_0/m-mcp-file-handoff.md` (on `origin/dev`).
- A local head-to-head run today (§4).

---

## 0. Answer in one screen

**The direct competitor for a document in a chat is the assistant itself, not another parser vendor.**

- **Inside Word, the parsing battle is already lost.** Claude for Word went GA on 2026-05-07 on paid plans and reads "text, comments, tracked changes, footnotes, tables, and bookmarks". ChatGPT for Word/Excel/PowerPoint is on every plan, including Free. We should not fight there.
- **The gap is files that are not open in Word.** These are attachments, uploads, files in agent pipelines, Claude Code and Codex. Both vendors document a lossy path for them:
  - Claude: "For other documents, Claude extracts text only".
  - Claude's own docx skill now reads with plain `pandoc -t markdown`.
  - ChatGPT runs docx/pptx through text-based retrieval: extract text, stuff the first 110k tokens into context, vector-search the rest.
  - Neither vendor documents what happens to tracked changes, comments, headers/footers or merged cells.
  - In today's run (§4), every naive path dropped revisions, comments, and headers/footers. That covers the ad-hoc zipfile/ElementTree script, python-docx defaults and MarkItDown, which Claude's pptx/xlsx skills use.

**Recommended positioning (one line):**

> **AILANG Parse reads Office files exactly, and the same way in every assistant and agent: tracked changes with author and date, comments, headers and footers, and merged cells. No AI touches the content, and the same engine runs on your own machine when a file must not leave it.**

**Three pillars:**
1. Nothing hidden: revisions and annotations.
2. Same answer everywhere.
3. Runs where the file must stay.

Generation with templates is a supporting feature, not a pillar. Redaction is the most valuable gap to build next (§6.3).

**The biggest risk to the listing is not positioning. It is that the core story fails on today's surface:**
- **Uploads don't arrive in claude.ai.** A user's uploaded file does not reach the connector there. A 14 KB docx sent as base64 hit the output limit, and Claude quietly parsed it itself (`m-mcp-file-handoff.md`).
- **Two of the drafted claims are false on our own samples.** These are "footnotes kept" and "spreadsheet formulas kept" (§4).
- **Our headline benchmark is self-authored, and its ground truth is derived from our own outputs** (§2.1).

---

## 1. Competitor baseline (primary sources, read 2026-10-07)

### 1.1 ChatGPT (OpenAI)

**Supported types and limits** ([Uploading files](https://help.openai.com/en/articles/8555545-file-uploads-faq)):
- Types: DOCX, PDF, TXT, XLSX/XLS/CSV/TSV, PPTX.
- Size: 512 MB per file. 2M tokens per text or document file. About 50 MB for spreadsheets.
- Rate: 80 files per 3 h. Free accounts get 3 uploads per day.

**How each type is processed** ([Optimizing file uploads](https://help.openai.com/en/articles/10029836-optimizing-file-uploads-in-chatgpt-enterprise)):
- **docx/pptx/txt/md use text-based retrieval.** The text is extracted. Up to 110k tokens are stuffed into context, and the rest goes to a vector store searched by hybrid search.
- **xlsx/csv go through Code Interpreter**, which runs Python.
- **"Images embedded in files other than PDFs are not processed."**
- **Visual PDF retrieval is Enterprise-only** ([visual retrieval FAQ](https://help.openai.com/en/articles/10416312-visual-retrieval-with-pdfs-faq)).

**What OpenAI does not say:** no OpenAI page states whether tracked changes, comments, headers/footers, merged cells or formulas survive.

**ChatGPT for Word, Excel and PowerPoint** ([help article](https://help.openai.com/en/articles/20001526-chatgpt-for-word)):
- One add-in covers all three apps, on all plans including Free.
- Enabled by default from 2026-10-01.
- Documented limits: complex formatting, tables and charts may need fixing. Plugins are not supported inside the add-in.
- Nothing is documented about redlining.

**Apps and plugins:**
- The Apps SDK is built on MCP ([Apps SDK](https://help.openai.com/en/articles/12515353-build-with-the-apps-sdk)).
- Apps now live under Settings → Plugins ([apps in ChatGPT](https://help.openai.com/en/articles/11487775-apps-in-chatgpt)).
- **ChatGPT has a host mechanism for passing user files to a tool, `openai/fileParams`.** Per the file-handoff research, it is the only shipped one.

**Redaction:** none first-party. The Adobe Acrobat plugin offers PDF redaction ([Adobe Acrobat plugin](https://openai.com/business/plugins/adobe-acrobat/)).

**Pricing:** tiers are Free / Go / Plus / Pro / Business / Enterprise ([pricing](https://chatgpt.com/pricing)). Dollar amounts did not render, so they are unverified. The Office add-in bills tokens against the plan allowance.

### 1.2 Claude (Anthropic)

**claude.ai uploads** ([Uploading files](https://support.claude.com/en/articles/8241126-uploading-files-to-claude), updated 2026-07-23):
- Types: PDF, DOCX, CSV, TXT, HTML, ODT, RTF, EPUB, JSON, XLSX.
- Size: 500 MB per file and 20 files per chat. Project files are capped at 30 MB.
- PDFs of 100 pages or fewer: text plus visuals. 101–1,000 pages: text only.
- **"For other documents, Claude extracts text only."**
- XLSX needs code execution switched on.

**API document blocks** ([PDF support](https://platform.claude.com/docs/en/build-with-claude/pdf-support)):
- A PDF is sent to the model as page images plus extracted text.
- ".xlsx or .docx are not supported in document blocks and must be converted to text or PDF first."

**Create and edit files** ([support article](https://support.claude.com/en/articles/12111783-create-and-edit-files-with-claude), [blog](https://claude.com/blog/create-files)):
- A sandbox that creates docx, pptx, xlsx and PDF.
- On by default for Free, Pro and Max.
- 30 MB per file.

**Agent Skills** ([anthropics/skills](https://github.com/anthropics/skills)), as of commit "Update docx, pptx, and xlsx skills (#1447)", 2026-07-17:
- **docx: reading is now plain `pandoc -t markdown`.** The older version used `pandoc --track-changes=all` plus raw-XML unpacking, and that flag has been dropped. Pandoc's default for `--track-changes` is `accept` ([pandoc manual](https://pandoc.org/MANUAL.html#option--track-changes)): insertions are silently merged, deletions disappear, and no author or date is kept.
- **docx: writing is strong.** It writes redlines in raw OOXML (`<w:ins>`/`<w:del>` with author and date), runs `validate.py` against the original to catch untracked edits, and handles comments with `comment.py`.
- **xlsx:** two openpyxl passes (formulas and values), with a mandatory LibreOffice recalc.
- **pptx:** reads with `markitdown`.
- **No skill has a redaction workflow.** A grep for "redact" over the four skills finds nothing.

**Claude for Microsoft 365** ([blog, 2026-05-07](https://claude.com/blog/collaborate-with-claude-across-excel-powerpoint-word-and-outlook), [Word docs](https://claude.com/docs/office-agents/word), [Excel docs](https://claude.com/docs/office-agents/excel)):
- Excel, Word and PowerPoint are GA on paid plans.
- Word reads "text, comments, tracked changes, footnotes, tables, and bookmarks". It has a tracked-changes edit mode and can "summarise counterparty redlines".
- Excel keeps formula relationships and cites cells.
- **This is the strongest direct overlap with our fidelity claim, but only for a document the user has open in Word on a paid plan.**

**Connectors directory:**
- Interactive "MCP Apps" launched 2026-01-26 ([blog](https://claude.com/blog/interactive-tools-in-claude)).
- **LlamaParse is already an Anthropic-verified connector** ([listing](https://claude.com/connectors/llamaparse)).

**Pricing** ([claude.com/pricing](https://claude.com/pricing)):
- Free $0, with file creation.
- Pro $20 a month, including the M365 integration.
- Max from $100.
- Team $25 / $125 per seat.

### 1.3 Dedicated parsers

Sources are per row. A ✗ means the feature is not documented or was not found in source.

| Vendor | How it handles Office files | Revisions with author/date | Comments | Hdr/ftr | Formulas | Deterministic | Deployment and residency | MCP | Price |
|---|---|---|---|---|---|---|---|---|---|
| [Unstructured](https://docs.unstructured.io/open-source/core-functionality/partitioning) | Native partitioners | ✗ | ✗ | DOCX only | ✗ | Office yes | SaaS, VPC on Business, ZDR claim | [Transform MCP](https://docs.unstructured.io/transform/overview.md) | 10k pages free, then $0.015/page ([pricing](https://unstructured.io/pricing)) |
| [LlamaParse](https://developers.llamaindex.ai/llamaparse/general/supported_document_types/index.md) | 130+ formats. LiteParse converts Office to PDF ([doc](https://developers.llamaindex.ai/liteparse/guides/multi-format/)). Cloud behaviour unverified | ✗ | ✗ | ✗ | ✗ | No on LLM/VLM tiers | SaaS, EU region, 48 h cache | **Claude-verified connector** | $1.25 per 1k credits; 1 to 45 credits/page ([pricing](https://developers.llamaindex.ai/llamaparse/general/pricing/index.md)) |
| [Docling](https://github.com/docling-project/docling) | Native OOXML backends | **✗ (insertions flattened, deletions dropped, accept-all text)** | **yes** (current `main`) | **yes** | ✗ (`data_only=True`, cached values only) | Office yes | Local | [docling-mcp](https://github.com/docling-project/docling-mcp) | Free (MIT) |
| [Azure DI Layout](https://learn.microsoft.com/en-us/azure/ai-services/document-intelligence/prebuilt/layout) | Text "as is". No table analysis for XLSX | ✗ | ✗ | Partial | ✗ | Model-based | Azure EU regions, containers | Content Understanding MCP "coming soon" | Layout $10 per 1k pages. DOCX: 3,000 characters = 1 page |
| [Google Doc AI Layout](https://docs.cloud.google.com/document-ai/docs/layout-parse-chunk) | DOCX/PPTX/XLSX via the Gemini-based parser | ✗ | ✗ | ✗ | Values only | No (Gemini) | Gemini versions "not compliant with Data Residency" | None found | $10 per 1k pages ([pricing](https://cloud.google.com/document-ai/pricing)) |
| [Reducto](https://docs.reducto.ai/cookbooks/redlined-legal-contracts) | Native types listed. `change_tracking` detects visual strike/underline (our inference: rendered, not OOXML) | Visual only, no author/date documented | ✗ | ✗ | ✗ | VLM tiers | EU on Growth plan, 24 h retention ([EU](https://docs.reducto.ai/security/eu-data-residency)) | [hosted MCP](https://docs.reducto.ai/mcp-server.md) | $10 per 1k pages ([pricing](https://reducto.ai/pricing.md)) |
| [Marker/Datalab](https://github.com/datalab-to/marker) | **Converts DOCX/PPTX/XLSX to PDF first** | ✗ by construction | ✗ | ✗ | ✗ | ML | On-prem on Enterprise | None found | $4 per 1k pages ([pricing](https://www.datalab.to/pricing)) |
| [Aspose.Words](https://docs.aspose.com/words/python-net/aspose-words-mcp-server/index.md) | Native object model (SDK) | **yes (full revision API)** | yes | yes | yes (Cells) | yes | Library or Docker | aspose-words-mcp (needs a licence) | Per call / licence |

**Takeaways:**
1. **MCP is table stakes.** LlamaParse is already in Claude's directory.
2. **Among parsers built for LLM pipelines, none documents revision metadata (who changed what, and when) from OOXML.**
   - Docling returns the accept-all text.
   - Reducto detects strike-through visually.
   - Only Aspose, a commercial SDK and not a pipeline parser, exposes the full revision model.
3. **Docling's DOCX backend on current `main` reads comments, headers/footers and footnotes.** Our OfficeDocBench shows Docling 2.84.0 at 0/2 comments and 0/2 headers. **That comparison is stale and must not be repeated without a re-run.**
4. **No public benchmark scores tracked changes, comments or formulas.**
   - [BizDocBench](https://github.com/DocSlicer/BizDocBench) has 9 DOCX files and tests neither.
   - OmniDocBench is PDF only.
   - This is an opening for OfficeDocBench, but only if its ground truth is made independent (§2.1).

**Redaction elsewhere:**
- Unstructured: Presidio-based, English only, dedicated or VPC deployments only ([doc](https://docs.unstructured.io/business/security-compliance/pii-redaction)).
- Azure and Google: separate PII services.
- Adobe Acrobat: PDF redaction, available as a ChatGPT plugin.
- **No one offers deterministic, OOXML-aware redaction as an MCP tool**, meaning redaction that also scrubs deleted-text revisions, comments, headers and document properties.

---

## 2. What AILANG Parse is and has (our evidence)

| Claim | Evidence (path or URL) | Status |
|---|---|---|
| Office, ODF, HTML, MD, CSV, EPUB, EML/MBOX, TEX and RTF are parsed by our own deterministic engine, with no AI | `ailang-parse/README.md` §intro; JSON output carries `"aiCallsUsed":0` (§4) | **Verified** |
| PDF defaults to `pdftotext`, escalates to local docling for scans, and uses AI only with `--pdf-backend ai` | `ailang-parse/README.md`; `/api/v1/capabilities` `pdfBackend` description | Documented |
| 16 input formats, 9 generation targets (DOCX, PPTX, XLSX, ODT, ODP, ODS, HTML, MD, QMD) | `ailang-parse/README.md` "Supported Formats" | Documented |
| Same input gives the same output | §4: 24/24 output files byte-identical across two runs | **Verified (local CLI)** |
| Generation from Markdown onto a brand template (`--reference-doc`, `--table-style`) | `ailang-parse/docs/templates.html`; README "Writing documents in Markdown" | Documented |
| Local verification tools `docparse-audit` and `docparse-render` (page renders, before/after compare) | `ailang-parse/README.md` "Local document verification" | Documented |
| Local CLI and in-browser WASM: "zero bytes sent" | `ailang-parse/docs/index.html`; `privacy.html` §2 | Documented |
| Hosted in EU (europe-west1); files not stored; request record kept 12 months; up to 10 KB of output kept for replay **unless history is turned off**; no training on content | `ailang-parse/docs/privacy.html` (updated 2026-10-06) | Documented. **Firebase sign-in runs in the US. Gemini (AI parses only) may run in the US.** |
| "Replay" | `docparse/docparse_api/services/request_log.ail:273-300` | **It returns stored blocks and re-renders them in another format. It does not re-execute the parse.** |
| `editDocument`: structured edit deltas on Office files, returns a copy | `docparse/docparse_api/services/api_server.ail:1901-1919`; `connect.html` | Documented. **No evidence that edits are written as tracked changes.** |
| One connector URL for Claude Code, claude.ai, Codex and ChatGPT, using OAuth 2.1 + PKCE + CIMD | `ailang-parse/docs/connect.html`; built on `sunholo/mcp_oauth` 0.1.1 (`docparse/ailang.toml:40`) | Verified on prod for claude.ai on 2026-10-07 (sample and public URL only) |
| Per-document pricing: Free 1,000 docs and 50 AI parses; Pro €29 for 100k; Business €99 for 500k | `ailang-parse/docs/pricing.html` | Documented |
| OfficeDocBench composite 92.6% vs Kreuzberg 71.3%, Raw OOXML 84.4% (62% coverage), Docling 64.0%, Unstructured 62.1%, LlamaParse 54.4% | `ailang-parse/benchmarks/officedocbench/results/summary.json` (run 2026-08-11) | Self-run; **see caveats** |
| Redaction | Only `convertRefRedact` exists, which redacts file paths from **error messages** (`docparse/docparse_api/services/convert.ail:238`) | **Does not exist as a feature** |

### 2.1 Caveats on our own numbers (read before quoting any of them)

1. **The ground truth is not independent.**
   - `benchmarks/officedocbench/annotate.py` says it "auto-generate[s] OfficeDocBench ground truth from existing golden files". Those goldens are AILANG Parse outputs (`benchmarks/office/golden/*.json`).
   - `index.html` nonetheless says "hand-verified ground truth".
   - **A benchmark scored against our own outputs cannot show we beat anyone on correctness.** It shows agreement with ourselves, and it rewards our omissions. For example, `poi_footnotes.docx.json`'s golden also lacks the footnote text.
2. **The headline number drifts across our own pages.**
   - 93.9% in the OfficeDocBench README (run 2026-04-06).
   - 92.6% in `summary.json` (2026-08-11).
   - "92%" on `index.html`.
   - 96.6% in `docparse/design_docs/competitive_landscape.md`.
3. **The contract count drifts too.** It is "28 contracts" in the `docparse` README and "63 Z3-verified contracts" in the GTM doc.
4. **The track-changes claim contradicts our own data.**
   - `index.html` says "0 of 5 other parsers tested extract track changes."
   - Our own heatmap shows Pandoc 3/3 and Raw OOXML 2/3, and the README's "Best Competitor" column says "Pandoc (3/3)".
5. **A careful stdlib script gets close.**
   - The benchmark's own "Raw OOXML" adapter is 623 lines of zipfile + ElementTree (`adapters/ooxml_adapter.py`). On the files it handles it scores 82.4% on DOCX, against our 88.6%.
   - **So the gap is to a hasty script, not to a careful one.** That sharpens the pitch: Parse is the careful parser that nobody writes again in each chat.
6. **Docling numbers are from v2.84.0.** Docling `main` now reads comments, headers and footnotes (§1.3).

---

## 4. Head-to-head, run today (local, $0, public samples only)

**Setup:**
- Engine: local `docparse` CLI (package v0.50.0) on AILANG v0.52.1.
- Samples: 14 public files from `ailang-parse/data/test_files/`.
- Baselines:
  - **(a) Naive.** Mimics what Claude wrote for Mark: zipfile + ElementTree over `word/document.xml`, joining `w:t` and reading `pStyle`.
  - **(b) python-docx 1.2.0** paragraph and table defaults.
  - **(c) MarkItDown**, which Claude's pptx skill and xlsx quick-look use.
  - **(d) openpyxl**, as Claude's xlsx skill uses it.
- Pandoc could not run on this machine (the wheel ships an x86 binary). Pandoc results are taken from OfficeDocBench.
- Not run: the hosted API. The plugin connector was not signed in, so all hosted claims rest on documentation.
- Scripts and outputs are in the session scratchpad and were not committed.

| Sample (sample id) | What is in the file | Naive zipfile/ET | python-docx | MarkItDown | **AILANG Parse** |
|---|---|---|---|---|---|
| `pandoc_tc_deletion.docx` | 1 tracked deletion | Deleted text silently gone. Reads as clean text | Same | Same | `change` block: `delete`, author `eng-dept`, 2014-06-25, text "n excessively modified". **But not anchored inline.** The position in the sentence is lost |
| `pandoc_tc_insertion.docx` | 1 tracked insertion | Inserted text merged invisibly | Same | Same | Text plus `change` block: `insert`, author, date, "two exciting" |
| `track_changes_move.docx` (`sample_docx_track_changes`) | Paragraph move | Moved paragraph appears **twice** with no explanation | – | – | `move-from` (struck) and `move-to` with author and date. The body still prints the paragraph twice, but now labelled |
| `comments.docx` (`sample_docx_comments`) | 5 comments, one spanning paragraphs, one nested | **0 comments** | 0 by default (the `comments` API sees 5 if asked) | 0 | All 5, each with author, timestamp and the **exact anchored text range** |
| `challenge_comment_ranges.docx` | 3 comments on phrases | 0 | – | – | 3, with author, date and range ("deadline is March 15") |
| `docx-hdrftr.docx` | Header and footer with tables | Body only: "Main document" | Same | Same | Header and footer content, including tables and colspan. The footer table renders messily in Markdown |
| `poi_footnotes.docx` | One footnote ("snoska") | Missing | – | – | **Also missing.** Footnote text dropped. The golden agrees, which is why the benchmark doesn't notice |
| `challenge_footnotes.docx` (**`sample_docx_footnotes`**) | Described as "Footnotes and endnotes" | – | – | – | **The file has no `footnotes.xml` at all.** The public sample is mislabelled |
| `challenge_numbering.docx`, `challenge_nested_lists.docx` | 3-level numbered and bullet lists | Keeps `ListNumber2`/`ListBullet3` style names, so depth is recoverable | – | Flat `1. 2. 3. …` | **Flat.** Each item is its own `list` block with no level; Markdown renders "1. 1. 1." **Nesting lost; the naive script keeps more here** |
| `challenge_formulas.xlsx` (**`sample_xlsx_formulas`**) | Formulas with no cached values (written by openpyxl) | – | – | `NaN` | **Formula cells empty**, and the formula text is not in the JSON. openpyxl's default load shows `=B2-C2`, `=SUM(B2:B3)` |
| `challenge_formula_cached.xlsx` | Same | – | – | – | Empty cells. `xlsx_parser.ail` has no formula extraction |
| `merged_cells.docx`, `challenge_formulas.xlsx` row 6 | Merged ranges | Flattened | Table object only | Flattened | `colSpan`/`merged` typed in JSON |
| Determinism | 12 files × 2 runs | – | – | – | **24/24 outputs byte-identical** |
| Speed | 12 files | – | – | – | 10.3 s wall total, about 0.86 s per file **including AILANG runtime start-up** (`pricing.html`'s "~50 ms" is the hosted in-process figure; not re-measured) |

**What this proves:**
- **Revisions and comments are the decisive gap.**
  - Every path an assistant reaches for by default silently drops deletions, folds insertions into the text, and loses comments and headers/footers. That is ad-hoc zipfile, python-docx defaults, MarkItDown, and per pandoc's manual the docx skill's plain `pandoc`.
  - **Parse returns them with author, date and anchor range.** This is the one result that is robust and easy to demonstrate.
- **Formulas, footnotes and list nesting are currently weaker than claimed.** The kit's long description says "footnotes, … spreadsheet formulas are all kept". **That is false today and would be caught by a reviewer running our own `sample_xlsx_formulas`.**
- **Deletions are not positioned inline.** For a contract reviewer, "what was deleted *where*" matters. Changes should be anchored like comments are.

---

## 5. Candidate pillars, assessed

The "survives the objection" column asks whether the pillar still holds when the user says "Claude/ChatGPT already does this".

| Pillar | Evidence for | Evidence against | Who it matters to | Survives the objection? |
|---|---|---|---|---|
| **Fidelity and structure** (tracked changes, comments, tables, formulas, layout) | §4: revisions, comments, headers and merged cells survive only in Parse among default paths. No LLM-pipeline parser documents revision metadata (§1.3). The vendor upload pages say "text only" (Claude) or text retrieval (ChatGPT) | Claude for Word reads comments, tracked changes and footnotes natively. Docling `main` now does comments, headers and footnotes. **Our footnotes, formulas and nesting have gaps (§4).** A careful 600-line script gets 82% (§2.1) | Legal ops, contract managers, policy and regulatory editors (buyer: legal-tech / GC office). Agent builders ingesting Office files | **Yes, narrowed to revisions and annotations outside Word.** No for "generic better parsing" |
| **Determinism and auditability** | 24/24 byte-identical; `aiCallsUsed:0`; request record per call; AILANG contracts | Claude's ad-hoc script is also deterministic *as code*; the issue is that it is a **different script each time**. "Replay" re-renders stored output; it does not re-run. Contracts are not a user-visible guarantee. The ground-truth circularity undercuts "verified" | Compliance, risk, eval and RAG teams; regulated industries; anyone who must show the same extraction twice | **Yes, if phrased as "the same parser every time, everywhere"** rather than "formally verified" |
| **Round-trip editing and generation** | Templates via `--reference-doc` with no LLM layout; 9 output formats; `editDocument` deltas; `docparse-render` before/after compare | Both assistants create docx/pptx/xlsx natively. Claude's docx skill **writes real tracked changes**, and we have no evidence that `editDocument` does. Download link valid 1 h. In claude.ai the input file can't reach us yet | Teams with a house template (proposals, reports); agents in Claude Code and Codex | **Mostly no** in chat. Yes for "on-brand documents from Markdown, repeatably", which is a niche |
| **Privacy** | Local CLI and WASM with the same engine; EU-hosted API; files not stored; no training; history switch | **Through a connector the document has already gone to Anthropic or OpenAI.** Parse adds a processor and removes no exposure. Firebase and Gemini may run in the US. Output is kept by default (10 KB) | DPOs, EU public sector, legal | **Not for the connector.** Yes for the local product. Use it as a bridge in the listing ("for files that must not leave your machine, the same engine runs locally"), not as the reason to install |
| **Redaction** | Nothing exists today. No competitor offers deterministic OOXML-aware redaction over MCP (§1.3). Parse already reads every part where hidden text lives (comments, `w:delText`, headers, footnotes, docProps), and `editDocument` already applies deltas | Building it right is a real feature: detection, application across all parts, residue proof. Unstructured, Azure and Google have PII services; Adobe redacts PDFs in ChatGPT | Legal (disclosure, FOIA/DSAR), HR, healthcare; anyone sharing a docx externally | **Yes, strongly, once built.** Assistants can *propose* what to redact but cannot reliably *prove* a docx is clean. **Don't claim it yet** |
| **Cross-assistant consistency** | One URL, one account across Claude Code, Claude, Codex and ChatGPT (`connect.html`); same engine in the CLI, SDKs (py/npm/go) and WASM. Each assistant otherwise parses differently (§1.1–1.2) | Users rarely use more than one assistant on the same file. Team value is speculative and not measured | Platform teams, agent builders, multi-vendor enterprises | **Yes, as a supporting proof point** for pillar 2 |
| **Cost and latency for agents** | Per-document price; Office is $0 marginal for us; ~0.9 s local wall time | Inside a chat the user already pays a subscription, and assistants don't vision-parse docx anyway. "100× cheaper" compares against per-page PDF APIs, not the assistant. No token-saving measurement exists | API and pipeline buyers (not chat users) | **No in the listing.** Keep it for the API page |
| **Compliance workflows** (redlines, contract review) | Revision and comment extraction with anchors is the demo that works; legal is the persona in the GTM doc | Claude for Word targets exactly "summarise counterparty redlines". We lack inline deletion anchoring, a compare tool and a redline summary tool | Legal ops at firms not standardised on M365 paid seats; inbound redlines by email; bulk review | **Partly.** Strong for redlines that arrive as attachments or in bulk, weak inside Word |

---

## 6. Recommendation

### 6.1 Positioning statement

> **Exact, repeatable reading of Office files for any AI assistant or agent: the tracked changes, comments and structure that uploads flatten, with no AI in the loop.**

Longer form for the website:

> **AILANG Parse reads Office files exactly, the same way in every assistant and agent. Tracked changes come with author and date, comments with the text they refer to, plus headers and footers and merged cells. No AI touches the content, and the same engine runs on your own machine when a file must not leave it.**

### 6.2 Three pillars and their proof points

1. **Nothing hidden: revisions and annotations.**
   - Proof: the §4 table (deletion, insertion, move and 5 comments, each with author, date and anchor, against default paths that drop all of them).
   - Vendor sources: "Claude extracts text only"; ChatGPT text retrieval; the docx skill's plain-pandoc read and pandoc's `accept` default.
   - Lead demo: `sample_docx_comments` and `sample_docx_track_changes`.
2. **Same answer, every time, everywhere.**
   - Proof: 24/24 byte-identical outputs; `aiCallsUsed: 0`; one connector URL across Claude Code, Claude, Codex and ChatGPT; the same engine in the SDKs, CLI and browser; a request record per call.
   - Pitch: "a parser, not a new script per conversation".
3. **Runs where the file has to stay.**
   - Proof: CLI and WASM with zero upload; EU-hosted API (Holosun ApS, Denmark); files not stored; no training; request-history switch.
   - State the US caveats (Firebase sign-in; Gemini for AI parses only) up front, not in the footnotes.

Generation with templates stays a feature line, not a pillar.

### 6.3 Do not claim (today)

- **"Footnotes … spreadsheet formulas are all kept"** (kit §4.1 long description). It is false on `poi_footnotes.docx` and on `sample_xlsx_formulas` (§4).
- **"Nested lists" or list structure.** Depth is lost in output (§4).
- **"Only AILANG Parse extracts track changes" / "0 of 5 other parsers"** (`index.html`).
  - Pandoc gets 3/3 in our own benchmark.
  - Claude for Word reads them.
  - Aspose models them fully.
  - Narrow the claim to: "with author and date, through an MCP connector, outside Word".
- **"Hand-verified ground truth", or any OfficeDocBench number against Docling or Unstructured**, until the ground truth is independent and Docling has been re-run on current `main`.
- **"Replay"** as if it re-runs the parse. It returns stored output, capped at 10 KB.
- **"Formally verified" / "N verified contracts"** as a correctness guarantee to end users. The counts drift (28 vs 63) and do not cover output correctness.
- **"Private" for the connector path.** The assistant already has the file.
- **"Parse the file I uploaded" in Claude**, until file handoff ships (`m-mcp-file-handoff.md`). ChatGPT can use `openai/fileParams`.
- **"~100× cheaper"** in a listing. That comparison is against per-page PDF APIs, not the assistant.
- **"Never stored"** without the qualifier. Files are not stored; up to 10 KB of output is kept by default.
- **"EU-only."** Sign-in and AI parses may run in the US.
- **Redaction**, until it exists.

### 6.4 Listing copy (replaces kit §4.1 and §4.4)

**Anthropic one-liner (≤ 200 characters; this one is 184):**
> Read Word, PowerPoint and Excel files exactly: tracked changes with author and date, comments, headers, footers and merged cells. No AI on Office files, and the same result every time.

**OpenAI shortDescription (≤ 30 characters; 29):** `Exact Office document reading`

**Long description (no pricing):**
> Most ways of reading a Word file give an AI assistant the final text only. Deleted passages vanish, insertions blend in, and comments, headers and footers are dropped. AILANG Parse reads the document's own structure instead.
>
> Ask it to parse a document and you get Markdown, HTML or structured JSON blocks with:
> • tracked insertions, deletions and moved text, each with author and date
> • comments, with who wrote them, when, and the exact text they refer to
> • headers and footers, tables with merged cells, slide speaker notes, and spreadsheet sheets
>
> Word, PowerPoint, Excel, OpenDocument, HTML, Markdown, CSV, EPUB, email (EML/MBOX), LaTeX and RTF are read by AILANG Parse's own engine, deterministically. The same file gives the same output every time, and no AI model sees the content. PDFs use text extraction by default. AI is used only for scans and images, and the formats tool tells you beforehand which requests need it.
>
> It also converts documents: to Markdown, HTML or Quarto, or to Word, PowerPoint, Excel or OpenDocument, with a download link.
>
> You sign in with Google or GitHub. AILANG Parse receives only the document and options the assistant sends to a tool, never your conversation. Uploaded files are not stored. Each request keeps a short record, and you can switch off stored output or delete your history. The service is hosted in the EU (Belgium) and operated by Holosun ApS, Denmark.
>
> For documents that must not leave your machine, the same engine runs locally as a command-line tool or in your browser: https://www.sunholo.com/ailang-parse/

Two conditions on this copy:
- Remove "with a download link" if G4 is not live on prod at submission.
- **Do not add "formulas" or "footnotes" until §6.5 items 3–4 land.**

**Example prompts.** Each uses a sample that today returns the claimed structure (§4):

| # | Prompt | Tool / sample |
|---|---|---|
| 1 | "Show every comment in the AILANG Parse sample Word document: who wrote it, when, and the exact text it's attached to." | `mcpParse` / `sample_docx_comments` |
| 2 | "List each tracked change in the AILANG Parse track-changes sample, with its author and date, and say what was moved." | `mcpParse` / `sample_docx_track_changes` |
| 3 | "Convert https://www.sunholo.com/ailang-parse/assets/sample.docx to Markdown, keeping headers, footers and comments." | `mcpConvert` |
| 4 | "Parse the merged-cells spreadsheet sample and give me the table with its merged ranges." | `mcpParse` / `sample_xlsx_merged` (verify before submitting; not run today) |
| 5 | "What can AILANG Parse read, and which formats would need AI?" | `mcpFormats` |

OpenAI `defaultPrompt`: use #2, #1 and #5, shortened.

Drop the kit's "Plan before spending" and account-management use cases from the headline use cases. They describe our billing, not the user's job.

### 6.5 Product gaps, ranked by how much each strengthens the position

1. **File handoff** (`m-mcp-file-handoff.md`; ChatGPT `openai/fileParams` first). Without it, pillar 1 cannot be demonstrated on the user's own file in claude.ai. This blocks the whole positioning, not just the listing.
2. **Fix the false or embarrassing samples before any reviewer sees them.**
   - `sample_docx_footnotes` has no footnotes.
   - `sample_xlsx_formulas` shows empty cells.
   - The footnote text in `poi_footnotes.docx` is dropped.
   - These are small, routed to the ailang-parse lane.
3. **Keep formulas.** Emit the formula text alongside the cached value in XLSX blocks and Markdown (`=SUM(B2:B3)` → `120 (=SUM(B2:B3))`). Both Claude's skill (two openpyxl passes) and Claude for Excel already do this, so today we are behind the assistant here.
4. **Anchor changes inline and keep list depth.**
   - Deletions and insertions positioned in their paragraph, the way comment ranges already are.
   - A `level` on list items.
   - These make the redline demo credible to a lawyer.
5. **`summarizeChanges` / `listRevisions` tool.** A deterministic redline report:
   - per author, per date, per section
   - counts of insertions, deletions and moves
   - every comment thread with its status
   This is the "tracked-changes summary" Mark proposed. It turns pillar 1 into a one-call answer, and assistants can only approximate it.
6. **`compareVersions(a, b)`.** A deterministic structural diff of two Office files, for the case where the other side sent a clean copy with no tracked changes. It pairs with `docparse-render --compare`, which today compares rasters only. Strong for legal; nothing comparable is listed in either directory.
7. **Redaction (`redactDocument`).** Assessed as feasible and highly defensible.
   - **Detection** stays with the assistant or a regex/pattern library. The tool takes explicit spans or patterns, so we make no AI claim.
   - **Application** is deterministic and covers every part: body runs, `w:delText` inside tracked deletions, comments and their replies, headers and footers, footnotes and endnotes, text boxes, alt text, document properties (author, last-modified-by), custom XML, and optionally accepting or stripping all revisions first.
   - **Proof** re-parses the output and asserts that no redacted string remains in any part. Encode this as an AILANG contract and return a residue report.
   - Output is a new file via `mcpConvert`-style download.
   - This is the only candidate pillar where "the assistant can do it too" fails: the assistant would hand-edit XML each time and could not prove the result clean.
   - Estimate: one sprint for DOCX text redaction plus residue proof; PPTX/XLSX later; images out of scope.
8. **Make OfficeDocBench independent.**
   - Hand-annotate the ground truth for at least the 15 challenge files, from the XML and not from our outputs.
   - Re-run Docling `main` and add Aspose as a ceiling.
   - Publish the per-feature heatmap rather than a composite.
   - Only then is a benchmark claim usable, and it would be the only public benchmark on revisions and comments (§1.3).
9. **Tracked-change output from `editDocument`.** This matches Claude's docx skill on writing, so that "edit" round-trips stay reviewable.

---

## 7. What Parse proves for the platform

Parse is the first AILANG-built service going into the Claude and ChatGPT directories, and a spike test for the shared stack:
- `serve-api`'s listed `/mcp/connect/` surface
- `sunholo/mcp_oauth`
- the planned `sunholo/mcp_files`

Of Parse's differentiators, these generalise to any AILANG MCP service:

| Differentiator | Generalises? | Evidence now | Platform claim it supports |
|---|---|---|---|
| **Determinism** | Yes, for any service whose logic is AILANG and not an LLM call | 24/24 byte-identical (§4); `aiCallsUsed` reported per call | "An AILANG tool returns the same answer for the same input, and says when AI was involved" |
| **Contracts and verification** | Yes, but only for pure predicates | `sunholo/mcp_oauth` targets Z3 `verified == total` for PKCE, redirect matching, CIMD validation and code TTL (`design_docs/planned/v0_52_0/m-mcp-oauth-package.md`). The redaction residue proof in §6.5 would be the first *user-visible* one | "Security-critical logic is proved, not just tested". Count and publish proved vs runtime-asserted, never a single "N contracts" figure |
| **Capability-gated effects** | Yes | Every entry point declares its effects (`! {IO, FS, Net, Env, Clock, Declassify}` on `requestReplay`). The CLI is run with explicit `--caps`, and AI is only reachable with the `AI` cap | "A tool cannot touch the network, files or an AI model unless its signature says so". This is a reviewable answer to the directories' data-handling questions |
| **IFC labels for secrets** | Yes | `M-IFC-DECLARED-RECORD-LABELS` (implemented, v0.52.0); `Declassify` as an effect; token digests in `mcp_oauth` | "Secrets cannot reach a log or a response without an explicit, greppable declassification" |
| **OAuth with no infrastructure** | Yes. Already reused as a package | `sunholo/mcp_oauth` 0.1.1 in `docparse/ailang.toml:40`; Claude sign-in verified on prod 2026-10-07; `ailang mcp check --target both` | "Any AILANG service gets directory-grade sign-in by adding a package and three hooks" |
| **Small server footprint** | Yes | Parse package about 400 KB (`README.md` install); one runtime dependency; Cloud Run | "One binary, one package, no sidecar auth server" |
| **Same engine local, browser and hosted** | Partly. Needs WASM-safe effects | Parse CLI, WASM and API share modules (`docparse/services/*.ail`) | "The connector and the local tool are the same code", which is the bridge behind the privacy pillar |

**How the listing can hint at this without distracting from Parse:**
- One sentence in the long description's last paragraph, and not more: *"Built in AILANG, a language for verifiable AI-written software."* It links to the AILANG site, which carries the platform story.
- On the website, a short "How it's built" section on `why.html`, not on the listing:
  - capabilities declared per tool
  - sign-in via the open-source `sunholo/mcp_oauth`
  - deterministic engine
  This turns the platform into a credibility signal for Parse rather than a second product pitch.
- Once redaction ships, its residue proof is the best platform demo. It is a guarantee a user can see ("we checked every part of the file; nothing remains") that comes directly from AILANG contracts. Lead with that when the platform story goes public.

---

## Appendix: reproduction (today's run)

- Inputs were copied from `ailang-parse/data/test_files/` into the session scratchpad.
- Commands:
  - `docparse <file> --output-dir out1`, then the same into `out2`, and `cmp` each pair.
  - `uv run --no-project --with python-docx|openpyxl` for the baselines.
  - `uvx --from 'markitdown[docx,xlsx]' markitdown <file>`.
- The naive baseline is about 15 lines: `zipfile` + `ElementTree` over `word/document.xml`, joining `w:t` per `w:p` and reading `w:pStyle`.
- Nothing was sent to any hosted service. The one attempted hosted call returned `AUTH_REQUIRED` and consumed no quota.
