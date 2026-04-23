import React, { useState, useEffect } from 'react';

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];

function pct(v) { return v == null ? null : Math.round(v * 100); }

function heatBg(rate) {
  if (rate == null) return 'transparent';
  if (rate >= 0.85) return 'rgba(34,197,94,0.25)';
  if (rate >= 0.70) return 'rgba(34,197,94,0.12)';
  if (rate >= 0.50) return 'rgba(234,179,8,0.18)';
  if (rate >= 0.30) return 'rgba(249,115,22,0.18)';
  return 'rgba(239,68,68,0.20)';
}

function Cell({ rate }) {
  const p = pct(rate);
  return (
    <td style={{
      textAlign: 'center', padding: '8px 12px', background: heatBg(rate),
      fontWeight: rate >= 0.85 ? 700 : 400,
      color: rate == null ? 'var(--ifm-color-emphasis-400)' : rate >= 0.85 ? '#15803d' : rate < 0.30 ? '#b91c1c' : 'inherit',
    }}>
      {p == null ? '—' : `${p}%`}
    </td>
  );
}

function modelShort(key) {
  return key.replace('claude-', 'Claude ').replace('gemini-', 'Gemini ').replace('opencode-', 'OC/').replace('gpt5', 'GPT-5').replace(/-/g, ' ');
}

/**
 * Language × model pass-rate heatmap.
 * Rows = languages from data.languages (dynamic — new languages appear automatically).
 * Columns = models from data.models.
 * Rate source: data.models[model].languages[lang].successRate (overall).
 */
export default function LanguageLeaderboard() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then(r => r.json())
      .then(setData)
      .catch(e => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load: {error}</p>;
  if (!data) return <p>Loading benchmark data…</p>;

  const langs = Object.keys(data.languages || {}).sort((a, b) => {
    const ia = LANG_ORDER.indexOf(a), ib = LANG_ORDER.indexOf(b);
    if (ia < 0 && ib < 0) return a.localeCompare(b);
    if (ia < 0) return 1; if (ib < 0) return -1;
    return ia - ib;
  });

  const models = Object.keys(data.models || {}).sort();

  const th = { padding: '8px 12px', background: 'var(--ifm-color-emphasis-100)', fontWeight: 600, textAlign: 'center', fontSize: '0.75rem', whiteSpace: 'nowrap', borderBottom: '2px solid var(--ifm-color-emphasis-200)' };
  const rowHeader = { padding: '8px 14px', fontWeight: 700, textAlign: 'left', whiteSpace: 'nowrap', borderRight: '2px solid var(--ifm-color-emphasis-200)' };

  return (
    <div>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        Pass rate (%) by language × model. Green ≥ 85% · Yellow 50–74% · Red &lt; 30% · — = no results yet.
      </p>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Language</th>
              {models.map(m => <th key={m} style={th} title={m}>{modelShort(m)}</th>)}
            </tr>
          </thead>
          <tbody>
            {langs.map(lang => (
              <tr key={lang} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
                <td style={rowHeader}>{LANG_LABEL[lang] || lang}</td>
                {models.map(m => (
                  <Cell key={m} rate={data.models[m]?.languages?.[lang]?.successRate ?? null} />
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
