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
	Text       string  `json:"text"`
	Source     string  `json:"source"` // "document" or "meeting"
	PageNumber int     `json:"page_number"`
	Similarity float32 `json:"similarity"`
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
		documentContext = []ContextChunk{}
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
	prompt := c.buildRAGPrompt(req.Query, allContext)

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
			Text:       result.ChunkText,
			Source:     "document",
			PageNumber: result.PageNumber,
			Similarity: result.Similarity,
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
func (c *ChatbotService) buildRAGPrompt(query string, contexts []ContextChunk) string {
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

	prompt.WriteString("You are an intelligent assistant helping analyze startup meetings and pitch documents. ")

	// Provide clear guidance based on available data
	if docContextCount > 0 && meetingContextCount > 0 {
		prompt.WriteString("You have access to both pitch documents and meeting transcripts. ")
		prompt.WriteString("Use all available context to provide a comprehensive answer. ")
	} else if docContextCount > 0 {
		prompt.WriteString("You have access to pitch documents only. ")
		prompt.WriteString("If the question is about meeting discussions or conversations, acknowledge that meeting transcripts are not available. ")
		prompt.WriteString("Focus on what you can learn from the documents. ")
	} else if meetingContextCount > 0 {
		prompt.WriteString("You have access to meeting transcripts only. ")
		prompt.WriteString("If the question requires information from pitch documents or detailed startup materials, acknowledge that documents are not available. ")
		prompt.WriteString("Focus on what was discussed in the meeting. ")
	} else if systemContextCount > 0 {
		// System context indicates data exists but wasn't relevant
		prompt.WriteString("Note: The available data did not contain specific information matching this query. ")
		prompt.WriteString("Provide a helpful response based on your general knowledge, but clearly indicate that this is not from the uploaded data. ")
	}

	prompt.WriteString("Always be honest about what information you have and don't have. ")
	prompt.WriteString("If you can partially answer, do so and explain what additional information would be needed.\n\n")

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
