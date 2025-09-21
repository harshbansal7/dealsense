# DealSense

A production-grade web dashboard for managing multiple joinly.ai agents in video meetings.

## Overview

This dashboard provides a user-friendly interface to:
- Spawn multiple joinly clients to different meetings
- Monitor active agents and their status
- Manage agent configurations
- View real-time transcripts and logs
- Control meeting interactions

## Architecture

# DealSense

**DealSense** is a comprehensive web-based management interface for deploying and managing AI agents that participate in online meetings. This full-stack application provides an intuitive way to create, monitor, and control AI meeting assistants that can join video conferences across multiple platforms including Google Meet, Zoom, Microsoft Teams, and more.

## 🎯 Overview

DealSense bridges the gap between AI meeting automation and user-friendly management. It consists of a robust Go backend that manages AI agent lifecycles and a modern Next.js frontend that provides an elegant interface for controlling these agents. The system supports two primary agent modes:

- **Conversational Mode**: AI agents that actively participate in meetings with natural, contextual responses
- **Analyst Mode**: Silent observers that provide comprehensive analysis, meeting summaries, and actionable insights

## 🏗️ Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                    DealSense                            │
├─────────────────────┬───────────────────────────────────────────┤
│     Frontend        │             Backend                       │
│   (Next.js 15)      │           (Go 1.24)                      │
│                     │                                          │
│ ┌─────────────────┐ │ ┌─────────────────┐ ┌─────────────────┐ │
│ │   Web Interface │ │ │  Agent Manager  │ │  MCP Client     │ │
│ │   - Dashboard   │ │ │  - Lifecycle    │ │  - Communication│ │
│ │   - Agent CRUD  │ │ │  - Monitoring   │ │  - with Joinly  │ │
│ │   - Real-time   │ │ │  - WebSocket    │ │  - API Calls    │ │
│ │     Updates     │ │ │    Hub          │ │                 │ │
│ └─────────────────┘ │ └─────────────────┘ └─────────────────┘ │
│                     │                                          │
│ ┌─────────────────┐ │ ┌─────────────────────────────────────┐ │
│ │  State Mgmt     │ │ │           HTTP API                  │ │
│ │  - Zustand      │ │ │  - RESTful Endpoints               │ │
│ │  - React Query  │ │ │  - WebSocket Connections           │ │
│ │  - Real-time    │ │ │  - Discord Webhooks                │ │
│ └─────────────────┘ │ └─────────────────────────────────────┘ │
└─────────────────────┴───────────────────────────────────────────┘
                              │
                              ▼
                   ┌─────────────────────┐
                   │   Joinly Core       │
                   │   AI Agent System   │
                   │   (External)        │
                   └─────────────────────┘
