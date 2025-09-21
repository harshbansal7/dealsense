package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Joinly   JoinlyConfig   `yaml:"joinly"`
	Database DatabaseConfig `yaml:"database"`
}

// ServerConfig represents the server configuration
type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	CORS         CORSConfig    `yaml:"cors"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level   string               `yaml:"level"`
	Format  string               `yaml:"format"`
	Discord DiscordWebhookConfig `yaml:"discord"`
}

// DiscordWebhookConfig holds the configuration for Discord webhooks
type DiscordWebhookConfig struct {
	InfoWebhook   string `yaml:"info_webhook"`
	WarnWebhook   string `yaml:"warn_webhook"`
	ErrorWebhook  string `yaml:"error_webhook"`
	DebugWebhook  string `yaml:"debug_webhook"`
	GeminiWebhook string `yaml:"gemini_webhook"`
	Enabled       bool   `yaml:"enabled"`
	GeminiEnabled bool   `yaml:"gemini_enabled"`
	Username      string `yaml:"username"`
}

// DiscordHook is a logrus hook for sending logs to Discord webhooks
type DiscordHook struct {
	config     DiscordWebhookConfig
	httpClient *http.Client
}

// DiscordMessage represents the payload sent to Discord webhooks
type DiscordMessage struct {
	Username  string         `json:"username,omitempty"`
	Content   string         `json:"content,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
}

// DiscordEmbed represents an embed in a Discord message
type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

// DiscordEmbedField represents a field in a Discord embed
type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

// DiscordEmbedFooter represents a footer in a Discord embed
type DiscordEmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

// NewDiscordHook creates a new Discord webhook hook
func NewDiscordHook(config DiscordWebhookConfig) *DiscordHook {
	return &DiscordHook{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Levels returns the levels this hook should fire for
func (hook *DiscordHook) Levels() []logrus.Level {
	if !hook.config.Enabled {
		return []logrus.Level{}
	}

	levels := []logrus.Level{}
	if hook.config.InfoWebhook != "" {
		levels = append(levels, logrus.InfoLevel)
	}
	if hook.config.WarnWebhook != "" {
		levels = append(levels, logrus.WarnLevel)
	}
	if hook.config.ErrorWebhook != "" {
		levels = append(levels, logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel)
	}
	if hook.config.DebugWebhook != "" {
		levels = append(levels, logrus.DebugLevel, logrus.TraceLevel)
	}

	// Add InfoLevel for Gemini logging if Gemini webhook is configured
	if hook.config.GeminiWebhook != "" && hook.config.GeminiEnabled {
		levels = append(levels, logrus.InfoLevel)
	}

	return levels
}

// Fire sends the log entry to the appropriate Discord webhook
func (hook *DiscordHook) Fire(entry *logrus.Entry) error {
	if !hook.config.Enabled {
		return nil
	}

	webhook := hook.getWebhookForLevel(entry.Level, entry)
	if webhook == "" {
		return nil // No webhook configured for this level
	}

	message := hook.createDiscordMessage(entry)
	return hook.sendToDiscord(webhook, message)
}

// getWebhookForLevel returns the appropriate webhook URL for the given log level
func (hook *DiscordHook) getWebhookForLevel(level logrus.Level, entry *logrus.Entry) string {
	// Check if this is a Gemini-specific log
	if hook.config.GeminiEnabled && hook.config.GeminiWebhook != "" {
		if message := entry.Message; strings.Contains(message, "Gemini") {
			return hook.config.GeminiWebhook
		}
	}

	// Default webhook selection by level
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return hook.config.DebugWebhook
	case logrus.InfoLevel:
		return hook.config.InfoWebhook
	case logrus.WarnLevel:
		return hook.config.WarnWebhook
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return hook.config.ErrorWebhook
	default:
		return ""
	}
}

// createDiscordMessage creates a Discord message from a logrus entry
func (hook *DiscordHook) createDiscordMessage(entry *logrus.Entry) DiscordMessage {
	color := hook.getColorForLevel(entry.Level)
	title := hook.getTitleForLevel(entry.Level)

	embed := DiscordEmbed{
		Title:       title,
		Description: entry.Message,
		Color:       color,
		Timestamp:   entry.Time.Format(time.RFC3339),
		Footer: &DiscordEmbedFooter{
			Text: "DealSense",
		},
	}

	// Add fields for any additional data
	if len(entry.Data) > 0 {
		for key, value := range entry.Data {
			// Skip internal logrus fields
			if key == "level" || key == "msg" || key == "time" {
				continue
			}

			fieldValue := fmt.Sprintf("%v", value)
			// Truncate long values
			if len(fieldValue) > 1024 {
				fieldValue = fieldValue[:1021] + "..."
			}

			embed.Fields = append(embed.Fields, DiscordEmbedField{
				Name:   strings.Title(key),
				Value:  fieldValue,
				Inline: true,
			})
		}
	}

	return DiscordMessage{
		Username: hook.config.Username,
		Embeds:   []DiscordEmbed{embed},
	}
}

// getColorForLevel returns the Discord embed color for the given log level
func (hook *DiscordHook) getColorForLevel(level logrus.Level) int {
	switch level {
	case logrus.DebugLevel, logrus.TraceLevel:
		return 0x808080 // Gray
	case logrus.InfoLevel:
		return 0x0099ff // Blue
	case logrus.WarnLevel:
		return 0xff9900 // Orange
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return 0xff0000 // Red
	default:
		return 0x000000 // Black
	}
}

// getTitleForLevel returns the title for the given log level
func (hook *DiscordHook) getTitleForLevel(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "🐛 Debug"
	case logrus.TraceLevel:
		return "🔍 Trace"
	case logrus.InfoLevel:
		return "ℹ️ Info"
	case logrus.WarnLevel:
		return "⚠️ Warning"
	case logrus.ErrorLevel:
		return "❌ Error"
	case logrus.FatalLevel:
		return "💀 Fatal"
	case logrus.PanicLevel:
		return "🚨 Panic"
	default:
		return "📝 Log"
	}
}

