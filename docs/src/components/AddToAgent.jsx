// AddToAgent — landing-page CTA that adds the AILANG **hosted docs MCP**
// (mcp.ailang.sunholo.com — read-only knowledge: docs, stdlib, examples,
// benchmarks, submit_feedback) to the visitor's AI coding harness in one click.
// This is NOT the local execution MCP that ships in ailang_bootstrap; that one
// wraps the local `ailang` binary and is installed via the plugin, not here.
// Three buttons:
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

// AddToAgentButtons — chrome-less variant for embedding inside docs pages
// (e.g. getting-started.mdx) where the surrounding heading and manual-config
// snippet are already present. Renders just the three deeplink buttons,
// reusing deeplink() and HARNESSES from above.
export function AddToAgentButtons() {
  return (
    <div className="add-to-agent-buttons add-to-agent-buttons--inline">
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
      <style>{`
        .add-to-agent-buttons--inline {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 0.75rem;
          margin: 1rem 0 1.25rem 0;
        }
        @media (max-width: 720px) {
          .add-to-agent-buttons--inline { grid-template-columns: 1fr; }
        }
        .add-to-agent-buttons--inline .add-to-agent-btn {
          display: inline-flex;
          align-items: center;
          gap: 0.6rem;
          padding: 0.7rem 0.9rem;
          border: 1px solid var(--ifm-color-emphasis-300);
          border-radius: 8px;
          background: var(--ifm-card-background-color, var(--ifm-background-surface-color));
          color: var(--ifm-color-content);
          text-decoration: none;
          transition: border-color 0.15s ease, transform 0.15s ease;
        }
        .add-to-agent-buttons--inline .add-to-agent-btn:hover {
          border-color: var(--ifm-color-primary);
          transform: translateY(-1px);
          text-decoration: none;
        }
        .add-to-agent-buttons--inline .add-to-agent-btn span {
          display: flex;
          flex-direction: column;
          line-height: 1.2;
        }
        .add-to-agent-buttons--inline .add-to-agent-btn small {
          color: var(--ifm-color-content-secondary);
          font-family: 'JetBrains Mono', monospace;
          font-size: 0.75rem;
        }
      `}</style>
    </div>
  );
}
