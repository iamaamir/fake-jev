package cli

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// validYAML and validJSON encode the same logical configuration in the two
// accepted input formats (specification §12): both must load and validate.
const (
	validYAML = "schemaVersion: 1\nmode: strict\nstubs:\n  - id: hello\n    profile: jev/v1\n    when: {}\n    then:\n      answers:\n        urgent:\n          noul: 0.95\n"
	validJSON = `{"schemaVersion":1,"mode":"strict","stubs":[{"id":"hello","profile":"jev/v1","when":{},"then":{"answers":{"urgent":{"noul":0.95}}}}]}`
)

// validateSuccessLine is the success output the command writes to stdout.
// The text itself is not a contract (§15.2); the exit code is.
func validateSuccessLine(path string) string {
	return "fake-jev: " + path + ": configuration is valid\n"
}

func TestValidateAcceptsValidConfigurationFiles(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
	}{
		{"yaml", "config.yaml", validYAML},
		{"json", "config.json", validJSON},
		{"defaults only", "defaults.yaml", "schemaVersion: 1\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.file)
			if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}

			var stdout, stderr bytes.Buffer
			code := Run([]string{"validate", path}, &stdout, &stderr)

			if code != exitOK {
				t.Fatalf("exit = %d, want %d (stderr: %s)", code, exitOK, stderr.String())
			}
			if stdout.String() != validateSuccessLine(path) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), validateSuccessLine(path))
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestValidateRejectsUsageFailuresAndInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		file    string   // file or directory inside the temp dir; empty means no path argument
		extra   []string // arguments after the path
		write   bool     // create file with content before invoking validate
		dir     bool     // create file as a directory instead of a regular file
		content string
		usage   bool // usage failure: stderr must carry the usage text too
	}{
		{name: "missing path", usage: true},
		{name: "extra argument", file: "config.yaml", write: true, content: validYAML, extra: []string{"second.yaml"}, usage: true},
		{name: "unknown flag", extra: []string{"--strict"}, usage: true},
		{name: "unknown flag after path", file: "config.yaml", write: true, content: validYAML, extra: []string{"--strict"}, usage: true},
		{name: "help flag after command", extra: []string{"--help"}, usage: true},
		{name: "missing file", file: "absent.yaml"},
		{name: "directory path", file: "adir", dir: true},
		{name: "empty file", file: "empty.yaml", write: true},
		{name: "utf-8 bom with unknown key", file: "bom.yaml", write: true, content: "\ufeffschemaVersion: 1\nextra: true\n"},
		{name: "malformed yaml", file: "broken.yaml", write: true, content: "schemaVersion: [1\n"},
		{name: "malformed json", file: "broken.json", write: true, content: `{"schemaVersion": 1`},
		{name: "multiple documents", file: "multi.yaml", write: true, content: "---\nschemaVersion: 1\n---\nschemaVersion: 1\n"},
		{name: "missing schemaVersion", file: "noschema.yaml", write: true, content: "mode: strict\n"},
		{name: "unknown top-level key", file: "top.yaml", write: true, content: "schemaVersion: 1\nextra: true\n"},
		{name: "unknown nested key", file: "nested.yaml", write: true, content: "schemaVersion: 1\nserver:\n  nope: true\n"},
		{name: "non strict mode", file: "mode.yaml", write: true, content: "schemaVersion: 1\nmode: permissive\n"},
		{name: "unknown profile", file: "profile.yaml", write: true, content: "schemaVersion: 1\ncompatibility: [other]\n"},
		{name: "port out of range", file: "port.yaml", write: true, content: "schemaVersion: 1\nserver: {port: 70000}\n"},
		{name: "wrong type for stubs", file: "stubs.yaml", write: true, content: "schemaVersion: 1\nstubs: 3\n"},
		{name: "duplicate stub id", file: "duplicate.yaml", write: true,
			content: "schemaVersion: 1\nstubs:\n  - {id: x, profile: jev/v1, when: {}, then: {raw: {status: 200, body: null}}}\n  - {id: x, profile: jev/v1, when: {}, then: {raw: {status: 200, body: null}}}\n"},
		{name: "invalid expect", file: "expect.yaml", write: true,
			content: "schemaVersion: 1\nstubs:\n  - id: x\n    profile: jev/v1\n    when: {}\n    then: {answers: {urgent: {noul: 1}}}\n    expect: {exactly: 1, atLeast: 1}\n"},
		{name: "invalid response form", file: "form.yaml", write: true,
			content: "schemaVersion: 1\nstubs:\n  - id: x\n    profile: jev/v1\n    when: {}\n    then: {answers: {urgent: {noul: 1}}, raw: {status: 200, body: null}}\n"},
		{name: "missing response form", file: "empty-then.yaml", write: true,
			content: "schemaVersion: 1\nstubs:\n  - id: x\n    profile: jev/v1\n    when: {}\n    then: {}\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), test.file)
			switch {
			case test.dir:
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			case test.write:
				if err := os.WriteFile(path, []byte(test.content), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			args := []string{"validate"}
			if test.file != "" {
				args = append(args, path)
			}
			args = append(args, test.extra...)

			var stdout, stderr bytes.Buffer
			code := Run(args, &stdout, &stderr)

			if code != exitFailure {
				t.Fatalf("args %q: exit = %d, want %d", args, code, exitFailure)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty: diagnostics and usage belong on stderr", stdout.String())
			}
			if stderr.Len() == 0 {
				t.Fatal("stderr is empty, want a diagnostic")
			}
			if got := strings.Contains(stderr.String(), "usage: fake-jev"); got != test.usage {
				t.Fatalf("stderr carries usage = %v, want %v\nstderr: %s", got, test.usage, stderr.String())
			}
		})
	}
}

