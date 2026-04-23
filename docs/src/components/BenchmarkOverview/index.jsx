import React, { useState, useEffect } from 'react';
import LocalCloudBadge from '@site/src/components/LocalCloudBadge';

const LANG_DISPLAY = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];

function pct(v) {
  return v != null ? `${Math.round(v * 100)}%` : '—';
}

function MetricCard({ label, value, sub, color }) {
  return (
    <div
      style={{
        flex: '1 1 160px',
        minWidth: 140,
        padding: '16px 20px',
        borderRadius: '10px',
        background: 'var(--ifm-color-emphasis-100)',
        borderLeft: `4px solid ${color || 'var(--ifm-color-primary)'}`,
      }}
    >
      <div style={{ fontSize: '1.6rem', fontWeight: 700, color: color || 'var(--ifm-color-primary)' }}>
        {value}
      </div>
      <div style={{ fontWeight: 600, marginTop: 2 }}>{label}</div>
      {sub && <div style={{ fontSize: '0.75rem', color: 'var(--ifm-color-emphasis-600)', marginTop: 2 }}>{sub}</div>}
    </div>
  );
}

function LangCard({ lang, stats }) {
  if (!stats) return null;
  const sr = stats.success_rate ?? stats.agent_success_rate;
  const runs = stats.total_runs ?? stats.agent_runs;
  return (
    <div
      style={{
        flex: '1 1 140px',
        minWidth: 120,
        padding: '12px 16px',
        borderRadius: '8px',
        background: 'var(--ifm-color-emphasis-100)',
        border: '1px solid var(--ifm-color-emphasis-200)',
      }}
    >
      <div style={{ fontWeight: 700, marginBottom: 4 }}>{LANG_DISPLAY[lang] || lang}</div>
      <div style={{ fontSize: '1.3rem', fontWeight: 700, color: sr >= 0.8 ? '#15803d' : sr >= 0.6 ? '#b45309' : '#b91c1c' }}>
        {pct(sr)}
      </div>
      <div style={{ fontSize: '0.75rem', color: 'var(--ifm-color-emphasis-600)' }}>{runs ? `${runs} runs` : 'No results yet'}</div>
    </div>
  );
}

function HarnessCard({ name, stats }) {
  const pt = stats.provider_type;
  const ts = stats.timeout_scale;
  return (
    <div
      style={{
        flex: '1 1 180px',
        minWidth: 160,
        padding: '12px 16px',
        borderRadius: '8px',
        background: 'var(--ifm-color-emphasis-100)',
        border: '1px solid var(--ifm-color-emphasis-200)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
        <span style={{ fontWeight: 700 }}>{stats.display_name || name}</span>
        {pt && <LocalCloudBadge providerType={pt} timeoutScale={ts} />}
      </div>
      <div style={{ fontSize: '1.2rem', fontWeight: 700, color: 'var(--ifm-color-primary)' }}>
        {pct(stats.success_rate)}
      </div>
      <div style={{ fontSize: '0.75rem', color: 'var(--ifm-color-emphasis-600)' }}>
        {stats.total_runs} runs · {stats.models?.length || 0} models
      </div>
    </div>
  );
}

/**
 * Landing overview: headline metrics + per-language cards + per-harness cards.
 * Links to dedicated dimension pages for drill-down.
 */
export default function BenchmarkOverview() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then((r) => r.json())
      .then(setData)
      .catch((e) => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load benchmark data: {error}</p>;
  if (!data) return <p>Loading benchmark data…</p>;

  const core = data.tiers?.core;
  const totalRuns = data.totalRuns || 0;
  const agg = data.aggregates || {};

  const coreAilangRate = core?.ailang_success_rate;
  const coreModels = core ? Object.keys(core.model_stats || {}).length : 0;

  // Sorted languages: known order first, then alphabetical
  const langKeys = Object.keys(data.languages || {}).sort((a, b) => {
    const ia = LANG_ORDER.indexOf(a);
    const ib = LANG_ORDER.indexOf(b);
    if (ia === -1 && ib === -1) return a.localeCompare(b);
    if (ia === -1) return 1;
    if (ib === -1) return -1;
    return ia - ib;
  });

  const harnessKeys = Object.keys(data.harnesses || {}).sort();

  return (
    <div>
      {/* Headline metrics */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '12px', marginBottom: '28px' }}>
        <MetricCard
          label="Core Tier Pass Rate"
          value={pct(coreAilangRate)}
          sub="AILANG, core benchmarks"
          color="#2563eb"
        />
        <MetricCard
          label="Total Runs"
          value={totalRuns.toLocaleString()}
          sub={`${coreModels} models · v${data.version}`}
          color="#7c3aed"
        />
        {agg.agentSuccessRate != null && (
          <MetricCard
            label="Agent Pass Rate"
            value={pct(agg.agentSuccessRate)}
            sub={`${agg.agentRuns || 0} agentic runs`}
            color="#059669"
          />
        )}
        {agg.totalCostUSD != null && (
          <MetricCard
            label="Total Cost"
            value={`$${agg.totalCostUSD?.toFixed(2) ?? '—'}`}
            sub="across all runs"
            color="#b45309"
          />
        )}
      </div>

      {/* Per-language summary */}
      <h3 style={{ marginBottom: '10px' }}>
        By Language{' '}
        <a href="/docs/benchmarks/by-language" style={{ fontSize: '0.8rem', fontWeight: 400 }}>
          → full view
        </a>
      </h3>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px', marginBottom: '28px' }}>
        {langKeys.map((l) => (
          <LangCard key={l} lang={l} stats={data.languages[l]} />
        ))}
        {langKeys.length === 0 && (
          <p style={{ color: 'var(--ifm-color-emphasis-500)' }}>No language data available.</p>
        )}
      </div>

      {/* Per-harness summary */}
      {harnessKeys.length > 0 && (
        <>
          <h3 style={{ marginBottom: '10px' }}>
            By Harness{' '}
            <a href="/docs/benchmarks/by-harness" style={{ fontSize: '0.8rem', fontWeight: 400 }}>
              → full view
            </a>
          </h3>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px', marginBottom: '28px' }}>
            {harnessKeys.map((h) => (
              <HarnessCard key={h} name={h} stats={data.harnesses[h]} />
            ))}
          </div>
        </>
      )}

      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)' }}>
        Data from baseline <strong>v{data.version}</strong> · Generated{' '}
        {data.timestamp ? new Date(data.timestamp).toLocaleDateString() : '—'} ·{' '}
        <a href="/docs/benchmarks/performance">Full standard eval details →</a>
      </p>
    </div>
  );
}
