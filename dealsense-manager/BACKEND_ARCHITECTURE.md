<!--
  Comprehensive Backend Architecture Documentation for <product_name>
  Focus: Go backend (backend_v2) + MCP integration + AI configuration pathways
  NOTE: Replace <product_name> with the actual product brand/name during distribution.
-->

# <product_name> Backend Architecture (Go + MCP)

> Version: 1.0  
> Last Updated: 2025-09-21  
> Scope: `joinly-manager-ui/backend_v2` (Go 1.24) – excludes React/Next.js frontend except where interaction contracts matter.

---
## 1. Executive Summary
The <product_name> backend is a **concurrent, event‑driven orchestration layer** for managing autonomous AI meeting agents. It mediates between:

1. User / UI control surface (REST + WebSocket)  
2. AI service configuration (LLM, STT, TTS)  
3. External meeting participation + analysis via MCP (Model Context Protocol) compatible Joinly core endpoint (`JOINLY_URL`)  
4. Real‑time telemetry distribution (logs, status, transcript, analysis)

The design emphasizes **isolation per agent**, **graceful lifecycle control**, **low-latency propagation of state**, and **observability hooks** (Discord + WebSocket + structured logs).

---
## 2. High-Level Architecture

```
┌────────────────────────────────────────────────────────────────────────────┐
│                                <product_name>                              │
│                                                                            │
│  ┌──────────────┐     ┌──────────────────┐      ┌─────────────────────┐    │
│  │  Frontend    │ --> │  REST API Layer  │ -->  │   Agent Manager      │    │
│  │(Next.js App) │     │ (Gin Handlers)   │      │ (Lifecycle + State) │    │
│  └──────────────┘     └──────────────────┘      └─────────┬───────────┘    │
│           ▲                     │  ▲                               │        │
│           │ WebSocket Session   │  │ Agent-Specific                 │        │
│           │ (aggregate stream)  │  │ WebSocket Streams              │        │
│           │                     │  │                               │        │
│           │             ┌───────▼──▼──────┐                       │        │
│           │             │  WebSocket Hub  │ <──────────────┐      │        │
│           │             └─────────────────┘                │      │        │
│           │                                                │      │        │
│           │                                      ┌──────────▼──────▼───┐   │
│           │                                      │  MCP / Joinly Core  │   │
│           │                                      │ (External Runtime)  │   │
│           │                                      └──────────┬──────────┘   │
│           │                                                 │              │
│  ┌────────▼────────┐    ┌────────────────┐    ┌─────────────▼─────────┐    │
│  │Discord Webhooks │←---│ Structured Logs│←---│  Agent Execution Loops │    │
│  └─────────────────┘    └────────────────┘    └───────────────────────┘    │
└────────────────────────────────────────────────────────────────────────────┘
```

Key Pillars:
- **Agent Manager**: In-memory authoritative state machine per agent
- **WebSocket Hub**: Fan‑out event bus with separation between *session* listeners (all agents) and *agent* listeners
- **Configuration Layer**: Env-driven overrides + .env loading cascade + defaults
- **MCP Integration**: Abstract client enabling AI and meeting operations via standardized protocol
- **Extensibility**: Future database layer (placeholder in `DatabaseConfig`) and additional transports.

---
## 3. Source Layout (Backend)

| Path | Responsibility | Notes |
|------|----------------|-------|
| `cmd/server/main.go` | Process bootstrap | Config load, logging init, manager + router start, graceful shutdown |
| `internal/config/` | Configuration + Discord logging hook | Merges defaults + env + .env; sets log format & hooks |
| `internal/api/` | HTTP handlers & routing | Thin translation layer to manager operations |
| `internal/manager/` | Core orchestration | Agent maps, meeting maps, lifecycle control, concurrency primitives |
| `internal/models/` | Domain types | Strong typed configurations, status enums, analysis types |
| `internal/websocket/` | Hub & client pumps | Broadcast, session vs agent channels |
| `internal/client/` | (Implied) MCP adapter & Analyst agent | Bridges to external Joinly/MCP runtime for speech / analysis |
| `data/analysis/` | Persisted meeting analysis artifacts | JSON structured summaries (analyst mode) |

---
## 4. Configuration System

### 4.1 Load Order
1. Hardcoded defaults (`DefaultConfig()`)  
2. Local `.env` in `backend_v2/` (higher priority)  
3. Parent repo `.env` (fallback)  
4. Process environment variables (override)  

