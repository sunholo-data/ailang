/**
 * ChainExplorer - Master-detail container for browsing and inspecting execution chains.
 * Replaces TaskHierarchyGraph + ExecHierarchyTree with a chain-centric view.
 *
 * Two modes:
 * - Event mode (default): Full-width ChainDetail showing chain from parent's filtering pipeline.
 *   The parent (ControlPlane) provides chainData via useChainData which respects the event
 *   queue selection + aggregation filters. When no real chain exists, a virtual chain is
 *   synthesized from the filtered spans.
 * - Browse mode: Left panel (ChainList) + right panel (ChainDetail). Entered only when the
 *   user explicitly clicks "Browse All Chains". Makes independent API calls to /api/chains.
 */
import React, { useState, useCallback, useEffect } from 'react';
import type { ChainData, Span, HierarchyNode, FilterCriteria, ControlPlaneFilters } from '../ExecHierarchy/types';
import { ChainList } from './ChainList';
import { ChainDetail } from './ChainDetail';
import { CliCommandHint } from '../CliCommandHint';
import styles from './ChainExplorer.module.css';

// ============================================================================
// Props
// ============================================================================

export interface ChainExplorerProps {
  isExpanded: boolean;
  spans?: Span[];
  loading?: boolean;
  chainData?: ChainData | null;
  filterCriteria?: FilterCriteria;
  hiddenSpanTypes?: Set<string>;
  onToggleSpanType?: (spanType: string) => void;
  filters?: ControlPlaneFilters;
  theme?: 'dark' | 'light';
  onNodeClick?: (node: HierarchyNode, event?: React.MouseEvent) => void;
  selectedNodeId?: string | null;
}

// Convert raw API span to the Span type used by views
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

// Build the equivalent `ailang chains` CLI command for the current state.
// Virtual chains (synthesized from spans) don't exist in the database,
// so we show a trace-based command instead.
function buildChainsCliCommand(
  mode: 'event' | 'browse',
  chainId: string | null | undefined,
): string {
  if (chainId) {
    if (chainId.startsWith('virtual-chain-')) {
      // Virtual chain — extract the task/message ID suffix and show find command
      const suffix = chainId.replace('virtual-chain-', '');
      return `ailang chains find --task-id ${suffix}`;
    }
    return `ailang chains view ${chainId}`;
  }
  if (mode === 'browse') {
    return 'ailang chains list';
  }
  return 'ailang chains list';
}

// ============================================================================
// Component
// ============================================================================

export const ChainExplorer: React.FC<ChainExplorerProps> = ({
  isExpanded,
  spans,
  loading,
  chainData: eventChainData,
  hiddenSpanTypes,
  theme,
}) => {
  // Always default to 'event' mode — this respects the parent's filtering pipeline.
  // Browse mode is only entered when the user explicitly clicks "Browse All Chains".
  const [mode, setMode] = useState<'event' | 'browse'>('event');
  const [browseSelectedChainId, setBrowseSelectedChainId] = useState<string | null>(null);

  // For browse mode: fetch chain by ID directly
  const [fetchedBrowseChain, setFetchedBrowseChain] = useState<ChainData | null>(null);
  const [browseFetchLoading, setBrowseFetchLoading] = useState(false);

  useEffect(() => {
    if (!browseSelectedChainId || mode !== 'browse') {
      setFetchedBrowseChain(null);
      return;
    }

    let cancelled = false;
    setBrowseFetchLoading(true);

    async function fetchChain() {
      try {
        const resp = await fetch(
          `/api/chains/${encodeURIComponent(browseSelectedChainId!)}?include_spans=true&include_sessions=true`
        );
        if (!resp.ok || cancelled) return;
        const data = await resp.json();

        // Convert span timestamps
        if (data.stages) {
          for (const stage of data.stages) {
            if (stage.spans) {
              stage.spans = stage.spans.map(convertApiSpan);
            }
          }
        }

        if (!cancelled) setFetchedBrowseChain(data);
      } catch {
        // Silently fail - chain just won't show detail
      } finally {
        if (!cancelled) setBrowseFetchLoading(false);
      }
    }

    fetchChain();
    return () => { cancelled = true; };
  }, [browseSelectedChainId, mode]);

  // When chainData arrives from parent, switch back to event mode
  // UNLESS user has explicitly selected a chain in browse mode
  useEffect(() => {
    if (eventChainData && mode === 'browse' && !browseSelectedChainId) {
      setMode('event');
    }
  }, [eventChainData, mode, browseSelectedChainId]);

  // The chain to display
  const activeChain = mode === 'event' ? eventChainData : fetchedBrowseChain;

  // Current chain ID for CLI hint
  const currentChainId = mode === 'event'
    ? eventChainData?.id
    : browseSelectedChainId;

  const handleBrowseAll = useCallback(() => {
    setMode('browse');
  }, []);

  const handleSelectChain = useCallback((chainId: string) => {
    setBrowseSelectedChainId(chainId);
  }, []);

  const handleBackToEvent = useCallback(() => {
    setMode('event');
    setBrowseSelectedChainId(null);
  }, []);

  // Loading state
  if (loading) {
    return (
      <div className={styles.explorerContainer}>
        <div className={styles.loadingState}>Loading chain data...</div>
      </div>
    );
  }

  // Browse mode: two-panel layout
  if (mode === 'browse') {
    return (
      <div className={styles.explorerContainer}>
        <div className={styles.browseModeLayout}>
          <div className={styles.chainListPanel}>
            <ChainList
              selectedChainId={browseSelectedChainId}
              onSelectChain={handleSelectChain}
            />
          </div>
          <div className={styles.chainDetailPanel}>
            {browseFetchLoading && (
              <div className={styles.loadingState}>Loading chain...</div>
            )}
            {!browseFetchLoading && fetchedBrowseChain && (
              <ChainDetail
                chain={fetchedBrowseChain}
                onBrowseAll={handleBackToEvent}
                hiddenSpanTypes={hiddenSpanTypes}
                theme={theme}
              />
            )}
            {!browseFetchLoading && !fetchedBrowseChain && !browseSelectedChainId && (
              <div className={styles.selectChainPrompt}>
                Select a chain from the list to view details
              </div>
            )}
            {eventChainData && !fetchedBrowseChain && !browseFetchLoading && (
              <div className={styles.selectChainPrompt}>
                <button className={styles.browseAllButton} onClick={handleBackToEvent}>
                  Back to Selected Event
                </button>
              </div>
            )}
          </div>
        </div>
        <CliCommandHint
          command={buildChainsCliCommand('browse', browseSelectedChainId)}
          compact
        />
      </div>
    );
  }

  // Event mode: full-width detail (chain from parent's filtering pipeline)
  if (activeChain) {
    return (
      <div className={styles.explorerContainer}>
        <ChainDetail
          chain={activeChain}
          onBrowseAll={handleBrowseAll}
          hiddenSpanTypes={hiddenSpanTypes}
          theme={theme}
        />
        <CliCommandHint
          command={buildChainsCliCommand('event', currentChainId)}
          compact
        />
      </div>
    );
  }

  // No chain and no spans — show empty state with browse option
  return (
    <div className={styles.explorerContainer}>
      <div className={styles.emptyExplorer}>
        <p>Select an event to view its chain, or browse all chains.</p>
        <button className={styles.browseAllButton} onClick={handleBrowseAll}>
          Browse All Chains
        </button>
      </div>
      <CliCommandHint command="ailang chains list" compact />
    </div>
  );
};

export default ChainExplorer;
