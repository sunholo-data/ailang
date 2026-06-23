import React, { useState } from 'react';
import { CheckCircle, XCircle, AlertCircle, ChevronDown, ChevronUp } from 'lucide-react';
import styles from './styles.module.css';

const TIER_ORDER = ['core', 'stretch', 'vision', 'smoke'];
const TIER_LABELS = { core: 'Core', stretch: 'Stretch', vision: 'Vision', smoke: 'Smoke' };
// `gemini` rows are historical (Gemini CLI executor retired v0.22.0); do not relabel them.
const HARNESS_LABEL = { claude: 'Claude CLI', codex: 'Codex', gemini: 'Gemini CLI (retired)', managed_agents: 'Managed Agents', opencode: 'opencode', pi: 'Pi' };
const HARNESS_ORDER = ['claude', 'codex', 'gemini', 'managed_agents', 'opencode', 'pi'];
const LANG_LABEL = { ailang: 'AILANG', python: 'Python', javascript: 'JavaScript', go: 'Go' };
const LANG_ORDER = ['ailang', 'python', 'javascript', 'go'];

// Adjusted success rate: excludes API errors from denominator.
// API errors are infrastructure failures (quota, CLI version mismatch) that
// don't reflect model code-generation quality. Adj rate = passes / non-api-runs.
function adjRate(rate, apiErrorRate) {
  if (rate == null) return null;
  if (!apiErrorRate || apiErrorRate <= 0) return rate;
  const adj = rate / (1 - apiErrorRate);
  return Math.min(1.0, adj);
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

export default function BenchmarkGallery({ benchmarks }) {
  const allArray = Object.entries(benchmarks).map(([id, stats]) => ({ id, ...stats }));

  // Detect which tiers are present
  const tierCounts = {};
  for (const b of allArray) {
    const t = b.tier || 'core';
    tierCounts[t] = (tierCounts[t] || 0) + 1;
  }
  const tiersPresent = TIER_ORDER.filter(t => tierCounts[t] > 0);
  const showTierFilter = tiersPresent.length > 0;

  const defaultTier = tiersPresent.includes('core') ? 'core' : (tiersPresent[0] || null);
  const [localTier, setLocalTier] = useState(defaultTier);

  const filtered = localTier
    ? allArray.filter(b => (b.tier || 'core') === localTier)
    : allArray;
  const sorted = [...filtered].sort((a, b) => b.successRate - a.successRate);

  return (
    <div>
      {showTierFilter && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16, alignItems: 'center' }}>
          <span style={{ fontWeight: 600, fontSize: '0.85rem', color: 'var(--ifm-color-emphasis-700)' }}>Tier:</span>
          {[null, ...tiersPresent].map(t => {
            const label = t ? `${TIER_LABELS[t]} (${tierCounts[t]})` : `All (${allArray.length})`;
            const active = localTier === t;
            return (
              <button
                key={t ?? 'all'}
                onClick={() => setLocalTier(t)}
                style={{
                  padding: '3px 12px', borderRadius: 14, cursor: 'pointer', fontSize: '0.8rem',
                  fontWeight: active ? 600 : 400,
                  border: active ? '2px solid var(--ifm-color-primary)' : '1px solid var(--ifm-color-emphasis-300)',
                  background: active ? 'var(--ifm-color-primary)' : 'transparent',
                  color: active ? '#fff' : 'var(--ifm-color-emphasis-700)',
                  transition: 'all 0.15s',
                }}
              >
                {label}
              </button>
            );
          })}
        </div>
      )}
      <div className={styles.benchmarkGallery}>
        {sorted.map(benchmark => (
          <BenchmarkCard key={benchmark.id} benchmark={benchmark} />
        ))}
      </div>
    </div>
  );
}

