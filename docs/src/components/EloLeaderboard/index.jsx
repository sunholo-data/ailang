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

function BandPill({ band }) {
  const c = BAND_COLOR[band] || 'var(--ifm-color-emphasis-500)';
  return (
    <span style={{
      display: 'inline-block', fontSize: '0.75em', fontWeight: 700, padding: '2px 9px',
      borderRadius: 20, whiteSpace: 'nowrap', color: c,
      background: band === 'Trivial' ? 'var(--ifm-color-emphasis-200)' : `${c}1f`,
      border: `1px solid ${c}44`,
    }}>{band}</span>
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

  // ELO range for the leaderboard bars (relative fill).
  const eloVals = models.map((m) => m.elo).filter((v) => v != null);
  const maxElo = eloVals.length ? Math.max(...eloVals) : 1;
  const minElo = eloVals.length ? Math.min(...eloVals) : 0;
  const eloPct = (v) => {
    if (maxElo === minElo) return 100;
    return Math.max(6, Math.round(((v - minElo) / (maxElo - minElo)) * 100));
  };
  const MEDALS = ['🥇', '🥈', '🥉'];

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
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em', tableLayout: 'fixed' }}>
              <colgroup>
                <col style={{ width: '36px' }} />
                <col />
                <col style={{ width: '62px' }} />
              </colgroup>
              <thead>
                <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                  <th style={{ padding: '6px 8px', textAlign: 'center', verticalAlign: 'bottom' }}>#</th>
                  <th style={{ padding: '6px 10px', verticalAlign: 'bottom' }}>Model</th>
                  <th style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'bottom' }}>ELO</th>
                </tr>
              </thead>
              <tbody>
                {models.map((m, i) => {
                  const pct = eloPct(m.elo);
                  return (
                    <tr key={m.id} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)' }}>
                      <td style={{ padding: '6px 8px', textAlign: 'center', verticalAlign: 'middle', color: 'var(--ifm-color-emphasis-500)', fontVariantNumeric: 'tabular-nums' }}>
                        {MEDALS[i] || i + 1}
                      </td>
                      <td style={{ padding: 0, verticalAlign: 'middle' }}>
                        <div style={{ position: 'relative', padding: '6px 10px' }}>
                          <div style={{
                            position: 'absolute', top: 3, bottom: 3, left: 0, width: `${pct}%`,
                            background: 'var(--ifm-color-primary)', opacity: i === 0 ? 0.24 : 0.13,
                            borderRadius: '0 4px 4px 0',
                          }} />
                          <span style={{ position: 'relative', fontWeight: i === 0 ? 700 : 400 }}>{modelShort(m.id)}</span>
                        </div>
                      </td>
                      <td style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'middle', fontVariantNumeric: 'tabular-nums', fontWeight: 700 }}>
                        {Math.round(m.elo)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          {/* Benchmark difficulty */}
          <div>
            <h4 style={{ marginBottom: 6 }}>Benchmark difficulty{activeLang !== 'combined' ? ` — ${LANG_LABEL[activeLang]}` : ''}</h4>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.9em', tableLayout: 'fixed' }}>
              <colgroup>
                <col />
                <col style={{ width: '96px' }} />
                <col style={{ width: '64px' }} />
                <col style={{ width: '56px' }} />
              </colgroup>
              <thead>
                <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                  <th style={{ padding: '6px 10px', verticalAlign: 'bottom' }}>Benchmark</th>
                  <th style={{ padding: '6px 10px', verticalAlign: 'bottom' }}>Band</th>
                  <th style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'bottom' }}>ELO</th>
                  <th style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'bottom' }}>pass</th>
                </tr>
              </thead>
              <tbody>
                {benches.map((b) => (
                  <tr key={b.id} style={{
                    borderBottom: '1px solid var(--ifm-color-emphasis-200)',
                    background: bandBg(b.band), opacity: b.saturated ? 0.6 : 1,
                  }}>
                    <td style={{ padding: '6px 10px', verticalAlign: 'middle' }}>
                      <span style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                        <code style={{ background: 'transparent', padding: 0, fontSize: '0.92em', wordBreak: 'break-word' }}>{b.id}</code>
                        {b.graderFlag && <Badge color="#a855f7">⚠ artifact</Badge>}
                      </span>
                    </td>
                    <td style={{ padding: '6px 10px', verticalAlign: 'middle' }}><BandPill band={b.band} /></td>
                    <td style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'middle', fontVariantNumeric: 'tabular-nums', fontWeight: 600 }}>{Math.round(b.elo)}</td>
                    <td style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'middle', color: 'var(--ifm-color-emphasis-600)', fontVariantNumeric: 'tabular-nums' }}>
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
