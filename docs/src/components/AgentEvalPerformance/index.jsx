import React, { useState, useEffect } from 'react';
import { benchmarkFetch } from '@site/src/lib/benchmarkFetch';
import AgentRadar from '@site/src/components/BenchmarkDashboard/AgentRadar';

// Standalone wrapper for the "Agent Evaluation Performance" view (executor cards +
// success-rate-by-executor). Moved off the performance leaderboard to the Agent
// Harness Explorer page (M-EVAL dashboard rework). AgentRadar takes the full
// latest.json as a `data` prop, so we fetch it here (runtime, with in-build
// fallback) and hand it over.
export default function AgentEvalPerformance() {
  const [data, setData] = useState(null);
  useEffect(() => {
    benchmarkFetch('latest.json')
      .then((r) => (r.ok ? r.json() : null))
      .then(setData)
      .catch(() => setData(null));
  }, []);
  if (!data) return null;
  return <AgentRadar data={data} />;
}
