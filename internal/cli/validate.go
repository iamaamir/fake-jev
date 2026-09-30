package cli

import (
	"fmt"
	"io"

	"fake-jev/internal/config"
)

// runValidate implements specification §15.2: load the named configuration
// file in YAML or JSON, run every static check the shared configuration loader
// performs, and report the outcome. It starts no server, opens no listener,
// and makes no network request; the loader is the only validation surface, so
// no rule is restated here (§15.2, §42.1).
func runValidate(args []string, stdout, stderr io.Writer) int {
	path, err := validatePath(args)
	if err != nil {
		return usageError(stderr, "%s", err)
	}
	if _, err := config.LoadFile(path); err != nil {
		writef(stderr, "fake-jev: %s: %v\n", path, err)
		return exitFailure
	}
	if _, err := fmt.Fprintf(stdout, "fake-jev: %s: configuration is valid\n", path); err != nil {
		return exitFailure
	}
	return exitOK
}

// validatePath extracts the single configuration path from the command line.
// validate defines no flags, so a flag-shaped argument is a usage failure
// rather than a new option (§15.2, §42.1). A lone "-" is a path, because
// validate never reads stdin (§15).
func validatePath(args []string) (string, error) {
	path := ""
	found := false
	for _, arg := range args {
		if len(arg) > 1 && arg[0] == '-' {
			return "", fmt.Errorf("unknown flag %q", arg)
		}
		if found {
			return "", fmt.Errorf("validate takes exactly one configuration path")
		}
		path, found = arg, true
	}
	if !found {
		return "", fmt.Errorf("validate requires a configuration path")
	}
	return path, nil
}
