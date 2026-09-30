package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	nethttp "net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// runWaitTimeout bounds every process-exit wait in this file. It is
// gracefulShutdownSeconds (default 5) plus the specifier's H5 margin, used only
// as a failure bound: run's own exit is the synchronization point.
const runWaitTimeout = 15 * time.Second

// Environment variables the helper child reads. They name the helper's
// parameters, not the variables run injects, so they cannot be confused with
// FAKE_JEV_URL or a --url-env name.
const (
	helperModeEnv     = "FAKE_JEV_HELPER_MODE"
	helperExitEnv     = "FAKE_JEV_HELPER_EXIT"
	helperSnapshotEnv = "FAKE_JEV_HELPER_SNAPSHOT"
	helperStartedEnv  = "FAKE_JEV_HELPER_STARTED"
	helperSignaledEnv = "FAKE_JEV_HELPER_SIGNALED"
	helperURLEnvName  = "FAKE_JEV_HELPER_URL_ENV"
	helperAliasEnv    = "FAKE_JEV_HELPER_ALIAS"
)

// runLaunch describes one child `run` invocation. dropEnv names variables to
// remove from the inherited environment so the case is not perturbed by the
// developer's shell.
type runLaunch struct {
	args    []string
	env     []string
	dropEnv []string
	stdin   io.Reader
}

// runChild is a running `run` process with its captured streams and exit status.
// run's exit is the only synchronization point the lifecycle cases need.
type runChild struct {
	cmd    *exec.Cmd
	stdout syncBuffer
	stderr syncBuffer
	exited chan int
	waited bool
	code   int
}

func startRun(t *testing.T, launch runLaunch) *runChild {
	t.Helper()
	cmd := exec.Command(serveBinary, launch.args...)
	drop := append([]string{fakeJevURLEnv}, launch.dropEnv...)
	for _, entry := range launch.env {
		name, _, _ := strings.Cut(entry, "=")
		drop = append(drop, name)
	}
	cmd.Env = append(environWithout(os.Environ(), drop...), launch.env...)
	cmd.Stdin = launch.stdin
	child := &runChild{cmd: cmd, exited: make(chan int, 1)}
	cmd.Stdout = &child.stdout
	cmd.Stderr = &child.stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start run: %v", err)
	}
	go func() { child.exited <- waitExitCode(cmd.Wait()) }()
	t.Cleanup(func() { child.stop(t) })
	return child
}

func (child *runChild) stop(t *testing.T) {
	if child.waited {
		return
	}
	if child.cmd.Process != nil {
		if err := child.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("kill run process: %v", err)
		}
	}
	select {
	case <-child.exited:
		child.waited = true
	case <-time.After(10 * time.Second):
		t.Errorf("run process did not exit after being killed")
	}
}

func (child *runChild) wait(t *testing.T, timeout time.Duration) int {
	t.Helper()
	if child.waited {
		return child.code
	}
	select {
	case code := <-child.exited:
		child.waited, child.code = true, code
		return code
	case <-time.After(timeout):
		t.Fatalf("run process did not exit within %s (stderr: %s)", timeout, child.stderr.String())
		return -1
	}
}

func (child *runChild) signal(t *testing.T, sig os.Signal) {
	t.Helper()
	if err := child.cmd.Process.Signal(sig); err != nil {
		t.Fatalf("signal %s: %v", sig, err)
	}
}