// TestValidateStreamsAreDisjoint pins the stream split (§42.5): a run writes
// a success line only on stdout, and a rejection writes only on stderr.
func TestValidateStreamsAreDisjoint(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.yaml")
	if err := os.WriteFile(good, []byte(validYAML), 0o600); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("schemaVersion: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var successOut, successErr bytes.Buffer
	if code := Run([]string{"validate", good}, &successOut, &successErr); code != exitOK {
		t.Fatalf("valid file: exit = %d, want %d", code, exitOK)
	}
	if successOut.Len() == 0 || successErr.Len() != 0 {
		t.Fatalf("valid file: stdout = %q, stderr = %q", successOut.String(), successErr.String())
	}

	var failureOut, failureErr bytes.Buffer
	if code := Run([]string{"validate", bad}, &failureOut, &failureErr); code != exitFailure {
		t.Fatalf("invalid file: exit = %d, want %d", code, exitFailure)
	}
	if failureOut.Len() != 0 || failureErr.Len() == 0 {
		t.Fatalf("invalid file: stdout = %q, stderr = %q", failureOut.String(), failureErr.String())
	}
}

func TestUsageListsValidateAsACommand(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run([]string{arg}, &stdout, &stderr); code != exitOK {
				t.Fatalf("exit = %d, want %d", code, exitOK)
			}
			out := stdout.String()
			if !strings.Contains(out, "\n  validate <path> ") {
				t.Fatalf("usage does not list validate as a command:\n%s", out)
			}
			if strings.Contains(out, "serve, run, validate, verify") {
				t.Fatalf("usage still lists validate as a future command:\n%s", out)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

// TestValidateRejectsUnreadableFile covers the read failure that is not a
// missing file: the path exists but cannot be opened. Permission bits do not
// deny reads to a privileged user, so that case is skipped rather than asserted.
func TestValidateRejectsUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permission bits do not deny reads")
	}
	path := filepath.Join(t.TempDir(), "unreadable.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0o000); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"validate", path}, &stdout, &stderr)
	if code != exitFailure {
		t.Fatalf("exit = %d, want %d", code, exitFailure)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "read configuration") {
		t.Fatalf("stderr = %q, want a read diagnostic", stderr.String())
	}
}

// TestValidateOpensNoListener pins the no-listener half of §15.2 behaviorally:
// the configured server port is held open for the whole run, so a command that
// tried to bind it would fail. validate must still report success for a valid
// file and the ordinary configuration diagnostic for an invalid one. The
// listener is loopback-only and ephemeral, so the test stays hermetic.
func TestValidateOpensNoListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close listener: %v", err)
		}
	}()
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not a TCP address", listener.Addr())
	}

	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.yaml")
	if err := os.WriteFile(valid, []byte(fmt.Sprintf("schemaVersion: 1\nserver: {port: %d}\n", addr.Port)), 0o600); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(invalid, []byte(fmt.Sprintf("schemaVersion: 1\nserver: {port: %d}\nextra: true\n", addr.Port)), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := Run([]string{"validate", valid}, &stdout, &stderr); code != exitOK {
		t.Fatalf("valid file: exit = %d, want %d (stderr: %s)", code, exitOK, stderr.String())
	}
	if stdout.String() != validateSuccessLine(valid) || stderr.Len() != 0 {
		t.Fatalf("valid file: stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}

	var failureOut, failureErr bytes.Buffer
	if code := Run([]string{"validate", invalid}, &failureOut, &failureErr); code != exitFailure {
		t.Fatalf("invalid file: exit = %d, want %d", code, exitFailure)
	}
	if failureOut.Len() != 0 || failureErr.Len() == 0 {
		t.Fatalf("invalid file: stdout = %q, stderr = %q", failureOut.String(), failureErr.String())
	}
}

// modulePath is the Go module path prefix that marks this repository's own
// packages in an import path.
const modulePath = "fake-jev"

// TestValidatePathLinksNoNetworkingPackage proves the §15.2/§19.2 guarantee
// structurally: it walks the import closure that starts at validate.go, the
// only production file implementing the validate path, and fails if any file
// on that path imports net or a net/... package, so the command cannot dial or
// listen. Only that closure is checked, so unrelated files in package cli — a
// later `verify` client, for example — are not constrained. The test parses
// sources only and stays hermetic: it opens no socket and starts no process.
func TestValidatePathLinksNoNetworkingPackage(t *testing.T) {
	const seed = "validate.go"

	scanned := 0
	queue := []string{seed}
	visited := map[string]bool{seed: true}
	for len(queue) > 0 {
		unit := queue[0]
		queue = queue[1:]

		files := []string{unit}
		if info, err := os.Stat(unit); err == nil && info.IsDir() {
			entries, err := os.ReadDir(unit)
			if err != nil {
				t.Fatalf("read %s: %v", unit, err)
			}
			files = nil
			for _, entry := range entries {
				name := entry.Name()
				if !entry.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
					files = append(files, filepath.Join(unit, name))
				}
			}
			if len(files) == 0 {
				t.Fatalf("no production Go files found in %s", unit)
			}
		}

		for _, file := range files {
			scanned++
			imports, err := fileImports(file)
			if err != nil {
				t.Fatalf("imports of %s: %v", file, err)
			}
			for _, imported := range imports {
				if imported == "net" || strings.HasPrefix(imported, "net/") {
					t.Errorf("%s imports %q: the validate path must not be able to dial or listen", file, imported)
					continue
				}
				if !strings.HasPrefix(imported, modulePath+"/") {
					continue
				}
				dir := filepath.Join("..", "..", filepath.FromSlash(strings.TrimPrefix(imported, modulePath+"/")))
				if !visited[dir] {
					visited[dir] = true
					queue = append(queue, dir)
				}
			}
		}
	}
	if scanned == 0 {
		t.Fatal("no production Go files found on the validate path")
	}
}

// fileImports returns the import paths of one Go source file.
func fileImports(file string) ([]string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(parsed.Imports))
	for _, spec := range parsed.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, fmt.Errorf("import path %s: %w", spec.Path.Value, err)
		}
		imports = append(imports, path)
	}
	return imports, nil
}
