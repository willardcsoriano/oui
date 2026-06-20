package providers

import (
	"fmt"
	"os"
	"strings"
)

// providerDefaults maps a bare provider name to its qualified default model.
var providerDefaults = map[string]string{
	"anthropic": "claude-sonnet-4-6",
	"openai":    "gpt-4o",
	"gemini":    "gemini-2.0-flash",
	"deepseek":  "deepseek-chat",
	"ollama":    "llama3.2",
}

// ResolveModel returns the qualified "provider/model" string by reading
// WI_PROVIDER and WI_MODEL env vars, with the same logic as the Python port:
//
//   - Explicit arg wins unconditionally.
//   - WI_MODEL containing "/" is taken as-is.
//   - WI_PROVIDER + WI_MODEL (bare) → "provider/model".
//   - WI_PROVIDER alone → provider default model.
//   - Nothing set → "anthropic/claude-sonnet-4-6".
func ResolveModel(explicit string) string {
	if explicit != "" {
		return explicit
	}
	envModel := os.Getenv("WI_MODEL")
	envProvider := os.Getenv("WI_PROVIDER")

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

// New returns a Provider for the given qualified model string.
// The model string format is "provider/model", e.g. "anthropic/claude-sonnet-4-6".
func New(qualifiedModel string) (Provider, error) {
	parts := strings.SplitN(qualifiedModel, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("provider: model must be in provider/model form, got %q", qualifiedModel)
	}
	providerName := parts[0]
	modelName := parts[1]

	switch providerName {
	case "anthropic":
		return newAnthropicProvider(modelName)
	case "openai":
		return newOpenAIProvider(modelName)
	case "gemini":
		// Gemini supports the OpenAI-compatible endpoint.
		return newGeminiProvider(modelName)
	case "deepseek":
		return newDeepSeekProvider(modelName)
	case "ollama":
		return newOllamaProvider(modelName)
	default:
		return nil, fmt.Errorf("provider: unknown provider %q", providerName)
	}
}

func newOpenAIProvider(model string) (*openAIProvider, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	return &openAIProvider{model: model, apiKey: key}, nil
}

func newGeminiProvider(model string) (*openAIProvider, error) {
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is not set")
	}
	return &openAIProvider{
		model:   model,
		apiKey:  key,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
	}, nil
}

func newDeepSeekProvider(model string) (*openAIProvider, error) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set")
	}
	return &openAIProvider{
		model:   model,
		apiKey:  key,
		baseURL: "https://api.deepseek.com/v1",
	}, nil
}

func newOllamaProvider(model string) (*openAIProvider, error) {
	base := os.Getenv("WI_OLLAMA_BASE_URL")
	if base == "" {
		base = os.Getenv("OLLAMA_API_BASE")
	}
	if base == "" {
		base = "http://localhost:11434/v1"
	}
	return &openAIProvider{model: model, apiKey: "ollama", baseURL: base}, nil
}
