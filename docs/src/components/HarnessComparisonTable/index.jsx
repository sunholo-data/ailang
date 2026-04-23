import React, { useState, useEffect } from 'react';
import LocalCloudBadge from '@site/src/components/LocalCloudBadge';

function pct(v) {
  return v != null ? `${Math.round(v * 100)}%` : '—';
}
function cost(v) {
  return v != null ? `$${v.toFixed(4)}` : '—';
}
function dur(ms) {
  if (ms == null) return '—';
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;
}

function deltaStyle(v) {
  if (v == null) return {};
  if (v > 0.005) return { color: '#15803d', fontWeight: 700 };
  if (v < -0.005) return { color: '#b91c1c', fontWeight: 700 };
  return { color: 'var(--ifm-color-emphasis-600)' };
}

const thStyle = {
  padding: '8px 12px',
  background: 'var(--ifm-color-emphasis-100)',
  fontWeight: 600,
  textAlign: 'right',
  fontSize: '0.8rem',
  borderBottom: '2px solid var(--ifm-color-emphasis-200)',
};
const tdStyle = { padding: '7px 12px', textAlign: 'right', fontSize: '0.85rem' };
const deltaRowStyle = {
  background: 'rgba(0,0,0,0.03)',
  borderTop: '1px dashed var(--ifm-color-emphasis-300)',
  fontStyle: 'italic',
  fontSize: '0.8rem',
};

/**
 * Groups models by model_family and shows per-harness rows + a Δ delta row.
 * When model_family data is absent (M-EVAL-CROSS-HARNESS not yet run), falls
 * back to listing all harnesses ungrouped.
 */
