/**
 * WebSocket singleton service
 *
 * This module manages a single WebSocket connection shared across all components.
 * Uses window-level singleton to survive HMR and module re-evaluation.
 */

import {
  WSEvent,
  SubscribeEvent,
  AckEvent,
  MessageEvent,
  BatchEvent,
  ErrorEvent,
} from '../types';

// Connection state type
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting';

// Extend window type for our singleton
declare global {
  interface Window {
    __AILANG_WS_SERVICE__?: WebSocketService;
  }
}

// Singleton instance - stored on window to survive HMR
class WebSocketService {
  private ws: WebSocket | null = null;
  private wsUrl: string | null = null;
  private isConnecting = false;
  private reconnectTimeout: NodeJS.Timeout | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 10;

  private connectionState: ConnectionState = 'disconnected';
  private stateListeners = new Set<(state: ConnectionState, attempts: number) => void>();
  private messageHandlers = new Set<(msg: MessageEvent) => void>();
  private batchHandlers = new Set<(batch: BatchEvent) => void>();
  private errorHandlers = new Set<(error: ErrorEvent) => void>();

  private subscriptions = new Map<string, number>(); // threadID -> from_seq
  private hookCount = 0;

  // Get current state
  getState(): { isConnected: boolean; connectionState: ConnectionState; reconnectAttempts: number } {
    return {
      isConnected: this.connectionState === 'connected',
      connectionState: this.connectionState,
      reconnectAttempts: this.reconnectAttempts,
    };
  }

  // Subscribe to state changes
  subscribeToState(listener: (state: ConnectionState, attempts: number) => void): () => void {
    this.stateListeners.add(listener);
    listener(this.connectionState, this.reconnectAttempts);
    return () => this.stateListeners.delete(listener);
  }

  private setConnectionState(state: ConnectionState) {
    this.connectionState = state;
    this.stateListeners.forEach(listener => listener(state, this.reconnectAttempts));
  }

  // Register a hook instance
  registerHook(
    onMessage?: (msg: MessageEvent) => void,
    onBatch?: (batch: BatchEvent) => void,
    onError?: (error: ErrorEvent) => void
  ): () => void {
    this.hookCount++;
    console.log(`[WebSocketService] Hook registered, count: ${this.hookCount}`);

    // Create wrapper handlers
    const msgHandler = onMessage ? (msg: MessageEvent) => onMessage(msg) : null;
    const batchHandler = onBatch ? (batch: BatchEvent) => onBatch(batch) : null;
    const errHandler = onError ? (error: ErrorEvent) => onError(error) : null;

    if (msgHandler) this.messageHandlers.add(msgHandler);
    if (batchHandler) this.batchHandlers.add(batchHandler);
    if (errHandler) this.errorHandlers.add(errHandler);

    // Return cleanup function
    return () => {
      this.hookCount--;
      console.log(`[WebSocketService] Hook unregistered, count: ${this.hookCount}`);

      if (msgHandler) this.messageHandlers.delete(msgHandler);
      if (batchHandler) this.batchHandlers.delete(batchHandler);
      if (errHandler) this.errorHandlers.delete(errHandler);

      // Close connection when all hooks unregister
      if (this.hookCount === 0) {
        console.log('[WebSocketService] All hooks unregistered, closing connection');
        this.disconnect();
      }
    };
  }

  // Connect to WebSocket server
  connect(url: string, instanceId: string, maxRetries = 10): void {
    this.maxReconnectAttempts = maxRetries;

    // Already connected to same URL
    if (this.ws && this.ws.readyState === WebSocket.OPEN && this.wsUrl === `${url}?instance_id=${instanceId}`) {
      console.log('[WebSocketService] Already connected');
      return;
    }

    // Already connecting
    if (this.isConnecting) {
      console.log('[WebSocketService] Already connecting');
      return;
    }

    const wsUrl = `${url}?instance_id=${instanceId}`;

    // If URL changed, close existing
    if (this.ws && this.wsUrl !== wsUrl) {
      console.log('[WebSocketService] URL changed, closing old connection');
      this.ws.close();
      this.ws = null;
    }

    this.isConnecting = true;
    this.wsUrl = wsUrl;

    console.log(`[WebSocketService] Connecting to ${wsUrl}`);
    this.setConnectionState(this.reconnectAttempts > 0 ? 'reconnecting' : 'connecting');

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('[WebSocketService] Connected');
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.setConnectionState('connected');

        // Re-subscribe to all threads
        this.subscriptions.forEach((fromSeq, threadId) => {
          this.subscribe(threadId, fromSeq);
        });
      };