```

### Backend Architecture (Go)

The backend is built using Go 1.24 with a modular architecture:

#### Core Components

1. **Agent Manager** (`internal/manager/manager.go`)
   - Central orchestrator for all AI agents
   - Manages agent lifecycle (create, start, stop, delete)
   - Maintains WebSocket connections for real-time updates
   - Handles concurrent agent operations with goroutines
   - Tracks conversation history and meeting analytics

2. **API Layer** (`internal/api/`)
   - RESTful HTTP endpoints using Gin framework
   - WebSocket handlers for real-time communication
   - CORS configuration for frontend integration
   - Comprehensive error handling and validation

3. **Model System** (`internal/models/models.go`)
   - Type-safe data structures for agents, meetings, and configurations
   - Support for multiple AI providers (OpenAI, Anthropic, Google, Ollama)
   - Voice synthesis options (Kokoro, ElevenLabs, Deepgram)
   - Speech recognition providers (Whisper, Deepgram)

4. **Client Interface** (`internal/client/`)
   - MCP (Model Context Protocol) integration
   - Communication with Joinly core services
   - Analyst agent implementation for meeting analysis
   - Connection management and health monitoring

5. **Configuration Management** (`internal/config/config.go`)
   - Environment-based configuration loading
   - Discord webhook integration for logging
   - Comprehensive logging system with multiple levels
   - Server and CORS configuration

#### Key Features

- **Concurrent Agent Management**: Supports up to 10 simultaneous agents (configurable)
- **Real-time Monitoring**: WebSocket-based live updates for agent status and logs
- **Discord Integration**: Advanced logging with Discord webhooks for different log levels
- **Health Monitoring**: Built-in health checks and graceful shutdown
- **Docker Support**: Production-ready containerization

### Frontend Architecture (Next.js)

The frontend is a modern React application built with Next.js 15 and TypeScript:

#### Technology Stack

- **Framework**: Next.js 15 with App Router
- **Language**: TypeScript for type safety
- **UI Library**: Radix UI components with custom theming
- **Styling**: Tailwind CSS for utility-first styling
- **State Management**: Zustand for global state
- **Data Fetching**: TanStack React Query for server state
- **Real-time**: Socket.IO client for WebSocket connections
- **Forms**: React Hook Form with Zod validation

#### Key Pages and Components

1. **Landing Page** (`src/app/page.tsx`)
   - Modern, responsive landing page
   - Feature showcase and benefits
   - Call-to-action for agent creation

2. **Agent Management** (`src/app/agents/page.tsx`)
   - Comprehensive agent dashboard
   - Real-time status updates
   - Agent creation, monitoring, and control
   - Visual status indicators and metrics

3. **Agent Details** (`src/app/agents/[id]/page.tsx`)
   - Detailed agent view with logs
   - Real-time transcript display
   - Meeting analysis (for analyst mode agents)
   - Performance metrics and statistics

4. **Agent Creation** (`src/app/agents/create/page.tsx`)
   - Step-by-step agent configuration
   - AI provider selection and configuration
   - Voice and language settings
   - Advanced transcription parameters

#### State Management

The application uses a sophisticated state management system:

```typescript
// Agent Store (Zustand)
interface AgentStore {
  agents: Agent[];
  isLoading: boolean;
  setAgents: (agents: Agent[]) => void;
  updateAgent: (agent: Agent) => void;
  removeAgent: (agentId: string) => void;
  connectSessionWebSocket: (onMessage: Function) => void;
  // ... more methods
}

// UI Store for notifications and global UI state
interface UIStore {
  notifications: Notification[];
  addNotification: (notification: Notification) => void;
  removeNotification: (id: string) => void;
  // ... more methods
}
```

## 🚀 Features

### Agent Management

#### Agent Types

1. **Conversational Agents**
   - Actively participate in meetings
   - Natural language responses
   - Context-aware interactions
   - Name-trigger activation support
   - Custom personality prompts

2. **Analyst Agents**
   - Silent meeting observers
   - Real-time transcription
   - Comprehensive meeting analysis
   - Automatic summary generation
   - Key points extraction
   - Action items identification

#### Configuration Options

- **AI Providers**: OpenAI, Anthropic (Claude), Google (Gemini), Ollama
- **Voice Synthesis**: Kokoro, ElevenLabs, Deepgram TTS
- **Speech Recognition**: OpenAI Whisper, Deepgram STT
- **Languages**: Multi-language support
- **Meeting Platforms**: Google Meet, Zoom, Microsoft Teams, and more

#### Advanced Settings

- **Utterance Tail Seconds**: Controls speech detection sensitivity
- **No Speech Event Delay**: Manages silence detection
- **Max STT Tasks**: Limits concurrent speech recognition tasks
- **Window Queue Size**: Controls audio processing buffer

### Real-time Monitoring

#### WebSocket Integration

The system provides real-time updates through WebSocket connections:

```javascript
// Session-wide updates
connectSessionWebSocket((message) => {
  switch (message.type) {
    case 'status':
      // Agent status changes
      break;
    case 'log':
      // Real-time log entries
      break;
    case 'error':
      // Error notifications
      break;
    case 'transcript':
      // Live meeting transcription
      break;
  }
});
```

#### Status Tracking

- **Agent Lifecycle**: Created → Starting → Running → Stopping → Stopped
- **Error Handling**: Detailed error messages and recovery suggestions
- **Performance Metrics**: Uptime tracking and resource usage
- **Meeting Analytics**: Participant tracking and conversation analysis

### Meeting Analysis

For analyst mode agents, the system provides comprehensive meeting insights:

#### Analysis Components

1. **Transcript Processing**
   ```json
   {
     "timestamp": "2025-09-21T01:15:39.542847+05:30",
     "speaker": "John Doe",
     "text": "Let's discuss the quarterly results",
     "is_agent": false
   }
   ```

2. **Intelligent Summarization**
   - Meeting overview with key discussion points
   - Grounded summaries with source attribution
   - Topic segmentation with time codes
   - Participant contribution analysis

3. **Key Points Extraction**
   - Automated bullet-point summaries
   - Action item identification
   - Decision tracking
   - Follow-up task generation

4. **Metrics and Insights**
   - Meeting duration and word count
   - Sentiment analysis
   - Keyword extraction
   - Participant engagement metrics

## 🛠️ Installation and Setup

### Prerequisites

- **Go 1.24+** for backend development
- **Node.js 20+** for frontend development
- **Docker & Docker Compose** for containerized deployment
- **Git** for version control

### Local Development Setup

#### Backend Setup

1. **Clone and navigate to backend directory**:
   ```bash
   cd joinly-manager-ui/backend_v2
   ```

2. **Install Go dependencies**:
   ```bash
   go mod download
   ```

3. **Configure environment variables**:
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Run the backend server**:
   ```bash
   go run cmd/server/main.go
   ```

The backend will start on `http://localhost:8001`.