### 4.2 Structured Config Object (Selected Fields)

```go
type Config struct {
  Server   ServerConfig
  Logging  LoggingConfig
  Joinly   JoinlyConfig
  Database DatabaseConfig // reserved for future persistence
}

type ServerConfig struct {
  Host string
  Port int
  ReadTimeout  time.Duration
  WriteTimeout time.Duration
  CORS CORSConfig
}

type JoinlyConfig struct {
  DefaultURL     string        // MCP endpoint (Joinly core)
  DefaultTimeout time.Duration // Default operation timeout
  MaxAgents      int           // Concurrency cap
}
```

### 4.3 Environment Variable Mapping

| Env Var | Field | Purpose |
|---------|-------|---------|
| `SERVER_HOST` | `Server.Host` | Bind address |
| `SERVER_PORT` | `Server.Port` | HTTP port |
| `LOG_LEVEL` | `Logging.Level` | Verbosity (debug, info, warn, error) |
| `LOG_FORMAT` | `Logging.Format` | `json` or text |
| `DISCORD_LOGGING_ENABLED` | `Logging.Discord.Enabled` | Enable webhook dispatch |
| `DISCORD_GEMINI_LOGGING_ENABLED` | `Logging.Discord.GeminiEnabled` | Special Gemini filtering channel |
| `DISCORD_INFO_WEBHOOK` | Info channel | General informational logs |
| `DISCORD_ERROR_WEBHOOK` | Error channel | Error / panic alerts |
| `DISCORD_DEBUG_WEBHOOK` | Debug channel | Verbose dev traces |
| `DISCORD_GEMINI_WEBHOOK` | Gemini channel | LLM provider (Google) specific introspection |
| `JOINLY_URL` | `Joinly.DefaultURL` | MCP root endpoint |
| `MAX_AGENTS` | `Joinly.MaxAgents` | Capacity throttle |

### 4.4 Discord Logging Hook
The custom `DiscordHook` implements `logrus.Hook`. Filtering logic:
1. Determines appropriate webhook by level (or Gemini substring match if enabled).
2. Builds an embed with truncated (≤1024 chars) fields for auxiliary key-value pairs.
3. Emits asynchronous HTTP POST (10s timeout).

Failure to deliver does not abort core process — resilience over reliability.

---
## 5. Domain & Data Model Overview

### 5.1 Agent Lifecycle States
`created → starting → running → stopping → stopped` (terminal) or `error` (recoverable if restartable). 

State transitions enforced by `AgentManager` methods (atomic under lock). Example transitions:

```
 [createAgent] -> created
 [startAgent]  created|stopped -> starting -> running | error
 [stopAgent]   running|starting -> stopping -> stopped
 [runtimeFault] running -> error
```

### 5.2 Key Structs (Excerpt)
```go
type Agent struct {
  ID        string
  Config    AgentConfig
  Status    AgentStatus
  CreatedAt time.Time
  StartedAt *time.Time
  Logs      []LogEntry // Rolling in-memory (buffered via manager)
}

type AgentConfig struct {
  Name             string
  MeetingURL       string
  LLMProvider      LLMProvider  // openai | anthropic | google | ollama
  LLMModel         string
  TTSProvider      TTSProvider  // kokoro | elevenlabs | deepgram
  STTProvider      STTProvider  // whisper | deepgram
  Language         string       // ISO code
  CustomPrompt     *string
  NameTrigger      bool
  AutoJoin         bool
  ConversationMode ConversationMode // conversational | analyst
  UtteranceTailSeconds *float64
  NoSpeechEventDelay   *float64
  MaxSTTTasks          *int
  WindowQueueSize      *int
  EnvVars map[string]string // Provider credentials & overrides
}
```

### 5.3 Meeting & Analysis Storage
Analyst-mode output persisted as JSON in `data/analysis/meeting_analysis_<agentID>_<timestamp>.json`. Structure includes transcript segments, summary, grounded summary, key points, topics (time-coded), participants, sentiment & keyword extraction.

---
## 6. Concurrency & Synchronization Strategy

| Mechanism | Purpose |
|-----------|---------|
| `sync.RWMutex` (manager) | Protect shared maps: agents, meetings, logs, conversation history |
| `context.Context` per agent | Cancellation cascade for goroutines on stop / shutdown |
| `sync.WaitGroup` | Graceful drain during manager shutdown |
| Channel buffers (WebSocket hub) | Decouple producer (agent events) from slow consumers |
| Log buffering (slice per agent) | Fixed-size ring (logical – current implementation truncation handled externally) |

