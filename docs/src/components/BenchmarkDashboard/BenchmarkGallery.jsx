import React, { useState } from 'react';
import { CheckCircle, XCircle, AlertCircle, ChevronDown, ChevronUp } from 'lucide-react';
import styles from './styles.module.css';

export default function BenchmarkGallery({ benchmarks }) {
  // Convert benchmarks object to array and sort by success rate
  const benchmarkArray = Object.entries(benchmarks).map(([id, stats]) => ({
    id,
    ...stats
  })).sort((a, b) => b.successRate - a.successRate);

  return (
    <div className={styles.benchmarkGallery}>
      {benchmarkArray.map(benchmark => (
        <BenchmarkCard key={benchmark.id} benchmark={benchmark} />
      ))}
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
    <div className={`${styles.benchmarkCard} ${styles[statusColor]}`}>
      <div className={styles.benchmarkHeader} onClick={() => setExpanded(!expanded)}>
        <div className={styles.benchmarkTitle}>
          <StatusIcon className={styles.benchmarkIcon} size={24} />
          <span className={styles.benchmarkName}>{formatBenchmarkName(id)}</span>
        </div>
        <div className={styles.benchmarkMeta}>
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
          {codeSamples && (codeSamples.ailang || codeSamples.python) && (
            <div className={styles.codeComparison}>
              <h4 className={styles.comparisonTitle}>Generated Code Comparison</h4>
              <div className={styles.codeGrid}>
                {codeSamples.ailang && (
                  <div className={styles.codeBlock}>
                    <div className={styles.codeHeader}>AILANG</div>
                    <pre className={styles.codePre}><code>{codeSamples.ailang}</code></pre>
                  </div>
                )}
                {codeSamples.python && (
                  <div className={styles.codeBlock}>
                    <div className={styles.codeHeader}>Python</div>
                    <pre className={styles.codePre}><code>{codeSamples.python}</code></pre>
                  </div>
                )}
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
                          {(ailangAgent.successRate * 100).toFixed(0)}%
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
                          {(pythonAgent.successRate * 100).toFixed(0)}%
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