// environWithout returns environ with every entry naming one of the listed
// variables removed, so an appended value cannot be shadowed by an inherited
// duplicate.
func environWithout(environ []string, names ...string) []string {
	drop := make(map[string]bool, len(names))
	for _, name := range names {
		drop[name] = true
	}
	kept := make([]string, 0, len(environ))
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if drop[name] {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// helperEnv builds the environment for one helper child invocation.
func helperEnv(mode string, exitCode int, snapshotPath, startedPath, signaledPath, urlEnvName string) []string {
	env := []string{
		helperModeEnv + "=" + mode,
		helperExitEnv + "=" + strconv.Itoa(exitCode),
	}
	for _, entry := range []struct {
		name  string
		value string
	}{
		{helperSnapshotEnv, snapshotPath},
		{helperStartedEnv, startedPath},
		{helperSignaledEnv, signaledPath},
		{helperURLEnvName, urlEnvName},
	} {
		if entry.value != "" {
			env = append(env, entry.name+"="+entry.value)
		}
	}
	return env
}

// helperSnapshot is the document the helper child writes before it acts: the URL
// run injected, the value of the --url-env variable, and how the server answered
// its data-plane probe.
type helperSnapshot struct {
	FakeJevURL    string `json:"fakeJevURL"`
	URLEnvName    string `json:"urlEnvName"`
	URLEnvValue   string `json:"urlEnvValue"`
	URLEnvPresent bool   `json:"urlEnvPresent"`
	ModelsStatus  int    `json:"modelsStatus"`
	UnknownStatus int    `json:"unknownStatus"`
	AliasStatus   int    `json:"aliasStatus"`
	AliasError    string `json:"aliasError"`
	RequestError  string `json:"requestError"`
	PID           int    `json:"pid"`
}

func readHelperSnapshot(t *testing.T, path string) helperSnapshot {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read helper snapshot: %v", err)
	}
	var snapshot helperSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("decode helper snapshot %s: %v", data, err)
	}
	return snapshot
}

// awaitFile polls until path exists, bounded by timeout. The file is the
// helper's own readiness barrier, so no sleep is used as synchronization.
func awaitFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("file %s did not appear within %s", path, timeout)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// assertURLRefused requires the injected server URL to be closed once run has
// exited: §42.2 requires a clean shutdown even after a nonzero or interrupted
// child, so nothing may keep listening.
func assertURLRefused(t *testing.T, rawURL string) {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse injected URL %q: %v", rawURL, err)
	}
	conn, err := net.DialTimeout("tcp", parsed.Host, 2*time.Second)
	if err == nil {
		if closeErr := conn.Close(); closeErr != nil {
			t.Errorf("close probe connection: %v", closeErr)
		}
		t.Fatalf("server at %s still accepts connections after run exited", rawURL)
	}
}

// assertLoopbackURL requires the injected URL to be an http URL on a loopback
// host and an OS-assigned port (H3/H4, §19.1).
func assertLoopbackURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse injected URL %q: %v", rawURL, err)
	}
	if parsed.Scheme != "http" {
		t.Fatalf("injected URL scheme = %q, want http", parsed.Scheme)
	}
	host, port := parsed.Hostname(), parsed.Port()
	if parsedIP := net.ParseIP(host); parsedIP == nil || !parsedIP.IsLoopback() {
		t.Fatalf("injected URL host %q is not loopback", host)
	}
	number, err := strconv.Atoi(port)
	if err != nil || number <= 0 {
		t.Fatalf("injected URL port = %q, want an OS-assigned port", port)
	}
	return parsed
}

