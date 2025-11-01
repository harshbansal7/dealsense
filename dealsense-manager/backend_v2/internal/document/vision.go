package document

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/vertexai/genai"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/option"
)

// VisionConfig holds Gemini Vision configuration
type VisionConfig struct {
	ProjectID       string
	Location        string // e.g., "us-central1"
	Model           string // e.g., "gemini-1.5-flash" or "gemini-1.5-pro"
	CredentialsJSON string
	UseDefaultCreds bool
}

// VisionService handles image analysis using Gemini Vision (multimodal)
type VisionService struct {
	client *genai.Client
	model  *genai.GenerativeModel
	config VisionConfig
	ctx    context.Context
}

// ImageAnalysisResult represents the result of image analysis
type ImageAnalysisResult struct {
	Description    string                 `json:"description"`
	DetectedText   string                 `json:"detected_text"`
	VisualElements []string               `json:"visual_elements"` // charts, graphs, diagrams
	Numbers        []ExtractedNumber      `json:"numbers"`         // extracted metrics
	Confidence     float32                `json:"confidence"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ExtractedNumber represents a number/metric found in an image
type ExtractedNumber struct {
	Value   string `json:"value"`
	Label   string `json:"label"`   // e.g., "Revenue", "Growth Rate"
	Context string `json:"context"` // surrounding context
	Unit    string `json:"unit"`    // e.g., "$M", "%", "users"
}

// NewVisionService creates a new Gemini Vision service
func NewVisionService(config VisionConfig) (*VisionService, error) {
	ctx := context.Background()

	// Set defaults
	if config.Model == "" {
		config.Model = "gemini-1.5-flash" // Fast and cost-effective for image analysis
	}
	if config.Location == "" {
		config.Location = "us-central1"
	}

	var client *genai.Client
	var err error

	if config.UseDefaultCreds {
		client, err = genai.NewClient(ctx, config.ProjectID, config.Location)
	} else if config.CredentialsJSON != "" {
		client, err = genai.NewClient(ctx, config.ProjectID, config.Location,
			option.WithCredentialsJSON([]byte(config.CredentialsJSON)))
	} else {
		return nil, fmt.Errorf("no credentials provided for Vision service")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	model := client.GenerativeModel(config.Model)
	
	// Configure model for detailed analysis
	model.SetTemperature(0.2) // Lower temperature for more factual, consistent outputs
	model.SetTopK(40)
	model.SetTopP(0.95)

	logrus.Infof("Gemini Vision Service initialized: %s (%s)", config.Model, config.Location)

	return &VisionService{
		client: client,
		model:  model,
		config: config,
		ctx:    ctx,
	}, nil
}

// AnalyzeImage analyzes an image and extracts comprehensive information
// imageData should be base64 encoded image data
// imageType should be the MIME type (e.g., "image/png", "image/jpeg")
func (v *VisionService) AnalyzeImage(imageData []byte, imageType string, context string) (*ImageAnalysisResult, error) {
	logrus.Debugf("Analyzing image of type %s (size: %d bytes)", imageType, len(imageData))

	// Create the prompt for comprehensive analysis
	prompt := v.buildAnalysisPrompt(context)

	// Create image part
	imgPart := genai.ImageData(imageType, imageData)

	// Generate content
	resp, err := v.model.GenerateContent(v.ctx, genai.Text(prompt), imgPart)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze image: %w", err)
	}

	// Extract response text
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	
	// Parse the structured response
	result := v.parseAnalysisResponse(responseText)
	
	logrus.Debugf("Image analysis complete: %d chars description, %d numbers extracted", 
		len(result.Description), len(result.Numbers))

	return result, nil
}

// AnalyzeChart specifically analyzes charts and graphs for data extraction
func (v *VisionService) AnalyzeChart(imageData []byte, imageType string) (*ImageAnalysisResult, error) {
	logrus.Debug("Analyzing chart/graph for data extraction")

	prompt := `Analyze this chart or graph in detail. Extract:
1. Chart type (bar, line, pie, etc.)
2. Title and axis labels
3. All visible data points and values
4. Key insights and trends
5. Any numbers, percentages, or metrics shown

Format your response as:
DESCRIPTION: [overall description]
TYPE: [chart type]
DATA: [list all data points]
INSIGHTS: [key takeaways]
NUMBERS: [all numbers with their labels and units]`

	imgPart := genai.ImageData(imageType, imageData)

	resp, err := v.model.GenerateContent(v.ctx, genai.Text(prompt), imgPart)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze chart: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no response from vision model")
	}

	responseText := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	result := v.parseAnalysisResponse(responseText)
	
	logrus.Debugf("Chart analysis complete: extracted %d numbers", len(result.Numbers))

	return result, nil
}

// AnalyzeBatch analyzes multiple images in batch for efficiency
func (v *VisionService) AnalyzeBatch(images []ImageInput) ([]*ImageAnalysisResult, error) {
	results := make([]*ImageAnalysisResult, 0, len(images))
	
	for i, img := range images {
		logrus.Debugf("Analyzing image %d/%d", i+1, len(images))
		
		result, err := v.AnalyzeImage(img.Data, img.MimeType, img.Context)
		if err != nil {
			logrus.Warnf("Failed to analyze image %d: %v", i, err)
			// Continue with other images
			results = append(results, &ImageAnalysisResult{
				Description: fmt.Sprintf("[Image analysis failed: %v]", err),
				Confidence:  0.0,
			})
			continue
		}
		
		results = append(results, result)
	}
	
	logrus.Infof("Batch analysis complete: %d/%d images analyzed successfully", len(results), len(images))
	return results, nil
}

// ImageInput represents an image to be analyzed
type ImageInput struct {
	Data     []byte
	MimeType string
	Context  string // Optional context about where this image appears
}

// buildAnalysisPrompt creates a comprehensive prompt for image analysis
func (v *VisionService) buildAnalysisPrompt(context string) string {
	basePrompt := `Analyze this image in detail and extract all relevant information. Focus on:

1. **Visual Description**: Describe what you see (charts, diagrams, photos, text, etc.)
2. **Text Content**: Extract any text visible in the image (OCR)
3. **Numbers and Metrics**: Extract all numbers, percentages, currency values, and metrics with their labels
4. **Visual Elements**: Identify charts, graphs, diagrams, tables, or other structured visual elements
5. **Business Context**: If this appears to be from a business document, identify key business metrics, KPIs, or financial data

Format your response as:
DESCRIPTION: [comprehensive description]
TEXT: [any text found in the image]
NUMBERS: [all numbers with labels, e.g., "Revenue: $5M", "Growth: 150%", "Users: 10K"]
VISUAL_ELEMENTS: [list of charts, graphs, diagrams found]
KEY_INSIGHTS: [important takeaways or patterns]`

	if context != "" {
		basePrompt += fmt.Sprintf("\n\nCONTEXT: This image appears on %s", context)
	}

	return basePrompt
}

// parseAnalysisResponse parses the structured response from Gemini
func (v *VisionService) parseAnalysisResponse(response string) *ImageAnalysisResult {
	result := &ImageAnalysisResult{
		VisualElements: []string{},
		Numbers:        []ExtractedNumber{},
		Metadata:       make(map[string]interface{}),
		Confidence:     0.8, // Default confidence
	}

	lines := strings.Split(response, "\n")
	currentSection := ""
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for section headers
		if strings.HasPrefix(line, "DESCRIPTION:") {
			currentSection = "description"
			result.Description = strings.TrimSpace(strings.TrimPrefix(line, "DESCRIPTION:"))
		} else if strings.HasPrefix(line, "TEXT:") {
			currentSection = "text"
			result.DetectedText = strings.TrimSpace(strings.TrimPrefix(line, "TEXT:"))
		} else if strings.HasPrefix(line, "NUMBERS:") {
			currentSection = "numbers"
			numbersText := strings.TrimSpace(strings.TrimPrefix(line, "NUMBERS:"))
			if numbersText != "" {
				v.parseNumbers(numbersText, result)
			}
		} else if strings.HasPrefix(line, "VISUAL_ELEMENTS:") {
			currentSection = "visual"
			elementsText := strings.TrimSpace(strings.TrimPrefix(line, "VISUAL_ELEMENTS:"))
			if elementsText != "" {
				result.VisualElements = append(result.VisualElements, elementsText)
			}
		} else if strings.HasPrefix(line, "KEY_INSIGHTS:") {
			currentSection = "insights"
			insights := strings.TrimSpace(strings.TrimPrefix(line, "KEY_INSIGHTS:"))
			result.Metadata["insights"] = insights
		} else {
			// Continue current section
			switch currentSection {
			case "description":
				result.Description += " " + line
			case "text":
				result.DetectedText += " " + line
			case "numbers":
				v.parseNumbers(line, result)
			case "visual":
				result.VisualElements = append(result.VisualElements, line)
			}
		}
	}

	// If parsing failed, use entire response as description
	if result.Description == "" {
		result.Description = response
	}

	return result
}

// parseNumbers extracts structured number information from text
func (v *VisionService) parseNumbers(text string, result *ImageAnalysisResult) {
	// Split by common delimiters
	parts := strings.Split(text, ",")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Try to parse "Label: Value" format
		if strings.Contains(part, ":") {
			colonParts := strings.SplitN(part, ":", 2)
			if len(colonParts) == 2 {
				label := strings.TrimSpace(colonParts[0])
				value := strings.TrimSpace(colonParts[1])
				
				// Extract unit if present
				unit := v.extractUnit(value)
				
				result.Numbers = append(result.Numbers, ExtractedNumber{
					Value:   value,
					Label:   label,
					Unit:    unit,
					Context: text,
				})
			}
		} else {
			// Just a number without label
			result.Numbers = append(result.Numbers, ExtractedNumber{
				Value:   part,
				Context: text,
			})
		}
	}
}

// extractUnit extracts the unit from a value string (e.g., "$5M" -> "$M", "150%" -> "%")
func (v *VisionService) extractUnit(value string) string {
	// Common units
	units := []string{"$M", "$B", "$K", "$", "%", "K", "M", "B", "users", "customers", "months", "years"}
	
	for _, unit := range units {
		if strings.Contains(value, unit) {
			return unit
		}
	}
	
	return ""
}

// Close closes the vision service client
func (v *VisionService) Close() error {
	if v.client != nil {
		return v.client.Close()
	}
	return nil
}

// ConvertImageToText is a simplified method that returns just the textual description
// Useful for quick embedding generation
func (v *VisionService) ConvertImageToText(imageData []byte, imageType string, context string) (string, error) {
	result, err := v.AnalyzeImage(imageData, imageType, context)
	if err != nil {
		return "", err
	}

	// Combine all textual information
	var textBuilder strings.Builder
	
	if result.Description != "" {
		textBuilder.WriteString(result.Description)
		textBuilder.WriteString("\n\n")
	}
	
	if result.DetectedText != "" {
		textBuilder.WriteString("Text in image: ")
		textBuilder.WriteString(result.DetectedText)
		textBuilder.WriteString("\n\n")
	}
	
	if len(result.Numbers) > 0 {
		textBuilder.WriteString("Key metrics: ")
		for i, num := range result.Numbers {
			if i > 0 {
				textBuilder.WriteString(", ")
			}
			if num.Label != "" {
				textBuilder.WriteString(fmt.Sprintf("%s: %s", num.Label, num.Value))
			} else {
				textBuilder.WriteString(num.Value)
			}
		}
		textBuilder.WriteString("\n\n")
	}
	
	if len(result.VisualElements) > 0 {
		textBuilder.WriteString("Visual elements: ")
		textBuilder.WriteString(strings.Join(result.VisualElements, ", "))
		textBuilder.WriteString("\n")
	}
	
	return textBuilder.String(), nil
}

