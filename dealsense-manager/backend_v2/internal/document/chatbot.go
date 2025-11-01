package document

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/database"
)

// ChatbotService handles RAG-based chatbot queries over meeting data and documents
type ChatbotService struct {
	db          *database.Database
	docService  *Service
	llmProvider llm.GroundingCapableProvider
}

// NewChatbotService creates a new chatbot service
func NewChatbotService(db *database.Database, docService *Service, llmProvider llm.GroundingCapableProvider) *ChatbotService {
	return &ChatbotService{
		db:          db,
		docService:  docService,
		llmProvider: llmProvider,
	}
}

// ChatRequest represents a chat query request
type ChatRequest struct {
	AgentID    uuid.UUID  `json:"agent_id"`
	SessionID  string     `json:"session_id"`
	Query      string     `json:"query"`
	DocumentID *uuid.UUID `json:"document_id,omitempty"` // Optional: limit to specific document
	TopK       int        `json:"top_k"`                 // Number of context chunks to retrieve
}

// ChatResponse represents a chat response
type ChatResponse struct {
	SessionID     string         `json:"session_id"`
	Query         string         `json:"query"`
	Response      string         `json:"response"`
	ContextChunks []ContextChunk `json:"context_chunks"`
	Sources       []Source       `json:"sources"`
	TokenCount    int            `json:"token_count"`
	ResponseTime  float64        `json:"response_time_ms"`
}

// ContextChunk represents a retrieved context chunk
type ContextChunk struct {
	Text          string  `json:"text"`
	Source        string  `json:"source"` // "document" or "meeting"
	PageNumber    int     `json:"page_number"`
	Similarity    float32 `json:"similarity"`
	DisplaySource string  `json:"display_source"`
}

