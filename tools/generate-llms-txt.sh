#!/usr/bin/env bash
# Generate llms.txt (concise llmsstxt.org routing index, <35 KB) and
# docs/static/llms-full.txt (complete reference corpus + verbatim active prompt).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

DOCS_OUTPUT="$REPO_ROOT/docs/llms.txt"
STATIC_OUTPUT="$REPO_ROOT/docs/static/llms.txt"
ROOT_OUTPUT="$REPO_ROOT/llms.txt"
FULL_OUTPUT="$REPO_ROOT/docs/static/llms-full.txt"

NOW="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

ACTIVE_VERSION="unknown"
PROMPT_FILE=""
if [ -f "$REPO_ROOT/prompts/versions.json" ]; then
    ACTIVE_VERSION=$(jq -r '.active' "$REPO_ROOT/prompts/versions.json")
    PROMPT_FILE=$(jq -r ".versions.\"$ACTIVE_VERSION\".file" "$REPO_ROOT/prompts/versions.json")
fi

echo "=== Generating llms.txt + llms-full.txt (active prompt: $ACTIVE_VERSION) ==="

# ---------------------------------------------------------------------------
# 1. Concise /llms.txt (llmsstxt.org format, <35 KB)
# ---------------------------------------------------------------------------
cat > "$DOCS_OUTPUT" << EOF
# AILANG

