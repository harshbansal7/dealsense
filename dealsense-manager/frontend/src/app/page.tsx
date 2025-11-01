/**
 * Production-grade landing page with striking, unique design
 */

'use client';

import { useRouter } from 'next/navigation';
import { useState, useEffect } from 'react';
import {
  ArrowRight,
  Bot,
  Brain,
  CheckCircle,
  MessageSquare,
  Zap,
  Eye,
  TrendingUp,
  Sparkles,
  Play,
  ChevronRight,
  Star,
  Cpu,
  Shield,
  Layers,
  FileText,
  Image as ImageIcon,
  Search,
  Database,
  MessageCircle,
  FileSearch,
} from 'lucide-react';

import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';

export default function LandingPage() {
  const router = useRouter();
  const [hoveredFeature, setHoveredFeature] = useState<number | null>(null);
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 });

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      setMousePosition({ x: e.clientX, y: e.clientY });
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  const features = [
    {
      icon: <FileText className="w-7 h-7" />,
      title: 'Document Intelligence',
      description: 'Upload and process documents, PDFs, and files. AI extracts insights, key points, and actionable data from any document type.',
      color: 'from-blue-500 to-cyan-500',
      metric: 'Any format',
      details: ['GCP Document AI', 'Multi-format support', 'Instant extraction'],
      poweredBy: 'Google Cloud Document AI',
    },
    {
      icon: <Database className="w-7 h-7" />,
      title: 'RAG-Powered Chat',
      description: 'Chat directly with your documents and meeting transcriptions. Get instant answers grounded in your actual data with context-aware responses.',
      color: 'from-purple-500 to-pink-500',
      metric: 'Real-time',
      details: ['Vertex AI Search', 'Semantic search', 'Citation tracking'],
      poweredBy: 'Google Vertex AI Search',
    },
    {
      icon: <Search className="w-7 h-7" />,
      title: 'Google Search Integration',
      description: 'Get live data and insights on-the-go. Gemini with grounding pulls real-time information from Google Search to enrich answers with up-to-date external data.',
      color: 'from-green-500 to-emerald-500',
      metric: 'Live data',
      details: ['Gemini with Grounding', 'Fact verification', 'Data enrichment'],
      poweredBy: 'Gemini with Grounding',
    },
    {
      icon: <ImageIcon className="w-7 h-7" />,
      title: 'Vision AI',
      description: 'Process images, charts, and diagrams. Extract data from screenshots, analyze visual content, and answer questions about images in your documents.',
      color: 'from-orange-500 to-red-500',
      metric: 'Multi-modal',
      details: ['Gemini Vision', 'Chart extraction', 'Visual Q&A'],
      poweredBy: 'Gemini Vision API',
    },
  ];

  const stats = [
    { value: '99.8%', label: 'Accuracy Rate', icon: <Star className="w-4 h-4" /> },
    { value: '<2min', label: 'Setup Time', icon: <Zap className="w-4 h-4" /> },
    { value: '24/7', label: 'Availability', icon: <Cpu className="w-4 h-4" /> },
    { value: '100%', label: 'Secure', icon: <Shield className="w-4 h-4" /> },
  ];

  return (
    <div className="min-h-screen bg-black relative overflow-hidden">
      {/* Animated grid background */}
      <div className="fixed inset-0 pointer-events-none">
        <div className="absolute inset-0" style={{
          backgroundImage: `
            linear-gradient(to right, rgb(30 41 59 / 0.1) 1px, transparent 1px),
            linear-gradient(to bottom, rgb(30 41 59 / 0.1) 1px, transparent 1px)
          `,
          backgroundSize: '80px 80px',
        }} />
        {/* Dynamic light orbs following cursor */}
        <div 
          className="absolute w-[500px] h-[500px] bg-blue-500/20 rounded-full blur-3xl transition-all duration-700 ease-out"
          style={{
            left: `${mousePosition.x - 250}px`,
            top: `${mousePosition.y - 250}px`,
          }}
        />
        <div className="absolute top-1/4 -left-48 w-96 h-96 bg-purple-600/20 rounded-full blur-3xl animate-pulse" />
        <div className="absolute bottom-1/4 -right-48 w-96 h-96 bg-cyan-600/20 rounded-full blur-3xl animate-pulse" style={{ animationDelay: '1s' }} />
        <div className="absolute top-1/2 left-1/2 w-96 h-96 bg-pink-600/10 rounded-full blur-3xl animate-pulse" style={{ animationDelay: '2s' }} />
      </div>

      {/* Header */}
      <header className="relative border-b border-white/5 backdrop-blur-2xl bg-black/40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-6">
            <div className="flex items-center space-x-3 group cursor-pointer" onClick={() => router.push('/')}>
              <div className="relative">
                <div className="absolute inset-0 bg-gradient-to-r from-blue-500 to-purple-600 rounded-2xl blur-md opacity-75 group-hover:opacity-100 transition-opacity" />
                <div className="relative w-11 h-11 bg-gradient-to-br from-blue-500 via-purple-500 to-pink-500 rounded-2xl flex items-center justify-center transform group-hover:scale-110 transition-transform duration-300">
                  <Bot className="w-6 h-6 text-white" />
                </div>
              </div>
              <div>
                <h1 className="text-xl font-black bg-gradient-to-r from-white via-blue-100 to-purple-100 bg-clip-text text-transparent">
                  DealSense
                </h1>
                <p className="text-[10px] text-gray-500 -mt-1">AI Meeting Intelligence</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <Button
                onClick={() => router.push('/agents')}
                variant="ghost"
                className="text-gray-300 hover:text-white hover:bg-white/10 backdrop-blur-sm border border-transparent hover:border-white/10 transition-all"
              >
                Dashboard
              </Button>
              <Button
                onClick={() => router.push('/agents/create')}
                className="relative group bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 hover:from-blue-700 hover:via-purple-700 hover:to-pink-700 text-white shadow-xl shadow-blue-500/30 hover:shadow-2xl hover:shadow-purple-500/40 transition-all duration-300 transform hover:scale-105"
              >
                <span className="relative z-10 flex items-center">
                  Create Agent
                  <ArrowRight className="ml-2 w-4 h-4 group-hover:translate-x-1 transition-transform" />
                </span>
              </Button>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="relative">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-16 pb-20 lg:pt-24 lg:pb-32">
          <div className="grid lg:grid-cols-2 gap-16 items-center">
            {/* Left: Content */}
            <div className="space-y-8">
              {/* Badge */}
              <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-purple-500/30 bg-purple-500/10 backdrop-blur-sm group hover:border-purple-400/50 transition-all cursor-default">
                <Sparkles className="w-4 h-4 text-yellow-400 animate-pulse" />
                <span className="text-sm text-purple-200 font-medium">Powered by Google Cloud AI</span>
                <ChevronRight className="w-3 h-3 text-purple-400 group-hover:translate-x-1 transition-transform" />
              </div>

              {/* Headline */}
              <div className="space-y-4">
                <h1 className="text-5xl md:text-6xl lg:text-7xl font-black leading-[1.1] tracking-tight">
                  <span className="text-white">Turn Every</span>
                  <br />
                  <span className="bg-gradient-to-r from-blue-400 via-purple-400 to-pink-400 bg-clip-text text-transparent animate-gradient">
                    Meeting Into
                  </span>
                  <br />
                  <span className="text-white">Actionable</span>
                  <span className="text-transparent bg-gradient-to-r from-emerald-400 to-cyan-400 bg-clip-text"> Insights</span>
                </h1>
                <div className="h-1 w-24 bg-gradient-to-r from-blue-500 to-purple-500 rounded-full" />
              </div>

              {/* Subtitle */}
              <p className="text-xl text-gray-300 leading-relaxed max-w-xl">
                Deploy AI agents that <span className="text-white font-semibold">join your meetings</span>, capture insights in real-time, and deliver intelligent analysis. Purpose-built for <span className="text-purple-400 font-semibold">deal-makers</span> and <span className="text-blue-400 font-semibold">decision-makers</span>.
              </p>
              
              {/* Google Cloud Badge */}
              <div className="inline-flex items-center gap-2 px-4 py-2 rounded-lg border border-blue-500/20 bg-blue-500/5 backdrop-blur-sm">
                <Shield className="w-4 h-4 text-blue-400" />
                <span className="text-sm text-blue-200 font-medium">Enterprise-grade • Powered by Google Cloud Platform</span>
              </div>

              {/* CTA Buttons */}
              <div className="flex flex-col sm:flex-row items-start gap-4 pt-4">
                <Button
                  onClick={() => router.push('/agents/create')}
                  size="lg"
                  className="relative group bg-gradient-to-r from-blue-600 via-purple-600 to-pink-600 hover:from-blue-700 hover:via-purple-700 hover:to-pink-700 text-white px-10 py-7 text-lg font-bold shadow-2xl shadow-purple-500/30 hover:shadow-purple-500/50 transform hover:scale-105 transition-all duration-300 rounded-xl"
                >
                  <span className="relative z-10 flex items-center gap-2">
                    Get Started Free
                    <ArrowRight className="w-5 h-5 group-hover:translate-x-2 transition-transform" />
                  </span>
                  <div className="absolute inset-0 bg-gradient-to-r from-blue-400 to-purple-400 rounded-xl blur opacity-0 group-hover:opacity-50 transition-opacity" />
                </Button>
                <Button
                  onClick={() => router.push('/agents')}
                  variant="outline"
                  size="lg"
                  className="border-2 border-white/20 text-white hover:bg-white/10 hover:border-white/40 px-10 py-7 text-lg font-semibold backdrop-blur-sm rounded-xl transition-all group"
                >
                  <Play className="w-5 h-5 mr-2 group-hover:scale-110 transition-transform" />
                  Watch Demo
                </Button>
              </div>

              {/* Stats Row */}
              <div className="grid grid-cols-4 gap-4 pt-8">
                {stats.map((stat, index) => (
                  <div key={index} className="text-center">
                    <div className="flex items-center justify-center mb-2 text-purple-400">
                      {stat.icon}
                    </div>
                    <div className="text-2xl font-black text-white">{stat.value}</div>
                    <div className="text-xs text-gray-500 font-medium">{stat.label}</div>
                  </div>
                ))}
              </div>
            </div>

            {/* Right: Visual Element - Floating Documents */}
            <div className="relative hidden lg:block h-[600px]">
              {/* Background glow */}
              <div className="absolute inset-0 bg-gradient-to-br from-blue-500/10 via-purple-500/10 to-pink-500/10 rounded-3xl blur-3xl" />
              
              {/* Floating document cards */}
              <div className="relative h-full">
                {/* Main document card - center */}
                <Card className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-80 bg-gradient-to-br from-gray-900/95 to-black/95 border-2 border-purple-500/30 backdrop-blur-2xl shadow-2xl z-10 animate-float">
                  <CardContent className="p-6">
                    <div className="space-y-4">
                      <div className="flex items-center gap-3 mb-4">
                        <div className="w-12 h-12 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl flex items-center justify-center">
                          <FileText className="w-6 h-6 text-white" />
                        </div>
                        <div>
                          <div className="text-sm font-bold text-white">Q4_Report.pdf</div>
                          <div className="text-xs text-gray-400">Processing...</div>
                        </div>
                      </div>
                      
                      <div className="space-y-2">
                        <div className="flex items-center justify-between text-xs">
                          <span className="text-gray-400">Extraction</span>
                          <span className="text-purple-400 font-bold">87%</span>
                        </div>
                        <div className="h-1.5 bg-gray-800 rounded-full overflow-hidden">
                          <div className="h-full w-[87%] bg-gradient-to-r from-purple-500 to-pink-500 rounded-full animate-pulse" />
                        </div>
                        
                        <div className="flex items-center gap-2 mt-3 p-2 bg-white/5 rounded-lg">
                          <Search className="w-3 h-3 text-green-400" />
                          <span className="text-xs text-gray-300">Searching Google...</span>
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>

                {/* Floating card - top right */}
                <Card className="absolute top-8 right-4 w-56 bg-gradient-to-br from-gray-900/90 to-black/90 border border-blue-500/30 backdrop-blur-xl shadow-xl animate-float" style={{ animationDelay: '0.5s' }}>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-lg flex items-center justify-center">
                        <MessageCircle className="w-4 h-4 text-white" />
                      </div>
                      <div className="text-xs font-semibold text-white">RAG Chat</div>
                    </div>
                    <div className="text-xs text-gray-400 leading-relaxed">
                      "What were the key decisions in the Q4 meeting?"
                    </div>
                    <div className="mt-2 flex items-center gap-1">
                      <div className="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse" />
                      <span className="text-[10px] text-green-400">Analyzing transcript...</span>
                    </div>
                  </CardContent>
                </Card>

                {/* Floating card - bottom left */}
                <Card className="absolute bottom-16 left-8 w-48 bg-gradient-to-br from-gray-900/90 to-black/90 border border-green-500/30 backdrop-blur-xl shadow-xl animate-float" style={{ animationDelay: '1s' }}>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <div className="w-8 h-8 bg-gradient-to-br from-green-500 to-emerald-600 rounded-lg flex items-center justify-center">
                        <ImageIcon className="w-4 h-4 text-white" />
                      </div>
                      <div className="text-xs font-semibold text-white">Vision AI</div>
                    </div>
                    <div className="w-full h-16 bg-gradient-to-br from-gray-800 to-gray-900 rounded-lg mb-2 flex items-center justify-center">
                      <ImageIcon className="w-6 h-6 text-gray-600" />
                    </div>
                    <div className="text-[10px] text-gray-400">Extracting chart data...</div>
                  </CardContent>
                </Card>

                {/* Floating card - top left */}
                <Card className="absolute top-24 left-4 w-52 bg-gradient-to-br from-gray-900/90 to-black/90 border border-orange-500/30 backdrop-blur-xl shadow-xl animate-float" style={{ animationDelay: '1.5s' }}>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <div className="w-8 h-8 bg-gradient-to-br from-orange-500 to-red-600 rounded-lg flex items-center justify-center">
                        <FileSearch className="w-4 h-4 text-white" />
                      </div>
                      <div className="text-xs font-semibold text-white">Smart Search</div>
                    </div>
                    <div className="space-y-1.5">
                      <div className="flex items-center gap-2 text-[10px]">
                        <CheckCircle className="w-3 h-3 text-green-400" />
                        <span className="text-gray-300">Document indexed</span>
                      </div>
                      <div className="flex items-center gap-2 text-[10px]">
                        <CheckCircle className="w-3 h-3 text-green-400" />
                        <span className="text-gray-300">Images processed</span>
                      </div>
                      <div className="flex items-center gap-2 text-[10px]">
                        <div className="w-3 h-3 border-2 border-orange-400 border-t-transparent rounded-full animate-spin" />
                        <span className="text-gray-300">Enriching with search...</span>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features - Bento Grid */}
      <section className="relative py-32">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="text-center mb-20">
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full border border-cyan-500/30 bg-cyan-500/10 backdrop-blur-sm mb-6">
              <FileText className="w-4 h-4 text-cyan-400" />
              <span className="text-sm text-cyan-200 font-medium">Document Intelligence Platform</span>
            </div>
            <h2 className="text-4xl md:text-5xl font-black text-white mb-6">
              Your Documents.
              <br />
              <span className="text-transparent bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text">Infinitely Smarter.</span>
            </h2>
            <p className="text-gray-400 text-xl max-w-3xl mx-auto">
              Process documents, extract insights from images, chat with your data, and enrich answers with real-time Google Search
            </p>
          </div>

          {/* Features list - clean, non-boxy design */}
          <div className="space-y-8 max-w-5xl mx-auto">
            {features.map((feature, index) => (
              <div
                key={index}
                onMouseEnter={() => setHoveredFeature(index)}
                onMouseLeave={() => setHoveredFeature(null)}
                className="group relative"
              >
                {/* Subtle background glow on hover */}
                <div className={`absolute -inset-4 bg-gradient-to-r ${feature.color} opacity-0 group-hover:opacity-5 rounded-3xl blur-2xl transition-opacity duration-700`} />
                
                <div className="relative flex gap-8 items-start p-8 rounded-2xl border border-white/5 hover:border-white/10 bg-gradient-to-br from-white/[0.02] to-transparent backdrop-blur-sm transition-all duration-500 group-hover:transform group-hover:translate-x-2">
                  {/* Icon */}
                  <div className="relative flex-shrink-0">
                    <div className={`absolute inset-0 bg-gradient-to-br ${feature.color} rounded-2xl blur-xl opacity-0 group-hover:opacity-60 transition-opacity duration-500`} />
                    <div className={`relative w-16 h-16 rounded-2xl bg-gradient-to-br ${feature.color} flex items-center justify-center shadow-xl transform group-hover:scale-110 group-hover:rotate-6 transition-all duration-500`}>
                      <div className="text-white">{feature.icon}</div>
                    </div>
                  </div>

                  {/* Content */}
                  <div className="flex-1 space-y-4">
                    <div className="flex items-start justify-between gap-4">
                      <h3 className="text-2xl font-black text-white group-hover:text-transparent group-hover:bg-gradient-to-r group-hover:from-white group-hover:to-gray-300 group-hover:bg-clip-text transition-all duration-300">
                        {feature.title}
                      </h3>
                      <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-white/5 border border-white/10 group-hover:border-white/20 transition-colors">
                        <div className="w-1.5 h-1.5 bg-green-400 rounded-full animate-pulse" />
                        <span className="text-xs text-gray-400 font-bold">{feature.metric}</span>
                      </div>
                    </div>
                    
                    <p className="text-gray-400 text-lg leading-relaxed group-hover:text-gray-300 transition-colors">
                      {feature.description}
                    </p>

                    {/* Feature details */}
                    <div className="flex flex-wrap gap-3 pt-2">
                      {feature.details.map((detail, i) => (
                        <div key={i} className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white/5 border border-white/10">
                          <CheckCircle className="w-3.5 h-3.5 text-green-400" />
                          <span className="text-xs text-gray-300 font-medium">{detail}</span>
                        </div>
                      ))}
                    </div>

                    {/* Powered by badge */}
                    <div className="flex items-center gap-2 pt-3">
                      <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-gradient-to-r from-blue-500/10 to-purple-500/10 border border-blue-500/20">
                        <Sparkles className="w-3 h-3 text-blue-400" />
                        <span className="text-[10px] text-gray-400 font-semibold">POWERED BY</span>
                        <span className="text-xs text-white font-bold">{feature.poweredBy}</span>
                      </div>
                    </div>

                    {/* Learn more */}
                    <div className="flex items-center gap-2 text-sm text-gray-500 group-hover:text-purple-400 transition-colors pt-2">
                      <span className="font-semibold">Explore feature</span>
                      <ChevronRight className="w-4 h-4 group-hover:translate-x-2 transition-transform" />
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="relative py-32">
        <div className="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8">
          <Card className="relative bg-gradient-to-br from-blue-600/10 via-purple-600/10 to-pink-600/10 border-2 border-white/10 backdrop-blur-2xl overflow-hidden group">
            {/* Animated border gradient */}
            <div className="absolute inset-0 bg-gradient-to-r from-blue-500 via-purple-500 to-pink-500 opacity-0 group-hover:opacity-20 blur-xl transition-opacity duration-700" />
            
            {/* Grid pattern overlay */}
            <div className="absolute inset-0 opacity-10" style={{
              backgroundImage: `
                linear-gradient(to right, white 1px, transparent 1px),
                linear-gradient(to bottom, white 1px, transparent 1px)
              `,
              backgroundSize: '40px 40px',
            }} />

            <CardContent className="relative p-16 text-center">
              {/* Icon array */}
              <div className="flex items-center justify-center gap-4 mb-8">
                {[Bot, Brain, Zap, TrendingUp].map((Icon, i) => (
                  <div
                    key={i}
                    className="w-12 h-12 rounded-xl bg-gradient-to-br from-white/10 to-white/5 border border-white/20 flex items-center justify-center backdrop-blur-sm transform hover:scale-110 transition-transform"
                    style={{ animationDelay: `${i * 0.1}s` }}
                  >
                    <Icon className="w-6 h-6 text-white" />
                  </div>
                ))}
              </div>

              <h2 className="text-4xl md:text-5xl font-black text-white mb-6 leading-tight">
                Start Your First Meeting
                <br />
                <span className="text-transparent bg-gradient-to-r from-blue-400 via-purple-400 to-pink-400 bg-clip-text">
                  In Under 2 Minutes
                </span>
              </h2>
              <p className="text-gray-300 text-xl mb-10 max-w-2xl mx-auto leading-relaxed">
                Join thousands of teams using DealSense to capture insights, track commitments, and accelerate decision-making.
              </p>
              
              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <Button
                  onClick={() => router.push('/agents/create')}
                  size="lg"
                  className="relative bg-white text-black hover:bg-gray-100 px-10 py-7 text-lg font-bold shadow-2xl shadow-white/10 hover:shadow-white/20 rounded-xl transform hover:scale-105 transition-all group/btn"
                >
                  <span className="flex items-center gap-2">
                    Create Your First Agent
                    <ArrowRight className="w-5 h-5 group-hover/btn:translate-x-1 transition-transform" />
                  </span>
                </Button>
                <div className="text-sm text-gray-500 flex items-center gap-2">
                  <CheckCircle className="w-4 h-4 text-green-400" />
                  <span>Free forever • No credit card required</span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Footer */}
      <footer className="relative border-t border-white/5 backdrop-blur-xl bg-black/40 mt-32">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="flex flex-col gap-8">
            {/* Top row */}
            <div className="flex flex-col md:flex-row justify-between items-center gap-6">
              <div className="flex items-center space-x-3">
                <div className="relative">
                  <div className="absolute inset-0 bg-gradient-to-r from-blue-500 to-purple-600 rounded-xl blur-md opacity-50" />
                  <div className="relative w-10 h-10 bg-gradient-to-br from-blue-500 to-purple-600 rounded-xl flex items-center justify-center">
                    <Bot className="w-5 h-5 text-white" />
                  </div>
                </div>
                <div>
                  <div className="text-sm font-bold text-white">DealSense</div>
                  <span className="text-xs text-gray-500">© 2025 All rights reserved.</span>
                </div>
              </div>
              <div className="flex items-center gap-8 text-sm">
                <button className="text-gray-400 hover:text-white transition-colors font-medium">Privacy</button>
                <button className="text-gray-400 hover:text-white transition-colors font-medium">Terms</button>
                <button className="text-gray-400 hover:text-white transition-colors font-medium">Contact</button>
                <button className="text-gray-400 hover:text-white transition-colors font-medium">Documentation</button>
              </div>
            </div>
            
            {/* Google Cloud branding section */}
            <div className="flex flex-col items-center gap-4 pt-6 border-t border-white/5">
              <div className="flex items-center gap-2">
                <Sparkles className="w-4 h-4 text-blue-400" />
                <span className="text-sm text-gray-400 font-medium">Built with enterprise-grade Google Cloud technologies</span>
              </div>
              <div className="flex flex-wrap items-center justify-center gap-4 text-xs text-gray-500">
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10">
                  <Brain className="w-3 h-3 text-purple-400" />
                  <span>Gemini AI</span>
                </div>
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10">
                  <Database className="w-3 h-3 text-blue-400" />
                  <span>Vertex AI Search</span>
                </div>
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10">
                  <FileText className="w-3 h-3 text-cyan-400" />
                  <span>Document AI</span>
                </div>
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10">
                  <Shield className="w-3 h-3 text-green-400" />
                  <span>Google Cloud Platform</span>
                </div>
                <div className="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-white/5 border border-white/10">
                  <Zap className="w-3 h-3 text-yellow-400" />
                  <span>Vertex AI Embeddings</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}