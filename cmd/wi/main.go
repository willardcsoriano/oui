package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/willardcsoriano/wi/internal/agent"
	"github.com/willardcsoriano/wi/internal/hooks"
	"github.com/willardcsoriano/wi/internal/providers"
	"github.com/willardcsoriano/wi/internal/tui"
)

var version = "dev"

func main() {
	var modelFlag string

	root := &cobra.Command{
		Use:   "wi [prompt]",
		Short: "Provider-agnostic agentic coding shell",
		Long: `wi is a provider-agnostic agentic coding shell.

Run without arguments to enter the interactive REPL.
Pass a prompt as an argument for one-shot execution.

Provider selection is controlled by WI_PROVIDER and WI_MODEL env vars.
See https://github.com/willardcsoriano/wi for documentation.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved := providers.ResolveModel(modelFlag)
			p, err := providers.New(resolved)
			if err != nil {
				return err
			}

			hooksCfg, err := hooks.Load("")
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: hooks config error: %s\n", err)
				hooksCfg = hooks.Config{}
			}

			hooks.RunLifecycle(hooksCfg.SessionStart)
			defer hooks.RunLifecycle(hooksCfg.SessionEnd)

			if len(args) == 0 {
				return runRepl(p, hooksCfg)
			}
			return runOneshot(args[0], p, hooksCfg)
		},
	}

	root.Flags().StringVarP(&modelFlag, "model", "m", "", "Model override (e.g. anthropic/claude-opus-4-8)")

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

func runRepl(p providers.Provider, hooksCfg hooks.Config) error {
	fmt.Printf("\033[1mwi\033[0m \033[2m%s\033[0m  \033[2mctrl-c or /bye to exit\033[0m\n", version)
	return tui.RunRepl(p, hooksCfg)
}

func runOneshot(prompt string, p providers.Provider, hooksCfg hooks.Config) error {
	return agent.Run(context.Background(), prompt, p, hooksCfg, os.Stdout)
}
