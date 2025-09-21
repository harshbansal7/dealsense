/**
 * Global state management using Zustand.
 */

import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { Agent, MeetingInfo, WebSocketManager, SessionWebSocketManager, WebSocketMessage } from './api';

interface AgentState {
  agents: Agent[];
  meetings: MeetingInfo[];
  selectedAgent: Agent | null;
  isLoading: boolean;
  error: string | null;
  webSocketManagers: Map<string, WebSocketManager>;
  sessionWebSocket: SessionWebSocketManager | null;
  sessionWebSocketListeners: Set<(message: WebSocketMessage) => void>;
  isConnecting: boolean; // Add flag to prevent concurrent connections
}

interface AgentActions {
  addAgent: (agent: Agent) => void;
  removeAgent: (id: string) => void;
  updateAgent: (agent: Agent) => void;
  setAgents: (agents: Agent[]) => void;
  setSelectedAgent: (agent: Agent | null) => void;
  setLoading: (isLoading: boolean) => void;
  setError: (error: string | null) => void;
  addMeeting: (meeting: MeetingInfo) => void;
  removeMeeting: (id: string) => void;
  setMeetings: (meetings: MeetingInfo[]) => void;
  connectWebSocket: (agentId: string, onMessage: (message: WebSocketMessage) => void) => void;
  disconnectWebSocket: (agentId: string) => void;
  addSessionWebSocketListener: (listener: (message: WebSocketMessage) => void) => () => void;
  ensureSessionWebSocket: () => void;
  disconnectSessionWebSocket: () => void;
  connectSessionWebSocket: (onMessage: (message: WebSocketMessage) => void) => void;
}

export const useAgentStore = create<AgentState & AgentActions>()(
  devtools(
    (set, get) => ({
      // State
      agents: [],
      meetings: [],
      selectedAgent: null,
      isLoading: false,
      error: null,
      webSocketManagers: new Map(),
      sessionWebSocket: null,
      sessionWebSocketListeners: new Set(),
      isConnecting: false,

      // Actions
      setAgents: (agents) => set({ agents }),
      setMeetings: (meetings) => set({ meetings }),
      addAgent: (agent) =>
        set((state) => ({
          agents: [...state.agents, agent],
        })),
      updateAgent: (updatedAgent) =>
        set((state) => {
          // If the agent is newly created and has auto_join enabled, optimistically update its status
          const optimisticAgent = { ...updatedAgent };
          if (updatedAgent.status === 'created' && updatedAgent.config?.auto_join) {
            optimisticAgent.status = 'starting';
          }

          return {
            agents: state.agents.map((agent) =>
              agent.id === optimisticAgent.id ? optimisticAgent : agent
            ),
            selectedAgent:
              state.selectedAgent?.id === optimisticAgent.id ? optimisticAgent : state.selectedAgent,
          };
        }),
      removeAgent: (agentId) =>
        set((state) => ({
          agents: state.agents.filter((agent) => agent.id !== agentId),
          selectedAgent: state.selectedAgent?.id === agentId ? null : state.selectedAgent,
        })),
      setSelectedAgent: (agent) => set({ selectedAgent: agent }),
      setLoading: (isLoading) => set({ isLoading }),
      setError: (error) => set({ error }),

      connectWebSocket: (agentId, onMessage) => {
        const state = get();
        const existingManager = state.webSocketManagers.get(agentId);

        if (existingManager) {
          existingManager.disconnect();
        }

        const manager = new WebSocketManager(
          agentId,
          onMessage,
          (error) => {
            console.error(`WebSocket error for agent ${agentId}:`, error);
          },
          () => {
            console.log(`WebSocket closed for agent ${agentId}`);
          }
        );

        manager.connect();
        state.webSocketManagers.set(agentId, manager);
      },

      disconnectWebSocket: (agentId) => {
        const state = get();
        const manager = state.webSocketManagers.get(agentId);
        if (manager) {
          manager.disconnect();
          state.webSocketManagers.delete(agentId);
        }
      },

      addSessionWebSocketListener: (listener) => {
        const state = get();
        
        // Add listener to set
        state.sessionWebSocketListeners.add(listener);
        
        // Ensure we have a WebSocket connection (but prevent concurrent attempts)
        if (!state.sessionWebSocket && !state.isConnecting) {
          set({ isConnecting: true });
          
          const manager = new SessionWebSocketManager(
            (message) => {
              // Broadcast to all listeners with error handling
              const currentState = get();
              currentState.sessionWebSocketListeners.forEach(listener => {
                try {
                  listener(message);
                } catch (error) {
                  console.error('Error in WebSocket listener:', error);
                  // Remove broken listener
                  currentState.sessionWebSocketListeners.delete(listener);
                }
              });
            },
            (error) => {
              console.error('Session WebSocket error:', error);
              // Reset connecting flag on error
              set({ isConnecting: false });
            },
            () => {
              console.log('Session WebSocket closed');
              // Reset connecting flag on close
              set({ isConnecting: false });
            }
          );

          try {
            manager.connect();
            set({ sessionWebSocket: manager, isConnecting: false });
          } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            set({ isConnecting: false });
          }
        }

        // Return cleanup function
        return () => {
          const currentState = get();
          currentState.sessionWebSocketListeners.delete(listener);
          
          // If no more listeners, disconnect after a delay to prevent rapid reconnections
          if (currentState.sessionWebSocketListeners.size === 0) {
            setTimeout(() => {
              const finalState = get();
              if (finalState.sessionWebSocketListeners.size === 0 && finalState.sessionWebSocket) {
                console.log('No more listeners, disconnecting WebSocket');
                finalState.sessionWebSocket.disconnect();
                set({ sessionWebSocket: null, isConnecting: false });
              }
            }, 5000); // Increased delay to 5 seconds
          }
        };
      },

      ensureSessionWebSocket: () => {
        const state = get();
        if (!state.sessionWebSocket) {
          const manager = new SessionWebSocketManager(
            (message) => {
              // Broadcast to all listeners
              state.sessionWebSocketListeners.forEach(listener => {
                try {
                  listener(message);
                } catch (error) {
                  console.error('Error in WebSocket listener:', error);
                }
              });
            },
            (error) => {
              console.error('Session WebSocket error:', error);
            },
            () => {
              console.log('Session WebSocket closed');
            }
          );

          manager.connect();
          set({ sessionWebSocket: manager });
        }
      },

      disconnectSessionWebSocket: () => {
        const state = get();
        console.log('Manually disconnecting session WebSocket');
        if (state.sessionWebSocket) {
          state.sessionWebSocket.disconnect();
          set({ 
            sessionWebSocket: null, 
            sessionWebSocketListeners: new Set(),
            isConnecting: false 
          });
        }
      },

      connectSessionWebSocket: (onMessage) => {
        const state = get();
        
        // Disconnect existing session WebSocket if any
        if (state.sessionWebSocket) {
          state.sessionWebSocket.disconnect();
        }

        const manager = new SessionWebSocketManager(
          onMessage,
          (error) => {
            console.error('Session WebSocket error:', error);
          },
          () => {
            console.log('Session WebSocket closed');
          }
        );

        manager.connect();
        set({ sessionWebSocket: manager });
      },
    }),
    {
      name: 'agent-store',
    }
  )
);

