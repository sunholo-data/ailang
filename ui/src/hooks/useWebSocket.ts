import { useEffect, useState, useCallback, useRef } from 'react';
import { wsService, ConnectionState } from '../services/websocket';
import { MessageEvent, BatchEvent, ErrorEvent } from '../types';

// Re-export types for backwards compatibility
export type { ConnectionState };

// Subscribe to global connection state changes
export function subscribeToGlobalConnection(
  listener: (state: ConnectionState, attempts: number) => void
): () => void {
  return wsService.subscribeToState(listener);
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
  const [isConnected, setIsConnected] = useState(wsService.getState().isConnected);
  const [connectionError, setConnectionError] = useState<string | null>(null);

  // Store callback refs to avoid stale closures
  const onMessageRef = useRef(onMessage);
  const onBatchRef = useRef(onBatch);
  const onErrorRef = useRef(onError);

  // Update refs when callbacks change
  useEffect(() => {
    onMessageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    onBatchRef.current = onBatch;
  }, [onBatch]);

  useEffect(() => {
    onErrorRef.current = onError;
  }, [onError]);

  // Register with service and connect
  useEffect(() => {
    // Create handlers that use refs
    const messageHandler = (msg: MessageEvent) => {
      if (onMessageRef.current) onMessageRef.current(msg);
    };
    const batchHandler = (batch: BatchEvent) => {
      if (onBatchRef.current) onBatchRef.current(batch);
    };
    const errorHandler = (error: ErrorEvent) => {
      if (onErrorRef.current) onErrorRef.current(error);
    };

    // Register handlers
    const unregister = wsService.registerHook(messageHandler, batchHandler, errorHandler);

    // Connect
    wsService.connect(url, instanceId, maxReconnectAttempts);

    // Cleanup
    return unregister;
  }, [url, instanceId, maxReconnectAttempts]);

  // Subscribe to state changes
  useEffect(() => {
    const unsubscribe = wsService.subscribeToState((state, attempts) => {
      setIsConnected(state === 'connected');
      if (attempts >= maxReconnectAttempts) {
        setConnectionError('Connection lost. Please refresh the page.');
      } else {
        setConnectionError(null);
      }
    });
    return unsubscribe;
  }, [maxReconnectAttempts]);

  // Heartbeat - only one instance should ping
  useEffect(() => {
    if (!isConnected) return;

    const interval = setInterval(() => {
      wsService.ping();
    }, 30000);

    return () => clearInterval(interval);
  }, [isConnected]);

  // API methods
  const subscribe = useCallback((threadId: string, fromSeq = 0) => {
    wsService.subscribe(threadId, fromSeq);
  }, []);

  const unsubscribe = useCallback((threadId: string) => {
    wsService.unsubscribe(threadId);
  }, []);

  const acknowledge = useCallback((threadId: string, ackSeq: number) => {
    wsService.acknowledge(threadId, ackSeq);
  }, []);

  const ping = useCallback(() => {
    wsService.ping();
  }, []);

  return {
    isConnected,
    connectionError,
    subscribe,
    unsubscribe,
    acknowledge,
    ping,
  };
};
