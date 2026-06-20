package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/willardcsoriano/wi/internal/agent"
	"github.com/willardcsoriano/wi/internal/config"
	"github.com/willardcsoriano/wi/internal/hooks"
	"github.com/willardcsoriano/wi/internal/providers"
	"github.com/willardcsoriano/wi/internal/session"
	"github.com/willardcsoriano/wi/internal/tui"
)

var version = "dev"

func main() {
	var modelFlag string
	var profileFlag string

	root := &cobra.Command{
		Use:   "wi [prompt]",
		Short: "Provider-agnostic agentic coding shell",
		Long: `wi is a provider-agnostic agentic coding shell.

Run without arguments to enter the interactive REPL.
Pass a prompt as an argument for one-shot execution.

Config file: ~/.wi/config.toml (created on first run with all providers pre-filled).
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
			fmt.Println("wi " + version)
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

func runRepl(hooksCfg hooks.Config) error {
	fmt.Printf("\033[1mwi\033[0m \033[2m%s\033[0m  \033[2mctrl-c or /bye to exit\033[0m\n", version)
	return tui.RunRepl(hooksCfg)
}

func runOneshot(prompt string, hooksCfg hooks.Config) error {
	return agent.Run(context.Background(), prompt, session.Get(), hooksCfg, os.Stdout)
}
