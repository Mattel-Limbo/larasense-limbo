package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the full application configuration.
type Config struct {
	Provider ProviderConfig `mapstructure:"provider"`
	Review   ReviewConfig   `mapstructure:"review"`
	Filters  FilterConfig   `mapstructure:"filters"`
}

// ProviderConfig holds AI provider settings.
type ProviderConfig struct {
	Name        string  `mapstructure:"name"`        // Provider preset: "openai", "anthropic", "gemini", "ollama", or empty for custom
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	Endpoint    string  `mapstructure:"endpoint"`    // "chat" or "responses" (default: "chat")
	MaxTokens   int     `mapstructure:"max_tokens"`  // Max completion tokens (default: 1024, 0 = no limit)
	Temperature float64 `mapstructure:"temperature"` // Sampling temperature (default: 0.0 = fully deterministic)
	Seed        int     `mapstructure:"seed"`        // Fixed seed for reproducible results (default: 42, 0 = random)
}

// providerPreset holds default settings for known AI providers.
type providerPreset struct {
	BaseURL  string
	Model    string
	Endpoint string
}

// providerPresets maps provider names to their default configurations.
var providerPresets = map[string]providerPreset{
	"openai": {
		BaseURL:  "https://api.openai.com",
		Model:    "gpt-4.1",
		Endpoint: "chat",
	},
	"anthropic": {
		BaseURL:  "https://api.anthropic.com",
		Model:    "claude-sonnet-4-20250514",
		Endpoint: "chat",
	},
	"gemini": {
		BaseURL:  "https://generativelanguage.googleapis.com/v1beta/openai",
		Model:    "gemini-2.5-flash",
		Endpoint: "chat",
	},
	"ollama": {
		BaseURL:  "http://localhost:11434",
		Model:    "llama3.1",
		Endpoint: "chat",
	},
	"openrouter": {
		BaseURL:  "https://openrouter.ai/api",
		Model:    "openai/gpt-4.1",
		Endpoint: "chat",
	},
}

// ReviewConfig holds review behavior settings.
type ReviewConfig struct {
	MaxIssues         int    `mapstructure:"max_issues"`
	SeverityThreshold string `mapstructure:"severity_threshold"`
	CustomPrompt      string `mapstructure:"custom_prompt"`
	ContextLines      int    `mapstructure:"context_lines"` // Surrounding context lines (default: 10, 0 = disabled)
}

// FilterConfig holds file inclusion/exclusion patterns.
type FilterConfig struct {
	Include []string `mapstructure:"include"`
	Exclude []string `mapstructure:"exclude"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			Model:       "gpt-4.1",
			Endpoint:    "chat",
			MaxTokens:   16384,
			Temperature: 0.0,
			Seed:        42,
		},
		Review: ReviewConfig{
			MaxIssues:         5,
			SeverityThreshold: "medium",
			ContextLines:      10,
		},
		Filters: FilterConfig{
			Include: []string{
				"app/**",
				"routes/**",
				"resources/views/**",
			},
			Exclude: []string{
				"tests/**",
				"database/seeders/**",
			},
		},
	}
}

// Load reads the config file and environment variables, returning a Config.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	defaults := DefaultConfig()
	v.SetDefault("provider.model", defaults.Provider.Model)
	v.SetDefault("provider.endpoint", defaults.Provider.Endpoint)
	v.SetDefault("provider.max_tokens", defaults.Provider.MaxTokens)
	v.SetDefault("provider.temperature", defaults.Provider.Temperature)
	v.SetDefault("provider.seed", defaults.Provider.Seed)
	v.SetDefault("review.max_issues", defaults.Review.MaxIssues)
	v.SetDefault("review.severity_threshold", defaults.Review.SeverityThreshold)
	v.SetDefault("review.context_lines", defaults.Review.ContextLines)
	v.SetDefault("filters.include", defaults.Filters.Include)
	v.SetDefault("filters.exclude", defaults.Filters.Exclude)

	// Config file
	v.SetConfigName(".larasense-limbo")
	v.SetConfigType("yml")
	v.AddConfigPath(".")

	// Environment variables
	v.SetEnvPrefix("LARASENSE_LIMBO")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional — not fatal if missing)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand environment variables in api_key (supports ${VAR} syntax)
	cfg.Provider.APIKey = expandEnvVars(cfg.Provider.APIKey)

	// Apply provider preset defaults (only fills empty fields)
	if cfg.Provider.Name != "" {
		if preset, ok := providerPresets[strings.ToLower(cfg.Provider.Name)]; ok {
			if cfg.Provider.BaseURL == "" {
				cfg.Provider.BaseURL = preset.BaseURL
			}
			if cfg.Provider.Model == "" || cfg.Provider.Model == defaults.Provider.Model {
				cfg.Provider.Model = preset.Model
			}
			if cfg.Provider.Endpoint == "" || cfg.Provider.Endpoint == defaults.Provider.Endpoint {
				cfg.Provider.Endpoint = preset.Endpoint
			}
		} else {
			return nil, fmt.Errorf("unknown provider name %q (available: openai, anthropic, gemini, ollama, openrouter)", cfg.Provider.Name)
		}
	}

	// Validate required fields
	if cfg.Provider.BaseURL == "" {
		return nil, fmt.Errorf("provider.base_url is required (set in config or LARASENSE_LIMBO_PROVIDER_BASE_URL env)")
	}
	if cfg.Provider.APIKey == "" {
		// Ollama doesn't require API key
		if strings.ToLower(cfg.Provider.Name) != "ollama" {
			return nil, fmt.Errorf("provider.api_key is required (set in config or LARASENSE_LIMBO_PROVIDER_API_KEY env)")
		}
	}

	return cfg, nil
}

// expandEnvVars replaces ${VAR} patterns with their environment variable values.
func expandEnvVars(s string) string {
	return os.Expand(s, func(key string) string {
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return ""
	})
}