#### Frontend Setup

1. **Navigate to frontend directory**:
   ```bash
   cd joinly-manager-ui/frontend
   ```

2. **Install Node.js dependencies**:
   ```bash
   npm install
   ```

3. **Configure environment variables**:
   ```bash
   # Create .env.local file with your configuration
   echo "NEXT_PUBLIC_API_URL=http://localhost:8001" > .env.local
   ```

4. **Run the development server**:
   ```bash
   npm run dev
   ```

The frontend will start on `http://localhost:3000`.

### Docker Deployment

#### Complete Stack with Docker Compose

The easiest way to deploy the entire Joinly ecosystem (Core + Manager UI) is using Docker Compose:

1. **Setup environment configuration**:
   ```bash
   # Copy the comprehensive environment template
   cp .env.example .env
   
   # Edit with your API keys and configuration
   nano .env
   ```

2. **Configure required API keys** in `.env`:
   ```bash
   # At minimum, one LLM provider
   OPENAI_API_KEY=sk-your-openai-key
   # AND one voice service  
   ELEVENLABS_API_KEY=your-elevenlabs-key
   ```

3. **Deploy the complete stack**:
   ```bash
   # Start all services (Joinly Core + Manager Backend + Frontend)
   docker-compose up -d
   
   # View logs
   docker-compose logs -f
   
   # Check status
   docker-compose ps
   ```

4. **Access the services**:
   - **Frontend Dashboard**: http://localhost:3000
   - **Manager API**: http://localhost:8001  
   - **Joinly Core**: http://localhost:8000

The Docker Compose setup includes:
- **Joinly Core** (`ghcr.io/joinly-ai/joinly:latest`) on port 8000
- **Manager Backend** (Go) on port 8001
- **Manager Frontend** (Next.js) on port 3000
- Health checks and service dependencies
- Persistent volumes for data and logs
- Isolated network for internal communication

#### Individual Container Deployment

For more control, you can deploy containers individually:

**Joinly Core**:
```bash
docker run --env-file .env -p 8000:8000 ghcr.io/joinly-ai/joinly:latest
```

**Backend Container**:
```bash
cd backend_v2
docker build -t joinly-manager-backend .
docker run --env-file ../.env -p 8001:8001 \
  -e SERVER_HOST=0.0.0.0 \
  -e JOINLY_URL=http://host.docker.internal:8000/mcp/ \
  joinly-manager-backend
```

**Frontend Container**:
```bash
cd frontend
docker build -t joinly-manager-frontend .
docker run --env-file ../.env -p 3000:3000 \
  -e NEXT_PUBLIC_API_URL=http://localhost:8001 \
  joinly-manager-frontend
```

For detailed deployment instructions, see [DOCKER_DEPLOYMENT.md](./DOCKER_DEPLOYMENT.md).

### Production Configuration

#### Environment Variables

