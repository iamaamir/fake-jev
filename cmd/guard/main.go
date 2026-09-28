package main

import (
	"fmt"
	"io"
	"os"
)

const usage = "usage: guard <arch|lint|trace|fuzz|complexity|mutation>\n"

// knownChecks is the design §2 registry. An arm in handlers is added by the
// same task that adds the implementation and its verify row (interpretation
// note 4); complexity/mutation stay unimplemented until FJ-044/FJ-045.
var handlers = map[string]func(*Guards) ([]Finding, map[string]int, error){
	"arch": runArch,
	// Task 3:   "lint": runLint,
	// Task 5:  "trace": runTrace,
	// Task 6:   "fuzz": runFuzz,
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the whole CLI contract: 0 = ran with zero findings, 1 = ran with
// findings (JSON report on stdout), 2 = usage/config/tool error (message on
// stderr, stdout stays empty).
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		stderrf(stderr, "%s", usage)
		return 2
	}
	sub := args[0]
	known := false
	for _, name := range []string{"arch", "lint", "trace", "fuzz", "complexity", "mutation"} {
		if name == sub {
			known = true
			break
		}
	}
	if !known {
		stderrf(stderr, "guard: unknown check %q\n", sub)
		return 2
	}
	g, err := loadGuards(".agent/guards.json")
	if err != nil {
		stderrf(stderr, "guard: %v\n", err)
		return 2
	}
	handler, ok := handlers[sub]
	if !ok {
		stderrf(stderr, "guard: check %q not implemented yet\n", sub)
		return 2
	}
	findings, stats, err := handler(g)
	if err != nil {
		stderrf(stderr, "guard: %v\n", err)
		return 2
	}
	if err := emitReport(stdout, &Report{Check: sub, Findings: findings, Stats: stats}); err != nil {
		stderrf(stderr, "guard: emit report: %v\n", err)
		return 2
	}
	if len(findings) > 0 {
		return 1
	}
	return 0
}

// stderrf writes to stderr, consuming the write error in one checked place:
// a failed write already lost the message and run() exits 2 regardless, but
// L1's zero-tolerance rule (interpretation note 7) forbids discarding an
// error-returning call at the call site, and there is no suppression list.
func stderrf(w io.Writer, format string, a ...any) {
	if _, err := fmt.Fprintf(w, format, a...); err != nil {
		return
	}
}
