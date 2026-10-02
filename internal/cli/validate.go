package cli

import (
	"fmt"
	"io"

	"fake-jev/internal/compat/jev/v1"
	"fake-jev/internal/config"
)

// runValidate implements specification §15.2: load the named configuration
// file in YAML or JSON, run every static check the shared configuration loader
// performs, then run the profile-specific fixture checks the provider-neutral
// loader must not own. It starts no server, opens no listener, and makes no
// network request (§15.2, §42.1).
func runValidate(args []string, stdout, stderr io.Writer) int {
	path, err := validatePath(args)
	if err != nil {
		return usageError(stderr, "%s", err)
	}
	cfg, err := config.LoadFile(path)
	if err != nil {
		writef(stderr, "fake-jev: %s: %v\n", path, err)
		return exitFailure
	}
	if err := validateFixtureData(cfg); err != nil {
		writef(stderr, "fake-jev: %s: %v\n", path, err)
		return exitFailure
	}
	if _, err := fmt.Fprintf(stdout, "fake-jev: %s: configuration is valid\n", path); err != nil {
		return exitFailure
	}
	return exitOK
}

// validateFixtureData runs the jev/v1 checks that §15.2 requires of every
// enabled profile's fixture data and that the provider-neutral loader cannot
// own. The compat layer owns the v1 answer rules; this composition root only
// drives the per-stub loop and names the offending stub in the diagnostic.
// validate is legitimately stricter than the serve load path, so a violation
// found here does not change what serve accepts.
func validateFixtureData(cfg *config.Config) error {
	for i := range cfg.Stubs {
		stub := &cfg.Stubs[i]
		if err := v1.ValidateFixtureAnswers(stub.When.Questions, stub.Then.Answers); err != nil {
			return fmt.Errorf("stub %q: %w", stub.ID, err)
		}
		for index, response := range stub.Then.Sequence {
			if err := v1.ValidateFixtureAnswers(stub.When.Questions, response.Answers); err != nil {
				return fmt.Errorf("stub %q sequence[%d]: %w", stub.ID, index, err)
			}
		}
	}
	return nil
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
