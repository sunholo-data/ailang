import React, { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';

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

// On-device GPU agents: a local Qwen/Gemma run through an agentic harness
// (motoko/opencode/pi), $0/run. The "-or-" models are cloud-via-OpenRouter, NOT local.
function isLocalAgent(id) {
  return /qwen3|gemma4/i.test(id || '') && !String(id).includes('-or-');
}

function modelShort(key) {
  // Local GPU agents get a clean "<model> · <harness>" label instead of the raw
  // "motoko-local-qwen3-6-35b-a3b-mxfp8" string.
  if (isLocalAgent(key)) {
    let harness = 'agent';
    if (key.startsWith('motoko-')) harness = 'motoko';
    else if (key.startsWith('opencode-')) harness = 'opencode';
    else if (key.startsWith('pi-')) harness = 'Pi';
    const qm = /qwen3-(\d+)/.exec(key);
    const gm = /gemma4-(\d+)/.exec(key);
    const model = qm ? `Qwen3.${qm[1]}` : gm ? `Gemma4.${gm[1]}` : 'local';
    return `${model} · ${harness}`;
  }
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
    benchmarkFetch('latest.json')
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
  const allModels = block.models || [];
  const benches = block.benchmarks || [];
  const sat = block.saturation || {};

  // Coverage awareness: ELO across models that ran DIFFERENT benchmark counts
  // isn't strictly comparable, so we keep every model VISIBLE but mark those with
  // partial coverage as "provisional" (dimmed + a coverage badge) so a 6-benchmark
  // ELO can't be misread as beating a 55-benchmark one. Full ranking is earned
  // once coverage catches up. (M-EVAL-VALIDITY-DISCIPLINE)
  const maxCov = block.maxCoverage || Math.max(1, ...allModels.map((m) => m.benchmarks || 0));
  // 90%: ELO is only comparable on a near-identical benchmark set. Missing runs
  // are rarely random — an API-quota death mid-run skips the alphabetical tail,
  // which is where the hardest (frontier) benchmarks live, inflating the ELO of
  // exactly the models with holes (v0.30.0: claude-sonnet-5 topped the board on
  // 44/56 coverage that excluded gauntlet_10/quine/ssa_constant_fold/...).
  const covThreshold = Math.max(1, Math.ceil(maxCov * 0.9));
  const isProvisional = (m) => (m.benchmarks || 0) < covThreshold;
  const models = allModels; // all shown; provisional ones are flagged, not hidden

  // ELO range for the leaderboard bars — over FULL-coverage models so a sparse
  // model's inflated ELO doesn't rescale everyone else's bars.
  const eloVals = models.filter((m) => !isProvisional(m)).map((m) => m.elo).filter((v) => v != null);
  const maxElo = eloVals.length ? Math.max(...eloVals) : 1;
  const minElo = eloVals.length ? Math.min(...eloVals) : 0;
  const eloPct = (v) => {
    if (maxElo === minElo) return 100;
    return Math.min(100, Math.max(6, Math.round(((v - minElo) / (maxElo - minElo)) * 100)));
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
        {activeMode === 'standard' && (ratings.agent?.models || []).some((m) => isLocalAgent(m.id)) && (
          <button
            onClick={() => setMode('agent')}
            style={{ ...smallBtn(false), border: '1px dashed #0891b2', color: '#0891b2', whiteSpace: 'nowrap' }}
            title="On-device GPU agents (local Qwen via motoko/opencode/pi, ~$0/run) run agent mode only — switch to Agent to see them ranked."
          >
            🖥️ Local GPU agents rank in Agent mode →
          </button>
        )}
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
                <col style={{ width: '34px' }} />
                <col />
                <col style={{ width: '56px' }} />
                <col style={{ width: '50px' }} />
              </colgroup>
              <thead>
                <tr style={{ textAlign: 'left', borderBottom: '2px solid var(--ifm-color-emphasis-300)' }}>
                  <th style={{ padding: '6px 8px', textAlign: 'center', verticalAlign: 'bottom' }}>#</th>
                  <th style={{ padding: '6px 10px', verticalAlign: 'bottom' }}>Model</th>
                  <th style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'bottom' }}>ELO</th>
                  <th style={{ padding: '6px 8px', textAlign: 'right', verticalAlign: 'bottom', fontWeight: 400, color: 'var(--ifm-color-emphasis-500)' }} title="benchmarks run (of the max any model ran)">cov</th>
                </tr>
              </thead>
              <tbody>
                {models.map((m, i) => {
                  const pct = eloPct(m.elo);
                  const prov = isProvisional(m);
                  const local = isLocalAgent(m.id);
                  return (
                    <tr key={m.id} style={{ borderBottom: '1px solid var(--ifm-color-emphasis-200)', opacity: prov ? 0.65 : 1, background: local ? 'rgba(8,145,178,0.08)' : undefined, boxShadow: local ? 'inset 3px 0 0 #0891b2' : undefined }}>
                      <td style={{ padding: '6px 8px', textAlign: 'center', verticalAlign: 'middle', color: 'var(--ifm-color-emphasis-500)', fontVariantNumeric: 'tabular-nums' }}>
                        {prov ? '·' : (MEDALS[i] || i + 1)}
                      </td>
                      <td style={{ padding: 0, verticalAlign: 'middle' }}>
                        <div style={{ position: 'relative', padding: '6px 10px' }}>
                          <div style={{
                            position: 'absolute', top: 3, bottom: 3, left: 0, width: `${pct}%`,
                            background: prov ? 'var(--ifm-color-emphasis-500)' : 'var(--ifm-color-primary)',
                            opacity: prov ? 0.1 : (i === 0 ? 0.24 : 0.13),
                            borderRadius: '0 4px 4px 0',
                          }} />
                          <span style={{ position: 'relative', fontWeight: (!prov && i === 0) ? 700 : 400, fontStyle: prov ? 'italic' : 'normal' }}>{modelShort(m.id)}</span>
                          {local && (
                            <span style={{ position: 'relative', marginLeft: 6 }}
                              title="On-device GPU agent: a local Qwen run through an agentic harness — slow, ~$0/run. Not directly comparable to hosted 0-shot models, shown for the free-local-option story.">
                              <Badge color="#0891b2">🖥️ local · ~$0</Badge>
                            </span>
                          )}
                        </div>
                      </td>
                      <td style={{ padding: '6px 10px', textAlign: 'right', verticalAlign: 'middle', fontVariantNumeric: 'tabular-nums', fontWeight: 700 }}>
                        {Math.round(m.elo)}
                      </td>
                      <td style={{ padding: '6px 8px', textAlign: 'right', verticalAlign: 'middle', fontVariantNumeric: 'tabular-nums', fontSize: '0.85em', color: prov ? '#b45309' : 'var(--ifm-color-emphasis-500)', fontWeight: prov ? 700 : 400 }}
                          title={prov ? `provisional — only ${m.benchmarks} of ${maxCov} benchmarks run so far; ELO not yet comparable` : `${m.benchmarks} benchmarks`}>
                        {m.benchmarks != null ? m.benchmarks : '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            {models.some(isProvisional) && (
              <p style={{ fontSize: '0.8em', color: 'var(--ifm-color-emphasis-600)', marginTop: 6 }}>
                <span style={{ color: '#b45309', fontWeight: 700 }}>Provisional</span> rows (italic, low <strong>cov</strong>) have only run a fraction of the {maxCov} benchmarks — their ELO isn't yet comparable and settles as the rotation fills coverage in.
              </p>
            )}
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
                        <a href={`/docs/benchmarks/gallery#bench-${encodeURIComponent(b.id)}`} style={{ color: 'var(--ifm-color-primary)', textDecoration: 'none' }} title="Open in the benchmark gallery">
                          <code style={{ background: 'transparent', padding: 0, fontSize: '0.92em', wordBreak: 'break-word', color: 'inherit' }}>{b.id}</code>
                        </a>
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
