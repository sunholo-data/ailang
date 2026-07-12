import React, { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import styles from './styles.module.css';

const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_SHORT = { ailang: 'AILANG', python: 'Python', javascript: 'JS', go: 'Go' };
const LANG_COLOR = { ailang: '#6366f1', python: '#eab308', javascript: '#f97316', go: '#06b6d4' };
// `gemini` rows are historical (Gemini CLI executor retired v0.22.0); do not relabel them.
// `managed_agents` is the Antigravity successor (Vertex Managed Agents API).
const HARNESS_LABEL = { claude: 'Claude CLI', gemini: 'Gemini CLI (retired)', managed_agents: 'Managed Agents', opencode: 'opencode', codex: 'Codex', pi: 'Pi', motoko: 'motoko_agent' };
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

// Cell display priority: ADJUSTED rate is the headline (true model strength),
// raw rate shown only when meaningfully different (≥1pp delta) so users can
// see the infra noise that's been factored out. Heatmap color follows the
// number being shown as primary.
function Cell({ rate, adjusted, apiErrorRate }) {
  // Hide adjusted when apiErrorRate > 0.5: at that threshold the formula
  // successRate / (1 - apiErrorRate) amplifies noise to absurd values
  // (e.g. 9% raw / 0.09 denominator = 100% "adjusted").
  const showAdjusted = adjusted != null && apiErrorRate != null
    && apiErrorRate > 0.05 && apiErrorRate <= 0.5
    && rate != null && Math.abs(adjusted - rate) >= 0.01;
  const primary = showAdjusted ? adjusted : rate;
  const p = pct(primary);
  const showRaw = showAdjusted && rate != null;
  const highErrorRate = apiErrorRate != null && apiErrorRate > 0.5;
  return (
    <td style={{ textAlign: 'center', padding: '8px 12px', background: heatBg(primary), color: rateColor(primary), fontWeight: primary >= 0.85 ? 700 : 400 }}>
      {p == null ? '—' : `${p}%`}
      {showRaw && (
        <span style={{ display: 'block', fontSize: '0.7em', color: 'var(--ifm-color-emphasis-500)', fontWeight: 400, fontStyle: 'italic' }}
              title={`Raw rate before excluding API errors`}>
          (raw {Math.round(rate * 100)}%)
        </span>
      )}
      {highErrorRate && (
        <span style={{ display: 'block', fontSize: '0.65em', color: 'var(--ifm-color-emphasis-400)', fontWeight: 400 }}
              title={`${Math.round(apiErrorRate * 100)}% of runs were API errors — adjusted rate unreliable`}>
          ⚠ {Math.round(apiErrorRate * 100)}% errors
        </span>
      )}
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

const TIER_LABEL = { smoke: 'Smoke', core: 'Core', stretch: 'Stretch', frontier: 'Frontier', vision: 'Vision' };
const TIER_ORDER = ['smoke', 'core', 'stretch', 'frontier', 'vision'];

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
  // Surface harness + provider as explicit suffixes instead of cryptic prefix
  // chains. Examples:
  //   opencode-or-glm-5    → "GLM 5 (agent · OR)"
  //   opencode-sonnet-4-6  → "Sonnet 4.6 (agent)"
  //   or-glm-5             → "GLM 5 (OR)"
  //   claude-sonnet-4-6    → "Claude Sonnet 4.6"
  //   gpt5-4-mini          → "GPT-5 4 mini"
  let s = key;
  let suffix = '';
  if (s.startsWith('opencode-or-')) { suffix = ' (agent · OR)'; s = s.slice('opencode-or-'.length); }
  else if (s.startsWith('opencode-')) { suffix = ' (agent)';     s = s.slice('opencode-'.length); }
  else if (s.startsWith('pi-'))       { suffix = ' (Pi)';        s = s.slice('pi-'.length); }
  else if (s.startsWith('or-'))       { suffix = ' (OR)';        s = s.slice('or-'.length); }
  s = s
    .replace(/^claude-/, 'Claude ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^gpt5/, 'GPT-5')
    .replace(/^minimax-/, 'MiniMax ')
    .replace(/^glm-/, 'GLM ')
    .replace(/^kimi-/, 'Kimi ')
    .replace(/^qwen3-/, 'Qwen3 ')
    .replace(/^gemma4-/, 'Gemma4 ')
    .replace(/^deepseek-/, 'DeepSeek ')
    .replace(/-/g, ' ');
  return s + suffix;
}

function familyLabel(fam) {
  if (!fam) return 'Unknown';
  return fam.replace('claude-', 'Claude ').replace('gemini-', 'Gemini ').replace(/-/g, ' ');
}

// Mini bar row: label | ████░░░░ 91%
//                              (raw 18%)
// Display priority: ADJUSTED is primary (true model strength when infra works);
// RAW shown on a sub-row so the bar's flex-1 area is the same width across
// all rows regardless of whether an adjustment annotation is present.
function MiniBar({ lang, rate, adjusted, apiErrorRate, apiErrors }) {
  const showAdjusted = apiErrorRate != null && apiErrorRate > 0.05 && apiErrorRate <= 0.5
    && adjusted != null && rate != null && Math.abs(adjusted - rate) >= 0.01;
  const primary = showAdjusted ? adjusted : rate;
  const p = primary != null ? Math.round(primary * 100) : null;
  const color = LANG_COLOR[lang] || '#94a3b8';
  return (
    <div style={{ marginBottom: 5 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
        <span style={{ width: 44, fontSize: '0.72rem', color: 'var(--ifm-color-emphasis-600)', flexShrink: 0, textAlign: 'right' }}>
          {LANG_SHORT[lang] || lang}
        </span>
        <div style={{ flex: 1, background: 'var(--ifm-color-emphasis-100)', borderRadius: 3, height: 10, overflow: 'hidden' }}>
          {p != null && (
            <div style={{ width: `${p}%`, height: '100%', background: color, borderRadius: 3, transition: 'width 0.3s' }} />
          )}
        </div>
        <span style={{ width: 30, fontSize: '0.72rem', fontWeight: p >= 85 ? 700 : 400, color: p == null ? 'var(--ifm-color-emphasis-300)' : rateColor(primary), flexShrink: 0 }}>
          {p != null ? `${p}%` : '—'}
        </span>
      </div>
      {showAdjusted && (
        <div
          style={{ paddingLeft: 50, fontSize: '0.62rem', color: 'var(--ifm-color-emphasis-500)', fontStyle: 'italic', marginTop: 1 }}
          title={`Raw rate before excluding ${apiErrors ?? 0} API errors`}
        >
          (raw {Math.round(rate * 100)}%)
        </div>
      )}
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
              <div style={{ fontWeight: 700, fontSize: '0.85rem', marginBottom: 2 }}>
                {modelShort(m)}
                {md?.source === 'local' && (
                  <span title="On-device model (local Ollama, $0) — from the continuous OS rotation"
                        style={{ marginLeft: 6, fontSize: '0.62rem', fontWeight: 600, padding: '1px 5px', borderRadius: 8, background: 'rgba(99,102,241,0.15)', color: '#6366f1', verticalAlign: 'middle' }}>
                    on-device
                  </span>
                )}
              </div>
              <div style={{ fontSize: '0.72rem', color: 'var(--ifm-color-emphasis-500)', marginBottom: 10 }}>{harness}</div>
              {allLangs.map(l => {
                const ld = md?.languages?.[l];
                // Agent-mode only: no fallback to ld.successRate (that's 0-shot
                // API). Models without agent runs were already filtered out
                // upstream, so ld.agentSuccessRate is the only honest source.
                const rate = ld?.agentSuccessRate ?? null;
                return (
                  <MiniBar
                    key={l}
                    lang={l}
                    rate={rate}
                    adjusted={ld?.agentSuccessRateAdjusted}
                    apiErrorRate={ld?.agentApiErrorRate}
                    apiErrors={ld?.agentApiErrors}
                  />
                );
              })}
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
    // Agent-mode only: skip variants that have no agent runs. Otherwise a
    // standard-only sibling in the model family (e.g. claude-sonnet-4-6 the
    // 0-shot row vs claude-sonnet-4-6 the agent row in older data) would
    // appear as a harness with 0-shot data and inflate the cross-harness
    // delta misleadingly.
    if (m.agentStats == null) continue;
    const fam = m.model_family || id;
    if (!families[fam]) families[fam] = [];
    families[fam].push({ id, ...m });
  }
  // "Different Harness" means exactly that: only show families whose variants
  // span more than one harness (agent_cli). A family with two variants on the
  // SAME harness (e.g. deepseek-v4 flash vs pro, both opencode — different model
  // sizes, not harnesses) is not a cross-harness comparison and would mislead.
  const crossHarness = Object.entries(families)
    .filter(([, ms]) => new Set(ms.map(m => m.agent_cli)).size > 1);
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
          ⚠ <strong>Headline rates exclude API errors</strong> (quota, CLI version mismatches, harness crashes) — these are infrastructure failures, not model failures. <em>(raw N%)</em> shows the unfiltered rate before exclusion.
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
                const order = { claude: 0, gemini: 1, codex: 2, opencode: 3, pi: 4, motoko: 5 };
                return (order[a.agent_cli] ?? 9) - (order[b.agent_cli] ?? 9);
              });
              // Baseline for cross-harness delta uses ADJUSTED rate when available
              // so we compare like-for-like (model strength vs model strength).
              const baseline = {};
              for (const l of allLangs) {
                const ld0 = sorted[0]?.languages?.[l];
                const apiErr0 = sorted[0]?.agentStats?.apiErrorRate ?? 0;
                // Agent-mode only: see note in upstream table.
                const agentRate0 = ld0?.agentSuccessRate ?? null;
                const adj0 = adjRate(agentRate0, apiErr0);
                baseline[l] = (apiErr0 > 0.05 && adj0 != null) ? adj0 : agentRate0;
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
                      const rate = v.languages?.[l]?.agentSuccessRate ?? null;
                      const adj = adjRate(rate, apiErrRate);
                      // ADJUSTED is primary when api errors are non-trivial; raw is secondary.
                      const showAdjusted = apiErrRate > 0.05 && apiErrRate <= 0.5
                        && adj != null && rate != null && Math.abs(adj - rate) >= 0.01;
                      const primary = showAdjusted ? adj : rate;
                      const base = i > 0 ? baseline[l] : null;
                      const delta = (base != null && primary != null) ? Math.round((primary - base) * 100) : null;
                      return (
                        <td key={l} style={{ textAlign: 'center', padding: '8px 12px', background: heatBg(primary), verticalAlign: 'middle' }}>
                          {primary == null ? '—' : (
                            <>
                              <span style={{ color: rateColor(primary), fontWeight: primary >= 0.85 ? 700 : 400 }}>
                                {Math.round(primary * 100)}%
                              </span>
                              {delta != null && delta !== 0 && (
                                <span style={{ fontSize: '0.7rem', marginLeft: 4, color: delta > 0 ? '#15803d' : '#b91c1c', fontWeight: 700 }}>
                                  {delta > 0 ? `↑+${delta}` : `↓${delta}`}pp
                                </span>
                              )}
                              {showAdjusted && rate != null && (
                                <div style={{ fontSize: '0.68rem', color: 'var(--ifm-color-emphasis-500)', marginTop: 1, fontStyle: 'italic' }}>
                                  (raw {Math.round(rate * 100)}%)
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

// Strip the harness prefix to get the cross-harness family key, so the same
// on-device model run through opencode vs pi groups together in Section B.
function osFamilyKey(model) {
  return model
    .replace(/^opencode-or-/, '')
    .replace(/^opencode-/, '')
    .replace(/^pi-/, '')
    .replace(/^motoko-/, '');
}

// Union the continuous OS/Local rotation (os/latest.json) into the baseline
// agent data. The rotation publishes aggregate agent pass rates per
// (model, harness, language) for on-device models, which never appear in the
// release baseline (latest.json) — so without this the Explorer shows only
// cloud harnesses. Merging here (rather than at publish time) keeps the local
// rows live with every rotation cycle instead of freezing them to the release
// cadence. Additive and guarded: a missing/!malformed os file is a no-op.
function mergeOSData(data, os) {
  // Agent-only models (e.g. modern OS models on opencode: deepseek-v4, glm-5.1,
  // minimax-m3) live in a separate `agentModels` map — deliberately kept out of
  // `models` so the *standard* dashboard doesn't render phantom 0% rows for
  // models that have no 0-shot runs. The *agent* Explorer wants them, so fold
  // them into the working models map. Without this, opencode shows "4 models"
  // but zero breakout rows (the harness lists the IDs, but they're absent from
  // `models`), and they miss the heatmap + cross-harness table entirely.
  // Disjoint in practice; let `models` win on any overlap to be safe.
  if (data && data.agentModels) {
    data.models = { ...data.agentModels, ...(data.models || {}) };
  }

  if (!os || !Array.isArray(os.rows)) return data;
  data.models = data.models || {};
  data.harnesses = data.harnesses || {};
  for (const row of os.rows) {
    const { model, harness, lang } = row || {};
    if (!model || !harness || !lang) continue;

    // 1. Model entry — drives the model×language heatmap, the language-spread
    //    cards, and (via model_family) the cross-harness table.
    const m = data.models[model] || {};
    m.agent_cli = harness;
    m.model_family = m.model_family || osFamilyKey(model);
    m.agentStats = m.agentStats || { apiErrorRate: 0, apiErrors: 0 };
    m.source = 'local';
    m.languages = m.languages || {};
    for (const [l, rate] of Object.entries(lang)) {
      // latest.json is authoritative (version-specific, same source as every other
      // table). Only FILL IN a rate the baseline lacks — never overwrite a current
      // rate with the CUMULATIVE os rate. That overwrite is what made motoko-qwen
      // read 100% (v0.29.2, 27 runs) in the Per-Model Breakdown but 93% (cumulative
      // across versions) in the Harness×Language table.
      if (m.languages[l] && m.languages[l].agentSuccessRate != null) continue;
      m.languages[l] = { ...(m.languages[l] || {}), agentSuccessRate: rate };
    }
    data.models[model] = m;

    // 2. Harness entry (harness×language table). Append the model. Only
    //    synthesize aggregates for harnesses the baseline lacks (e.g. pi) so we
    //    never overwrite backend-computed cloud aggregates (e.g. opencode).
    const h = data.harnesses[harness] || { models: [], languages: {}, source: 'local' };
    h.models = h.models || [];
    if (!h.models.includes(model)) h.models.push(model);
    // Recompute the harness aggregate from its models' LIVE os rates whenever EVERY
    // model under it is on-device (local). Otherwise a stale baseline aggregate —
    // e.g. motoko's 100% over 9 banked runs from the unified publish — contradicts
    // the live per-model rate (93%) the breakdown row shows, producing an impossible
    // "100% aggregate, 93% only-model". Cloud/mixed harnesses (opencode) keep their
    // backend-computed aggregate.
    const allLocal = h.models.length > 0 && h.models.every(mm => data.models[mm]?.source === 'local');
    if (h.source === 'local' || allLocal) {
      for (const l of LANG_ORDER) {
        const rates = h.models
          .map(mm => data.models[mm]?.languages?.[l]?.agentSuccessRate)
          .filter(v => v != null);
        if (rates.length) {
          h.languages[l] = { successRate: rates.reduce((s, v) => s + v, 0) / rates.length };
        }
      }
      h.avg_cost_usd = 0; // on-device = free
    }
    data.harnesses[harness] = h;
  }
  return data;
}

export default function BenchmarkExplorer() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [activeHarness, setActiveHarness] = useState(null);
  const [activeLang, setActiveLang] = useState(null);
  const [activeTier, setActiveTier] = useState(null);

  useEffect(() => {
    // Baseline (cloud, release/nightly) + live OS/Local rotation, unioned.
    // The OS file is optional: if it's missing, fall back to baseline only.
    Promise.all([
      benchmarkFetch('latest.json').then(r => r.json()),
      benchmarkFetch('os/latest.json').then(r => (r.ok ? r.json() : null)).catch(() => null),
    ])
      .then(([base, os]) => setData(mergeOSData(base, os)))
      .catch(e => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load: {error}</p>;
  if (!data) return <p>Loading benchmark data…</p>;

  // Fixed 4-language set for agent mode explorer
  const allLangs = LANG_ORDER;
  const langs = activeLang ? [activeLang] : allLangs;

  const allHarnesses = Object.keys(data.harnesses || {}).sort();
  // Collect tiers that appear in at least one harness
  const allTiers = TIER_ORDER.filter(t =>
    Object.values(data.harnesses || {}).some(h => h.tiers?.[t])
  );
  // Agent Harness Explorer = agent-mode only. A model is "agent-mode" iff it
  // actually has agent runs (agentStats != null) — not just `agent_cli` set
  // (which only declares that an executor binding exists). Without this guard,
  // standard-only models like claude-opus-4-7 (no agent runs, but agent_cli:
  // "claude" is configured) leak into the table with their 0-shot successRate
  // shown alongside genuine agent pass rates — apples-to-oranges.
  const allModels = Object.keys(data.models || {})
    .filter(m => data.models[m]?.agentStats != null)
    .sort();
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
      <h3 style={{ marginTop: 0, marginBottom: 8, fontSize: '1rem' }}>Pass Rate by Model × Language</h3>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: 8 }}>
        <strong>Agent mode only.</strong> Standard 0-shot API results are excluded — see the{' '}
        <a href="/ailang/docs/benchmarks/performance">benchmarks page</a> for those.
        Green ≥ 85% · Yellow 50–84% · Red &lt; 30% · — = no agent results for this model × language.
      </p>

      <div className={styles.tableScroll} style={{ marginBottom: 32 }}>
        <table style={{ borderCollapse: 'collapse', fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Model</th>
              {langs.map(lang => <th key={lang} style={th}>{LANG_LABEL[lang] || lang}</th>)}
            </tr>
          </thead>
          <tbody>
            {models.map(m => (
              <tr key={m} style={rowStyle}>
                <td style={rowHeader} title={m}>{modelShort(m)}</td>
                {langs.map(lang => {
                  const ld = data.models[m]?.languages?.[lang];
                  // Agent-mode only: never fall back to ld.successRate (that's
                  // 0-shot API mode). If a model lacks agent runs for this
                  // language, render — rather than mixing modes silently.
                  return (
                    <Cell
                      key={lang}
                      rate={ld?.agentSuccessRate ?? null}
                      adjusted={ld?.agentSuccessRateAdjusted}
                      apiErrorRate={ld?.agentApiErrorRate}
                    />
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Table 2: harness × language summary, with per-model drill-down rows */}
      <h3 style={{ marginTop: 0, marginBottom: 6, fontSize: '1rem' }}>Pass Rate by Harness × Language</h3>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, alignItems: 'center', marginBottom: 8, fontSize: '0.82rem' }}>
        <span style={{ color: 'var(--ifm-color-emphasis-600)' }}>Tier:</span>
        <Chip label="All" active={!activeTier} onClick={() => setActiveTier(null)} />
        {allTiers.map(t => (
          <Chip key={t} label={TIER_LABEL[t] || t} active={activeTier === t} onClick={() => setActiveTier(activeTier === t ? null : t)} />
        ))}
        <span style={{ color: 'var(--ifm-color-emphasis-500)', marginLeft: 8, fontSize: '0.78rem' }}>
          {activeTier
            ? 'Per-harness rates for selected tier. Model sub-rows hidden (no per-tier model breakdown yet).'
            : 'Bold rows are per-harness aggregates; indented rows show each model\'s contribution.'}
        </span>
      </div>

      <div className={styles.tableScroll}>
        <table style={{ borderCollapse: 'collapse', fontSize: '0.85rem', whiteSpace: 'nowrap' }}>
          <thead>
            <tr>
              <th style={{ ...th, textAlign: 'left' }}>Harness / Model</th>
              {langs.map(l => <th key={l} style={th}>{LANG_LABEL[l] || l}</th>)}
              <th style={th}>Avg Cost/run</th>
              <th style={th}>Runs</th>
            </tr>
          </thead>
          <tbody>
            {(activeHarness ? [activeHarness] : allHarnesses).map(h => {
              const hr = data.harnesses[h];
              if (!hr) return null;
              // When a tier filter is active, skip harnesses with no data for that tier
              if (activeTier && !hr.tiers?.[activeTier]) return null;
              const harnessModels = (hr.models || []).slice().sort();
              return (
                <React.Fragment key={h}>
                  <tr style={{ ...rowStyle, background: 'var(--ifm-color-emphasis-100)' }}>
                    <td style={{ ...rowHeader, fontWeight: 700 }}>
                      {HARNESS_LABEL[h] || h}
                      {activeTier && <span style={{ fontWeight: 400, fontSize: '0.75rem', color: 'var(--ifm-color-emphasis-500)', marginLeft: 6 }}>({TIER_LABEL[activeTier] || activeTier})</span>}
                    </td>
                    {langs.map(l => {
                      const hl = activeTier
                        ? hr.tiers?.[activeTier]?.[l]
                        : hr.languages?.[l];
                      return (
                        <Cell
                          key={l}
                          rate={hl?.successRate ?? null}
                          adjusted={hl?.successRateAdjusted}
                          apiErrorRate={hl?.apiErrorRate}
                        />
                      );
                    })}
                    <td style={{ textAlign: 'center', padding: '8px 12px', fontSize: '0.8rem', fontWeight: 600 }}>
                      {hr.avg_cost_usd != null ? `$${hr.avg_cost_usd.toFixed(4)}` : '—'}
                    </td>
                    <td style={{ textAlign: 'center', padding: '8px 12px', fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', fontWeight: 600 }}>
                      {activeTier
                        ? (() => { const runs = langs.reduce((s, l) => s + (hr.tiers?.[activeTier]?.[l]?.runs ?? 0), 0); return runs > 0 ? `${runs} runs` : '—'; })()
                        : `${harnessModels.length} model${harnessModels.length === 1 ? '' : 's'}`}
                    </td>
                  </tr>
                  {!activeTier && harnessModels.map(modelName => {
                    const m = data.models?.[modelName];
                    if (!m) return null;
                    const modelLangs = m.languages || {};
                    // Per-model average cost: total cost / total runs across all langs.
                    const totalRuns = m.aggregates?.totalRuns || m.totalRuns || 0;
                    const totalCost = m.aggregates?.totalCostUSD || (m.agentStats?.avgCost || 0) * totalRuns || 0;
                    const avgCost = totalRuns > 0 ? totalCost / totalRuns : null;
                    return (
                      <tr key={`${h}-${modelName}`} style={rowStyle}>
                        <td style={{ ...rowHeader, paddingLeft: 32, fontWeight: 400, fontSize: '0.82rem', color: 'var(--ifm-color-emphasis-700)' }}>
                          ↳ {modelShort(modelName)}
                        </td>
                        {langs.map(l => {
                          const ld = modelLangs[l];
                          // Agent-mode only: see note on the model × language
                          // table above — no fallback to ld.successRate here.
                          return (
                            <Cell
                              key={l}
                              rate={ld?.agentSuccessRate ?? null}
                              adjusted={ld?.agentSuccessRateAdjusted}
                              apiErrorRate={ld?.agentApiErrorRate}
                            />
                          );
                        })}
                        <td style={{ textAlign: 'center', padding: '6px 12px', fontSize: '0.78rem', color: 'var(--ifm-color-emphasis-700)' }}>
                          {avgCost != null ? `$${avgCost.toFixed(4)}` : '—'}
                        </td>
                        <td style={{ textAlign: 'center', padding: '6px 12px', fontSize: '0.78rem', color: 'var(--ifm-color-emphasis-600)' }}>
                          {totalRuns}
                        </td>
                      </tr>
                    );
                  })}
                </React.Fragment>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
