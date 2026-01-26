/**
 * ChatHistory - Shows conversation history for selected task/span
 *
 * Supports THREE data sources (in priority order):
 * 1. Claude Code JSONL files - full conversations with thinking blocks via /api/claude-history/
 * 2. Coordinator tasks - fetches events from /api/coordinator/tasks/{taskId}/events
 * 3. OTEL hierarchy - uses hierarchy children (turns/tool_uses) as fallback
 *
 * Also supports SEARCH mode (M5):
 * - Search across all Claude Code sessions by keyword
 * - Results show context snippets
 * - Click result to open full conversation
 * - Filters: project, model
 *
 * PR 4 Refactoring (M-DASHBOARD-SIMPLIFICATION):
 * - Utility functions extracted to utils/chatHistoryUtils.ts
 */
import React, { useState, useEffect, useMemo, useCallback } from 'react';
import type { HierarchyNode, Span } from './types';
import styles from './ExecHierarchy.module.css';

// Import extracted utilities (PR 4 - M-DASHBOARD-SIMPLIFICATION)
import {
  groupEventsByTurn,
  extractHierarchyTurns,
  formatTime,
  formatChatDuration,
  formatJsonString,
  extractTaskIdFromContext,
  extractClaudeSessionId,
  findSelectedNode,
  findSelectedSpan,
  type TaskEvent,
  type Turn,
  type HierarchyTurn,
} from '../../utils/chatHistoryUtils';

export interface ChatHistoryProps {
  /** All hierarchy nodes */
  nodes: HierarchyNode[];
  /** Currently selected node ID */
  selectedNodeId?: string | null;
  /** Callback when a node is clicked */
  onNodeClick?: (node: HierarchyNode) => void;
  /** Loading state */
  loading?: boolean;
  /** All spans (for finding selected span) */
  spans?: Span[];
}

// Claude Code JSONL types
interface ClaudeCodeSession {
  id: string;
  project_path: string;
  project_name: string;
  messages: ClaudeCodeMessage[];
  start_time: string;
  end_time: string;
  turn_count: number;
  total_in: number;
  total_out: number;
  cache_read: number;
  cache_write: number;
  model: string;
  git_branch: string;
  cwd: string;
}

interface ClaudeCodeMessage {
  uuid: string;
  parent_uuid?: string;
  session_id: string;
  type: string; // "user" or "assistant"
  timestamp: string;
  model?: string;
  message_id?: string;
  request_id?: string;
  content: ClaudeCodeContentBlock[];
  usage?: ClaudeCodeUsage;
  git_branch?: string;
  cwd?: string;
  stop_reason?: string;
}

interface ClaudeCodeContentBlock {
  type: string; // "thinking", "text", "tool_use", "tool_result"
  text?: string;
  thinking?: string;
  tool_use?: {
    id: string;
    name: string;
    input: unknown;
  };
  tool_result?: {
    tool_use_id: string;
    content: string;
    is_error: boolean;
  };
}

interface ClaudeCodeUsage {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens?: number;
  cache_creation_tokens?: number;
}

// Search result from /api/claude-history/search
interface SearchResult {
  session_id: string;
  project_path: string;
  project_name: string;
  model: string;
  timestamp: string;
  snippet: string;
  turn_count: number;
}

// Search mode state
interface SearchState {
  query: string;
  results: SearchResult[];
  loading: boolean;
  error: string | null;
  filters: {
    project: string;
    model: string;
  };
}

// Types and utility functions imported from utils/chatHistoryUtils.ts
// NOTE: formatJson remains inline as it returns JSX

/**
 * Format JSON for display (returns JSX, kept inline)
 */
function formatJson(input: string | undefined): React.ReactNode {
  if (!input) return null;

  const formatted = formatJsonString(input);
  return <pre className={styles.chatJsonContent}>{formatted}</pre>;
}

/**
 * formatDuration alias for backward compatibility
 */
const formatDuration = formatChatDuration;

