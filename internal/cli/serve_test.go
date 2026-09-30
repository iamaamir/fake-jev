package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// serveBinary is the CLI built once per test binary. Signals, exit status,
// stream separation, and a real listener are only observable from a child
// process (specifier H1), so every end-to-end case launches this binary.
var serveBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fake-jev-serve-bin-")
	if err != nil {
		writef(os.Stderr, "create temp dir: %v\n", err)
		os.Exit(1)
	}
	serveBinary = filepath.Join(dir, "fake-jev")
	build := exec.Command("go", "build", "-o", serveBinary, "fake-jev/cmd/fake-jev")
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		writef(os.Stderr, "build fake-jev: %v\n%s", err, output)
		if removeErr := os.RemoveAll(dir); removeErr != nil {
			writef(os.Stderr, "remove temp dir: %v\n", removeErr)
		}
		os.Exit(1)
	}
	code := m.Run()
	if err := os.RemoveAll(dir); err != nil {
		writef(os.Stderr, "remove temp dir: %v\n", err)
	}
	os.Exit(code)
}

// syncBuffer is a stream sink safe to read while the child is still writing:
// the os/exec copy goroutine and the test goroutine never race (-race).
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (buffer *syncBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.Write(data)
}

func (buffer *syncBuffer) String() string {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.buf.String()
}

// serveLaunch describes one child `serve` invocation.
type serveLaunch struct {
	readyPath string
	dir       string
	env       []string
	args      []string
	// keepReadyFile declares that this launch is expected to leave the ready
	// file behind: a startup failure must not disturb a pre-existing document,
	// and a shutdown must not delete one an observer replaced (§42.4). Every
	// other launch must leave no ready-file residue for the cleanup check.
	keepReadyFile bool
}

// serveChild is a running `serve` process with its captured streams and exit
// status. The readiness barrier is the ready file; the shutdown barrier is the
// exit channel.
type serveChild struct {
	cmd           *exec.Cmd
	readyPath     string
	keepReadyFile bool
	stdout        syncBuffer
	stderr        syncBuffer
	exited        chan int
	waited        bool
	code          int
}

func startServe(t *testing.T, launch serveLaunch) *serveChild {
	t.Helper()
	arguments := append([]string{"serve", "--ready-file", launch.readyPath}, launch.args...)
	cmd := exec.Command(serveBinary, arguments...)
	if launch.dir != "" {
		cmd.Dir = launch.dir
	}
	cmd.Env = append(os.Environ(), launch.env...)
	child := &serveChild{cmd: cmd, readyPath: launch.readyPath, keepReadyFile: launch.keepReadyFile, exited: make(chan int, 1)}
	cmd.Stdout = &child.stdout
	cmd.Stderr = &child.stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start serve: %v", err)
	}
	go func() { child.exited <- waitExitCode(cmd.Wait()) }()
	t.Cleanup(func() { child.stop(t) })
	return child
}

func waitExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}

func (child *serveChild) stop(t *testing.T) {
	if !child.waited {
		killed := false
		if child.cmd.Process != nil {
			err := child.cmd.Process.Kill()
			if err == nil {
				killed = true
			} else if !errors.Is(err, os.ErrProcessDone) {
				t.Errorf("kill serve process: %v", err)
			}
		}
		select {
		case <-child.exited:
			child.waited = true
		case <-time.After(10 * time.Second):
			t.Errorf("serve process did not exit after being killed")
			return
		}
		if killed {
			// Cleanup abandoned a live child, so no shutdown happened and any
			// residue is a consequence of the failure, not a separate
			// observation.
			return
		}
	}
	child.assertReadyFileGone(t)
}

// assertReadyFileGone is the H6 lifecycle check: a serve child that exited on
// its own must leave no ready file behind, unless the launch declared the file
// as expected to survive.
func (child *serveChild) assertReadyFileGone(t *testing.T) {
	t.Helper()
	if child.keepReadyFile || child.readyPath == "" {
		return
	}
	if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ready file %s survives serve shutdown (stat error: %v)", child.readyPath, err)
	}
}

func (child *serveChild) wait(t *testing.T, timeout time.Duration) int {
	t.Helper()
	if child.waited {
		return child.code
	}
	select {
	case code := <-child.exited:
		child.waited, child.code = true, code
		return code
	case <-time.After(timeout):
		t.Fatalf("serve process did not exit within %s (stderr: %s)", timeout, child.stderr.String())
		return -1
	}
}

// exitedAlready reports a consumed exit without blocking. It is used for the
// bounded negative observation "still draining after the signal".
func (child *serveChild) exitedAlready() (int, bool) {
	if child.waited {
		return child.code, true
	}
	select {
	case code := <-child.exited:
		child.waited, child.code = true, code
		return code, true
	default:
		return 0, false
	}
}

func (child *serveChild) signal(t *testing.T, sig os.Signal) {
	t.Helper()
	if err := child.cmd.Process.Signal(sig); err != nil {
		t.Fatalf("signal %s: %v", sig, err)
	}
}

func (child *serveChild) waitReady(t *testing.T) readyDocument {
	t.Helper()
	var samples []string
	document, err := awaitReadyShape(child.readyPath, 10*time.Second, &samples)
	if err != nil {
		t.Fatalf("ready file: %v (stderr: %s)", err, child.stderr.String())
	}
	return document
}

