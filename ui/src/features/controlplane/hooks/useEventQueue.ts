/**
 * Hook for subscribing to live events via WebSocket
 */
import { useState, useEffect, useCallback, useRef } from 'react';

export interface EventMessage {
  id: string;
  timestamp: string;
  type: 'task_start' | 'task_complete' | 'task_error' | 'handoff' | 'approval' | 'message';
  source: string;
  target?: string;
  content: string;
  metadata?: Record<string, unknown>;
}

interface UseEventQueueOptions {
  maxEvents?: number;
  wsUrl?: string;
}

export function useEventQueue(options: UseEventQueueOptions = {}) {
  const { maxEvents = 100, wsUrl } = options;
  const [events, setEvents] = useState<EventMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);

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

  return { events, connected, error, clearEvents };
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
