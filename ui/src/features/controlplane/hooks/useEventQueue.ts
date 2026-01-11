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
}

interface UseEventQueueOptions {
  maxEvents?: number;
  wsUrl?: string;
  filters?: ControlPlaneFilters;
}

export function useEventQueue(options: UseEventQueueOptions = {}) {
  const { maxEvents = 100, wsUrl, filters } = options;
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
        params.set('inbox', filters.source_type);
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
      params.set('limit', String(maxEvents));

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
        type: mapMessageTypeToEventType(msg.message_type),
        source: msg.from_agent || 'unknown',
        target: msg.to_inbox,
        content: msg.title || msg.payload,
        // Extended fields for correlation
        from_agent: msg.from_agent,
        to_inbox: msg.to_inbox,
        inbox: msg.to_inbox,
        task_id: msg.task_id || msg.parent_task_id || msg.correlation_id,
        metadata: {
          payload: msg.payload,
          status: msg.status,
          message_type: msg.message_type,
          correlation_id: msg.correlation_id,
          parent_task_id: msg.parent_task_id,
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
  }, [filters?.source_type, filters?.provider, filters?.model, filters?.workspace, filters?.status, filters?.start_date, filters?.end_date, maxEvents]);

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
              type: mapMessageTypeToEventType(msg.message_type),
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
              return updated.slice(0, maxEvents);
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
              metadata: taskEvent,
            };

            setEvents((prev) => {
              const updated = [newEvent, ...prev];
              return updated.slice(0, maxEvents);
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

  // Combine real-time events with historical, apply filters, sort by timestamp
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
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, maxEvents);

  return {
    events: allEvents,
    connected,
    loading,
    error,
    clearEvents,
    refetch: fetchMessages,
  };
}

// Map message types to event types
function mapMessageTypeToEventType(msgType: string): EventMessage['type'] {
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
    default:
      return 'message';
  }
}

// Map stream types to event types
function mapStreamTypeToEventType(streamType: string): EventMessage['type'] {
  switch (streamType) {
    case 'task_started':
      return 'task_start';
    case 'task_completed':
      return 'task_complete';
    case 'task_failed':
    case 'task_error':
      return 'task_error';
    case 'handoff':
      return 'handoff';
    case 'approval_request':
      return 'approval';
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