// runServeChild runs a short-lived child (usage/config/startup failures) and
// returns its captured streams and exit code.
func runServeChild(t *testing.T, launch serveLaunch) (string, string, int) {
	t.Helper()
	child := startServe(t, launch)
	code := child.wait(t, 10*time.Second)
	return child.stdout.String(), child.stderr.String(), code
}

var readyDocumentKeys = []string{"url", "host", "port", "pid", "controlApiVersion"}

// awaitReadyShape polls until the ready file parses as the exact five-key
// document, recording every whole-file sample read on the way. It is the
// readiness barrier and the atomicity observer in one pass (§15.1, §42.4).
func awaitReadyShape(path string, timeout time.Duration, samples *[]string) (readyDocument, error) {
	deadline := time.Now().Add(timeout)
	for {
		data, err := os.ReadFile(path)
		if err == nil {
			*samples = append(*samples, string(data))
			document, parseErr := decodeReadyDocument(data)
			if parseErr == nil {
				return document, nil
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return readyDocument{}, err
		}
		if time.Now().After(deadline) {
			return readyDocument{}, fmt.Errorf("ready file %s did not reach the ready document within %s", path, timeout)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func decodeReadyDocument(data []byte) (readyDocument, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return readyDocument{}, fmt.Errorf("ready file is not a JSON object: %w", err)
	}
	if len(fields) != len(readyDocumentKeys) {
		return readyDocument{}, fmt.Errorf("ready file has %d keys, want %d: %s", len(fields), len(readyDocumentKeys), data)
	}
	for _, key := range readyDocumentKeys {
		if _, ok := fields[key]; !ok {
			return readyDocument{}, fmt.Errorf("ready file is missing %q: %s", key, data)
		}
	}
	var document readyDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return readyDocument{}, err
	}
	return document, nil
}

func authorityOf(t *testing.T, document readyDocument) string {
	t.Helper()
	return net.JoinHostPort(document.Host, strconv.Itoa(document.Port))
}

func serveRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string) (int, []byte) {
	t.Helper()
	request, err := nethttp.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	client := &nethttp.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	data, readErr := io.ReadAll(response.Body)
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Fatalf("close response body: %v", closeErr)
	}
	if readErr != nil {
		t.Fatalf("read response body: %v", readErr)
	}
	return response.StatusCode, data
}

// holdRequest opens a raw connection and sends the request head with
// Expect: 100-continue. Reading the interim response proves the host is
// actively reading the request body, which is the deterministic drain
// synchronization point (specifier S8b/S8c, H2).
func holdRequest(t *testing.T, authority string, declared int) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", authority, 10*time.Second)
	if err != nil {
		t.Fatalf("dial %s: %v", authority, err)
	}
	reader := bufio.NewReader(conn)
	head := "POST /v1/systemone HTTP/1.1\r\nHost: " + authority +
		"\r\nContent-Length: " + strconv.Itoa(declared) +
		"\r\nExpect: 100-continue\r\nContent-Type: application/json\r\n\r\n"
	if _, err := io.WriteString(conn, head); err != nil {
		t.Fatalf("write request head: %v", err)
	}
	statusLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read interim status: %v", err)
	}
	if !strings.Contains(statusLine, "100") {
		t.Fatalf("interim status = %q, want 100 Continue", statusLine)
	}
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("read interim terminator: %v", err)
	}
	if err := conn.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatalf("set connection deadline: %v", err)
	}
	return conn, reader
}

func awaitDialRefused(t *testing.T, authority string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", authority, 500*time.Millisecond)
		if err != nil {
			return
		}
		if err := conn.Close(); err != nil {
			t.Fatalf("close probe connection: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("listener %s still accepting connections after %s", authority, timeout)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func writeConfigFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// alternativeLoopbackHost returns a bindable loopback host that differs from
// the default 127.0.0.1, so the host-precedence rows are observable. On hosts
// where the /8 aliases are unavailable it falls back to localhost; if neither
// binds, the sub-check is reported as not exercised (specifier H7).
func alternativeLoopbackHost(t *testing.T) string {
	t.Helper()
	for _, host := range []string{"127.0.0.2", "localhost"} {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			continue
		}
		if err := listener.Close(); err != nil {
			t.Fatalf("close probe listener: %v", err)
		}
		return host
	}
	t.Skip("no alternative loopback host is bindable in this environment")
	return ""
}

// TestServeUsageFailures pins C-CLI-001 and the S1 usage-failure contract:
// every malformed command line exits 2, writes the diagnostic and usage on
// stderr, writes nothing on stdout, and creates no ready file. These cases
// return before any bind attempt, so they are exercised in-process.
func TestServeUsageFailures(t *testing.T) {
	readyPath := filepath.Join(t.TempDir(), "ready.json")
	cases := []struct {
		name string
		args []string
	}{
		{"unknown flag", []string{"--nope"}},
		{"missing flag value", []string{"--port"}},
		{"non numeric port", []string{"--port", "abc"}},
		{"port above range", []string{"--port", "70000"}},
		{"negative port", []string{"--port", "-1"}},
		{"mode not strict", []string{"--mode", "loose"}},
		{"log format unknown", []string{"--log-format", "xml"}},
		{"compat unknown", []string{"--compat", "unknown/v1"}},
		{"compat duplicate after normalization", []string{"--compat", "jev/v1", "--compat", "jev"}},
		{"compat empty", []string{"--compat", ""}},
		{"stray positional", []string{"extra"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"serve"}, test.args...)
			args = append(args, "--ready-file", readyPath)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != exitFailure {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage: fake-jev") {
				t.Fatalf("stderr does not carry usage: %q", stderr.String())
			}
			if _, err := os.Stat(readyPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("ready file was created by a usage failure")
			}
		})
	}
	t.Run("empty ready file", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		if code := Run([]string{"serve", "--ready-file", ""}, &stdout, &stderr); code != exitFailure {
			t.Fatalf("exit = %d, want %d", code, exitFailure)
		}
		if stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage: fake-jev") {
			t.Fatalf("stdout = %q stderr = %q", stdout.String(), stderr.String())
		}
	})
}

