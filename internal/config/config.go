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
	BaseURL  string `mapstructure:"base_url"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"` // "chat" or "responses" (default: "chat")
}

// ReviewConfig holds review behavior settings.
type ReviewConfig struct {
	MaxIssues         int    `mapstructure:"max_issues"`
	SeverityThreshold string `mapstructure:"severity_threshold"`
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
			Model:    "gpt-4.1",
			Endpoint: "chat",
		},
		Review: ReviewConfig{
			MaxIssues:         5,
			SeverityThreshold: "medium",
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
	v.SetDefault("review.max_issues", defaults.Review.MaxIssues)
	v.SetDefault("review.severity_threshold", defaults.Review.SeverityThreshold)
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

	// Validate required fields
	if cfg.Provider.BaseURL == "" {
		return nil, fmt.Errorf("provider.base_url is required (set in config or LARASENSE_LIMBO_PROVIDER_BASE_URL env)")
	}
	if cfg.Provider.APIKey == "" {
		return nil, fmt.Errorf("provider.api_key is required (set in config or LARASENSE_LIMBO_PROVIDER_API_KEY env)")
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