// runHelperSource is the child program the lifecycle cases execute. It records
// what run injected into its environment and how the running server answered,
// then performs the mode's action. It is built into the test's own temporary
// directory, so the suite leaves no artifact behind and needs no network.
const runHelperSource = `package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Modes: "pass" probes the models route and exits; "fail" performs one failing
// data-plane request (an unknown route) and exits; "signal" waits for a
// forwarded interrupt, records it, and exits; "hold" ignores interrupts until
// it is force terminated; "wildcard" additionally probes the 127.0.0.2 loopback
// alias to show whether the server is bound to the wildcard address.
func main() {
	url := os.Getenv("FAKE_JEV_URL")
	name := os.Getenv("FAKE_JEV_HELPER_URL_ENV")
	mode := os.Getenv("FAKE_JEV_HELPER_MODE")
	aliasHost := os.Getenv("FAKE_JEV_HELPER_ALIAS")
	document := map[string]any{
		"fakeJevURL":    url,
		"urlEnvName":    name,
		"urlEnvValue":   "",
		"urlEnvPresent": false,
		"modelsStatus":  -1,
		"unknownStatus": -1,
		"aliasStatus":   -1,
		"aliasError":    "",
		"requestError":  "",
		"pid":           os.Getpid(),
	}
	if name != "" {
		value, present := os.LookupEnv(name)
		document["urlEnvValue"], document["urlEnvPresent"] = value, present
	}
	// The handler is installed before the snapshot and the "started" file are
	// published, so a signal sent after the barrier is always delivered to the
	// handler rather than killing the process through the default disposition.
	received := make(chan os.Signal, 1)
	if mode == "signal" || mode == "hold" {
		signal.Notify(received, os.Interrupt, syscall.SIGTERM)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	if mode == "fail" {
		document["unknownStatus"], document["requestError"] = request(client, url+"/nope")
	} else {
		document["modelsStatus"], document["requestError"] = request(client, url+"/v1/models")
	}
	if mode == "wildcard" && aliasHost != "" {
		document["aliasStatus"], document["aliasError"] = request(client, aliasURL(url, aliasHost)+"/v1/models")
	}
	fmt.Println("fake-jev helper:", mode)
	writeJSON(os.Getenv("FAKE_JEV_HELPER_SNAPSHOT"), document)
	writeFile(os.Getenv("FAKE_JEV_HELPER_STARTED"), strconv.Itoa(os.Getpid()))
	switch mode {
	case "signal":
		<-received
		writeFile(os.Getenv("FAKE_JEV_HELPER_SIGNALED"), "received")
	case "hold":
		for {
			time.Sleep(time.Hour)
		}
	}
	os.Exit(exitCode())
}

func request(client *http.Client, target string) (int, string) {
	response, err := client.Get(target)
	if err != nil {
		return -1, err.Error()
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return response.StatusCode, err.Error()
	}
	return response.StatusCode, ""
}

// aliasURL rewrites the injected host to the given one, keeping the port, so a
// probe on it distinguishes a loopback-only bind from a wildcard bind.
func aliasURL(rawURL, host string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	parsed.Host = net.JoinHostPort(host, parsed.Port())
	return parsed.String()
}

func writeJSON(path string, document any) {
	if path == "" {
		return
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper: encode snapshot:", err)
		os.Exit(2)
	}
	writeFile(path, string(encoded))
}

func writeFile(path, content string) {
	if path == "" {
		return
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "helper: write", path+":", err)
		os.Exit(2)
	}
}

func exitCode() int {
	value := os.Getenv("FAKE_JEV_HELPER_EXIT")
	if value == "" {
		return 0
	}
	code, err := strconv.Atoi(value)
	if err != nil {
		fmt.Fprintln(os.Stderr, "helper: bad exit code", value)
		return 2
	}
	return code
}
`

// buildRunHelper compiles the helper child into a temporary directory owned by
// the test. The build is offline (stdlib only) and cached by the Go build cache.
func buildRunHelper(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fake-jev-test-helper\n\ngo 1.21\n"), 0o600); err != nil {
		t.Fatalf("write helper module: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(runHelperSource), 0o600); err != nil {
		t.Fatalf("write helper source: %v", err)
	}
	binary := filepath.Join(dir, "helper")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = dir
	build.Env = append(environWithout(os.Environ(), "GOFLAGS", "GOPROXY", "GOWORK"), "GOFLAGS=", "GOPROXY=off", "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, output)
	}
	return binary
}

// helperDir prepares the per-case directory holding the helper's files.
type helperDir struct {
	snapshot string
	started  string
	signaled string
}

func newHelperDir(t *testing.T) helperDir {
	t.Helper()
	dir := t.TempDir()
	return helperDir{
		snapshot: filepath.Join(dir, "snapshot.json"),
		started:  filepath.Join(dir, "started"),
		signaled: filepath.Join(dir, "signaled"),
	}
}

