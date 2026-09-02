import React, { useState, useEffect } from 'react';
import { ArrowUpDown, TrendingUp, TrendingDown } from 'lucide-react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import { localRate, formatLocalName, LOCAL_CAVEAT } from '@site/src/lib/localModel';
import styles from './styles.module.css';

// Short label for an on-device model string, e.g.
// "motoko-local-qwen3-6-35b-a3b-mxfp8" -> "Qwen3.6". Falls back to the raw model.
function shortLocalModel(model) {
  const m = /qwen3-(\d+)/.exec(model || '');
  return m ? `Qwen3.${m[1]}` : (model || 'local');
}

// RefusalNote annotates a success cell with the share of runs that were safety
// refusals (model declined). Refusals are still counted as non-passes in the rate
// ("keep counting") — this badge just contextualizes a number depressed by declines
// (e.g. Fable's Python) so it doesn't read as "can't code".
function RefusalNote({ rate }) {
  if (!rate || rate <= 0) return null;
  const pct = (rate * 100).toFixed(0);
  return (
    <span
      title={`${pct}% of these runs were safety refusals — the model declined the prompt. Counted as non-passes in the rate above; shown here so a decline-driven number isn't misread as a coding failure.`}
      style={{
        marginLeft: 5, fontSize: '0.7em', whiteSpace: 'nowrap', verticalAlign: 'middle',
        cursor: 'help', padding: '0 3px', borderRadius: 3,
        color: 'var(--ifm-color-warning-darkest)',
        background: 'var(--ifm-color-warning-lightest)',
        border: '1px solid var(--ifm-color-warning-light)',
      }}
    >
      ⚠{pct}% refused
    </span>
  );
}

// RotationRateNote marks a Local GPU agent rate that came from the rotation
// (os/latest.json, trial-normalised) instead of the release baseline
// (latest.json, runs-normalised) — which happens whenever latest.json has not
// been refreshed since the rotation moved to a newer AILANG version. The two
// denominators differ, so the number must not read as directly comparable to the
// cloud rows beside it. Visible on purpose: the alternative was the row silently
// disappearing, which is what it did for two release cycles.
function RotationRateNote({ baselineVersion, rotationVersion }) {
  return (
    <span
      title={
        'Rate from the continuous rig rotation (' + (rotationVersion || 'current') +
        '), divided by TRIALS. The cloud rows come from the release baseline' +
        (baselineVersion ? ' (' + baselineVersion + ')' : '') +
        ', divided by RUNS. Same measurement, different denominator — so read this ' +
        'as the on-device option existing and roughly where it lands, not as a ' +
        'like-for-like number. It resolves to the canonical rate once the release ' +
        'baseline is refreshed.'
      }
      style={{
        marginLeft: 5, fontSize: '0.7em', whiteSpace: 'nowrap', verticalAlign: 'middle',
        cursor: 'help', padding: '0 3px', borderRadius: 3,
        color: 'var(--ifm-color-warning-darkest)',
        background: 'var(--ifm-color-warning-lightest)',
        border: '1px solid var(--ifm-color-warning-light)',
      }}
    >
      ~rotation
    </span>
  );
}

