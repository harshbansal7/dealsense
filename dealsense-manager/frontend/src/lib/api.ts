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

