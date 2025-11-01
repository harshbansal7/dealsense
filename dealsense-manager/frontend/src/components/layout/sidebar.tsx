/**
 * Production-grade sidebar with striking, modern design
 */

'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState, useEffect } from 'react';
import {
  Bot,
  ChevronLeft,
  ChevronRight,
  LogOut,
  Video,
  Plus,
  Activity,
  Sparkles,
  Zap,
} from 'lucide-react';

import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useUIStore, useAgentStore } from '@/lib/store';

const navigation = [
  { name: 'Agents', href: '/agents', icon: Bot, color: 'from-purple-500 to-pink-500' },
  { name: 'Meetings', href: '/meetings', icon: Video, color: 'from-green-500 to-emerald-500' },
];

function StatCard({ icon, label, value, color }: { 
  icon: React.ReactNode; 
  label: string; 
  value: number; 
  color: string;
}) {
  const colorClasses = {
    blue: 'from-blue-500/20 to-cyan-500/20 border-blue-500/30',
    purple: 'from-purple-500/20 to-pink-500/20 border-purple-500/30',
    green: 'from-green-500/20 to-emerald-500/20 border-green-500/30',
  };
  
  const iconColors = {
    blue: 'text-blue-400',
    purple: 'text-purple-400',
    green: 'text-green-400',
  };
  
  return (
    <div className={cn(
      'relative overflow-hidden rounded-xl border backdrop-blur-sm p-4 group hover:scale-[1.02] transition-all duration-300 cursor-pointer',
      'bg-gradient-to-br',
      colorClasses[color as keyof typeof colorClasses]
    )}>
      {/* Glow effect */}
      <div className="absolute inset-0 bg-gradient-to-br from-white/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
      
      <div className="relative flex items-start justify-between">
        <div className="flex-1">
          <div className={cn('mb-2', iconColors[color as keyof typeof iconColors])}>
            {icon}
          </div>
          <div className="text-2xl font-black text-white mb-1">{value}</div>
          <div className="text-xs font-medium text-gray-400">{label}</div>
        </div>
      </div>
    </div>
  );
}

interface SidebarProps {
  className?: string;
}

