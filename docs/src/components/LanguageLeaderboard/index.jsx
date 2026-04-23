import React, { useState, useEffect } from 'react';
import DimensionSelector from '@site/src/components/DimensionSelector';

const LANG_DISPLAY = {
  ailang: 'AILANG',
  python: 'Python',
  javascript: 'JavaScript',
  go: 'Go',
};

function pct(v) {
  if (v == null) return null;
  return Math.round(v * 100);
}

function heatColor(rate) {
  if (rate == null) return 'transparent';
  if (rate >= 0.85) return 'rgba(34,197,94,0.25)';
  if (rate >= 0.7) return 'rgba(34,197,94,0.12)';
  if (rate >= 0.5) return 'rgba(234,179,8,0.18)';
  if (rate >= 0.3) return 'rgba(249,115,22,0.18)';
  return 'rgba(239,68,68,0.18)';
}

function Cell({ rate }) {
  if (rate == null) {
    return (
      <td style={{ textAlign: 'center', color: 'var(--ifm-color-emphasis-400)', padding: '8px 12px' }}>
        —
      </td>
    );
  }
  return (
    <td
      style={{
        textAlign: 'center',
        padding: '8px 12px',
        background: heatColor(rate),
        fontWeight: rate >= 0.85 ? 700 : 400,
        color: rate >= 0.85 ? '#15803d' : rate < 0.3 ? '#b91c1c' : 'inherit',
      }}
    >
      {pct(rate)}%
    </td>
  );
}

/**
 * Heatmap table: rows = languages, columns = models.
 * Reads `languages` and `models` from latest.json.
 * New languages appear automatically — no hardcoding.
 */
export default function LanguageLeaderboard() {
  const [data, setData] = useState(null);
  const [selectedTier, setSelectedTier] = useState('core');
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then((r) => r.json())
      .then(setData)
      .catch((e) => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load benchmark data: {error}</p>;
  if (!data) return <p>Loading benchmark data…</p>;

  // Collect all languages from the languages map (dynamic — includes JS/Go when results exist)
  const allLangs = Object.keys(data.languages || {}).sort((a, b) => {
    const order = ['ailang', 'python', 'javascript', 'go'];
    const ia = order.indexOf(a);
    const ib = order.indexOf(b);
    if (ia === -1 && ib === -1) return a.localeCompare(b);
    if (ia === -1) return 1;
    if (ib === -1) return -1;
    return ia - ib;
  });

  // Collect all languages that appear in any model's languages breakdown
  const modelLangSet = new Set();
  for (const stats of Object.values(data.models || {})) {
    for (const l of Object.keys(stats.languages || {})) modelLangSet.add(l);
  }
  // Union with top-level languages (covers languages that have results but no model breakdown yet)
  for (const l of allLangs) modelLangSet.add(l);
  const langs = [...modelLangSet].sort((a, b) => {
    const order = ['ailang', 'python', 'javascript', 'go'];
    const ia = order.indexOf(a);
    const ib = order.indexOf(b);
    if (ia === -1 && ib === -1) return a.localeCompare(b);
    if (ia === -1) return 1;
    if (ib === -1) return -1;
    return ia - ib;
  });

  // Collect model names — prefer tier model_stats when a tier is active
  let modelNames = [];
  if (selectedTier && data.tiers?.[selectedTier]?.model_stats) {
    modelNames = Object.keys(data.tiers[selectedTier].model_stats).sort();
  } else {
    modelNames = Object.keys(data.models || {}).sort();
  }

  // Available tiers
  const availTiers = Object.keys(data.tiers || {}).filter(
    (t) => ['smoke', 'core', 'stretch', 'vision'].includes(t)
  );

  // Helper: get success rate for a language in a model under active tier
  function getRate(modelKey, lang) {
    if (selectedTier && data.tiers?.[selectedTier]?.model_stats) {
      const ms = data.tiers[selectedTier].model_stats[modelKey];
      return ms?.languages?.[lang]?.success_rate ?? null;
    }
    return data.models?.[modelKey]?.languages?.[lang]?.successRate ?? null;
  }

  function displayName(key) {
    // Shorten long model names for column headers
    return key
      .replace('claude-', 'Claude ')
      .replace('gemini-', 'Gemini ')
      .replace('opencode-', 'OC/')
      .replace('gpt5', 'GPT-5')
      .replace(/-/g, ' ');
  }

  const tableStyle = { width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' };
  const thStyle = {
    padding: '8px 12px',
    background: 'var(--ifm-color-emphasis-100)',
    fontWeight: 600,
    textAlign: 'center',
    fontSize: '0.75rem',
    whiteSpace: 'nowrap',
    borderBottom: '2px solid var(--ifm-color-emphasis-200)',
  };
  const rowHeaderStyle = {
    padding: '8px 14px',
    fontWeight: 700,
    textAlign: 'left',
    whiteSpace: 'nowrap',
    borderRight: '2px solid var(--ifm-color-emphasis-200)',
  };

  return (
    <div>
      <DimensionSelector
        tiers={availTiers}
        selectedTier={selectedTier}
        onTierChange={setSelectedTier}
      />

      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: '8px' }}>
        Pass rate (%) by language × model. Green ≥ 85% · Yellow 50–74% · Red &lt; 30% · — = no results yet.
      </p>

      <div style={{ overflowX: 'auto' }}>
        <table style={tableStyle}>
          <thead>
            <tr>
              <th style={{ ...thStyle, textAlign: 'left' }}>Language</th>
              {modelNames.map((m) => (
                <th key={m} style={thStyle} title={m}>
                  {displayName(m)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {langs.map((lang) => (
              <tr key={lang} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
                <td style={rowHeaderStyle}>{LANG_DISPLAY[lang] || lang}</td>
                {modelNames.map((m) => (
                  <Cell key={m} rate={getRate(m, lang)} />
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