function formatModelName(name) {
  // Surface harness + provider as explicit suffixes. See
  // BenchmarkExplorer/index.jsx::modelShort for the canonical version.
  let s = name;
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

export default function ModelComparisonTable({ models, coverage, showLocalAgent = false }) {
  const [sortColumn, setSortColumn] = useState('ailangSuccess');
  const [sortDirection, setSortDirection] = useState('desc');

  // Optional on-device "Local GPU agent" row (M-EVAL): the rig's best agentic
  // config (opencode/pi/motoko on a local qwen) fetched from os/latest.json. It's
  // agent-mode + ~$0/run — a slow, free option — so we surface only its AILANG
  // success against the 0-shot cloud rows (the thesis), not a full head-to-head.
  const [localRow, setLocalRow] = useState(null);
  useEffect(() => {
    if (!showLocalAgent) return;
    let alive = true;
    // The RATE must come from latest.json (runs-based), the same accumulator the
    // cloud rows in this table use. os/latest.json divides by trials and publishes
    // on its own cadence, so sourcing the rate there put a drifting, differently-
    // normalised number in a column of cloud rates. os/* is still the source for
    // which harnesses exist — latest.json doesn't carry that.
    Promise.all([
      benchmarkFetch('os/latest.json').then((r) => (r.ok ? r.json() : null)).catch(() => null),
      benchmarkFetch('latest.json').then((r) => (r.ok ? r.json() : null)).catch(() => null),
    ])
      .then(([os, main]) => {
        if (!alive || !os || !Array.isArray(os.rows) || !main) return;
        let best = null;
        for (const row of os.rows) {
          const canonical = localRate(main, row.model, 'ailang');
          if (!canonical) continue;
          if (!best || canonical.rate > best.canonical.rate) best = { canonical, harness: row.harness, model: row.model };
        }
        // FALLBACK to the rotation's own rate when latest.json carries no agent
        // entry for ANY on-device model.
        //
        // WHY THIS EXISTS. latest.json is the RELEASE baseline: it is refreshed by
        // post-release, while the rig's rotation rolls forward continuously. So
        // between releases latest.json can sit versions behind the rotation and
        // contain no local models at all — measured 2026-09-02, latest.json was
        // v0.32.0 (2026-08-27) while the rotation was banking v0.34.0, and
        // localRate() returned null for all three on-device models. `best` stayed
        // null, setLocalRow was never called, and the row this component is
        // explicitly asked to show (showLocalAgent) just VANISHED — for two whole
        // release cycles, with nothing said anywhere.
        //
        // Omitting rather than inventing a number was the right instinct; making the
        // omission SILENT was the bug (see "no silent fallbacks" in CLAUDE.md). The
        // rotation rate is a real measurement — it is simply normalised differently
        // (os/* divides by trials, latest.json by runs), so it is shown with a
        // visible badge saying exactly that, never passed off as a cloud-comparable
        // number.
        let normalisation = 'canonical';
        if (!best) {
          for (const row of os.rows) {
            const rate = row && row.lang && row.lang.ailang;
            if (typeof rate !== 'number') continue;
            if (!best || rate > best.canonical.rate) {
              best = { canonical: { rate, runs: null }, harness: row.harness, model: row.model };
            }
          }
          if (best) {
            normalisation = 'rotation';
            // Loud in the console: the row is degraded, and the remedy is a
            // post-release refresh of latest.json, not a frontend change.
            console.warn(
              '[ModelComparisonTable] latest.json (version ' + (main.version || '?') +
              ') has no agentModels entry for any on-device model; the Local GPU agent row ' +
              'is falling back to the os/latest.json rotation rate (trial-normalised). ' +
              'Refresh the release baseline via post-release to restore the canonical rate.'
            );
          }
        }
        if (best) {
          setLocalRow({
            ailangSuccess: best.canonical.rate * 100,
            runs: best.canonical.runs,
            normalisation,
            baselineVersion: main.version || '',
            rotationVersion: os.ailang_version || os.version || '',
            harness: best.harness || 'agent',
            modelId: best.model,
            model: formatLocalName(best.model),
            configCount: os.rows.length,
            version: os.ailang_version || os.version || '',
            benchmarks: coverage ? coverage.benchmarksFor(best.model) : null,
            provisional: coverage ? coverage.isProvisional(best.model) : false,
          });
        }
      })
      .catch(() => { /* os data optional — omit the row on failure */ });
    return () => { alive = false; };
    // `coverage` is a dep: it arrives asynchronously with the parent's fetch, and
    // without it the row would keep whatever gating state it had when coverage was
    // still null — i.e. render ungated, which is the bug this fix exists for.
  }, [showLocalAgent, coverage]);

  // Transform models data into table rows
  const tableData = Object.entries(models)
    .filter(([name, stats]) => {
      return stats.languages && stats.languages.ailang && stats.languages.python;
    })
    .map(([name, stats]) => {
      const ailang = stats.languages?.ailang;
      const python = stats.languages?.python;
      const ailangSuccess = (ailang?.successRate || 0) * 100;
      const pythonSuccess = (python?.successRate || 0) * 100;
      const ailangTokens = ailang?.avgTokens || 0;
      const pythonTokens = python?.avgTokens || 1; // Avoid div by zero
      const gap = ailangSuccess - pythonSuccess;

      return {
        modelName: name,
        displayName: formatModelName(name),
        ailangSuccess: ailangSuccess,
        ailangRuns: ailang?.totalRuns || 0,
        ailangTokens: Math.round(ailangTokens),
        pythonSuccess: pythonSuccess,
        pythonRuns: python?.totalRuns || 0,
        pythonTokens: Math.round(python?.avgTokens || 0),
        gap: gap,
        tokenRatio: ailangTokens / pythonTokens,
        ailangRefusalRate: ailang?.refusalRate || 0,
        pythonRefusalRate: python?.refusalRate || 0,
      };
    });

  // Sample-size sanity check: lang_harness_suite models (claude-haiku-4-5,
  // opencode-haiku) only ran the core tier (~23 benchmarks per language)
  // while agent_suite/extended_suite models ran core+stretch (~34 per
  // language). Same percentages computed on different benchmark sets are not
  // directly comparable — flag the partial samples so readers don't conclude
  // "haiku is the best model" when it actually skipped the hardest 11 tasks.
  const fullRuns = Math.max(0, ...tableData.map(r => Math.max(r.ailangRuns, r.pythonRuns)));
  for (const row of tableData) {
    const minRuns = Math.min(row.ailangRuns || 0, row.pythonRuns || 0);
    row.isPartialSample = fullRuns > 0 && minRuns > 0 && minRuns < fullRuns;
    row.runs = minRuns;
    // M-EVAL-VALIDITY-DISCIPLINE (W2): prefer TRUE benchmark coverage (distinct
    // benchmarks from the ratings block) over the run-count heuristic when present.
    row.benchmarks = coverage ? coverage.benchmarksFor(row.modelName) : null;
    row.provisional = coverage ? coverage.isProvisional(row.modelName) : row.isPartialSample;
  }
  const maxCoverage = coverage ? coverage.maxCoverage : fullRuns;

  // Sort table data
  const sortedData = [...tableData].sort((a, b) => {
    let aVal = a[sortColumn];
    let bVal = b[sortColumn];

    if (sortDirection === 'asc') {
      return aVal > bVal ? 1 : -1;
    } else {
      return aVal < bVal ? 1 : -1;
    }
  });

  const handleSort = (column) => {
    if (sortColumn === column) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortColumn(column);
      setSortDirection('desc');
    }
  };

  const SortIcon = ({ column }) => {
    if (sortColumn !== column) {
      return <ArrowUpDown size={14} className={styles.sortIconInactive} />;
    }
    return sortDirection === 'asc' ?
      <TrendingUp size={14} className={styles.sortIconActive} /> :
      <TrendingDown size={14} className={styles.sortIconActive} />;
  };

  return (
    <div className={styles.tableContainer}>
      <table className={styles.comparisonTable}>
        <thead>
          <tr>
            <th className={styles.tableHeaderSticky}>Model</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>AILANG</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>Python</th>
            <th colSpan="2" className={styles.tableHeaderGroup}>Comparison</th>
          </tr>
          <tr>
            <th className={styles.tableHeaderSticky}></th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('ailangSuccess')}>
              % <SortIcon column="ailangSuccess" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('ailangTokens')}>
              Tok <SortIcon column="ailangTokens" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('pythonSuccess')}>
              % <SortIcon column="pythonSuccess" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('pythonTokens')}>
              Tok <SortIcon column="pythonTokens" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('gap')}>
              Gap <SortIcon column="gap" />
            </th>
            <th className={styles.tableHeaderClickable} onClick={() => handleSort('tokenRatio')}>
              Ratio <SortIcon column="tokenRatio" />
            </th>
          </tr>
        </thead>
        <tbody>
          {localRow && (
            <tr style={{
              background: 'var(--ifm-color-info-contrast-background, rgba(8,145,178,0.10))',
              borderLeft: '4px solid #0891b2',
              // Same provisional treatment the cloud rows get below — this row used to
              // render undimmed at full confidence while a better-covered cloud model
              // beside it was dimmed.
              ...(localRow.provisional ? { opacity: 0.6, fontStyle: 'italic' } : null),
            }}>
              <td className={styles.tableModelName}>
                <span
                  title={`Best of ${localRow.configCount} on-device agent configs (${localRow.harness} · ${localRow.model}${localRow.version ? ', ' + localRow.version : ''}). ${LOCAL_CAVEAT}`}
                  style={{ fontWeight: 700, cursor: 'help' }}
                >
                  🖥️ Local GPU agent
                </span>
                <span style={{ marginLeft: 6, fontSize: '0.7em', color: 'var(--ifm-color-emphasis-600)' }}>
                  best of {localRow.configCount} · {localRow.harness} · {localRow.model} · agent · ~$0
                </span>
              </td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: localRow.ailangSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    localRow.ailangSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}
                  title={localRow.provisional && localRow.benchmarks != null
                    ? `Provisional: ran ${localRow.benchmarks}/${maxCoverage} benchmarks so far${localRow.runs != null ? ` (${localRow.runs} runs)` : ''} — not comparable to full-coverage rows.`
                    : (localRow.runs != null ? `${localRow.runs} runs` : 'Trial-normalised rate from the rig rotation')}>
                  {localRow.ailangSuccess.toFixed(1)}
                  {localRow.provisional && <span style={{ marginLeft: 3 }}>⚠</span>}
                </span>
                {localRow.normalisation === 'rotation' && (
                  <RotationRateNote
                    baselineVersion={localRow.baselineVersion}
                    rotationVersion={localRow.rotationVersion}
                  />
                )}
              </td>
              <td className={styles.tableNumber} title="No token data for on-device runs">—</td>
              <td className={styles.tableNumber} title="Local agent runs AILANG only">—</td>
              <td className={styles.tableNumber}>—</td>
              <td className={styles.tableNumber}>—</td>
              <td className={styles.tableNumber}>—</td>
            </tr>
          )}
          {sortedData.map((row) => (
            <tr key={row.modelName} style={row.provisional ? { opacity: 0.6, fontStyle: 'italic' } : undefined}>
              <td className={styles.tableModelName}>
                {row.displayName}
                {row.provisional && (
                  <span
                    title={row.benchmarks != null
                      ? `Provisional: ran ${row.benchmarks}/${maxCoverage} benchmarks so far — pass-rate not comparable to full-coverage models until the rotation fills coverage in.`
                      : `Partial sample: ran ${row.runs}/${fullRuns} benchmarks (core tier only). Pass-rate not directly comparable to models that ran the full core+stretch suite.`}
                    style={{
                      marginLeft: 6,
                      fontSize: '0.7em',
                      padding: '1px 5px',
                      borderRadius: 3,
                      background: 'var(--ifm-color-warning-lightest)',
                      color: 'var(--ifm-color-warning-darkest)',
                      border: '1px solid var(--ifm-color-warning-light)',
                      verticalAlign: 'middle',
                      cursor: 'help',
                      fontWeight: 600,
                    }}
                  >
                    ⚠ {row.benchmarks != null ? `${row.benchmarks}/${maxCoverage}` : `${row.runs}/${fullRuns}`}
                  </span>
                )}
              </td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: row.ailangSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    row.ailangSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}>
                  {row.ailangSuccess.toFixed(1)}
                </span>
                <RefusalNote rate={row.ailangRefusalRate} />
              </td>
              <td className={styles.tableNumber}>{row.ailangTokens}</td>
              <td className={styles.tableNumber}>
                <span className={styles.successBadge} style={{
                  backgroundColor: row.pythonSuccess >= 70 ? 'var(--ifm-color-success)' :
                                    row.pythonSuccess >= 50 ? 'var(--ifm-color-warning)' :
                                    'var(--ifm-color-danger)'
                }}>
                  {row.pythonSuccess.toFixed(1)}
                </span>
                <RefusalNote rate={row.pythonRefusalRate} />
              </td>
              <td className={styles.tableNumber}>{row.pythonTokens}</td>
              <td className={styles.tableNumber}>
                <span className={row.gap >= 0 ? styles.gapPositive : styles.gapNegative}>
                  {row.gap >= 0 ? '+' : ''}{row.gap.toFixed(1)}
                </span>
              </td>
              <td className={styles.tableNumber}>
                <span className={row.tokenRatio > 1 ? styles.ratioHigher : styles.ratioLower}>
                  {row.tokenRatio.toFixed(2)}x
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className={styles.tableFootnote}>
        💡 <strong>Gap</strong> = AILANG - Python success % (positive = better) · <strong>Ratio</strong> = AILANG/Python tokens (lower = more efficient) · <strong>Tok</strong> = avg output tokens
        {localRow && (
          <>
            {' · '}<strong>🖥️ Local GPU agent</strong> = best on-device config (qwen via an agentic harness), agent-mode + ~$0/run. Shown for the free-local-option thesis — <em>agent-mode, so not directly comparable to the 0-shot cloud rows</em>.
            {localRow.normalisation === 'rotation' && (
              <> Its rate is currently <strong>~rotation</strong>: taken from the live rig rotation{localRow.rotationVersion ? ` (${localRow.rotationVersion})` : ''} and divided by trials, because the release baseline{localRow.baselineVersion ? ` (${localRow.baselineVersion})` : ''} has not been refreshed since the rotation moved on.</>
            )}
          </>
        )}
      </div>
    </div>
  );
}
