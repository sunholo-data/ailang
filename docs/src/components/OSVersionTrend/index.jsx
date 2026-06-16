import React, { useState, useEffect } from 'react';

// M-EVAL-OS-LONGITUDINAL: local-rig eval performance per AILANG release.
// Reads docs/static/benchmarks/os/history.json (an array of per-version
// snapshots written by tools/os-release-snapshot.sh) and renders a
// version-over-version trend: rows = (model, harness), columns = AILANG
// versions, cells = pass rate for the selected language. Degrades gracefully:
// no history → renders nothing; one version → a single column (still valid).

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];
const HARNESS_LABEL = { claude: 'Claude CLI', gemini: 'Gemini CLI', opencode: 'opencode', codex: 'Codex', pi: 'Pi', motoko: 'motoko_agent' };

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
  return m.replace(/^motoko-local-/, '').replace(/^(opencode|pi|motoko)-/, '');
}

export default function OSVersionTrend() {
  const [history, setHistory] = useState(null);
  const [lang, setLang] = useState('ailang');

  useEffect(() => {
    fetch('/benchmarks/os/history.json')
      .then((r) => (r.ok ? r.json() : []))
      .then((h) => setHistory(Array.isArray(h) ? h : []))
      .catch(() => setHistory([]));
  }, []);

  if (history == null) return <p>Loading version trend…</p>;
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
      if (!seriesMap.has(keyOf(r))) seriesMap.set(keyOf(r), { model: r.model, harness: r.harness });
    }),
  );
  const series = [...seriesMap.values()].sort(
    (a, b) => a.harness.localeCompare(b.harness) || a.model.localeCompare(b.model),
  );

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
      </p>
    </div>
  );
}
