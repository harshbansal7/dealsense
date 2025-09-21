package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/models"
)

// AnalysisData represents the comprehensive analysis data for a meeting
type AnalysisData struct {
	MeetingID         string            `json:"meeting_id"`
	MeetingURL        string            `json:"meeting_url"`
	StartTime         time.Time         `json:"start_time"`
	LastUpdated       time.Time         `json:"last_updated"`
	Transcript        []TranscriptEntry `json:"transcript"`
	Summary           string            `json:"summary"`
	GroundedSummary   *GroundedContent  `json:"grounded_summary,omitempty"`
	KeyPoints         []string          `json:"key_points"`
	GroundedKeyPoints *GroundedContent  `json:"grounded_key_points,omitempty"`
	ActionItems       []ActionItem      `json:"action_items"`
	Topics            []TopicDiscussion `json:"topics"`
	Participants      []string          `json:"participants"`
	DurationMinutes   float64           `json:"duration_minutes"`
	WordCount         int               `json:"word_count"`
	Sentiment         string            `json:"sentiment"`
	Keywords          []string          `json:"keywords"`
}

// TranscriptEntry represents a single transcript entry
type TranscriptEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Speaker   string    `json:"speaker"`
	Text      string    `json:"text"`
	IsAgent   bool      `json:"is_agent"`
}

// GroundedContent represents content with grounding information for the analyzer
type GroundedContent struct {
	Text              string                 `json:"text"`
	TextWithCitations string                 `json:"text_with_citations"`
	GroundingMetadata *llm.GroundingMetadata `json:"grounding_metadata,omitempty"`
}

// ActionItem represents an actionable item identified in the meeting
type ActionItem struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee,omitempty"`
	Priority    string    `json:"priority"`       // high, medium, low
	Type        string    `json:"type,omitempty"` // task, research, investigation, follow-up, decision
	Status      string    `json:"status"`         // pending, in_progress, completed
	CreatedAt   time.Time `json:"created_at"`
}

// TopicDiscussion represents a discussion topic identified in the meeting
type TopicDiscussion struct {
	Topic        string   `json:"topic"`
	StartTime    string   `json:"start_time"` // Changed to string to handle "HH:MM" format
	Duration     float64  `json:"duration_minutes"`
	Summary      string   `json:"summary"`
	Participants []string `json:"participants"`
}

// AnalystAgent handles meeting analysis and maintains comprehensive meeting notes
type AnalystAgent struct {
	agentID                 string
	config                  models.AgentConfig
	data                    *AnalysisData
	dataMutex               sync.RWMutex
	filePath                string
	llmClient               *JoinlyClient
	llmProvider             llm.LLMProvider
	lastAnalysis            time.Time
	analysisMutex           sync.Mutex
	currentAnalysisSnapshot []TranscriptEntry // Snapshot used during analysis to ensure consistency
}

// NewAnalystAgent creates a new analyst agent
func NewAnalystAgent(agentID string, config models.AgentConfig, llmClient *JoinlyClient) *AnalystAgent {
	// Create data directory if it doesn't exist
	dataDir := "data/analysis"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logrus.Errorf("Failed to create analysis data directory: %v", err)
	}

	fileName := fmt.Sprintf("meeting_analysis_%s_%d.json", agentID, time.Now().Unix())
	filePath := filepath.Join(dataDir, fileName)

	// Get LLM provider for structured responses
	llmProvider, err := llm.GetProvider(string(config.LLMProvider), config.LLMModel)
	if err != nil {
		logrus.Errorf("Failed to get LLM provider for analyst %s: %v", agentID, err)
		llmProvider = nil
	}

	analyst := &AnalystAgent{
		agentID:     agentID,
		config:      config,
		filePath:    filePath,
		llmClient:   llmClient,
		llmProvider: llmProvider,
		data: &AnalysisData{
			MeetingID:    agentID,
			MeetingURL:   config.MeetingURL,
			StartTime:    time.Now(),
			LastUpdated:  time.Now(),
			Transcript:   []TranscriptEntry{},
			KeyPoints:    []string{},
			ActionItems:  []ActionItem{},
			Topics:       []TopicDiscussion{},
			Participants: []string{},
		},
	}

	// Load existing analysis if file exists
	if err := analyst.loadAnalysis(); err != nil {
		logrus.Warnf("Could not load existing analysis for agent %s: %v", agentID, err)
	}

	return analyst
}

