import React, { useState, useEffect } from 'react';
import styles from './styles.module.css';

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_SHORT = { ailang: 'AILANG', python: 'Python', javascript: 'JS', go: 'Go' };
const LANG_COLOR = { ailang: '#6366f1', python: '#eab308', javascript: '#f97316', go: '#06b6d4' };
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

// Adjusted success rate: excludes API errors from denominator
// adj = passes / (total - api_errors) = rate / (1 - apiErrorRate)
function adjRate(rate, apiErrorRate) {
  if (rate == null) return null;
  if (!apiErrorRate || apiErrorRate <= 0) return rate;
  const adj = rate / (1 - apiErrorRate);
  return Math.min(1.0, adj);
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

// Mini bar row: label | ████░░░░ 72%
function MiniBar({ lang, rate }) {
  const p = rate != null ? Math.round(rate * 100) : null;
  const color = LANG_COLOR[lang] || '#94a3b8';
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 5 }}>
      <span style={{ width: 44, fontSize: '0.72rem', color: 'var(--ifm-color-emphasis-600)', flexShrink: 0, textAlign: 'right' }}>
        {LANG_SHORT[lang] || lang}
      </span>
      <div style={{ flex: 1, background: 'var(--ifm-color-emphasis-100)', borderRadius: 3, height: 10, overflow: 'hidden' }}>
        {p != null && (
          <div style={{ width: `${p}%`, height: '100%', background: color, borderRadius: 3, transition: 'width 0.3s' }} />
        )}
      </div>
      <span style={{ width: 30, fontSize: '0.72rem', fontWeight: p >= 85 ? 700 : 400, color: p == null ? 'var(--ifm-color-emphasis-300)' : rateColor(rate), flexShrink: 0 }}>
        {p != null ? `${p}%` : '—'}
      </span>
    </div>
  );
}

