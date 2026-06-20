package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ProviderConfig holds per-provider settings.
type ProviderConfig struct {
	APIKey       string `toml:"api_key"`
	BaseURL      string `toml:"base_url"`
	DefaultModel string `toml:"default_model"`
}

// Config is the full ~/.oui/config.toml structure.
type Config struct {
	ActiveProvider string                    `toml:"active_provider"`
	ActiveModel    string                    `toml:"active_model"`
	Providers      map[string]ProviderConfig `toml:"providers"`
}

// Path returns the canonical config file path.
func Path() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".oui", "config.toml")
}

// Load reads the config file, creating it with defaults if missing.
func Load() (*Config, error) {
	path := Path()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := defaults()
		_ = Save(cfg) // best-effort; proceed even if write fails
		return cfg, nil
	}
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}
	if cfg.ActiveProvider == "" {
		cfg.ActiveProvider = "anthropic"
	}
	return &cfg, nil
}

// Save writes the config to disk atomically (write to tmp, rename).
func Save(cfg *Config) error {
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("config: mkdir: %w", err)
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("config: open: %w", err)
	}
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		f.Close()
		return fmt.Errorf("config: encode: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// envKeys maps provider names to their conventional environment variable names.
var envKeys = map[string]string{
	"anthropic":  "ANTHROPIC_API_KEY",
	"openai":     "OPENAI_API_KEY",
	"gemini":     "GEMINI_API_KEY",
	"deepseek":   "DEEPSEEK_API_KEY",
	"groq":       "GROQ_API_KEY",
	"mistral":    "MISTRAL_API_KEY",
	"cohere":     "COHERE_API_KEY",
	"xai":        "XAI_API_KEY",
	"together":   "TOGETHER_API_KEY",
	"fireworks":  "FIREWORKS_API_KEY",
	"perplexity": "PERPLEXITY_API_KEY",
}

// ResolveKey returns the API key for a provider: config file first, then env var.
func (c *Config) ResolveKey(provider string) string {
	if p, ok := c.Providers[provider]; ok && p.APIKey != "" {
		return p.APIKey
	}
	if env, ok := envKeys[provider]; ok {
		return os.Getenv(env)
	}
	return ""
}

// IsConfigured reports whether a provider has a usable key (or is local-only).
func (c *Config) IsConfigured(provider string) bool {
	if provider == "ollama" {
		return true
	}
	return c.ResolveKey(provider) != ""
}

func defaults() *Config {
	return &Config{
		ActiveProvider: "anthropic",
		ActiveModel:    "",
		Providers: map[string]ProviderConfig{
			"anthropic": {
				DefaultModel: "claude-sonnet-4-6",
			},
			"openai": {
				DefaultModel: "gpt-4o",
			},
			"gemini": {
				DefaultModel: "gemini-2.0-flash",
			},
			"deepseek": {
				DefaultModel: "deepseek-chat",
			},
			"groq": {
				BaseURL:      "https://api.groq.com/openai/v1",
				DefaultModel: "llama-3.3-70b-versatile",
			},
			"mistral": {
				BaseURL:      "https://api.mistral.ai/v1",
				DefaultModel: "mistral-large-latest",
			},
			"cohere": {
				BaseURL:      "https://api.cohere.ai/compatibility/v1",
				DefaultModel: "command-r-plus",
			},
			"xai": {
				BaseURL:      "https://api.x.ai/v1",
				DefaultModel: "grok-3",
			},
			"together": {
				BaseURL:      "https://api.together.xyz/v1",
				DefaultModel: "meta-llama/Llama-3-70b-chat-hf",
			},
			"fireworks": {
				BaseURL:      "https://api.fireworks.ai/inference/v1",
				DefaultModel: "accounts/fireworks/models/firefunction-v2",
			},
			"perplexity": {
				BaseURL:      "https://api.perplexity.ai",
				DefaultModel: "llama-3.1-sonar-large-128k-online",
			},
			"ollama": {
				BaseURL:      "http://localhost:11434/v1",
				DefaultModel: "llama3.2",
			},
		},
	}
}
