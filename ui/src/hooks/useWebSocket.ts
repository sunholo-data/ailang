import { useEffect, useRef, useState, useCallback } from 'react';
import {
  WSEvent,
  SubscribeEvent,
  AckEvent,
  MessageEvent,
  BatchEvent,
  ErrorEvent,
} from '../types';

// Connection state type for global tracking
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting';

// Global connection state singleton
let globalConnectionState: ConnectionState = 'disconnected';
let globalReconnectAttempts = 0;
const globalListeners = new Set<(state: ConnectionState, attempts: number) => void>();

function setGlobalState(state: ConnectionState, attempts: number) {
  globalConnectionState = state;
  globalReconnectAttempts = attempts;
  globalListeners.forEach((listener) => listener(state, attempts));
}

export function subscribeToGlobalConnection(
  listener: (state: ConnectionState, attempts: number) => void
): () => void {
  globalListeners.add(listener);
  listener(globalConnectionState, globalReconnectAttempts);
  return () => globalListeners.delete(listener);
}

// Exponential backoff with jitter
function getBackoffDelay(attempt: number, baseDelay = 1000, maxDelay = 30000): number {
  const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
  const jitter = delay * Math.random() * 0.3;
  return Math.round(delay + jitter);
}

interface UseWebSocketOptions {
  url: string;
  instanceId: string;
  onMessage?: (message: MessageEvent) => void;
  onBatch?: (batch: BatchEvent) => void;
  onError?: (error: ErrorEvent) => void;
  maxReconnectAttempts?: number;
}

export const useWebSocket = ({
  url,
  instanceId,
  onMessage,
  onBatch,
  onError,
  maxReconnectAttempts = 10,
}: UseWebSocketOptions) => {
  const ws = useRef<WebSocket | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionError, setConnectionError] = useState<string | null>(null);
  const [reconnectAttempts, setReconnectAttempts] = useState(0);
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const subscriptions = useRef<Map<string, number>>(new Map()); // threadID -> from_seq

  // Connect to WebSocket
  const connect = useCallback(() => {
    try {
      const wsUrl = `${url}?instance_id=${instanceId}`;
      ws.current = new WebSocket(wsUrl);

      // Update global state
      setGlobalState(reconnectAttempts > 0 ? 'reconnecting' : 'connecting', reconnectAttempts);

      ws.current.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        setConnectionError(null);
        setReconnectAttempts(0);
        setGlobalState('connected', 0);

        // Re-subscribe to all threads after reconnection
        subscriptions.current.forEach((fromSeq, threadId) => {
          subscribe(threadId, fromSeq);
        });
      };

      ws.current.onmessage = (event) => {
        try {
          const wsEvent: WSEvent = JSON.parse(event.data);
          handleEvent(wsEvent);
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      ws.current.onerror = (event) => {
        console.error('WebSocket error:', event);
        setConnectionError('Connection error');
      };

      ws.current.onclose = () => {
        console.log('WebSocket disconnected');
        setIsConnected(false);
        setGlobalState('disconnected', reconnectAttempts);

        // Attempt to reconnect with exponential backoff
        if (reconnectAttempts < maxReconnectAttempts) {
          const delay = getBackoffDelay(reconnectAttempts);
          console.log(`WebSocket reconnecting in ${delay}ms (attempt ${reconnectAttempts + 1}/${maxReconnectAttempts})`);

          reconnectTimeout.current = setTimeout(() => {
            setReconnectAttempts((prev) => prev + 1);
            connect();
          }, delay);
        } else {
          console.error('Max reconnection attempts reached');
          setConnectionError('Connection lost. Please refresh the page.');
        }
      };
    } catch (err) {
      console.error('Failed to connect to WebSocket:', err);
      setConnectionError('Failed to connect');
      setGlobalState('disconnected', reconnectAttempts);
    }
  }, [url, instanceId, reconnectAttempts, maxReconnectAttempts]);

  // Handle incoming WebSocket events
  const handleEvent = useCallback(
    (event: WSEvent) => {
      switch (event.type) {
        case 'message':
          if (onMessage && event.data) {
            onMessage(event.data as MessageEvent);
          }
          break;

        case 'batch':
          if (onBatch && event.data) {
            const batch = event.data as BatchEvent;
            onBatch(batch);
            // Also call onMessage for each message in the batch
            if (onMessage) {
              batch.messages.forEach((msg) => onMessage(msg));
            }
          }
          break;

        case 'error':
          if (onError && event.data) {
            onError(event.data as ErrorEvent);
          }
          console.error('WebSocket error event:', event.data);
          break;

        case 'pong':
          // Pong received, connection is healthy
          break;

        default:
          console.log('Unknown event type:', event.type);
      }
    },
    [onMessage, onBatch, onError]
  );

  // Send event to server
  const sendEvent = useCallback((event: WSEvent) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify(event));
    } else {
      console.warn('WebSocket not connected, cannot send event');
    }
  }, []);

  // Subscribe to a thread
  const subscribe = useCallback(
    (threadId: string, fromSeq: number = 0) => {
      subscriptions.current.set(threadId, fromSeq);

      const subscribeEvent: WSEvent = {
        type: 'subscribe',
        timestamp: Date.now(),
        data: {
          thread_id: threadId,
          from_seq: fromSeq,
        } as SubscribeEvent,
      };

      sendEvent(subscribeEvent);
    },
    [sendEvent]
  );

  // Acknowledge messages up to a sequence number
  const acknowledge = useCallback(
    (threadId: string, ackSeq: number) => {
      // Update local tracking
      const currentFromSeq = subscriptions.current.get(threadId) || 0;
      if (ackSeq > currentFromSeq) {
        subscriptions.current.set(threadId, ackSeq);
      }

      const ackEvent: WSEvent = {
        type: 'ack',
        timestamp: Date.now(),
        data: {
          thread_id: threadId,
          ack_seq: ackSeq,
        } as AckEvent,
      };

      sendEvent(ackEvent);
    },
    [sendEvent]
  );

  // Send ping
  const ping = useCallback(() => {
    const pingEvent: WSEvent = {
      type: 'ping',
      timestamp: Date.now(),
    };
    sendEvent(pingEvent);
  }, [sendEvent]);

  // Unsubscribe from a thread
  const unsubscribe = useCallback((threadId: string) => {
    subscriptions.current.delete(threadId);
  }, []);

  // Connect on mount
  useEffect(() => {
    connect();

    // Cleanup on unmount
    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [connect]);

  // Heartbeat ping every 30 seconds
  useEffect(() => {
    if (!isConnected) return;

    const interval = setInterval(() => {
      ping();
    }, 30000);

    return () => clearInterval(interval);
  }, [isConnected, ping]);

  return {
    isConnected,
    connectionError,
    subscribe,
    unsubscribe,
    acknowledge,
    ping,
  };
};