// TestRunExitPrecedence pins C-CLI-002, C-GOLD-013 and §44.13 end to end: the
// exact child/verify precedence table, with verification pass/fail produced by
// real server state (a matched models request versus an unknown route). The
// child=7 + verify-fail row asserts both failures §42.2 requires: the frozen
// C-GOLD-013 token "verification" and the child's own nonzero exit.
func TestRunExitPrecedence(t *testing.T) {
	helper := buildRunHelper(t)
	cases := []struct {
		name              string
		helperExit        int
		mode              string
		wantCode          int
		wantFailureReport bool
		wantChildReport   bool
	}{
		{"child 0 verify pass", 0, "pass", exitOK, false, false},
		{"child 0 verify fail", 0, "fail", exitVerify, true, false},
		{"child 7 verify pass", 7, "pass", 7, false, true},
		{"child 7 verify fail", 7, "fail", 7, true, true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			files := newHelperDir(t)
			child := startRun(t, runLaunch{
				args: []string{"run", "--", helper},
				env:  helperEnv(test.mode, test.helperExit, files.snapshot, files.started, "", ""),
			})
			if code := child.wait(t, runWaitTimeout); code != test.wantCode {
				t.Fatalf("exit = %d, want %d (stdout: %q stderr: %q)", code, test.wantCode, child.stdout.String(), child.stderr.String())
			}
			report := child.stderr.String()
			// The frozen C-GOLD-013 token for this row, not an implementation
			// fault code, so a rewording that dropped the word would be caught.
			if got := strings.Contains(report, "verification"); got != test.wantFailureReport {
				t.Fatalf("stderr reports a verification failure = %v, want %v: %q", got, test.wantFailureReport, report)
			}
			// §42.2: when the child exits nonzero and verification also fails, both
			// failures are reported; either way the child's code is named.
			wantChildLine := fmt.Sprintf("fake-jev: child exited with code %d", test.helperExit)
			if got := strings.Contains(report, wantChildLine); got != test.wantChildReport {
				t.Fatalf("stderr names the child exit %q = %v, want %v: %q", wantChildLine, got, test.wantChildReport, report)
			}
			// The child ran against the injected server and its own output
			// reached run's caller (§42.5, §28).
			if !strings.Contains(child.stdout.String(), "fake-jev helper: "+test.mode) {
				t.Fatalf("child output missing from run stdout: %q", child.stdout.String())
			}
			snapshot := readHelperSnapshot(t, files.snapshot)
			switch test.mode {
			case "pass":
				if snapshot.ModelsStatus != nethttp.StatusOK {
					t.Fatalf("models probe = %d, want 200 (error: %s)", snapshot.ModelsStatus, snapshot.RequestError)
				}
			case "fail":
				if snapshot.UnknownStatus != nethttp.StatusNotFound {
					t.Fatalf("unknown-route probe = %d, want 404 (error: %s)", snapshot.UnknownStatus, snapshot.RequestError)
				}
			}
			// §42.2: the server is stopped after a nonzero child too.
			assertURLRefused(t, snapshot.FakeJevURL)
		})
	}
}

// TestRunInjectsServerURLAndURLEnv pins C-CLI-003 (§15.4 steps 4-5): FAKE_JEV_URL
// always carries the live server URL even when the parent exported a stale one,
// and --url-env additionally sets (and overrides) the named variable.
func TestRunInjectsServerURLAndURLEnv(t *testing.T) {
	helper := buildRunHelper(t)
	const extraVar = "JEV_BASE_URL"
	const staleURL = "http://127.0.0.1:1"
	cases := []struct {
		name        string
		args        []string
		env         []string
		dropEnv     []string
		wantPresent bool
		wantValue   string
	}{
		{
			name: "always sets FAKE_JEV_URL", env: []string{fakeJevURLEnv + "=" + staleURL},
			dropEnv: []string{extraVar}, wantPresent: false,
		},
		{
			name: "url-env adds an absent variable",
			args: []string{"--url-env", extraVar}, dropEnv: []string{extraVar},
			wantPresent: true, wantValue: "",
		},
		{
			name: "url-env overrides an inherited value",
			args: []string{"--url-env", extraVar}, env: []string{extraVar + "=stale", fakeJevURLEnv + "=" + staleURL},
			wantPresent: true, wantValue: "",
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			files := newHelperDir(t)
			args := append([]string{"run"}, test.args...)
			args = append(args, "--", helper)
			child := startRun(t, runLaunch{
				args:    args,
				env:     append(helperEnv("pass", 0, files.snapshot, files.started, "", extraVar), test.env...),
				dropEnv: test.dropEnv,
			})
			if code := child.wait(t, runWaitTimeout); code != exitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, child.stderr.String())
			}

			snapshot := readHelperSnapshot(t, files.snapshot)
			assertLoopbackURL(t, snapshot.FakeJevURL)
			if snapshot.FakeJevURL == staleURL {
				t.Fatalf("child saw the inherited FAKE_JEV_URL %q", snapshot.FakeJevURL)
			}
			if snapshot.ModelsStatus != nethttp.StatusOK {
				t.Fatalf("models probe = %d, want 200 (error: %s)", snapshot.ModelsStatus, snapshot.RequestError)
			}
			if snapshot.URLEnvPresent != test.wantPresent {
				t.Fatalf("child %s present = %v, want %v", extraVar, snapshot.URLEnvPresent, test.wantPresent)
			}
			if test.wantPresent && snapshot.URLEnvValue != snapshot.FakeJevURL {
				t.Fatalf("child %s = %q, want the same URL as FAKE_JEV_URL %q", extraVar, snapshot.URLEnvValue, snapshot.FakeJevURL)
			}
			if !test.wantPresent && snapshot.URLEnvValue != "" {
				t.Fatalf("child %s = %q, want the variable untouched", extraVar, snapshot.URLEnvValue)
			}
			assertURLRefused(t, snapshot.FakeJevURL)
		})
	}
}

