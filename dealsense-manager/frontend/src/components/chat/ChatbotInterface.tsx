/**
 * Professional Chatbot interface for querying meeting and document data
 * Features: Markdown support, animated thinking states, modern UI
 */

'use client';

import { useState, useRef, useEffect } from 'react';
import { Send, Bot, User, MessageSquare, FileText, RotateCcw, Sparkles } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { chatbotApi, ChatMessage as ApiChatMessage } from '@/lib/api';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

interface ChatbotInterfaceProps {
  agentId: string;
  sessionId?: string;
}

interface ChatMessage extends ApiChatMessage {
  isLoading?: boolean;
}

interface ChatSession {
  sessionId: string;
  messages: ChatMessage[];
  createdAt: string;
  lastActivity: string;
}

// Engaging startup-related thinking messages
const THINKING_MESSAGES = [
  "🚀 Analyzing startup metrics and growth potential...",
  "💡 Synthesizing insights from pitch materials...",
  "📊 Evaluating market opportunity and competition...",
  "🎯 Identifying key value propositions...",
  "💰 Assessing financial viability and projections...",
  "🌟 Examining team composition and expertise...",
  "📈 Reviewing traction and growth indicators...",
  "🔍 Searching for relevant market intelligence...",
  "💼 Analyzing business model and revenue streams...",
  "⚡ Processing meeting insights and key points...",
  "🎨 Evaluating product-market fit signals...",
  "🌐 Researching industry trends and dynamics...",
  "🏆 Identifying competitive advantages...",
  "📝 Extracting actionable insights...",
  "🤝 Analyzing stakeholder feedback patterns...",
];

const generateSessionId = () => {
  return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

const getStorageKey = (agentId: string) => `dealsense_chat_${agentId}`;

const saveChatSession = (agentId: string, session: ChatSession) => {
  try {
    const sessions = loadChatSessions(agentId);
    const updatedSessions = sessions.filter(s => s.sessionId !== session.sessionId);
    updatedSessions.unshift(session);
    const trimmedSessions = updatedSessions.slice(0, 5);
    localStorage.setItem(getStorageKey(agentId), JSON.stringify(trimmedSessions));
  } catch (error) {
    console.warn('Failed to save chat session:', error);
  }
};

const loadChatSessions = (agentId: string): ChatSession[] => {
  try {
    const stored = localStorage.getItem(getStorageKey(agentId));
    if (!stored) return [];
    const sessions: ChatSession[] = JSON.parse(stored);
    return sessions.filter(session =>
      session.sessionId &&
      Array.isArray(session.messages) &&
      session.messages.length > 0
    );
  } catch (error) {
    console.warn('Failed to load chat sessions:', error);
    return [];
  }
};

// Animated thinking indicator component
const ThinkingIndicator = ({ message }: { message: string }) => {
  const [dots, setDots] = useState('');
  
  useEffect(() => {
    const interval = setInterval(() => {
      setDots(prev => prev.length >= 3 ? '' : prev + '.');
    }, 500);
    return () => clearInterval(interval);
  }, []);

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-3">
        <Sparkles className="h-4 w-4 text-purple-600 dark:text-purple-400 animate-pulse" />
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {message}
          <span className="inline-block w-8 text-left">{dots}</span>
        </span>
      </div>
      
      {/* Animated progress bar */}
      <div className="relative h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-r from-blue-500 via-purple-500 to-pink-500 animate-shimmer"></div>
      </div>
    </div>
  );
};

