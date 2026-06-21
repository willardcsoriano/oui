package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/spf13/cobra"
	"github.com/willardcsoriano/oui/internal/agent"
	"github.com/willardcsoriano/oui/internal/config"
	"github.com/willardcsoriano/oui/internal/hooks"
	"github.com/willardcsoriano/oui/internal/providers"
	"github.com/willardcsoriano/oui/internal/session"
	"github.com/willardcsoriano/oui/internal/tui"
)

var version = "dev"

func main() {
	var modelFlag string
	var profileFlag string

	root := &cobra.Command{
		Use:   "oui [prompt]",
		Short: "Provider-agnostic agentic coding shell",
		Long: `oui is a provider-agnostic agentic coding shell.

Run without arguments to enter the interactive REPL.
Pass a prompt as an argument for one-shot execution.

Config file: ~/.oui/config.toml (created on first run with all providers pre-filled).
Switch providers and models interactively with /provider and /model.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: config error: %s\n", err)
				cfg = &config.Config{
					ActiveProvider: "anthropic",
					Providers:      map[string]config.ProviderConfig{},
				}
			}

			// --model flag overrides everything; --profile selects active provider.
			if modelFlag != "" {
				p, err := providers.New(providers.ResolveModel(modelFlag))
				if err != nil {
					return err
				}
				// Patch cfg so session has the right name for display.
				parts := strings.SplitN(modelFlag, "/", 2)
				if len(parts) == 2 {
					cfg.ActiveProvider = parts[0]
					cfg.ActiveModel = parts[1]
				}
				session.Init(cfg, p)
			} else {
				if profileFlag != "" {
					cfg.ActiveProvider = profileFlag
				}
				if !cfg.IsConfigured(cfg.ActiveProvider) {
					if err := runSetup(cfg); err != nil {
						return err
					}
				}
				p, err := buildProviderFromConfig(cfg)
				if err != nil {
					return err
				}
				session.Init(cfg, p)
			}

			hooksCfg, err := hooks.Load("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: hooks config error: %s\n", err)
				hooksCfg = hooks.Config{}
			}

			hooks.RunLifecycle(hooksCfg.SessionStart)
			defer hooks.RunLifecycle(hooksCfg.SessionEnd)

			if len(args) == 0 {
				return runRepl(hooksCfg)
			}
			return runOneshot(args[0], hooksCfg)
		},
	}

	root.Flags().StringVarP(&modelFlag, "model", "m", "", "Model override, e.g. anthropic/claude-opus-4-8")
	root.Flags().StringVarP(&profileFlag, "provider", "p", "", "Provider to activate, e.g. openai")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version and exit",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("oui " + version)
		},
	}
	root.AddCommand(versionCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// buildProviderFromConfig constructs the initial provider from config.
func buildProviderFromConfig(cfg *config.Config) (providers.Provider, error) {
	name := cfg.ActiveProvider
	pc, ok := cfg.Providers[name]
	if !ok {
		// Unknown provider — fall through to env-var path.
		return providers.New(providers.ResolveModel(""))
	}
	model := cfg.ActiveModel
	if model == "" {
		model = pc.DefaultModel
	}
	return providers.NewFromConfig(name, model, cfg.ResolveKey(name), pc.BaseURL)
}

// providerOrder defines the display order for the setup wizard.
var providerOrder = []string{
	"anthropic", "openai", "gemini", "deepseek", "groq",
	"mistral", "cohere", "xai", "together", "fireworks", "perplexity", "ollama",
}

// runSetup prompts the user to pick a provider and API key when none is configured.
// It modifies cfg in place and optionally persists the choice to disk.
func runSetup(cfg *config.Config) error {
	var available []string
	for _, name := range providerOrder {
		if _, ok := cfg.Providers[name]; ok {
			available = append(available, name)
		}
	}

	fmt.Println("No API key configured. Pick a provider:")
	for i, name := range available {
		note := ""
		if name == "ollama" {
			note = "  (no key required)"
		} else if cfg.IsConfigured(name) {
			note = "  (already configured)"
		}
		fmt.Printf("  %d) %s%s\n", i+1, name, note)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("\n> ")
	line, _ := reader.ReadString('\n')
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(available) {
		return fmt.Errorf("invalid selection")
	}
	chosen := available[n-1]
	cfg.ActiveProvider = chosen

	if chosen != "ollama" {
		fmt.Printf("%s API key: ", chosen)
		key, _ := reader.ReadString('\n')
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("API key is required for %s", chosen)
		}
		pc := cfg.Providers[chosen]
		pc.APIKey = key
		cfg.Providers[chosen] = pc

		fmt.Print("Save to ~/.oui/config.toml? [Y/n]: ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			if err := config.Save(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not save config: %s\n", err)
			} else {
				fmt.Printf("saved to %s\n\n", config.Path())
			}
		}
	}

	return nil
}

func printBanner() {
	cwd, _ := os.Getwd()
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}

	const (
		bold  = "\033[1m"
		white = "\033[97m"
		dim   = "\033[2m"
		reset = "\033[0m"
	)

	art := []string{
		" ██████╗ ██╗   ██╗██╗",
		"██╔═══██╗██║   ██║██║",
		"██║   ██║██║   ██║██║",
		"██║   ██║██║   ██║██║",
		"╚██████╔╝╚██████╔╝██║",
		" ╚═════╝  ╚═════╝ ╚═╝",
	}

	fmt.Println()
	for _, line := range art {
		fmt.Println(bold + white + line + reset)
	}
	fmt.Println(dim + "  provider-agnostic agentic coding shell" + reset)
	fmt.Println()

	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || termWidth < 44 {
		termWidth = 80
	}
	innerWidth := termWidth - 4 // 2 for border chars + 1 margin each side
	if innerWidth > 72 {
		innerWidth = 72 // cap so it doesn't stretch absurdly wide
	}

	top := dim + "╭" + strings.Repeat("─", innerWidth) + "╮" + reset
	bot := dim + "╰" + strings.Repeat("─", innerWidth) + "╯" + reset

	row := func(label, value string) {
		const labelCols = 13 // "  " + 10-char padded label + " "
		if len(value) > innerWidth-labelCols {
			value = value[:innerWidth-labelCols-1] + "…"
		}
		content := fmt.Sprintf("  %-10s %s", label, value)
		pad := strings.Repeat(" ", innerWidth-len(content))
		fmt.Println(dim + "│" + reset + content + pad + dim + "│" + reset)
	}

	fmt.Println(top)
	row("provider", session.ActiveProvider())
	row("model", session.ActiveModel())
	row("version", version)
	row("cwd", cwd)
	fmt.Println(bot)
	fmt.Println()
	fmt.Println(dim + "  enter send  ·  ctrl-c exit  ·  / commands" + reset)
	fmt.Println()
}

func runRepl(hooksCfg hooks.Config) error {
	printBanner()
	return tui.RunRepl(hooksCfg)
}

func runOneshot(prompt string, hooksCfg hooks.Config) error {
	return agent.Run(context.Background(), prompt, session.Get(), hooksCfg, os.Stdout)
}
