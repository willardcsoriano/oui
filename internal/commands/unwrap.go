package commands

import (
	"fmt"
	"io"
	"os"
)

func handleUnwrap(_ string, w io.Writer) (bool, error) {
	content, err := os.ReadFile(handoffPath)
	if err != nil {
		fmt.Fprintln(w, "no HANDOFF.md found — run /wrap first")
		return true, nil
	}
	fmt.Fprintf(w, "%s\n", content)
	return true, nil
}