export function ChatbotInterface({ agentId, sessionId: initialSessionId }: ChatbotInterfaceProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState(initialSessionId || '');
  const [availableSessions, setAvailableSessions] = useState<ChatSession[]>([]);
  const [currentSession, setCurrentSession] = useState<ChatSession | null>(null);
  const [currentThinkingMessage, setCurrentThinkingMessage] = useState('');
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Load existing chat sessions on mount
  useEffect(() => {
    const sessions = loadChatSessions(agentId);
    setAvailableSessions(sessions);

    if (initialSessionId) {
      const targetSession = sessions.find(s => s.sessionId === initialSessionId);
      if (targetSession) {
        setCurrentSession(targetSession);
        setMessages(targetSession.messages);
        setSessionId(targetSession.sessionId);
      } else {
        setSessionId(initialSessionId);
        setCurrentSession({
          sessionId: initialSessionId,
          messages: [],
          createdAt: new Date().toISOString(),
          lastActivity: new Date().toISOString(),
        });
      }
    } else if (sessions.length > 0) {
      const mostRecent = sessions[0];
      setCurrentSession(mostRecent);
      setMessages(mostRecent.messages);
      setSessionId(mostRecent.sessionId);
    } else {
      startNewChat();
    }
  }, [agentId]); // eslint-disable-line react-hooks/exhaustive-deps

  // Save current session whenever messages change
  useEffect(() => {
    if (sessionId && messages.length > 0 && !messages.some(m => m.isLoading)) {
      const createdAt = currentSession?.createdAt || new Date().toISOString();
      const session: ChatSession = {
        sessionId,
        messages,
        createdAt,
        lastActivity: new Date().toISOString(),
      };
      saveChatSession(agentId, session);
      setCurrentSession(session);
      const sessions = loadChatSessions(agentId);
      setAvailableSessions(sessions);
    }
  }, [messages, sessionId, agentId, currentSession?.createdAt]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Rotate thinking messages
  useEffect(() => {
    if (isLoading) {
      const getRandomMessage = () => THINKING_MESSAGES[Math.floor(Math.random() * THINKING_MESSAGES.length)];
      setCurrentThinkingMessage(getRandomMessage());
      
      const interval = setInterval(() => {
        setCurrentThinkingMessage(getRandomMessage());
      }, 3000);
      
      return () => clearInterval(interval);
    }
  }, [isLoading]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  const startNewChat = () => {
    const newSessionId = generateSessionId();
    setSessionId(newSessionId);
    setMessages([]);
    setCurrentSession({
      sessionId: newSessionId,
      messages: [],
      createdAt: new Date().toISOString(),
      lastActivity: new Date().toISOString(),
    });
  };

  const loadChatSession = (session: ChatSession) => {
    setCurrentSession(session);
    setMessages(session.messages);
    setSessionId(session.sessionId);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!input.trim() || isLoading) return;

    const trimmedInput = input.trim();
    const userMessage: ChatMessage = {
      id: `temp_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'user',
      content: trimmedInput,
      token_count: 0,
      created_at: new Date().toISOString(),
    };

    // Add loading message
    const loadingMessage: ChatMessage = {
      id: `loading_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'assistant',
      content: '',
      token_count: 0,
      created_at: new Date().toISOString(),
      isLoading: true,
    };

    // Update state in a single batch: add user message + loading message, clear input, set loading
    setMessages(prev => [...prev, userMessage, loadingMessage]);
    setInput('');
    setIsLoading(true);

    try {
      const response = await chatbotApi.query(agentId, {
        query: userMessage.content,
        session_id: sessionId,
        top_k: 5,
      });

      const chatResponse = response.data;

      // Create assistant message
      const assistantMessage: ChatMessage = {
        id: `assistant_${Date.now()}`,
        agent_id: agentId,
        session_id: chatResponse.session_id,
        role: 'assistant',
        content: chatResponse.response,
        token_count: chatResponse.token_count,
        created_at: new Date().toISOString(),
        context_chunks: JSON.stringify(chatResponse.context_chunks),
      };

      // Replace loading message with actual response instantly
      setMessages(prev => prev.map(m => m.id === loadingMessage.id ? assistantMessage : m));
      setSessionId(chatResponse.session_id);
    } catch (error) {
      console.error('Chat query failed:', error);
      
      // Create error message
      const errorMessage: ChatMessage = {
        id: `error_${Date.now()}`,
        agent_id: agentId,
        session_id: sessionId,
        role: 'assistant',
        content: '⚠️ Sorry, I encountered an error processing your request. Please try again.',
        token_count: 0,
        created_at: new Date().toISOString(),
      };

      // Replace loading message with error message instantly
      setMessages(prev => prev.map(m => m.id === loadingMessage.id ? errorMessage : m));
    } finally {
      setIsLoading(false);
    }
  };

  const getSuggestionQuestions = () => {
    return [
      "What are the key highlights from the pitch?",
      "Analyze the market opportunity and competition",
      "What are the main risks and red flags?",
      "Summarize the team's background and expertise",
      "What's the revenue model and financial projection?",
    ];
  };

  const handleSuggestionClick = (question: string) => {
    setInput(question);
  };

  return (
    <Card className="h-full flex flex-col shadow-lg border-gray-200 dark:border-gray-700">
      <CardHeader className="border-b bg-gradient-to-r from-blue-50 to-purple-50 dark:from-gray-800 dark:to-gray-850">
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2 text-xl">
              <div className="p-2 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg">
                <MessageSquare className="h-5 w-5 text-white" />
              </div>
              <span className="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent font-bold">
              AI Assistant
              </span>
            </CardTitle>
            <CardDescription className="mt-1">
              Powered by Google Gemini with real-time search capabilities
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {availableSessions.length > 0 && (
              <>
                <select
                  value={sessionId}
                  onChange={(e) => {
                    const selectedSession = availableSessions.find(s => s.sessionId === e.target.value);
                    if (selectedSession) {
                      loadChatSession(selectedSession);
                    }
                  }}
                  className="text-xs px-3 py-1.5 border rounded-lg bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-750 transition-colors"
                >
                  {availableSessions.map((session) => (
                    <option key={session.sessionId} value={session.sessionId}>
                      Session {session.sessionId.slice(8, 16)}... ({session.messages.length})
                    </option>
                  ))}
                </select>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={startNewChat}
                  className="text-xs hover:bg-blue-50 dark:hover:bg-blue-900/20"
                >
                  <RotateCcw className="h-3 w-3 mr-1" />
                  New Chat
                </Button>
              </>
            )}
          </div>
        </div>
      </CardHeader>
      
      <CardContent className="flex-1 flex flex-col min-h-0 pb-4 pt-4">
        {/* Messages Area */}
        <ScrollArea className="flex-1 pr-4 mb-4" ref={scrollAreaRef}>
          <div className="space-y-6">
            {messages.length === 0 && (
              <div className="text-center py-12">
                <div className="relative inline-block mb-6">
                  <div className="absolute inset-0 bg-gradient-to-r from-blue-400 to-purple-600 rounded-full blur-xl opacity-30 animate-pulse"></div>
                  <div className="relative bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-900/40 dark:to-purple-900/40 p-6 rounded-full">
                    <Bot className="h-12 w-12 text-blue-600 dark:text-blue-400" />
                  </div>
                </div>
                <h3 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
                  {currentSession ? 'Welcome back!' : 'Ready to analyze?'}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-6 max-w-md mx-auto">
                  I can help you analyze meetings, pitch documents, market trends, and more using advanced AI and real-time data.
                </p>

                {/* Suggested Questions */}
                <div className="text-left max-w-2xl mx-auto">
                  <p className="text-xs font-semibold text-gray-500 dark:text-gray-400 mb-3 uppercase tracking-wide">
                    Try asking:
                  </p>
                  <div className="grid grid-cols-1 gap-2">
                    {getSuggestionQuestions().map((question, index) => (
                      <Button
                        key={index}
                        variant="outline"
                        size="sm"
                        onClick={() => handleSuggestionClick(question)}
                        className="text-xs text-left justify-start h-auto py-3 px-4 hover:bg-blue-50 dark:hover:bg-blue-900/20 hover:border-blue-300 dark:hover:border-blue-700 transition-all"
                      >
                        <span className="mr-2 text-blue-600 dark:text-blue-400">→</span>
                        {question}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            )}
            
            {messages.map((message) => (
              <div
                key={message.id}
                className={`flex gap-4 ${
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                } animate-fadeIn`}
              >
                {message.role === 'assistant' && (
                  <div className="flex-shrink-0">
                    <div className="h-10 w-10 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-md">
                      <Bot className="h-6 w-6 text-white" />
                    </div>
                  </div>
                )}
                
                <div
                  className={`max-w-[80%] rounded-2xl px-5 py-3 shadow-sm ${
                    message.role === 'user'
                      ? 'bg-gradient-to-br from-blue-600 to-blue-700 text-white'
                      : 'bg-white dark:bg-gray-800 text-gray-900 dark:text-white border border-gray-200 dark:border-gray-700'
                  }`}
                >
                  {message.isLoading ? (
                    <ThinkingIndicator message={currentThinkingMessage} />
                  ) : (
                    <>
                      {message.role === 'user' ? (
                        <p className="text-sm whitespace-pre-wrap leading-relaxed">{message.content}</p>
                      ) : (
                        <div className="prose prose-sm dark:prose-invert max-w-none prose-headings:font-bold prose-headings:text-gray-900 dark:prose-headings:text-white prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-strong:text-gray-900 dark:prose-strong:text-white prose-ul:text-gray-700 dark:prose-ul:text-gray-300 prose-code:text-purple-600 dark:prose-code:text-purple-400 prose-code:bg-gray-100 dark:prose-code:bg-gray-900 prose-code:px-1 prose-code:py-0.5 prose-code:rounded">
                          <ReactMarkdown remarkPlugins={[remarkGfm]}>
                            {message.content}
                          </ReactMarkdown>
                        </div>
                      )}
                      
                      {/* Show sources if available */}
                      {message.context_chunks && message.role === 'assistant' && (
                        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
                          <div className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                            <FileText className="h-3.5 w-3.5" />
                            <span className="font-medium">
                              {(() => {
                                try {
                                  const contexts = JSON.parse(message.context_chunks);
                                  const count = Array.isArray(contexts) ? contexts.length : 0;
                                  return count > 0 ? `Based on ${count} source${count !== 1 ? 's' : ''}` : 'Based on available data';
                                } catch {
                                  return 'Based on available sources';
                                }
                              })()}
                            </span>
                          </div>
                        </div>
                      )}
                    </>
                  )}
                </div>
                
                {message.role === 'user' && (
                  <div className="flex-shrink-0">
                    <div className="h-10 w-10 rounded-xl bg-gradient-to-br from-gray-200 to-gray-300 dark:from-gray-700 dark:to-gray-600 flex items-center justify-center shadow-md">
                      <User className="h-6 w-6 text-gray-600 dark:text-gray-300" />
                    </div>
                  </div>
                )}
              </div>
            ))}
            
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Input Area */}
        <form onSubmit={handleSubmit} className="flex gap-3">
          <div className="flex-1 relative">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
              placeholder="Ask anything about your startup, meeting, or documents..."
            disabled={isLoading}
              className="pr-12 h-12 border-gray-300 dark:border-gray-600 focus:border-blue-500 dark:focus:border-blue-400 focus:ring-2 focus:ring-blue-200 dark:focus:ring-blue-800 rounded-xl"
            />
            {input.length > 0 && (
              <div className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400">
                {input.length}
              </div>
            )}
          </div>
          <Button 
            type="submit" 
            disabled={isLoading || !input.trim()}
            className="h-12 px-6 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white rounded-xl shadow-md disabled:opacity-50 disabled:cursor-not-allowed transition-all"
          >
            {isLoading ? (
              <Sparkles className="h-5 w-5 animate-spin" />
            ) : (
              <Send className="h-5 w-5" />
            )}
          </Button>
        </form>

        {/* Session Info */}
        <div className="mt-3 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <div className="flex items-center gap-4">
            <span className="flex items-center gap-1">
              <span className="inline-block w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
              {messages.filter(m => m.role === 'assistant' && !m.isLoading).length} responses
            </span>
            {currentSession && (
              <span>
                Started: {new Date(currentSession.createdAt).toLocaleDateString()}
              </span>
            )}
          </div>
          <Badge variant="outline" className="text-xs font-mono">
            {sessionId.slice(0, 16)}...
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}
