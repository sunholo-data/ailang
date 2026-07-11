import React, { useState, useEffect } from 'react';

// ELO leaderboard + difficulty-banded benchmark view (M-EVAL-DASHBOARD-REDESIGN).
// Reads the per-mode `ratings` block emitted into latest.json by eval-report:
//   ratings[mode] = {
//     models:[{id,elo,band}], benchmarks:[{id,elo,band,saturated,passRate,graderFlag?}],
//     saturation:{...},
//     byLang: { ailang:{models,benchmarks,saturation}, python:{...} }   // per-language fits
//   }
// Standard and agent are different difficulty regimes (agent saturates), so the mode
// toggle switches the whole view. Within a mode, the language toggle switches between the
// combined fit, a single language, or the AILANG-vs-Python delta (the split matters because
// e.g. claude-fable-5 tops AILANG but is dragged down on the combined board by genuine
// Python safety-refusals).

const BAND_COLOR = {
  Trivial: 'var(--ifm-color-emphasis-400)',
  Easy: '#16a34a',
  Moderate: '#ca8a04',
  Hard: '#ea580c',
  'Very hard': '#dc2626',
};

function bandBg(band) {
  const c = BAND_COLOR[band] || 'var(--ifm-color-emphasis-500)';
  return band === 'Trivial' ? 'transparent' : `${c}22`;
}

function modelShort(key) {
  return key
    .replace('claude-', 'Claude ')
    .replace('gemini-', 'Gemini ')
    .replace('opencode-or-', 'OC/')
    .replace('opencode-', 'OC/')
    .replace('gpt5', 'GPT-5')
    .replace(/-/g, ' ');
}

function Badge({ children, color }) {
  return (
    <span style={{
      fontSize: '0.72em', fontWeight: 700, padding: '1px 7px', borderRadius: 10,
      background: `${color}22`, color, border: `1px solid ${color}55`, whiteSpace: 'nowrap',
    }}>{children}</span>
  );
}

const LANG_LABEL = { combined: 'Combined', ailang: 'AILANG', python: 'Python', avp: 'AILANG vs Python' };

function eloMap(block) {
  const m = {};
  (block?.models || []).forEach((x) => { m[x.id] = x.elo; });
  return m;
}