// freePort returns an OS-assigned loopback port that is not currently bound.
// The listening socket is closed immediately, so the value is used only to
// write a configuration whose port a correct run must ignore; it is never the
// port a test binds.
func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not a TCP address", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	return address.Port
}

// TestRunUsesEphemeralPortDespiteConfiguredPort pins C-CLI-004 (§15.4): run
// ignores the file's server.port unless --port is passed. The test holds a
// listener on an OS-chosen port, so an implementation that honoured the file
// value could not bind and would fail instead of exiting 0. The --port leg uses
// a configuration whose port differs from the flag value, so a bind attempt can
// only be attributed to the flag.
func TestRunUsesEphemeralPortDespiteConfiguredPort(t *testing.T) {
	helper := buildRunHelper(t)
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if err := held.Close(); err != nil {
			t.Errorf("close held listener: %v", err)
		}
	})
	heldAddress, ok := held.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("held listener address %v is not a TCP address", held.Addr())
	}
	heldConfig := writeConfigFile(t, t.TempDir(), fmt.Sprintf("schemaVersion: 1\nserver:\n  port: %d\n", heldAddress.Port))
	otherConfig := writeConfigFile(t, t.TempDir(), fmt.Sprintf("schemaVersion: 1\nserver:\n  port: %d\n", freePort(t)))

	t.Run("file port is ignored", func(t *testing.T) {
		files := newHelperDir(t)
		child := startRun(t, runLaunch{
			args: []string{"run", "--config", heldConfig, "--", helper},
			env:  helperEnv("pass", 0, files.snapshot, files.started, "", ""),
		})
		if code := child.wait(t, runWaitTimeout); code != exitOK {
			t.Fatalf("exit = %d, want %d: the configured port must not be bound (stderr: %s)", code, exitOK, child.stderr.String())
		}
		snapshot := readHelperSnapshot(t, files.snapshot)
		injected := assertLoopbackURL(t, snapshot.FakeJevURL)
		if injected.Port() == strconv.Itoa(heldAddress.Port) {
			t.Fatalf("injected port = %s, want an ephemeral port distinct from the configured %d", injected.Port(), heldAddress.Port)
		}
		assertURLRefused(t, snapshot.FakeJevURL)
	})

	t.Run("explicit port is honoured", func(t *testing.T) {
		// otherConfig names a free port that run must not bind: only the --port
		// value (the held port) can produce the bind failure below.
		files := newHelperDir(t)
		child := startRun(t, runLaunch{
			args: []string{"run", "--config", otherConfig, "--port", strconv.Itoa(heldAddress.Port), "--", helper},
			env:  helperEnv("pass", 0, files.snapshot, files.started, "", ""),
		})
		if code := child.wait(t, runWaitTimeout); code != exitFailure {
			t.Fatalf("exit = %d, want %d for an unbindable --port (stderr: %s)", code, exitFailure, child.stderr.String())
		}
		if child.stderr.String() == "" {
			t.Fatal("stderr is empty, want a startup diagnostic")
		}
		if _, err := os.Stat(files.snapshot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("the child ran although the server could not start")
		}
	})
}

