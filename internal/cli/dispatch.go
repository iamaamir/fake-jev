// Package cli owns the fake-jev command line: argument dispatch and the
// subcommands implemented so far. It is the only package that talks to the
// process environment; subcommands receive their streams explicitly so they
// stay testable without touching the real process.
package cli

import (
	"fmt"
	"io"
)

// Process exit codes (specification §42.1).
const (
	exitOK      = 0
	exitFailure = 2
	exitVerify  = 3
)

// Run executes one fake-jev invocation and returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return exitFailure
	}

	command, rest := args[0], args[1:]
	switch command {
	case "version":
		if len(rest) > 0 {
			return usageError(stderr, "version takes no arguments")
		}
		return runVersion(stdout)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return exitOK
	default:
		return usageError(stderr, "unknown command %q", command)
	}
}

// usageError reports a CLI usage failure on stderr and returns the usage exit
// code (§42.1).
func usageError(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "fake-jev: "+format+"\n", args...)
	fmt.Fprint(stderr, usage)
	return exitFailure
}

const usage = `usage: fake-jev <command> [flags]

commands:
  version    print version, build, profile, and contract information
  help       print this message

Further subcommands (serve, run, validate, verify) land in later work items.
`