// ProcessUtterance processes a new utterance and updates the analysis
func (a *AnalystAgent) ProcessUtterance(segments []map[string]interface{}) {
	if len(segments) == 0 {
		return
	}

	a.dataMutex.Lock()
	defer a.dataMutex.Unlock()

	// Extract transcript text and speaker
	var fullText strings.Builder
	speaker := "Participant"
	timestamp := time.Now()

	for i, segment := range segments {
		if speakerVal, ok := segment["speaker"].(string); ok && speakerVal != "" {
			speaker = speakerVal
		}
		if text, ok := segment["text"].(string); ok && text != "" {
			if i > 0 {
				fullText.WriteString(" ")
			}
			fullText.WriteString(text)
		}
		if ts, ok := segment["timestamp"].(float64); ok {
			timestamp = time.Unix(int64(ts), 0)
		}
	}

	transcriptText := fullText.String()
	if transcriptText == "" {
		return
	}

	// Add to transcript
	entry := TranscriptEntry{
		Timestamp: timestamp,
		Speaker:   speaker,
		Text:      transcriptText,
		IsAgent:   false,
	}
	a.data.Transcript = append(a.data.Transcript, entry)

	// Update participants list
	a.updateParticipants(speaker)

	// Update metadata
	a.data.LastUpdated = time.Now()
	a.data.WordCount += len(strings.Fields(transcriptText))
	a.data.DurationMinutes = time.Since(a.data.StartTime).Minutes()

	// Save updated analysis
	if err := a.saveAnalysis(); err != nil {
		logrus.Errorf("Failed to save analysis for agent %s: %v", a.agentID, err)
	}

	// Trigger analysis update if enough time has passed (every 5 minutes or significant new content)
	if time.Since(a.lastAnalysis) > 5*time.Minute || len(a.data.Transcript)%20 == 0 {
		go a.updateAnalysis()
	}
}

// updateParticipants adds a speaker to the participants list if not already present
func (a *AnalystAgent) updateParticipants(speaker string) {
	for _, p := range a.data.Participants {
		if p == speaker {
			return
		}
	}
	a.data.Participants = append(a.data.Participants, speaker)
}

// updateAnalysis performs comprehensive analysis using LLM
func (a *AnalystAgent) updateAnalysis() {
	a.analysisMutex.Lock()
	defer a.analysisMutex.Unlock()

	a.lastAnalysis = time.Now()

	// Take a snapshot of the transcript with proper locking to ensure consistency
	a.dataMutex.RLock()
	transcriptSnapshot := make([]TranscriptEntry, len(a.data.Transcript))
	copy(transcriptSnapshot, a.data.Transcript)
	a.dataMutex.RUnlock()

	if len(transcriptSnapshot) == 0 {
		return
	}

	logrus.Infof("Updating analysis for agent %s with %d total transcript entries", a.agentID, len(transcriptSnapshot))

	// Store the snapshot temporarily for use by analysis functions
	// We'll modify the analysis functions to use this snapshot instead of calling getRecentTranscript
	a.currentAnalysisSnapshot = transcriptSnapshot

	// Generate summary
	if err := a.generateSummary(); err != nil {
		logrus.Errorf("Failed to generate summary for agent %s: %v", a.agentID, err)
	}

	// Extract key points
	if err := a.extractKeyPoints(); err != nil {
		logrus.Errorf("Failed to extract key points for agent %s: %v", a.agentID, err)
	}

	// Identify action items
	if err := a.identifyActionItems(); err != nil {
		logrus.Errorf("Failed to identify action items for agent %s: %v", a.agentID, err)
	}

	// Extract topics
	if err := a.extractTopics(); err != nil {
		logrus.Errorf("Failed to extract topics for agent %s: %v", a.agentID, err)
	}

	// Analyze sentiment and extract keywords
	if err := a.analyzeSentimentAndKeywords(); err != nil {
		logrus.Errorf("Failed to analyze sentiment for agent %s: %v", a.agentID, err)
	}

	// Clear the snapshot
	a.currentAnalysisSnapshot = nil

	// Save the updated analysis
	a.dataMutex.Lock()
	a.data.LastUpdated = time.Now()
	a.dataMutex.Unlock()

	if err := a.saveAnalysis(); err != nil {
		logrus.Errorf("Failed to save updated analysis for agent %s: %v", a.agentID, err)
	}

	logrus.Infof("Analysis updated for agent %s", a.agentID)
}

// generateSummary creates a comprehensive meeting summary
func (a *AnalystAgent) generateSummary() error {
	// Get recent transcript (last 50 entries or all if less)
	transcript := a.getRecentTranscript(50)
	if len(transcript) == 0 {
		return nil
	}

	logrus.Infof("Agent %s: Generating summary with %d transcript entries", a.agentID, len(transcript))

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("summary",
		`Analyze this meeting transcript and provide a comprehensive summary. You MUST use the google_search tool to validate and verify any factual claims, statistics, figures, technical details, company information, or specific statements that can be fact-checked.

		Focus on:
		- Main topics discussed
		- Key decisions made  
		- Important information shared
		- Overall meeting progress and outcomes
		- Validation of any claims, facts, or figures mentioned

		IMPORTANT: For any factual statements, statistics, company data, technical specifications, or verifiable claims mentioned in the meeting:
		1. Use google_search to verify the accuracy
		2. Cross-reference multiple sources when possible
		3. Note if information cannot be verified or appears outdated
		4. Include relevant context from your search results

		Transcript:
		%s

		Provide your response in the following JSON format within a code block:
		`+"`"+`json
		{
			"summary": "Your comprehensive summary here with validated facts and verified information",
			"key_themes": ["theme1", "theme2", "theme3"]
		}
		`+"`"+``,
		a.formatTranscriptForLLM(transcript))

	// Try grounded call first if provider supports it
	if groundingProvider, ok := a.llmProvider.(llm.GroundingCapableProvider); ok {
		logrus.Infof("Agent %s: Using grounded call for summary generation", a.agentID)

		groundedResponse, err := groundingProvider.CallWithGrounding(prompt)
		if err != nil {
			logrus.Warnf("Grounded call failed for summary, falling back to regular call: %v", err)
			return err
		}

		return a.processSummaryWithGrounding(groundedResponse)
	}

	// Fallback to regular LLM call
	response, err := a.callLLM(prompt)
	if err != nil {
		logrus.Warnf("Failed to generate summary: %v", err)
		return err
	}

	if response != "" {
		// Try to parse JSON from response
		if jsonData := a.extractJSONFromResponse(response); jsonData != "" {
			var result struct {
				Summary   string   `json:"summary"`
				KeyThemes []string `json:"key_themes"`
			}
			if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
				logrus.Warnf("Failed to parse summary JSON: %v", err)
				return err
			}
			a.data.Summary = result.Summary
			logrus.Infof("Agent %s: Successfully generated summary (%d characters)",
				a.agentID, len(result.Summary))
		}
	}

	return nil
}

