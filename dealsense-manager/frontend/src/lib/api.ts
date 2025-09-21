/**
 * API client for communicating with the DealSense backend.
 */

import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

// Request interceptor for auth if needed
api.interceptors.request.use(
  (config) => {
    // Add auth headers here if needed
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    return response;
  },
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized
      console.error('Unauthorized access');
    }
    return Promise.reject(error);
  }
);

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

export interface Agent {
  id: string;
  config: AgentConfig;
  status: AgentStatus;
  created_at: string;
  started_at?: string;
  stopped_at?: string;
  error_message?: string;
  process_id?: number;
  logs: LogEntry[];
}

export interface AgentConfig {
  name: string;
  meeting_url: string;
  conversation_mode?: ConversationMode;
  llm_provider: LLMProvider;
  llm_model: string;
  tts_provider: TTSProvider;
  stt_provider: STTProvider;
  language: string;
  prompt_style: string;
  custom_prompt?: string;
  stt_custom_prompt?: string;

  // Analyst-specific custom prompts
  summary_prompt?: string;
  key_points_prompt?: string;
  action_items_prompt?: string;
  topics_prompt?: string;
  sentiment_prompt?: string;

  name_trigger: boolean;
  auto_join: boolean;
  enable_conversation_context: boolean;
  env_vars: Record<string, string>;
}

export type AgentStatus = 'created' | 'starting' | 'running' | 'stopping' | 'stopped' | 'error';
export type ConversationMode = 'conversational' | 'analyst';
export type LLMProvider = 'openai' | 'anthropic' | 'google' | 'ollama';
export type TTSProvider = 'kokoro' | 'elevenlabs' | 'deepgram';
export type STTProvider = 'whisper' | 'deepgram' | 'gemini' | 'gemini_direct';

export interface MeetingInfo {
  url: string;
  agent_count: number;
  agent_ids: string[];
  created_at: string;
}

export interface CreateAgentRequest {
  name: string;
  meeting_url: string;
  conversation_mode?: ConversationMode;
  llm_provider?: LLMProvider;
  llm_model?: string;
  tts_provider?: TTSProvider;
  stt_provider?: STTProvider;
  language?: string;
  prompt_style?: string;
  custom_prompt?: string;
  stt_custom_prompt?: string;

  name_trigger?: boolean;
  auto_join?: boolean;
  enable_conversation_context?: boolean;
  env_vars?: Record<string, string>;
}

export interface GroundingChunk {
  web: {
    uri: string;
    title: string;
  };
}

export interface GroundingSupport {
  segment: {
    start_index: number;
    end_index: number;
    text: string;
  };
  grounding_chunk_indices: number[];
}

export interface GroundingMetadata {
  web_search_queries: string[];
  grounding_chunks: GroundingChunk[];
  grounding_supports: GroundingSupport[];
  search_entry_point?: unknown;
}

export interface GroundedContent {
  text: string;
  text_with_citations: string;
  grounding_metadata?: GroundingMetadata;
}

export interface AnalysisData {
  meeting_id: string;
  meeting_url: string;
  start_time: string;
  last_updated: string;
  transcript: TranscriptEntry[];
  summary: string;
  grounded_summary?: GroundedContent;
  key_points: string[];
  grounded_key_points?: GroundedContent;
  action_items: ActionItem[];
  topics: TopicDiscussion[];
  participants: string[];
  duration_minutes: number;
  word_count: number;
  sentiment: string;
  keywords: string[];
}

export interface TranscriptEntry {
  timestamp: string;
  speaker: string;
  text: string;
  is_agent: boolean;
}

export interface ActionItem {
  id: string;
  description: string;
  assignee?: string;
  priority: 'high' | 'medium' | 'low';
  type?: 'task' | 'research' | 'investigation' | 'follow-up' | 'decision';
  status: 'pending' | 'in_progress' | 'completed';
  created_at: string;
  due_date?: string;
}

export interface TopicDiscussion {
  topic: string;
  start_time: string;
  end_time: string;
  duration: number;
  summary: string;
  participants: string[];
}

export interface WebSocketMessage {
  type: string;
  agent_id: string;
  data: Record<string, unknown>;
  timestamp: string;
}