// TestUsageListsServeAsACommand pins the S1 usage-text update.
func TestUsageListsServeAsACommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := Run([]string{"help"}, &stdout, &stderr); code != exitOK {
		t.Fatalf("exit = %d, want %d", code, exitOK)
	}
	if !strings.Contains(stdout.String(), "\n  serve [flags] ") {
		t.Fatalf("usage does not list serve as a command:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "serve, run, verify") {
		t.Fatalf("usage still lists serve as unimplemented:\n%s", stdout.String())
	}
}

// TestServeConfigPrecedenceSeam pins C-CFG-011 exhaustively: resolveServeConfig
// is the merge seam for the §12.1 chain CLI flag > configuration file >
// built-in default, so every precedence direction is asserted directly without
// binding a listener.
func TestServeConfigPrecedenceSeam(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name          string
		config        string
		flags         []string
		wantHost      string
		wantPort      int
		wantProfile   []string
		wantMaxInter  int
		wantDataPlane int
		wantErr       bool
	}{
		{
			name:     "defaults",
			wantHost: "127.0.0.1", wantPort: 8787, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "file over default host and port",
			config:   "schemaVersion: 1\nserver:\n  host: 127.0.0.2\n  port: 1234\n",
			wantHost: "127.0.0.2", wantPort: 1234, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "file over default limits",
			config:   "schemaVersion: 1\nlimits:\n  maxInteractions: 7\n",
			wantHost: "127.0.0.1", wantPort: 8787, wantProfile: []string{"jev/v1"},
			wantMaxInter: 7, wantDataPlane: 8388608,
		},
		{
			name:     "flag over file port",
			config:   "schemaVersion: 1\nserver:\n  port: 8787\n",
			flags:    []string{"--port", "0"},
			wantHost: "127.0.0.1", wantPort: 0, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "flag over file host",
			config:   "schemaVersion: 1\nserver:\n  host: 127.0.0.2\n  port: 0\n",
			flags:    []string{"--host", "127.0.0.1"},
			wantHost: "127.0.0.1", wantPort: 0, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "flag port zero is not the default",
			flags:    []string{"--port", "0"},
			wantHost: "127.0.0.1", wantPort: 0, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "configured port zero is ephemeral",
			config:   "schemaVersion: 1\nserver:\n  port: 0\n",
			wantHost: "127.0.0.1", wantPort: 0, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:     "flag compat replaces file list",
			config:   "schemaVersion: 1\ncompatibility: [jev/v1]\n",
			flags:    []string{"--compat", "jev"},
			wantHost: "127.0.0.1", wantPort: 8787, wantProfile: []string{"jev/v1"},
			wantMaxInter: 10000, wantDataPlane: 8388608,
		},
		{
			name:    "empty host fails closed",
			config:  "schemaVersion: 1\nserver:\n  host: \"\"\n",
			wantErr: true,
		},
		{
			name:    "file mode not strict fails",
			config:  "schemaVersion: 1\nmode: loose\n",
			wantErr: true,
		},
		{
			name:    "file port out of range fails",
			config:  "schemaVersion: 1\nserver:\n  port: 70000\n",
			wantErr: true,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{}, test.flags...)
			if test.config != "" {
				path := filepath.Join(dir, strings.ReplaceAll(test.name, " ", "-")+".yaml")
				if err := os.WriteFile(path, []byte(test.config), 0o600); err != nil {
					t.Fatal(err)
				}
				args = append(args, "--config", path)
			}
			flags, err := parseServeFlags(args)
			if err != nil {
				t.Fatalf("parse %q: %v", args, err)
			}
			cfg, err := resolveServeConfig(flags)
			if test.wantErr {
				if err == nil {
					t.Fatalf("resolve succeeded, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if cfg.Server.Host != test.wantHost || cfg.Server.Port != test.wantPort {
				t.Fatalf("server = %s:%d, want %s:%d", cfg.Server.Host, cfg.Server.Port, test.wantHost, test.wantPort)
			}
			if cfg.Mode != "strict" {
				t.Fatalf("mode = %q, want strict", cfg.Mode)
			}
			if strings.Join(cfg.Compatibility, ",") != strings.Join(test.wantProfile, ",") {
				t.Fatalf("compatibility = %v, want %v", cfg.Compatibility, test.wantProfile)
			}
			if cfg.Limits.MaxInteractions != test.wantMaxInter || cfg.Limits.DataPlaneBodyBytes != test.wantDataPlane {
				t.Fatalf("limits = %+v", cfg.Limits)
			}
		})
	}
}