// Source represents a source document or meeting
type Source struct {
	Type string `json:"type"` // "document" or "meeting"
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Query processes a chatbot query with RAG
func (c *ChatbotService) Query(req ChatRequest) (*ChatResponse, error) {
	startTime := time.Now()
	responseTime := int64(0)

	// Validate request
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}
	if req.SessionID == "" {
		req.SessionID = uuid.New().String()
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	logrus.Infof("Processing chatbot query for agent %s: %s", req.AgentID.String(), req.Query)

	// Step 1: Retrieve relevant context from documents
	documentContext, docSources, err := c.retrieveDocumentContext(req)
	if err != nil {
		logrus.Warnf("Failed to retrieve document context: %v", err)
		return &ChatResponse{
			SessionID:    req.SessionID,
			Query:        req.Query,
			Response:     "I encountered an error while retrieving document context. Please try again later.",
			ResponseTime: float64(time.Since(startTime).Milliseconds()) / 1000.0,
		}, nil
	}

	// Step 2: Retrieve relevant context from meeting transcripts
	meetingContext, meetingSources, err := c.retrieveMeetingContext(req)
	if err != nil {
		logrus.Warnf("Failed to retrieve meeting context: %v", err)
		meetingContext = []ContextChunk{}
	}

	// Step 3: Combine contexts
	allContext := append(documentContext, meetingContext...)
	allSources := append(docSources, meetingSources...)

	// Get counts for better messaging
	var docCount int64
	c.db.Model(&database.Document{}).Where("agent_id = ? AND status = ?", req.AgentID, "processed").Count(&docCount)

	var meetingCount int64
	c.db.Model(&database.TranscriptSegment{}).Where("agent_id = ?", req.AgentID).Count(&meetingCount)

	var pastChatHistory []database.ChatMessage
	err = c.db.Where("agent_id = ? AND session_id = ?", req.AgentID, req.SessionID).
		Order("created_at ASC").
		Select("role, content").
		Find(&pastChatHistory).Error
	if err != nil {
		logrus.Warnf("Failed to retrieve past chat history: %v", err)
	}

	// Log what context is available
	hasDocContext := len(documentContext) > 0
	hasMeetingContext := len(meetingContext) > 0
	hasAnyData := docCount > 0 || meetingCount > 0

	if hasDocContext && !hasMeetingContext {
		logrus.Infof("Using document context only (%d chunks)", len(documentContext))
	} else if !hasDocContext && hasMeetingContext {
		logrus.Infof("Using meeting transcript context only (%d chunks)", len(meetingContext))
	} else if hasDocContext && hasMeetingContext {
		logrus.Infof("Using both document (%d) and meeting (%d) context", len(documentContext), len(meetingContext))
	} else if hasAnyData {
		logrus.Infof("Data available (docs: %d, transcripts: %d) but no relevant context found", docCount, meetingCount)
	} else {
		logrus.Infof("No data available for this agent")
	}

	// ONLY return predefined message if BOTH documents AND transcripts are completely absent
	if len(allContext) == 0 && docCount == 0 && meetingCount == 0 {
		return &ChatResponse{
			SessionID:    req.SessionID,
			Query:        req.Query,
			Response:     "I don't have any data to work with. No documents or meeting transcripts have been uploaded for this agent yet. Please upload some documents or start a meeting to enable me to answer your questions.",
			ResponseTime: float64(time.Since(startTime).Milliseconds()) / 1000.0,
		}, nil
	}

	// If we have data but no relevant context, let the LLM handle it gracefully
	// The LLM will be informed about what data sources are available
	if len(allContext) == 0 {
		logrus.Warnf("No relevant context found, but proceeding with LLM using general knowledge")
		// Create an informative context chunk to guide the LLM
		infoChunk := ContextChunk{
			Text:       fmt.Sprintf("Note: The system searched through %d document(s) and meeting transcripts but did not find specific content matching this query. The available data may not contain information relevant to this question.", docCount),
			Source:     "system",
			PageNumber: 0,
			Similarity: 0,
		}
		allContext = append(allContext, infoChunk)
	}

	// Step 4: Build prompt with retrieved context
	prompt := c.buildRAGPrompt(req.Query, allContext, pastChatHistory)

	// Step 5: Call LLM
	response, err := c.llmProvider.CallWithGrounding(prompt, map[string]interface{}{
		"maxOutputTokens": 2000,
		"temperature":     1,
		"topP":            0.6,
		"topK":            40,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get LLM response: %w", err)
	}

	responseTime = time.Since(startTime).Milliseconds()

	// Step 6: Store chat message
	chatResp := &ChatResponse{
		SessionID:     req.SessionID,
		Query:         req.Query,
		Response:      response.Text,
		ContextChunks: allContext,
		Sources:       allSources,
		TokenCount:    len(strings.Fields(response.Text)),
		ResponseTime:  float64(responseTime) / 1000.0, // Convert milliseconds to seconds
	}

	// Store user message
	c.storeChatMessage(req.AgentID, req.DocumentID, req.SessionID, "user", req.Query, []ContextChunk{})

	// Store assistant response
	c.storeChatMessage(req.AgentID, req.DocumentID, req.SessionID, "assistant", response.Text, allContext)

	logrus.Infof("Chatbot query completed in %dms with %d context chunks", responseTime, len(allContext))
	return chatResp, nil
}

// retrieveDocumentContext retrieves relevant document chunks
func (c *ChatbotService) retrieveDocumentContext(req ChatRequest) ([]ContextChunk, []Source, error) {
	// Use document service to search for similar chunks
	results, err := c.docService.SearchDocuments(req.AgentID, req.Query, req.TopK)
	if err != nil {
		return nil, nil, err
	}

	var contexts []ContextChunk
	sourcesMap := make(map[string]Source)

	for _, result := range results {
		contexts = append(contexts, ContextChunk{
			Text:          result.ChunkText,
			Source:        "document",
			PageNumber:    result.PageNumber,
			Similarity:    result.Similarity,
			DisplaySource: "Document ID: " + result.DocumentID.String(),
		})

		// Track unique document sources (would need to fetch document info)
		sourcesMap["doc_"+fmt.Sprint(result.ChunkIndex)] = Source{
			Type: "document",
			ID:   fmt.Sprint(result.ChunkIndex),
			Name: "Document", // Could enhance with actual document name
		}
	}

	sources := make([]Source, 0, len(sourcesMap))
	for _, src := range sourcesMap {
		sources = append(sources, src)
	}

	return contexts, sources, nil
}

// retrieveMeetingContext retrieves relevant meeting transcript chunks
func (c *ChatbotService) retrieveMeetingContext(req ChatRequest) ([]ContextChunk, []Source, error) {
	// Fetch recent meeting transcripts for the agent
	var transcripts []database.TranscriptSegment
	err := c.db.
		Where("agent_id = ?", req.AgentID).
		Order("timestamp DESC").
		Limit(100). // Last 100 segments
		Find(&transcripts).Error

	if err != nil {
		return nil, nil, err
	}

	if len(transcripts) == 0 {
		return []ContextChunk{}, []Source{}, nil
	}

	// For simplicity, use keyword matching or could enhance with embeddings
	// Here we'll do simple keyword-based filtering
	queryKeywords := extractKeywords(req.Query)
	var relevantSegments []database.TranscriptSegment

	for _, seg := range transcripts {
		if containsAnyKeyword(seg.Text, queryKeywords) {
			relevantSegments = append(relevantSegments, seg)
			if len(relevantSegments) >= req.TopK {
				break
			}
		}
	}

	var contexts []ContextChunk
	sourcesMap := make(map[string]Source)

	for _, seg := range relevantSegments {
		speakerName := "Unknown"
		if seg.Speaker != nil {
			speakerName = *seg.Speaker
		}

		contexts = append(contexts, ContextChunk{
			Text:       fmt.Sprintf("[%s]: %s", speakerName, seg.Text),
			Source:     "meeting",
			PageNumber: 0,
			Similarity: 0.5, // Placeholder similarity
		})

		sourcesMap["meeting"] = Source{
			Type: "meeting",
			ID:   req.AgentID.String(),
			Name: "Meeting Transcript",
		}
	}

	sources := make([]Source, 0, len(sourcesMap))
	for _, src := range sourcesMap {
		sources = append(sources, src)
	}

	return contexts, sources, nil
}

// buildRAGPrompt constructs a prompt with retrieved context
func (c *ChatbotService) buildRAGPrompt(query string, contexts []ContextChunk, pastChatHistory []database.ChatMessage) string {
	var prompt strings.Builder

	// Count context sources
	docContextCount := 0
	meetingContextCount := 0
	systemContextCount := 0
	for _, ctx := range contexts {
		if ctx.Source == "document" {
			docContextCount++
		} else if ctx.Source == "meeting" {
			meetingContextCount++
		} else if ctx.Source == "system" {
			systemContextCount++
		}
	}

	prompt.WriteString("You are an intelligent AI assistant helping analyze startup meetings and pitch documents. ")
	prompt.WriteString("You are equipped with advanced capabilities including access to Google Search (Google Grounding) for real-time information when needed.\n\n")

	// Provide clear guidance based on available data
	prompt.WriteString("CAPABILITIES:\n")
	if docContextCount > 0 && meetingContextCount > 0 {
		prompt.WriteString("- Access to pitch documents and meeting transcripts (provided below)\n")
		prompt.WriteString("- Google Search capability for latest market data, news, and external information\n")
		prompt.WriteString("- Use all available context to provide comprehensive answers\n\n")
	} else if docContextCount > 0 {
		prompt.WriteString("- Access to pitch documents (provided below)\n")
		prompt.WriteString("- Google Search capability for latest market data, news, and external information\n")
		prompt.WriteString("- If questions are about meetings, acknowledge transcripts aren't available but use documents and search\n\n")
	} else if meetingContextCount > 0 {
		prompt.WriteString("- Access to meeting transcripts (provided below)\n")
		prompt.WriteString("- Google Search capability for latest market data, news, and external information\n")
		prompt.WriteString("- If questions need detailed documents, acknowledge unavailability but use transcripts and search\n\n")
	} else if systemContextCount > 0 {
		prompt.WriteString("- Google Search capability for latest market data, news, and external information\n")
		prompt.WriteString("- No directly relevant context found in local data\n")
		prompt.WriteString("- Use Google Search for real-time information and general knowledge to provide helpful responses\n\n")
	}

	prompt.WriteString("RESPONSE FORMATTING GUIDELINES:\n")
	prompt.WriteString("- Structure your responses with clear sections using markdown headers (##)\n")
	prompt.WriteString("- Use bullet points (- or *) for lists and key points\n")
	prompt.WriteString("- Use **bold** for emphasis on important terms or figures\n")
	prompt.WriteString("- For numerical data or metrics, format them clearly (e.g., $5.2M, 45% growth)\n")
	prompt.WriteString("- Include relevant emojis sparingly for visual appeal (📊 📈 💡 ⚠️ ✅)\n")
	prompt.WriteString("- When citing information from Google Search, mention it naturally (e.g., 'According to recent market data...')\n")
	prompt.WriteString("- Be concise yet comprehensive - aim for clarity and readability\n\n")

	prompt.WriteString("GUIDELINES:\n")
	prompt.WriteString("- Always be honest about information sources and limitations\n")
	prompt.WriteString("- Use Google Search for current market trends, competitor info, or external validation\n")
	prompt.WriteString("- If you can partially answer, do so and explain what additional information would help\n")
	prompt.WriteString("- Provide actionable insights when possible\n\n")

	// Only include context section if there's actual content (not just system messages)
	if docContextCount > 0 || meetingContextCount > 0 {
		prompt.WriteString("AVAILABLE CONTEXT:\n")
		prompt.WriteString("---\n")
		for i, ctx := range contexts {
			// Skip system context in the detailed listing
			if ctx.Source == "system" {
				continue
			}
			prompt.WriteString(fmt.Sprintf("\n[Context %d - Source: %s", i+1, ctx.Source))
			if ctx.PageNumber > 0 {
				prompt.WriteString(fmt.Sprintf(", Page: %d", ctx.PageNumber))
			}
			if ctx.Similarity > 0 {
				prompt.WriteString(fmt.Sprintf(", Relevance: %.2f", ctx.Similarity))
			}
			prompt.WriteString("]\n")
			prompt.WriteString(ctx.Text)
			prompt.WriteString("\n")
		}
		prompt.WriteString("---\n\n")
	} else {
		// Only system context - remind about limitations
		for _, ctx := range contexts {
			if ctx.Source == "system" {
				prompt.WriteString(fmt.Sprintf("SYSTEM NOTE: %s\n\n", ctx.Text))
			}
		}
	}

	if len(pastChatHistory) > 0 {
		prompt.WriteString("PAST CONVERSATION HISTORY:\n")
		prompt.WriteString("---\n")
		for _, msg := range pastChatHistory {
			prompt.WriteString(fmt.Sprintf("[%s]: %s\n", msg.Role, msg.Content))
		}
		prompt.WriteString("---\n\n")
	}

	prompt.WriteString(fmt.Sprintf("USER QUESTION: %s\n\n", query))

	if docContextCount > 0 || meetingContextCount > 0 {
		prompt.WriteString("ANSWER: Based on the context provided above, ")
	} else {
		prompt.WriteString("ANSWER: ")
	}

	return prompt.String()
}

// storeChatMessage stores a chat message in the database
func (c *ChatbotService) storeChatMessage(agentID uuid.UUID, documentID *uuid.UUID, sessionID string, role string, content string, contexts []ContextChunk) {
	// Convert contexts to JSON string
	contextsJSON, err := json.Marshal(contexts)
	if err != nil {
		logrus.Errorf("Failed to marshal contexts to JSON: %v", err)
		contextsJSON = []byte("[]")
	}

	chatMsg := &database.ChatMessage{
		AgentID:       agentID,
		DocumentID:    documentID,
		SessionID:     sessionID,
		Role:          role,
		Content:       content,
		ContextChunks: string(contextsJSON),
		TokenCount:    len(strings.Fields(content)),
	}

	if err := c.db.Create(chatMsg).Error; err != nil {
		logrus.Errorf("Failed to store chat message: %v", err)
	}
}

// GetChatHistory retrieves chat history for a session
func (c *ChatbotService) GetChatHistory(sessionID string) ([]database.ChatMessage, error) {
	var messages []database.ChatMessage
	err := c.db.
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	return messages, nil
}

// extractKeywords extracts keywords from a query (simple implementation)
func extractKeywords(query string) []string {
	// Remove common stop words and split
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"is": true, "are": true, "was": true, "were": true, "in": true, "on": true,
		"at": true, "to": true, "for": true, "of": true, "with": true, "by": true,
		"what": true, "when": true, "where": true, "who": true, "how": true, "why": true,
	}

	words := strings.Fields(strings.ToLower(query))
	var keywords []string

	for _, word := range words {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// containsAnyKeyword checks if text contains any of the keywords
func containsAnyKeyword(text string, keywords []string) bool {
	lowerText := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

// StreamQuery processes a query with streaming response (for future enhancement)
func (c *ChatbotService) StreamQuery(ctx context.Context, req ChatRequest, responseChan chan<- string) error {
	// Placeholder for streaming implementation
	// Would use streaming-capable LLM provider
	response, err := c.Query(req)
	if err != nil {
		return err
	}

	responseChan <- response.Response
	close(responseChan)
	return nil
}
