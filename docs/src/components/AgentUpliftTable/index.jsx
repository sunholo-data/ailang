import React, { useState, useEffect } from 'react';

// M-EVAL-VALIDITY-DISCIPLINE (W3): the like-for-like agent-uplift view.
// Reads ratings.uplift from /benchmarks/latest.json — standard→agent pass-rate
// delta computed over ONLY the benchmarks BOTH modes ran, for a MATCHING model
// identity (internal/eval_analysis/uplift.go). This is the only valid way to read
// "what does agent mode add": a model compared to ITSELF on a shared benchmark set,
// not a cheap agent model vs a frontier standard model. Runtime fetch + inline
// styles (same pattern as the working dashboards) — no build-time coupling.

function pct(v) {
  return v == null ? '—' : `${Math.round(v * 100)}%`;
}
function signedPct(v) {
  if (v == null) return '—';
  const p = Math.round(v * 100);
  return (p > 0 ? '+' : '') + p + '%';
}
function shortModel(m) {
  return (m || '')
    .replace(/^claude-/, 'Claude ')
    .replace(/^gpt5-?/, 'GPT-5 ')
    .replace(/^gemini-/, 'Gemini ')
    .replace(/^opencode-or-/, 'OC/')
    .replace(/^opencode-/, 'OC/')
    .replace(/^or-/, 'OR/')
    .replace(/-/g, ' ');
}

export default function AgentUpliftTable() {
  const [rows, setRows] = useState(null);

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then((r) => (r.ok ? r.json() : {}))
      .then((d) => setRows(Array.isArray(d && d.ratings && d.ratings.uplift) ? d.ratings.uplift : []))
      .catch(() => setRows([]));
  }, []);

  if (rows == null) return <p>Loading agent uplift…</p>;
  if (rows.length === 0) {
    return (
      <div style={{ padding: '12px 14px', border: '1px solid var(--ifm-color-emphasis-300)', borderRadius: 8, fontSize: '0.9em' }}>
        No like-for-like uplift available yet — this needs a model that ran <strong>both</strong> standard
        and agent mode on shared benchmarks (published on the next eval baseline).
      </div>
    );
  }

  const sorted = [...rows].sort((a, b) => (b.uplift ?? 0) - (a.uplift ?? 0));
  const upColor = (u) => (u > 0.001 ? '#16a34a' : u < -0.001 ? '#dc2626' : 'var(--ifm-color-emphasis-600)');
  const th = { padding: '8px 12px', textAlign: 'left' };
  const thR = { ...th, textAlign: 'right' };
  const td = { padding: '6px 12px' };
  const tdR = { ...td, textAlign: 'right' };

  return (
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
        <thead>
          <tr>
            <th style={th}>Model</th>
            <th style={th}>Lang</th>
            <th style={thR}>Standard</th>
            <th style={thR}>Agent</th>
            <th style={thR}>Uplift Δ</th>
            <th style={thR} title="benchmarks both modes ran">Shared</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((r) => (
            <tr key={`${r.model}|${r.lang}`} style={{ borderTop: '1px solid var(--ifm-color-emphasis-200)' }}>
              <td style={{ ...td, fontFamily: 'monospace', fontSize: '0.85em' }}>{shortModel(r.model)}</td>
              <td style={td}>{r.lang}</td>
              <td style={tdR}>{pct(r.standardPass)}</td>
              <td style={tdR}>{pct(r.agentPass)}</td>
              <td style={{ ...tdR, fontWeight: 700, color: upColor(r.uplift) }}>{signedPct(r.uplift)}</td>
              <td style={{ ...tdR, color: 'var(--ifm-color-emphasis-600)' }}>{r.sharedBenchmarks}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)', marginTop: 8 }}>
        Like-for-like: each model compared to <strong>itself</strong>, agent vs standard mode, over only
        the benchmarks <strong>both</strong> ran (the “Shared” column) — a cheap-model-vs-frontier-model
        “uplift” is meaningless, so the model is held constant. Negative Δ means agent mode <em>hurt</em>
        that model.
      </p>
    </div>
  );
}