// TestServeConfigFailuresAreNotUsageFailures pins the S11 split: an unreadable
// or invalid configuration file exits 2 with a plain diagnostic that names the
// file (as validate does, §15.2), not the usage text, and still binds nothing.
func TestServeConfigFailuresAreNotUsageFailures(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready.json")
	invalid := writeConfigFile(t, dir, "schemaVersion: 1\nmode: loose\n")
	cases := []struct {
		name       string
		args       []string
		configPath string
	}{
		{"missing config", []string{"--config", filepath.Join(dir, "absent.yaml")}, filepath.Join(dir, "absent.yaml")},
		{"invalid config", []string{"--config", invalid}, invalid},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"serve", "--ready-file", readyPath}, test.args...)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != exitFailure {
				t.Fatalf("exit = %d, want %d", code, exitFailure)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("stderr is empty, want a diagnostic")
			}
			if strings.Contains(stderr.String(), "usage: fake-jev") {
				t.Fatalf("configuration failure printed usage: %q", stderr.String())
			}
			if !strings.Contains(stderr.String(), test.configPath) {
				t.Fatalf("diagnostic does not name the configuration file %q: %q", test.configPath, stderr.String())
			}
			if _, err := os.Stat(readyPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("ready file was created by a configuration failure")
			}
		})
	}
}

// TestServeDefaultBindIsLoopback pins C-ARCH-006 and the §19.1 default: the
// listener is loopback, the reported authority describes the live socket, and
// the control API answers on the first request after readiness (C-CTRL-001).
func TestServeDefaultBindIsLoopback(t *testing.T) {
	dir := t.TempDir()
	child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}})
	document := child.waitReady(t)

	if document.Host != "127.0.0.1" {
		t.Fatalf("ready host = %q, want 127.0.0.1", document.Host)
	}
	if ip := net.ParseIP(document.Host); ip == nil || !ip.IsLoopback() {
		t.Fatalf("ready host %q is not loopback", document.Host)
	}
	if document.Port <= 0 {
		t.Fatalf("ready port = %d, want an OS-assigned port", document.Port)
	}
	if document.URL != "http://127.0.0.1:"+strconv.Itoa(document.Port) {
		t.Fatalf("ready url = %q, want http://127.0.0.1:%d", document.URL, document.Port)
	}
	if document.PID != child.cmd.Process.Pid {
		t.Fatalf("ready pid = %d, want %d", document.PID, child.cmd.Process.Pid)
	}
	if document.ControlAPIVersion != ControlAPIVersion {
		t.Fatalf("ready controlApiVersion = %q, want %q", document.ControlAPIVersion, ControlAPIVersion)
	}

	status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/health", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("first health request status = %d (%s)", status, body)
	}
	if document.URL != "http://"+authorityOf(t, document) {
		t.Fatalf("ready url %q does not describe the bound authority", document.URL)
	}

	assertNotBoundPublicly(t, document.Host, document.Port)

	child.signal(t, syscall.SIGTERM)
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
	if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("ready file survives clean shutdown")
	}
}

// assertNotBoundPublicly dials every non-loopback local IPv4 address on the
// reported port and requires refusal. When the host has no such interface the
// sub-check is reported as not exercised (specifier H7).
func assertNotBoundPublicly(t *testing.T, host string, port int) {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatalf("interface addresses: %v", err)
	}
	dialed := 0
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		dialed++
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ipNet.IP.String(), strconv.Itoa(port)), 500*time.Millisecond)
		if err == nil {
			if closeErr := conn.Close(); closeErr != nil {
				t.Fatalf("close probe connection: %v", closeErr)
			}
			t.Fatalf("default bind accepted a connection on non-loopback address %s", ipNet.IP)
		}
	}
	if dialed == 0 {
		t.Log("no non-loopback IPv4 interface available; public-bind sub-check not exercised")
	}
}

// TestServeEphemeralPortsAreReported pins C-CLI-004: --port 0 yields a real
// OS-assigned port, reported in the ready file, and two concurrent servers get
// different ports.
func TestServeEphemeralPortsAreReported(t *testing.T) {
	first := startServe(t, serveLaunch{readyPath: filepath.Join(t.TempDir(), "ready.json"), args: []string{"--port", "0"}})
	firstDocument := first.waitReady(t)
	second := startServe(t, serveLaunch{readyPath: filepath.Join(t.TempDir(), "ready.json"), args: []string{"--port", "0"}})
	secondDocument := second.waitReady(t)

	if firstDocument.Port <= 0 || secondDocument.Port <= 0 {
		t.Fatalf("ephemeral ports = %d and %d, want positive", firstDocument.Port, secondDocument.Port)
	}
	if firstDocument.Port == secondDocument.Port {
		t.Fatalf("two ephemeral servers reported the same port %d", firstDocument.Port)
	}
	for _, document := range []readyDocument{firstDocument, secondDocument} {
		if !strings.HasSuffix(document.URL, ":"+strconv.Itoa(document.Port)) {
			t.Fatalf("url %q does not carry port %d", document.URL, document.Port)
		}
		status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/health", nil, nil)
		if status != nethttp.StatusOK {
			t.Fatalf("dial %s status = %d (%s)", document.URL, status, body)
		}
	}

	for _, child := range []*serveChild{first, second} {
		child.signal(t, syscall.SIGTERM)
		if code := child.wait(t, 10*time.Second); code != 0 {
			t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
		}
	}
}

// TestServeReadyFileIsAtomicAndClean pins C-CLI-006: every observed sample is
// either the complete pre-existing document or the complete new document, no
// temporary file survives readiness, and clean shutdown removes the file.
func TestServeReadyFileIsAtomicAndClean(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready.json")
	foreign := `{"pid":999999,"sentinel":true}`
	if err := os.WriteFile(readyPath, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}
	child := startServe(t, serveLaunch{readyPath: readyPath, args: []string{"--port", "0"}})

	var samples []string
	document, err := awaitReadyShape(readyPath, 10*time.Second, &samples)
	if err != nil {
		t.Fatalf("ready file: %v (stderr: %s)", err, child.stderr.String())
	}
	published, err := os.ReadFile(readyPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range samples {
		if sample != foreign && sample != string(published) {
			t.Fatalf("observed a partial ready-file sample: %q", sample)
		}
	}
	assertDirectoryEntries(t, dir, []string{"ready.json"})

	status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/health", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("health after readiness = %d (%s)", status, body)
	}

	child.signal(t, syscall.SIGTERM)
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
	assertDirectoryEntries(t, dir, nil)
}