export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname();
  const { sidebarOpen, setSidebarOpen } = useUIStore();
  const agents = useAgentStore((state) => state.agents);
  const [time, setTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  const activeAgents = agents.filter(a => a.status === 'running').length;
  const totalAgents = agents.length;

  return (
    <div
      className={cn(
        'flex flex-col relative overflow-hidden transition-all duration-300',
        'bg-gradient-to-b from-black via-gray-900 to-black',
        'border-r border-white/10',
        sidebarOpen ? 'w-72' : 'w-20',
        className
      )}
    >
      {/* Animated background gradient */}
      <div className="absolute inset-0 bg-gradient-to-br from-blue-600/5 via-purple-600/5 to-pink-600/5 pointer-events-none" />
      
      {/* Grid pattern overlay */}
      <div className="absolute inset-0 opacity-[0.02] pointer-events-none" style={{
        backgroundImage: `
          linear-gradient(to right, white 1px, transparent 1px),
          linear-gradient(to bottom, white 1px, transparent 1px)
        `,
        backgroundSize: '20px 20px',
      }} />

      {/* Header */}
      <div className="relative flex items-center justify-between p-5 border-b border-white/10">
        <div className={cn('flex items-center gap-3', !sidebarOpen && 'justify-center')}>
          <div className="relative group">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl blur-md opacity-75 group-hover:opacity-100 transition-opacity" />
            <div className="relative w-10 h-10 bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500 rounded-2xl flex items-center justify-center transform group-hover:scale-110 transition-transform">
              <Bot className="h-5 w-5 text-white" />
            </div>
          </div>
          {sidebarOpen && (
            <div>
              <div className="text-base font-black bg-gradient-to-r from-white via-blue-100 to-purple-100 bg-clip-text text-transparent">
                DealSense
              </div>
              <div className="text-[10px] text-gray-500 -mt-0.5 font-medium">AI Platform</div>
            </div>
          )}
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setSidebarOpen(!sidebarOpen)}
          className="h-8 w-8 p-0 hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
        >
          {sidebarOpen ? (
            <ChevronLeft className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </Button>
      </div>

      {/* Content */}
      <div className="flex-1 relative px-3 py-6 space-y-6 overflow-y-auto custom-scrollbar">
        {/* Quick Create Button */}
        <div>
          <Link href="/agents/create">
            <Button
              className={cn(
                'relative w-full group overflow-hidden',
                'bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600',
                'hover:from-blue-700 hover:via-purple-700 hover:to-pink-700',
                'text-white font-bold shadow-xl shadow-purple-500/30',
                'hover:shadow-2xl hover:shadow-purple-500/50',
                'transform hover:scale-[1.02] transition-all duration-300',
                'rounded-xl',
                sidebarOpen ? 'justify-start gap-3 px-4 py-6' : 'justify-center p-4'
              )}
            >
              <div className="absolute inset-0 bg-gradient-to-r from-white/0 via-white/10 to-white/0 translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000" />
              <Plus className={cn('relative z-10', sidebarOpen ? 'h-5 w-5' : 'h-6 w-6')} />
              {sidebarOpen && (
                <div className="relative z-10 text-left">
                  <div className="text-sm font-bold">Create Agent</div>
                  <div className="text-xs opacity-90">Start a new AI agent</div>
                </div>
              )}
            </Button>
          </Link>
        </div>

        {/* Navigation */}
        <nav className="space-y-2">
          {sidebarOpen && (
            <div className="px-3 mb-3">
              <div className="text-xs font-bold text-gray-500 uppercase tracking-wider">Navigation</div>
            </div>
          )}
          {navigation.map((item) => {
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.name}
                href={item.href}
                className={cn(
                  'relative flex items-center gap-3 px-4 py-3 rounded-xl text-sm font-semibold transition-all duration-300 group',
                  isActive
                    ? 'bg-white/10 text-white border border-white/20 shadow-lg'
                    : 'text-gray-400 hover:text-white hover:bg-white/5'
                )}
              >
                {/* Active indicator */}
                {isActive && (
                  <div className={cn(
                    'absolute left-0 w-1 h-8 rounded-r-full bg-gradient-to-b',
                    item.color
                  )} />
                )}
                
                {/* Icon with gradient on hover */}
                <div className={cn(
                  'relative flex-shrink-0',
                  sidebarOpen ? '' : 'mx-auto'
                )}>
                  {isActive && (
                    <div className={cn(
                      'absolute inset-0 bg-gradient-to-br blur-lg opacity-50',
                      item.color
                    )} />
                  )}
                  <div className={cn(
                    'relative w-9 h-9 rounded-xl flex items-center justify-center transition-all',
                    isActive 
                      ? `bg-gradient-to-br ${item.color}` 
                      : 'bg-white/5 group-hover:bg-white/10'
                  )}>
                    <item.icon className="h-5 w-5" />
                  </div>
                </div>
                
                {sidebarOpen && (
                  <span className="relative z-10">{item.name}</span>
                )}
                
                {/* Hover effect */}
                {!isActive && (
                  <div className="absolute inset-0 bg-gradient-to-r from-white/0 via-white/5 to-white/0 opacity-0 group-hover:opacity-100 transition-opacity rounded-xl" />
                )}
              </Link>
            );
          })}
        </nav>

        {/* Quick Stats */}
        {sidebarOpen && (
          <div className="space-y-4">
            <div className="px-3">
              <div className="flex items-center gap-2 text-xs font-bold text-gray-500 uppercase tracking-wider">
                <Sparkles className="w-3 h-3" />
                <span>Live Stats</span>
              </div>
            </div>
            <div className="space-y-3">
              <StatCard
                icon={<Zap className="h-4 w-4" />}
                label="Active Now"
                value={activeAgents}
                color="blue"
              />
              <StatCard
                icon={<Bot className="h-4 w-4" />}
                label="Total Agents"
                value={totalAgents}
                color="purple"
              />
            </div>
          </div>
        )}

        {/* Time Widget (only when open) */}
        {sidebarOpen && (
          <div className="mt-auto px-3">
            <div className="relative overflow-hidden rounded-xl border border-white/10 bg-gradient-to-br from-white/5 to-transparent p-4 backdrop-blur-sm">
              <div className="text-xs text-gray-500 font-medium mb-1">Local Time</div>
              <div className="text-2xl font-black text-white">
                {time.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit' })}
              </div>
              <div className="text-xs text-gray-500">
                {time.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="relative p-4 border-t border-white/10">
        <Button
          variant="ghost"
          size="sm"
          className={cn(
            'w-full gap-3 text-gray-400 hover:text-red-400 hover:bg-red-500/10 transition-all rounded-xl group',
            sidebarOpen ? 'justify-start' : 'justify-center'
          )}
        >
          <div className={cn(
            'rounded-lg bg-white/5 group-hover:bg-red-500/20 flex items-center justify-center transition-colors',
            sidebarOpen ? 'w-8 h-8' : 'w-10 h-10'
          )}>
            <LogOut className="h-4 w-4" />
          </div>
          {sidebarOpen && <span className="text-sm font-semibold">Sign Out</span>}
        </Button>
      </div>
    </div>
  );
}

// Add custom scrollbar styles
const style = document.createElement('style');
style.textContent = `
  .custom-scrollbar::-webkit-scrollbar {
    width: 6px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.1);
    border-radius: 3px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.2);
  }
`;
if (typeof document !== 'undefined') {
  document.head.appendChild(style);
}