export default function EloLeaderboard() {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  const [mode, setMode] = useState('standard');
  const [lang, setLang] = useState('combined');

  useEffect(() => {
    fetch('/benchmarks/latest.json')
      .then((r) => r.json())
      .then(setData)
      .catch((e) => setError(e.message));
  }, []);

  if (error) return <p style={{ color: 'red' }}>Failed to load: {error}</p>;
  if (!data) return <p>Loading ratings…</p>;

  const ratings = data.ratings || {};
  const modes = ['standard', 'agent'].filter((m) => ratings[m]);
  if (modes.length === 0) {
    return <p style={{ color: 'var(--ifm-color-emphasis-600)' }}>No ELO ratings in this dataset yet (regenerate the dashboard with a current build).</p>;
  }
  const activeMode = modes.includes(mode) ? mode : modes[0];
  const view = ratings[activeMode] || {};
  const byLang = view.byLang || {};
  const availLangs = ['ailang', 'python'].filter((l) => byLang[l]);
  // language options: Combined, each available language, and A-vs-P when both exist
  const langOpts = ['combined', ...availLangs, ...(availLangs.length === 2 ? ['avp'] : [])];
  const activeLang = langOpts.includes(lang) ? lang : 'combined';

  // Pick the block to render from (combined view or a single-language byLang block).
  const block = activeLang === 'combined' || activeLang === 'avp'
    ? view
    : byLang[activeLang] || view;
  const models = block.models || [];
  const benches = block.benchmarks || [];
  const sat = block.saturation || {};
  const regraded = (data.grading || {}).regraded;

  // A-vs-P: merge AILANG + Python model ELOs into rows sorted by AILANG-ELO desc.
  const aMap = eloMap(byLang.ailang);
  const pMap = eloMap(byLang.python);
  const avpRows = Object.keys({ ...aMap, ...pMap })
    .map((id) => ({ id, a: aMap[id], p: pMap[id] }))
    .sort((x, y) => (y.a ?? -1) - (x.a ?? -1));

  const btn = (active) => ({
    padding: '4px 14px', cursor: 'pointer', border: '1px solid var(--ifm-color-emphasis-300)',
    borderRadius: 6, fontWeight: 600,
    background: active ? 'var(--ifm-color-primary)' : 'transparent',
    color: active ? '#fff' : 'var(--ifm-color-emphasis-800)',
  });
  const smallBtn = (active) => ({ ...btn(active), padding: '3px 11px', fontSize: '0.85em' });

  const deltaColor = (d) => (d > 0 ? '#16a34a' : d < 0 ? '#dc2626' : 'var(--ifm-color-emphasis-500)');

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 8, flexWrap: 'wrap' }}>
        {modes.map((m) => (
          <button key={m} style={btn(m === activeMode)} onClick={() => setMode(m)}>
            {m === 'standard' ? 'Standard' : 'Agent'}
          </button>
        ))}
        {regraded && <Badge color="#16a34a">regraded</Badge>}
      </div>

      {/* Language toggle (per-mode: only shows languages present in this mode's data) */}
      {langOpts.length > 1 && (
        <div style={{ display: 'flex', gap: 6, alignItems: 'center', marginBottom: 12, flexWrap: 'wrap' }}>
          <span style={{ fontSize: '0.82em', color: 'var(--ifm-color-emphasis-600)' }}>Language:</span>
          {langOpts.map((l) => (
            <button key={l} style={smallBtn(l === activeLang)} onClick={() => setLang(l)}>{LANG_LABEL[l]}</button>
          ))}
        </div>
      )}

      {sat.total != null && activeLang !== 'avp' && (
        <p style={{ fontSize: '0.85em', color: 'var(--ifm-color-emphasis-600)', marginTop: 0 }}>
          {sat.saturated}/{sat.total} saturated · {sat.discriminating} discriminating
          {sat.total > 0 && sat.saturated / sat.total >= 0.3 && (
            <strong style={{ color: '#ea580c' }}> · suite needs harder benchmarks</strong>
          )}
        </p>
      )}

      {activeLang === 'avp' ? (
        /* AILANG-vs-Python model delta — the split view */
        <div style={{ maxWidth: 560 }}>
          <h4 style={{ marginBottom: 6 }}>Model ELO — AILANG vs Python</h4>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
            <thead>
              <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                <th style={{ padding: '4px 8px' }}>Model</th>
                <th style={{ padding: '4px 8px', textAlign: 'right' }}>AILANG</th>
                <th style={{ padding: '4px 8px', textAlign: 'right' }}>Python</th>
                <th style={{ padding: '4px 8px', textAlign: 'right' }}>Δ (A−P)</th>
              </tr>
            </thead>
            <tbody>
              {avpRows.map((r) => {
                const d = r.a != null && r.p != null ? r.a - r.p : null;
                return (
                  <tr key={r.id} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
                    <td style={{ padding: '4px 8px' }}>{modelShort(r.id)}</td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>{r.a != null ? Math.round(r.a) : '—'}</td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{r.p != null ? Math.round(r.p) : '—'}</td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums', fontWeight: 700, color: d != null ? deltaColor(d) : 'var(--ifm-color-emphasis-400)' }}>
                      {d != null ? `${d > 0 ? '+' : ''}${Math.round(d)}` : '—'}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          <p style={{ fontSize: '0.78em', color: 'var(--ifm-color-emphasis-500)', marginTop: 6 }}>
            Positive Δ = the model is <strong>stronger on AILANG</strong> than Python (AILANG-native);
            negative Δ = Python-prior. Ratings are fit independently per language.
          </p>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))', gap: 20 }}>
          {/* Model capability */}
          <div>
            <h4 style={{ marginBottom: 6 }}>Model capability (ELO){activeLang !== 'combined' ? ` — ${LANG_LABEL[activeLang]}` : ''}</h4>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
              <thead>
                <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                  <th style={{ padding: '4px 8px' }}>#</th>
                  <th style={{ padding: '4px 8px' }}>Model</th>
                  <th style={{ padding: '4px 8px', textAlign: 'right' }}>ELO</th>
                </tr>
              </thead>
              <tbody>
                {models.map((m, i) => (
                  <tr key={m.id} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
                    <td style={{ padding: '4px 8px', color: 'var(--ifm-color-emphasis-500)' }}>{i + 1}</td>
                    <td style={{ padding: '4px 8px', fontWeight: i === 0 ? 700 : 400 }}>
                      {modelShort(m.id)} {i === 0 && '🥇'}
                    </td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>
                      {Math.round(m.elo)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* Benchmark difficulty */}
          <div>
            <h4 style={{ marginBottom: 6 }}>Benchmark difficulty{activeLang !== 'combined' ? ` — ${LANG_LABEL[activeLang]}` : ''}</h4>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em' }}>
              <thead>
                <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                  <th style={{ padding: '4px 8px' }}>Benchmark</th>
                  <th style={{ padding: '4px 8px' }}>Band</th>
                  <th style={{ padding: '4px 8px', textAlign: 'right' }}>ELO</th>
                  <th style={{ padding: '4px 8px', textAlign: 'right' }}>pass</th>
                </tr>
              </thead>
              <tbody>
                {benches.map((b) => (
                  <tr key={b.id} style={{
                    borderBottom: '1px solid var(--ifm-color-emphasis-200)',
                    background: bandBg(b.band), opacity: b.saturated ? 0.55 : 1,
                  }}>
                    <td style={{ padding: '4px 8px' }}>
                      {b.id}{' '}
                      {b.graderFlag && <Badge color="#a855f7">⚠ artifact</Badge>}
                    </td>
                    <td style={{ padding: '4px 8px', color: BAND_COLOR[b.band], fontWeight: 600 }}>{b.band}</td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', fontVariantNumeric: 'tabular-nums' }}>{Math.round(b.elo)}</td>
                    <td style={{ padding: '4px 8px', textAlign: 'right', color: 'var(--ifm-color-emphasis-600)' }}>
                      {b.passRate != null ? `${Math.round(b.passRate)}%` : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p style={{ fontSize: '0.78em', color: 'var(--ifm-color-emphasis-500)', marginTop: 6 }}>
              Difficulty is derived from ELO (a PASS = the model beating the benchmark). Saturated
              (Trivial) rows are dimmed — demotion candidates. <strong>⚠ artifact</strong> = the
              difficulty is a grader/benchmark artifact, not real hardness (fix pending).
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
