// Command cantcp-cli is the command-line client for cantcp servers: it
// listens to CAN frames, sends them and reads server statistics.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/burn-lab-dev/cantcp/internal/version"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cantcp-cli:", err)
		os.Exit(1)
	}
}

// run dispatches the client commands.
func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return errors.New("no command given")
	}
	switch args[0] {
	case "listen":
		return runListen(args[1:])
	case "send":
		return runSend(args[1:])
	case "stats":
		return runStats(args[1:])
	case "version", "--version", "-version":
		fmt.Println("cantcp-cli", version.Version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q; run \"cantcp-cli help\"", args[0])
	}
}
