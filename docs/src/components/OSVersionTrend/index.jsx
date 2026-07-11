import React, { useState } from 'react';
import historyData from '../../../static/benchmarks/os/history.json';

// M-EVAL-OS-LONGITUDINAL: local-rig eval performance per AILANG release.
// The version-trend snapshots (docs/static/benchmarks/os/history.json — written by
// tools/os-release-snapshot.sh and refreshed each rotation cycle) are BUNDLED AT
// BUILD TIME (imported, NOT fetched at runtime). A runtime fetch of that static
// JSON was subject to CDN edge-cache inconsistency: the same URL returned good data
// to one request and a stale/collapsed copy to another, which rendered as an empty
// table for weeks. The content-hashed bundle is cache-proof, always matches the
// deployed commit, and refreshes on every deploy (the rotation commits + pushes it).
// Renders a version-over-version trend: rows = (model, harness), columns = AILANG
// versions, cells = pass rate for the selected language.

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];
// `gemini` rows are historical (Gemini CLI executor retired v0.22.0); do not relabel them.
const HARNESS_LABEL = { claude: 'Claude CLI', gemini: 'Gemini CLI (retired)', managed_agents: 'Managed Agents', opencode: 'opencode', codex: 'Codex', pi: 'Pi', motoko: 'motoko_agent' };

function pct(v) {
  return v == null ? '—' : `${Math.round(v * 100)}%`;
}

function heat(r) {
  if (r == null) return 'transparent';
  if (r >= 0.85) return 'rgba(34,197,94,0.22)';
  if (r >= 0.70) return 'rgba(34,197,94,0.10)';
  if (r >= 0.50) return 'rgba(234,179,8,0.16)';
  if (r >= 0.30) return 'rgba(249,115,22,0.16)';
  return 'rgba(239,68,68,0.18)';
}

function shortModel(m) {
  return (m || '').replace(/^motoko-local-/, '').replace(/^(opencode|pi|motoko)-/, '');
}

export default function OSVersionTrend() {
  const [lang, setLang] = useState('ailang');

  // BUILD-TIME data (see file header): imported, not fetched — no CDN edge-cache
  // roulette, so the render is deterministic and matches the deployed commit.
  const history = Array.isArray(historyData) ? historyData : [];
  const valid = history.filter((e) => e && e.ailang_version);
  if (valid.length < 1) return null; // nothing published yet — show nothing

  // Oldest → newest, left to right.
  const ordered = [...valid].sort((a, b) =>
    (a.ailang_version || '').localeCompare(b.ailang_version || '', undefined, { numeric: true }),
  );

  const langs = LANG_ORDER.filter((l) => ordered.some((e) => (e.languages || []).includes(l)));
  const activeLang = langs.includes(lang) ? lang : (langs[0] || 'ailang');

  // Collect the union of (model, harness) series across all versions.
  const keyOf = (r) => `${r.model}||${r.harness}`;
  const seriesMap = new Map();
  ordered.forEach((entry) =>
    (entry.rows || []).forEach((r) => {
      if (!r || !r.model || !r.harness) return; // skip malformed rows defensively
      if (!seriesMap.has(keyOf(r))) seriesMap.set(keyOf(r), { model: r.model, harness: r.harness });
    }),
  );
  const series = [...seriesMap.values()].sort(
    (a, b) => a.harness.localeCompare(b.harness) || a.model.localeCompare(b.model),
  );

  // Self-diagnostic: an empty table must never be silent. If we loaded versions but
  // produced no series, show EXACTLY what the browser received — versions, total
  // rows, and whether rows were dropped for missing model/harness — so a blank
  // render becomes a legible report instead of a recurring mystery.
  const totalRows = ordered.reduce((n, e) => n + (e.rows || []).length, 0);
  const newestGen = ordered.map((e) => e.generated).filter(Boolean).sort().slice(-1)[0];
  if (series.length === 0) {
    return (
      <div
        style={{
          padding: '12px 14px',
          border: '1px solid var(--ifm-color-warning-dark, #b45309)',
          borderRadius: 8,
          background: 'rgba(234,179,8,0.10)',
          fontSize: '0.9em',
          lineHeight: 1.5,
        }}
      >
        <strong>No local-model rows to display.</strong>
        <br />
        Loaded <b>{ordered.length}</b> release{ordered.length === 1 ? '' : 's'} (
        {ordered.map((e) => e.ailang_version).join(', ') || '—'}), <b>{totalRows}</b> total row
        {totalRows === 1 ? '' : 's'} across them
        {totalRows > 0 ? ' — but none carried both a model and a harness field (shape mismatch).' : '.'}
        {totalRows === 0 && (
          <>
            {' '}
            The browser fetched version metadata but <b>no model rows</b> — a stale/edge-cached copy of{' '}
            <code>/benchmarks/os/history.json</code>. Hard-reload (Cmd-Shift-R); if it persists the CDN
            cache needs purging.
          </>
        )}
      </div>
    );
  }

  const rateAt = (entry, model, harness) => {
    const row = (entry.rows || []).find((r) => r.model === model && r.harness === harness);
    return row && row.lang ? row.lang[activeLang] : null;
  };

  return (
    <div style={{ overflowX: 'auto' }}>
      <div style={{ margin: '8px 0' }}>
        {langs.map((l) => (
          <button
            key={l}
            onClick={() => setLang(l)}
            style={{
              marginRight: 6,
              padding: '3px 10px',
              borderRadius: 6,
              border: '1px solid var(--ifm-color-emphasis-300)',
              background: activeLang === l ? 'var(--ifm-color-primary)' : 'transparent',
              color: activeLang === l ? '#fff' : 'inherit',
              cursor: 'pointer',
            }}
          >
            {LANG_LABEL[l]}
          </button>
        ))}
      </div>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
        <thead>
          <tr>
            <th style={{ textAlign: 'left', padding: '8px 12px' }}>Model</th>
            <th style={{ textAlign: 'left', padding: '8px 12px' }}>Harness</th>
            {ordered.map((e) => (
              <th key={e.ailang_version} style={{ textAlign: 'center', padding: '8px 12px' }}>
                {e.ailang_version}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {series.map((s) => (
            <tr key={keyOf(s)}>
              <td style={{ padding: '6px 12px', fontFamily: 'monospace', fontSize: '0.85em' }}>{shortModel(s.model)}</td>
              <td style={{ padding: '6px 12px' }}>{HARNESS_LABEL[s.harness] || s.harness}</td>
              {ordered.map((e) => {
                const r = rateAt(e, s.model, s.harness);
                return (
                  <td
                    key={e.ailang_version}
                    style={{ textAlign: 'center', padding: '6px 12px', background: heat(r), fontWeight: r >= 0.85 ? 700 : 400 }}
                  >
                    {pct(r)}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
      <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)', marginTop: 8 }}>
        Local-rig {LANG_LABEL[activeLang]} pass rate per AILANG release (N-trial rotation, $0). Columns are
        AILANG versions, newest on the right — the version-over-version evolution. Retired models freeze at
        their last version.
        <br />
        <span style={{ opacity: 0.75 }}>
          {ordered.length} release{ordered.length === 1 ? '' : 's'} · {series.length} local model
          {series.length === 1 ? '' : 's'}
          {newestGen ? ` · data generated ${String(newestGen).slice(0, 10)}` : ''}
        </span>
      </p>
    </div>
  );
}
