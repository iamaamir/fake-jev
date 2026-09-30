package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	nethttp "net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"fake-jev/internal/config"
)

// fakeJevURLEnv is the child environment variable §15.4 always sets to the
// server base URL.
const fakeJevURLEnv = "FAKE_JEV_URL"

// runFlags holds the §15.4 flag set. run accepts only the flags the
// specification names for it, plus the set of flags that were explicitly
// supplied, so an absent flag is distinguishable from one set to its zero value
// (§12.1 precedence).
type runFlags struct {
	config string
	port   int
	urlEnv string
	set    map[string]bool
}

// runRun implements §15.4: validate the configuration, start an in-process fake
// server on an ephemeral loopback port, run the child against it, verify the
// recorded interactions, stop the server, and return the §42.2 exit result.
func runRun(args []string, stdout, stderr io.Writer) int {
	flags, command, err := parseRunFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		writef(stdout, "%s", usage)
		return exitOK
	}
	if err != nil {
		return usageError(stderr, "%s", err)
	}
	if err := validateRunFlags(flags); err != nil {
		return usageError(stderr, "%s", err)
	}
	// §15.4 step 1: configuration is validated before anything is started, so
	// an invalid configuration binds no listener and runs no child.
	cfg, err := resolveRunConfig(flags)
	if err != nil {
		writef(stderr, "fake-jev: %v\n", err)
		return exitFailure
	}
	return runCommand(cfg, flags, command, stdout, stderr)
}

// parseRunFlags splits the command line at the "--" separator and parses run's
// own flags. The child command after "--" is REQUIRED (§15.4).
func parseRunFlags(args []string) (*runFlags, []string, error) {
	flagArgs := args
	var command []string
	for index, arg := range args {
		if arg == "--" {
			flagArgs, command = args[:index], args[index+1:]
			break
		}
	}
	flags := &runFlags{set: make(map[string]bool)}
	set := flag.NewFlagSet("run", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() {}
	set.StringVar(&flags.config, "config", "", "configuration file")
	set.IntVar(&flags.port, "port", 0, "bind port")
	set.StringVar(&flags.urlEnv, "url-env", "", "additional environment variable carrying the server URL")
	if err := set.Parse(flagArgs); err != nil {
		return nil, nil, err
	}
	set.Visit(func(f *flag.Flag) { flags.set[f.Name] = true })
	if extra := set.Args(); len(extra) > 0 {
		return nil, nil, fmt.Errorf("unexpected argument %q", extra[0])
	}
	if len(command) == 0 {
		return nil, nil, errors.New("run requires a child command after --")
	}
	return flags, command, nil
}

// validateRunFlags rejects flag values the §38 configuration rules reject by
// running the same validator a file passes through, so no rule is restated
// here, and requires --url-env to name a variable (§15.4 step 5).
func validateRunFlags(flags *runFlags) error {
	if flags.set["url-env"] && flags.urlEnv == "" {
		return errors.New("url-env requires a non-empty variable name")
	}
	probe, err := defaultConfig()
	if err != nil {
		return err
	}
	if flags.set["port"] {
		probe.Server.Port = flags.port
	}
	return config.Validate(probe)
}

// resolveRunConfig validates the effective configuration and applies the §15.4
// ephemeral-port rule: run ignores the file's server.port and binds an
// OS-assigned port unless the caller explicitly passes --port, so concurrent CI
// runs never collide on a fixed port. It also forces the loopback host: §15.4
// step 2 starts the server on an available loopback port, so the file's
// server.host cannot widen the bind or leak into the child's URL the way an
// explicit serve --host may (§15.1 documents a loopback default, not a forced
// value).
func resolveRunConfig(flags *runFlags) (*config.Config, error) {
	// run reuses the serve lifecycle's §12.1 chain (flag > file > default);
	// resolveServeConfig reads only the config and port entries of the flag set,
	// so run's extra flag name does not participate.
	cfg, err := resolveServeConfig(&serveFlags{config: flags.config, port: flags.port, set: flags.set})
	if err != nil {
		return nil, err
	}
	cfg.Server.Host = config.DefaultHost
	if !flags.set["port"] {
		cfg.Server.Port = 0
	}
	return cfg, nil
}

// runCommand is §15.4 steps 2-10: the server is started before the child, the
// child's URL environment is injected, the child is waited for, verification is
// always attempted, and the server is stopped on every path.
func runCommand(cfg *config.Config, flags *runFlags, command []string, stdout, stderr io.Writer) int {
	logger := log.New(stderr, "fake-jev: ", 0)
	handle, err := startServer(cfg, logger)
	if err != nil {
		writef(stderr, "fake-jev: %v\n", err)
		return exitFailure
	}
	// §42.2: the server is stopped cleanly even when the child fails to start,
	// exits nonzero, or run is interrupted.
	defer handle.stop()

	// Signals are handled before the child exists, so a signal observed the
	// moment the child becomes observable is forwarded by the graceful path
	// instead of killing run through the default disposition (a buffered channel
	// holds it until awaitChild selects).
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	child := exec.Command(command[0], command[1:]...)
	child.Env = childEnvironment(os.Environ(), handle.URL(), flags.urlEnv)
	// The child inherits run's streams so CI sees its output (§42.5, §28). Its
	// stdin stays the null device: run never prompts and never consumes stdin
	// (§15).
	child.Stdout = stdout
	child.Stderr = stderr
	if err := child.Start(); err != nil {
		writef(stderr, "fake-jev: start %s: %v\n", command[0], err)
		return exitFailure
	}

	childCode, received := awaitChild(child, signals, cfg, stderr)
	// §42.3 governs signals only while the child runs. Once it has been reaped,
	// run must be interruptible again during the verification and shutdown
	// window, so signal notification stops and the default disposition is
	// restored.
	signal.Stop(signals)
	if serveErr := handle.serveFailure(); serveErr != nil {
		// §42.5: the mid-run server error is an operational diagnostic on
		// stderr, not a change to the §42.2 exit precedence.
		writef(stderr, "fake-jev: serve: %v\n", serveErr)
	}
	return runResult(handle, childCode, received, stderr)
}

// childEnvironment builds the child environment for §15.4 steps 4-5: the parent
// environment with FAKE_JEV_URL always set to the server base URL and, when
// --url-env names a variable, that variable set to the same URL. Both replace an
// inherited value, so a stale value exported by the caller cannot survive into
// the child.
func childEnvironment(parent []string, url, urlEnvName string) []string {
	environment := make([]string, 0, len(parent)+2)
	for _, entry := range parent {
		name, _, _ := strings.Cut(entry, "=")
		if name == fakeJevURLEnv || (urlEnvName != "" && name == urlEnvName) {
			continue
		}
		environment = append(environment, entry)
	}
	environment = append(environment, fakeJevURLEnv+"="+url)
	if urlEnvName != "" && urlEnvName != fakeJevURLEnv {
		environment = append(environment, urlEnvName+"="+url)
	}
	return environment
}

// awaitChild waits for the child, forwarding SIGINT/SIGTERM to it while it runs
// (§42.3). The returned signal is non-nil when run itself was interrupted, and
// the child is always reaped before it returns.
func awaitChild(child *exec.Cmd, signals <-chan os.Signal, cfg *config.Config, stderr io.Writer) (int, os.Signal) {
	exited := make(chan error, 1)
	go func() { exited <- child.Wait() }()

	select {
	case err := <-exited:
		return childExitCode(err), nil
	case received := <-signals:
		// §42.3 step 1: forward to the child process where the platform supports
		// sending the signal.
		writef(stderr, "fake-jev: received %s; forwarding to %s\n", received, child.Path)
		if err := child.Process.Signal(received); err != nil && !errors.Is(err, os.ErrProcessDone) {
			writef(stderr, "fake-jev: forward %s: %v\n", received, err)
		}
		timeout := time.Duration(cfg.Limits.GracefulShutdownSeconds) * time.Second
		select {
		case err := <-exited:
			return childExitCode(err), received
		case <-time.After(timeout):
			// §42.3 step 3: the child did not honor the signal within the
			// graceful shutdown timeout, so it is force terminated.
			if err := child.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
				writef(stderr, "fake-jev: force terminate %s: %v\n", child.Path, err)
			}
			return childExitCode(<-exited), received
		}
	}
}

