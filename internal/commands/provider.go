package commands

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/willardcsoriano/oui/internal/session"
)

func handleProvider(args string, w io.Writer) (bool, error) {
	cfg := session.Cfg()
	if cfg == nil {
		fmt.Fprintln(w, "error: no config loaded")
		return true, nil
	}

	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)

	arg := strings.TrimSpace(args)

	if arg == "" {
		active := session.ActiveProvider()
		fmt.Fprintln(w, "providers:")
		for i, name := range names {
			status := "✗"
			if cfg.IsConfigured(name) {
				status = "✓"
			}
			suffix := ""
			if name == active {
				suffix = "  ← active"
			}
			pc := cfg.Providers[name]
			fmt.Fprintf(w, "  %2d. %-14s %-40s %s%s\n", i+1, name, pc.DefaultModel, status, suffix)
		}
		fmt.Fprintln(w, "\nuse /provider <number> to switch")
		return true, nil
	}

	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > len(names) {
		fmt.Fprintf(w, "invalid selection %q — enter a number between 1 and %d\n", arg, len(names))
		return true, nil
	}
	name := names[n-1]
	if !cfg.IsConfigured(name) {
		fmt.Fprintf(w, "provider %q has no API key — add it to ~/.oui/config.toml\n", name)
		return true, nil
	}
	if err := session.SetProvider(name); err != nil {
		return true, err
	}
	fmt.Fprintf(w, "switched to %s / %s\n", name, session.ActiveModel())
	return true, nil
}