// TestRunBindsLoopbackHostDespiteConfiguredWildcardHost pins the §15.4 step 2
// rule that run starts the server on a loopback port: a configuration naming the
// wildcard host must not widen the bind, and the child's injected FAKE_JEV_URL
// must stay loopback (§19.1, §30.8). A server bound to the wildcard address also
// answers on this host's non-loopback address, which a loopback-only bind
// refuses.
func TestRunBindsLoopbackHostDespiteConfiguredWildcardHost(t *testing.T) {
	helper := buildRunHelper(t)
	address := nonLoopbackLocalAddress()
	configPath := writeConfigFile(t, t.TempDir(), "schemaVersion: 1\nserver:\n  host: 0.0.0.0\n")
	files := newHelperDir(t)
	env := helperEnv("wildcard", 0, files.snapshot, files.started, "", "")
	if address != "" {
		env = append(env, helperAliasEnv+"="+address)
	}
	child := startRun(t, runLaunch{
		args: []string{"run", "--config", configPath, "--", helper},
		env:  env,
	})
	if code := child.wait(t, runWaitTimeout); code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, child.stderr.String())
	}
	snapshot := readHelperSnapshot(t, files.snapshot)
	injected := assertLoopbackURL(t, snapshot.FakeJevURL)
	if snapshot.ModelsStatus != nethttp.StatusOK {
		t.Fatalf("models probe = %d, want 200 (error: %s)", snapshot.ModelsStatus, snapshot.RequestError)
	}
	if address == "" {
		t.Log("no non-loopback local address is available; the wildcard-bind check is not exercised")
	} else if snapshot.AliasStatus != -1 || snapshot.AliasError == "" {
		t.Fatalf("the server answered on %s:%s (status %d), so it is bound to the wildcard address, not %s", address, injected.Port(), snapshot.AliasStatus, injected.Host)
	}
	assertURLRefused(t, snapshot.FakeJevURL)
}

// nonLoopbackLocalAddress returns a global-unicast IPv4 address of this host, or
// "" when there is none. A wildcard bind answers on it while a loopback-only
// bind refuses it immediately.
func nonLoopbackLocalAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range interfaces {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			network, ok := address.(*net.IPNet)
			if !ok {
				continue
			}
			ip := network.IP
			if ip.To4() == nil || ip.IsLoopback() || !ip.IsGlobalUnicast() {
				continue
			}
			return ip.String()
		}
	}
	return ""
}

// TestRunUsageFailures pins R1 (§15.4, §42.1): a malformed command line exits 2
// with the diagnostic and usage on stderr, claims no result on stdout, and runs
// no child. The helper would record its snapshot if it ever started.
func TestRunUsageFailures(t *testing.T) {
	helper := buildRunHelper(t)
	dir := t.TempDir()
	files := newHelperDir(t)
	validConfig := writeConfigFile(t, dir, "schemaVersion: 1\n")
	t.Setenv(helperSnapshotEnv, files.snapshot)
	t.Setenv(helperStartedEnv, files.started)

	cases := []struct {
		name string
		args []string
	}{
		{"no arguments", nil},
		{"no separator", []string{helper}},
		{"empty command after separator", []string{"--"}},
		{"config without separator", []string{"--config", validConfig}},
		{"unknown flag", []string{"--nope", "--", helper}},
		{"empty url-env", []string{"--url-env", "", "--", helper}},
		{"url-env without value", []string{"--url-env", "--", helper}},
		{"non numeric port", []string{"--port", "abc", "--", helper}},
		{"port above range", []string{"--port", "70000", "--", helper}},
		{"stray positional", []string{"stray", "--", helper}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"run"}, test.args...)
			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)
			if code != exitFailure {
				t.Fatalf("args %q: exit = %d, want %d (stderr: %s)", args, code, exitFailure, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), "usage: fake-jev") {
				t.Fatalf("stderr does not carry usage: %q", stderr.String())
			}
			if _, err := os.Stat(files.snapshot); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("a usage failure ran the child")
			}
		})
	}
}

