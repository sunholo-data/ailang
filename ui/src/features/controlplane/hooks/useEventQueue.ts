/**
 * Hook for subscribing to live events via WebSocket AND fetching historical events
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import type { ControlPlaneFilters } from '../types';

export interface EventMessage {
  id: string;
  timestamp: string;
  type: 'task_start' | 'task_complete' | 'task_error' | 'handoff' | 'approval' | 'message';
  source: string;
  target?: string;
  content: string;
  metadata?: Record<string, unknown>;
}

// Inbox message from the API
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
}

// Observatory task from the API
interface ObservatoryTask {
  id: string;
  workspace_id: string;
  title: string;
  description: string;
  source_type: string;
  status: string;
  priority: string;
  created_at: string;
  total_duration_ms: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_usd: number;
  span_count: number;
  error_count: number;
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
  const reconnectTimeoutRef = useRef<number | null>(null);

  // Fetch historical events from /api/inbox AND observatory tasks
  const fetchHistoricalEvents = useCallback(async () => {
    setLoading(true);
    try {
      // Fetch inbox messages
      const inboxParams = new URLSearchParams();
      if (filters?.source_type) {
        inboxParams.set('inbox', filters.source_type);
      }
      const inboxUrl = `/api/inbox${inboxParams.toString() ? '?' + inboxParams.toString() : ''}`;

      // Fetch observatory tasks (these have real trace data)
      const tasksUrl = '/api/observatory/tasks?limit=50';

      const [inboxResponse, tasksResponse] = await Promise.all([
        fetch(inboxUrl),
        fetch(tasksUrl),
      ]);

      const inboxEvents: EventMessage[] = [];
      const taskEvents: EventMessage[] = [];

      // Process inbox messages
      if (inboxResponse.ok) {
        const data = await inboxResponse.json();
        const messages: InboxMessage[] = data.messages || [];
        messages.forEach((msg) => {
          inboxEvents.push({
            id: msg.id,
            timestamp: msg.created_at,
            type: mapMessageTypeToEventType(msg.message_type),
            source: msg.from_agent || 'unknown',
            target: msg.to_inbox,
            content: msg.title || msg.payload,
            metadata: {
              payload: msg.payload,
              status: msg.status,
              correlation_id: msg.correlation_id,
              parent_task_id: msg.parent_task_id,
              message_type: msg.message_type,
            },
          });
        });
      }

      // Process observatory tasks (these have span data!)
      if (tasksResponse.ok) {
        const tasks: ObservatoryTask[] = await tasksResponse.json();
        tasks.forEach((task) => {
          taskEvents.push({
            id: task.id,  // task.id is the task_id for spans lookup
            timestamp: task.created_at,
            type: mapTaskStatusToEventType(task.status),
            source: task.source_type || 'coordinator',
            target: task.workspace_id,
            content: task.title,
            metadata: {
              task_id: task.id,  // This is the key for span lookup!
              workspace_id: task.workspace_id,
              status: task.status,
              priority: task.priority,
              total_duration_ms: task.total_duration_ms,
              total_tokens_in: task.total_tokens_in,
              total_tokens_out: task.total_tokens_out,
              total_cost_usd: task.total_cost_usd,
              span_count: task.span_count,
              error_count: task.error_count,
            },
          });
        });
      }

      // Combine inbox and task events
      setHistoricalEvents([...inboxEvents, ...taskEvents]);
      setError(null);
    } catch (err) {
      console.error('[EventQueue] Failed to fetch historical events:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch events');
    } finally {
      setLoading(false);
    }
  }, [filters?.source_type]);

  // Fetch historical events on mount and when filters change
  useEffect(() => {
    fetchHistoricalEvents();
  }, [fetchHistoricalEvents]);

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

          // Handle different message types
          if (data.type === 'task_stream_event') {
            const taskEvent = data.payload;
            const newEvent: EventMessage = {
              id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
              timestamp: new Date().toISOString(),
              type: mapStreamTypeToEventType(taskEvent.stream_type),
              source: taskEvent.task_id || 'coordinator',
              target: undefined,
              content: formatEventContent(taskEvent),
              metadata: taskEvent,
            };

            setEvents((prev) => {
              const updated = [newEvent, ...prev];
              return updated.slice(0, maxEvents);
            });
          } else if (data.type === 'inbox_message') {
            // Handle new inbox message via WebSocket
            const msg = data.payload as InboxMessage;
            const newEvent: EventMessage = {
              id: msg.id,
              timestamp: msg.created_at,
              type: mapMessageTypeToEventType(msg.message_type),
              source: msg.from_agent || 'unknown',
              target: msg.to_inbox,
              content: msg.title || msg.payload,
              metadata: {
                payload: msg.payload,
                status: msg.status,
                correlation_id: msg.correlation_id,
                message_type: msg.message_type,
              },
            };

            setEvents((prev) => {
              const updated = [newEvent, ...prev];
              return updated.slice(0, maxEvents);
            });
          }
        } catch (err) {
          console.error('[EventQueue] Failed to parse message:', err);
        }
      };

      ws.onclose = () => {
        setConnected(false);
        console.log('[EventQueue] WebSocket closed, reconnecting in 3s...');
        reconnectTimeoutRef.current = window.setTimeout(() => {
          connect();
        }, 3000);
      };

      ws.onerror = (err) => {
        console.error('[EventQueue] WebSocket error:', err);
        setError('WebSocket connection error');
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to connect');
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

  // Combine real-time events with historical events
  const allEvents = [...events, ...historicalEvents]
    .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
    .slice(0, maxEvents);

  return { events: allEvents, connected, loading, error, clearEvents, refetch: fetchHistoricalEvents };
}

// Helper to map message types to event types
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
    case 'notification':
    case 'info':
    default:
      return 'message';
  }
}

// Helper to map task status to event types
function mapTaskStatusToEventType(status: string): EventMessage['type'] {
  switch (status) {
    case 'pending':
    case 'running':
      return 'task_start';
    case 'completed':
    case 'complete':
      return 'task_complete';
    case 'failed':
    case 'error':
      return 'task_error';
    case 'needs_approval':
      return 'approval';
    default:
      return 'message';
  }
}

// Helper to map stream types to event types
function mapStreamTypeToEventType(
  streamType: string
): EventMessage['type'] {
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

// Helper to format event content
function formatEventContent(event: Record<string, unknown>): string {
  if (event.text) return String(event.text);
  if (event.tool_name) return `Tool: ${event.tool_name}`;
  if (event.status) return `Status: ${event.status}`;
  if (event.error_msg) return `Error: ${event.error_msg}`;
  return JSON.stringify(event);
}