// extractKeyPoints identifies the most important points from the transcript
func (a *AnalystAgent) extractKeyPoints() error {
	transcript := a.getRecentTranscript(30)
	if len(transcript) == 0 {
		return nil
	}

	logrus.Infof("Agent %s: Extracting key points with %d transcript entries", a.agentID, len(transcript))

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("key_points",
		`Extract the key points from this meeting transcript. Focus on:
- Important decisions or agreements
- Critical information shared
- Action-oriented statements
- Questions that need answers
- Commitments made

Provide the most important takeaways from the discussion.

Transcript:
%s

Provide your response in the following JSON format within a code block:
`+"`"+`json
{
  "key_points": ["point1", "point2", "point3"]
}
`+"`"+``,
		a.formatTranscriptForLLM(transcript))

	// Log the transcript being sent
	formattedTranscript := a.formatTranscriptForLLM(transcript)
	logrus.Debugf("Agent %s: Sending %d characters of transcript to LLM for key points",
		a.agentID, len(formattedTranscript))

	// Try grounded call first if provider supports it
	if groundingProvider, ok := a.llmProvider.(llm.GroundingCapableProvider); ok {
		logrus.Infof("Agent %s: Using grounded call for key points extraction", a.agentID)
		groundedResponse, err := groundingProvider.CallWithGrounding(prompt)
		if err != nil {
			logrus.Warnf("Grounded call failed for key points, falling back to regular call: %v", err)
		} else {
			return a.processKeyPointsWithGrounding(groundedResponse)
		}
	}

	// Fallback to regular LLM call
	response, err := a.callLLM(prompt)
	if err != nil {
		logrus.Warnf("Failed to extract key points: %v", err)
		return err
	}

	if response != "" {
		// Try to parse JSON from response
		if jsonData := a.extractJSONFromResponse(response); jsonData != "" {
			var result struct {
				KeyPoints []string `json:"key_points"`
			}
			if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
				logrus.Warnf("Failed to parse key points JSON: %v", err)
				return err
			}

			a.data.KeyPoints = result.KeyPoints
			logrus.Infof("Agent %s: Successfully extracted %d key points",
				a.agentID, len(result.KeyPoints))
		}
	}
	return nil
}

// identifyActionItems finds actionable items in the transcript
func (a *AnalystAgent) identifyActionItems() error {
	transcript := a.getRecentTranscript(40)
	if len(transcript) == 0 {
		return nil
	}

	logrus.Infof("Agent %s: Identifying action items with %d transcript entries", a.agentID, len(transcript))

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("action_items",
		`Identify action items from this meeting transcript. Be VERY AGGRESSIVE in finding actionables - look beyond explicit tasks to identify research opportunities, follow-ups, and valuable investigations.

Look for:
- Explicit tasks that need to be completed
- Follow-ups required from discussions
- Decisions that need implementation
- Assignments given to specific people
- Deadlines mentioned
- Research opportunities mentioned or implied
- Unresolved questions that need investigation
- Topics that participants showed interest in exploring further
- Problems or challenges that need solutions
- Ideas worth developing or validating
- Market research, competitive analysis, or data gathering needs
- Technical investigations or proof-of-concepts to build
- Stakeholder consultations or expert opinions to seek
- Tools, processes, or systems to evaluate
- Industry trends or best practices to research

For discussions about personalities, roles, or team dynamics:
- Research specific methodologies, frameworks, or tools mentioned
- Investigate industry best practices for team challenges discussed
- Find case studies or examples relevant to situations discussed
- Look up experts, books, or resources that could help

EVEN IF NO EXPLICIT TASKS ARE MENTIONED, identify valuable research directions, learning opportunities, or investigative actions that would benefit the participants based on their discussions.

For each action item, identify:
- Description of what needs to be done
- Who is responsible (if mentioned, otherwise suggest who might be best suited)
- Priority level (high/medium/low) based on urgency and importance
- Due date (if mentioned, otherwise suggest reasonable timeframe)
- Type: task/research/investigation/follow-up/decision

Transcript:
%s

Provide your response in the following JSON format within a code block:
`+"`"+`json
{
  "action_items": [
    {
      "description": "Task description - be specific about what needs to be researched/done/investigated",
      "assignee": "Person name (optional) or use 'Reviewer'",
      "priority": "high (try to not use high until very necessary) / medium / low",
      "type": "task/research/investigation/follow-up/decision"
    }
  ]
}
`+"`"+``,
		a.formatTranscriptForLLM(transcript))

	// Log the transcript being sent
	formattedTranscript := a.formatTranscriptForLLM(transcript)
	logrus.Debugf("Agent %s: Sending %d characters of transcript to LLM for action items",
		a.agentID, len(formattedTranscript))

	response, err := a.callLLM(prompt)
	if err != nil {
		logrus.Warnf("Failed to identify action items: %v", err)
		return err
	}

	// Debug: log the response for analysis
	previewLen := 200
	if len(response) < previewLen {
		previewLen = len(response)
	}
	logrus.Debugf("Action items LLM response length: %d, preview: %s", len(response), response[:previewLen])

	if response != "" {
		// Try to parse JSON from response
		if jsonData := a.extractJSONFromResponse(response); jsonData != "" {
			var result struct {
				ActionItems []ActionItem `json:"action_items"`
			}
			if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
				logrus.Warnf("Failed to parse action items JSON: %v", err)
				return err
			}

			a.data.ActionItems = result.ActionItems
			logrus.Infof("Agent %s: Successfully identified %d action items",
				a.agentID, len(result.ActionItems))
		}
	}
	return nil
}

