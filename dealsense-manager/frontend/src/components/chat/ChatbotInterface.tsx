/**
 * Chatbot interface for querying meeting and document data
 */

'use client';

import { useState, useRef, useEffect } from 'react';
import { Send, Bot, User, Loader2, MessageSquare, FileText, RotateCcw } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { chatbotApi, ChatMessage as ApiChatMessage } from '@/lib/api';

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

const generateSessionId = () => {
  return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
};

const getStorageKey = (agentId: string) => `dealsense_chat_${agentId}`;

const saveChatSession = (agentId: string, session: ChatSession) => {
  try {
    const sessions = loadChatSessions(agentId);
    const updatedSessions = sessions.filter(s => s.sessionId !== session.sessionId);
    updatedSessions.unshift(session); // Add to front (most recent)

    // Keep only last 5 sessions per agent
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

const clearChatSessions = (agentId: string) => {
  try {
    localStorage.removeItem(getStorageKey(agentId));
  } catch (error) {
    console.warn('Failed to clear chat sessions:', error);
  }
};

export function ChatbotInterface({ agentId, sessionId: initialSessionId }: ChatbotInterfaceProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState(initialSessionId || '');
  const [availableSessions, setAvailableSessions] = useState<ChatSession[]>([]);
  const [currentSession, setCurrentSession] = useState<ChatSession | null>(null);
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  // Load existing chat sessions on mount
  useEffect(() => {
    const sessions = loadChatSessions(agentId);
    setAvailableSessions(sessions);

    // If initialSessionId is provided, try to load that specific session
    if (initialSessionId) {
      const targetSession = sessions.find(s => s.sessionId === initialSessionId);
      if (targetSession) {
        setCurrentSession(targetSession);
        setMessages(targetSession.messages);
        setSessionId(targetSession.sessionId);
      } else {
        // Session not found, create new one with the provided ID
        setSessionId(initialSessionId);
        setCurrentSession({
          sessionId: initialSessionId,
          messages: [],
          createdAt: new Date().toISOString(),
          lastActivity: new Date().toISOString(),
        });
      }
    } else if (sessions.length > 0) {
      // Load most recent session
      const mostRecent = sessions[0];
      setCurrentSession(mostRecent);
      setMessages(mostRecent.messages);
      setSessionId(mostRecent.sessionId);
    } else {
      // No existing sessions, create new one
      startNewChat();
    }
  }, [agentId]); // eslint-disable-line react-hooks/exhaustive-deps -- initialSessionId intentionally excluded to avoid infinite loops

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

      // Update available sessions list
      const sessions = loadChatSessions(agentId);
      setAvailableSessions(sessions);
    }
  }, [messages, sessionId, agentId, currentSession?.createdAt]);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

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

  const deleteChatSession = (sessionIdToDelete: string) => {
    try {
      const sessions = loadChatSessions(agentId);
      const updatedSessions = sessions.filter(s => s.sessionId !== sessionIdToDelete);
      localStorage.setItem(getStorageKey(agentId), JSON.stringify(updatedSessions));
      setAvailableSessions(updatedSessions);

      // If we deleted the current session, start a new one
      if (sessionId === sessionIdToDelete) {
        startNewChat();
      }
    } catch (error) {
      console.warn('Failed to delete chat session:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!input.trim() || isLoading) return;

    const userMessage: ChatMessage = {
      id: `temp_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'user',
      content: input.trim(),
      token_count: 0,
      created_at: new Date().toISOString(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);

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
    setMessages(prev => [...prev, loadingMessage]);

    try {
      const response = await chatbotApi.query(agentId, {
        query: userMessage.content,
        session_id: sessionId,
        top_k: 5,
      });

      const chatResponse = response.data;

      // Remove loading message and add actual response
      setMessages(prev => {
        const filtered = prev.filter(m => m.id !== loadingMessage.id);
        return [
          ...filtered,
          {
            id: `assistant_${Date.now()}`,
            agent_id: agentId,
            session_id: chatResponse.session_id,
            role: 'assistant',
            content: chatResponse.response,
            token_count: chatResponse.token_count,
            created_at: new Date().toISOString(),
            context_chunks: JSON.stringify(chatResponse.context_chunks),
          },
        ];
      });

      setSessionId(chatResponse.session_id);
    } catch (error) {
      console.error('Chat query failed:', error);
      
      // Remove loading message and show error
      setMessages(prev => {
        const filtered = prev.filter(m => m.id !== loadingMessage.id);
        return [
          ...filtered,
          {
            id: `error_${Date.now()}`,
            agent_id: agentId,
            session_id: sessionId,
            role: 'assistant',
            content: 'Sorry, I encountered an error processing your request. Please try again.',
            token_count: 0,
            created_at: new Date().toISOString(),
          },
        ];
      });
    } finally {
      setIsLoading(false);
    }
  };

  const getSuggestionQuestions = () => {
    return [
      "What were the main points discussed in the meeting?",
      "What action items were identified?",
      "Summarize the key insights from the uploaded documents",
      "What are the strengths and weaknesses mentioned?",
      "Who are the main stakeholders involved?",
    ];
  };

  const handleSuggestionClick = (question: string) => {
    setInput(question);
  };

  return (
    <Card className="h-full flex flex-col">
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5" />
              AI Assistant
            </CardTitle>
            <CardDescription>
              Ask questions about the meeting transcript and uploaded documents
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {availableSessions.length > 0 && (
              <div className="flex items-center gap-2">
                <select
                  value={sessionId}
                  onChange={(e) => {
                    const selectedSession = availableSessions.find(s => s.sessionId === e.target.value);
                    if (selectedSession) {
                      loadChatSession(selectedSession);
                    }
                  }}
                  className="text-xs px-2 py-1 border rounded bg-white dark:bg-gray-800"
                >
                  {availableSessions.map((session) => (
                    <option key={session.sessionId} value={session.sessionId}>
                      Session {session.sessionId.slice(0, 8)}... ({session.messages.length} msgs)
                    </option>
                  ))}
                </select>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={startNewChat}
                  className="text-xs"
                >
                  <RotateCcw className="h-3 w-3 mr-1" />
                  New Chat
                </Button>
              </div>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex-1 flex flex-col min-h-0 pb-4">
        {/* Messages Area */}
        <ScrollArea className="flex-1 pr-4 mb-4" ref={scrollAreaRef}>
          <div className="space-y-4">
            {messages.length === 0 && (
              <div className="text-center py-8">
                <Bot className="h-16 w-16 text-gray-400 mx-auto mb-4" />
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
                  {currentSession ? 'Continue your conversation!' : 'Ask me anything!'}
                </h3>
                <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                  {currentSession
                    ? 'Pick up where you left off or start fresh'
                    : 'I can help you analyze meeting transcripts and documents'
                  }
                </p>

                {/* Suggested Questions */}
                <div className="text-left max-w-2xl mx-auto">
                  <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">
                    Suggested questions:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    {getSuggestionQuestions().map((question, index) => (
                      <Button
                        key={index}
                        variant="outline"
                        size="sm"
                        onClick={() => handleSuggestionClick(question)}
                        className="text-xs"
                      >
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
                className={`flex gap-3 ${
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                }`}
              >
                {message.role === 'assistant' && (
                  <div className="flex-shrink-0">
                    <div className="h-8 w-8 rounded-full bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center">
                      <Bot className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                    </div>
                  </div>
                )}
                
                <div
                  className={`max-w-[80%] rounded-lg px-4 py-2 ${
                    message.role === 'user'
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-100 dark:bg-gray-800 text-gray-900 dark:text-white'
                  }`}
                >
                  {message.isLoading ? (
                    <div className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span className="text-sm">Thinking...</span>
                    </div>
                  ) : (
                    <>
                      <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                      
                      {/* Show sources if available */}
                      {message.context_chunks && message.role === 'assistant' && (
                        <div className="mt-2 pt-2 border-t border-gray-200 dark:border-gray-700">
                          <div className="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-400">
                            <FileText className="h-3 w-3" />
                            <span>
                              {(() => {
                                try {
                                  const contexts = JSON.parse(message.context_chunks);
                                  return `Based on ${Array.isArray(contexts) ? contexts.length : 0} source(s)`;
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
                    <div className="h-8 w-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center">
                      <User className="h-5 w-5 text-gray-600 dark:text-gray-400" />
                    </div>
                  </div>
                )}
              </div>
            ))}
            
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Input Area */}
        <form onSubmit={handleSubmit} className="flex gap-2">
          <Input
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask a question about the meeting or documents..."
            disabled={isLoading}
            className="flex-1"
          />
          <Button type="submit" disabled={isLoading || !input.trim()}>
            {isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </form>

        {/* Session Info */}
        <div className="mt-2 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
          <div className="flex items-center gap-4">
            <span>{messages.filter(m => m.role === 'assistant' && !m.isLoading).length} responses</span>
            {currentSession && (
              <span>
                Started: {new Date(currentSession.createdAt).toLocaleDateString()}
              </span>
            )}
          </div>
          <Badge variant="outline" className="text-xs">
            Session: {sessionId.slice(0, 12)}...
          </Badge>
        </div>
      </CardContent>
    </Card>
  );
}