### 6.1 Backpressure Considerations
- WebSocket broadcast channel size: 256. If saturated, messages are dropped with warning (trade-off: resilience > strict delivery).
- Potential future improvement: pluggable queue with metrics & overflow strategies.

### 6.2 Cancellation Flow
`AgentManager.Stop()` → cancels global root context → per-agent cancelFuncs invoked → utterance task contexts aborted → websockets closed → WaitGroup drained.

---
## 7. WebSocket Event Distribution

### 7.1 Hub Design
`Hub` maintains:
- `clients` (all)  
- `clientsByAgent[agentID]`  
- `sessionClients` (receive every message)  

Broadcast strategies:
```go
BroadcastToAgent(agentID, message) // fan-out to agent listeners + session clients
Broadcast(message)                 // fan-out to ALL clients
```

### 7.2 Message Envelope
```go
type WebSocketMessage struct {
  Type    string                 // status | log | error | transcript | analysis
  AgentID string
  Data    map[string]interface{}
  Timestamp time.Time
}
```

### 7.3 Extension Hooks
Future extensions (e.g., command channel) can re-use `readPump()` to interpret inbound control frames (currently discarded).

---
## 8. MCP (Model Context Protocol) Integration

Although not fully expanded in the shared snippet, the backend imports `github.com/mark3labs/mcp-go`. The **MCP client layer** (in `internal/client/`) conceptually abstracts:

1. **Session Establishment**: Connect to `JoinlyConfig.DefaultURL` (e.g., `http://<host>:8000/mcp/`) establishing a protocol session.
2. **Tool Invocation**: Request transcription, synthesis, retrieval, or analysis tools exposed by the MCP server.
3. **Streaming Handling**: Subscribe to streaming events (e.g., partial transcription) and forward to hub.
4. **Analyst Agent Mode**: Special wrapper (`AnalystAgent`) performing post-processing summarization + extraction.

### 8.1 Why MCP?
- Provider neutrality via tool abstraction
- Formalized context passing (prompts, transcripts, roles)
- Enables multi-model orchestration under unified contract

### 8.2 MCP Error Domains (Conceptual)
| Domain | Example Cause | Handling Pattern |
|--------|---------------|------------------|
| Transport | Network timeout | Retry with backoff / escalate to error state |
| Tool | Unsupported model | Mark agent `error`; log with provider metadata |
| Auth | Invalid API key | Immediate fail; require user-supplied env var fix |
| Rate | Provider throttling | Adaptive delay (future) |

### 8.3 Credential Injection
`AgentConfig.EnvVars` supplies per-agent credentials (e.g., `OPENAI_API_KEY`, `DEEPGRAM_API_KEY`). These are marshaled into the MCP session context (implementation detail) so multiple agents can use distinct API keys concurrently.

### 8.4 Analyst Mode Post-Processing Pipeline (Conceptual)
```
Raw Transcript Segments --> Normalization --> Chunking --> LLM Summarization --> 
Grounded Summary Validation (optional) --> Key Point Extraction --> Sentiment + Keyword Pass --> Persist JSON
```

---
## 9. AI Provider Abstraction

### 9.1 Supported Provider Enumerations
```go
type LLMProvider string
const (
  LLMProviderOpenAI    LLMProvider = "openai"
  LLMProviderAnthropic LLMProvider = "anthropic"
  LLMProviderGoogle    LLMProvider = "google"
  LLMProviderOllama    LLMProvider = "ollama"
)
```

### 9.2 Model Selection
User supplies `LLMModel` (e.g., `gpt-4o`, `claude-3-opus`, `gemini-pro`, `llama3:70b`). Backend does **no internal gating**; validation delegated to downstream MCP tool invocation.

### 9.3 Speech Stack
| Layer | Enum | Purpose |
|-------|------|---------|
| STT | `whisper`, `deepgram` | Converts live meeting audio to text for triggers & analysis |
| TTS | `kokoro`, `elevenlabs`, `deepgram` | Synthesizes agent spoken responses |

### 9.4 Conversation Modes
| Mode | Behavior |
|------|----------|
| `conversational` | Bidirectional: listens + speaks; may gate responses via name trigger |
| `analyst` | Passive ingestion only; generates analysis artifacts; no speech synthesis |

