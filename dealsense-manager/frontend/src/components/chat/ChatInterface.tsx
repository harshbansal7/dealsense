/**
 * Next-Generation Chat Interface - Built from scratch
 * Features: Glassmorphism, advanced animations, syntax highlighting, 
 * message actions, typing indicators, source cards, and more
 */

'use client';

import { useState, useRef, useEffect, useCallback } from 'react';
import { 
  Send, Bot, User, Sparkles, Copy, RotateCw, ChevronDown, 
  Check, FileText, Clock, Search, Zap, ArrowDown,
  MessageCircle, Trash2, MoreVertical
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Badge } from '@/components/ui/badge';
import { chatbotApi, ChatMessage as ApiChatMessage, ContextChunk } from '@/lib/api';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import remarkBreaks from 'remark-breaks';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus, vs } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { motion, AnimatePresence } from 'framer-motion';
import toast, { Toaster } from 'react-hot-toast';

interface ChatInterfaceProps {
  agentId: string;
  sessionId?: string;
}

interface ChatMessage extends ApiChatMessage {
  isLoading?: boolean;
  timestamp?: Date;
}

interface ChatSession {
  sessionId: string;
  messages: ChatMessage[];
  createdAt: string;
  lastActivity: string;
  title?: string;
}

// Engaging thinking messages
const THINKING_MESSAGES = [
  'Analyzing documents and transcripts',
  'Searching for relevant information',
  'Processing query context',
  'Generating comprehensive response',
  'Synthesizing insights',
  'Reviewing data sources',
  'Formulating answer',
  'Cross-referencing information',
];

const getAllThinkingMessages = () => THINKING_MESSAGES;

