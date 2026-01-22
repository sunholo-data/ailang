/**
 * Hook for fetching events from the AILANG messages table
 *
 * Messages are the SINGLE SOURCE OF TRUTH for events.
 * Everything flows through `ailang messages` - coordinator, evals, user interaction, external projects.
 *
 * NO mixing with other data sources - just messages.
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { ControlPlaneFilters } from '../types';

export interface EventMessage {
  id: string;
  timestamp: string;
  type: 'task_start' | 'task_complete' | 'task_error' | 'handoff' | 'approval' | 'message';
  source: string;       // from_agent
  target?: string;      // to_inbox
  content: string;      // title
  // Extended fields for correlation with topology
  from_agent?: string;
  to_inbox?: string;
  inbox?: string;       // alias for to_inbox (compatibility)
  task_id?: string;     // correlation with exec hierarchy
  // Context fields for task identification (dashboard event queue)
  workspace?: string;         // Working directory path
  directive?: string;         // Initial user prompt (truncated)
  directive_full?: string;    // Full directive (for detail views)
  agent_id?: string;          // Agent identifier (e.g., "design-doc-creator")
  source_type?: string;       // Event source: coordinator, eval, github, direct
  // Sorting fields (from Claude Code events)
  turn_count?: number;        // Number of turns in session
  cost_usd?: number;          // AI cost in USD
  tokens_in?: number;         // Input tokens
  tokens_out?: number;        // Output tokens
  duration_ms?: number;       // Execution duration in ms
  metadata?: Record<string, unknown>;
}

// Message from the inbox API
interface InboxMessage {
  id: string;
  to_inbox: string;
  from_agent: string;
  title: string;
  payload: string;
  message_type: string;
  status: string;
  created_at: string;
  read_at?: string;
  correlation_id?: string;
  parent_task_id?: string;
  task_id?: string;
  // Dimension fields for filtering and highlighting
  workspace?: string;
  provider?: string;
  model?: string;
  source_type?: string;
  // Task context fields (from TaskStreamEvent enrichment)
  directive?: string;
  directive_full?: string;
  agent_id?: string;
  // Sorting fields (from Claude Code events)
  turn_count?: number;
  cost_usd?: number;
  tokens_in?: number;
  tokens_out?: number;
  duration_ms?: number;
}

interface UseEventQueueOptions {
  maxEvents?: number;
  wsUrl?: string;
  filters?: ControlPlaneFilters;
}

export function useEventQueue(options: UseEventQueueOptions = {}) {
  const { maxEvents = 0, wsUrl, filters } = options; // 0 = no limit
  const [events, setEvents] = useState<EventMessage[]>([]);
  const [historicalEvents, setHistoricalEvents] = useState<EventMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetch messages from /api/inbox - SINGLE SOURCE OF TRUTH
  const fetchMessages = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (filters?.source_type) {
        params.set('source_type', filters.source_type);
      }
      // Pass all filters to server for server-side filtering
      if (filters?.provider) {
        params.set('provider', filters.provider);
      }
      if (filters?.model) {
        params.set('model', filters.model);
      }
      if (filters?.workspace) {
        params.set('workspace', filters.workspace);
      }
      if (filters?.status) {
        params.set('status', filters.status);
      }
      if (filters?.start_date) {
        params.set('start_date', filters.start_date);
      }
      if (filters?.end_date) {
        params.set('end_date', filters.end_date);
      }
      // Sorting parameters
      if (filters?.sort) {
        params.set('sort', filters.sort);
      }
      if (filters?.order) {
        params.set('order', filters.order);
      }
      // Only set limit if maxEvents > 0 (0 = no limit)
      if (maxEvents > 0) {
        params.set('limit', String(maxEvents));
      }

      const url = `/api/inbox${params.toString() ? '?' + params.toString() : ''}`;
      const response = await fetch(url);

      if (!response.ok) {
        throw new Error(`HTTP error: ${response.status}`);
      }

      const data = await response.json();
      const messages: InboxMessage[] = data.messages || [];

      // Transform messages to EventMessage format
      const transformed: EventMessage[] = messages.map((msg) => ({
        id: msg.id,
        timestamp: msg.created_at,
        type: mapMessageTypeToEventType(msg.message_type, msg.title),
        source: msg.from_agent || 'unknown',
        target: msg.to_inbox,
        content: msg.title || msg.payload,
        // Extended fields for correlation
        from_agent: msg.from_agent,
        to_inbox: msg.to_inbox,
        inbox: msg.to_inbox,
        task_id: msg.task_id || msg.parent_task_id || msg.correlation_id,
        // Context fields for task identification
        workspace: msg.workspace,
        directive: msg.directive,
        directive_full: msg.directive_full,
        agent_id: msg.agent_id,
        source_type: msg.source_type,
        // Sorting fields
        turn_count: msg.turn_count,
        cost_usd: msg.cost_usd,
        tokens_in: msg.tokens_in,
        tokens_out: msg.tokens_out,
        duration_ms: msg.duration_ms,
        metadata: {
          payload: msg.payload,
          status: msg.status,
          message_type: msg.message_type,
          correlation_id: msg.correlation_id,
          parent_task_id: msg.parent_task_id,
          // Dimension fields for filtering and highlighting
          workspace: msg.workspace,
          provider: msg.provider,
          model: msg.model,
          source_type: msg.source_type,
        },
      }));

      setHistoricalEvents(transformed);
      setError(null);
    } catch (err) {
      // NO SILENT FALLBACKS - show the error
      console.error('[EventQueue] Failed to fetch messages:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch messages');
    } finally {
      setLoading(false);
    }
  }, [filters?.source_type, filters?.provider, filters?.model, filters?.workspace, filters?.status, filters?.start_date, filters?.end_date, filters?.sort, filters?.order, maxEvents]);

  // Fetch on mount and when filters change
  useEffect(() => {
    fetchMessages();
  }, [fetchMessages]);

  // WebSocket for real-time updates
  const connect = useCallback(() => {
    const url = wsUrl || `ws://${window.location.host}/ws`;

    try {
      const ws = new WebSocket(url);
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        setError(null);
        console.log('[EventQueue] WebSocket connected');
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          // Handle inbox_message events (new messages)
          if (data.type === 'inbox_message') {
            const msg = data.payload as InboxMessage;
            const newEvent: EventMessage = {
              id: msg.id,
              timestamp: msg.created_at || new Date().toISOString(),
              type: mapMessageTypeToEventType(msg.message_type, msg.title),
              source: msg.from_agent || 'unknown',
              target: msg.to_inbox,
              content: msg.title || msg.payload,
              from_agent: msg.from_agent,
              to_inbox: msg.to_inbox,
              inbox: msg.to_inbox,
              task_id: msg.task_id || msg.parent_task_id,
              metadata: {
                payload: msg.payload,
                status: msg.status,
                message_type: msg.message_type,
              },
            };

            setEvents((prev) => {
              const updated = [newEvent, ...prev];
              return maxEvents > 0 ? updated.slice(0, maxEvents) : updated;
            });
          }
          // Also handle task_stream_event for live coordinator updates
          else if (data.type === 'task_stream_event') {
            const taskEvent = data.payload;
            const newEvent: EventMessage = {
              id: `stream-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
              timestamp: new Date().toISOString(),
              type: mapStreamTypeToEventType(taskEvent.stream_type),
              source: taskEvent.agent_id || 'coordinator',
              target: undefined,
              content: formatStreamContent(taskEvent),
              task_id: taskEvent.task_id,
              // Context fields for task identification
              workspace: taskEvent.workspace,
              directive: taskEvent.directive,
              directive_full: taskEvent.directive_full,
              agent_id: taskEvent.agent_id,
              source_type: taskEvent.source_type || 'coordinator',
              metadata: taskEvent,
            };

            setEvents((prev) => {
              const updated = [newEvent, ...prev];
              return maxEvents > 0 ? updated.slice(0, maxEvents) : updated;
            });
          }
        } catch (err) {
          console.error('[EventQueue] Failed to parse WebSocket message:', err);
        }
      };

      ws.onclose = () => {
        setConnected(false);
        console.log('[EventQueue] WebSocket closed, reconnecting in 3s...');
        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, 3000);
      };

      ws.onerror = (err) => {
        console.error('[EventQueue] WebSocket error:', err);
        setError('WebSocket connection error');
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect to WebSocket');
    }
  }, [wsUrl, maxEvents]);

  useEffect(() => {
    connect();

    return () => {
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
    };
  }, [connect]);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  // Sort function that respects filter settings
  const sortEvents = (a: EventMessage, b: EventMessage): number => {
    const sortBy = filters?.sort || 'timestamp';
    const descending = filters?.order !== 'asc';

    let less: boolean;
    switch (sortBy) {
      case 'turns':
        less = (a.turn_count || 0) < (b.turn_count || 0);
        break;
      case 'cost':
        less = (a.cost_usd || 0) < (b.cost_usd || 0);
        break;
      case 'tokens': {
        const totalA = (a.tokens_in || 0) + (a.tokens_out || 0);
        const totalB = (b.tokens_in || 0) + (b.tokens_out || 0);
        less = totalA < totalB;
        break;
      }
      case 'duration':
        less = (a.duration_ms || 0) < (b.duration_ms || 0);
        break;
      default: // timestamp
        less = new Date(a.timestamp).getTime() < new Date(b.timestamp).getTime();
    }
    return descending ? (less ? 1 : -1) : (less ? -1 : 1);
  };

  // Combine real-time events with historical, apply filters and sort
  const allEvents = [...events, ...historicalEvents]
    .filter((event) => {
      // Date range filter
      if (filters?.start_date && filters?.end_date) {
        const eventDate = event.timestamp.split('T')[0];
        if (eventDate < filters.start_date || eventDate > filters.end_date) {
          return false;
        }
      }
      // Status filter
      if (filters?.status && filters.status !== 'all') {
        if (filters.status === 'completed' && event.type !== 'task_complete') return false;
        if (filters.status === 'failed' && event.type !== 'task_error') return false;
        if (filters.status === 'pending' && event.type !== 'task_start' && event.type !== 'approval') return false;
      }
      // Search filter
      if (filters?.search) {
        const searchLower = filters.search.toLowerCase();
        const matches = [event.content, event.source, event.target, event.from_agent, event.to_inbox]
          .filter(Boolean)
          .some((val) => val?.toLowerCase().includes(searchLower));
        if (!matches) return false;
      }
      return true;
    })
    .sort(sortEvents);

  // Only apply limit if maxEvents > 0
  const finalEvents = maxEvents > 0 ? allEvents.slice(0, maxEvents) : allEvents;

  return {
    events: finalEvents,
    connected,
    loading,
    error,
    clearEvents,
    refetch: fetchMessages,
  };
}

// Map message types to event types, considering title patterns for notifications
function mapMessageTypeToEventType(msgType: string, title?: string): EventMessage['type'] {
  // First check explicit message types
  switch (msgType) {
    case 'task':
    case 'task_request':
      return 'task_start';
    case 'task_result':
    case 'result':
      return 'task_complete';
    case 'error':
      return 'task_error';
    case 'handoff':
    case 'agent_handoff':
      return 'handoff';
    case 'approval':
    case 'approval_request':
      return 'approval';
  }

  // For 'notification' type, infer from title patterns
  if (msgType === 'notification' && title) {
    const lowerTitle = title.toLowerCase();

    if (lowerTitle.startsWith('approval required:')) {
      return 'approval';
    }
    if (lowerTitle.startsWith('sprint ready:') || lowerTitle.includes('handoff')) {
      return 'handoff';
    }
    if (lowerTitle.startsWith('task completed:') || lowerTitle.startsWith('completed:')) {
      return 'task_complete';
    }
    if (lowerTitle.startsWith('task failed:') || lowerTitle.startsWith('error:')) {
      return 'task_error';
    }
    if (lowerTitle.startsWith('feature:') || lowerTitle.startsWith('bug:')) {
      return 'task_start';
    }
  }

  return 'message';
}

// Map stream types to event types
function mapStreamTypeToEventType(streamType: string): EventMessage['type'] {
  switch (streamType) {
    // Task lifecycle
    case 'task_started':
    case 'turn_start':        // Turn start = task activity
      return 'task_start';
    case 'task_completed':
    case 'turn_end':          // Turn end = completion signal
      return 'task_complete';
    case 'task_failed':
    case 'task_error':
    case 'error':             // Generic error events
      return 'task_error';
    // Coordination
    case 'handoff':
      return 'handoff';
    case 'approval_request':
    case 'human_approval':      // Human approved a task
      return 'approval';
    // Execution details → generic message
    case 'text':
    case 'tool_use':
    case 'tool_result':
    case 'status':
    case 'human_feedback':      // Human provided feedback
    case 'iteration_start':     // New iteration started
    default:
      return 'message';
  }
}

// Format stream event content
function formatStreamContent(event: Record<string, unknown>): string {
  if (event.text) return String(event.text);
  if (event.tool_name) return `Tool: ${event.tool_name}`;
  if (event.status) return `Status: ${event.status}`;
  if (event.error_msg) return `Error: ${event.error_msg}`;
  return JSON.stringify(event);
}
