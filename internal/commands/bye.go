package commands

import (
	"fmt"
	"io"
)

func handleBye(args string, w io.Writer) (bool, error) {
	_, _ = handleWrap(args, w)
	fmt.Fprintln(w, "bye")
	return false, nil // signal REPL exit
}
