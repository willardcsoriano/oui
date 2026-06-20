package providers

import (
	"fmt"
	"os"
	"strings"
)

// providerDefaults maps a bare provider name to its default model (env-var path).
var providerDefaults = map[string]string{
	"anthropic":  "claude-sonnet-4-6",
	"openai":     "gpt-4o",
	"gemini":     "gemini-2.0-flash",
	"deepseek":   "deepseek-chat",
	"groq":       "llama-3.3-70b-versatile",
	"mistral":    "mistral-large-latest",
	"cohere":     "command-r-plus",
	"xai":        "grok-3",
	"together":   "meta-llama/Llama-3-70b-chat-hf",
	"fireworks":  "accounts/fireworks/models/firefunction-v2",
	"perplexity": "llama-3.1-sonar-large-128k-online",
	"ollama":     "llama3.2",
}

// providerBaseURLs maps providers that need a non-default base URL.
var providerBaseURLs = map[string]string{
	"gemini":     "https://generativelanguage.googleapis.com/v1beta/openai",
	"deepseek":   "https://api.deepseek.com/v1",
	"groq":       "https://api.groq.com/openai/v1",
	"mistral":    "https://api.mistral.ai/v1",
	"cohere":     "https://api.cohere.ai/compatibility/v1",
	"xai":        "https://api.x.ai/v1",
	"together":   "https://api.together.xyz/v1",
	"fireworks":  "https://api.fireworks.ai/inference/v1",
	"perplexity": "https://api.perplexity.ai",
}

// ResolveModel returns the qualified "provider/model" string via env vars.
// Used by the --model flag path; config-driven startup goes through NewFromConfig.
func ResolveModel(explicit string) string {
	if explicit != "" {
		return explicit
	}
	envModel := os.Getenv("OUI_MODEL")
	envProvider := os.Getenv("OUI_PROVIDER")

	if envModel != "" && strings.Contains(envModel, "/") {
		return envModel
	}
	if envProvider != "" && envModel != "" {
		return envProvider + "/" + envModel
	}
	if envProvider != "" {
		if def, ok := providerDefaults[envProvider]; ok {
			return envProvider + "/" + def
		}
		return envProvider + "/" + envModel
	}
	if envModel != "" {
		return "anthropic/" + envModel
	}
	return "anthropic/claude-sonnet-4-6"
}

// New builds a Provider from a "provider/model" string using environment variables
// for authentication. Used by the --model flag override path.
func New(qualifiedModel string) (Provider, error) {
	parts := strings.SplitN(qualifiedModel, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("provider: model must be in provider/model form, got %q", qualifiedModel)
	}
	name := parts[0]
	model := parts[1]

	envKeyMap := map[string]string{
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

	apiKey := ""
	if env, ok := envKeyMap[name]; ok {
		apiKey = os.Getenv(env)
	}

	baseURL := ""
	if name == "ollama" {
		baseURL = os.Getenv("OUI_OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = os.Getenv("OLLAMA_API_BASE")
		}
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		apiKey = "ollama"
	} else if u, ok := providerBaseURLs[name]; ok {
		baseURL = u
	}

	return NewFromConfig(name, model, apiKey, baseURL)
}

// NewFromConfig builds a Provider from explicit parameters supplied by the config layer.
// Any provider not named "anthropic" is handled via the OpenAI-compatible endpoint.
func NewFromConfig(provider, model, apiKey, baseURL string) (Provider, error) {
	if provider == "anthropic" {
		if apiKey == "" {
			return nil, fmt.Errorf("anthropic: no API key — set api_key in ~/.oui/config.toml or ANTHROPIC_API_KEY")
		}
		return &anthropicProvider{model: model, apiKey: apiKey}, nil
	}

	if provider == "ollama" {
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}
		if apiKey == "" {
			apiKey = "ollama"
		}
		return &openAIProvider{model: model, apiKey: apiKey, baseURL: baseURL}, nil
	}

	// All other providers: OpenAI-compatible endpoint.
	if apiKey == "" {
		return nil, fmt.Errorf("%s: no API key — set api_key in ~/.oui/config.toml", provider)
	}
	if baseURL == "" {
		if u, ok := providerBaseURLs[provider]; ok {
			baseURL = u
		}
	}
	return &openAIProvider{model: model, apiKey: apiKey, baseURL: baseURL}, nil
}

// KnownModels returns the curated model list for a provider, or nil for unknown/custom ones.
func KnownModels(provider string) []string {
	catalog := map[string][]string{
		"anthropic": {
			"claude-sonnet-4-6",
			"claude-opus-4-8",
			"claude-haiku-4-5-20251001",
			"claude-3-5-sonnet-20241022",
			"claude-3-5-haiku-20241022",
		},
		"openai": {
			"gpt-4o",
			"gpt-4o-mini",
			"o3",
			"o4-mini",
			"gpt-4-turbo",
		},
		"gemini": {
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
			"gemini-2.5-pro",
			"gemini-1.5-pro",
			"gemini-1.5-flash",
		},
		"deepseek": {
			"deepseek-chat",
			"deepseek-reasoner",
		},
		"groq": {
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
			"gemma2-9b-it",
		},
		"mistral": {
			"mistral-large-latest",
			"mistral-medium-latest",
			"mistral-small-latest",
			"codestral-latest",
		},
		"cohere": {
			"command-r-plus",
			"command-r",
			"command-light",
		},
		"xai": {
			"grok-3",
			"grok-3-mini",
			"grok-2",
		},
		"together": {
			"meta-llama/Llama-3-70b-chat-hf",
			"meta-llama/Llama-3-8b-chat-hf",
			"mistralai/Mixtral-8x7B-Instruct-v0.1",
		},
		"fireworks": {
			"accounts/fireworks/models/firefunction-v2",
			"accounts/fireworks/models/llama-v3p1-70b-instruct",
		},
		"perplexity": {
			"llama-3.1-sonar-large-128k-online",
			"llama-3.1-sonar-small-128k-online",
		},
		"ollama": {
			"llama3.2",
			"llama3.1",
			"mistral",
			"codellama",
			"phi3",
		},
	}
	return catalog[provider]
}