**Backend Configuration (.env)**:
```bash
# Server Configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8001

# Joinly Integration
JOINLY_URL=http://your-joinly-instance:8000/mcp/
MAX_AGENTS=10

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json

# Discord Webhook Integration (Optional)
DISCORD_LOGGING_ENABLED=true
DISCORD_INFO_WEBHOOK=https://discord.com/api/webhooks/...
DISCORD_ERROR_WEBHOOK=https://discord.com/api/webhooks/...
DISCORD_BOT_USERNAME=Joinly Manager

# Security
CORS_ALLOWED_ORIGINS=https://your-frontend-domain.com
```

**Frontend Configuration (.env.local)**:
```bash
# API Configuration
NEXT_PUBLIC_API_URL=https://your-backend-domain.com

# Application Configuration
NEXT_PUBLIC_APP_NAME=Joinly Manager
NEXT_PUBLIC_MAX_AGENTS=10
```

## 📚 API Documentation

### Base URL
```
http://localhost:8001
```

### Authentication
Currently, the API does not require authentication. In production, implement appropriate security measures.

### Endpoints

#### Agent Management

##### List All Agents
```http
GET /agents
```

**Response**:
```json
[
  {
    "id": "agent_123456",
    "config": {
      "name": "Sales Meeting Assistant",
      "meeting_url": "https://meet.google.com/abc-def-ghi",
      "llm_provider": "openai",
      "llm_model": "gpt-4",
      "tts_provider": "elevenlabs",
      "stt_provider": "whisper",
      "language": "en",
      "conversation_mode": "conversational",
      "auto_join": true,
      "name_trigger": false
    },
    "status": "running",
    "created_at": "2024-01-15T10:30:00Z",
    "started_at": "2024-01-15T10:31:00Z",
    "logs": []
  }
]
```

##### Create Agent
```http
POST /agents
Content-Type: application/json
```

**Request Body**:
```json
{
  "name": "Marketing Meeting Bot",
  "meeting_url": "https://meet.google.com/xyz-abc-123",
  "llm_provider": "google",
  "llm_model": "gemini-pro",
  "tts_provider": "kokoro",
  "stt_provider": "deepgram",
  "language": "en",
  "conversation_mode": "analyst",
  "auto_join": false,
  "name_trigger": true,
  "custom_prompt": "You are a helpful marketing meeting assistant...",
  "env_vars": {
    "OPENAI_API_KEY": "sk-...",
    "DEEPGRAM_API_KEY": "..."
  }
}
```

**Response**:
```json
{
  "id": "agent_789012",
  "config": { ... },
  "status": "created",
  "created_at": "2024-01-15T10:35:00Z",
  "logs": []
}
```

##### Get Agent Details
```http
GET /agents/{agent_id}
```

##### Start Agent
```http
POST /agents/{agent_id}/start
```

##### Stop Agent
```http
POST /agents/{agent_id}/stop
```

##### Delete Agent
```http
DELETE /agents/{agent_id}
```

##### Join Meeting
```http
POST /agents/{agent_id}/join-meeting
```

#### Monitoring and Logs

##### Get Agent Logs
```http
GET /agents/{agent_id}/logs?lines=100
```

**Response**:
```json
{
  "logs": [
    {
      "timestamp": "2024-01-15T10:35:00Z",
      "level": "info",
      "message": "Agent started successfully"
    }
  ]
}
```

##### Get Agent Analysis (Analyst Mode Only)
```http
GET /agents/{agent_id}/analysis
```

**Response**:
```json
{
  "meeting_id": "agent_123456",
  "meeting_url": "https://meet.google.com/abc-def-ghi",
  "start_time": "2024-01-15T10:30:00Z",
  "last_updated": "2024-01-15T11:00:00Z",
  "transcript": [
    {
      "timestamp": "2024-01-15T10:31:00Z",
      "speaker": "John Doe",
      "text": "Let's start the meeting",
      "is_agent": false
    }
  ],
  "summary": "Meeting discussion about quarterly sales targets...",
  "key_points": [
    "Q1 target set at $500K",
    "New product launch scheduled for March"
  ],
  "action_items": [
    "John to prepare sales forecast by Friday",
    "Marketing team to finalize campaign materials"
  ],
  "participants": ["John Doe", "Jane Smith"],
  "duration_minutes": 30,
  "sentiment": "positive"
}
```