// extractTopics identifies main discussion topics
func (a *AnalystAgent) extractTopics() error {
	transcript := a.getRecentTranscript(50)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("topics",
		`Analyze this meeting transcript and identify the main discussion topics. For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s

Provide your response in the following JSON format within a code block:
`+"`"+`json
{
  "topics": [
    {
      "topic": "Topic name",
      "summary": "Brief summary of discussion",
      "participants": ["Speaker1", "Speaker2"],
      "start_time": "HH:MM",
      "duration_minutes": 15
    }
  ]
}
`+"`"+``,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLM(prompt)
	if err != nil {
		logrus.Warnf("Failed to extract topics: %v", err)
		return err
	}

	if response != "" {
		// Try to parse JSON from response
		if jsonData := a.extractJSONFromResponse(response); jsonData != "" {
			var result struct {
				Topics []TopicDiscussion `json:"topics"`
			}
			if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
				logrus.Warnf("Failed to parse topics JSON: %v", err)
				// Don't return error, just log and continue
				return nil
			}
			a.data.Topics = result.Topics
		}
	}
	return nil
}

// analyzeSentimentAndKeywords performs sentiment analysis and keyword extraction
func (a *AnalystAgent) analyzeSentimentAndKeywords() error {
	transcript := a.getRecentTranscript(20)
	if len(transcript) == 0 {
		return nil
	}

	// Use custom prompt if provided, otherwise use default
	prompt := a.buildAnalysisPrompt("sentiment_keywords",
		`Analyze the sentiment and extract keywords from this meeting transcript.

Determine the overall sentiment of the discussion and identify the most important keywords and phrases.

Transcript:
%s

Provide your response in the following JSON format within a code block:
`+"`"+`json
{
  "sentiment": "positive/negative/neutral/mixed",
  "keywords": ["keyword1", "keyword2", "keyword3"],
  "confidence": 0.85
}
`+"`"+``,
		a.formatTranscriptForLLM(transcript))

	response, err := a.callLLM(prompt)
	if err != nil {
		logrus.Warnf("Failed to perform sentiment analysis: %v", err)
		return err
	}

	if response != "" {
		// Try to parse JSON from response
		if jsonData := a.extractJSONFromResponse(response); jsonData != "" {
			var analysis struct {
				Sentiment  string   `json:"sentiment"`
				Keywords   []string `json:"keywords"`
				Confidence float64  `json:"confidence"`
			}
			if err := json.Unmarshal([]byte(jsonData), &analysis); err != nil {
				logrus.Warnf("Failed to parse sentiment & keywords analysis JSON: %v", err)
				return err
			}
			a.data.Sentiment = analysis.Sentiment
			a.data.Keywords = analysis.Keywords
		}
	}
	return nil
}

// Helper methods

// callLLM calls the LLM with a simple prompt
func (a *AnalystAgent) callLLM(prompt string) (string, error) {
	if a.llmProvider == nil || !a.llmProvider.IsAvailable() {
		return "", fmt.Errorf("LLM provider not available")
	}

	return a.llmProvider.Call(prompt)
}

// extractJSONFromResponse extracts JSON content from ```json blocks
func (a *AnalystAgent) extractJSONFromResponse(response string) string {
	// Look for ```json ... ``` blocks
	startMarker := "```json"
	endMarker := "```"

	startIdx := strings.Index(response, startMarker)
	if startIdx == -1 {
		return ""
	}

	// Move past the start marker
	startIdx += len(startMarker)

	// Find the end marker
	endIdx := strings.Index(response[startIdx:], endMarker)
	if endIdx == -1 {
		return ""
	}

	// Extract the JSON content
	jsonContent := strings.TrimSpace(response[startIdx : startIdx+endIdx])
	return jsonContent
}

