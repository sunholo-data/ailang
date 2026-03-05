/**
 * Hook for fetching execution chain data from the Chains API.
 *
 * Attempts to find a chain associated with a selected event (by task_id or message_id).
 * When a chain is found, fetches stages with spans for rich visualization.
 * When no chain exists, synthesizes a virtual single-stage chain from fallback spans.
 *
 * M-CHAIN-DASHBOARD: "Everything is a chain" — all data gets chain representation.
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { ChainData, ChainStageData, Span } from '../components/ExecHierarchy/types';

export interface UseChainDataOptions {
  /** Task ID to look up chain for (from event selection) */
  taskId?: string | null;
  /** Message ID fallback for chain lookup */
  messageId?: string | null;
  /** Whether to include spans in each stage (default false — use useStageSpans for L2 loading) */
  includeSpans?: boolean;
  /** Spans from useTraceData — used to synthesize a virtual chain when no real chain exists */
  fallbackSpans?: Span[];
}

export interface UseChainDataResult {
  /** The execution chain with stages, or null if not found */
  chain: ChainData | null;
  /** Loading state */
  loading: boolean;
  /** Error message if fetch failed */
  error: string | null;
  /** Whether a chain was found for the current selection (real or synthesized) */
  hasChain: boolean;
  /** Flattened spans from all stages (bridge to existing span-based views) */
  allStageSpans: Span[];
  /** The currently active stage (based on chain.current_stage) */
  currentStage: ChainStageData | null;
  /** Force re-fetch */
  refetch: () => void;
}

// Convert raw API span to the Span type used by ExecHierarchy views
function convertApiSpan(raw: any): Span {
  return {
    id: raw.id,
    name: raw.name,
    display_name: raw.display_name,
    startMs: raw.start_time ? new Date(raw.start_time).getTime() : 0,
    durationMs: raw.duration_ms || 0,
    status: raw.status === 'error' || raw.status === 'ERROR' ? 'error' : 'ok',
    attributes: raw.attributes,
    cost_usd: raw.cost_usd,
    tokens_in: raw.tokens_in,
    tokens_out: raw.tokens_out,
    provider: raw.provider,
    chat_context: raw.chat_context,
    children: raw.children?.map(convertApiSpan) || [],
  };
}

/**
 * Synthesize a virtual single-stage ChainData from raw spans.
 * Used when no real chain exists in the database (legacy data, standalone CLI, etc).
 * Virtual IDs are prefixed with "virtual-" to avoid collision with real UUIDs.
 */
function synthesizeChainFromSpans(
  spans: Span[],
  taskId?: string | null,
  messageId?: string | null,
): ChainData | null {
  if (!spans || spans.length === 0) return null;

  const rootSpan = spans[0];
  const rootName = rootSpan.name || '';
  const idSuffix = taskId || messageId || rootSpan.id;

  // Infer source_type from span characteristics
  let sourceType = 'session';
  if (rootName.includes('eval')) sourceType = 'eval_suite';
  else if (rootName.startsWith('compile.') || rootName.startsWith('ailang.')) sourceType = 'cli';

  // Infer agent_id from root span name
  let agentId = 'unknown';
  const provider = rootSpan.attributes?.['provider'] || rootSpan.attributes?.['service.name'];
  if (rootName === 'claude_code.session' || rootName === 'claude.execute') agentId = 'claude-code';
  else if (rootName === 'gemini.execute') agentId = 'gemini-cli';
  else if (rootName.startsWith('eval.')) agentId = 'eval-runner';
  else if (rootName.startsWith('compile.') || rootName.startsWith('ailang.')) agentId = 'ailang-cli';
  else if (provider) agentId = String(provider);

  // Derive status from spans
  const hasError = spans.some(s => s.status === 'error');
  const status: ChainData['status'] = hasError ? 'failed' : 'completed';

  // Sum metrics from root-level spans
  let totalCost = 0;
  let totalTokensIn = 0;
  let totalTokensOut = 0;
  let maxDuration = 0;
  for (const span of spans) {
    totalCost += span.cost_usd || 0;
    totalTokensIn += span.tokens_in || 0;
    totalTokensOut += span.tokens_out || 0;
    if (span.durationMs > maxDuration) maxDuration = span.durationMs;
  }

  // Count turns by looking for turn-like spans in the tree
  let totalTurns = 0;
  const countTurns = (spanList: Span[]) => {
    for (const s of spanList) {
      if (s.name.includes('turn') || s.name === 'api_request') totalTurns++;
      if (s.children) countTurns(s.children);
    }
  };
  countTurns(spans);

  const virtualStage: ChainStageData = {
    id: `virtual-stage-${idSuffix}`,
    chain_id: `virtual-chain-${idSuffix}`,
    stage_number: 1,
    agent_id: agentId,
    provider: provider ? String(provider) : undefined,
    task_id: taskId || undefined,
    // For Claude Code sessions, task_id IS the session UUID — set session_id so ChatHistory can find it
    session_id: taskId && /^[0-9a-f]{8}-[0-9a-f]{4}-/i.test(taskId) ? taskId : undefined,
    status: hasError ? 'failed' : 'completed',
    iteration: 1,
    cost: totalCost,
    tokens_in: totalTokensIn,
    tokens_out: totalTokensOut,
    turns: totalTurns,
    tool_calls: 0,
    duration_ms: maxDuration,
    spans: spans,
  };

  return {
    id: `virtual-chain-${idSuffix}`,
    source_type: sourceType,
    status,
    current_stage: 1,
    created_at: rootSpan.startMs ? new Date(rootSpan.startMs).toISOString() : new Date().toISOString(),
    total_cost: totalCost,
    total_tokens: totalTokensIn + totalTokensOut,
    total_turns: totalTurns,
    stages_completed: 1,
    stages: [virtualStage],
  };
}