      this.ws.onmessage = (event) => {
        try {
          const wsEvent: WSEvent = JSON.parse(event.data);
          this.handleEvent(wsEvent);
        } catch (err) {
          console.error('[WebSocketService] Failed to parse message:', err);
        }
      };

      this.ws.onerror = (event) => {
        console.error('[WebSocketService] Error:', event);
        this.isConnecting = false;
      };

      this.ws.onclose = () => {
        console.log('[WebSocketService] Disconnected');
        this.isConnecting = false;
        this.setConnectionState('disconnected');

        // Reconnect if hooks still registered
        if (this.hookCount > 0 && this.reconnectAttempts < this.maxReconnectAttempts) {
          const delay = this.getBackoffDelay(this.reconnectAttempts);
          console.log(`[WebSocketService] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts + 1}/${this.maxReconnectAttempts})`);

          this.reconnectTimeout = setTimeout(() => {
            this.reconnectAttempts++;
            this.connect(url, instanceId, maxRetries);
          }, delay);
        }
      };
    } catch (err) {
      console.error('[WebSocketService] Failed to connect:', err);
      this.isConnecting = false;
      this.setConnectionState('disconnected');
    }
  }

  // Disconnect
  disconnect(): void {
    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    this.wsUrl = null;
    this.reconnectAttempts = 0;
    this.subscriptions.clear();
    this.setConnectionState('disconnected');
  }

  // Send event
  private send(event: WSEvent): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(event));
    } else {
      console.warn('[WebSocketService] Not connected, cannot send');
    }
  }

  // Handle incoming event
  private handleEvent(event: WSEvent): void {
    switch (event.type) {
      case 'message':
        if (event.data) {
          this.messageHandlers.forEach(handler => handler(event.data as MessageEvent));
        }
        break;

      case 'batch':
        if (event.data) {
          const batch = event.data as BatchEvent;
          this.batchHandlers.forEach(handler => handler(batch));
          batch.messages.forEach(msg => {
            this.messageHandlers.forEach(handler => handler(msg));
          });
        }
        break;

      case 'error':
        if (event.data) {
          this.errorHandlers.forEach(handler => handler(event.data as ErrorEvent));
        }
        console.error('[WebSocketService] Error event:', event.data);
        break;

      case 'pong':
        // Heartbeat response
        break;

      default:
        console.log('[WebSocketService] Unknown event:', event.type);
    }
  }

  // Calculate backoff delay
  private getBackoffDelay(attempt: number, baseDelay = 1000, maxDelay = 30000): number {
    const delay = Math.min(baseDelay * Math.pow(2, attempt), maxDelay);
    const jitter = delay * Math.random() * 0.3;
    return Math.round(delay + jitter);
  }

  // Subscribe to thread
  subscribe(threadId: string, fromSeq = 0): void {
    this.subscriptions.set(threadId, fromSeq);

    this.send({
      type: 'subscribe',
      timestamp: Date.now(),
      data: {
        thread_id: threadId,
        from_seq: fromSeq,
      } as SubscribeEvent,
    });
  }

  // Unsubscribe from thread
  unsubscribe(threadId: string): void {
    this.subscriptions.delete(threadId);
  }

  // Acknowledge messages
  acknowledge(threadId: string, ackSeq: number): void {
    const currentSeq = this.subscriptions.get(threadId) || 0;
    if (ackSeq > currentSeq) {
      this.subscriptions.set(threadId, ackSeq);
    }

    this.send({
      type: 'ack',
      timestamp: Date.now(),
      data: {
        thread_id: threadId,
        ack_seq: ackSeq,
      } as AckEvent,
    });
  }

  // Send ping
  ping(): void {
    this.send({
      type: 'ping',
      timestamp: Date.now(),
    });
  }
}

// Get or create singleton instance on window
function getWebSocketService(): WebSocketService {
  if (typeof window !== 'undefined') {
    if (!window.__AILANG_WS_SERVICE__) {
      console.log('[WebSocketService] Creating new singleton instance');
      window.__AILANG_WS_SERVICE__ = new WebSocketService();
    } else {
      console.log('[WebSocketService] Reusing existing singleton instance');
    }
    return window.__AILANG_WS_SERVICE__;
  }
  // Fallback for SSR or non-browser environments
  return new WebSocketService();
}

// Export singleton instance
export const wsService = getWebSocketService();