// getRecentTranscript returns the last N transcript entries
func (a *AnalystAgent) getRecentTranscript(count int) []TranscriptEntry {
	// If we're in the middle of analysis, use the snapshot to ensure consistency
	if a.currentAnalysisSnapshot != nil {
		total := len(a.currentAnalysisSnapshot)
		if total == 0 {
			logrus.Debugf("Agent %s: No transcript entries available in snapshot", a.agentID)
			return []TranscriptEntry{}
		}

		start := total - count
		if start < 0 {
			start = 0
		}

		// Log the transcript range being returned from snapshot
		actualCount := total - start
		logrus.Infof("Agent %s: Using analysis snapshot - returning %d transcript entries out of %d total (requested %d)",
			a.agentID, actualCount, total, count)

		result := make([]TranscriptEntry, actualCount)
		copy(result, a.currentAnalysisSnapshot[start:])
		return result
	}

	// Otherwise, acquire read lock to prevent race conditions
	a.dataMutex.RLock()
	defer a.dataMutex.RUnlock()

	total := len(a.data.Transcript)
	if total == 0 {
		logrus.Debugf("Agent %s: No transcript entries available", a.agentID)
		return []TranscriptEntry{}
	}

	start := total - count
	if start < 0 {
		start = 0
	}

	// Log the transcript range being returned
	actualCount := total - start
	logrus.Debugf("Agent %s: Returning %d transcript entries out of %d total (requested %d)",
		a.agentID, actualCount, total, count)

	result := make([]TranscriptEntry, actualCount)
	copy(result, a.data.Transcript[start:])
	return result
}

// formatTranscriptForLLM formats transcript entries for LLM consumption
func (a *AnalystAgent) formatTranscriptForLLM(entries []TranscriptEntry) string {
	var result strings.Builder
	for _, entry := range entries {
		result.WriteString(fmt.Sprintf("[%s] %s: %s\n",
			entry.Timestamp.Format("15:04:05"),
			entry.Speaker,
			entry.Text))
	}
	return result.String()
}

// buildAnalysisPrompt builds a secure prompt for analysis using custom instructions
func (a *AnalystAgent) buildAnalysisPrompt(analysisType, defaultPrompt, transcript string) string {
	// Check if custom prompt is set - if so, use custom prompt-driven prompts
	if a.config.CustomPrompt != nil && *a.config.CustomPrompt != "" {
		return a.buildSecurePromptFromInstructions(analysisType, *a.config.CustomPrompt, transcript)
	}

	// Use default prompt if no custom instructions
	return fmt.Sprintf(defaultPrompt, transcript)
}

// buildSecurePromptFromInstructions creates task-specific prompts based on custom instructions
func (a *AnalystAgent) buildSecurePromptFromInstructions(analysisType, customInstructions, transcript string) string {
	// Get custom prompt from agent config
	customPrompt := a.config.CustomPrompt
	if customPrompt == nil || *customPrompt == "" {
		// Fall back to direct instruction insertion if no custom prompt is set
		return a.buildDirectPrompt(analysisType, customInstructions, transcript)
	}

	// Basic validation for harmful content in custom prompt
	if !a.isSafeInstruction(*customPrompt) {
		logrus.Warnf("Potentially harmful custom prompt detected, using default prompt")
		return a.getDefaultPrompt(analysisType, transcript)
	}

	// Generate task-specific prompt based on custom instructions
	taskPrompt, err := a.generateTaskPromptFromCustomInstructions(analysisType, *customPrompt)
	if err != nil {
		logrus.Warnf("Failed to generate task prompt from custom instructions, falling back to direct: %v", err)
		return a.buildDirectPrompt(analysisType, customInstructions, transcript)
	}

	// Build final prompt with generated task instructions
	var basePrompt string

	switch analysisType {
	case "summary":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and role described above, analyze this meeting transcript and provide a comprehensive summary.

Transcript:
%s`, taskPrompt, transcript)

	case "key_points":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and role described above, extract the most important key points from this meeting transcript.

Transcript:
%s`, taskPrompt, transcript)

	case "action_items":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and role described above, identify all actionable items from this meeting transcript.

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low)
- Due date (if mentioned)

Transcript:
%s`, taskPrompt, transcript)

	case "topics":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and role described above, analyze this meeting transcript and identify the main discussion topics.

For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, taskPrompt, transcript)

	case "sentiment_keywords":
		basePrompt = fmt.Sprintf(`%s

Based on your expertise and role described above, analyze the sentiment and extract important keywords from this meeting transcript.

Determine the overall sentiment and identify key themes and important terms.

Transcript:
%s`, taskPrompt, transcript)

	default:
		return a.getDefaultPrompt(analysisType, transcript)
	}

	logrus.Debugf("Built custom prompt-driven %s prompt", analysisType)
	return basePrompt
}

// buildDirectPrompt creates a prompt by directly inserting client instructions (fallback)
func (a *AnalystAgent) buildDirectPrompt(analysisType, clientInstructions, transcript string) string {
	// Basic validation for harmful content
	if !a.isSafeInstruction(clientInstructions) {
		logrus.Warnf("Potentially harmful instruction detected, using default prompt")
		return a.getDefaultPrompt(analysisType, transcript)
	}

	// Build prompt by directly inserting instructions into base template
	var basePrompt string

	switch analysisType {
	case "summary":
		basePrompt = fmt.Sprintf(`Analyze this meeting transcript and provide a comprehensive summary.

Additional Instructions: %s

Transcript:
%s`, clientInstructions, transcript)

	case "key_points":
		basePrompt = fmt.Sprintf(`Extract the most important key points from this meeting transcript.

Additional Instructions: %s

Transcript:
%s`, clientInstructions, transcript)

	case "action_items":
		basePrompt = fmt.Sprintf(`Identify all actionable items from this meeting transcript.

Additional Instructions: %s

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned)
- Priority level (high/medium/low)
- Due date (if mentioned)