// Section A: one card per model, bars inside = languages side-by-side
function ModelLanguageSpread({ models, data, allLangs }) {
  return (
    <div style={{ marginBottom: 32 }}>
      <h3 style={{ marginTop: 0, marginBottom: 4, fontSize: '1rem' }}>Language Spread by Model</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        Agent pass rate per language, one card per model (core tier).
      </p>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12 }}>
        {models.map(m => {
          const md = data.models[m];
          const harness = HARNESS_LABEL[md?.agent_cli] || md?.agent_cli || '?';
          return (
            <div key={m} style={{
              border: '1px solid var(--ifm-color-emphasis-200)', borderRadius: 8,
              padding: '10px 14px', minWidth: 220, flex: '1 1 220px', maxWidth: 300,
              background: 'var(--ifm-background-color)',
            }}>
              <div style={{ fontWeight: 700, fontSize: '0.85rem', marginBottom: 2 }}>{modelShort(m)}</div>
              <div style={{ fontSize: '0.72rem', color: 'var(--ifm-color-emphasis-500)', marginBottom: 10 }}>{harness}</div>
              {allLangs.map(l => (
                <MiniBar key={l} lang={l} rate={md?.languages?.[l]?.successRate ?? null} />
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}

// Section B: same model, different harness — grouped by model_family
// Shows raw rate + adjusted rate (excludes API errors from denominator)
function CrossHarnessTable({ data, allLangs }) {
  const families = {};
  for (const [id, m] of Object.entries(data.models || {})) {
    const fam = m.model_family || id;
    if (!families[fam]) families[fam] = [];
    families[fam].push({ id, ...m });
  }
  const crossHarness = Object.entries(families).filter(([, ms]) => ms.length > 1);
  if (crossHarness.length === 0) return null;

  // Check if any model has non-trivial API error rates
  const anyApiErrors = crossHarness.some(([, variants]) =>
    variants.some(v => (v.agentStats?.apiErrorRate ?? 0) > 0.05)
  );

  return (
    <div style={{ marginBottom: 32 }}>
      <h3 style={{ marginTop: 0, marginBottom: 4, fontSize: '1rem' }}>Same Model, Different Harness</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 4 }}>
        Effect of harness choice on the same underlying model. ↑ = harness improves over baseline.
      </p>
      {anyApiErrors && (
        <div style={{ fontSize: '0.75rem', padding: '6px 10px', background: 'rgba(239,68,68,0.08)', borderRadius: 6, marginBottom: 10, borderLeft: '3px solid #ef4444' }}>
          ⚠ <strong>API errors inflate failure rates</strong> for some harnesses (shown in red). <em>Adj%</em> = pass rate excluding API errors from the denominator — a fairer measure of code generation quality.
        </div>
      )}
      <div className={styles.tableScroll}>
        <table style={{ borderCollapse: 'collapse', fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Model / Harness</th>
              {allLangs.map(l => <th key={l} style={th}>{LANG_LABEL[l] || l}</th>)}
              {anyApiErrors && <th style={th}>API Errors</th>}
            </tr>
          </thead>
          <tbody>
            {crossHarness.map(([fam, variants]) => {
              const sorted = [...variants].sort((a, b) => {
                const order = { claude: 0, gemini: 1, codex: 2, opencode: 3 };
                return (order[a.agent_cli] ?? 9) - (order[b.agent_cli] ?? 9);
              });
              const baseline = {};
              for (const l of allLangs) {
                baseline[l] = sorted[0]?.languages?.[l]?.successRate ?? null;
              }
              return sorted.map((v, i) => {
                const apiErrRate = v.agentStats?.apiErrorRate ?? 0;
                const apiErrCount = v.agentStats?.apiErrors ?? 0;
                return (
                  <tr key={v.id} style={{ ...rowStyle, ...(i === sorted.length - 1 ? { borderBottom: '2px solid var(--ifm-color-emphasis-300)' } : {}) }}>
                    <td style={{ ...rowHeader, fontWeight: i === 0 ? 700 : 400, paddingLeft: i === 0 ? 14 : 28, color: i === 0 ? 'inherit' : 'var(--ifm-color-emphasis-600)' }}>
                      {i === 0 && <span style={{ display: 'block', fontWeight: 700 }}>{familyLabel(fam)}</span>}
                      <span style={{ fontSize: '0.78rem', color: 'var(--ifm-color-emphasis-500)' }}>
                        {HARNESS_LABEL[v.agent_cli] || v.agent_cli}
                      </span>
                    </td>
                    {allLangs.map(l => {
                      const rate = v.languages?.[l]?.successRate ?? null;
                      const base = i > 0 ? baseline[l] : null;
                      const delta = (base != null && rate != null) ? Math.round((rate - base) * 100) : null;
                      const adj = adjRate(rate, apiErrRate);
                      const showAdj = apiErrRate > 0.05 && adj != null && Math.round(adj * 100) !== pct(rate);
                      return (
                        <td key={l} style={{ textAlign: 'center', padding: '8px 12px', background: heatBg(rate), verticalAlign: 'middle' }}>
                          {rate == null ? '—' : (
                            <>
                              <span style={{ color: rateColor(rate), fontWeight: rate >= 0.85 ? 700 : 400 }}>
                                {Math.round(rate * 100)}%
                              </span>
                              {delta != null && delta !== 0 && (
                                <span style={{ fontSize: '0.7rem', marginLeft: 4, color: delta > 0 ? '#15803d' : '#b91c1c', fontWeight: 700 }}>
                                  {delta > 0 ? `↑+${delta}` : `↓${delta}`}pp
                                </span>
                              )}
                              {showAdj && (
                                <div style={{ fontSize: '0.68rem', color: '#6366f1', marginTop: 1 }}>
                                  adj {Math.round(adj * 100)}%
                                </div>
                              )}
                            </>
                          )}
                        </td>
                      );
                    })}
                    {anyApiErrors && (
                      <td style={{ textAlign: 'center', padding: '8px 12px', fontSize: '0.78rem', color: apiErrRate > 0.05 ? '#b91c1c' : 'var(--ifm-color-emphasis-500)' }}>
                        {apiErrCount > 0 ? (
                          <>
                            <span style={{ fontWeight: apiErrRate > 0.05 ? 700 : 400 }}>{Math.round(apiErrRate * 100)}%</span>
                            <span style={{ display: 'block', fontSize: '0.65rem', color: 'var(--ifm-color-emphasis-400)' }}>({apiErrCount} runs)</span>
                          </>
                        ) : '—'}
                      </td>
                    )}
                  </tr>
                );
              });
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

  // Fixed 4-language set for agent mode explorer
  const allLangs = LANG_ORDER;
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

      {/* Section A: language spread bar chart */}
      <ModelLanguageSpread models={models} data={data} allLangs={langs} />

      {/* Section B: cross-harness delta (hidden when a single harness is selected) */}
      {!activeHarness && <CrossHarnessTable data={data} allLangs={langs} />}

      {/* Table 1: language × model heatmap */}
      <h3 style={{ marginTop: 0, marginBottom: 8, fontSize: '1rem' }}>Pass Rate by Language × Model</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        Green ≥ 85% · Yellow 50–84% · Red &lt; 30% · — = no results yet
      </p>

      <div className={styles.tableScroll} style={{ marginBottom: 32 }}>
        <table style={{ borderCollapse: 'collapse', fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
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

      <div className={styles.tableScroll}>
        <table style={{ borderCollapse: 'collapse', fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
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
