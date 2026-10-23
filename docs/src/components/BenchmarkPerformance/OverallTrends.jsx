import React, { useMemo, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import { historyPoints, eloSeries } from '../../lib/benchmarkHistory.mjs';
import { modelName } from '../../lib/benchmarkEvidence.mjs';
import styles from './styles.module.css';

const COLORS = ['var(--sunholo-blue)', 'var(--ifm-color-primary)', 'var(--sunholo-teal)', 'var(--sunholo-purple)', 'var(--ifm-color-emphasis-700)', 'var(--sunholo-orange)'];
const DASHES = ['', '6 3', '2 4', '8 3 2 3', '4 4', '10 3'];
const value = n => n == null ? 'Not retained' : `${n.toFixed(1)}%`;

export default function OverallTrends({ latest }) {
  const [scope, setScope] = useState('core'), [range, setRange] = useState('all'), [index, setIndex] = useState(-1);
  const available = useMemo(() => Array.from(new Set((latest.history || []).flatMap(h => Object.keys(h.ratings?.models || {})))).sort(), [latest]);
  const [chosen, setChosen] = useState(() => {
    const defaults = ['gemini-3-1-pro', 'gpt5-6-sol', 'or-glm-5-2', 'claude-opus-5-5'].filter(m => available.includes(m));
    return defaults.length ? defaults : available.slice(0, 4);
  });
  const models = chosen;
  const points = useMemo(() => historyPoints(latest, scope, range), [latest, scope, range]);
  const elo = useMemo(() => eloSeries(points, models), [points, models.join('|')]);
  if (!points.length) return <p className={styles.notice}>No saved release history in this snapshot.</p>;
  const selectedIndex = index < 0 ? points.length - 1 : Math.min(index, points.length - 1);
  const selected = points[selectedIndex];
  function inspect(e) {
    if (e?.activeTooltipIndex != null) {
      const next = Number(e.activeTooltipIndex);
      if (Number.isInteger(next) && next >= 0 && next < points.length) setIndex(next);
    }
  }
  const axes = <><CartesianGrid stroke="var(--ifm-color-emphasis-300)" vertical={false} /><XAxis dataKey="version" minTickGap={36} tick={{ fontSize: 11, fill: 'var(--ifm-color-emphasis-700)' }} />
    <Tooltip /><ReferenceLine x={selected.version} stroke="var(--ifm-color-emphasis-600)" strokeDasharray="3 4" /></>;
  return <section id="overall-trends" className={styles.trends}>
    <div className={styles.trendHead}><div><h2>Overall trends</h2><p>Saved direct-model evidence across release snapshots</p></div>
      <div className={styles.trendControls}><label>Pass-rate tasks<select value={scope} onChange={e => setScope(e.target.value)}>{['core','all','stretch','frontier'].map(t => <option key={t} value={t}>{t === 'all' ? 'All measured tasks' : t}</option>)}</select></label>
        <label>History<select value={range} onChange={e => { setRange(e.target.value); setIndex(-1); }}><option value="all">Full history</option><option value="recent">Recent 20 snapshots</option></select></label></div></div>
    <div className={styles.trendGrid}>
      <div><h3>AILANG and Python pass rates</h3><p className={styles.note}>Share of saved runs that passed</p>
        <div role="img" aria-label="Historical AILANG and Python pass rates. Use the snapshot inspector below for exact values.">
          <ResponsiveContainer width="100%" height={240}><LineChart data={points} onMouseMove={inspect} margin={{ top: 8, right: 12, bottom: 8, left: -18 }}>
            {axes}<YAxis domain={[0,100]} tickFormatter={n => `${n}%`} tick={{ fontSize: 11, fill: 'var(--ifm-color-emphasis-700)' }} />
            <Line dataKey="ailang" name="AILANG" stroke="var(--ifm-color-primary)" dot={false} strokeWidth={2} connectNulls={false} isAnimationActive={false} />
            <Line dataKey="python" name="Python" stroke="var(--sunholo-teal)" strokeDasharray="6 4" dot={false} strokeWidth={2} connectNulls={false} isAnimationActive={false} />
          </LineChart></ResponsiveContainer></div>
        <div className={styles.legend}><span>━ AILANG</span><span>┄ Python</span></div>
      </div>
      <div><h3>Model ELO over releases</h3><p className={styles.note}>All tasks · model identities and anchors stay separate</p>
        <div role="img" aria-label="Historical model ELO ratings. Missing observations and anchor changes break each line.">
          <ResponsiveContainer width="100%" height={240}><LineChart data={elo.data} onMouseMove={inspect} margin={{ top: 8, right: 12, bottom: 8, left: -10 }}>
            {axes}<YAxis domain={['auto','auto']} tick={{ fontSize: 11, fill: 'var(--ifm-color-emphasis-700)' }} />
            {elo.series.map(s => { const i = models.indexOf(s.model); return <Line key={s.key} dataKey={s.key} name={modelName(s.model)} stroke={COLORS[i % COLORS.length]} strokeDasharray={DASHES[i % DASHES.length]} dot={false} strokeWidth={2} connectNulls={false} isAnimationActive={false} />; })}
          </LineChart></ResponsiveContainer></div>
        <div className={styles.legend}>{models.map((m,i) => <span key={m}><i style={{ borderColor: COLORS[i % COLORS.length], borderTopStyle: i % 2 ? 'dashed' : 'solid' }} />{modelName(m)}</span>)}</div>
      </div>
    </div>
    <div className={styles.inspector}><label htmlFor="snapshot-inspector">Inspect a saved release snapshot</label>
      <input id="snapshot-inspector" type="range" min={0} max={points.length - 1} value={selectedIndex} onChange={e => setIndex(Number(e.target.value))} />
      <div aria-live="polite"><strong>{selected.version}</strong><span>Published {selected.published?.slice(0,10) || 'date unavailable'}</span>
        <span>AILANG <b>{value(selected.ailang)}</b> ({selected.ailangRuns ?? 'unknown'} runs)</span><span>Python <b>{value(selected.python)}</b> ({selected.pythonRuns ?? 'unknown'} runs)</span><span>{selected.modelCount} model records</span><span>{selected.taskCount ?? 'Unknown'} tasks in scope</span>
        <div className={styles.eloValues}>{models.map(m => <span key={m}>{modelName(m)}: {Number.isFinite(selected.elo[m]) ? Math.round(selected.elo[m]) : 'Not in snapshot'}</span>)}</div>
        {selected.events.map((e,i) => <span key={i}>Suite event: {e.label}</span>)}
      </div>
    </div>
    <p className={styles.note}>Historical model and task mixes vary. A snapshot can reuse older runs; it does not mean every model was tested for that release. The pass-rate task filter does not change the all-task ELO fit.</p>
    <details className={styles.definitions}><summary>Recorded changes to the benchmark suite</summary>
      {latest.events?.length ? <ul>{latest.events.map((e,i) => <li key={i}><strong>{e.version}</strong> — {e.label}{e.affects_tiers?.length ? ` (${e.affects_tiers.join(', ')})` : ''}</li>)}</ul>
        : <p>No suite events retained in this snapshot.</p>}</details>
    <details className={styles.modelChoices}><summary>Choose ELO models (up to six)</summary><div>{available.map(m => <label key={m}><input type="checkbox" checked={models.includes(m)} onChange={() => setChosen(prev => {
      const current = prev;
      return current.includes(m) ? current.filter(x => x !== m) : current.length < 6 ? [...current, m] : current;
    })} />{modelName(m)}</label>)}</div></details>
    <details className={styles.definitions}><summary>Fixed-panel language-direction evidence</summary>
      {points.some(p => p.direction != null) ? <div className={styles.tableWrap}><table className={styles.table}><thead><tr><th>Snapshot</th><th>Direction index (lower is easier)</th><th>Panel / anchor</th><th>Trials</th></tr></thead>
        <tbody>{points.filter(p => p.direction != null).map(p => <tr key={p.version}><td>{p.version}</td><td>{p.direction}</td><td>{p.panel || 'Not retained'} / {p.anchor}</td><td>{p.directionTrials ?? 'Not retained'}</td></tr>)}</tbody></table></div>
        : <p>No fixed-panel direction index was retained in these snapshots. The historical mix above is not a controlled measure of language improvement alone.</p>}</details>
  </section>;
}