Transcript:
%s`, clientInstructions, transcript)

	case "topics":
		basePrompt = fmt.Sprintf(`Analyze this meeting transcript and identify the main discussion topics.

Additional Instructions: %s

For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, clientInstructions, transcript)

	case "sentiment_keywords":
		basePrompt = fmt.Sprintf(`Analyze the sentiment and extract important keywords from this meeting transcript.

Additional Instructions: %s

Determine the overall sentiment and identify key themes and important terms.

Transcript:
%s`, clientInstructions, transcript)

	default:
		return a.getDefaultPrompt(analysisType, transcript)
	}

	logrus.Debugf("Built direct %s prompt with client instructions", analysisType)
	return basePrompt
}

// generateTaskPromptFromCustomInstructions uses LLM to generate task-specific instructions based on custom prompt
func (a *AnalystAgent) generateTaskPromptFromCustomInstructions(analysisType, customInstructions string) (string, error) {
	var taskDescription string

	switch analysisType {
	case "summary":
		taskDescription = "creating comprehensive meeting summaries"
	case "key_points":
		taskDescription = "extracting key points and important takeaways"
	case "action_items":
		taskDescription = "identifying actionable items and next steps"
	case "topics":
		taskDescription = "analyzing discussion topics and themes"
	case "sentiment_keywords":
		taskDescription = "analyzing sentiment and extracting keywords"
	default:
		taskDescription = "analyzing meeting content"
	}

	prompt := fmt.Sprintf(`Given this role description for an analyst agent:

%s

Generate specific instructions for how this agent should approach %s in meetings. Focus on their expertise, experience level, analytical style, and specific methodologies they should use. Provide clear, actionable guidance that captures their unique approach to this type of analysis.

Keep the response focused and professional, as these instructions will be used directly in LLM prompts.`, customInstructions, taskDescription)

	// Use the same LLM provider as configured for the agent
	response, err := a.llmProvider.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate task prompt: %w", err)
	}

	// Clean up the response
	taskPrompt := strings.TrimSpace(response)
	if taskPrompt == "" {
		return "", fmt.Errorf("empty task prompt generated")
	}

	return taskPrompt, nil
}

// isSafeInstruction performs basic validation for harmful content
func (a *AnalystAgent) isSafeInstruction(instructions string) bool {
	// Basic length check
	if len(instructions) > 5000 {
		return false
	}

	// Check for obviously harmful patterns
	harmfulPatterns := []string{
		"<script", "javascript:", "eval(", "function(",
		"import ", "require(", "exec(", "system(",
		"rm ", "del ", "format ", "drop table",
		"alter table", "truncate table",
	}

	instructionsLower := strings.ToLower(instructions)
	for _, pattern := range harmfulPatterns {
		if strings.Contains(instructionsLower, pattern) {
			logrus.Warnf("Potentially harmful pattern detected: %s", pattern)
			return false
		}
	}

	return true
}

// getDefaultPrompt returns the default prompt for an analysis type
func (a *AnalystAgent) getDefaultPrompt(analysisType, transcript string) string {
	switch analysisType {
	case "summary":
		return fmt.Sprintf(`Analyze this meeting transcript and provide a comprehensive summary. You MUST use the google_search tool to validate and verify any factual claims, statistics, figures, technical details, company information, or specific statements that can be fact-checked.

Focus on:
- Main topics discussed
- Key decisions made
- Important information shared
- Overall meeting progress and outcomes
- Validation of any claims, facts, or figures mentioned

IMPORTANT: For any factual statements, statistics, company data, technical specifications, or verifiable claims mentioned in the meeting:
1. Use google_search to verify the accuracy
2. Cross-reference multiple sources when possible
3. Note if information cannot be verified or appears outdated
4. Include relevant context from your search results

Transcript:
%s`, transcript)

	case "key_points":
		return fmt.Sprintf(`Extract the most important key points from this meeting transcript. Focus on:
- Important decisions or agreements
- Critical information shared
- Action-oriented statements
- Questions that need answers
- Commitments made

Transcript:
%s`, transcript)

	case "action_items":
		return fmt.Sprintf(`Identify action items from this meeting transcript. Be VERY AGGRESSIVE in finding actionables - look beyond explicit tasks to identify research opportunities, follow-ups, and valuable investigations.

Look for:
- Explicit tasks that need to be completed
- Follow-ups required from discussions
- Decisions that need implementation
- Assignments given to specific people
- Deadlines mentioned
- Research opportunities mentioned or implied
- Unresolved questions that need investigation
- Topics that participants showed interest in exploring further
- Problems or challenges that need solutions
- Ideas worth developing or validating
- Market research, competitive analysis, or data gathering needs
- Technical investigations or proof-of-concepts to build
- Stakeholder consultations or expert opinions to seek
- Tools, processes, or systems to evaluate
- Industry trends or best practices to research

For discussions about personalities, roles, or team dynamics:
- Research specific methodologies, frameworks, or tools mentioned
- Investigate industry best practices for team challenges discussed
- Find case studies or examples relevant to situations discussed
- Look up experts, books, or resources that could help

EVEN IF NO EXPLICIT TASKS ARE MENTIONED, identify valuable research directions, learning opportunities, or investigative actions that would benefit the participants based on their discussions.

For each action item, specify:
- Description of what needs to be done
- Who is responsible (if mentioned, otherwise suggest who might be best suited)
- Priority level (high/medium/low) based on urgency and importance
- Due date (if mentioned, otherwise suggest reasonable timeframe)
- Type: task/research/investigation/follow-up/decision

Transcript:
%s`, transcript)

	case "topics":
		return fmt.Sprintf(`Analyze this meeting transcript and identify the main discussion topics. For each topic, provide:
- Topic name/title
- Brief summary of what was discussed
- Key participants involved
- Approximate start time and duration

Transcript:
%s`, transcript)

	case "sentiment_keywords":
		return fmt.Sprintf(`Analyze the sentiment and extract keywords from this meeting transcript.

Determine the overall sentiment of the discussion and identify the most important keywords and phrases.

Transcript:
%s`, transcript)

	default:
		return fmt.Sprintf("Analyze this meeting transcript and provide insights.\n\nTranscript:\n%s", transcript)
	}
}

