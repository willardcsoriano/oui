package session

import (
	"fmt"
	"sync"

	"github.com/willardcsoriano/oui/internal/config"
	"github.com/willardcsoriano/oui/internal/providers"
)

var (
	mu           sync.RWMutex
	active       providers.Provider
	providerName string
	modelName    string
	cfg          *config.Config
)

// Init sets up the session from the loaded config and initial provider.
func Init(c *config.Config, p providers.Provider) {
	mu.Lock()
	defer mu.Unlock()
	cfg = c
	active = p
	providerName = c.ActiveProvider
	modelName = c.ActiveModel
	if modelName == "" {
		if pc, ok := c.Providers[providerName]; ok {
			modelName = pc.DefaultModel
		}
	}
}

// Get returns the currently active provider.
func Get() providers.Provider {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

// Cfg returns the loaded config.
func Cfg() *config.Config {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// ActiveProvider returns the current provider name.
func ActiveProvider() string {
	mu.RLock()
	defer mu.RUnlock()
	return providerName
}

// SystemPrompt returns the user-configured system prompt (may be empty).
func SystemPrompt() string {
	mu.RLock()
	defer mu.RUnlock()
	if cfg == nil {
		return ""
	}
	return cfg.SystemPrompt
}

// ActiveModel returns the current model name.
func ActiveModel() string {
	mu.RLock()
	defer mu.RUnlock()
	return modelName
}

// SetProvider switches the active provider and resets the model to its default.
func SetProvider(name string) error {
	mu.Lock()
	defer mu.Unlock()
	pc, ok := cfg.Providers[name]
	if !ok {
		return fmt.Errorf("provider %q not found in config — add it to ~/.oui/config.toml", name)
	}
	model := pc.DefaultModel
	p, err := providers.NewFromConfig(name, model, cfg.ResolveKey(name), pc.BaseURL)
	if err != nil {
		return err
	}
	active = p
	providerName = name
	modelName = model
	return nil
}

// SetModel switches the active model within the current provider.
func SetModel(model string) error {
	mu.Lock()
	defer mu.Unlock()
	pc := cfg.Providers[providerName]
	p, err := providers.NewFromConfig(providerName, model, cfg.ResolveKey(providerName), pc.BaseURL)
	if err != nil {
		return err
	}
	active = p
	modelName = model
	return nil
}