// TestRunStartupFailuresExitTwo pins §42.2 and R6: an invalid configuration, an
// unbindable port, and an unstartable child each exit 2 with a diagnostic on
// stderr, run no child, and leave no listener behind.
func TestRunStartupFailuresExitTwo(t *testing.T) {
	helper := buildRunHelper(t)
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		if err := held.Close(); err != nil {
			t.Errorf("close held listener: %v", err)
		}
	})
	heldAddress, ok := held.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("held listener address %v is not a TCP address", held.Addr())
	}
	dir := t.TempDir()
	invalidConfig := writeConfigFile(t, dir, "schemaVersion: 1\nmode: loose\n")

	t.Run("invalid configuration", func(t *testing.T) {
		files := newHelperDir(t)
		child := startRun(t, runLaunch{
			args: []string{"run", "--config", invalidConfig, "--", helper},
			env:  helperEnv("pass", 0, files.snapshot, files.started, "", ""),
		})
		if code := child.wait(t, runWaitTimeout); code != exitFailure {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, child.stderr.String())
		}
		if !strings.Contains(child.stderr.String(), invalidConfig) {
			t.Fatalf("diagnostic does not name the configuration file: %q", child.stderr.String())
		}
		if _, err := os.Stat(files.snapshot); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("the child ran although configuration validation failed")
		}
	})

	t.Run("missing configuration", func(t *testing.T) {
		absent := filepath.Join(dir, "absent.yaml")
		child := startRun(t, runLaunch{args: []string{"run", "--config", absent, "--", helper}})
		if code := child.wait(t, runWaitTimeout); code != exitFailure {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, child.stderr.String())
		}
		if !strings.Contains(child.stderr.String(), absent) {
			t.Fatalf("diagnostic does not name the configuration file: %q", child.stderr.String())
		}
	})

	t.Run("unbindable port", func(t *testing.T) {
		child := startRun(t, runLaunch{args: []string{"run", "--port", strconv.Itoa(heldAddress.Port), "--", helper}})
		if code := child.wait(t, runWaitTimeout); code != exitFailure {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, child.stderr.String())
		}
		if child.stderr.String() == "" {
			t.Fatal("stderr is empty, want a diagnostic")
		}
		conn, err := net.DialTimeout("tcp", heldAddress.String(), 2*time.Second)
		if err != nil {
			t.Fatalf("the held listener stopped accepting after run failed to bind: %v", err)
		}
		if err := conn.Close(); err != nil {
			t.Errorf("close probe connection: %v", err)
		}
	})

	t.Run("child cannot be started", func(t *testing.T) {
		const absent = "fake-jev-fj032-no-such-command"
		child := startRun(t, runLaunch{args: []string{"run", "--", absent}})
		if code := child.wait(t, runWaitTimeout); code != exitFailure {
			t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitFailure, child.stderr.String())
		}
		if !strings.Contains(child.stderr.String(), absent) {
			t.Fatalf("diagnostic does not name the child command: %q", child.stderr.String())
		}
	})
}

// TestRunForwardsSignalToChild pins C-CLI-007 (§42.3 steps 1-2, 4-5): an
// interrupt delivered to run reaches the child while run is still alive, the
// child is allowed to finish, the server is stopped, and the exit status is the
// conventional signal-derived one.
func TestRunForwardsSignalToChild(t *testing.T) {
	helper := buildRunHelper(t)
	for _, interrupt := range []struct {
		name   string
		signal syscall.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	} {
		t.Run(interrupt.name, func(t *testing.T) {
			files := newHelperDir(t)
			child := startRun(t, runLaunch{
				args: []string{"run", "--", helper},
				env:  helperEnv("signal", 0, files.snapshot, files.started, files.signaled, ""),
			})
			awaitFile(t, files.started, 10*time.Second)
			child.signal(t, interrupt.signal)
			code := child.wait(t, runWaitTimeout)
			if want := 128 + int(interrupt.signal); code != want {
				t.Fatalf("exit = %d, want %d (stderr: %q)", code, want, child.stderr.String())
			}
			if _, err := os.Stat(files.signaled); err != nil {
				t.Fatalf("the child never observed the forwarded %s: %v (stderr: %q)", interrupt.name, err, child.stderr.String())
			}
			snapshot := readHelperSnapshot(t, files.snapshot)
			assertURLRefused(t, snapshot.FakeJevURL)
		})
	}
}