// File operations

// processSummaryWithGrounding processes a grounded response for summary generation
func (a *AnalystAgent) processSummaryWithGrounding(groundedResponse *llm.GroundedResponse) error {
	if groundedResponse == nil {
		return fmt.Errorf("grounded response is nil")
	}

	// Try to parse JSON from response
	if jsonData := a.extractJSONFromResponse(groundedResponse.Text); jsonData != "" {
		var result struct {
			Summary   string   `json:"summary"`
			KeyThemes []string `json:"key_themes"`
		}
		if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
			logrus.Warnf("Failed to parse grounded summary JSON: %v", err)
			return err
		}

		// Store regular summary
		a.data.Summary = result.Summary

		// Create grounded content with citations
		groundedContent := &GroundedContent{
			Text:              result.Summary,
			GroundingMetadata: groundedResponse.GroundingMetadata,
		}

		// Add citations to the text
		if groundedResponse.GroundingMetadata != nil {
			groundedContent.TextWithCitations = a.addCitations(result.Summary, groundedResponse.GroundingMetadata)
		} else {
			groundedContent.TextWithCitations = result.Summary
		}

		a.data.GroundedSummary = groundedContent

		logrus.Infof("Agent %s: Successfully generated grounded summary (%d characters, %d grounding chunks)",
			a.agentID, len(result.Summary), len(groundedResponse.GroundingMetadata.GroundingChunks))
	}
	return nil
}

// processKeyPointsWithGrounding processes a grounded response for key points extraction
func (a *AnalystAgent) processKeyPointsWithGrounding(groundedResponse *llm.GroundedResponse) error {
	if groundedResponse == nil {
		return fmt.Errorf("grounded response is nil")
	}

	// Try to parse JSON from response
	if jsonData := a.extractJSONFromResponse(groundedResponse.Text); jsonData != "" {
		var result struct {
			KeyPoints []string `json:"key_points"`
		}
		if err := json.Unmarshal([]byte(jsonData), &result); err != nil {
			logrus.Warnf("Failed to parse grounded key points JSON: %v", err)
			return err
		}

		// Store regular key points
		a.data.KeyPoints = result.KeyPoints

		// Create grounded content with citations
		keyPointsText := strings.Join(result.KeyPoints, "\n• ")
		if keyPointsText != "" {
			keyPointsText = "• " + keyPointsText
		}

		groundedContent := &GroundedContent{
			Text:              keyPointsText,
			GroundingMetadata: groundedResponse.GroundingMetadata,
		}

		// Add citations to the text
		if groundedResponse.GroundingMetadata != nil {
			groundedContent.TextWithCitations = a.addCitations(keyPointsText, groundedResponse.GroundingMetadata)
		} else {
			groundedContent.TextWithCitations = keyPointsText
		}

		a.data.GroundedKeyPoints = groundedContent

		logrus.Infof("Agent %s: Successfully generated grounded key points (%d points, %d grounding chunks)",
			a.agentID, len(result.KeyPoints), len(groundedResponse.GroundingMetadata.GroundingChunks))
	}
	return nil
}

// addCitations adds citation links to text based on grounding metadata
func (a *AnalystAgent) addCitations(text string, groundingMetadata *llm.GroundingMetadata) string {
	if groundingMetadata == nil || len(groundingMetadata.GroundingSupports) == 0 {
		return text
	}

	result := text

	// Sort supports by end_index in descending order to avoid shifting issues when inserting
	supports := make([]llm.GroundingSupport, len(groundingMetadata.GroundingSupports))
	copy(supports, groundingMetadata.GroundingSupports)

	// Simple bubble sort for descending order by EndIndex
	for i := 0; i < len(supports); i++ {
		for j := 0; j < len(supports)-1-i; j++ {
			if supports[j].Segment.EndIndex < supports[j+1].Segment.EndIndex {
				supports[j], supports[j+1] = supports[j+1], supports[j]
			}
		}
	}

	for _, support := range supports {
		endIndex := support.Segment.EndIndex
		if len(support.GroundingChunkIndices) > 0 && endIndex <= len(result) {
			// Create citation string like [1](link1), [2](link2)
			var citationLinks []string
			for _, i := range support.GroundingChunkIndices {
				if i < len(groundingMetadata.GroundingChunks) {
					uri := groundingMetadata.GroundingChunks[i].Web.URI
					title := groundingMetadata.GroundingChunks[i].Web.Title
					citationLinks = append(citationLinks, fmt.Sprintf("[%d](%s)", i+1, uri))
					logrus.Debugf("Agent %s: Adding citation [%d] %s -> %s", a.agentID, i+1, title, uri)
				}
			}

			if len(citationLinks) > 0 {
				citationString := " " + strings.Join(citationLinks, ", ")
				result = result[:endIndex] + citationString + result[endIndex:]
			}
		}
	}

	return result
}