// UI State
interface UIState {
  sidebarOpen: boolean;
  theme: 'light' | 'dark' | 'system';
  notifications: Notification[];
}

interface UIActions {
  setSidebarOpen: (open: boolean) => void;
  setTheme: (theme: 'light' | 'dark' | 'system') => void;
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;
}

interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message: string;
  timestamp: Date;
  duration?: number;
}

export const useUIStore = create<UIState & UIActions>()(
  devtools(
    (set, get) => ({
      // State
      sidebarOpen: true,
      theme: 'system',
      notifications: [],

      // Actions
      setSidebarOpen: (open) => set({ sidebarOpen: open }),
      setTheme: (theme) => set({ theme }),
      addNotification: (notification) => {
        const id = Date.now().toString();
        const newNotification: Notification = {
          id,
          timestamp: new Date(),
          duration: 5000,
          ...notification,
        };

        set((state) => ({
          notifications: [...state.notifications, newNotification],
        }));

        // Auto-remove after duration
        if (newNotification.duration) {
          setTimeout(() => {
            get().removeNotification(id);
          }, newNotification.duration);
        }
      },
      removeNotification: (id) =>
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        })),
      clearNotifications: () => set({ notifications: [] }),
    }),
    {
      name: 'ui-store',
    }
  )
);

// Selectors
export const useAgents = () => useAgentStore((state) => state.agents);
export const useSelectedAgent = () => useAgentStore((state) => state.selectedAgent);
export const useMeetings = () => useAgentStore((state) => state.meetings);
export const useIsLoading = () => useAgentStore((state) => state.isLoading);
export const useError = () => useAgentStore((state) => state.error);

export const useSidebarOpen = () => useUIStore((state) => state.sidebarOpen);
export const useTheme = () => useUIStore((state) => state.theme);
export const useNotifications = () => useUIStore((state) => state.notifications);
