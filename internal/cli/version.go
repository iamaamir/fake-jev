package cli

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
)

// ProductVersion is the fake-jev product version. Release builds override it
// with -ldflags "-X fake-jev/internal/cli.ProductVersion=<version>"; a
// development build reports "dev".
var ProductVersion = "dev"

// ControlAPIVersion is the control API version, versioned in the URL path
// prefix /__fake/v1 (specification §25.2).
const ControlAPIVersion = "v1"

// ConfigSchemaVersion is the configuration schema version accepted in
// configuration files as schemaVersion (specification §25.3).
const ConfigSchemaVersion = 1

// BuiltInProfiles lists the compatibility profiles compiled into this binary.
// In v1 the only concrete built-in profile is jev/v1; the input alias "jev"
// normalizes to it and is not a separate profile (specification §25.4).
func BuiltInProfiles() []string {
	return []string{"jev/v1"}
}

// runVersion writes the version report required by specification §15.5. A
// write failure is an internal operational failure and exits 2 (§42.1).
func runVersion(w io.Writer) int {
	for _, line := range versionLines() {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return exitFailure
		}
	}
	return exitOK
}

// versionLines returns the version report as one label per element, in the
// order required by specification §15.5.
func versionLines() []string {
	return []string{
		"fake-jev " + ProductVersion,
		"go: " + goVersion(),
		"build: " + buildRevision(),
		"profiles: " + strings.Join(BuiltInProfiles(), ", "),
		"control-api: " + ControlAPIVersion,
		"config-schema: " + strconv.Itoa(ConfigSchemaVersion),
	}
}

// goVersion reports the Go toolchain that built this binary. It prefers the
// toolchain recorded in the build info and falls back to the runtime version
// when the binary was built without module metadata.
func goVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.GoVersion != "" {
		return info.GoVersion
	}
	return runtime.Version()
}

// buildRevision summarizes the VCS revision the binary was built from, or
// "unknown" when the build carried no revision metadata.
func buildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	var revision string
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}

	if revision == "" {
		return "unknown"
	}
	if modified {
		return revision + " (modified)"
	}
	return revision
}