// runResult computes the §42.2 exit status after the child exited. Verification
// is always attempted, including for a nonzero child and for an interrupted run,
// because the specification requires it before the clean shutdown.
func runResult(handle *serverHandle, childCode int, received os.Signal, stderr io.Writer) int {
	report, verifyErr := callVerification(&nethttp.Client{Timeout: verificationTimeout}, handle.URL())
	if received == nil && childCode != exitOK {
		// §42.2: name the child's nonzero exit so both failures are visible on
		// stderr when verification also fails.
		writef(stderr, "fake-jev: child exited with code %d\n", childCode)
	}
	reportVerificationFailure(handle.URL(), report, verifyErr, stderr)
	switch {
	case received != nil:
		// §42.3 step 5: an interrupted run reports the conventional
		// signal-derived status rather than the child's own code.
		return signalExitStatus(received)
	case childCode != exitOK:
		// §42.2: a nonzero child exit code is preserved verbatim, and a
		// verification failure has still been reported above.
		return childCode
	case verifyErr != nil:
		return exitFailure
	case !report.Passed:
		return exitVerify
	default:
		return exitOK
	}
}

// reportVerificationFailure writes the verification half of the two failures
// §42.2 requires to be reported when the child is nonzero and verification also
// fails.
func reportVerificationFailure(baseURL string, report verificationReport, verifyErr error, stderr io.Writer) {
	switch {
	case verifyErr != nil:
		writef(stderr, "fake-jev: verify %s: %v\n", baseURL, verifyErr)
	case !report.Passed:
		writeVerificationFailure(stderr, baseURL, report)
	}
}

// childExitCode maps a finished child to its exit code. A child terminated by a
// signal has no exit code, so it uses the conventional 128+signal status where
// the platform exposes the wait status; a status the platform does not expose
// fails closed rather than being reported as success (§42.3 step 5, §4.6).
func childExitCode(err error) int {
	if err == nil {
		return exitOK
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return exitFailure
	}
	if code, ok := signalTerminationStatus(exitErr.ProcessState); ok {
		return code
	}
	if code := exitErr.ExitCode(); code >= 0 {
		return code
	}
	return exitFailure
}

// signalTerminationStatus renders a child killed by a signal as 128+signal. It
// reports false on platforms whose wait status does not expose the signal and
// for a child that exited normally.
func signalTerminationStatus(state *os.ProcessState) (int, bool) {
	if state == nil {
		return 0, false
	}
	status, ok := state.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return 0, false
	}
	return 128 + int(status.Signal()), true
}

// signalExitStatus renders a received interrupt as the conventional
// signal-derived status 128+signal (§42.3 step 5). run subscribes only to
// os.Interrupt and syscall.SIGTERM, both of which are syscall.Signal values; an
// unexpected signal type falls back to the SIGINT convention so the status can
// never be mistaken for success.
func signalExitStatus(received os.Signal) int {
	value, ok := received.(syscall.Signal)
	if !ok {
		value = syscall.SIGINT
	}
	return 128 + int(value)
}