// API functions
export const agentsApi = {
  list: () => api.get<Agent[]>('/agents'),
  create: (config: CreateAgentRequest) => api.post<Agent>('/agents', config),
  get: (id: string) => api.get<Agent>(`/agents/${id}`),
  delete: (id: string) => api.delete(`/agents/${id}`),
  start: (id: string) => api.post(`/agents/${id}/start`),
  stop: (id: string) => api.post(`/agents/${id}/stop`),
  getLogs: (id: string, lines?: number) => api.get(`/agents/${id}/logs`, { params: { lines } }),
  getAnalysis: (id: string) => api.get<AnalysisData>(`/agents/${id}/analysis`),
  getFormattedAnalysis: (id: string) => api.get<string>(`/agents/${id}/analysis/formatted`),
};

export const meetingsApi = {
  list: () => api.get<MeetingInfo[]>('/meetings'),
};

// WebSocket connection for single session
export class SessionWebSocketManager {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 1; // Reduced to prevent spam
  private reconnectDelay = 10000; // Increased to 10 seconds
  private isIntentionallyDisconnected = false;
  private isConnecting = false;

  constructor(
    private onMessage: (message: WebSocketMessage) => void,
    private onError?: (error: Event) => void,
    private onClose?: () => void
  ) {}

  connect() {
    if (this.isIntentionallyDisconnected) {
      console.log('WebSocket was intentionally disconnected, not reconnecting');
      return;
    }

    if (this.isConnecting) {
      console.log('WebSocket connection already in progress');
      return;
    }

    if (this.ws?.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected');
      return;
    }

    if (this.ws?.readyState === WebSocket.CONNECTING) {
      console.log('WebSocket already connecting for session');
      return;
    }

    this.isConnecting = true;
    const wsUrl = 'ws://localhost:8001/ws/session';
    console.log(`🔌 Connecting to WebSocket (attempt ${this.reconnectAttempts + 1})`);
    
    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('✅ WebSocket connected successfully');
        this.reconnectAttempts = 0;
        this.isConnecting = false;
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          this.onMessage(message);
        } catch (error) {
          console.error('❌ Failed to parse WebSocket message:', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('❌ WebSocket error:', error);
        this.isConnecting = false;
        this.onError?.(error);
      };

      this.ws.onclose = (event) => {
        console.log(`🔌 WebSocket closed (code: ${event.code})`);
        this.isConnecting = false;
        this.onClose?.();
        
        // Only attempt reconnect if it wasn't intentional and within limits
        if (!this.isIntentionallyDisconnected && event.code !== 1000 && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.attemptReconnect();
        } else if (this.reconnectAttempts >= this.maxReconnectAttempts) {
          console.warn('🚫 Max reconnection attempts reached');
        }
      };
    } catch (error) {
      console.error('❌ Failed to create WebSocket connection:', error);
      this.isConnecting = false;
      this.onError?.(error as Event);
    }
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      console.log(`Attempting to reconnect session WebSocket (attempt ${this.reconnectAttempts})`);
      this.connect();
    }, delay);
  }
}

// Legacy WebSocket connection for individual agents (kept for backward compatibility)
export class WebSocketManager {
  private ws: WebSocket | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor(
    private agentId: string,
    private onMessage: (message: WebSocketMessage) => void,
    private onError?: (error: Event) => void,
    private onClose?: () => void
  ) {}

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    if (this.ws?.readyState === WebSocket.CONNECTING) {
      console.log(`WebSocket already connecting for agent ${this.agentId}`);
      return;
    }

    const wsUrl = `ws://localhost:8001/ws/agents/${this.agentId}`;
    console.log(`Connecting to WebSocket: ${wsUrl}`);
    this.ws = new WebSocket(wsUrl);

    this.ws.onopen = () => {
      console.log(`✅ WebSocket connected for agent ${this.agentId}`);
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        console.log(`📨 WebSocket message received for ${this.agentId}:`, message);
        this.onMessage(message);
      } catch (error) {
        console.error('❌ Failed to parse WebSocket message:', error, event.data);
      }
    };

    this.ws.onerror = (error) => {
      console.error(`❌ WebSocket error for agent ${this.agentId}:`, error);
      this.onError?.(error);
    };

    this.ws.onclose = (event) => {
      console.log(`🔌 WebSocket closed for agent ${this.agentId} (code: ${event.code}, reason: ${event.reason})`);
      this.onClose?.();
      // Only attempt reconnect if it wasn't a clean close
      if (event.code !== 1000) {
        this.attemptReconnect();
      }
    };
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    setTimeout(() => {
      console.log(`Attempting to reconnect WebSocket (attempt ${this.reconnectAttempts})`);
      this.connect();
    }, delay);
  }
}
