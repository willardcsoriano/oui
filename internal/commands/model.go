package commands

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/willardcsoriano/oui/internal/providers"
	"github.com/willardcsoriano/oui/internal/session"
)

func handleModel(args string, w io.Writer) (bool, error) {
	arg := strings.TrimSpace(args)
	activeProvider := session.ActiveProvider()
	activeModel := session.ActiveModel()
	known := providers.KnownModels(activeProvider)

	if arg == "" {
		fmt.Fprintf(w, "models for %s:\n", activeProvider)
		if len(known) == 0 {
			fmt.Fprintln(w, "  (custom provider — use /model <name> to specify a model)")
		} else {
			for i, m := range known {
				suffix := ""
				if m == activeModel {
					suffix = "  ← active"
				}
				fmt.Fprintf(w, "  %2d. %s%s\n", i+1, m, suffix)
			}
		}
		fmt.Fprintln(w, "\nuse /model <number> or /model <name> to switch")
		return true, nil
	}

	// Accept a number (from the list) or a bare model name.
	target := arg
	if n, err := strconv.Atoi(arg); err == nil {
		if n < 1 || n > len(known) {
			fmt.Fprintf(w, "invalid selection — enter 1-%d or a model name\n", len(known))
			return true, nil
		}
		target = known[n-1]
	}

	if err := session.SetModel(target); err != nil {
		return true, err
	}
	fmt.Fprintf(w, "switched to %s / %s\n", activeProvider, target)
	return true, nil
}