### 9.5 Trigger Controls
`NameTrigger == true`: Response pipeline only activates if agent name detected in recent utterance window (semantic matching can be extended; currently lexical assumption).

### 9.6 Transcription Tuning Parameters
| Parameter | Role |
|-----------|------|
| `UtteranceTailSeconds` | Silence padding post-speech before finalizing segment |
| `NoSpeechEventDelay` | Debounce interval for silent period detection |
| `MaxSTTTasks` | Upper bound on parallel STT operations (avoid CPU / API saturation) |
| `WindowQueueSize` | Buffer size for streaming frames (latency vs completeness) |

---
## 10. Lifecycle Management

### 10.1 Creation
`POST /agents` → Validate + fill defaults (e.g., `UtteranceTailSeconds`). Agent stored with `created` status. Optional `AutoJoin` triggers async start path.

### 10.2 Start
1. Transition `created|stopped` → `starting`  
2. Spin goroutine: establish MCP session, allocate cancellation context  
3. On success: set `running`, broadcast status  
4. On failure: set `error`, capture `ErrorMsg`

### 10.3 Meeting Join
Explicit `POST /agents/{id}/join-meeting` ensures separation of *process activation* vs *meeting embedding* (useful for pre-warming agent context). Guards prevent duplicate join attempts.

### 10.4 Stop
1. Transition `running|starting` → `stopping`  
2. Cancel agent context → terminate loops  
3. Flush pending log buffer  
4. Transition → `stopped`

### 10.5 Deletion
`DELETE /agents/{id}` only permitted when not `running` (safety invariant). Removes all in-memory references; analysis artifacts remain on disk (historical record).

### 10.6 Error Recovery
User may restart from `error` state via `start` (treated like `stopped`). Root cause introspection through logs & (future) structured `ErrorMsg` taxonomy.

---
## 11. Observability & Telemetry

| Channel | Data Types | Latency Profile |
|---------|------------|-----------------|
| WebSocket (agent) | status, log, transcript, error, analysis | Sub-100ms typical (LAN) |
| WebSocket (session) | Aggregate of all agents | Slightly higher due to fan-out |
| Logs (stdout) | Structured JSON (if chosen) | Immediate |
| Discord Webhooks | Leveled events / Gemini traces | Network-dependent |
| Analysis JSON | Post-processed meeting summaries | On analysis cycle completion |

### 11.1 Log Entry Structure
```go
type LogEntry struct {
  Timestamp time.Time
  Level     string
  Message   string
}
```

### 11.2 Usage Statistics Endpoint (`/usage`)
Returns aggregate counts (total agents, active agents, uptime, API call counters). API call tracking can be augmented to differentiate provider usage.

---
## 12. Security Considerations

| Concern | Current Handling | Future Hardening |
|---------|------------------|------------------|
| Auth / AuthZ | None (open) | API keys / JWT / mTLS |
| Secrets | Provided via runtime `EnvVars` | Vault integration / encryption at rest |
| CORS | Explicit allowed origins list | Dynamic tenancy-based policies |
| WebSocket Origin Check | Hardcoded `localhost:3000` | Configurable whitelist + CSRF tokens |
| Rate Limiting | None | Token bucket per endpoint |
| Input Validation | Gin binding + type safety | Schema versioning / JSON schema validation |

---
## 13. Scalability & Performance

### 13.1 Horizontal Concerns
Current state is **single-process in-memory**. To scale horizontally:
1. Externalize agent state (Redis / Postgres)  
2. Distributed event bus (NATS / Redis Streams / Kafka) for WebSocket hub fan-out  
3. Shard agents by consistent hash (agentID) across replicas  

### 13.2 Hot Paths
- WebSocket broadcast loops (optimize by coalescing bursts)
- STT segmentation pipeline (CPU / network bound)
- LLM invocation (latency hiding via concurrency + streaming)

### 13.3 Optimization Opportunities
| Area | Potential Improvement |
|------|-----------------------|
| Log Buffer | Ring buffer to avoid slice growth / copying |
| Broadcast | Multi-producer ring or lock-free queue |
| Analysis | Incremental summarization vs full-pass |
| Credentials | Pooled HTTP clients per provider |
| Backpressure | Adaptive shedding based on consumer drain rate |

---
## 14. Failure Modes & Resilience