func assertDirectoryEntries(t *testing.T, dir string, want []string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if len(names) != len(want) {
		t.Fatalf("directory entries = %v, want %v", names, want)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("directory entries = %v, want %v", names, want)
		}
	}
}

// TestServeStartupFailureLeavesReadyFileUntouched pins C-CLI-006 and §42.4: a
// failed start never disturbs a pre-existing ready file and still exits 2.
func TestServeStartupFailureLeavesReadyFileUntouched(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready.json")
	foreign := `{"pid":999999,"sentinel":true}`
	invalid := writeConfigFile(t, dir, "schemaVersion: 1\nmode: loose\n")

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	})
	held, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not a TCP address", listener.Addr())
	}

	cases := []struct {
		name string
		args []string
	}{
		{"missing config", []string{"--config", filepath.Join(dir, "absent.yaml")}},
		{"invalid config", []string{"--config", invalid}},
		{"port not bindable", []string{"--port", strconv.Itoa(held.Port)}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if err := os.WriteFile(readyPath, []byte(foreign), 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, stderr, code := runServeChild(t, serveLaunch{readyPath: readyPath, args: test.args, keepReadyFile: true})
			if code != exitFailure {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if stderr == "" {
				t.Fatal("stderr is empty, want a diagnostic")
			}
			data, err := os.ReadFile(readyPath)
			if err != nil {
				t.Fatalf("read ready file: %v", err)
			}
			if string(data) != foreign {
				t.Fatalf("ready file changed to %q, want %q", data, foreign)
			}
		})
	}
}

// TestRemoveReadyFilePidGuard pins C-CLI-006 (§42.4): removal happens only
// while the parsed pid still names this process.
func TestRemoveReadyFilePidGuard(t *testing.T) {
	dir := t.TempDir()
	mine := filepath.Join(dir, "mine.json")
	data, err := json.Marshal(readyDocument{URL: "http://127.0.0.1:1", Host: "127.0.0.1", Port: 1, PID: os.Getpid(), ControlAPIVersion: ControlAPIVersion})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mine, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeReadyFile(mine); err != nil {
		t.Fatalf("remove owned ready file: %v", err)
	}
	if _, err := os.Stat(mine); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("owned ready file was not removed")
	}

	theirs := filepath.Join(dir, "theirs.json")
	foreign := `{"url":"http://127.0.0.1:1","host":"127.0.0.1","port":1,"pid":1,"controlApiVersion":"v1"}`
	if err := os.WriteFile(theirs, []byte(foreign), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeReadyFile(theirs); err != nil {
		t.Fatalf("remove replacement ready file: %v", err)
	}
	if kept, err := os.ReadFile(theirs); err != nil || string(kept) != foreign {
		t.Fatalf("replacement ready file changed: %q %v", kept, err)
	}

	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeReadyFile(broken); err != nil {
		t.Fatalf("remove unparseable ready file: %v", err)
	}
	if kept, err := os.ReadFile(broken); err != nil || string(kept) != "not json" {
		t.Fatalf("unparseable ready file changed: %q %v", kept, err)
	}

	if err := removeReadyFile(filepath.Join(dir, "absent.json")); err != nil {
		t.Fatalf("remove absent ready file: %v", err)
	}
	if err := removeReadyFile(""); err != nil {
		t.Fatalf("remove with no ready-file flag: %v", err)
	}
}

// TestServeReadyFilePidGuardOnShutdown pins C-CLI-006 end to end: a ready file
// replaced while the server runs survives a clean shutdown untouched.
func TestServeReadyFilePidGuardOnShutdown(t *testing.T) {
	foreign := `{"url":"http://127.0.0.1:1","host":"127.0.0.1","port":1,"pid":1,"controlApiVersion":"v1"}`
	for _, test := range []struct {
		name    string
		content string
	}{
		{"foreign pid", foreign},
		{"unparseable", "not json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}, keepReadyFile: true})
			child.waitReady(t)
			if err := os.WriteFile(child.readyPath, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			child.signal(t, syscall.SIGTERM)
			if code := child.wait(t, 10*time.Second); code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
			}
			kept, err := os.ReadFile(child.readyPath)
			if err != nil {
				t.Fatalf("replacement ready file missing: %v", err)
			}
			if string(kept) != test.content {
				t.Fatalf("ready file = %q, want untouched %q", kept, test.content)
			}
		})
	}
}

// TestServeSignalShutdown pins C-CLI-007 (S8a): SIGINT and SIGTERM each stop
// the server, remove the ready file, and exit 0.
func TestServeSignalShutdown(t *testing.T) {
	for _, signal := range []struct {
		name  string
		value os.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	} {
		t.Run(signal.name, func(t *testing.T) {
			dir := t.TempDir()
			child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}})
			document := child.waitReady(t)
			child.signal(t, signal.value)
			if code := child.wait(t, 10*time.Second); code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
			}
			if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("ready file survives clean shutdown")
			}
			assertDirectoryEntries(t, dir, nil)
			if _, err := net.DialTimeout("tcp", authorityOf(t, document), 2*time.Second); err == nil {
				t.Fatal("listener still accepts connections after shutdown")
			}
		})
	}
}

