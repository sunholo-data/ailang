import { useEffect, useState, useCallback, useRef } from 'react';
import { TaskStreamEvent, TaskResourceMetrics, PendingApprovalRequest } from '../types';

interface UseTaskStreamOptions {
  taskId: string;
  onEvent?: (event: TaskStreamEvent) => void;
}

interface TaskStreamState {
  events: TaskStreamEvent[];
  metrics: TaskResourceMetrics | null;
  status: string;
  pendingApproval: PendingApprovalRequest | null;
  isConnected: boolean;
  error: string | null;
}

export const useTaskStream = ({ taskId, onEvent }: UseTaskStreamOptions) => {
  const [state, setState] = useState<TaskStreamState>({
    events: [],
    metrics: null,
    status: 'pending',
    pendingApproval: null,
    isConnected: false,
    error: null,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const onEventRef = useRef(onEvent);
  const reconnectTimeoutRef = useRef<number | null>(null);

  // Update callback ref
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  // Connect to task stream via the main WebSocket endpoint
  // Events are broadcast to all clients, we filter by taskId
  const connect = useCallback(() => {
    if (!taskId) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // Use the main /ws endpoint - events are broadcast to all clients
    const wsUrl = `${protocol}//${window.location.host}/ws`;

    try {
      const ws = new WebSocket(wsUrl);
      wsRef.current = ws;

      ws.onopen = () => {
        setState(prev => ({ ...prev, isConnected: true, error: null }));
      };

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);

          if (data.type === 'task_stream') {
            const streamEvent = data as TaskStreamEvent;

            // Filter by taskId - only process events for our task
            if (streamEvent.task_id !== taskId) {
              return;
            }

            // Update events list
            setState(prev => ({
              ...prev,
              events: [...prev.events, streamEvent].slice(-500), // Keep last 500 events
            }));

            // Update metrics if present
            if (streamEvent.event_type === 'metrics') {
              setState(prev => ({
                ...prev,
                metrics: {
                  task_id: streamEvent.task_id,
                  cpu_percent: streamEvent.cpu_percent || 0,
                  memory_mb: streamEvent.memory_mb || 0,
                  tokens_in: streamEvent.tokens_in || 0,
                  tokens_out: streamEvent.tokens_out || 0,
                  cost: streamEvent.cost || 0,
                  peak_cpu: Math.max(prev.metrics?.peak_cpu || 0, streamEvent.cpu_percent || 0),
                  peak_memory: Math.max(prev.metrics?.peak_memory || 0, streamEvent.memory_mb || 0),
                  updated_at: streamEvent.timestamp,
                },
              }));
            }

            // Update status if present
            if (streamEvent.event_type === 'status' && streamEvent.status) {
              setState(prev => ({ ...prev, status: streamEvent.status! }));
            }

            // Call external handler
            if (onEventRef.current) {
              onEventRef.current(streamEvent);
            }
          } else if (data.type === 'approval_request') {
            setState(prev => ({
              ...prev,
              pendingApproval: data as PendingApprovalRequest,
              status: 'awaiting_approval',
            }));
          } else if (data.type === 'approval_resolved') {
            setState(prev => ({
              ...prev,
              pendingApproval: null,
              status: data.new_status || 'running',
            }));
          }
        } catch (err) {
          console.error('Failed to parse task stream event:', err);
        }
      };

      ws.onerror = (error) => {
        console.error('Task stream WebSocket error:', error);
        setState(prev => ({ ...prev, error: 'Connection error' }));
      };

      ws.onclose = (event) => {
        setState(prev => ({ ...prev, isConnected: false }));
        wsRef.current = null;

        // Reconnect if not a normal close
        if (event.code !== 1000 && event.code !== 1001) {
          reconnectTimeoutRef.current = window.setTimeout(() => {
            connect();
          }, 2000);
        }
      };
    } catch (err) {
      console.error('Failed to connect to task stream:', err);
      setState(prev => ({ ...prev, error: 'Failed to connect' }));
    }
  }, [taskId]);

  // Disconnect
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close(1000, 'Client disconnected');
      wsRef.current = null;
    }
    setState(prev => ({ ...prev, isConnected: false }));
  }, []);

  // Connect on mount, disconnect on unmount
  useEffect(() => {
    connect();
    return () => disconnect();
  }, [connect, disconnect]);

  // Clear events
  const clearEvents = useCallback(() => {
    setState(prev => ({ ...prev, events: [] }));
  }, []);

  // Approve pending request
  const approve = useCallback(async () => {
    if (!state.pendingApproval) return;

    try {
      const response = await fetch(`/api/coordinator/approve/${state.pendingApproval.id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(await response.text());
      }

      setState(prev => ({
        ...prev,
        pendingApproval: null,
        status: 'running',
      }));
    } catch (err) {
      console.error('Failed to approve:', err);
      setState(prev => ({ ...prev, error: 'Failed to approve' }));
    }
  }, [state.pendingApproval]);

  // Reject pending request
  const reject = useCallback(async () => {
    if (!state.pendingApproval) return;

    try {
      const response = await fetch(`/api/coordinator/reject/${state.pendingApproval.id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });

      if (!response.ok) {
        throw new Error(await response.text());
      }

      setState(prev => ({
        ...prev,
        pendingApproval: null,
        status: 'rejected',
      }));
    } catch (err) {
      console.error('Failed to reject:', err);
      setState(prev => ({ ...prev, error: 'Failed to reject' }));
    }
  }, [state.pendingApproval]);

  return {
    ...state,
    connect,
    disconnect,
    clearEvents,
    approve,
    reject,
  };
};
