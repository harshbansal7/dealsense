package main

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"joinly-manager/internal/client/llm"
	"joinly-manager/internal/config"
)

func main() {
	fmt.Println("Testing Discord Webhook Integration...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging with Discord hooks
	if err := config.SetupLogging(&cfg.Logging); err != nil {
		logrus.Fatalf("Failed to setup logging: %v", err)
	}

	// Test different log levels
	logrus.WithField("test_field", "test_value").Debug("This is a debug message for Discord webhook testing")
	logrus.WithField("agent_id", "test-agent").Info("Agent started successfully")
	logrus.WithField("error_code", "E001").Warn("This is a warning message")

	// Test with structured data
	logrus.WithFields(logrus.Fields{
		"agent_id":    "test-123",
		"meeting_url": "https://meet.google.com/test",
		"error_count": 5,
	}).Error("Failed to process meeting data")

	// Test Gemini logging if enabled
	if cfg.Logging.Discord.GeminiEnabled {
		fmt.Println("Testing Gemini API logging...")

		// Create a Gemini provider and test a call
		provider := llm.NewGoogleProvider("gemini-1.5-flash")
		if provider.IsAvailable() {
			response, err := provider.Call("Test prompt for Discord logging: What is 2+2?")
			if err != nil {
				fmt.Printf("Gemini test call failed: %v\n", err)
			} else {
				fmt.Printf("Gemini test call succeeded. Response length: %d characters\n", len(response))
			}
		} else {
			fmt.Println("Gemini API not available (no API key configured)")
		}
	} else {
		fmt.Println("Gemini Discord logging is disabled")
	}

	fmt.Println("Test messages sent. Check your Discord channels!")
	fmt.Println("Configuration:")
	fmt.Printf("  Discord Enabled: %v\n", cfg.Logging.Discord.Enabled)
	fmt.Printf("  Gemini Logging: %v\n", cfg.Logging.Discord.GeminiEnabled)
	fmt.Printf("  Info Webhook: %s\n", maskWebhook(cfg.Logging.Discord.InfoWebhook))
	fmt.Printf("  Warn Webhook: %s\n", maskWebhook(cfg.Logging.Discord.WarnWebhook))
	fmt.Printf("  Error Webhook: %s\n", maskWebhook(cfg.Logging.Discord.ErrorWebhook))
	fmt.Printf("  Debug Webhook: %s\n", maskWebhook(cfg.Logging.Discord.DebugWebhook))
	fmt.Printf("  Gemini Webhook: %s\n", maskWebhook(cfg.Logging.Discord.GeminiWebhook))
}

// maskWebhook masks the webhook URL for security
func maskWebhook(webhook string) string {
	if webhook == "" {
		return "(not configured)"
	}
	if len(webhook) > 50 {
		return webhook[:30] + "..." + webhook[len(webhook)-10:]
	}
	return webhook
}