// File operations

// saveAnalysis saves the analysis data to file
func (a *AnalystAgent) saveAnalysis() error {
	data, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analysis data: %w", err)
	}

	return os.WriteFile(a.filePath, data, 0644)
}

// loadAnalysis loads analysis data from file
func (a *AnalystAgent) loadAnalysis() error {
	if _, err := os.Stat(a.filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, will create new
	}

	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return fmt.Errorf("failed to read analysis file: %w", err)
	}

	return json.Unmarshal(data, a.data)
}

// GetAnalysis returns a copy of the current analysis data
func (a *AnalystAgent) GetAnalysis() *AnalysisData {
	a.dataMutex.RLock()
	defer a.dataMutex.RUnlock()

	// Create a deep copy
	dataCopy := *a.data
	dataCopy.Transcript = make([]TranscriptEntry, len(a.data.Transcript))
	copy(dataCopy.Transcript, a.data.Transcript)

	dataCopy.KeyPoints = make([]string, len(a.data.KeyPoints))
	copy(dataCopy.KeyPoints, a.data.KeyPoints)

	dataCopy.ActionItems = make([]ActionItem, len(a.data.ActionItems))
	copy(dataCopy.ActionItems, a.data.ActionItems)

	dataCopy.Topics = make([]TopicDiscussion, len(a.data.Topics))
	copy(dataCopy.Topics, a.data.Topics)

	dataCopy.Participants = make([]string, len(a.data.Participants))
	copy(dataCopy.Participants, a.data.Participants)

	dataCopy.Keywords = make([]string, len(a.data.Keywords))
	copy(dataCopy.Keywords, a.data.Keywords)

	return &dataCopy
}

// GetFormattedAnalysis returns the analysis in a nicely formatted text format
func (a *AnalystAgent) GetFormattedAnalysis() string {
	data := a.GetAnalysis()

	var result strings.Builder

	result.WriteString("# Meeting Analysis Report\n\n")
	result.WriteString(fmt.Sprintf("**Meeting URL:** %s\n", data.MeetingURL))
	result.WriteString(fmt.Sprintf("**Start Time:** %s\n", data.StartTime.Format("2006-01-02 15:04:05")))
	result.WriteString(fmt.Sprintf("**Last Updated:** %s\n", data.LastUpdated.Format("2006-01-02 15:04:05")))
	result.WriteString(fmt.Sprintf("**Duration:** %.1f minutes\n", data.DurationMinutes))
	result.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(data.Participants, ", ")))
	result.WriteString(fmt.Sprintf("**Total Words:** %d\n", data.WordCount))
	if data.Sentiment != "" {
		result.WriteString(fmt.Sprintf("**Overall Sentiment:** %s\n", data.Sentiment))
	}
	result.WriteString("\n")

	if data.Summary != "" {
		result.WriteString("## Summary\n\n")
		result.WriteString(data.Summary)
		result.WriteString("\n\n")
	}

	if len(data.KeyPoints) > 0 {
		result.WriteString("## Key Points\n\n")
		for i, point := range data.KeyPoints {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, point))
		}
		result.WriteString("\n")
	}

	if len(data.ActionItems) > 0 {
		result.WriteString("## Action Items\n\n")
		for _, item := range data.ActionItems {
			result.WriteString(fmt.Sprintf("- **%s** (%s priority)", item.Description, item.Priority))
			if item.Type != "" {
				result.WriteString(fmt.Sprintf(" - Type: %s", item.Type))
			}
			if item.Assignee != "" {
				result.WriteString(fmt.Sprintf(" - Assigned to: %s", item.Assignee))
			}
			result.WriteString(fmt.Sprintf(" - Status: %s\n", item.Status))
		}
		result.WriteString("\n")
	}

	if len(data.Topics) > 0 {
		result.WriteString("## Discussion Topics\n\n")
		for _, topic := range data.Topics {
			result.WriteString(fmt.Sprintf("### %s\n", topic.Topic))
			result.WriteString(fmt.Sprintf("**Duration:** %.1f minutes\n", topic.Duration))
			result.WriteString(fmt.Sprintf("**Participants:** %s\n", strings.Join(topic.Participants, ", ")))
			result.WriteString(fmt.Sprintf("**Summary:** %s\n\n", topic.Summary))
		}
	}

	if len(data.Keywords) > 0 {
		result.WriteString("## Keywords\n\n")
		result.WriteString(strings.Join(data.Keywords, ", "))
		result.WriteString("\n\n")
	}

	if len(data.Transcript) > 0 {
		result.WriteString("## Full Transcript\n\n")
		for _, entry := range data.Transcript {
			result.WriteString(fmt.Sprintf("[%s] **%s:** %s\n\n",
				entry.Timestamp.Format("15:04:05"),
				entry.Speaker,
				entry.Text))
		}
	}

	return result.String()
}