// sendToDiscord sends the message to the Discord webhook
func (hook *DiscordHook) sendToDiscord(webhookURL string, message DiscordMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord message: %w", err)
	}

	resp, err := hook.httpClient.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// JoinlyConfig represents the joinly-specific configuration
type JoinlyConfig struct {
	DefaultURL     string        `yaml:"default_url"`
	DefaultTimeout time.Duration `yaml:"default_timeout"`
	MaxAgents      int           `yaml:"max_agents"`
}

// DatabaseConfig represents database configuration (for future use)
type DatabaseConfig struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8001,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			CORS: CORSConfig{
				AllowedOrigins: []string{"http://localhost:3000"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
			},
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "json",
			Discord: DiscordWebhookConfig{
				Enabled:  false,
				Username: "Joinly Bot",
			},
		},
		Joinly: JoinlyConfig{
			DefaultURL:     "http://135.235.237.143:8000/mcp/",
			DefaultTimeout: 30 * time.Second,
			MaxAgents:      10,
		},
		Database: DatabaseConfig{
			Type: "memory",
			URL:  "",
		},
	}
}

// LoadConfig loads configuration from environment variables and .env files
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Load .env file from current directory first (higher priority)
	localEnvPath := ".env"
	if _, err := os.Stat(localEnvPath); err == nil {
		if err := godotenv.Load(localEnvPath); err != nil {
			logrus.Warnf("Failed to load .env file from %s: %v", localEnvPath, err)
		} else {
			logrus.Infof("Successfully loaded environment variables from %s", localEnvPath)
		}
	}

	// Load .env file from parent joinly directory if it exists (lower priority)
	joinlyEnvPath := filepath.Join("..", "..", "..", ".env")
	if _, err := os.Stat(joinlyEnvPath); err == nil {
		if err := godotenv.Load(joinlyEnvPath); err != nil {
			logrus.Warnf("Failed to load .env file from %s: %v", joinlyEnvPath, err)
		} else {
			logrus.Infof("Successfully loaded environment variables from %s", joinlyEnvPath)
		}
	}

	// Override with environment variables
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}

	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		cfg.Logging.Format = format
	}

	// Discord webhook configuration
	if os.Getenv("DISCORD_LOGGING_ENABLED") == "true" {
		cfg.Logging.Discord.Enabled = true
	}

	if os.Getenv("DISCORD_GEMINI_LOGGING_ENABLED") == "true" {
		cfg.Logging.Discord.GeminiEnabled = true
	}

	if infoWebhook := os.Getenv("DISCORD_INFO_WEBHOOK"); infoWebhook != "" {
		cfg.Logging.Discord.InfoWebhook = infoWebhook
	}

	if warnWebhook := os.Getenv("DISCORD_WARN_WEBHOOK"); warnWebhook != "" {
		cfg.Logging.Discord.WarnWebhook = warnWebhook
	}

	if errorWebhook := os.Getenv("DISCORD_ERROR_WEBHOOK"); errorWebhook != "" {
		cfg.Logging.Discord.ErrorWebhook = errorWebhook
	}

	if debugWebhook := os.Getenv("DISCORD_DEBUG_WEBHOOK"); debugWebhook != "" {
		cfg.Logging.Discord.DebugWebhook = debugWebhook
	}

	if geminiWebhook := os.Getenv("DISCORD_GEMINI_WEBHOOK"); geminiWebhook != "" {
		cfg.Logging.Discord.GeminiWebhook = geminiWebhook
	}

	if username := os.Getenv("DISCORD_BOT_USERNAME"); username != "" {
		cfg.Logging.Discord.Username = username
	}

	if url := os.Getenv("JOINLY_URL"); url != "" {
		cfg.Joinly.DefaultURL = url
	}

	if maxAgents := os.Getenv("MAX_AGENTS"); maxAgents != "" {
		if ma, err := strconv.Atoi(maxAgents); err == nil {
			cfg.Joinly.MaxAgents = ma
		}
	}

	return cfg, nil
}

// SetupLogging configures the logging system
func SetupLogging(cfg *LoggingConfig) error {
	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}

	// Setup Discord webhook hook if enabled
	if cfg.Discord.Enabled {
		discordHook := NewDiscordHook(cfg.Discord)
		logrus.AddHook(discordHook)
		logrus.Info("Discord webhook logging enabled")
	}

	return nil
}