// TestServeGracefulShutdownDrainsInflight pins C-CLI-007 (S8b): new accepts
// stop, the admitted request still receives its complete response, and the
// process exits 0.
func TestServeGracefulShutdownDrainsInflight(t *testing.T) {
	dir := t.TempDir()
	child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}})
	document := child.waitReady(t)
	authority := authorityOf(t, document)

	const declared = 65536
	payload := `{"state":"s","model":"m","questions":{"q":{"type":"noul"}}}`
	if declared < len(payload) {
		t.Fatalf("declared body %d smaller than payload %d", declared, len(payload))
	}
	body := payload + strings.Repeat(" ", declared-len(payload))

	conn, reader := holdRequest(t, authority, declared)
	child.signal(t, syscall.SIGTERM)
	awaitDialRefused(t, authority, 10*time.Second)
	if code, exited := child.exitedAlready(); exited {
		t.Fatalf("serve exited with %d while an in-flight request was still draining", code)
	}
	if _, err := io.WriteString(conn, body); err != nil {
		t.Fatalf("write remaining body: %v", err)
	}
	response, err := nethttp.ReadResponse(reader, &nethttp.Request{Method: nethttp.MethodPost})
	if err != nil {
		t.Fatalf("read final response: %v", err)
	}
	data, readErr := io.ReadAll(response.Body)
	if closeErr := response.Body.Close(); closeErr != nil {
		t.Fatalf("close held connection body: %v", closeErr)
	}
	if readErr != nil {
		t.Fatalf("read held response body: %v", readErr)
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close held connection: %v", err)
	}
	if response.StatusCode != nethttp.StatusNotImplemented {
		t.Fatalf("held request status = %d, want 501 (%s)", response.StatusCode, data)
	}
	if !strings.Contains(string(data), "fake_jev_unmatched_request") {
		t.Fatalf("held request body = %s", data)
	}
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
	if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("ready file survives clean shutdown")
	}
}

// TestServeForceCloseAfterGracefulTimeout pins C-CLI-007 (S8c): the configured
// gracefulShutdownSeconds is honored before remaining work is force-closed. The
// configured value is deliberately not the §12.2 default (5), and the process
// is observed still draining past that default, so the test distinguishes
// reading the configured limit from hardcoding the default constant.
func TestServeForceCloseAfterGracefulTimeout(t *testing.T) {
	const configuredSeconds = 10
	// A lower-bound observation longer than the default 5 s: an implementation
	// that ignores the configured value and uses the constant must have exited
	// by now. A correct implementation cannot exit before its grace elapses.
	const drainObservation = 6 * time.Second

	dir := t.TempDir()
	configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  port: 0\nlimits:\n  gracefulShutdownSeconds: "+strconv.Itoa(configuredSeconds)+"\n")
	child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath}})
	document := child.waitReady(t)
	authority := authorityOf(t, document)

	conn, reader := holdRequest(t, authority, 65536)
	child.signal(t, syscall.SIGTERM)
	select {
	case code := <-child.exited:
		child.waited, child.code = true, code
		t.Fatalf("serve exited with %d before the configured %d s grace period elapsed", code, configuredSeconds)
	case <-time.After(drainObservation):
	}
	if code := child.wait(t, time.Duration(configuredSeconds+10)*time.Second); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := reader.ReadByte(); err == nil {
		t.Fatal("held connection produced a response after force close")
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close held connection: %v", err)
	}
}