> AILANG is a deterministic, purely functional, effect-typed programming language designed as a safe execution substrate for AI-generated code. Side effects are explicit in function signatures (\`! {IO, FS, Net}\`), contracts are Z3-verifiable (\`requires\`/\`ensures\`), execution is reproducible, and runs emit structured traces.

- **Full single-file reference + active teaching prompt (${ACTIVE_VERSION}):** https://ailang.sunholo.com/llms-full.txt
- **Hosted MCP server (version-pinned docs, stdlib, examples, benchmarks):** https://mcp.ailang.sunholo.com/mcp/
- **Local CLI canonical syntax:** \`ailang prompt\` (active: \`${ACTIVE_VERSION}\`)
- **Last updated:** ${NOW}

## Agent Task Routing Table

| Agent Task | Local CLI Command | Hosted MCP Tool (\`mcp.ailang.sunholo.com/mcp/\`) | Reference / Example |
|---|---|---|---|
| **1. Load current syntax before coding** | \`ailang prompt\` | \`prompt_get(part="core")\` + \`prompt_get(part="reference")\` | https://ailang.sunholo.com/docs/prompts/current |
| **2. X-ray an existing \`.ail\` file/dir (\`<100ms\`)** | \`ailang inspect --format=agent <path>\` | — (local CLI) | https://ailang.sunholo.com/docs/guides/debugging |
| **3. Type + effect check (compact agent output)** | \`ailang check --format=agent <file.ail>\` | \`ailang_check\` (executor MCP) | https://ailang.sunholo.com/docs/reference/language-syntax |
| **4. Auto-repair missing \`! {Effects}\`, \`++\`, \`TC_MATCH_002\`** | \`ailang check --fix <file.ail>\` | — (local CLI / \`ailang lsp\`) | https://ailang.sunholo.com/docs/guides/lsp |
| **5. Query expected type & bindings at expression** | Put \`_?\` or \`?goal\` in code -> \`ailang check --format=agent\` | \`ailang_check\` | https://ailang.sunholo.com/docs/guides/lsp |
| **6. Prove \`requires\`/\`ensures\` contracts via Z3** | \`ailang verify --json <file.ail>\` (add \`--int-model=int64\` for overflow) | — (local CLI) | https://ailang.sunholo.com/docs/guides/contracts |
| **7. Run inline \`tests [...]\` & property checks** | \`ailang test <file.ail>\` (or \`--package\`) | — (local CLI) | https://ailang.sunholo.com/docs/guides/testing |
| **8. Execute deterministically (interpreter or VM)** | \`ailang run --caps IO,FS --bytecode <file.ail>\` | \`ailang_run\` (sandboxed) | https://ailang.sunholo.com/docs/reference/effects |
| **9. Look up stdlib module exports & signatures** | \`ailang docs --list\` / \`ailang docs std/json\` | \`stdlib_modules\`, \`stdlib_module\` | https://ailang.sunholo.com/docs/reference/stdlib |
| **10. Find working runnable \`.ail\` examples** | \`ailang examples search "<query>"\` | \`example_for_concept\`, \`example_get\` | https://ailang.sunholo.com/docs/examples |
| **11. Inspect JSON module interface & lock packages** | \`ailang iface <mod>\` / \`ailang pkg docs <pkg>\` | \`docs_search\` | https://ailang.sunholo.com/docs/guides/packages |
| **12. Capture JSONL execution trace for repair** | \`ailang run --emit-trace jsonl --quiet --caps IO <file.ail>\` | — (local CLI) | https://ailang.sunholo.com/docs/guides/traces |

## Start Here

- [Introduction](https://ailang.sunholo.com/docs/intro): 60-second overview, visual architecture, and persona entry paths
- [Why AILANG](https://ailang.sunholo.com/docs/why-ailang): Comparison vs Python, Go, and TypeScript for AI-generated code
- [Why 1.0](https://ailang.sunholo.com/docs/why-1-0): 1.x stability contract and honest capability boundaries
- [Quick Start](https://ailang.sunholo.com/docs/start-here/quick-start): Install CLI, check, test, and run in 2 minutes
- [For AI Agents](https://ailang.sunholo.com/docs/start-here/for-ai-agents): Agent repair loop, CLI/MCP routing, and harness setup
- [Examples](https://ailang.sunholo.com/docs/examples): Verified runnable \`.ail\` programs
- [Playground](https://ailang.sunholo.com/docs/playground): Interactive browser WASM REPL

## Learn AILANG

- [Getting Started](https://ailang.sunholo.com/docs/guides/getting-started): Modern syntax (\`|>\`, destructuring, \`deriving (Json)\`, \`expr?\`, \`inspect\`, \`check --fix\`)
- [Quick-Start Examples](https://ailang.sunholo.com/docs/guides/quick-start-examples): Idiomatic snippets for common tasks
- [Upgrading to 1.0](https://ailang.sunholo.com/docs/guides/upgrading-to-1-0): Migration guide from pre-1.0 releases
- [Contracts & Z3 Verification](https://ailang.sunholo.com/docs/guides/contracts): \`requires\`/\`ensures\`, \`--int-model=int64\`, tuples, counterexamples
- [IFC Taint Labels](https://ailang.sunholo.com/docs/guides/ifc-labels): Static information-flow tracking (\`T<label>\`, \`T{not L}\`, \`Declassify\`)
- [Agent Tool Policy](https://ailang.sunholo.com/docs/guides/agent-tool-policy): \`--policy\` confinement and capability ceilings
- [AI Effect (\`std/ai\`)](https://ailang.sunholo.com/docs/guides/ai-effect): Typed \`callJsonResult\`, \`step\`, and tool loops
- [LSP & Quick-Fix](https://ailang.sunholo.com/docs/guides/lsp): \`ailang lsp --stdio\`, typed holes (\`_?\`, \`?goal\`), and \`check --fix\`
- [Testing](https://ailang.sunholo.com/docs/guides/testing): Inline \`tests [...]\` and property-based checks
- [Packages](https://ailang.sunholo.com/docs/guides/packages): \`ailang.toml\`, atomic \`ailang.lock\`, and \`ailang pkg docs\`

## For AI Agents

- [Current Teaching Prompt (${ACTIVE_VERSION})](https://ailang.sunholo.com/docs/prompts/current): Verbatim canonical prompt used in evals
- [Agent Integration](https://ailang.sunholo.com/docs/guides/agent-integration): Pre-flight checklist for coding agents
- [Hosted Docs MCP](https://ailang.sunholo.com/docs/guides/agent-mcp): Connect \`https://mcp.ailang.sunholo.com/mcp/\`
- [Claude Code Integration](https://ailang.sunholo.com/docs/guides/claude-code-integration): Plugin skills, hooks, and local MCP
- [Three Camps Comparison](https://ailang.sunholo.com/docs/guides/three-camps-comparison): Deterministic substrate vs agent frameworks

## Normative Reference

- [Language Syntax](https://ailang.sunholo.com/docs/reference/language-syntax): Grammar, types, ADTs, records, pipes, destructuring
- [Standard Library](https://ailang.sunholo.com/docs/reference/stdlib): Built-in modules and signatures (\`std/io\`, \`std/json\`, \`std/fs\`, \`std/net\`, \`std/ai\`, etc.)
- [Effects Reference](https://ailang.sunholo.com/docs/reference/effects): Capability grants (\`--caps\`), effect rows, and budgets (\`@limit=N\`)
- [Modules](https://ailang.sunholo.com/docs/reference/modules): Module resolution, imports, exports, and interfaces
- [Option vs Result](https://ailang.sunholo.com/docs/reference/option-vs-result): Choosing \`Option[T]\` vs \`Result[T, E]\` and \`let x = expr?\`
- [CLI Reference](https://ailang.sunholo.com/docs/reference/cli): All \`ailang\` subcommands and flags
- [REPL Commands](https://ailang.sunholo.com/docs/reference/repl-commands): Interactive \`λ[Debug,IO]>\` session commands
- [Diagnostic Errors](https://ailang.sunholo.com/docs/reference/errors): Structured compiler and runtime error codes
- [Reserved Keywords](https://ailang.sunholo.com/docs/reference/reserved-keywords): Lexer keywords and contextual identifiers
- [Security Model](https://ailang.sunholo.com/docs/reference/security-model): Capability boundaries, IFC, and sandbox guarantees
- [Stability Promise (1.x)](https://ailang.sunholo.com/docs/reference/stability): Stable, experimental, and removed surfaces
- [Known Limitations](https://ailang.sunholo.com/docs/reference/limitations): Live-verified capability and cliff matrix

## Operate & Deploy

- [Debugging](https://ailang.sunholo.com/docs/guides/debugging): \`ailang inspect\`, \`check --fix\`, and trace workflows
- [Execution Traces](https://ailang.sunholo.com/docs/guides/traces): \`--emit-trace jsonl\` structured replay traces
- [Go Interop](https://ailang.sunholo.com/docs/guides/go-interop): Embedding AILANG in Go via \`internal/embed\`
- [WASM Integration](https://ailang.sunholo.com/docs/guides/wasm-integration): Running AILANG in the browser
- [Benchmarks](https://ailang.sunholo.com/docs/benchmarks/performance): Live model leaderboard and ELO ratings

## Optional

- [Full Single-File Corpus (llms-full.txt)](https://ailang.sunholo.com/llms-full.txt): Complete README, guides, normative reference, and verbatim active teaching prompt (${ACTIVE_VERSION})
EOF

mkdir -p "$(dirname "$STATIC_OUTPUT")"
cp "$DOCS_OUTPUT" "$ROOT_OUTPUT"
cp "$DOCS_OUTPUT" "$STATIC_OUTPUT"

# ---------------------------------------------------------------------------
# 2. Full /llms-full.txt (docs/static/llms-full.txt)
# ---------------------------------------------------------------------------
cat > "$FULL_OUTPUT" << EOF
# AILANG Full Documentation & Active Teaching Prompt

> Complete single-file LLM corpus for AILANG: README, agent routing hub, getting-started guide, normative language/stdlib/effects/modules/security/stability/CLI/limitations reference, and verbatim active teaching prompt (${ACTIVE_VERSION}).

- Concise routing index: https://ailang.sunholo.com/llms.txt
- Active prompt: ${ACTIVE_VERSION}
- Last updated: ${NOW}

---

EOF

append_section() {
    local title="$1"
    local relpath="$2"
    local filepath="$REPO_ROOT/$relpath"
    if [ -f "$filepath" ]; then
        {
            echo ""
            echo "# ${title}"
            echo ""
            cat "$filepath"
            echo ""
            echo "---"
            echo ""
        } >> "$FULL_OUTPUT"
    fi
}

append_section "README" "README.md"
append_section "Start Here: For AI Agents" "docs/docs/start-here/for-ai-agents.mdx"
append_section "Guide: Getting Started" "docs/docs/guides/getting-started.mdx"
append_section "Reference: Language Syntax" "docs/docs/reference/language-syntax.md"
append_section "Reference: Standard Library" "docs/docs/reference/stdlib.md"
append_section "Reference: Effects" "docs/docs/reference/effects.md"
append_section "Reference: Modules" "docs/docs/reference/modules.md"
append_section "Reference: Security Model" "docs/docs/reference/security-model.md"
append_section "Reference: Stability" "docs/docs/reference/stability.md"
append_section "Reference: CLI" "docs/docs/reference/cli.md"
append_section "Reference: Reserved Keywords" "docs/docs/reference/reserved-keywords.md"
append_section "Reference: Known Limitations" "docs/docs/reference/limitations.md"

if [ -n "$PROMPT_FILE" ] && [ -f "$REPO_ROOT/$PROMPT_FILE" ]; then
    {
        echo ""
        echo "# AI Teaching Prompt (Latest: $ACTIVE_VERSION)"
        echo ""
        cat "$REPO_ROOT/$PROMPT_FILE"
        echo ""
        echo "---"
        echo ""
    } >> "$FULL_OUTPUT"
fi

INDEX_SIZE=$(wc -c < "$DOCS_OUTPUT" | tr -d ' ')
INDEX_LINES=$(wc -l < "$DOCS_OUTPUT" | tr -d ' ')
FULL_SIZE=$(wc -c < "$FULL_OUTPUT" | tr -d ' ')
FULL_LINES=$(wc -l < "$FULL_OUTPUT" | tr -d ' ')

echo "✓ llms.txt + llms-full.txt generated successfully"
echo "  Index:  $INDEX_SIZE bytes, $INDEX_LINES lines -> llms.txt, docs/llms.txt, docs/static/llms.txt"
echo "  Full:   $FULL_SIZE bytes, $FULL_LINES lines -> docs/static/llms-full.txt"
echo "  URLs:   https://ailang.sunholo.com/llms.txt | https://ailang.sunholo.com/llms-full.txt"