export default function HarnessComparisonTable() {
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

  const harnesses = data.harnesses || {};
  const models = data.models || {};

  if (Object.keys(harnesses).length === 0) {
    return (
      <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>
        No agent harness results yet. Run <code>ailang eval-suite --agent --models agent_suite</code> to
        populate this view.
      </p>
    );
  }

  // Group models by model_family where available
  const families = {}; // familyKey -> { models: [{key, harness, stats}] }
  const ungrouped = []; // models without model_family

  for (const [modelKey, modelStats] of Object.entries(models)) {
    const family = modelStats.model_family;
    const harness = modelStats.agent_cli;
    if (!harness) continue; // standard-eval-only model, skip
    const entry = { key: modelKey, harness, stats: modelStats };
    if (family) {
      if (!families[family]) families[family] = { models: [] };
      families[family].models.push(entry);
    } else {
      ungrouped.push(entry);
    }
  }

  function renderFamilyGroup(familyKey, { models: familyModels }) {
    // Sort so rows are stable
    const sorted = [...familyModels].sort((a, b) => a.harness.localeCompare(b.harness));

    // Compute delta when exactly 2 harnesses present
    let deltaRow = null;
    if (sorted.length === 2) {
      const [a, b] = sorted;
      const srA = a.stats.agentStats?.successRate ?? null;
      const srB = b.stats.agentStats?.successRate ?? null;
      const costA = a.stats.agentStats?.avgCost ?? null;
      const costB = b.stats.agentStats?.avgCost ?? null;
      const durA = a.stats.agentStats?.avgDurationMs ?? null;
      const durB = b.stats.agentStats?.avgDurationMs ?? null;

      const srDelta = srA != null && srB != null ? srB - srA : null;
      const costDelta = costA != null && costB != null ? costB - costA : null;
      const durDelta = durA != null && durB != null ? durB - durA : null;

      deltaRow = { label: `Δ (${b.harness}−${a.harness})`, srDelta, costDelta, durDelta };
    }

    return (
      <React.Fragment key={familyKey}>
        <tr>
          <td
            colSpan={6}
            style={{
              padding: '10px 12px',
              fontWeight: 700,
              background: 'var(--ifm-color-primary-lightest, rgba(99,102,241,0.06))',
              borderTop: '2px solid var(--ifm-color-emphasis-300)',
              fontSize: '0.9rem',
            }}
          >
            {familyKey}
          </td>
        </tr>
        {sorted.map(({ key, harness, stats }) => {
          const pt = stats.provider_type;
          const ts = stats.timeout_scale;
          const agentS = stats.agentStats || {};
          return (
            <tr key={key} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
              <td style={{ ...tdStyle, textAlign: 'left', paddingLeft: '24px' }}>
                {harness}
                {' '}
                <LocalCloudBadge providerType={pt} timeoutScale={ts} />
              </td>
              <td style={tdStyle}>{pct(agentS.successRate)}</td>
              <td style={tdStyle}>{agentS.runs ?? '—'}</td>
              <td style={tdStyle}>{agentS.avgTurns != null ? agentS.avgTurns.toFixed(1) : '—'}</td>
              <td style={tdStyle}>{cost(agentS.avgCost)}</td>
              <td style={tdStyle}>{dur(agentS.avgDurationMs)}</td>
            </tr>
          );
        })}
        {deltaRow && (
          <tr style={deltaRowStyle}>
            <td style={{ ...tdStyle, textAlign: 'left', paddingLeft: '24px', color: 'var(--ifm-color-emphasis-600)' }}>
              {deltaRow.label}
            </td>
            <td style={{ ...tdStyle, ...deltaStyle(deltaRow.srDelta) }}>
              {deltaRow.srDelta != null ? `${deltaRow.srDelta > 0 ? '+' : ''}${Math.round(deltaRow.srDelta * 100)}%` : '—'}
            </td>
            <td style={tdStyle}>—</td>
            <td style={tdStyle}>—</td>
            <td style={{ ...tdStyle, ...deltaStyle(deltaRow.costDelta != null ? -deltaRow.costDelta : null) }}>
              {deltaRow.costDelta != null ? `${deltaRow.costDelta > 0 ? '+' : ''}$${deltaRow.costDelta.toFixed(4)}` : '—'}
            </td>
            <td style={{ ...tdStyle, ...deltaStyle(deltaRow.durDelta != null ? -deltaRow.durDelta : null) }}>
              {deltaRow.durDelta != null ? `${deltaRow.durDelta > 0 ? '+' : ''}${dur(Math.abs(deltaRow.durDelta))}` : '—'}
            </td>
          </tr>
        )}
      </React.Fragment>
    );
  }

  function renderHarnessRow({ key, harness, stats }) {
    const pt = stats.provider_type;
    const ts = stats.timeout_scale;
    const agentS = stats.agentStats || {};
    return (
      <tr key={key} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
        <td style={{ ...tdStyle, textAlign: 'left' }}>
          <strong>{key}</strong> ({harness})
          {' '}
          <LocalCloudBadge providerType={pt} timeoutScale={ts} />
        </td>
        <td style={tdStyle}>{pct(agentS.successRate)}</td>
        <td style={tdStyle}>{agentS.runs ?? '—'}</td>
        <td style={tdStyle}>{agentS.avgTurns != null ? agentS.avgTurns.toFixed(1) : '—'}</td>
        <td style={tdStyle}>{cost(agentS.avgCost)}</td>
        <td style={tdStyle}>{dur(agentS.avgDurationMs)}</td>
      </tr>
    );
  }

  const hasFamilies = Object.keys(families).length > 0;

  return (
    <div>
      <p style={{ fontSize: '0.8rem', color: 'var(--ifm-color-emphasis-600)', marginBottom: '8px' }}>
        Models grouped by <code>model_family</code>. Δ row shows second harness minus first (green = improvement).
        Local (Ollama) models show timeout-scale badge.
      </p>
      <div style={{ overflowX: 'auto' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          <thead>
            <tr>
              <th style={{ ...thStyle, textAlign: 'left' }}>Model / Harness</th>
              <th style={thStyle}>Pass Rate</th>
              <th style={thStyle}>Runs</th>
              <th style={thStyle}>Avg Turns</th>
              <th style={thStyle}>Avg Cost</th>
              <th style={thStyle}>Avg Duration</th>
            </tr>
          </thead>
          <tbody>
            {hasFamilies
              ? Object.entries(families)
                  .sort(([a], [b]) => a.localeCompare(b))
                  .map(([fk, fv]) => renderFamilyGroup(fk, fv))
              : ungrouped.map(renderHarnessRow)}
            {hasFamilies && ungrouped.length > 0 && (
              <>
                <tr>
                  <td
                    colSpan={6}
                    style={{
                      padding: '10px 12px',
                      fontWeight: 700,
                      background: 'var(--ifm-color-emphasis-100)',
                      borderTop: '2px solid var(--ifm-color-emphasis-300)',
                    }}
                  >
                    Other (no model_family)
                  </td>
                </tr>
                {ungrouped.map(renderHarnessRow)}
              </>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
