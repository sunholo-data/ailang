import React, { useState, useEffect } from 'react';

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const HARNESS_LABEL = { claude: 'Claude CLI', gemini: 'Gemini CLI', opencode: 'opencode', codex: 'Codex' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];

function pct(v) {
  return v == null ? null : Math.round(v * 100);
}

function heatBg(rate) {
  if (rate == null) return 'transparent';
  if (rate >= 0.85) return 'rgba(34,197,94,0.25)';
  if (rate >= 0.70) return 'rgba(34,197,94,0.12)';
  if (rate >= 0.50) return 'rgba(234,179,8,0.18)';
  if (rate >= 0.30) return 'rgba(249,115,22,0.18)';
  return 'rgba(239,68,68,0.20)';
}

function Cell({ rate }) {
  const bg = heatBg(rate);
  const p = pct(rate);
  const color = rate == null ? 'var(--ifm-color-emphasis-400)' : rate >= 0.85 ? '#15803d' : rate < 0.30 ? '#b91c1c' : 'inherit';
  return (
    <td style={{ textAlign: 'center', padding: '8px 12px', background: bg, color, fontWeight: rate >= 0.85 ? 700 : 400 }}>
      {p == null ? '—' : `${p}%`}
    </td>
  );
}

function Chip({ label, active, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: '3px 10px', borderRadius: 14, cursor: 'pointer', fontSize: '0.8rem',
        fontWeight: active ? 600 : 400,
        border: active ? '2px solid var(--ifm-color-primary)' : '1px solid var(--ifm-color-emphasis-300)',
        background: active ? 'var(--ifm-color-primary)' : 'transparent',
        color: active ? '#fff' : 'var(--ifm-color-emphasis-700)',
      }}
    >
      {label}
    </button>
  );
}

function FilterBar({ harnesses, activeHarness, onHarness, langs, activeLang, onLang }) {
  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, padding: '10px 14px', background: 'var(--ifm-color-emphasis-100)', borderRadius: 8, marginBottom: 16, fontSize: '0.85rem', alignItems: 'center' }}>
      <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
        <strong style={{ color: 'var(--ifm-color-emphasis-700)' }}>Harness:</strong>
        <Chip label="All" active={!activeHarness} onClick={() => onHarness(null)} />
        {harnesses.map(h => (
          <Chip key={h} label={HARNESS_LABEL[h] || h} active={activeHarness === h} onClick={() => onHarness(activeHarness === h ? null : h)} />
        ))}
      </div>
      <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
        <strong style={{ color: 'var(--ifm-color-emphasis-700)' }}>Language:</strong>
        <Chip label="All" active={!activeLang} onClick={() => onLang(null)} />
        {langs.map(l => (
          <Chip key={l} label={LANG_LABEL[l] || l} active={activeLang === l} onClick={() => onLang(activeLang === l ? null : l)} />
        ))}
      </div>
    </div>
  );
}

const th = { padding: '8px 12px', background: 'var(--ifm-color-emphasis-100)', fontWeight: 600, textAlign: 'center', fontSize: '0.75rem', whiteSpace: 'nowrap', borderBottom: '2px solid var(--ifm-color-emphasis-200)' };
const rowHeader = { padding: '8px 14px', fontWeight: 700, textAlign: 'left', whiteSpace: 'nowrap', borderRight: '2px solid var(--ifm-color-emphasis-200)' };
const subHeader = { ...rowHeader, fontWeight: 400, color: 'var(--ifm-color-emphasis-600)', fontSize: '0.8rem' };
const rowStyle = { borderBottom: '1px solid var(--ifm-color-emphasis-200)' };

function modelShort(key) {
  return key.replace('claude-', 'Claude ').replace('gemini-', 'Gemini ').replace('opencode-', 'OC/').replace('gpt5', 'GPT-5').replace(/-/g, ' ');
}

/**
 * BenchmarkExplorer: unified filterable heatmap.
 *
 * Two linked tables:
 *   1. Language × Model — overall pass rate per model per language
 *   2. Harness summary — pass rate per harness per language
 *
 * Filters: harness (scope model columns) + language (scope lang rows in table 2).
 */
export default function BenchmarkExplorer() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [activeHarness, setActiveHarness] = useState(null);
  const [activeLang, setActiveLang] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then(r => r.json())
      .then(setData)
      .catch(e => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load: {error}</p>;
  if (!data) return <p>Loading benchmark data…</p>;

  // --- derive lists ---
  const allLangs = Object.keys(data.languages || {}).sort((a, b) => {
    const ia = LANG_ORDER.indexOf(a), ib = LANG_ORDER.indexOf(b);
    if (ia < 0 && ib < 0) return a.localeCompare(b);
    if (ia < 0) return 1; if (ib < 0) return -1;
    return ia - ib;
  });
  const langs = activeLang ? [activeLang] : allLangs;

  const allHarnesses = Object.keys(data.harnesses || {}).sort();
  const allModels = Object.keys(data.models || {}).sort();
  // Filter models by active harness using agent_cli field
  const models = activeHarness
    ? allModels.filter(m => data.models[m]?.agent_cli === activeHarness)
    : allModels;

  // --- table 1: language × model ---
  return (
    <div>
      <FilterBar
        harnesses={allHarnesses}
        activeHarness={activeHarness}
        onHarness={setActiveHarness}
        langs={allLangs}
        activeLang={activeLang}
        onLang={setActiveLang}
      />

      <h3 style={{ marginTop: 0, marginBottom: 8, fontSize: '1rem' }}>Pass Rate by Language × Model</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        Green ≥ 85% · Yellow 50–84% · Red &lt; 30% · — = no results
      </p>

      <div style={{ overflowX: 'auto', marginBottom: 32 }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Language</th>
              {models.map(m => <th key={m} style={th} title={m}>{modelShort(m)}</th>)}
            </tr>
          </thead>
          <tbody>
            {langs.map(lang => (
              <tr key={lang} style={rowStyle}>
                <td style={rowHeader}>{LANG_LABEL[lang] || lang}</td>
                {models.map(m => (
                  <Cell key={m} rate={data.models[m]?.languages?.[lang]?.successRate ?? null} />
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <h3 style={{ marginTop: 0, marginBottom: 8, fontSize: '1rem' }}>Pass Rate by Harness × Language</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        Aggregated across all models using each harness.
      </p>

      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Harness</th>
              {langs.map(l => <th key={l} style={th}>{LANG_LABEL[l] || l}</th>)}
              <th style={th}>Avg Cost/run</th>
              <th style={th}>Models</th>
            </tr>
          </thead>
          <tbody>
            {(activeHarness ? [activeHarness] : allHarnesses).map(h => {
              const hr = data.harnesses[h];
              if (!hr) return null;
              return (
                <tr key={h} style={rowStyle}>
                  <td style={rowHeader}>{HARNESS_LABEL[h] || h}</td>
                  {langs.map(l => (
                    <Cell key={l} rate={hr.languages?.[l]?.successRate ?? null} />
                  ))}
                  <td style={{ textAlign: 'center', padding: '8px 12px', fontSize: '0.8rem' }}>
                    {hr.avg_cost_usd != null ? `$${hr.avg_cost_usd.toFixed(4)}` : '—'}
                  </td>
                  <td style={{ textAlign: 'center', padding: '8px 12px', fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)' }}>
                    {(hr.models || []).length}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
