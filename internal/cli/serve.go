package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	nethttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"fake-jev/internal/config"
	"fake-jev/internal/control"
	hosthttp "fake-jev/internal/host/http"
)

// runServe implements specification §15.1. Flag values are validated against
// the same §38 rules that gate a configuration file, the effective
// configuration is resolved by the §12.1 precedence chain, and the listener is
// only opened once parsing and configuration resolution have succeeded, so a
// usage or configuration failure never binds anything (§42.1, §4.6).
func runServe(args []string, stdout, stderr io.Writer) int {
	flags, err := parseServeFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		writef(stdout, "%s", usage)
		return exitOK
	}
	if err != nil {
		return usageError(stderr, "%s", err)
	}
	if err := validateServeFlags(flags); err != nil {
		return usageError(stderr, "%s", err)
	}
	cfg, err := resolveServeConfig(flags)
	if err != nil {
		writef(stderr, "fake-jev: %v\n", err)
		return exitFailure
	}
	return serve(cfg, flags.readyFile, stderr)
}

// serveFlags holds the §15.1 flag values plus the set of flags that were
// explicitly supplied. §12.1 precedence requires distinguishing an absent flag
// from one set to its zero value: --port 0 means "ephemeral", not "unset", and
// a config file that omits a field must not be confused with the default.
type serveFlags struct {
	host      string
	port      int
	config    string
	compat    stringList
	mode      string
	logFormat string
	readyFile string
	set       map[string]bool
}

// stringList collects a repeatable string flag while preserving every supplied
// value in supply order (§15.1 --compat).
type stringList []string

func (list *stringList) String() string { return strings.Join(*list, ",") }

func (list *stringList) Set(value string) error {
	*list = append(*list, value)
	return nil
}

// parseServeFlags parses the serve command line. The parser output is
// suppressed because usage failures are reported once, by usageError, with the
// command usage text (§42.1).
func parseServeFlags(args []string) (*serveFlags, error) {
	flags := &serveFlags{set: make(map[string]bool)}
	set := flag.NewFlagSet("serve", flag.ContinueOnError)
	set.SetOutput(io.Discard)
	set.Usage = func() {}
	set.StringVar(&flags.host, "host", "", "bind host")
	set.IntVar(&flags.port, "port", 0, "bind port")
	set.StringVar(&flags.config, "config", "", "configuration file")
	set.Var(&flags.compat, "compat", "compatibility profile (repeatable)")
	set.StringVar(&flags.mode, "mode", "", "execution mode")
	set.StringVar(&flags.logFormat, "log-format", "", "log format")
	set.StringVar(&flags.readyFile, "ready-file", "", "ready file path")
	if err := set.Parse(args); err != nil {
		return nil, err
	}
	set.Visit(func(f *flag.Flag) { flags.set[f.Name] = true })
	if extra := set.Args(); len(extra) > 0 {
		return nil, fmt.Errorf("unexpected argument %q", extra[0])
	}
	return flags, nil
}

// validateServeFlags rejects flag values the §38 configuration rules reject by
// running the same validator a file passes through, so no rule is restated
// here. These are CLI usage failures (§42.1).
func validateServeFlags(flags *serveFlags) error {
	if flags.set["log-format"] && flags.logFormat != "text" && flags.logFormat != "json" {
		return fmt.Errorf("log-format %q is not supported; use text or json", flags.logFormat)
	}
	if flags.set["ready-file"] && flags.readyFile == "" {
		return errors.New("ready-file requires a non-empty path")
	}
	probe, err := defaultConfig()
	if err != nil {
		return err
	}
	applyServeFlags(probe, flags)
	return config.Validate(probe)
}

// defaultConfig returns the §12.2 built-in defaults.
func defaultConfig() (*config.Config, error) {
	return config.Load([]byte("schemaVersion: 1\n"))
}