// TestServeControlAPIReachableThroughListener pins §19.5 and §41: the real
// listener serves both planes, stateful control methods are forwarded, and
// control traffic never enters the data-plane journal.
func TestServeControlAPIReachableThroughListener(t *testing.T) {
	dir := t.TempDir()
	child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--port", "0"}})
	document := child.waitReady(t)

	status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/health", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("health status = %d (%s)", status, body)
	}
	var health struct {
		Status            string `json:"status"`
		ServerVersion     string `json:"serverVersion"`
		ControlAPIVersion string `json:"controlApiVersion"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if health.Status != "ok" || health.ControlAPIVersion != ControlAPIVersion || health.ServerVersion == "" {
		t.Fatalf("health = %+v", health)
	}
	if health.ControlAPIVersion != document.ControlAPIVersion {
		t.Fatalf("health controlApiVersion %q != ready file %q", health.ControlAPIVersion, document.ControlAPIVersion)
	}

	status, body = serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/meta", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("meta status = %d (%s)", status, body)
	}
	var meta struct {
		ServerVersion       string   `json:"serverVersion"`
		ConfigSchemaVersion int      `json:"configSchemaVersion"`
		Mode                string   `json:"mode"`
		ActiveProfiles      []string `json:"activeProfiles"`
		Limits              struct {
			DataPlaneBodyBytes    int `json:"dataPlaneBodyBytes"`
			ControlPlaneBodyBytes int `json:"controlPlaneBodyBytes"`
			MaxInteractions       int `json:"maxInteractions"`
			LogBodyBytes          int `json:"logBodyBytes"`
			GracefulShutdown      int `json:"gracefulShutdownSeconds"`
		} `json:"limits"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		t.Fatalf("decode meta: %v", err)
	}
	if meta.Mode != "strict" || strings.Join(meta.ActiveProfiles, ",") != "jev/v1" {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.Limits.DataPlaneBodyBytes != 8388608 || meta.Limits.ControlPlaneBodyBytes != 2097152 ||
		meta.Limits.MaxInteractions != 10000 || meta.Limits.LogBodyBytes != 4096 || meta.Limits.GracefulShutdown != 5 {
		t.Fatalf("meta limits = %+v", meta.Limits)
	}

	if records := listRequests(t, document.URL); len(records) != 0 {
		t.Fatalf("control traffic was journalled: %v", records)
	}
	stub := `{"id":"dyn","profile":"jev/v1","when":{},"then":{"answers":{"q":{"noul":1}}}}`
	status, body = serveRequest(t, nethttp.MethodPost, document.URL+"/__fake/v1/stubs", strings.NewReader(stub), map[string]string{"Content-Type": "application/json"})
	if status != nethttp.StatusCreated {
		t.Fatalf("create stub status = %d (%s)", status, body)
	}
	var created struct {
		ID                string `json:"id"`
		RegistrationIndex uint64 `json:"registrationIndex"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created stub: %v", err)
	}
	if created.ID != "dyn" || created.RegistrationIndex != 1 {
		t.Fatalf("created = %+v", created)
	}

	status, body = serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/nope", nil, nil)
	if status != nethttp.StatusNotFound || !strings.Contains(string(body), "fake_jev_control_not_found") {
		t.Fatalf("unsupported control path = %d (%s)", status, body)
	}
	if records := listRequests(t, document.URL); len(records) != 0 {
		t.Fatalf("control traffic was journalled: %v", records)
	}

	status, body = serveRequest(t, nethttp.MethodGet, document.URL+"/v1/models", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("models status = %d (%s)", status, body)
	}
	records := listRequests(t, document.URL)
	if len(records) != 1 {
		t.Fatalf("request history = %v, want one record", records)
	}
	if sequence, ok := records[0]["sequence"].(float64); !ok || sequence != 1 {
		t.Fatalf("first sequence = %v, want 1", records[0]["sequence"])
	}
	if records[0]["path"] != "/v1/models" || records[0]["outcome"] != "matched" {
		t.Fatalf("record = %v", records[0])
	}

	child.signal(t, syscall.SIGTERM)
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
}

func listRequests(t *testing.T, baseURL string) []map[string]any {
	t.Helper()
	status, body := serveRequest(t, nethttp.MethodGet, baseURL+"/__fake/v1/requests", nil, nil)
	if status != nethttp.StatusOK {
		t.Fatalf("requests status = %d (%s)", status, body)
	}
	var payload struct {
		Requests []map[string]any `json:"requests"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode requests: %v", err)
	}
	return payload.Requests
}

// TestServePerPlaneBodyLimits pins the preserved per-plane limits through the
// real listener: an oversized control request is rejected without entering the
// journal, an oversized data request is journalled as payload_too_large.
func TestServePerPlaneBodyLimits(t *testing.T) {
	dir := t.TempDir()
	configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  port: 0\nlimits:\n  dataPlaneBodyBytes: 1024\n  controlPlaneBodyBytes: 1024\n")
	child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath}})
	document := child.waitReady(t)

	oversized := strings.Repeat("x", 2048)
	status, body := serveRequest(t, nethttp.MethodPost, document.URL+"/__fake/v1/stubs", strings.NewReader(oversized), nil)
	if status != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("oversized control status = %d (%s)", status, body)
	}
	if !strings.Contains(string(body), "fake_jev_payload_too_large") {
		t.Fatalf("oversized control body = %s", body)
	}
	if records := listRequests(t, document.URL); len(records) != 0 {
		t.Fatalf("oversized control request was journalled: %v", records)
	}

	status, body = serveRequest(t, nethttp.MethodPost, document.URL+"/v1/systemone", strings.NewReader(oversized), nil)
	if status != nethttp.StatusRequestEntityTooLarge {
		t.Fatalf("oversized data status = %d (%s)", status, body)
	}
	records := listRequests(t, document.URL)
	if len(records) != 1 || records[0]["outcome"] != "payload_too_large" {
		t.Fatalf("oversized data record = %v", records)
	}

	child.signal(t, syscall.SIGTERM)
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
}

// TestServeLogsGoToStderrAndRedactAuthorization pins C-CLI-010: operational
// logs stay on stderr, stdout carries no log record, and an Authorization
// value never appears in either stream, for both accepted --log-format values.
func TestServeLogsGoToStderrAndRedactAuthorization(t *testing.T) {
	const secret = "abababababababababababababababababababababababababababababababab"
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			dir := t.TempDir()
			child := startServe(t, serveLaunch{
				readyPath: filepath.Join(dir, "ready.json"),
				args:      []string{"--port", "0", "--log-format", format},
			})
			document := child.waitReady(t)
			headers := map[string]string{"Authorization": "Bearer " + secret}

			status, body := serveRequest(t, nethttp.MethodPost, document.URL+"/v1/systemone", strings.NewReader(`{"state":"s","model":"m","questions":{"q":{"type":"noul"}}}`), headers)
			if status != nethttp.StatusNotImplemented {
				t.Fatalf("data request status = %d (%s)", status, body)
			}
			status, body = serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/health", nil, headers)
			if status != nethttp.StatusOK {
				t.Fatalf("control request status = %d (%s)", status, body)
			}

			child.signal(t, syscall.SIGTERM)
			if code := child.wait(t, 10*time.Second); code != 0 {
				t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
			}
			stdout, stderr := child.stdout.String(), child.stderr.String()
			if strings.Contains(stdout, secret) || strings.Contains(stderr, secret) {
				t.Fatalf("Authorization value leaked: stdout=%q stderr=%q", stdout, stderr)
			}
			if stdout != "" {
				t.Fatalf("serve wrote operational data to stdout: %q", stdout)
			}
			if stderr == "" {
				t.Fatal("stderr is empty, want operational logs")
			}
		})
	}
}

