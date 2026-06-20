package hooks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var DefaultConfigPath = filepath.Join(os.Getenv("HOME"), ".wi", "hooks.toml")

type HookEntry struct {
	Command string `toml:"command"`
	Match   string `toml:"match"`
}

type Config struct {
	PreToolUse  []HookEntry
	PostToolUse []HookEntry
	SessionStart []string
	SessionEnd   []string
}

type rawSession struct {
	Start any `toml:"start"`
	End   any `toml:"end"`
}

type rawConfig struct {
	Session     rawSession  `toml:"session"`
	PreToolUse  []rawHook   `toml:"pre_tool_use"`
	PostToolUse []rawHook   `toml:"post_tool_use"`
}

type rawHook struct {
	Command string `toml:"command"`
	Match   string `toml:"match"`
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultConfigPath
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Config{}, nil
	}

	var raw rawConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		return Config{}, fmt.Errorf("hooks: parse %s: %w", path, err)
	}

	cfg := Config{
		SessionStart: toStringSlice(raw.Session.Start),
		SessionEnd:   toStringSlice(raw.Session.End),
	}
	for _, h := range raw.PreToolUse {
		match := h.Match
		if match == "" {
			match = "*"
		}
		cfg.PreToolUse = append(cfg.PreToolUse, HookEntry{Command: h.Command, Match: match})
	}
	for _, h := range raw.PostToolUse {
		match := h.Match
		if match == "" {
			match = "*"
		}
		cfg.PostToolUse = append(cfg.PostToolUse, HookEntry{Command: h.Command, Match: match})
	}
	return cfg, nil
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	switch s := v.(type) {
	case string:
		return []string{s}
	case []any:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}