| Failure | Effect | Mitigation |
|---------|--------|------------|
| MCP endpoint unreachable | Agent start fails → `error` | Retry with exponential backoff (future) |
| Provider quota exceeded | Responses degrade | Circuit breaker + fallback model |
| WebSocket push slow consumer | Channel saturation → drop | Per-client send queue + eviction |
| Discord webhook down | Lost external log event | Local log unchanged | 
| Agent goroutine panic | Inconsistent state | Recover wrapper + mark `error` |

---
## 15. Extensibility Roadmap (Suggested)
| Feature | Rationale |
|---------|-----------|
| Persistent store | Survive process restarts; analytics retention |
| Pluggable auth | Multi-tenant SaaS deployment |
| Metrics exporter | Prometheus integration for ops |
| Structured error codes | Improve frontend remediation UX |
| AI policy layer | Enforce content filters / compliance |
| Event replay | Postmortem analysis / debugging |

---
## 16. Integration Contract (Frontend ↔ Backend)

| Operation | HTTP | Payload Notes | Side Effects |
|-----------|------|---------------|--------------|
| Create Agent | `POST /agents` | `AgentConfig` JSON | Broadcast `status` (created) |
| Start Agent | `POST /agents/{id}/start` | none | `starting`→`running` or `error` |
| Stop Agent | `POST /agents/{id}/stop` | none | `stopping`→`stopped` |
| Join Meeting | `POST /agents/{id}/join-meeting` | none | Meeting association updates |
| Fetch Logs | `GET /agents/{id}/logs?lines=N` | query-bound | Returns latest N entries |
| Analysis (raw) | `GET /agents/{id}/analysis` | Analyst only | JSON analysis artifact |
| Analysis (text) | `GET /agents/{id}/analysis/formatted` | Analyst only | Plain text summary |
| Usage Stats | `GET /usage` | none | Aggregate counters |

WebSocket URIs:
```
/ws/agents/{agent_id}   # Scoped stream
/ws/session             # Global stream
```

---
## 17. Sample Event Flow (Conversational Agent Startup)

```
Client -> POST /agents { auto_join: true } -------------------------------> Backend
Backend: Create agent (status=created), broadcast {type:status}
Auto-join goroutine: StartAgent()
  -> status: starting (broadcast)
  -> MCP session connect
  -> STT/TTS resource init
  -> status: running (broadcast)
Meeting join (auto or manual) triggers MCP tool chain → transcripts → broadcast {type:transcript}
User stops agent -> POST /agents/{id}/stop -> status: stopping -> context cancel -> status: stopped (broadcast)
```

---
## 18. Security / Compliance Extension Hooks
Potential insertion points:
| Hook Point | Purpose |
|------------|---------|
| Pre-Agent Creation | Validate provider key ownership / quotas |
| Pre-LLM Call (MCP) | PII redaction / policy tagging |
| Transcript Pipeline | Keyword watchlists / sentiment gating |
| Outbound TTS | Content moderation |

---
## 19. Operational Runbook (Core Tasks)

| Task | Command / Action | Notes |
|------|------------------|-------|
| Start backend (dev) | `go run cmd/server/main.go` | Requires `.env` for provider URLs |
| Health check | `curl :8001/` | 200 OK JSON |
| List agents | `curl :8001/agents` | JSON array |
| Tail logs (agent) | `curl :8001/agents/<id>/logs?lines=50` | Not streaming; use WebSocket for live |
| Get analysis | `curl :8001/agents/<id>/analysis` | Analyst mode only |

---
## 20. Glossary
| Term | Definition |
|------|------------|
| MCP | Model Context Protocol – standard for structured model tool invocation |
| STT | Speech-To-Text transcription layer |
| TTS | Text-To-Speech synthesis layer |
| Analyst Mode | Passive, analytical agent mode generating structured insights |
| Conversation Mode | Active speaking/participating agent mode |

---
## 21. Placeholder Compliance
All product references intentionally use `<product_name>`. Replace this token during distribution automation (e.g., CI substitution or templating pre-publish).

---
## 22. Summary
The <product_name> Go backend provides a **lean, composable control plane** over AI meeting agents with deliberate separation of *state orchestration*, *real-time dissemination*, and *provider abstraction*. Its current in-memory design accelerates iteration while leaving clear seams for persistence, scaling, and advanced governance.

---
**End of Document**