// TestServePrecedenceEndToEnd exercises the §12.1 chain and the §12.1
// no-environment rule against real child processes (specifier S2).
func TestServePrecedenceEndToEnd(t *testing.T) {
	t.Run("file port over default", func(t *testing.T) {
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  port: 0\n")
		document := startAndStop(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath}})
		if document.Port <= 0 || document.Port == 8787 {
			t.Fatalf("port = %d, want a file-selected ephemeral port", document.Port)
		}
	})
	t.Run("flag port over file", func(t *testing.T) {
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  port: 8787\n")
		document := startAndStop(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath, "--port", "0"}})
		if document.Port <= 0 || document.Port == 8787 {
			t.Fatalf("port = %d, want the flag to win", document.Port)
		}
	})
	t.Run("file host over default", func(t *testing.T) {
		host := alternativeLoopbackHost(t)
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  host: "+host+"\n  port: 0\n")
		document := startAndStop(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath}})
		if document.Host != host {
			t.Fatalf("host = %q, want %q", document.Host, host)
		}
	})
	t.Run("flag host over file host", func(t *testing.T) {
		host := alternativeLoopbackHost(t)
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  host: "+host+"\n  port: 0\n")
		document := startAndStop(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath, "--host", "127.0.0.1"}})
		if document.Host != "127.0.0.1" {
			t.Fatalf("host = %q, want the flag to win", document.Host)
		}
	})
	t.Run("default port", func(t *testing.T) {
		probe, err := net.Listen("tcp", "127.0.0.1:8787")
		if err != nil {
			t.Skipf("default port 8787 is not bindable in this environment: %v", err)
		}
		if err := probe.Close(); err != nil {
			t.Fatalf("close probe listener: %v", err)
		}
		dir := t.TempDir()
		document := startAndStop(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json")})
		if document.Port != 8787 || document.Host != "127.0.0.1" {
			t.Fatalf("default authority = %s:%d, want 127.0.0.1:8787", document.Host, document.Port)
		}
	})
	t.Run("no implicit discovery", func(t *testing.T) {
		workDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(workDir, "fake-jev.yaml"), []byte("schemaVersion: 1\nserver:\n  host: 127.0.0.2\n  port: 0\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		document := startAndStop(t, serveLaunch{
			readyPath: filepath.Join(t.TempDir(), "ready.json"),
			dir:       workDir,
			args:      []string{"--port", "0"},
		})
		if document.Host != "127.0.0.1" {
			t.Fatalf("host = %q, want the cwd configuration to be ignored", document.Host)
		}
	})
	t.Run("environment is ignored", func(t *testing.T) {
		env := []string{
			"FAKE_JEV_HOST=0.0.0.0",
			"FAKE_JEV_PORT=9",
			"FAKE_JEV_CONFIG=/nonexistent",
			"FAKE_JEV_READY_FILE=/nonexistent",
			"FAKE_JEV_LOG_FORMAT=json",
			"HOST=0.0.0.0",
			"PORT=9",
		}
		document := startAndStop(t, serveLaunch{
			readyPath: filepath.Join(t.TempDir(), "ready.json"),
			env:       env,
			args:      []string{"--port", "0"},
		})
		if document.Host != "127.0.0.1" || document.Port == 9 {
			t.Fatalf("authority = %s:%d, want environment to be ignored", document.Host, document.Port)
		}
	})
	t.Run("file limits reach meta", func(t *testing.T) {
		dir := t.TempDir()
		configPath := writeConfigFile(t, dir, "schemaVersion: 1\nserver:\n  port: 0\nlimits:\n  maxInteractions: 7\n")
		child := startServe(t, serveLaunch{readyPath: filepath.Join(dir, "ready.json"), args: []string{"--config", configPath}})
		document := child.waitReady(t)
		status, body := serveRequest(t, nethttp.MethodGet, document.URL+"/__fake/v1/meta", nil, nil)
		if status != nethttp.StatusOK {
			t.Fatalf("meta status = %d (%s)", status, body)
		}
		var meta struct {
			Limits struct {
				MaxInteractions int `json:"maxInteractions"`
			} `json:"limits"`
		}
		if err := json.Unmarshal(body, &meta); err != nil {
			t.Fatalf("decode meta: %v", err)
		}
		if meta.Limits.MaxInteractions != 7 {
			t.Fatalf("maxInteractions = %d, want 7", meta.Limits.MaxInteractions)
		}
		child.signal(t, syscall.SIGTERM)
		if code := child.wait(t, 10*time.Second); code != 0 {
			t.Fatalf("exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
		}
	})
}

func startAndStop(t *testing.T, launch serveLaunch) readyDocument {
	t.Helper()
	child := startServe(t, launch)
	document := child.waitReady(t)
	child.signal(t, syscall.SIGTERM)
	if code := child.wait(t, 10*time.Second); code != 0 {
		t.Fatalf("serve exit = %d, want 0 (stderr: %s)", code, child.stderr.String())
	}
	if _, err := os.Stat(child.readyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ready file %s survives clean shutdown", child.readyPath)
	}
	return document
}
