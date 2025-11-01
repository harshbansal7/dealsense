/**
 * API client for communicating with the DealSense backend.
 */

import axios from 'axios';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8001';

export const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
});

// Separate API client for long-running operations like chat with LLM
export const apiLongTimeout = axios.create({
  baseURL: API_BASE_URL,
  timeout: 60000, // 60 seconds for LLM processing
});

// Request interceptor for auth if needed
const requestInterceptor = (config: import('axios').InternalAxiosRequestConfig) => {
  // Add auth headers here if needed
  return config;
};

const requestErrorInterceptor = (error: import('axios').AxiosError) => {
  return Promise.reject(error);
};

// Response interceptor for error handling
const responseInterceptor = (response: import('axios').AxiosResponse) => {
  return response;
};

const responseErrorInterceptor = (error: import('axios').AxiosError) => {
  if (error.response?.status === 401) {
    // Handle unauthorized
    console.error('Unauthorized access');
  }
  return Promise.reject(error);
};

api.interceptors.request.use(requestInterceptor, requestErrorInterceptor);
api.interceptors.response.use(responseInterceptor, responseErrorInterceptor);

// Apply same interceptors to long timeout client
apiLongTimeout.interceptors.request.use(requestInterceptor, requestErrorInterceptor);
apiLongTimeout.interceptors.response.use(responseInterceptor, responseErrorInterceptor);

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
  tts_provider?: TTSProvider; // Optional for analyst mode
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
  tts_provider?: TTSProvider; // Optional for analyst mode
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

// Document types
export interface Document {
  id: string;
  agent_id: string;
  meeting_id?: string;
  name: string;
  original_name: string;
  file_type: string;
  file_size: number;
  storage_path: string;
  gcs_bucket: string;
  processed_at?: string;
  status: 'uploaded' | 'processing' | 'processed' | 'failed';
  error_message?: string;
  extracted_text: string;
  page_count: number;
  metadata: string;
  created_at: string;
  updated_at: string;
}

export interface DocumentSearchResult {
  chunk_text: string;
  chunk_index: number;
  page_number: number;
  similarity: number;
  metadata?: Record<string, unknown>;
}

export interface ChatMessage {
  id: string;
  agent_id: string;
  document_id?: string;
  session_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  context_chunks?: string;
  token_count: number;
  created_at: string;
}

export interface ChatRequest {
  session_id?: string;
  query: string;
  document_id?: string;
  top_k?: number;
}

export interface ContextChunk {
  text: string;
  source: 'document' | 'meeting';
  page_number: number;
  similarity: number;
  display_source: string;
}

export interface Source {
  type: 'document' | 'meeting';
  id: string;
  name: string;
}

export interface ChatResponse {
  session_id: string;
  query: string;
  response: string;
  context_chunks: ContextChunk[];
  sources: Source[];
  token_count: number;
  response_time_ms: number;
}

export interface AnalysisSection {
  type: string;
  score: number;
  summary: string;
  key_findings: string[];
  red_flags: string[];
  opportunities: string[];
  recommendations: string[];
}

export interface StartupAnalysisResult {
  agent_id: string;
  overall_score: number;
  analysis_sections: Record<string, AnalysisSection>;
  summary: string;
  generated_at: string;
}

export interface AnalyzeStartupRequest {
  document_ids: string[];
  meeting_id?: string;
  analysis_types?: string[];
}

// Document API
export const documentsApi = {
  upload: (agentId: string, file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    return api.post<Document>(`/agents/${agentId}/documents`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 60000, // 60 second timeout for large files
    });
  },
  list: (agentId: string) => api.get<Document[]>(`/agents/${agentId}/documents`),
  get: (documentId: string) => api.get<Document>(`/documents/${documentId}`),
  delete: (documentId: string) => api.delete(`/documents/${documentId}`),
  getDownloadUrl: (documentId: string) => api.get<{ download_url: string }>(`/documents/${documentId}/download`),
  search: (agentId: string, query: string, topK: number = 5) => 
    api.post<{ results: DocumentSearchResult[] }>(`/agents/${agentId}/documents/search`, {
      query,
      top_k: topK,
    }),
};

// Chatbot API
export const chatbotApi = {
  query: (agentId: string, request: ChatRequest) =>
    apiLongTimeout.post<ChatResponse>(`/agents/${agentId}/chat`, request),
  getHistory: (agentId: string, sessionId: string) =>
    api.get<{ messages: ChatMessage[] }>(`/agents/${agentId}/chat/${sessionId}`),
};

// Startup Analysis API
export const analysisApi = {
  analyze: (agentId: string, request: AnalyzeStartupRequest) =>
    apiLongTimeout.post<StartupAnalysisResult>(`/agents/${agentId}/analyze`, request),
  getLatest: (agentId: string) =>
    api.get<StartupAnalysisResult>(`/agents/${agentId}/analysis/startup`),
};