##### Get Formatted Analysis
```http
GET /agents/{agent_id}/analysis/formatted
```

Returns a formatted text version of the analysis.

#### System Information

##### Health Check
```http
GET /
```

##### List Meetings
```http
GET /meetings
```

##### Usage Statistics
```http
GET /usage
```

##### WebSocket Statistics
```http
GET /ws/stats
```

### WebSocket Connections

#### Agent-Specific WebSocket
```
ws://localhost:8001/ws/agents/{agent_id}
```

#### Session-Wide WebSocket
```
ws://localhost:8001/ws/session
```

**Message Types**:
- `status`: Agent status changes
- `log`: Real-time log entries
- `error`: Error notifications
- `transcript`: Live meeting transcription
- `analysis`: Meeting analysis updates

## 🧪 Usage Examples

### Creating a Conversational Agent

```javascript
const agent = await fetch('/agents', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: "Customer Support Bot",
    meeting_url: "https://meet.google.com/customer-support",
    llm_provider: "openai",
    llm_model: "gpt-4",
    tts_provider: "elevenlabs",
    stt_provider: "whisper",
    language: "en",
    conversation_mode: "conversational",
    name_trigger: true,
    custom_prompt: "You are a helpful customer support representative...",
    auto_join: true
  })
});
```

### Setting Up an Analyst Agent

```javascript
const analystAgent = await fetch('/agents', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    name: "Meeting Analyst",
    meeting_url: "https://zoom.us/j/123456789",
    llm_provider: "google",
    llm_model: "gemini-pro",
    tts_provider: "kokoro",
    stt_provider: "deepgram",
    language: "en",
    conversation_mode: "analyst",
    auto_join: true,
    utterance_tail_seconds: 2.0,
    no_speech_event_delay: 1.5
  })
});
```

### Real-time Monitoring

```javascript
const ws = new WebSocket('ws://localhost:8001/ws/session');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch (message.type) {
    case 'status':
      console.log(`Agent ${message.agent_id} status: ${message.data.status}`);
      break;
    case 'transcript':
      console.log(`New transcript: ${message.data.text}`);
      break;
    case 'log':
      console.log(`Log: ${message.data.message}`);
      break;
  }
};
```

## 🔧 Configuration Guide

### AI Provider Configuration

#### OpenAI
```bash
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_ORG_ID=org-your-org-id  # Optional
```

#### Anthropic (Claude)
```bash
ANTHROPIC_API_KEY=sk-ant-your-api-key-here
```

#### Google (Gemini)
```bash
GOOGLE_API_KEY=your-google-api-key
GOOGLE_PROJECT_ID=your-project-id  # Optional
```

#### Ollama (Local)
```bash
OLLAMA_HOST=http://localhost:11434
```

### Voice Service Configuration

#### ElevenLabs TTS
```bash
ELEVENLABS_API_KEY=your-elevenlabs-api-key
ELEVENLABS_VOICE_ID=your-preferred-voice-id
```

#### Deepgram (STT/TTS)
```bash
DEEPGRAM_API_KEY=your-deepgram-api-key
```

### Discord Webhook Integration

```bash
DISCORD_LOGGING_ENABLED=true
DISCORD_INFO_WEBHOOK=https://discord.com/api/webhooks/your-info-webhook
DISCORD_ERROR_WEBHOOK=https://discord.com/api/webhooks/your-error-webhook
DISCORD_DEBUG_WEBHOOK=https://discord.com/api/webhooks/your-debug-webhook
DISCORD_BOT_USERNAME=Joinly Manager Bot
```

### Advanced Configuration

#### Transcription Parameters
- **utterance_tail_seconds**: Time to wait after speech detection (default: 1.0)
- **no_speech_event_delay**: Delay before processing silence (default: 0.5)
- **max_stt_tasks**: Maximum concurrent STT operations (default: 3)
- **window_queue_size**: Audio processing buffer size (default: 10)