// resolveServeConfig builds the effective configuration by the §12.1 chain:
// built-in defaults, overridden by the configuration file, overridden by the
// explicitly supplied flags. No environment variable participates.
func resolveServeConfig(flags *serveFlags) (*config.Config, error) {
	var cfg *config.Config
	var err error
	if flags.set["config"] {
		cfg, err = config.LoadFile(flags.config)
		if err != nil {
			// Name the offending file, as validate does (§15.2), so the §42.1
			// diagnostic can be acted on without guessing which file was read.
			return nil, fmt.Errorf("%s: %w", flags.config, err)
		}
	} else {
		cfg, err = defaultConfig()
		if err != nil {
			return nil, err
		}
	}
	applyServeFlags(cfg, flags)
	if err := config.Validate(cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Host == "" {
		// §19.1/§30.8: an empty host would bind the wildcard address, so it
		// fails closed rather than exposing the server publicly.
		return nil, errors.New("server.host must not be empty")
	}
	return cfg, nil
}

// applyServeFlags overlays the explicitly supplied flags onto a configuration,
// implementing the CLI-flag tier of §12.1.
func applyServeFlags(cfg *config.Config, flags *serveFlags) {
	if flags.set["host"] {
		cfg.Server.Host = flags.host
	}
	if flags.set["port"] {
		cfg.Server.Port = flags.port
	}
	if flags.set["mode"] {
		cfg.Mode = flags.mode
	}
	if flags.set["compat"] {
		cfg.Compatibility = append([]string(nil), flags.compat...)
	}
}

// readyDocument is the exact §15.1 ready-file JSON document.
type readyDocument struct {
	URL               string `json:"url"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	PID               int    `json:"pid"`
	ControlAPIVersion string `json:"controlApiVersion"`
}

// serverHandle is one running in-process fake server: the engine-backed HTTP
// host and the start/stop operations that `serve` and `run` share so neither
// command forks the lifecycle. startServer returns only after the listener is
// bound and the control API attached, so a caller holding a handle has a server
// whose next connection is already accepted from the listen backlog (§15.1,
// §15.4 step 3).
type serverHandle struct {
	cfg      *config.Config
	logger   *log.Logger
	http     *nethttp.Server
	url      string
	port     int
	serveErr chan error
}

// startServer compiles the configuration into the engine-backed host, attaches
// the control API, and binds the configured authority. Everything that can fail
// before serving is reported here, so a caller never holds a handle for a
// server that is not listening (§42.1, §4.6).
func startServer(cfg *config.Config, logger *log.Logger) (*serverHandle, error) {
	server, err := hosthttp.NewServerFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	// The control API is attached before the listener is opened, so a bound
	// listener implies a ready control API (§15.4 step 3).
	server.AttachControl(control.NewAPI(server.Engine(), cfg))
	authority := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	listener, err := net.Listen("tcp", authority)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", authority, err)
	}
	bound, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		addressErr := fmt.Errorf("listener address %v is not a TCP address", listener.Addr())
		if closeErr := listener.Close(); closeErr != nil {
			return nil, fmt.Errorf("%v; close listener: %w", addressErr, closeErr)
		}
		return nil, addressErr
	}
	// The resolved limits.logBodyBytes the host enforces bounds every request
	// body preview the operational logger emits (§19.4, §20). The host is the
	// authority on the resolved value, so the logger does not re-derive it.
	operational := newOperationalLogger(logger, server.Limits().LogBodyBytes)
	handle := &serverHandle{
		cfg:      cfg,
		logger:   logger,
		http:     &nethttp.Server{Handler: logHandler{next: server, logger: operational}, ErrorLog: logger},
		url:      "http://" + net.JoinHostPort(cfg.Server.Host, strconv.Itoa(bound.Port)),
		port:     bound.Port,
		serveErr: make(chan error, 1),
	}
	go func() { handle.serveErr <- handle.http.Serve(listener) }()
	return handle, nil
}

// URL is the base URL of the live server. When the configuration asked for an
// ephemeral port it carries the port the listener actually bound.
func (handle *serverHandle) URL() string { return handle.url }

// Port is the port the listener actually bound.
func (handle *serverHandle) Port() int { return handle.port }

// forceClose abandons the server without draining. It is the failure path for a
// start that could not publish readiness, which must leave no listener behind
// (§42.1).
func (handle *serverHandle) forceClose() error { return handle.http.Close() }

// serveFailure reports a Serve failure that has already been recorded, without
// blocking. http.ErrServerClosed is the normal result of the graceful shutdown
// and is not a failure. It lets `run` name the server error on stderr instead of
// discarding it, while leaving the §42.2 exit precedence untouched (§42.5).
func (handle *serverHandle) serveFailure() error {
	select {
	case err := <-handle.serveErr:
		if err != nil && !errors.Is(err, nethttp.ErrServerClosed) {
			return err
		}
	default:
	}
	return nil
}

// stop shuts the server down gracefully: new accepts stop, in-flight requests
// drain for up to gracefulShutdownSeconds, and whatever remains is force closed
// (§42.3). Shutdown errors are logged rather than returned: the command's exit
// code already reports whether the work it was asked to do succeeded.
func (handle *serverHandle) stop() {
	timeout := time.Duration(handle.cfg.Limits.GracefulShutdownSeconds) * time.Second
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := handle.http.Shutdown(shutdownCtx); err != nil {
		if closeErr := handle.http.Close(); closeErr != nil {
			handle.logger.Printf("force close after %s: %v", timeout, closeErr)
		}
	}
}

// serve runs the server lifecycle: bind, publish readiness, serve both planes
// over one listener, and shut down gracefully on SIGINT/SIGTERM (§42.3).
func serve(cfg *config.Config, readyPath string, stderr io.Writer) (status int) {
	logger := log.New(stderr, "fake-jev: ", 0)
	handle, err := startServer(cfg, logger)
	if err != nil {
		logger.Printf("%v", err)
		return exitFailure
	}
	// Signal handling is installed before readiness is published, so a signal
	// observed immediately after the ready file appears is always handled by
	// the graceful path (a buffered channel holds it until the select below).
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)

	if readyPath != "" {
		document := readyDocument{
			URL:               handle.URL(),
			Host:              cfg.Server.Host,
			Port:              handle.Port(),
			PID:               os.Getpid(),
			ControlAPIVersion: ControlAPIVersion,
		}
		if err := writeReadyFile(readyPath, document); err != nil {
			// A startup failure must not disturb a pre-existing ready file
			// (§42.4) and must leave no listener behind (§42.1).
			if closeErr := handle.forceClose(); closeErr != nil {
				logger.Printf("%v; close listener: %v", err, closeErr)
			} else {
				logger.Printf("%v", err)
			}
			return exitFailure
		}
		// The published file names this process, so every exit path from here
		// on removes it: a fatal serve failure must not leave a ready document
		// naming a dead pid behind for a wrapper to wait on. §42.4 is preserved
		// because removeReadyFile still requires the parsed pid to match, so a
		// file an observer replaced survives untouched.
		defer func() {
			if err := removeReadyFile(readyPath); err != nil {
				logger.Printf("remove ready file: %v", err)
				status = exitFailure
			}
		}()
	}
	logger.Printf("listening on %s", handle.URL())

	select {
	case received := <-signals:
		logger.Printf("received %s; shutting down", received)
	case err := <-handle.serveErr:
		logger.Printf("serve: %v", err)
		return exitFailure
	}

	handle.stop()
	return exitOK
}

// writeReadyFile publishes the ready document with create-temp + atomic rename
// in the destination directory, so an observer never sees a partial document
// (§42.4). It only ever runs after the listener and the control API are ready
// (§15.1).
func writeReadyFile(path string, document readyDocument) error {
	encoded, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("encode ready file: %w", err)
	}
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".fake-jev-ready-*")
	if err != nil {
		return fmt.Errorf("create ready file in %q: %w", dir, err)
	}
	tempName := temp.Name()
	if _, err := temp.Write(encoded); err != nil {
		return errors.Join(fmt.Errorf("write ready file: %w", err), closeAndRemoveTemp(temp, tempName))
	}
	if err := temp.Close(); err != nil {
		return errors.Join(fmt.Errorf("close ready file: %w", err), removeTempFile(tempName))
	}
	if err := os.Rename(tempName, path); err != nil {
		return errors.Join(fmt.Errorf("publish ready file %q: %w", path, err), removeTempFile(tempName))
	}
	return nil
}

func closeAndRemoveTemp(temp *os.File, name string) error {
	if err := temp.Close(); err != nil {
		return errors.Join(fmt.Errorf("close temporary ready file: %w", err), removeTempFile(name))
	}
	return removeTempFile(name)
}

func removeTempFile(name string) error {
	if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove temporary ready file %q: %w", name, err)
	}
	return nil
}

// removeReadyFile deletes the ready file only while it still names this
// process (§42.4). A file whose pid cannot be parsed or differs is left
// untouched, as is an absent file (interpretation recorded by the specifier).
func removeReadyFile(path string) error {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read ready file %q: %w", path, err)
	}
	var document readyDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return nil
	}
	if document.PID != os.Getpid() {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove ready file %q: %w", path, err)
	}
	return nil
}
