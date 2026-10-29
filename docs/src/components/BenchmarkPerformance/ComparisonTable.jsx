import React, { Fragment } from 'react';
import { scopeStats, capabilityGroup, isProvisional, undefeated, costMetric, timingMetric, modelName,
  percent, dollars, milliseconds, costLabels, timingLabels } from '../../lib/benchmarkEvidence.mjs';
import styles from './styles.module.css';

function Result({ stats }) {
  if (!stats) return <><span>—</span><small>Not measured</small></>;
  return <><strong>{percent(stats.rate)}</strong><small>{stats.passes != null ? `${stats.passes} / ${stats.runs} runs` : `${stats.runs} recorded runs`}</small>
    <div className={styles.meter} aria-hidden="true"><span style={{ width: `${stats.rate * 100}%` }} /></div></>;
}

export default function ComparisonTable({ rows, scope, lang, sort, showPython, onSelect, selected }) {
  const cols = showPython && lang === 'ailang' ? 7 : 6;
  return <div className={styles.tableWrap}><table className={styles.table}>
    <caption className={styles.srOnly}>Saved results by model. Capability ordering groups undefeated samples before numerically rated models.</caption>
    <thead><tr><th scope="col">Model</th><th scope="col">ELO / status</th><th scope="col">{lang === 'ailang' ? 'AILANG' : 'Python'} passed</th>
      {cols === 7 && <th scope="col">Python reference</th>}<th scope="col">Cost / solved task</th><th scope="col">Timing</th><th scope="col">Measured on</th></tr></thead>
    <tbody>{rows.map((row, i) => {
      const stats = scopeStats(row, scope, lang), all = scopeStats(row, 'all', lang);
      const group = capabilityGroup(row, lang), provisional = isProvisional(row, lang);
      const cost = costMetric(stats), timing = timingMetric(stats);
      const versions = Object.keys(stats?.versions || {});
      return <Fragment key={row.id}>
        {sort === 'elo' && (i === 0 || capabilityGroup(rows[i - 1], lang) !== group) && <tr className={styles.group}><th colSpan={cols}>
          {['Undefeated on measured tasks — complete Core coverage first, smaller samples provisional', 'Rated models — highest ELO first', 'Rating unavailable'][group]}</th></tr>}
        <tr className={selected === row.id ? styles.selected : undefined}>
          <td><button type="button" className={styles.modelButton} onClick={() => onSelect(row.id)}>{modelName(row.model)}</button>
            <small>{row.harness === 'direct_api' ? 'Direct API' : row.harness === 'unknown' ? 'Harness not retained' : row.harness}</small>
            {provisional && <><span className={styles.badge}>Provisional{undefeated(row, lang) ? ' · undefeated' : ''}</span>
              <small>{all?.runs || 0} runs · {all?.coreCompleted?.length ?? 'Unknown'}/{row.requiredCoreIds.length || '?'} Core tasks</small></>}
          </td>
          <td>{row.ratings[lang] != null ? <><strong>{Math.round(row.ratings[lang])}</strong><small>{lang} · all tasks</small></>
            : undefeated(row, lang) ? <><span className={styles.badge}>Undefeated</span><small>No losses to calibrate ELO</small></> : <><span>—</span><small>Not retained</small></>}</td>
          <td><Result stats={stats} /></td>
          {cols === 7 && <td className={styles.python}><Result stats={scopeStats(row, scope, 'python')} /></td>}
          <td>{dollars(cost.value)}<small>{costLabels[cost.kind] || cost.kind}</small></td>
          <td>{milliseconds(timing.value)}<small>{timingLabels[timing.kind]}</small>{timing.samples != null && <small>{timing.samples} timing samples</small>}</td>
          <td>{stats?.lastRun ? stats.lastRun.slice(0, 10) : '—'}<small>{stats?.latestVersion || 'Version not retained'}</small>
            {versions.length > 1 && <small>Spans {versions.length} versions</small>}</td>
        </tr>
      </Fragment>;
    })}</tbody></table></div>;
}