export const ChatHistory: React.FC<ChatHistoryProps> = ({
  nodes,
  selectedNodeId,
  onNodeClick,
  loading,
  spans,
}) => {
  const [coordinatorEvents, setCoordinatorEvents] = useState<TaskEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [eventsError, setEventsError] = useState<string | null>(null);
  const [expandedTools, setExpandedTools] = useState<Set<string>>(new Set());
  const [expandedThinking, setExpandedThinking] = useState<Set<string>>(new Set());

  // Claude Code JSONL state
  const [claudeCodeSession, setClaudeCodeSession] = useState<ClaudeCodeSession | null>(null);
  const [claudeCodeLoading, setClaudeCodeLoading] = useState(false);
  const [claudeCodeError, setClaudeCodeError] = useState<string | null>(null);
  const [isTimeFiltered, setIsTimeFiltered] = useState(false); // true when using by-span endpoint
  const [spanTimeWindow, setSpanTimeWindow] = useState<{ start: string; end: string } | null>(null);

  // Search mode state (M5)
  const [searchMode, setSearchMode] = useState(false);
  const [searchState, setSearchState] = useState<SearchState>({
    query: '',
    results: [],
    loading: false,
    error: null,
    filters: { project: '', model: '' },
  });
  const [searchDebounceTimer, setSearchDebounceTimer] = useState<ReturnType<typeof setTimeout> | null>(null);

  // Find selected node in hierarchy - uses helper from chatHistoryUtils.ts
  const selectedNode = useMemo(
    () => findSelectedNode(nodes, selectedNodeId),
    [nodes, selectedNodeId]
  );

  // Find selected span - uses helper from chatHistoryUtils.ts
  const selectedSpan = useMemo(
    () => findSelectedSpan(spans, selectedNodeId),
    [spans, selectedNodeId]
  );

  // Extract task ID - uses helper from chatHistoryUtils.ts
  const taskId = useMemo(
    () => extractTaskIdFromContext(selectedNode, selectedSpan, selectedNodeId),
    [selectedNode, selectedSpan, selectedNodeId]
  );

  // Extract Claude session ID - uses helper from chatHistoryUtils.ts
  const claudeSessionId = useMemo(
    () => extractClaudeSessionId(selectedSpan, selectedNode, selectedNodeId),
    [selectedSpan, selectedNode, selectedNodeId]
  );

  // Extract hierarchy turns (for OTEL spans)
  const hierarchyTurns = useMemo(() => {
    return selectedNode ? extractHierarchyTurns(selectedNode) : [];
  }, [selectedNode]);

  // Fetch Claude Code session using by-span endpoint (preferred) or session endpoint (fallback)
  // by-span gives us time-filtered messages relevant to the selected span
  useEffect(() => {
    // Need either span ID or session ID to fetch
    if (!selectedSpan?.id && !claudeSessionId) {
      setClaudeCodeSession(null);
      return;
    }

    const fetchClaudeSession = async () => {
      setClaudeCodeLoading(true);
      setClaudeCodeError(null);

      try {
        // Try by-span endpoint first (returns time-filtered messages)
        if (selectedSpan?.id) {
          const bySpanResponse = await fetch(`/api/claude-history/by-span/${selectedSpan.id}`);
          if (bySpanResponse.ok) {
            const data = await bySpanResponse.json();
            if (data && data.messages && data.messages.length > 0) {
              setClaudeCodeSession(data);
              setIsTimeFiltered(true);
              // Store time window info if available
              if (data.span_start && data.span_end) {
                setSpanTimeWindow({ start: data.span_start, end: data.span_end });
              } else {
                setSpanTimeWindow(null);
              }
              return;
            }
          }
          // by-span returned 404 or no messages - fall through to session endpoint
        }

        // Fallback: fetch full session by session.id
        if (claudeSessionId) {
          const response = await fetch(`/api/claude-history/session/${claudeSessionId}`);
          if (!response.ok) {
            if (response.status === 404) {
              // Session not found - this is okay, will fallback to coordinator/OTEL
              setClaudeCodeSession(null);
              setIsTimeFiltered(false);
              setSpanTimeWindow(null);
              return;
            }
            throw new Error(`Failed to fetch session: ${response.statusText}`);
          }
          const data = await response.json();
          setClaudeCodeSession(data);
          setIsTimeFiltered(false);
          setSpanTimeWindow(null);
          return;
        }

        // No data available
        setClaudeCodeSession(null);
        setIsTimeFiltered(false);
        setSpanTimeWindow(null);
      } catch (err) {
        setClaudeCodeError(err instanceof Error ? err.message : 'Failed to load Claude session');
        setClaudeCodeSession(null);
      } finally {
        setClaudeCodeLoading(false);
      }
    };

    fetchClaudeSession();
  }, [selectedSpan?.id, claudeSessionId]);

  // Fetch coordinator events if we have a taskId
  // ALWAYS fetch for coordinator tasks - they have actual conversation text
  // OTEL spans only have tool metadata, not the assistant's text
  useEffect(() => {
    if (!taskId) {
      setCoordinatorEvents([]);
      return;
    }

    const fetchEvents = async () => {
      setEventsLoading(true);
      setEventsError(null);

      try {
        const response = await fetch(`/api/coordinator/tasks/${taskId}/events?limit=500`);
        if (!response.ok) {
          throw new Error(`Failed to fetch events: ${response.statusText}`);
        }
        const data = await response.json();
        setCoordinatorEvents(data.events || []);
      } catch (err) {
        setEventsError(err instanceof Error ? err.message : 'Failed to load events');
        setCoordinatorEvents([]);
      } finally {
        setEventsLoading(false);
      }
    };

    fetchEvents();
  }, [taskId]);

  // Group coordinator events by turn
  const coordinatorTurns = useMemo(() => {
    return groupEventsByTurn(coordinatorEvents);
  }, [coordinatorEvents]);

  // Determine if we have hierarchy data (OTEL spans) - used as fallback only
  const hasHierarchyData = hierarchyTurns.length > 0;

  // Toggle tool expansion
  const toggleTool = (id: string) => {
    setExpandedTools(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Toggle thinking block expansion
  const toggleThinking = (id: string) => {
    setExpandedThinking(prev => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Search functionality (M5)
  const performSearch = useCallback(async (query: string, filters: { project: string; model: string }) => {
    if (!query.trim()) {
      setSearchState(prev => ({ ...prev, results: [], error: null }));
      return;
    }

    setSearchState(prev => ({ ...prev, loading: true, error: null }));

    try {
      const params = new URLSearchParams({ q: query, limit: '20' });
      if (filters.project) params.append('project', filters.project);
      if (filters.model) params.append('model', filters.model);

      const response = await fetch(`/api/claude-history/search?${params}`);
      if (!response.ok) {
        throw new Error(`Search failed: ${response.statusText}`);
      }

      const results = await response.json();
      setSearchState(prev => ({
        ...prev,
        results: results || [],
        loading: false,
      }));
    } catch (err) {
      setSearchState(prev => ({
        ...prev,
        error: err instanceof Error ? err.message : 'Search failed',
        loading: false,
      }));
    }
  }, []);

  // Debounced search handler
  const handleSearchInput = useCallback((query: string) => {
    setSearchState(prev => ({ ...prev, query }));

    // Clear existing timer
    if (searchDebounceTimer) {
      clearTimeout(searchDebounceTimer);
    }

    // Set new debounce timer (300ms)
    const timer = setTimeout(() => {
      performSearch(query, searchState.filters);
    }, 300);
    setSearchDebounceTimer(timer);
  }, [searchDebounceTimer, searchState.filters, performSearch]);

  // Handle filter changes
  const handleFilterChange = useCallback((filterType: 'project' | 'model', value: string) => {
    const newFilters = { ...searchState.filters, [filterType]: value };
    setSearchState(prev => ({ ...prev, filters: newFilters }));
    // Re-run search with new filters
    if (searchState.query.trim()) {
      performSearch(searchState.query, newFilters);
    }
  }, [searchState.query, searchState.filters, performSearch]);

  // Load full session from search result
  const loadSearchResult = useCallback(async (result: SearchResult) => {
    setSearchMode(false);
    setClaudeCodeLoading(true);
    setClaudeCodeError(null);

    try {
      const response = await fetch(`/api/claude-history/session/${result.session_id}`);
      if (!response.ok) {
        throw new Error(`Failed to load session: ${response.statusText}`);
      }
      const data = await response.json();
      setClaudeCodeSession(data);
      setIsTimeFiltered(false);
      setSpanTimeWindow(null);
    } catch (err) {
      setClaudeCodeError(err instanceof Error ? err.message : 'Failed to load session');
      setClaudeCodeSession(null);
    } finally {
      setClaudeCodeLoading(false);
    }
  }, []);

  // Toggle search mode
  const toggleSearchMode = useCallback(() => {
    setSearchMode(prev => !prev);
    if (!searchMode) {
      // Entering search mode - clear previous session
      setClaudeCodeSession(null);
    }
  }, [searchMode]);

  // Render loading state
  if (loading || eventsLoading || claudeCodeLoading) {
    return (
      <div className={styles.chatContainer}>
        <div className={styles.chatLoading}>Loading conversation...</div>
      </div>
    );
  }

  // Render no selection state - but still show search option
  if (!selectedNodeId && !searchMode) {
    return (
      <div className={styles.chatContainer}>
        {/* Search toggle button */}
        <div className={styles.chatSearchToggle}>
          <button
            className={styles.chatSearchToggleBtn}
            onClick={toggleSearchMode}
            title="Search all conversations"
          >
            🔍 Search Conversations
          </button>
        </div>
        <div className={styles.chatEmpty}>
          <div className={styles.chatEmptyIcon}>💬</div>
          <div className={styles.chatEmptyText}>Select a task to view conversation</div>
          <div className={styles.chatEmptyHint}>
            Click on a session in the tree or graph view, or use Search above
          </div>
        </div>
      </div>
    );
  }

  // ========================================
  // RENDER: Search mode (M5)
  // ========================================
  if (searchMode) {
    return (
      <div className={styles.chatContainer}>
        {/* Search header */}
        <div className={styles.chatSearchHeader}>
          <button
            className={styles.chatSearchBackBtn}
            onClick={toggleSearchMode}
            title="Back to conversation view"
          >
            ← Back
          </button>
          <span className={styles.chatSearchTitle}>🔍 Search Conversations</span>
        </div>

        {/* Search input */}
        <div className={styles.chatSearchInputArea}>
          <input
            type="text"
            className={styles.chatSearchInput}
            placeholder="Search across all Claude Code sessions..."
            value={searchState.query}
            onChange={(e) => handleSearchInput(e.target.value)}
            autoFocus
          />
        </div>

        {/* Filters */}
        <div className={styles.chatSearchFilters}>
          <div className={styles.chatSearchFilter}>
            <label className={styles.chatSearchFilterLabel}>Project:</label>
            <input
              type="text"
              className={styles.chatSearchFilterInput}
              placeholder="Filter by project..."
              value={searchState.filters.project}
              onChange={(e) => handleFilterChange('project', e.target.value)}
            />
          </div>
          <div className={styles.chatSearchFilter}>
            <label className={styles.chatSearchFilterLabel}>Model:</label>
            <input
              type="text"
              className={styles.chatSearchFilterInput}
              placeholder="e.g., claude-opus"
              value={searchState.filters.model}
              onChange={(e) => handleFilterChange('model', e.target.value)}
            />
          </div>
        </div>

        {/* Search status */}
        {searchState.loading && (
          <div className={styles.chatSearchStatus}>Searching...</div>
        )}
        {searchState.error && (
          <div className={styles.chatSearchError}>⚠️ {searchState.error}</div>
        )}

        {/* Search results */}
        <div className={styles.chatSearchResults}>
          {!searchState.loading && searchState.query && searchState.results.length === 0 && (
            <div className={styles.chatSearchNoResults}>
              No conversations found matching "{searchState.query}"
            </div>
          )}

          {searchState.results.map((result, idx) => (
            <button
              key={`${result.session_id}-${idx}`}
              className={styles.chatSearchResult}
              onClick={() => loadSearchResult(result)}
            >
              <div className={styles.chatSearchResultHeader}>
                <span className={styles.chatSearchResultProject}>
                  📁 {result.project_name}
                </span>
                <span className={styles.chatSearchResultModel}>
                  {result.model}
                </span>
                <span className={styles.chatSearchResultTime}>
                  {formatTime(result.timestamp)}
                </span>
              </div>
              <div className={styles.chatSearchResultSnippet}>
                {result.snippet}
              </div>
              <div className={styles.chatSearchResultMeta}>
                <span>{result.turn_count} turn{result.turn_count !== 1 ? 's' : ''}</span>
              </div>
            </button>
          ))}
        </div>

        {/* Search hint */}
        {!searchState.query && (
          <div className={styles.chatSearchHint}>
            <p>💡 Search tips:</p>
            <ul>
              <li>Search for function names, error messages, or topics</li>
              <li>Use filters to narrow by project or model</li>
              <li>Click a result to view the full conversation</li>
            </ul>
          </div>
        )}
      </div>
    );
  }

  // Render error state (only if no fallback data available)
  if (eventsError && !hasHierarchyData) {
    return (
      <div className={styles.chatContainer}>
        <div className={styles.chatError}>
          <div className={styles.chatErrorIcon}>⚠️</div>
          <div className={styles.chatErrorText}>{eventsError}</div>
        </div>
      </div>
    );
  }

  // ========================================
  // RENDER: Claude Code JSONL (HIGHEST PRIORITY - full conversation with thinking)
  // ========================================
  if (claudeCodeSession && claudeCodeSession.messages.length > 0) {
    return (
      <div className={styles.chatContainer}>
        {/* Header with search toggle */}
        <div className={styles.chatHeader}>
          <span className={styles.chatHeaderIcon}>{isTimeFiltered ? '🎯' : '🧠'}</span>
          <span className={styles.chatHeaderTitle}>
            {isTimeFiltered ? 'Span Context' : 'Claude Code Conversation'}
          </span>
          {isTimeFiltered && (
            <span className={styles.chatHeaderBadge}>
              Time-filtered
            </span>
          )}
          <span className={styles.chatHeaderTurnCount}>
            {claudeCodeSession.messages.length} message{claudeCodeSession.messages.length !== 1 ? 's' : ''}
          </span>
          <button
            className={styles.chatHeaderSearchBtn}
            onClick={toggleSearchMode}
            title="Search all conversations"
          >
            🔍
          </button>
        </div>

        {/* Session metadata bar */}
        <div className={styles.chatMetaBar}>
          <span className={styles.chatMetaItem}>
            <span className={styles.chatMetaLabel}>Model:</span> {claudeCodeSession.model}
          </span>
          <span className={styles.chatMetaItem}>
            <span className={styles.chatMetaLabel}>Project:</span> {claudeCodeSession.project_name}
          </span>
          <span className={styles.chatMetaItem}>
            <span className={styles.chatMetaLabel}>Tokens:</span>{' '}
            {(claudeCodeSession.total_in + claudeCodeSession.total_out).toLocaleString()}
          </span>
          {claudeCodeSession.cache_read > 0 && (
            <span className={styles.chatMetaItem}>
              <span className={styles.chatMetaLabel}>Cache:</span>{' '}
              {claudeCodeSession.cache_read.toLocaleString()} read
            </span>
          )}
          {spanTimeWindow && (
            <span className={styles.chatMetaItem}>
              <span className={styles.chatMetaLabel}>Window:</span>{' '}
              {formatTime(spanTimeWindow.start)} - {formatTime(spanTimeWindow.end)}
            </span>
          )}
        </div>

        {/* Messages */}
        <div className={styles.chatMessages}>
          {claudeCodeSession.messages.map((msg, idx) => (
            <div
              key={msg.uuid || idx}
              className={`${styles.chatMessage} ${styles[`chatMessage_${msg.type}`]}`}
            >
              {/* Message header */}
              <div className={styles.chatMessageHeader}>
                <span className={styles.chatMessageRole}>
                  {msg.type === 'user' ? '👤 User' : '🤖 Assistant'}
                </span>
                <span className={styles.chatMessageTime}>
                  {formatTime(msg.timestamp)}
                </span>
                {msg.usage && (
                  <span className={styles.chatMessageTokens}>
                    {msg.usage.input_tokens + msg.usage.output_tokens} tokens
                  </span>
                )}
              </div>

              {/* Content blocks */}
              <div className={styles.chatMessageContent}>
                {(msg.content || []).map((block, blockIdx) => {
                  const blockKey = `${msg.uuid || idx}-${blockIdx}`;

                  // Thinking block
                  if (block.type === 'thinking' && block.thinking) {
                    const preview = block.thinking.slice(0, 150);
                    const isExpanded = expandedThinking.has(blockKey);

                    return (
                      <div key={blockKey} className={styles.chatThinkingBlock}>
                        <button
                          className={styles.chatThinkingHeader}
                          onClick={() => toggleThinking(blockKey)}
                        >
                          <span className={styles.chatThinkingIcon}>💭</span>
                          <span className={styles.chatThinkingLabel}>Thinking</span>
                          <span className={styles.chatThinkingPreview}>
                            {!isExpanded && preview}
                            {!isExpanded && block.thinking.length > 150 && '...'}
                          </span>
                          <span className={styles.chatThinkingExpand}>
                            {isExpanded ? '▼' : '▶'}
                          </span>
                        </button>
                        {isExpanded && (
                          <pre className={styles.chatThinkingContent}>
                            {block.thinking}
                          </pre>
                        )}
                      </div>
                    );
                  }

                  // Text block
                  if (block.type === 'text' && block.text) {
                    return (
                      <div key={blockKey} className={styles.chatTextBlock}>
                        {block.text}
                      </div>
                    );
                  }

                  // Tool use block
                  if (block.type === 'tool_use' && block.tool_use) {
                    const isExpanded = expandedTools.has(blockKey);

                    return (
                      <div key={blockKey} className={styles.chatToolBlock}>
                        <button
                          className={styles.chatToolHeader}
                          onClick={() => toggleTool(blockKey)}
                        >
                          <span className={styles.chatToolIcon}>🔧</span>
                          <span className={styles.chatToolName}>{block.tool_use.name}</span>
                          <span className={styles.chatToolExpand}>
                            {isExpanded ? '▼' : '▶'}
                          </span>
                        </button>
                        {isExpanded && (
                          <div className={styles.chatToolDetails}>
                            <div className={styles.chatToolSection}>
                              <div className={styles.chatToolSectionLabel}>Input</div>
                              <pre className={styles.chatJsonContent}>
                                {JSON.stringify(block.tool_use.input, null, 2)}
                              </pre>
                            </div>
                          </div>
                        )}
                      </div>
                    );
                  }

                  // Tool result block
                  if (block.type === 'tool_result' && block.tool_result) {
                    const isError = block.tool_result.is_error;
                    const content = block.tool_result.content;
                    const preview = content.slice(0, 200);
                    const isExpanded = expandedTools.has(blockKey);

                    return (
                      <div
                        key={blockKey}
                        className={`${styles.chatToolResultBlock} ${isError ? styles.chatToolResultError : ''}`}
                      >
                        <button
                          className={styles.chatToolResultHeader}
                          onClick={() => toggleTool(blockKey)}
                        >
                          <span className={styles.chatToolResultIcon}>
                            {isError ? '❌' : '✓'}
                          </span>
                          <span className={styles.chatToolResultLabel}>
                            Tool Result
                          </span>
                          <span className={styles.chatToolResultPreview}>
                            {!isExpanded && preview}
                            {!isExpanded && content.length > 200 && '...'}
                          </span>
                          <span className={styles.chatToolExpand}>
                            {isExpanded ? '▼' : '▶'}
                          </span>
                        </button>
                        {isExpanded && (
                          <pre className={styles.chatToolResultContent}>
                            {content}
                          </pre>
                        )}
                      </div>
                    );
                  }

                  return null;
                })}
              </div>
            </div>
          ))}
        </div>

        {/* CLI hint for Claude Code session */}
        <div className={styles.chatCliHint}>
          <span className={styles.chatCliLabel}>Session ID:</span>
          <code className={styles.chatCliCommand}>{claudeCodeSession.id}</code>
          <button
            className={styles.chatCliCopy}
            onClick={() => {
              navigator.clipboard.writeText(claudeCodeSession.id);
            }}
            title="Copy session ID"
          >
            ⎘
          </button>
        </div>
      </div>
    );
  }

  // ========================================
  // RENDER: Coordinator task events (SECOND PRIORITY - has conversation text)
  // ========================================
  if (coordinatorTurns.length > 0) {
    return (
      <div className={styles.chatContainer}>
        {/* Header */}
        <div className={styles.chatHeader}>
          <span className={styles.chatHeaderIcon}>💬</span>
          <span className={styles.chatHeaderTitle}>
            {selectedNode?.label || 'Task Conversation'}
          </span>
          <span className={styles.chatHeaderTurnCount}>
            {coordinatorTurns.length} turn{coordinatorTurns.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Turns from coordinator */}
        <div className={styles.chatTurns}>
          {coordinatorTurns.map(turn => (
            <div key={turn.turnNumber} className={styles.chatTurn}>
              <div className={styles.chatTurnHeader}>
                <span className={styles.chatTurnNumber}>Turn {turn.turnNumber}</span>
                <span className={styles.chatTurnTime}>{formatTime(turn.startTime)}</span>
              </div>

              <div className={styles.chatTurnEvents}>
                {turn.events.map(event => (
                  <div
                    key={event.id}
                    className={`${styles.chatEvent} ${styles[`chatEvent_${event.stream_type}`]}`}
                  >
                    {event.stream_type === 'text' && event.text && (
                      <div className={styles.chatTextEvent}>
                        <div className={styles.chatTextContent}>{event.text}</div>
                      </div>
                    )}

                    {event.stream_type === 'tool_use' && (
                      <div className={styles.chatToolEvent}>
                        <button
                          className={styles.chatToolHeader}
                          onClick={() => toggleTool(event.id)}
                        >
                          <span className={styles.chatToolIcon}>🔧</span>
                          <span className={styles.chatToolName}>{event.tool_name || 'Tool'}</span>
                          <span className={styles.chatToolExpand}>
                            {expandedTools.has(event.id) ? '▼' : '▶'}
                          </span>
                        </button>

                        {expandedTools.has(event.id) && (
                          <div className={styles.chatToolDetails}>
                            {event.tool_input && (
                              <div className={styles.chatToolSection}>
                                <div className={styles.chatToolSectionLabel}>Input</div>
                                {formatJson(event.tool_input)}
                              </div>
                            )}
                            {event.tool_output && (
                              <div className={styles.chatToolSection}>
                                <div className={styles.chatToolSectionLabel}>Output</div>
                                {formatJson(event.tool_output)}
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    )}

                    {event.stream_type === 'tool_result' && (
                      <div className={styles.chatToolResultEvent}>
                        <span className={styles.chatToolResultIcon}>✓</span>
                        <span className={styles.chatToolResultText}>
                          {event.tool_name ? `${event.tool_name} completed` : 'Tool completed'}
                        </span>
                      </div>
                    )}

                    {event.stream_type === 'error' && (
                      <div className={styles.chatErrorEvent}>
                        <span className={styles.chatErrorEventIcon}>❌</span>
                        <span className={styles.chatErrorEventText}>
                          {event.error_msg || event.text || 'Error occurred'}
                        </span>
                      </div>
                    )}

                    {event.stream_type === 'status' && (
                      <div className={styles.chatStatusEvent}>
                        <span className={styles.chatStatusEventIcon}>ℹ️</span>
                        <span className={styles.chatStatusEventText}>{event.text}</span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* CLI hint for coordinator task */}
        {taskId && (
          <div className={styles.chatCliHint}>
            <span className={styles.chatCliLabel}>CLI:</span>
            <code className={styles.chatCliCommand}>
              ailang coordinator logs {taskId} --limit 100
            </code>
            <button
              className={styles.chatCliCopy}
              onClick={() => {
                navigator.clipboard.writeText(`ailang coordinator logs ${taskId} --limit 100`);
              }}
              title="Copy full command"
            >
              ⎘
            </button>
          </div>
        )}
      </div>
    );
  }

  // ========================================
  // RENDER: OTEL hierarchy-based conversation (fallback for non-coordinator tasks)
  // ========================================
  if (hasHierarchyData) {
    return (
      <div className={styles.chatContainer}>
        {/* Header */}
        <div className={styles.chatHeader}>
          <span className={styles.chatHeaderIcon}>💬</span>
          <span className={styles.chatHeaderTitle}>
            {selectedNode?.label || 'Session Conversation'}
          </span>
          <span className={styles.chatHeaderTurnCount}>
            {hierarchyTurns.length} turn{hierarchyTurns.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Turns from hierarchy */}
        <div className={styles.chatTurns}>
          {hierarchyTurns.map((turn, idx) => (
            <div key={turn.node.id || idx} className={styles.chatTurn}>
              {/* Turn header */}
              <div className={styles.chatTurnHeader}>
                <span className={styles.chatTurnNumber}>Turn {turn.turnNumber}</span>
                <span className={styles.chatTurnTime}>
                  {turn.node.startTime ? formatTime(turn.node.startTime) : '—'}
                </span>
                {turn.node.durationMs && (
                  <span className={styles.chatTurnDuration}>
                    {formatDuration(turn.node.durationMs)}
                  </span>
                )}
              </div>

              {/* Tool uses in this turn */}
              <div className={styles.chatTurnEvents}>
                {turn.toolUses.length === 0 ? (
                  <div className={styles.chatTextEvent}>
                    <div className={styles.chatTextContent}>
                      {turn.node.label || 'Processing...'}
                    </div>
                  </div>
                ) : (
                  turn.toolUses.map(tool => (
                    <div key={tool.id} className={styles.chatEvent}>
                      <div className={styles.chatToolEvent}>
                        <button
                          className={styles.chatToolHeader}
                          onClick={() => toggleTool(tool.id)}
                        >
                          <span className={styles.chatToolIcon}>🔧</span>
                          <span className={styles.chatToolName}>
                            {tool.label || tool._span?.name || 'Tool'}
                          </span>
                          {tool.durationMs && (
                            <span className={styles.chatToolDuration}>
                              {formatDuration(tool.durationMs)}
                            </span>
                          )}
                          <span className={`${styles.chatToolStatus} ${styles[`status_${tool.status}`]}`}>
                            {tool.status === 'completed' ? '✓' : tool.status === 'error' ? '✗' : '•'}
                          </span>
                          <span className={styles.chatToolExpand}>
                            {expandedTools.has(tool.id) ? '▼' : '▶'}
                          </span>
                        </button>

                        {expandedTools.has(tool.id) && tool._span?.attributes && (
                          <div className={styles.chatToolDetails}>
                            <div className={styles.chatToolSection}>
                              <div className={styles.chatToolSectionLabel}>Attributes</div>
                              <div className={styles.chatToolAttributes}>
                                {Object.entries(tool._span.attributes).map(([key, value]) => (
                                  <div key={key} className={styles.chatMetaRow}>
                                    <span className={styles.chatMetaLabel}>{key}</span>
                                    <span className={styles.chatMetaValue}>{value}</span>
                                  </div>
                                ))}
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          ))}
        </div>

        {/* CLI hint for OTEL trace */}
        {selectedSpan && (
          <div className={styles.chatCliHint}>
            <span className={styles.chatCliLabel}>CLI:</span>
            <code className={styles.chatCliCommand}>
              ailang trace view {selectedSpan.id}
            </code>
            <button
              className={styles.chatCliCopy}
              onClick={() => {
                navigator.clipboard.writeText(`ailang trace view ${selectedSpan.id}`);
              }}
              title="Copy full command"
            >
              ⎘
            </button>
          </div>
        )}
      </div>
    );
  }

  // ========================================
  // RENDER: Fallback for spans without conversation data
  // ========================================
  return (
    <div className={styles.chatContainer}>
      <div className={styles.chatFallback}>
        <div className={styles.chatFallbackHeader}>
          <span className={styles.chatFallbackIcon}>📋</span>
          <span className={styles.chatFallbackTitle}>
            {selectedNode?.label || selectedSpan?.display_name || selectedSpan?.name || 'Span Details'}
          </span>
        </div>

        {selectedNode && (
          <div className={styles.chatFallbackMeta}>
            <div className={styles.chatMetaRow}>
              <span className={styles.chatMetaLabel}>Type</span>
              <span className={styles.chatMetaValue}>{selectedNode.type}</span>
            </div>
            <div className={styles.chatMetaRow}>
              <span className={styles.chatMetaLabel}>Status</span>
              <span className={`${styles.chatMetaValue} ${styles[`status_${selectedNode.status}`]}`}>
                {selectedNode.status}
              </span>
            </div>
            {selectedNode.durationMs !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Duration</span>
                <span className={styles.chatMetaValue}>{formatDuration(selectedNode.durationMs)}</span>
              </div>
            )}
            {selectedNode.startTime && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Started</span>
                <span className={styles.chatMetaValue}>
                  {new Date(selectedNode.startTime).toLocaleString()}
                </span>
              </div>
            )}
            {selectedNode.cost !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Cost</span>
                <span className={styles.chatMetaValue}>${selectedNode.cost.toFixed(4)}</span>
              </div>
            )}
            {selectedNode.tokensIn !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Tokens In</span>
                <span className={styles.chatMetaValue}>{selectedNode.tokensIn.toLocaleString()}</span>
              </div>
            )}
            {selectedNode.tokensOut !== undefined && (
              <div className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>Tokens Out</span>
                <span className={styles.chatMetaValue}>{selectedNode.tokensOut.toLocaleString()}</span>
              </div>
            )}
          </div>
        )}

        {selectedSpan?.attributes && Object.keys(selectedSpan.attributes).length > 0 && (
          <div className={styles.chatFallbackAttributes}>
            <div className={styles.chatAttributesHeader}>Attributes</div>
            {Object.entries(selectedSpan.attributes).map(([key, value]) => (
              <div key={key} className={styles.chatMetaRow}>
                <span className={styles.chatMetaLabel}>{key}</span>
                <span className={styles.chatMetaValue}>{value}</span>
              </div>
            ))}
          </div>
        )}

        <div className={styles.chatFallbackHint}>
          No conversation data available for this span.
          {selectedNode?.children && selectedNode.children.length > 0 && (
            <span> This span has {selectedNode.children.length} child span(s) - try expanding in tree view.</span>
          )}
        </div>
      </div>
    </div>
  );
};

export default ChatHistory;
