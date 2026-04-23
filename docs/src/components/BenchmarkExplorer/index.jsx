import React, { useState, useEffect } from 'react';

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_SHORT = { ailang: 'AI', python: 'Py', javascript: 'JS', go: 'Go' };
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

function rateColor(rate) {
  if (rate == null) return 'var(--ifm-color-emphasis-400)';
  if (rate >= 0.85) return '#15803d';
  if (rate < 0.30) return '#b91c1c';
  return 'inherit';
}

function Cell({ rate }) {
  const p = pct(rate);
  return (
    <td style={{ textAlign: 'center', padding: '8px 12px', background: heatBg(rate), color: rateColor(rate), fontWeight: rate >= 0.85 ? 700 : 400 }}>
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
const rowStyle = { borderBottom: '1px solid var(--ifm-color-emphasis-200)' };

function modelShort(key) {
  return key.replace('claude-', 'Claude ').replace('gemini-', 'Gemini ').replace('opencode-', 'OC/').replace('gpt5', 'GPT-5').replace(/-/g, ' ');
}

function familyLabel(fam) {
  if (!fam) return 'Unknown';
  return fam.replace('claude-', 'Claude ').replace('gemini-', 'Gemini ').replace(/-/g, ' ');
}

// Section A: one card per model showing all 4 language rates
function ModelLanguageSpread({ models, data, allLangs }) {
  return (
    <div style={{ marginBottom: 28 }}>
      <h3 style={{ marginTop: 0, marginBottom: 6, fontSize: '1rem' }}>Language Spread by Model</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 12 }}>
        Pass rate across all 4 languages for each model — agent mode, core tier.
      </p>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
        {models.map(m => {
          const md = data.models[m];
          const harness = HARNESS_LABEL[md?.agent_cli] || md?.agent_cli || '?';
          const rates = allLangs.map(l => {
            const r = md?.languages?.[l]?.successRate;
            return { l, r };
          }).filter(x => x.r != null);
          return (
            <div key={m} style={{
              border: '1px solid var(--ifm-color-emphasis-200)',
              borderRadius: 8, padding: '10px 14px', minWidth: 200,
              background: 'var(--ifm-background-color)',
            }}>
              <div style={{ fontWeight: 700, fontSize: '0.85rem', marginBottom: 4 }}>{modelShort(m)}</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--ifm-color-emphasis-500)', marginBottom: 8 }}>{harness}</div>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                {rates.map(({ l, r }) => (
                  <span key={l} style={{
                    fontSize: '0.78rem', padding: '2px 7px', borderRadius: 10,
                    background: heatBg(r), color: rateColor(r),
                    fontWeight: r >= 0.85 ? 700 : 400,
                    border: '1px solid var(--ifm-color-emphasis-200)',
                  }}>
                    {LANG_SHORT[l] || l} {Math.round(r * 100)}%
                  </span>
                ))}
                {allLangs.filter(l => md?.languages?.[l]?.successRate == null).map(l => (
                  <span key={l} style={{
                    fontSize: '0.78rem', padding: '2px 7px', borderRadius: 10,
                    color: 'var(--ifm-color-emphasis-400)',
                    border: '1px solid var(--ifm-color-emphasis-200)',
                  }}>
                    {LANG_SHORT[l] || l} —
                  </span>
                ))}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Section B: same model, different harness — grouped by model_family
function CrossHarnessTable({ data, allLangs }) {
  // Group models by model_family
  const families = {};
  for (const [id, m] of Object.entries(data.models || {})) {
    const fam = m.model_family || id;
    if (!families[fam]) families[fam] = [];
    families[fam].push({ id, ...m });
  }
  const crossHarness = Object.entries(families).filter(([, ms]) => ms.length > 1);
  if (crossHarness.length === 0) return null;

  return (
    <div style={{ marginBottom: 28 }}>
      <h3 style={{ marginTop: 0, marginBottom: 6, fontSize: '1rem' }}>Same Model, Different Harness</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        When the same underlying model runs through different agent CLIs. ↑ = harness improves over the baseline.
      </p>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Model Family</th>
              {allLangs.map(l => <th key={l} style={th}>{LANG_LABEL[l] || l}</th>)}
            </tr>
          </thead>
          <tbody>
            {crossHarness.map(([fam, variants]) => {
              // Sort: native CLI first, opencode second
              const sorted = [...variants].sort((a, b) => {
                const order = { claude: 0, gemini: 1, codex: 2, opencode: 3 };
                return (order[a.agent_cli] ?? 9) - (order[b.agent_cli] ?? 9);
              });
              // Find the baseline (first native CLI rate) per language
              const baseline = {};
              for (const l of allLangs) {
                baseline[l] = sorted[0]?.languages?.[l]?.successRate ?? null;
              }
              return sorted.map((v, i) => (
                <tr key={v.id} style={{ ...rowStyle, ...(i === sorted.length - 1 ? { borderBottom: '2px solid var(--ifm-color-emphasis-300)' } : {}) }}>
                  <td style={{ ...rowHeader, fontWeight: i === 0 ? 700 : 400, color: i === 0 ? 'inherit' : 'var(--ifm-color-emphasis-600)', paddingLeft: i === 0 ? 14 : 28 }}>
                    {i === 0 ? familyLabel(fam) : ''}
                    <span style={{ fontSize: '0.75rem', fontWeight: 400, marginLeft: 6, color: 'var(--ifm-color-emphasis-500)' }}>
                      {HARNESS_LABEL[v.agent_cli] || v.agent_cli}
                    </span>
                  </td>
                  {allLangs.map(l => {
                    const rate = v.languages?.[l]?.successRate ?? null;
                    const base = i > 0 ? baseline[l] : null;
                    const delta = (base != null && rate != null) ? Math.round((rate - base) * 100) : null;
                    const apiErrRate = v.agentStats?.apiErrorRate;
                    return (
                      <td key={l} style={{ textAlign: 'center', padding: '8px 12px', background: heatBg(rate), color: rateColor(rate), fontWeight: rate >= 0.85 ? 700 : 400 }}>
                        {rate == null ? '—' : (
                          <>
                            {Math.round(rate * 100)}%
                            {delta != null && delta !== 0 && (
                              <span style={{ fontSize: '0.7rem', marginLeft: 4, color: delta > 0 ? '#15803d' : '#b91c1c', fontWeight: 700 }}>
                                {delta > 0 ? `↑+${delta}` : `↓${delta}`}pp
                              </span>
                            )}
                            {i === 0 && apiErrRate != null && apiErrRate > 0.05 && (
                              <span style={{ display: 'block', fontSize: '0.65rem', color: '#b91c1c', marginTop: 1 }}>
                                {Math.round(apiErrRate * 100)}% API err
                              </span>
                            )}
                          </>
                        )}
                      </td>
                    );
                  })}
                </tr>
              ));
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

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

  const allLangs = Object.keys(data.languages || {}).sort((a, b) => {
    const ia = LANG_ORDER.indexOf(a), ib = LANG_ORDER.indexOf(b);
    if (ia < 0 && ib < 0) return a.localeCompare(b);
    if (ia < 0) return 1; if (ib < 0) return -1;
    return ia - ib;
  });
  const langs = activeLang ? [activeLang] : allLangs;

  const allHarnesses = Object.keys(data.harnesses || {}).sort();
  const allModels = Object.keys(data.models || {}).sort();
  const models = activeHarness
    ? allModels.filter(m => data.models[m]?.agent_cli === activeHarness)
    : allModels;

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

      {/* Section A: language spread cards */}
      <ModelLanguageSpread models={models} data={data} allLangs={allLangs} />

      {/* Section B: cross-harness delta (hidden when a single harness is selected) */}
      {!activeHarness && <CrossHarnessTable data={data} allLangs={langs} />}

      {/* Table 1: language × model heatmap */}
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

      {/* Table 2: harness × language summary */}
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