// TestRunForceTerminatesChildAfterGracefulTimeout pins C-CLI-007 (§42.3 step 3):
// a child that ignores the forwarded signal is allowed the configured graceful
// shutdown timeout and then force terminated, and the server is still stopped.
func TestRunForceTerminatesChildAfterGracefulTimeout(t *testing.T) {
	helper := buildRunHelper(t)
	const configuredSeconds = 2
	// A bounded negative observation below the configured grace: an
	// implementation that force terminated immediately (or did not wait at all)
	// would already have exited.
	const drainObservation = 1 * time.Second

	dir := t.TempDir()
	configPath := writeConfigFile(t, dir, "schemaVersion: 1\nlimits:\n  gracefulShutdownSeconds: 2\n")
	files := newHelperDir(t)
	child := startRun(t, runLaunch{
		args: []string{"run", "--config", configPath, "--", helper},
		env:  helperEnv("hold", 0, files.snapshot, files.started, files.signaled, ""),
	})
	awaitFile(t, files.started, 10*time.Second)
	child.signal(t, syscall.SIGTERM)
	select {
	case code := <-child.exited:
		child.waited, child.code = true, code
		t.Fatalf("run exited with %d before the configured %d s graceful timeout elapsed", code, configuredSeconds)
	case <-time.After(drainObservation):
	}
	code := child.wait(t, time.Duration(configuredSeconds+10)*time.Second)
	if want := 128 + int(syscall.SIGTERM); code != want {
		t.Fatalf("exit = %d, want %d (stderr: %q)", code, want, child.stderr.String())
	}
	if _, err := os.Stat(files.signaled); err == nil {
		t.Fatal("the helper recorded a handled signal although it ignores them")
	}
	snapshot := readHelperSnapshot(t, files.snapshot)
	if snapshot.PID <= 0 {
		t.Fatalf("helper pid = %d, want the recorded child pid", snapshot.PID)
	}
	// run reaps the child, so the pid must be gone once run exits. The probe is
	// Unix kill(pid, 0) semantics, as elsewhere in this package's tests.
	process, err := os.FindProcess(snapshot.PID)
	if err != nil {
		t.Fatalf("find helper process: %v", err)
	}
	if err := process.Signal(syscall.Signal(0)); err == nil {
		t.Fatalf("helper process %d outlived run's force termination", snapshot.PID)
	}
	assertURLRefused(t, snapshot.FakeJevURL)
}

// TestRunChildStdinIsNotRequired pins R8: the child reads the null device, so a
// closed or absent stdin cannot make run hang or prompt (§15).
func TestRunChildStdinIsNotRequired(t *testing.T) {
	helper := buildRunHelper(t)
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if err := writeEnd.Close(); err != nil {
		t.Fatalf("close write end: %v", err)
	}
	t.Cleanup(func() {
		if err := readEnd.Close(); err != nil {
			t.Errorf("close read end: %v", err)
		}
	})

	files := newHelperDir(t)
	child := startRun(t, runLaunch{
		args:  []string{"run", "--", helper},
		env:   helperEnv("pass", 0, files.snapshot, files.started, "", ""),
		stdin: readEnd,
	})
	if code := child.wait(t, runWaitTimeout); code != exitOK {
		t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, child.stderr.String())
	}
	snapshot := readHelperSnapshot(t, files.snapshot)
	if snapshot.ModelsStatus != nethttp.StatusOK {
		t.Fatalf("models probe = %d, want 200 (error: %s)", snapshot.ModelsStatus, snapshot.RequestError)
	}
}

// TestRunHelpPrintsUsage pins the V-U6 implementation choice: a help flag is not
// a usage failure, so it prints the usage on stdout and exits 0 as serve does,
// and it starts no server.
func TestRunHelpPrintsUsage(t *testing.T) {
	for _, args := range [][]string{{"run", "--help"}, {"run", "-h"}} {
		var stdout, stderr bytes.Buffer
		if code := Run(args, &stdout, &stderr); code != exitOK {
			t.Fatalf("args %q: exit = %d, want %d (stderr: %s)", args, code, exitOK, stderr.String())
		}
		if !strings.Contains(stdout.String(), "usage: fake-jev") {
			t.Fatalf("args %q: stdout = %q, want usage", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("args %q: stderr = %q, want empty", args, stderr.String())
		}
	}
}
