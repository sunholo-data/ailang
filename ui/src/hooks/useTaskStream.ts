import { useEffect, useState, useCallback, useRef } from 'react';
import { TaskStreamEvent, TaskResourceMetrics, PendingApprovalRequest } from '../types';
import { wsService } from '../services/websocket';

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

  const onEventRef = useRef(onEvent);
  const taskIdRef = useRef(taskId);

  // Update refs when props change
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    taskIdRef.current = taskId;
    // When taskId changes, fetch historical events from database
    if (taskId) {
      // Clear current events and fetch historical ones
      setState(prev => ({ ...prev, events: [] }));

      // Fetch historical events from API
      fetch(`/api/coordinator/tasks/${taskId}/events`)
        .then(res => res.json())
        .then((historicalEvents: TaskStreamEvent[]) => {
          console.log('[useTaskStream] Loaded historical events:', historicalEvents?.length || 0);
          if (historicalEvents && historicalEvents.length > 0) {
            setState(prev => ({
              ...prev,
              events: historicalEvents.map((e: TaskStreamEvent) => ({ ...e, type: 'task_stream' })),
            }));
          }
        })
        .catch(err => {
          console.error('[useTaskStream] Failed to fetch historical events:', err);
        });
    }
  }, [taskId]);

  // Subscribe to task stream events via the shared WebSocket service
  useEffect(() => {
    // Track connection state from service
    const unsubscribeState = wsService.subscribeToState((connectionState) => {
      setState(prev => ({
        ...prev,
        isConnected: connectionState === 'connected',
      }));
    });

    // Subscribe to task stream events
    const unsubscribeEvents = wsService.subscribeToTaskStream((streamEvent) => {
      // Filter by taskId - only process events for our task
      // If taskId is empty, accept ALL events (useful for "running tasks" view)
      const currentTaskId = taskIdRef.current;
      if (currentTaskId && streamEvent.task_id !== currentTaskId) {
        return;
      }

      console.log('[useTaskStream] Received:', streamEvent.stream_type, 'for task', streamEvent.task_id);

      // Add type for consistency
      const eventWithType = { ...streamEvent, type: 'task_stream' as const };

      // Update events list
      setState(prev => ({
        ...prev,
        events: [...prev.events, eventWithType].slice(-500), // Keep last 500 events
      }));

      // Update metrics from status events
      if (streamEvent.stream_type === 'status') {
        setState(prev => ({
          ...prev,
          metrics: {
            task_id: streamEvent.task_id,
            cpu_percent: 0,
            memory_mb: 0,
            tokens_in: streamEvent.tokens_in || 0,
            tokens_out: streamEvent.tokens_out || 0,
            cost: streamEvent.cost || 0,
            peak_cpu: 0,
            peak_memory: 0,
            updated_at: streamEvent.timestamp || Date.now(),
          },
        }));
        if (streamEvent.status) {
          setState(prev => ({ ...prev, status: streamEvent.status! }));
        }
      }

      // Update status to 'running' on turn_start
      if (streamEvent.stream_type === 'turn_start') {
        setState(prev => ({ ...prev, status: 'running' }));
      }

      // Call external handler
      if (onEventRef.current) {
        onEventRef.current(streamEvent);
      }
    });

    return () => {
      unsubscribeState();
      unsubscribeEvents();
    };
  }, []); // Empty deps - subscribe once on mount

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
    clearEvents,
    approve,
    reject,
  };
};
