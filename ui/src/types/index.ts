// Type definitions for the UI Collaboration Hub

export interface Thread {
  id: string;
  title: string;
  created_at: number;
  created_by_type: string;
  created_by_id: string;
  status: 'active' | 'paused' | 'resolved' | 'archived';
  context_json?: string;
  last_seq: number;
  updated_at: number;
}

export interface Message {
  id: string;
  thread_id: string;
  message_seq: number;
  created_at: number;
  from_type: string;
  from_id: string;
  to_type: string;
  to_id: string;
  kind: 'directive' | 'question' | 'proposal' | 'status' | 'result' | 'approval_request';
  subject?: string;
  content: string;
  metadata_json?: string;
  delivery_state: 'pending' | 'visible' | 'acked';
  business_state: 'open' | 'resolved' | 'archived';
}

export interface Approval {
  id: string;
  thread_id: string;
  instance_id: string;
  created_at: number;
  effect_delta_json: string;
  proposal: string;
  impact: 'low' | 'medium' | 'high';
  estimated_cost: number;
  status: 'pending' | 'approved' | 'rejected' | 'modified';
  reviewed_by?: string;
  reviewed_at?: number;
  review_notes?: string;
  capability_token?: string;
  token_expires_at?: number;
}

export interface EffectDelta {
  cap_type: string;
  paths: string[];
  budget_delta: number;
}

// WebSocket Event Types
export type EventType =
  | 'subscribe'
  | 'ack'
  | 'message'
  | 'batch'
  | 'error'
  | 'ping'
  | 'pong'
  | 'thread_state';

export interface WSEvent {
  type: EventType;
  timestamp: number;
  data?: any;
}

export interface SubscribeEvent {
  thread_id: string;
  from_seq: number;
}

export interface AckEvent {
  thread_id: string;
  ack_seq: number;
}

export interface MessageEvent {
  id: string;
  thread_id: string;
  message_seq: number;
  created_at: number;
  from_type: string;
  from_id: string;
  to_type: string;
  to_id: string;
  kind: string;
  subject?: string;
  content: string;
  metadata_json?: string;
}

export interface BatchEvent {
  thread_id: string;
  messages: MessageEvent[];
  has_more: boolean;
}

export interface ErrorEvent {
  code: string;
  message: string;
}

export interface ThreadStateEvent {
  thread_id: string;
  status: string;
  last_seq: number;
  updated_at: number;
}