export function useChainData(options: UseChainDataOptions = {}): UseChainDataResult {
  const { taskId, messageId, includeSpans = false, fallbackSpans } = options;
  const [chain, setChain] = useState<ChainData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fetchTrigger, setFetchTrigger] = useState(0);
  // Track whether API found a real chain (vs synthesized)
  const apiFoundChainRef = useRef(false);

  // Track the last lookup to avoid duplicate fetches
  const lastLookupRef = useRef<string>('');

  const refetch = useCallback(() => {
    setFetchTrigger(prev => prev + 1);
  }, []);

  // Effect 1: Fetch real chain from API
  useEffect(() => {
    // No task or message to look up
    if (!taskId && !messageId) {
      setChain(null);
      setLoading(false);
      setError(null);
      apiFoundChainRef.current = false;
      return;
    }

    const lookupKey = `${taskId || ''}-${messageId || ''}-${fetchTrigger}`;
    if (lookupKey === lastLookupRef.current && fetchTrigger === 0) {
      return; // Same lookup, skip
    }
    lastLookupRef.current = lookupKey;

    let cancelled = false;

    async function fetchChain() {
      setLoading(true);
      setError(null);
      apiFoundChainRef.current = false;

      try {
        // Step 1: Find chain by task_id or message_id
        let chainSummary: ChainData | null = null;

        if (taskId) {
          const resp = await fetch(`/api/chains/by-task/${encodeURIComponent(taskId)}`);
          if (resp.ok) {
            const data = await resp.json();
            // Only treat as a chain if it has the expected chain shape (has `id` and `source_type`).
            // The by-task endpoint may return a TaskSpanSummary fallback (has `task_id` but no `id`).
            if (data && data.id && data.source_type) {
              chainSummary = data;
            }
          }
        }

        // Fallback: try by message_id
        if (!chainSummary && messageId) {
          const resp = await fetch(`/api/chains/by-message/${encodeURIComponent(messageId)}`);
          if (resp.ok) {
            const data = await resp.json();
            if (data && data.id && data.source_type) {
              chainSummary = data;
            }
          }
        }

        if (cancelled) return;

        // No chain found from API — will synthesize from spans in Effect 2
        if (!chainSummary) {
          setChain(null);
          setLoading(false);
          return;
        }

        // Step 2: Fetch full chain with stages and spans
        apiFoundChainRef.current = true;
        const params = new URLSearchParams();
        if (includeSpans) params.set('include_spans', 'true');
        params.set('include_sessions', 'true');

        const fullResp = await fetch(`/api/chains/${encodeURIComponent(chainSummary.id)}?${params.toString()}`);
        if (cancelled) return;

        if (!fullResp.ok) {
          // Have chain summary but can't get full data - use what we have
          setChain(chainSummary);
          setLoading(false);
          return;
        }

        const fullChain: ChainData = await fullResp.json();

        // Convert raw spans in stages to our Span format
        if (fullChain.stages) {
          for (const stage of fullChain.stages) {
            if (stage.spans) {
              stage.spans = stage.spans.map(convertApiSpan);
            }
          }
        }

        if (!cancelled) {
          setChain(fullChain);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to fetch chain');
          setChain(null);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchChain();

    return () => {
      cancelled = true;
    };
  }, [taskId, messageId, includeSpans, fetchTrigger]);

  // Effect 2: Synthesize virtual chain from fallback spans when no real chain exists.
  // Real chains with empty stages rely on useStageSpans for lazy span loading (L2).
  // Frontend does NOT enrich backend data — that was a source of re-render loops.
  useEffect(() => {
    if (loading) return;
    if (!fallbackSpans || fallbackSpans.length === 0) return;
    if (apiFoundChainRef.current) return; // Real chain found — use it as-is
    if (chain !== null) return; // Already have a chain

    const synthetic = synthesizeChainFromSpans(fallbackSpans, taskId, messageId);
    if (synthetic) {
      setChain(synthetic);
    }
  }, [chain, fallbackSpans, loading, taskId, messageId]);

  // Derived: flatten spans from all stages
  const allStageSpans: Span[] = chain?.stages
    ?.flatMap(stage => stage.spans || [])
    || [];

  // Derived: current stage
  const currentStage: ChainStageData | null = chain?.stages
    ?.find(s => s.stage_number === chain.current_stage)
    || null;

  return {
    chain,
    loading,
    error,
    hasChain: chain !== null,
    allStageSpans,
    currentStage,
    refetch,
  };
}
