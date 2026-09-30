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
		writef(stderr, "%s", usage)
		return exitFailure
	}

	command, rest := args[0], args[1:]
	switch command {
	case "version":
		if len(rest) > 0 {
			return usageError(stderr, "version takes no arguments")
		}
		return runVersion(stdout)
	case "validate":
		return runValidate(rest, stdout, stderr)
	case "help", "-h", "--help":
		writef(stdout, "%s", usage)
		return exitOK
	default:
		return usageError(stderr, "unknown command %q", command)
	}
}

// usageError reports a CLI usage failure on stderr and returns the usage exit
// code (§42.1).
func usageError(stderr io.Writer, format string, args ...any) int {
	writef(stderr, "fake-jev: "+format+"\n", args...)
	writef(stderr, "%s", usage)
	return exitFailure
}

// writef writes formatted output and consumes the write error in one checked
// place: a failed write already lost the message and the caller's exit code
// does not change, but lint L1 (zero tolerance, interpretation note 7)
// forbids discarding an error-returning call at the call site.
func writef(w io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		return
	}
}

const usage = `usage: fake-jev <command> [flags]

commands:
  version          print version, build, profile, and contract information
  validate <path>  validate a configuration file without starting a server
  help             print this message

Further subcommands (serve, run, verify) land in later work items.
`
