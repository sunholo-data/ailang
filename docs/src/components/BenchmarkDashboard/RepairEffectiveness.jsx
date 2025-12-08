import React from 'react';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell, Label } from 'recharts';
import styles from './styles.module.css';

export default function RepairEffectiveness({ data }) {
  const languages = data.languages || {};
  const aggregates = data.aggregates || {};
  const hasAgentData = aggregates.agentRuns > 0;

  if (!hasAgentData) {
    return null;
  }

  // Calculate repair lift for each language
  const getRepairMetrics = (langData) => {
    const zeroShot = langData.zero_shot_success_comparable || 0;
    const withRepair = langData.final_success_comparable || 0;
    const zeroShotCost = langData.zero_shot_cost_per_success || 0;
    const repairCost = langData.final_cost_per_success_comparable || 0;

    const successImprovement = (withRepair - zeroShot) * 100; // Percentage points
    const repairLift = zeroShot < 1.0 ? ((withRepair - zeroShot) / (1.0 - zeroShot)) * 100 : 0;
    const costEfficiencyChange = repairCost > 0 && zeroShotCost > 0
      ? ((repairCost - zeroShotCost) / zeroShotCost) * 100
      : 0;

    return {
      successImprovement,
      repairLift,
      costEfficiencyChange
    };
  };

  const ailangMetrics = languages.ailang ? getRepairMetrics(languages.ailang) : null;
  const pythonMetrics = languages.python ? getRepairMetrics(languages.python) : null;

  if (!ailangMetrics && !pythonMetrics) {
    return null;
  }

  // Data for repair lift chart
  const repairLiftData = [
    ...(ailangMetrics ? [{
      language: 'AILANG',
      'Failures Fixed by Repair': ailangMetrics.repairLift,
      fill: 'var(--ifm-color-primary)'
    }] : []),
    ...(pythonMetrics ? [{
      language: 'Python',
      'Failures Fixed by Repair': pythonMetrics.repairLift,
      fill: 'var(--ifm-color-success)'
    }] : [])
  ];

  // Data for cost efficiency change
  const costEfficiencyData = [
    ...(ailangMetrics ? [{
      language: 'AILANG',
      'Cost per Success Change': ailangMetrics.costEfficiencyChange,
      fill: ailangMetrics.costEfficiencyChange < 0 ? 'var(--ifm-color-success)' : 'var(--ifm-color-danger)'
    }] : []),
    ...(pythonMetrics ? [{
      language: 'Python',
      'Cost per Success Change': pythonMetrics.costEfficiencyChange,
      fill: pythonMetrics.costEfficiencyChange < 0 ? 'var(--ifm-color-success)' : 'var(--ifm-color-danger)'
    }] : [])
  ];

  // Data for impossibility coverage visualization (stacked bar showing gap closing)
  const getImpossibilityCoverageData = (langData) => {
    const zeroShot = langData.zero_shot_success_comparable || 0;
    const agent = langData.agent_success_rate || 0;
    const agentImprovement = agent - zeroShot;
    const remainingGap = 1.0 - agent;

    return {
      zeroShotPct: zeroShot * 100,
      agentImprovementPct: agentImprovement * 100,
      remainingGapPct: remainingGap * 100,
      impossibilityCoverage: langData.agent_impossibility_coverage || 0
    };
  };

  const impossibilityData = [
    ...(languages.ailang ? [{
      language: 'AILANG',
      ...getImpossibilityCoverageData(languages.ailang)
    }] : []),
    ...(languages.python ? [{
      language: 'Python',
      ...getImpossibilityCoverageData(languages.python)
    }] : [])
  ];

  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      const value = payload[0].value;
      const metricName = payload[0].name;
      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}>{label}</p>
          <p className={styles.tooltipValue}>
            {metricName}: <strong>{value.toFixed(1)}%</strong>
          </p>
        </div>
      );
    }
    return null;
  };

  const ImpossibilityCoverageTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      // Find the data entry to get impossibility coverage
      const dataEntry = impossibilityData.find(d => d.language === label);
      const coverage = dataEntry ? (dataEntry.impossibilityCoverage * 100).toFixed(1) : 0;

      return (
        <div className={styles.chartTooltip}>
          <p className={styles.tooltipLabel}><strong>{label}</strong></p>
          {payload.map((entry, index) => (
            <p key={index} className={styles.tooltipValue}>
              {entry.name}: <strong>{entry.value.toFixed(1)}%</strong>
            </p>
          ))}
          <p className={styles.tooltipValue} style={{ marginTop: '8px', borderTop: '1px solid var(--ifm-color-emphasis-300)', paddingTop: '8px' }}>
            <strong>Impossibility Coverage: {coverage}%</strong>
          </p>
        </div>
      );
    }
    return null;
  };

  return (
    <div className={styles.repairEffectivenessContainer}>
      <h3>Repair Effectiveness Analysis</h3>
      <p className={styles.sectionDescription}>
        How much does one repair iteration (compiler feedback + retry) improve success rates?
        Higher repair lift means the language benefits more from iterative error feedback.
      </p>

      <div className={styles.repairChartsGrid}>
        {/* Chart 1: Repair Lift */}
        <div className={styles.repairChartCard}>
          <h4 className={styles.chartCardTitle}>Failures Fixed by Repair</h4>
          <p className={styles.chartCardSubtitle}>% of initial failures that repair can fix</p>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={repairLiftData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-300)" />
              <XAxis
                dataKey="language"
                tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fill: 'var(--ifm-color-emphasis-600)' }}
                label={{ value: 'Repair Lift (%)', angle: -90, position: 'insideLeft', fill: 'var(--ifm-color-emphasis-700)' }}
              />
              <Tooltip content={<CustomTooltip />} />
              <Bar dataKey="Failures Fixed by Repair">
                {repairLiftData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Chart 2: Cost Efficiency Change */}
        <div className={styles.repairChartCard}>
          <h4 className={styles.chartCardTitle}>Cost Efficiency Impact</h4>
          <p className={styles.chartCardSubtitle}>% change in cost per success (negative = better)</p>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={costEfficiencyData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-300)" />
              <XAxis
                dataKey="language"
                tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
              />
              <YAxis
                tick={{ fill: 'var(--ifm-color-emphasis-600)' }}
                label={{ value: 'Cost Change (%)', angle: -90, position: 'insideLeft', fill: 'var(--ifm-color-emphasis-700)' }}
              />
              <Tooltip content={<CustomTooltip />} />
              <Bar dataKey="Cost per Success Change">
                {costEfficiencyData.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.fill} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>

        {/* Chart 3: Impossibility Coverage (Gap Closing) */}
        <div className={styles.repairChartCard} style={{ gridColumn: 'span 2' }}>
          <h4 className={styles.chartCardTitle}>Impossibility Coverage: Closing the Gap</h4>
          <p className={styles.chartCardSubtitle}>How much of the "impossible" (0-shot failures) did agent solve?</p>
          <ResponsiveContainer width="100%" height={250}>
            <BarChart data={impossibilityData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--ifm-color-emphasis-300)" />
              <XAxis
                dataKey="language"
                tick={{ fill: 'var(--ifm-color-emphasis-800)' }}
              />
              <YAxis
                domain={[0, 100]}
                tick={{ fill: 'var(--ifm-color-emphasis-600)' }}
                label={{ value: 'Success Rate (%)', angle: -90, position: 'insideLeft', fill: 'var(--ifm-color-emphasis-700)' }}
              />
              <Tooltip content={<ImpossibilityCoverageTooltip />} />
              <Bar dataKey="zeroShotPct" stackId="a" fill="var(--ifm-color-primary)" name="0-Shot Success" />
              <Bar dataKey="agentImprovementPct" stackId="a" fill="var(--ifm-color-success)" name="Agent Improvement" />
              <Bar dataKey="remainingGapPct" stackId="a" fill="var(--ifm-color-danger-light)" name="Still Unsolved" />
            </BarChart>
          </ResponsiveContainer>
          <div style={{ marginTop: '12px', fontSize: '0.9em', color: 'var(--ifm-color-emphasis-700)' }}>
            <strong>Reading this chart:</strong>
            <ul style={{ marginTop: '4px', paddingLeft: '20px' }}>
              <li><span style={{ color: 'var(--ifm-color-primary)', fontWeight: 'bold' }}>■</span> Bottom section = 0-shot solved these</li>
              <li><span style={{ color: 'var(--ifm-color-success)', fontWeight: 'bold' }}>■</span> Middle section = Agent closed this much of the gap (impossibility coverage)</li>
              <li><span style={{ color: 'var(--ifm-color-danger)', fontWeight: 'bold' }}>■</span> Top section = Still unsolved even with agent</li>
            </ul>
            <p style={{ marginTop: '8px' }}>
              <strong>Python 100% coverage:</strong> Agent solved ALL benchmarks that 0-shot failed (8.8% → 0% unsolved).<br/>
              <strong>AILANG 38.6% coverage:</strong> Agent solved 38.6% of what 0-shot failed (38.6% → 23.7% still unsolved).
            </p>
          </div>
        </div>
      </div>

      {/* Insight Box */}
      <div className={styles.insightBox}>
        <h4>🔍 Key Insight: Iterative Feedback Value</h4>
        <div className={styles.insightContent}>
          {ailangMetrics && (
            <div className={styles.insightRow}>
              <strong>AILANG:</strong> Repairs fix <span className={styles.highlightGood}>{ailangMetrics.repairLift.toFixed(0)}%</span> of failures,
              making each success <span className={ailangMetrics.costEfficiencyChange < 0 ? styles.highlightGood : styles.highlightBad}>
                {Math.abs(ailangMetrics.costEfficiencyChange).toFixed(0)}% {ailangMetrics.costEfficiencyChange < 0 ? 'cheaper' : 'more expensive'}</span>.
              <br/>
              <em>→ Compiler feedback is highly valuable for this new language</em>
            </div>
          )}
          {pythonMetrics && (
            <div className={styles.insightRow}>
              <strong>Python:</strong> Repairs fix <span className={pythonMetrics.repairLift > 10 ? styles.highlightGood : styles.highlightBad}>
                {pythonMetrics.repairLift.toFixed(0)}%</span> of failures,
              making each success <span className={styles.highlightBad}>{Math.abs(pythonMetrics.costEfficiencyChange).toFixed(0)}% more expensive</span>.
              <br/>
              <em>→ LLMs already know Python well; repairs just waste tokens</em>
            </div>
          )}
          <div className={styles.insightSummary}>
            <strong>Conclusion:</strong> Structured compiler feedback (M-EVAL-LOOP) is most valuable for
            new/complex languages where LLMs lack strong priors. For well-known languages,
            the overhead may not be justified.
          </div>
        </div>
      </div>
    </div>
  );
}
