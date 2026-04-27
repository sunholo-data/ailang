// AddToAgent — landing-page CTA that lets a visitor add the AILANG MCP server
// to their AI coding harness in one click. Three buttons:
//   - Claude Desktop (claude:// deeplink)
//   - Cursor (cursor:// deeplink)
//   - Cline (vscode:extension/saoudrizwan.claude-dev?addMcp=...)
//
// MCP_URL is the live remote endpoint. Each harness has its own URL scheme
// for "add this MCP server" — we pre-fill name + URL so the user just confirms.
//
// Each button also shows the manual fallback (the JSON snippet they can paste
// into ~/.claude/claude_desktop_config.json or settings.json) for the cases
// where the deeplink isn't supported.

import React, { useState } from 'react';
import { Bot, Copy, Check, ExternalLink } from 'lucide-react';

const MCP_URL = 'https://mcp.ailang.sunholo.com/mcp/';
const MCP_NAME = 'ailang-docs';

const HARNESS_CONFIG = JSON.stringify(
  {
    mcpServers: {
      [MCP_NAME]: { url: MCP_URL, transport: 'streamable-http' },
    },
  },
  null,
  2,
);

function deeplink(harness) {
  const enc = encodeURIComponent;
  switch (harness) {
    case 'claude':
      return `claude://mcp/install?name=${enc(MCP_NAME)}&url=${enc(MCP_URL)}&transport=streamable-http`;
    case 'cursor':
      return `cursor://anysphere.cursor-deeplink/mcp/install?name=${enc(MCP_NAME)}&config=${enc(JSON.stringify({ url: MCP_URL, transport: 'streamable-http' }))}`;
    case 'cline':
      return `vscode:extension/saoudrizwan.claude-dev?addMcp=${enc(JSON.stringify({ name: MCP_NAME, url: MCP_URL, transport: 'streamable-http' }))}`;
    default:
      return '#';
  }
}

const HARNESSES = [
  { id: 'claude', name: 'Claude Desktop', sub: 'claude://' },
  { id: 'cursor', name: 'Cursor', sub: 'cursor://' },
  { id: 'cline', name: 'Cline (VS Code)', sub: 'vscode://' },
];

export default function AddToAgent() {
  const [copied, setCopied] = useState(false);

  const copyConfig = async () => {
    try {
      await navigator.clipboard.writeText(HARNESS_CONFIG);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch (e) {
      // Clipboard may be unavailable (insecure context, browser block); the
      // <code> block is still selectable manually.
    }
  };

  return (
    <section className="add-to-agent">
      <div className="add-to-agent-inner">
        <div className="add-to-agent-header">
          <Bot size={24} />
          <h2>Give your AI coding agent live AILANG knowledge</h2>
        </div>
        <p className="add-to-agent-lede">
          One click adds <code>{MCP_URL}</code> as a remote MCP server. Your
          agent then has typed tools for stdlib, examples, design docs, and
          benchmarks — no more scraping the docs site.
        </p>

        <div className="add-to-agent-buttons">
          {HARNESSES.map((h) => (
            <a
              key={h.id}
              href={deeplink(h.id)}
              className="add-to-agent-btn"
              rel="noopener noreferrer"
            >
              <ExternalLink size={16} />
              <span>
                <strong>Add to {h.name}</strong>
                <small>{h.sub}</small>
              </span>
            </a>
          ))}
        </div>

        <details className="add-to-agent-fallback">
          <summary>Or copy the config manually</summary>
          <p>
            For harnesses without a one-click deeplink, paste this into your
            MCP config file (e.g.{' '}
            <code>~/.claude/claude_desktop_config.json</code>):
          </p>
          <div className="add-to-agent-code">
            <pre>
              <code>{HARNESS_CONFIG}</code>
            </pre>
            <button
              onClick={copyConfig}
              className="add-to-agent-copy"
              aria-label="Copy MCP config"
            >
              {copied ? <Check size={14} /> : <Copy size={14} />}
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <p>
            See the{' '}
            <a href="/docs/guides/agent-mcp">agent-MCP guide</a> for the full
            tool catalog and examples.
          </p>
        </details>
      </div>
    </section>
  );
}