function BenchmarkCard({ benchmark }) {
  const [expanded, setExpanded] = useState(false);

  const { id, successRate, attempts, avgTokens, languages, codeSamples, languageStats, taskPrompt, agentStats } = benchmark;

  // Get AILANG and Python specific stats if available
  const ailangStats = languageStats?.ailang;
  const pythonStats = languageStats?.python;

  // Get agent stats if available
  const ailangAgent = agentStats?.ailang;
  const pythonAgent = agentStats?.python;
  const hasAgentData = ailangAgent || pythonAgent;

  // API-error awareness: when infra failures inflate the failure rate,
  // surface a warning chip and show adjusted rates alongside raw ones.
  const ailangApiErrRate = ailangAgent?.apiErrorRate ?? 0;
  const pythonApiErrRate = pythonAgent?.apiErrorRate ?? 0;
  const showApiWarning = ailangApiErrRate > 0.05 || pythonApiErrRate > 0.05;
  const totalApiErrors = (ailangAgent?.apiErrors ?? 0) + (pythonAgent?.apiErrors ?? 0);

  // Per-harness breakdown across both languages, ordered for stable rendering.
  const harnessBreakdown = HARNESS_ORDER
    .map(h => ({
      name: h,
      ailang: ailangAgent?.byHarness?.[h] || null,
      python: pythonAgent?.byHarness?.[h] || null,
    }))
    .filter(row => row.ailang || row.python);

  // Determine status
  let status, statusColor, StatusIcon;
  if (successRate >= 0.8) {
    status = 'Passing';
    statusColor = 'success';
    StatusIcon = CheckCircle;
  } else if (successRate >= 0.5) {
    status = 'Partial';
    statusColor = 'warning';
    StatusIcon = AlertCircle;
  } else {
    status = 'Failing';
    statusColor = 'error';
    StatusIcon = XCircle;
  }

  return (
    <div className={`${styles.benchmarkCard} ${styles[statusColor]} ${expanded ? styles.expanded : ''}`}>
      <div
        className={styles.benchmarkHeader}
        onClick={() => setExpanded(!expanded)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            setExpanded(!expanded);
          }
        }}
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
      >
        <div className={styles.benchmarkTitle}>
          <StatusIcon className={styles.benchmarkIcon} size={24} />
          <span className={styles.benchmarkName}>{formatBenchmarkName(id)}</span>
        </div>
        <div className={styles.benchmarkMeta}>
          {showApiWarning && (
            <span className={styles.apiErrorChip} title={`${totalApiErrors} API errors inflate the failure rate. See per-harness breakdown.`}>
              ⚠ {Math.round(Math.max(ailangApiErrRate, pythonApiErrRate) * 100)}% API err
            </span>
          )}
          <span className={`${styles.statusBadge} ${styles[statusColor]}`}>
            {status}
          </span>
          <button className={styles.expandButton} aria-label="Expand details">
            {expanded ? <ChevronUp size={20} /> : <ChevronDown size={20} />}
          </button>
        </div>
      </div>

      <div className={styles.benchmarkProgress}>
        <div
          className={`${styles.progressBar} ${styles[statusColor]}`}
          style={{ width: `${successRate * 100}%` }}
        />
      </div>

      <div className={styles.benchmarkStats}>
        {ailangStats && pythonStats ? (
          <>
            <div className={styles.stat}>
              <span className={styles.statLabel}>AILANG Success</span>
              <span className={styles.statValue}>{(ailangStats.successRate * 100).toFixed(0)}%</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statLabel}>Python Success</span>
              <span className={styles.statValue}>{(pythonStats.successRate * 100).toFixed(0)}%</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statLabel}>AILANG Tokens</span>
              <span className={styles.statValue}>{Math.round(ailangStats.avgTokens)}</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statLabel}>Python Tokens</span>
              <span className={styles.statValue}>{Math.round(pythonStats.avgTokens)}</span>
            </div>
          </>
        ) : (
          <>
            <div className={styles.stat}>
              <span className={styles.statLabel}>Success</span>
              <span className={styles.statValue}>{(successRate * 100).toFixed(1)}%</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statLabel}>Attempts</span>
              <span className={styles.statValue}>{attempts}</span>
            </div>
            <div className={styles.stat}>
              <span className={styles.statLabel}>Avg Tokens</span>
              <span className={styles.statValue}>{Math.round(avgTokens)}</span>
            </div>
          </>
        )}
      </div>

      {expanded && (
        <div className={styles.benchmarkDetails}>
          {taskPrompt && (
            <div className={styles.detailRow} style={{flexDirection: 'column', alignItems: 'flex-start'}}>
              <span className={styles.detailLabel}>Task Prompt:</span>
              <div className={styles.taskPrompt}>{taskPrompt}</div>
            </div>
          )}
          <div className={styles.detailRow}>
            <span className={styles.detailLabel}>Languages:</span>
            <span className={styles.detailValue}>
              {languages && languages.length > 0 ? languages.join(', ') : 'N/A'}
            </span>
          </div>
          <div className={styles.detailRow}>
            <span className={styles.detailLabel}>Benchmark ID:</span>
            <span className={styles.detailValue}><code>{id}</code></span>
          </div>
          {successRate < 1.0 && (
            <div className={styles.detailHint}>
              <p>💡 {getHint(id, successRate)}</p>
            </div>
          )}
          {languageStats && (
            <div className={styles.languageStatsGrid}>
              {LANG_ORDER.map(lang => {
                const ls = languageStats[lang];
                if (!ls) return null;
                return (
                  <div key={lang} className={styles.langStatCard} style={{ background: heatBg(ls.successRate) }}>
                    <div className={styles.langStatHeader}>{LANG_LABEL[lang]}</div>
                    <div className={styles.langStatRate} style={{ color: rateColor(ls.successRate) }}>
                      {Math.round(ls.successRate * 100)}%
                    </div>
                    <div className={styles.langStatMeta}>
                      {ls.totalRuns} runs · {Math.round(ls.avgTokens)} tok
                    </div>
                  </div>
                );
              })}
            </div>
          )}
          {codeSamples && Object.keys(codeSamples).length > 0 && (
            <div className={styles.codeComparison}>
              <h4 className={styles.comparisonTitle}>Generated Code Comparison</h4>
              <div className={styles.codeGrid}>
                {['ailang', 'python', 'javascript', 'go'].map(lang => (
                  codeSamples[lang] && (
                    <div key={lang} className={styles.codeBlock}>
                      <div className={styles.codeHeader}>{LANG_LABEL[lang] || lang}</div>
                      <pre className={styles.codePre}><code>{codeSamples[lang]}</code></pre>
                    </div>
                  )
                ))}
              </div>
            </div>
          )}
          {hasAgentData && (
            <div className={styles.agentSection}>
              <h4 className={styles.agentTitle}>🤖 Agent Evaluation Performance</h4>
              <p className={styles.agentSubtitle}>
                Multi-turn iterative problem solving with Claude Code
              </p>
              <div className={styles.agentGrid}>
                {ailangAgent && (
                  <div className={styles.agentCard}>
                    <div className={styles.agentCardHeader}>AILANG</div>
                    <div className={styles.agentMetrics}>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Success Rate</span>
                        <span className={styles.agentValue}>
                          {ailangApiErrRate > 0.05
                            ? Math.round(adjRate(ailangAgent.successRate, ailangApiErrRate) * 100)
                            : Math.round(ailangAgent.successRate * 100)
                          }%
                          {ailangApiErrRate > 0.05 && (
                            <span className={styles.adjRate} title="Raw rate before excluding API-error runs">
                              {' '}(raw {Math.round(ailangAgent.successRate * 100)}%)
                            </span>
                          )}
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Avg Turns</span>
                        <span className={styles.agentValue}>
                          {ailangAgent.avgTurns.toFixed(1)}
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Tokens</span>
                        <span className={styles.agentValue}>
                          {(ailangAgent.avgTokens / 1000).toFixed(0)}K
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Runs</span>
                        <span className={styles.agentValue}>
                          {ailangAgent.runs}
                          {ailangApiErrRate > 0.05 && (
                            <span className={styles.apiErrInline} title="API errors (infrastructure failures, not model failures)">
                              {' '}({ailangAgent.apiErrors} api-err)
                            </span>
                          )}
                        </span>
                      </div>
                    </div>
                  </div>
                )}
                {pythonAgent && (
                  <div className={styles.agentCard}>
                    <div className={styles.agentCardHeader}>Python</div>
                    <div className={styles.agentMetrics}>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Success Rate</span>
                        <span className={styles.agentValue}>
                          {pythonApiErrRate > 0.05
                            ? Math.round(adjRate(pythonAgent.successRate, pythonApiErrRate) * 100)
                            : Math.round(pythonAgent.successRate * 100)
                          }%
                          {pythonApiErrRate > 0.05 && (
                            <span className={styles.adjRate} title="Raw rate before excluding API-error runs">
                              {' '}(raw {Math.round(pythonAgent.successRate * 100)}%)
                            </span>
                          )}
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Avg Turns</span>
                        <span className={styles.agentValue}>
                          {pythonAgent.avgTurns.toFixed(1)}
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Tokens</span>
                        <span className={styles.agentValue}>
                          {(pythonAgent.avgTokens / 1000).toFixed(0)}K
                        </span>
                      </div>
                      <div className={styles.agentMetric}>
                        <span className={styles.agentLabel}>Runs</span>
                        <span className={styles.agentValue}>
                          {pythonAgent.runs}
                          {pythonApiErrRate > 0.05 && (
                            <span className={styles.apiErrInline} title="API errors (infrastructure failures, not model failures)">
                              {' '}({pythonAgent.apiErrors} api-err)
                            </span>
                          )}
                        </span>
                      </div>
                    </div>
                  </div>
                )}
              </div>
              {ailangAgent && pythonAgent && (
                <div className={styles.agentComparison}>
                  <p>
                    <strong>Agent Efficiency:</strong> AILANG requires{' '}
                    {(ailangAgent.avgTurns / pythonAgent.avgTurns).toFixed(1)}x more turns
                    and uses {(ailangAgent.avgTokens / pythonAgent.avgTokens).toFixed(1)}x more tokens
                    compared to Python. This reflects the learning curve of a new language.
                  </p>
                </div>
              )}
              {harnessBreakdown.length > 0 && (
                <div className={styles.harnessBreakdown}>
                  <h5 className={styles.harnessTitle}>Per-Harness Breakdown</h5>
                  <p className={styles.harnessSubtitle}>
                    Pass rate per harness × language. Adjusted (adj) excludes API-error runs.
                  </p>
                  <table className={styles.harnessTable}>
                    <thead>
                      <tr>
                        <th>Harness</th>
                        <th>AILANG</th>
                        <th>Python</th>
                      </tr>
                    </thead>
                    <tbody>
                      {harnessBreakdown.map(row => {
                        const renderCell = (h) => {
                          if (!h) return <span style={{ color: 'var(--ifm-color-emphasis-400)' }}>—</span>;
                          const adj = adjRate(h.successRate, h.apiErrorRate);
                          const showAdjusted = h.apiErrorRate > 0.05 && Math.abs(adj - h.successRate) >= 0.01;
                          const primary = showAdjusted ? adj : h.successRate;
                          return (
                            <span style={{ color: rateColor(primary), fontWeight: primary >= 0.85 ? 700 : 400 }}>
                              {Math.round(primary * 100)}%
                              {showAdjusted && (
                                <span className={styles.adjRate} title={`Raw rate before excluding ${h.apiErrors}/${h.runs} API errors`}
                                      style={{ color: 'var(--ifm-color-emphasis-500)', fontStyle: 'italic' }}>
                                  {' '}(raw {Math.round(h.successRate * 100)}%)
                                </span>
                              )}
                              {h.apiErrorRate > 0.05 && (
                                <span className={styles.apiErrInline} title={`${h.apiErrors}/${h.runs} api errors`}>
                                  {' '}⚠ {h.apiErrors}
                                </span>
                              )}
                            </span>
                          );
                        };
                        return (
                          <tr key={row.name}>
                            <td>{HARNESS_LABEL[row.name] || row.name}</td>
                            <td style={{ background: heatBg(row.ailang ? (row.ailang.apiErrorRate > 0.05 ? adjRate(row.ailang.successRate, row.ailang.apiErrorRate) : row.ailang.successRate) : null) }}>{renderCell(row.ailang)}</td>
                            <td style={{ background: heatBg(row.python ? (row.python.apiErrorRate > 0.05 ? adjRate(row.python.successRate, row.python.apiErrorRate) : row.python.successRate) : null) }}>{renderCell(row.python)}</td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function formatBenchmarkName(id) {
  // Convert snake_case to Title Case
  return id
    .split('_')
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(' ');
}

function getHint(id, successRate) {
  // Provide contextual hints based on benchmark and success rate
  if (successRate === 0) {
    return 'This benchmark exposes a known limitation. Check the roadmap for planned fixes.';
  } else if (successRate < 0.5) {
    return 'Low success rate indicates AI models struggle with this pattern. Improving prompts may help.';
  } else {
    return 'Partially working - some edge cases need attention.';
  }
}