#### Performance Tuning
- **MAX_AGENTS**: Maximum concurrent agents (default: 10)
- **LOG_LEVEL**: Logging verbosity (debug, info, warn, error)
- **Server timeouts**: ReadTimeout and WriteTimeout for HTTP server

## 🔍 Troubleshooting

### Common Issues

#### Agent Fails to Start
**Symptoms**: Agent stuck in "starting" status
**Solutions**:
1. Check if Joinly core service is accessible
2. Verify environment variables for AI providers
3. Ensure meeting URL is valid and accessible
4. Check agent logs for specific error messages

#### No Audio/Video in Meeting
**Symptoms**: Agent joins but doesn't respond to speech
**Solutions**:
1. Verify STT provider API keys
2. Check microphone permissions in browser
3. Ensure meeting platform allows bots
4. Review audio processing parameters

#### WebSocket Connection Issues
**Symptoms**: No real-time updates in frontend
**Solutions**:
1. Check firewall settings for WebSocket connections
2. Verify backend WebSocket endpoint is accessible
3. Check browser console for connection errors
4. Ensure CORS configuration allows WebSocket upgrades

#### High Memory Usage
**Symptoms**: Backend consuming excessive memory
**Solutions**:
1. Reduce MAX_AGENTS limit
2. Adjust transcription buffer sizes
3. Implement log rotation
4. Monitor goroutine leaks

### Debug Mode

Enable debug logging for detailed troubleshooting:

```bash
LOG_LEVEL=debug
```

Debug logs will show:
- Detailed API request/response information
- WebSocket message flow
- Agent lifecycle events
- Memory and performance metrics

### Health Checks

Monitor system health through built-in endpoints:

```bash
# Backend health
curl http://localhost:8001/

# WebSocket statistics
curl http://localhost:8001/ws/stats

# System usage
curl http://localhost:8001/usage
```

## 🚀 Production Deployment

### Infrastructure Requirements

#### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 4GB
- **Storage**: 20GB SSD
- **Network**: 100 Mbps bandwidth

#### Recommended for High Load
- **CPU**: 8 cores
- **RAM**: 16GB
- **Storage**: 100GB SSD
- **Network**: 1 Gbps bandwidth
- **Load Balancer**: For multiple backend instances

### Security Considerations

#### Network Security
- Use HTTPS/WSS in production
- Implement proper CORS policies
- Set up firewall rules for required ports only
- Use VPN for internal communication

#### API Security
- Implement authentication and authorization
- Rate limiting for API endpoints
- Input validation and sanitization
- Regular security audits

#### Environment Variables
- Use secure secret management (HashiCorp Vault, AWS Secrets Manager)
- Rotate API keys regularly
- Limit access to environment files
- Use encrypted storage for sensitive data

### Monitoring and Alerting

#### Metrics to Monitor
- Agent creation/deletion rates
- WebSocket connection counts
- Memory and CPU usage
- API response times
- Error rates by endpoint

#### Recommended Tools
- **Prometheus** + **Grafana** for metrics visualization
- **Discord webhooks** for real-time alerts
- **Health check** endpoints for uptime monitoring
- **Log aggregation** (ELK stack or similar)

### Scaling Strategies

#### Horizontal Scaling
- Deploy multiple backend instances behind a load balancer
- Use Redis for session sharing across instances
- Implement sticky sessions for WebSocket connections

#### Vertical Scaling
- Increase server resources based on concurrent agent load
- Monitor memory usage per agent
- Optimize Go runtime parameters (GOMAXPROCS, GC settings)

## 🤝 Contributing

### Development Guidelines

#### Code Style
- **Go**: Follow standard Go formatting (gofmt, golint)
- **TypeScript**: Use Prettier and ESLint configurations
- **Commits**: Use conventional commit format
- **Documentation**: Update README for significant changes

#### Testing
- Unit tests for critical business logic
- Integration tests for API endpoints
- End-to-end tests for user workflows
- Performance testing for concurrent agent scenarios

#### Pull Request Process
1. Fork the repository
2. Create a feature branch
3. Implement changes with tests
4. Update documentation
5. Submit PR with detailed description

### Feature Requests

Priority areas for contributions:
- Enhanced meeting analytics
- Additional AI provider integrations
- Mobile-responsive UI improvements
- Advanced scheduling capabilities
- Webhook integrations for external systems

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## 🆘 Support