// Storage utilities
const generateSessionId = () => `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
const getStorageKey = (agentId: string) => `dealsense_chat_v2_${agentId}`;

const saveChatSession = (agentId: string, session: ChatSession) => {
  try {
    const sessions = loadChatSessions(agentId);
    const updatedSessions = sessions.filter(s => s.sessionId !== session.sessionId);
    updatedSessions.unshift(session);
    localStorage.setItem(getStorageKey(agentId), JSON.stringify(updatedSessions.slice(0, 10)));
  } catch (error) {
    console.warn('Failed to save chat session:', error);
  }
};

const loadChatSessions = (agentId: string): ChatSession[] => {
  try {
    const stored = localStorage.getItem(getStorageKey(agentId));
    if (!stored) return [];
    return JSON.parse(stored);
  } catch {
    return [];
  }
};

// Typing indicator component
const TypingIndicator = () => (
  <div className="flex items-center gap-1 px-4 py-2">
    {[0, 1, 2].map((i) => (
      <motion.div
        key={i}
        className="w-2 h-2 bg-blue-500 rounded-full"
        animate={{ y: [0, -8, 0] }}
        transition={{
          duration: 0.6,
          repeat: Infinity,
          delay: i * 0.1,
        }}
      />
    ))}
  </div>
);

// Advanced thinking indicator with sophisticated progress animation
const ThinkingIndicator = ({ message, progress }: { message: string; progress: number }) => {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      className="space-y-3"
    >
      <div className="flex items-center gap-3">
        <Sparkles className="h-4 w-4 text-blue-600 dark:text-blue-400 animate-pulse" />
        <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
          {message}
        </span>
      </div>
      
      {/* Sophisticated progress bar with glow effect */}
      <div className="relative h-2 bg-gradient-to-r from-gray-200 via-gray-100 to-gray-200 dark:from-gray-700 dark:via-gray-800 dark:to-gray-700 rounded-full overflow-hidden shadow-inner">
        {/* Background shimmer effect */}
        <motion.div
          className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent"
          animate={{ x: ['-100%', '200%'] }}
          transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
        />
        
        {/* Main progress bar with gradient and glow */}
        <motion.div
          className="absolute inset-y-0 left-0 rounded-full"
          style={{
            background: 'linear-gradient(90deg, #3b82f6 0%, #8b5cf6 50%, #ec4899 100%)',
            boxShadow: '0 0 20px rgba(139, 92, 246, 0.5)',
          }}
          initial={{ width: '0%' }}
          animate={{ width: `${Math.max(0, Math.min(100, progress))}%` }}
          transition={{ duration: 0.3, ease: "easeOut" }}
        />
        
        {/* Animated shine effect on progress bar */}
        <motion.div
          className="absolute inset-y-0 left-0 rounded-full"
          style={{
            width: `${Math.max(0, Math.min(100, progress))}%`,
            background: 'linear-gradient(90deg, transparent 0%, rgba(255,255,255,0.3) 50%, transparent 100%)',
          }}
          animate={{ 
            backgroundPosition: ['0% 0%', '200% 0%'],
          }}
          transition={{ duration: 1.5, repeat: Infinity, ease: "linear" }}
        />
      </div>
      
      <div className="flex items-center justify-between text-xs">
        <span className="text-gray-500 dark:text-gray-400">
          Processing
        </span>
        <span className="font-mono font-medium text-blue-600 dark:text-blue-400">
          {Math.round(progress)}%
        </span>
      </div>
    </motion.div>
  );
};

// Code block with copy button
const CodeBlock = ({ language, value, theme }: { language: string; value: string; theme: 'light' | 'dark' }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard.writeText(value);
    setCopied(true);
    toast.success('Code copied!', { duration: 2000 });
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="relative group">
      <Button
        onClick={handleCopy}
        size="sm"
        variant="ghost"
        className="absolute right-2 top-2 opacity-0 group-hover:opacity-100 transition-opacity z-10"
      >
        {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
      </Button>
      <SyntaxHighlighter
        language={language}
        style={theme === 'dark' ? vscDarkPlus : vs}
        customStyle={{
          margin: 0,
          borderRadius: '0.5rem',
          fontSize: '0.875rem',
        }}
      >
        {value}
      </SyntaxHighlighter>
    </div>
  );
};

// Source citation card
const SourceCard = ({ chunks }: { chunks: ContextChunk[] }) => {
  const [expanded, setExpanded] = useState(false);

  if (!chunks || chunks.length === 0) return null;

  return (
    <motion.div
      initial={{ height: 0, opacity: 0 }}
      animate={{ height: 'auto', opacity: 1 }}
      className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700"
    >
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center justify-between w-full text-xs text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 transition-colors"
      >
        <span className="flex items-center gap-2">
          <FileText className="h-3.5 w-3.5" />
          <span className="font-medium">
            {chunks.length} source{chunks.length !== 1 ? 's' : ''} referenced
          </span>
        </span>
        <motion.div
          animate={{ rotate: expanded ? 180 : 0 }}
          transition={{ duration: 0.2 }}
        >
          <ChevronDown className="h-4 w-4" />
        </motion.div>
      </button>

      <AnimatePresence>
        {expanded && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: 'auto', opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            className="mt-2 space-y-2 overflow-hidden"
          >
            {chunks.slice(0, 3).map((chunk, idx) => (
              <div
                key={idx}
                className="p-2 bg-gray-50 dark:bg-gray-800/50 rounded-lg text-xs"
              >
                <div className="flex items-center gap-2 mb-1">
                  <Badge variant="outline" className="text-xs">
                    {chunk.display_source}
                  </Badge>
                  {chunk.page_number > 0 && (
                    <span className="text-gray-500">Page {chunk.page_number}</span>
                  )}
                  {chunk.similarity > 0 && (
                    <span className="text-green-600 dark:text-green-400">
                      {(chunk.similarity * 100).toFixed(0)}% match
                    </span>
                  )}
                </div>
                <p className="text-gray-600 dark:text-gray-400 line-clamp-2">
                  {chunk.text}
                </p>
              </div>
            ))}
          </motion.div>
        )}
      </AnimatePresence>
    </motion.div>
  );
};

// Message actions menu
const MessageActions = ({ 
  message, 
  onCopy, 
  onRegenerate, 
  onDelete 
}: { 
  message: ChatMessage;
  onCopy: () => void;
  onRegenerate: () => void;
  onDelete: () => void;
}) => {
  const [showMenu, setShowMenu] = useState(false);

  return (
    <div className="relative">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => setShowMenu(!showMenu)}
        className="opacity-0 group-hover:opacity-100 transition-opacity h-6 w-6 p-0"
      >
        <MoreVertical className="h-3.5 w-3.5" />
      </Button>

      <AnimatePresence>
        {showMenu && (
          <>
            <div 
              className="fixed inset-0 z-10" 
              onClick={() => setShowMenu(false)}
            />
            <motion.div
              initial={{ opacity: 0, scale: 0.95, y: -10 }}
              animate={{ opacity: 1, scale: 1, y: 0 }}
              exit={{ opacity: 0, scale: 0.95, y: -10 }}
              className="absolute right-0 top-8 z-20 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1"
            >
              <button
                onClick={() => { onCopy(); setShowMenu(false); }}
                className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
              >
                <Copy className="h-4 w-4" />
                Copy message
              </button>
              {message.role === 'assistant' && (
                <button
                  onClick={() => { onRegenerate(); setShowMenu(false); }}
                  className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                >
                  <RotateCw className="h-4 w-4" />
                  Regenerate
                </button>
              )}
              <button
                onClick={() => { onDelete(); setShowMenu(false); }}
                className="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2 text-red-600 dark:text-red-400"
              >
                <Trash2 className="h-4 w-4" />
                Delete
              </button>
            </motion.div>
          </>
        )}
      </AnimatePresence>
    </div>
  );
};

export function ChatInterface({ agentId, sessionId: initialSessionId }: ChatInterfaceProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [sessionId, setSessionId] = useState(initialSessionId || '');
  const [availableSessions, setAvailableSessions] = useState<ChatSession[]>([]);
  const [currentSession, setCurrentSession] = useState<ChatSession | null>(null);
  const [currentThinkingMessage, setCurrentThinkingMessage] = useState(getAllThinkingMessages()[0]);
  const [loadingProgress, setLoadingProgress] = useState(0);
  const [showScrollButton, setShowScrollButton] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark'>('light');
  
  const scrollAreaRef = useRef<HTMLDivElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Detect theme
  useEffect(() => {
    const isDark = document.documentElement.classList.contains('dark');
    setTheme(isDark ? 'dark' : 'light');
    
    const observer = new MutationObserver(() => {
      const isDark = document.documentElement.classList.contains('dark');
      setTheme(isDark ? 'dark' : 'light');
    });
    
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] });
    return () => observer.disconnect();
  }, []);

  // Load sessions on mount
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
        startNewChat();
      }
    } else if (sessions.length > 0) {
      const mostRecent = sessions[0];
      setCurrentSession(mostRecent);
      setMessages(mostRecent.messages);
      setSessionId(mostRecent.sessionId);
    } else {
      startNewChat();
    }
  }, [agentId, initialSessionId]);

  // Save session
  useEffect(() => {
    if (sessionId && messages.length > 0 && !messages.some(m => m.isLoading)) {
      const session: ChatSession = {
        sessionId,
        messages,
        createdAt: currentSession?.createdAt || new Date().toISOString(),
        lastActivity: new Date().toISOString(),
        title: messages[0]?.content.slice(0, 50) || 'New Chat',
      };
      saveChatSession(agentId, session);
      setCurrentSession(session);
      setAvailableSessions(loadChatSessions(agentId));
    }
  }, [messages, sessionId, agentId, currentSession?.createdAt]);

  // Handle scroll for show/hide scroll button
  const handleScroll = useCallback(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        const isNearBottom = scrollContainer.scrollHeight - scrollContainer.scrollTop - scrollContainer.clientHeight < 100;
        setShowScrollButton(!isNearBottom && messages.length > 3);
      }
    }
  }, [messages.length]);

  // Scroll to bottom function
  const scrollToBottom = useCallback(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.scrollTop = scrollContainer.scrollHeight;
      }
    }
    // Fallback to direct element scroll
    setTimeout(() => {
      messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }, 100);
  }, []);

  // Auto scroll
  useEffect(() => {
    scrollToBottom();
  }, [messages, scrollToBottom]);

  // Setup scroll listener
  useEffect(() => {
    if (scrollAreaRef.current) {
      const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]');
      if (scrollContainer) {
        scrollContainer.addEventListener('scroll', handleScroll);
        return () => scrollContainer.removeEventListener('scroll', handleScroll);
      }
    }
  }, [handleScroll]);

  // Rotate thinking messages and update progress with randomization
  useEffect(() => {
    if (isLoading) {
      const messages = getAllThinkingMessages();
      let messageIndex = 0;
      let progress = 0;
      let timeAt90 = 0; // Track when we hit 90%
      const startTime = Date.now();
      
      setCurrentThinkingMessage(messages[messageIndex]);
      setLoadingProgress(0);
      
      // Update progress: 0-90% over 8 seconds, then +1% every 2 seconds until 99%
      const progressInterval = setInterval(() => {
        const elapsed = Date.now() - startTime;
        
        let target: number;
        
        if (elapsed < 8000) {
          // Phase 1: 0-90% over 8 seconds with smooth variation
          const baseProgress = Math.min((elapsed / 8000) * 90, 90);
          
          // Add smooth sine wave variation (smaller than before to avoid going backwards)
          const variation = Math.sin(elapsed / 400) * 1.5; // ±1.5% variation
          const randomJitter = (Math.random() - 0.5) * 0.2; // ±0.1% micro-jitter
          
          target = baseProgress + variation + randomJitter;
          target = Math.max(0, Math.min(90, target));
        } else {
          // Phase 2: After 8 seconds, increment by 1% every 2 seconds
          if (timeAt90 === 0) {
            timeAt90 = elapsed; // Mark when we entered this phase
          }
          
          const timeSince90 = elapsed - timeAt90;
          const incrementsPassed = Math.floor(timeSince90 / 2000); // +1 every 2 seconds
          
          target = Math.min(90 + incrementsPassed, 99);
          
          // Add tiny jitter only
          target += (Math.random() - 0.5) * 0.1;
          target = Math.min(target, 99);
        }
        
        // Smoothly move current progress towards target
        const diff = target - progress;
        
        if (Math.abs(diff) < 0.05) {
          // Very close to target, move directly with micro-jitter
          progress = target + (Math.random() - 0.5) * 0.05;
        } else {
          // Smooth interpolation
          progress += diff * 0.25;
        }
        
        // Clamp to valid range
        progress = Math.max(0, Math.min(99, progress));
        
        setLoadingProgress(progress);
      }, 50); // Update every 50ms for ultra-smooth animation
      
      // Change message every 2.5 seconds
      const messageInterval = setInterval(() => {
        messageIndex = (messageIndex + 1) % messages.length;
        setCurrentThinkingMessage(messages[messageIndex]);
      }, 2500);
      
      return () => {
        clearInterval(progressInterval);
        clearInterval(messageInterval);
      };
    } else {
      setLoadingProgress(0);
    }
  }, [isLoading]);

  const startNewChat = () => {
    const newSessionId = generateSessionId();
    setSessionId(newSessionId);
    setMessages([]);
    setCurrentSession({
      sessionId: newSessionId,
      messages: [],
      createdAt: new Date().toISOString(),
      lastActivity: new Date().toISOString(),
      title: 'New Chat',
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isLoading) return;

    const userMessage: ChatMessage = {
      id: `user_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'user',
      content: input.trim(),
      token_count: 0,
      created_at: new Date().toISOString(),
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setIsLoading(true);

    const loadingMessage: ChatMessage = {
      id: `loading_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'assistant',
      content: '',
      token_count: 0,
      created_at: new Date().toISOString(),
      isLoading: true,
      timestamp: new Date(),
    };
    setMessages(prev => [...prev, loadingMessage]);

    try {
      const response = await chatbotApi.query(agentId, {
        query: userMessage.content,
        session_id: sessionId,
        top_k: 5,
      });

      const chatResponse = response.data;

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
            timestamp: new Date(),
          },
        ];
      });

      setSessionId(chatResponse.session_id);
    } catch (error) {
      console.error('Chat query failed:', error);
      toast.error('Failed to get response. Please try again.');
      
      setMessages(prev => prev.filter(m => m.id !== loadingMessage.id));
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyMessage = (content: string) => {
    navigator.clipboard.writeText(content);
    toast.success('Copied to clipboard!');
  };

  const handleRegenerateMessage = async (messageId: string) => {
    // Find the user message before this assistant message
    const messageIndex = messages.findIndex(m => m.id === messageId);
    if (messageIndex <= 0) return;
    
    const userMessage = messages[messageIndex - 1];
    if (userMessage.role !== 'user') return;

    // Remove the assistant message and regenerate
    setMessages(prev => prev.filter(m => m.id !== messageId));
    toast.loading('Regenerating response...');
    
    // Resubmit the query
    setIsLoading(true);
    const loadingMessage: ChatMessage = {
      id: `loading_${Date.now()}`,
      agent_id: agentId,
      session_id: sessionId,
      role: 'assistant',
      content: '',
      token_count: 0,
      created_at: new Date().toISOString(),
      isLoading: true,
      timestamp: new Date(),
    };
    setMessages(prev => [...prev, loadingMessage]);

    try {
      const response = await chatbotApi.query(agentId, {
        query: userMessage.content,
        session_id: sessionId,
        top_k: 5,
      });

      const chatResponse = response.data;

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
            timestamp: new Date(),
          },
        ];
      });

      toast.dismiss();
      toast.success('Response regenerated!');
    } catch (error) {
      console.error('Regenerate failed:', error);
      toast.dismiss();
      toast.error('Failed to regenerate. Please try again.');
      setMessages(prev => prev.filter(m => m.id !== loadingMessage.id));
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteMessage = (messageId: string) => {
    setMessages(prev => prev.filter(m => m.id !== messageId));
    toast.success('Message deleted');
  };

  const suggestions = [
    { icon: <FileText className="h-4 w-4" />, text: "Summarize all documents and extract key insights" },
    { icon: <Search className="h-4 w-4" />, text: "What are the main points from the meeting transcript?" },
    { icon: <Zap className="h-4 w-4" />, text: "Compare this pitch with industry competitors" },
    { icon: <MessageCircle className="h-4 w-4" />, text: "Identify potential risks and red flags" },
  ];

  return (
    <div className="flex flex-col h-full relative overflow-hidden">
      <Toaster position="top-right" />
      
      {/* Animated background */}
      <div className="absolute inset-0 bg-gradient-to-br from-blue-50 via-purple-50 to-pink-50 dark:from-gray-900 dark:via-purple-950 dark:to-gray-900 opacity-50" />
      <div className="absolute inset-0 backdrop-blur-3xl" />
      
      {/* Header with glassmorphism */}
      <motion.div
        initial={{ y: -20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="relative z-10 border-b border-white/20 dark:border-gray-700/20 bg-white/70 dark:bg-gray-900/70 backdrop-blur-xl"
      >
        <div className="px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="relative">
                <motion.div
                  animate={{ 
                    boxShadow: ['0 0 0 0 rgba(59, 130, 246, 0.4)', '0 0 0 10px rgba(59, 130, 246, 0)']
                  }}
                  transition={{ duration: 1.5, repeat: Infinity }}
                  className="absolute inset-0 rounded-full"
                />
                <div className="relative w-12 h-12 bg-gradient-to-br from-blue-500 via-purple-600 to-pink-500 rounded-full flex items-center justify-center shadow-lg">
                  <Bot className="h-6 w-6 text-white" />
                </div>
              </div>
              <div>
                <h2 className="text-lg font-bold bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 bg-clip-text text-transparent">
                  Document Intelligence Chat
                </h2>
                <p className="text-xs text-gray-600 dark:text-gray-400 flex items-center gap-1">
                  <span className="inline-block w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                  AI-powered document analysis with real-time web search
                </p>
              </div>
            </div>
            
            <div className="flex items-center gap-2">
              {availableSessions.length > 1 && (
                <select
                  value={sessionId}
                  onChange={(e) => {
                    const session = availableSessions.find(s => s.sessionId === e.target.value);
                    if (session) {
                      setCurrentSession(session);
                      setMessages(session.messages);
                      setSessionId(session.sessionId);
                    }
                  }}
                  className="text-xs px-3 py-2 rounded-lg bg-white/50 dark:bg-gray-800/50 border border-gray-200 dark:border-gray-700 backdrop-blur-sm hover:bg-white/70 dark:hover:bg-gray-800/70 transition-all"
                >
                  {availableSessions.map((session, idx) => (
                    <option key={session.sessionId} value={session.sessionId}>
                      {session.title || `Chat ${idx + 1}`}
                    </option>
                  ))}
                </select>
              )}
              <Button
                onClick={startNewChat}
                size="sm"
                className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white shadow-lg"
              >
                <Sparkles className="h-4 w-4 mr-1" />
                New Chat
              </Button>
            </div>
          </div>
        </div>
      </motion.div>

      {/* Messages Area */}
      <div className="flex-1 relative z-10 overflow-hidden">
        <ScrollArea 
          className="h-full w-full"
          ref={scrollAreaRef}
        >
          <div className="px-6 py-6 space-y-6 max-w-4xl mx-auto">
          {messages.length === 0 ? (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              className="text-center py-12"
            >
              <motion.div
                animate={{ y: [0, -10, 0] }}
                transition={{ duration: 3, repeat: Infinity, ease: "easeInOut" }}
                className="relative inline-block mb-8"
              >
                <div className="absolute inset-0 bg-gradient-to-r from-blue-400 to-purple-600 rounded-full blur-2xl opacity-40 animate-pulse" />
                <div className="relative w-20 h-20 bg-gradient-to-br from-blue-500 via-purple-600 to-pink-500 rounded-full flex items-center justify-center shadow-2xl">
                  <Bot className="h-10 w-10 text-white" />
                </div>
              </motion.div>
              
                <h3 className="text-2xl font-bold bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 bg-clip-text text-transparent mb-3">
                  What would you like to know?
                </h3>
                <p className="text-gray-600 dark:text-gray-400 mb-8 max-w-md mx-auto">
                  I can analyze your documents, meeting transcripts, and search the web for additional insights.
                </p>

              {/* Suggestion chips */}
              <div className="grid grid-cols-2 gap-3 max-w-2xl mx-auto">
                {suggestions.map((suggestion, idx) => (
                  <motion.button
                    key={idx}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: idx * 0.1 }}
                    whileHover={{ scale: 1.02, y: -2 }}
                    whileTap={{ scale: 0.98 }}
                    onClick={() => {
                      setInput(suggestion.text);
                      // Auto-submit the suggestion
                      setTimeout(() => {
                        const form = document.querySelector('form');
                        if (form) {
                          form.requestSubmit();
                        }
                      }, 100);
                    }}
                    className="p-4 bg-white/50 dark:bg-gray-800/50 backdrop-blur-sm rounded-xl border border-gray-200/50 dark:border-gray-700/50 hover:border-blue-300 dark:hover:border-blue-600 hover:shadow-lg transition-all text-left group cursor-pointer"
                  >
                    <div className="flex items-center gap-3 mb-2">
                      <span className="text-blue-600 dark:text-blue-400 group-hover:text-purple-600 dark:group-hover:text-purple-400 transition-colors">
                        {suggestion.icon}
                      </span>
                      <span className="ml-auto text-xs text-gray-400 group-hover:text-gray-600 dark:group-hover:text-gray-300">
                        Click to ask →
                      </span>
                    </div>
                    <p className="text-sm font-medium text-gray-700 dark:text-gray-300 leading-snug">
                      {suggestion.text}
                    </p>
                  </motion.button>
                ))}
              </div>
            </motion.div>
          ) : (
            messages.map((message, idx) => (
              <motion.div
                key={message.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: idx * 0.05 }}
                className={`flex gap-4 ${message.role === 'user' ? 'justify-end' : 'justify-start'} group`}
              >
                {message.role === 'assistant' && (
                  <div className="flex-shrink-0">
                    <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg">
                      <Bot className="h-5 w-5 text-white" />
                    </div>
                  </div>
                )}

                <div className={`flex-1 max-w-[75%] ${message.role === 'user' ? 'flex flex-col items-end' : ''}`}>
                  <div
                    className={`rounded-2xl backdrop-blur-sm shadow-lg ${
                      message.role === 'user'
                        ? 'bg-gradient-to-br from-blue-600 to-purple-600 text-white px-6 py-3'
                        : 'bg-white/70 dark:bg-gray-800/70 border border-gray-200/50 dark:border-gray-700/50 px-6 py-4'
                    }`}
                  >
                    {message.isLoading ? (
                      <ThinkingIndicator message={currentThinkingMessage} progress={loadingProgress} />
                    ) : (
                      <>
                        {message.role === 'user' ? (
                          <p className="text-sm leading-relaxed whitespace-pre-wrap">
                            {message.content}
                          </p>
                        ) : (
                          <div className="prose prose-sm dark:prose-invert max-w-none prose-headings:font-bold prose-headings:text-gray-900 dark:prose-headings:text-white prose-h1:text-xl prose-h2:text-lg prose-h3:text-base prose-p:text-gray-700 dark:prose-p:text-gray-300 prose-p:leading-relaxed prose-p:my-2 prose-ul:my-2 prose-ol:my-2 prose-li:my-1 prose-strong:text-gray-900 dark:prose-strong:text-white prose-strong:font-semibold">
                            <ReactMarkdown
                              remarkPlugins={[remarkGfm, remarkBreaks]}
                              components={{
                                p: ({ children }) => (
                                  <p className="my-2 leading-relaxed text-sm">
                                    {children}
                                  </p>
                                ),
                                h1: ({ children }) => (
                                  <h1 className="text-xl font-bold mt-4 mb-2 text-gray-900 dark:text-white">
                                    {children}
                                  </h1>
                                ),
                                h2: ({ children }) => (
                                  <h2 className="text-lg font-bold mt-3 mb-2 text-gray-900 dark:text-white">
                                    {children}
                                  </h2>
                                ),
                                h3: ({ children }) => (
                                  <h3 className="text-base font-bold mt-3 mb-1 text-gray-900 dark:text-white">
                                    {children}
                                  </h3>
                                ),
                                ul: ({ children }) => (
                                  <ul className="my-2 ml-4 list-disc space-y-1">
                                    {children}
                                  </ul>
                                ),
                                ol: ({ children }) => (
                                  <ol className="my-2 ml-4 list-decimal space-y-1">
                                    {children}
                                  </ol>
                                ),
                                li: ({ children }) => (
                                  <li className="text-sm text-gray-700 dark:text-gray-300">
                                    {children}
                                  </li>
                                ),
                                strong: ({ children }) => (
                                  <strong className="font-semibold text-gray-900 dark:text-white">
                                    {children}
                                  </strong>
                                ),
                                em: ({ children }) => (
                                  <em className="italic text-gray-700 dark:text-gray-300">
                                    {children}
                                  </em>
                                ),
                                blockquote: ({ children }) => (
                                  <blockquote className="border-l-4 border-blue-500 pl-4 my-2 italic text-gray-700 dark:text-gray-400">
                                    {children}
                                  </blockquote>
                                ),
                                a: ({ href, children }) => (
                                  <a 
                                    href={href} 
                                    target="_blank" 
                                    rel="noopener noreferrer"
                                    className="text-blue-600 dark:text-blue-400 hover:underline"
                                  >
                                    {children}
                                  </a>
                                ),
                                code({ inline, className, children, ...props }: {
                                  inline?: boolean;
                                  className?: string;
                                  children?: React.ReactNode;
                                }) {
                                  const match = /language-(\w+)/.exec(className || '');
                                  const value = String(children).replace(/\n$/, '');
                                  
                                  return !inline && match ? (
                                    <CodeBlock 
                                      language={match[1]} 
                                      value={value}
                                      theme={theme}
                                    />
                                  ) : (
                                    <code className="px-1.5 py-0.5 rounded bg-gray-100 dark:bg-gray-900 text-purple-600 dark:text-purple-400 text-xs font-mono" {...props}>
                                      {children}
                                    </code>
                                  );
                                },
                              }}
                            >
                              {message.content}
                            </ReactMarkdown>
                          </div>
                        )}

                        {/* Sources */}
                        {message.context_chunks && message.role === 'assistant' && (
                          <SourceCard 
                            chunks={JSON.parse(message.context_chunks)} 
                          />
                        )}
                      </>
                    )}
                  </div>

                  {/* Timestamp and actions */}
                  <div className="flex items-center gap-2 mt-1 px-2">
                    {message.timestamp && (
                      <span className="text-xs text-gray-500 dark:text-gray-500 flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {new Date(message.timestamp).toLocaleTimeString([], { 
                          hour: '2-digit', 
                          minute: '2-digit' 
                        })}
                      </span>
                    )}
                    {!message.isLoading && (
                      <MessageActions
                        message={message}
                        onCopy={() => handleCopyMessage(message.content)}
                        onRegenerate={() => handleRegenerateMessage(message.id)}
                        onDelete={() => handleDeleteMessage(message.id)}
                      />
                    )}
                  </div>
                </div>

                {message.role === 'user' && (
                  <div className="flex-shrink-0">
                    <div className="w-10 h-10 bg-gradient-to-br from-gray-400 to-gray-600 rounded-xl flex items-center justify-center shadow-lg">
                      <User className="h-5 w-5 text-white" />
                    </div>
                  </div>
                )}
              </motion.div>
            ))
          )}
          <div ref={messagesEndRef} />
        </div>
        </ScrollArea>
      </div>

      {/* Scroll to bottom button */}
      <AnimatePresence>
        {showScrollButton && (
          <motion.button
            initial={{ opacity: 0, scale: 0.8 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0, scale: 0.8 }}
            onClick={scrollToBottom}
            className="absolute bottom-24 right-6 z-20 w-10 h-10 bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-full shadow-lg border border-gray-200 dark:border-gray-700 flex items-center justify-center hover:bg-white dark:hover:bg-gray-800 transition-colors"
          >
            <ArrowDown className="h-5 w-5 text-gray-600 dark:text-gray-400" />
          </motion.button>
        )}
      </AnimatePresence>

      {/* Input Area with glassmorphism */}
      <motion.div
        initial={{ y: 20, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        className="relative z-10 border-t border-white/20 dark:border-gray-700/20 bg-white/70 dark:bg-gray-900/70 backdrop-blur-xl"
      >
        <form onSubmit={handleSubmit} className="px-6 py-4">
          <div className="flex items-end gap-3 max-w-4xl mx-auto">
            <div className="flex-1 relative">
              <Input
                ref={inputRef}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder="Ask anything... (Try asking about market trends, competitors, or insights)"
                disabled={isLoading}
                className="pr-12 h-12 bg-white/50 dark:bg-gray-800/50 backdrop-blur-sm border-gray-300 dark:border-gray-600 focus:border-blue-500 dark:focus:border-blue-400 focus:ring-2 focus:ring-blue-200 dark:focus:ring-blue-900 rounded-xl shadow-sm text-sm"
              />
              {input.length > 0 && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-gray-400"
                >
                  {input.length}
                </motion.div>
              )}
            </div>
            
            <Button
              type="submit"
              disabled={isLoading || !input.trim()}
              className="h-12 px-6 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white rounded-xl shadow-lg disabled:opacity-50 disabled:cursor-not-allowed transition-all"
            >
              {isLoading ? (
                <TypingIndicator />
              ) : (
                <>
                  <Send className="h-5 w-5 mr-2" />
                  Send
                </>
              )}
            </Button>
          </div>

          {/* Info footer */}
          <div className="mt-3 text-center text-xs text-gray-500 dark:text-gray-500">
            AI can make mistakes. Verify important information.
          </div>
        </form>
      </motion.div>
    </div>
  );
}