### Documentation
- [API Reference](docs/api.md)
- [Configuration Guide](docs/configuration.md)
- [Deployment Guide](docs/deployment.md)

### Community
- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: Community Q&A and ideas
- **Discord**: Real-time chat support (if available)

### Professional Support
For enterprise deployments and professional support, contact the development team for consulting services.

---

**DealSense** - Empowering teams with intelligent AI meeting assistants. Transform your meetings from reactive to proactive with automated participation, real-time insights, and comprehensive analysis.

*Last Updated: September 2025*

## Features

- **Multi-Agent Management**: Start/stop multiple agents simultaneously
- **Real-time Monitoring**: Live status updates and transcripts
- **Configuration Management**: Save and reuse agent configurations
- **Meeting Control**: Join, leave, mute, and chat controls
- **Log Management**: View and export agent logs
- **Responsive Design**: Works on desktop and mobile devices

## Technology Stack

- **Frontend**: React with TypeScript, Tailwind CSS
- **Backend**: FastAPI with Python
- **Real-time**: WebSocket for live updates
- **State Management**: Zustand for client-side state
- **UI Components**: Shadcn/ui for consistent design

## Prerequisites

1. **joinly.ai server running on localhost:8080**
   ```bash
   # In your joinly directory
   docker run --env-file .env ghcr.io/joinly-ai/joinly:latest --server
   ```

2. **Python 3.12+** for the backend
3. **Node.js 18+** for the frontend

## Quick Start

### Backend Setup
```bash
cd backend
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r requirements.txt
python main.py
```

### Frontend Setup
```bash
cd frontend
npm install
npm run dev
```

### Access the Dashboard
Open http://localhost:3000

## Environment Variables

### Backend (.env in backend directory)
```bash
# Backend configuration
JOINLY_MANAGER_HOST=0.0.0.0
JOINLY_MANAGER_PORT=8001

# Joinly server connection
JOINLY_SERVER_URL=http://localhost:8080
```

### Frontend (.env.local in frontend directory)
```bash
# API endpoint
NEXT_PUBLIC_API_URL=http://localhost:8001
```

### Agent API Keys (passed when creating agents)
```bash
# LLM Keys
OPENAI_API_KEY=your_openai_key
ANTHROPIC_API_KEY=your_anthropic_key

# TTS Keys
ELEVENLABS_API_KEY=your_elevenlabs_key
DEEPGRAM_API_KEY=your_deepgram_key
```

## Usage

1. **Start the joinly server** on localhost:8080
2. **Launch the backend**: `cd backend && python main.py`
3. **Launch the frontend**: `cd frontend && npm run dev`
4. **Create agents** through the dashboard
5. **Monitor and manage** your agents in real-time

## API Documentation

### REST Endpoints
- `GET /agents` - List all agents
- `POST /agents` - Create new agent
- `GET /agents/{id}` - Get agent details
- `POST /agents/{id}/start` - Start agent
- `POST /agents/{id}/stop` - Stop agent
- `GET /agents/{id}/logs` - Get agent logs

### WebSocket
- `ws://localhost:8001/ws/agents/{id}` - Real-time agent updates

## Development

### Backend
```bash
cd backend
pip install -r requirements.txt
python main.py  # Auto-reloads on changes
```

### Frontend
```bash
cd frontend
npm install
npm run dev     # Auto-reloads on changes
npm run build   # Production build
```

## Deployment

### Docker Deployment
```dockerfile
# Backend
FROM python:3.12-slim
WORKDIR /app
COPY backend/requirements.txt .
RUN pip install -r requirements.txt
COPY backend/ .
EXPOSE 8001
CMD ["python", "main.py"]

# Frontend
FROM node:18-alpine
WORKDIR /app
COPY frontend/package*.json .
RUN npm ci
COPY frontend/ .
RUN npm run build
EXPOSE 3000
CMD ["npm", "start"]
```

### Production Considerations
- Use reverse proxy (nginx) for production
- Set up proper CORS configuration
- Configure SSL/TLS
- Use environment-specific configurations
- Set up monitoring and logging

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details.
