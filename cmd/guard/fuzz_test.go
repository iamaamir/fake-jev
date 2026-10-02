package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fuzzGuards(fuzztime string) *Guards {
	return &Guards{
		SkipDirs: []string{"agent", ".agents", ".claude", ".pi", ".scratch"},
		Fuzz:     FuzzGuards{Fuzztime: fuzztime},
	}
}

func TestDiscoverFuzzTargetsFiltersAndSorts(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "a/a.go", "package a\n")
	writeModuleFile(t, dir, "a/a_test.go",
		"package a\n\nimport \"testing\"\n\ntype recv struct{}\n\n"+
			"func FuzzOne(f *testing.F) {}\n"+
			"func FuzzSix(f *testing.F) {}\n"+
			"func (recv) FuzzTwo(f *testing.F) {}\n"+
			"func FuzzThree(f *testing.F) (int, error) { return 0, nil }\n"+
			"func FuzzFour(f, g *testing.F) {}\n"+
			"func FuzzFive(f *testing.T) {}\n"+
			"func NotFuzz(f *testing.F) {}\n")
	writeModuleFile(t, dir, "b/b.go", "package b\n")
	writeModuleFile(t, dir, "b/b_x_test.go",
		"package b_test\n\nimport \"testing\"\n\nfunc FuzzExt(f *testing.F) {}\n")
	targets, err := discoverFuzzTargets("example.test/m", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Sorted by (ImportPath, Name): internal a's FuzzOne/FuzzSix, then the
	// external b_test target. The receiver, results, multi-name parameter and
	// *testing.T variants are all rejected; NotFuzz fails the name prefix.
	want := []string{"example.test/m/a FuzzOne", "example.test/m/a FuzzSix",
		"example.test/m/b FuzzExt"}
	if len(targets) != len(want) {
		t.Fatalf("targets = %+v, want %v", targets, want)
	}
	for i, tgt := range targets {
		if got := tgt.ImportPath + " " + tgt.Name; got != want[i] {
			t.Fatalf("target[%d] = %q, want %q", i, got, want[i])
		}
	}
	// a_test.go: line 5 is `type recv struct{}`, line 7 is FuzzOne.
	if targets[0].File != "a/a_test.go" || targets[0].Line != 7 {
		t.Fatalf("declaring position = %s:%d, want a/a_test.go:7",
			targets[0].File, targets[0].Line)
	}
}

func TestRunFuzzRejectsZeroFuzztime(t *testing.T) {
	// "0s" is a well-formed guards.json pattern (validation passes it), so
	// runFuzz itself must reject a non-positive window before touching the
	// filesystem or spawning go - the fuzz budget floor is fail-closed.
	if _, _, err := runFuzz(fuzzGuards("0s")); err == nil ||
		!strings.Contains(err.Error(), "fuzz.fuzztime") {
		t.Fatalf("want positive fuzztime error, got %v", err)
	}
}

func TestRunFuzzZeroTargets(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	findings, stats, err := runFuzz(fuzzGuards("30s"))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["targets"] != 0 {
		t.Fatalf("findings=%v stats=%v, want zero-target vacuous pass",
			findings, stats)
	}
}

func TestRunFuzzSeedCrash(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport \"testing\"\n\nfunc FuzzBoom(f *testing.F) {\n"+
			"\tf.Add(\"boom\")\n"+
			"\tf.Fuzz(func(t *testing.T, s string) {\n"+
			"\t\tif s == \"boom\" {\n\t\t\tpanic(\"boom\")\n\t\t}\n"+
			"\t})\n}\n")
	findings, stats, err := runFuzz(fuzzGuards("30s"))
	if err != nil {
		t.Fatal(err)
	}
	if stats["targets"] != 1 || len(findings) != 1 {
		t.Fatalf("findings=%v stats=%v, want one seed-corpus crash",
			findings, stats)
	}
	// go1.26.3 reports seed crashes with `failure while testing seed corpus
	// entry` and writes no input file (the input is already committed in
	// f.Add), so the finding points at the declaring source, line 5.
	f := findings[0]
	if f.Code != "guard.fuzz.crash" || f.Path != "p/fuzz_p_test.go" ||
		f.Line != 5 || !strings.Contains(f.Message, "FuzzBoom") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestRunFuzzLiveCrash(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	// No f.Add: the seed corpus is empty, so ordinary `go test` never
	// executes the body - only -fuzz mode generates the input that panics.
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport \"testing\"\n\nfunc FuzzAlways(f *testing.F) {\n"+
			"\tf.Fuzz(func(t *testing.T, s string) {\n"+
			"\t\tpanic(\"live\")\n\t})\n}\n")
	findings, stats, err := runFuzz(fuzzGuards("1s"))
	if err != nil {
		t.Fatal(err)
	}
	if stats["targets"] != 1 || len(findings) != 1 {
		t.Fatalf("findings=%v stats=%v, want one live crash", findings, stats)
	}
	f := findings[0]
	if f.Code != "guard.fuzz.crash" ||
		!strings.HasPrefix(f.Path, "p/testdata/fuzz/FuzzAlways/") {
		t.Fatalf("finding = %+v, want preserved input under testdata/fuzz", f)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f.Path))); err != nil {
		t.Fatalf("preserved input %s: %v", f.Path, err)
	}
}

func TestRunFuzzPass(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport \"testing\"\n\nfunc FuzzSafe(f *testing.F) {\n"+
			"\tf.Add(\"x\")\n"+
			"\tf.Fuzz(func(t *testing.T, s string) {})\n}\n")
	findings, stats, err := runFuzz(fuzzGuards("1s"))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["targets"] != 1 {
		t.Fatalf("findings=%v stats=%v, want a clean bounded run",
			findings, stats)
	}
}

// fdPressureOutput is the verbatim shape (go1.26.3, ulimit -n 64) of a
// `go test -fuzz` run whose fuzzing worker could not be started: the testing
// framework's `--- FAIL: <Name>` line followed by a worker-start diagnostic
// and neither corpus-attribution marker. FJ-060 requires this to classify as
// an operational error, never a guard.fuzz.crash finding.
const fdPressureOutput = "" +
	"=== RUN   FuzzSafe\n" +
	"--- FAIL: FuzzSafe (0.01s)\n" +
	"    fork/exec /var/folders/xx/fz.test: too many open files\n" +
	"FAIL\n" +
	"exit status 1\n"

func TestClassifyFuzzFailureFDPressureIsNotCrash(t *testing.T) {
	// The regression: a worker that never started emits a bare `--- FAIL:`
	// line, and no written-input or seed-corpus marker. It must not be read
	// as a target crash.
	kind, input := classifyFuzzFailure(fdPressureOutput)
	if kind != fuzzFailureOperational || input != "" {
		t.Fatalf("fd-pressure output classified as %d (input %q), want operational",
			kind, input)
	}
}

func TestClassifyFuzzFailure(t *testing.T) {
	// Table mirrors the marker catalogue verified verbatim against go1.26.3:
	// only the two corpus-attribution markers are target-level; every worker
	// start/communication/termination diagnostic and a bare `--- FAIL:` line
	// are operational errors.
	tests := []struct {
		name  string
		text  string
		kind  fuzzFailureKind
		input string
	}{
		{
			name: "worker start: fork/exec too many open files",
			text: fdPressureOutput,
			kind: fuzzFailureOperational,
		},
		{
			name: "worker start: pipe too many open files",
			text: "--- FAIL: FuzzSafe (0.01s)\n    pipe: too many open files\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "worker start: open /dev/null too many open files",
			text: "--- FAIL: FuzzSafe (0.01s)\n    open /dev/null: too many open files\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "worker communication failure",
			text: "--- FAIL: FuzzSafe\n    communicating with fuzzing process: EOF\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "worker terminated by unexpected signal",
			text: "--- FAIL: FuzzSafe\n    fuzzing process terminated by unexpected signal; no crash will be recorded: signal: terminated\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "worker terminated without fuzzing",
			text: "--- FAIL: FuzzSafe\n    fuzzing process terminated without fuzzing: EOF\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "worker internal failure",
			text: "--- FAIL: FuzzSafe\n    fuzzing process exited unexpectedly due to an internal failure: boom\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "bare --- FAIL with no marker",
			text: "--- FAIL: FuzzSafe (0.02s)\nFAIL\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "build failure",
			text: "# example.test/m/p\n./p.go:3:1: syntax error\nFAIL\t[build failed]\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "build failure echoing the seed marker",
			// A compiler diagnostic echoes the offending source line; the
			// marker is quoted, not at line start, so it must not match
			// (the package never ran an input).
			text: "# example.test/m/p [example.test/m/p.test]\n" +
				"./fuzz_p_test.go:6:2: \"failure while testing seed corpus entry: FuzzBroken/seed#0\" (untyped string constant) is not used\n" +
				"FAIL\texample.test/m/p [build failed]\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "unrecognized output",
			text: "something entirely unexpected\n",
			kind: fuzzFailureOperational,
		},
		{
			name: "genuine target failure: written input",
			text: "--- FAIL: FuzzSignal (0.04s)\n" +
				"    Failing input written to testdata/fuzz/FuzzSignal/0a1b2c3d\n" +
				"FAIL\nexit status 1\n",
			kind:  fuzzFailureWrittenInput,
			input: "testdata/fuzz/FuzzSignal/0a1b2c3d",
		},
		{
			name: "genuine target failure: seed corpus entry",
			text: "--- FAIL: FuzzBoom\n" +
				"    failure while testing seed corpus entry: FuzzBoom/seed#0\n" +
				"FAIL\nexit status 1\n",
			kind: fuzzFailureSeedCorpus,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			kind, input := classifyFuzzFailure(tc.text)
			if kind != tc.kind || input != tc.input {
				t.Fatalf("classifyFuzzFailure = (%d, %q), want (%d, %q)",
					kind, input, tc.kind, tc.input)
			}
		})
	}
}

func TestFuzzOutputBoundsRetainedBytes(t *testing.T) {
	// The shared writer backs both of the subprocess's standard streams. It
	// must accept every byte (a short write would stall the child pipe) while
	// retaining at most maxFuzzOutput, so a hostile target cannot grow guard
	// memory through its diagnostics. Truncated output still fails closed.
	var out fuzzOutput
	chunk := []byte(strings.Repeat("x", 4096))
	for i := 0; i < maxFuzzOutput/len(chunk)+8; i++ {
		n, err := out.Write(chunk)
		if n != len(chunk) || err != nil {
			t.Fatalf("Write = (%d, %v), want (%d, nil): a short write stalls the child",
				n, err, len(chunk))
		}
	}
	if out.buf.Len() != maxFuzzOutput {
		t.Fatalf("retained %d bytes, want %d", out.buf.Len(), maxFuzzOutput)
	}
	if !out.truncated {
		t.Fatal("truncated flag not set after exceeding the bound")
	}
	if kind, input := classifyFuzzFailure(out.String()); kind != fuzzFailureOperational || input != "" {
		t.Fatalf("truncated output classified as %d (input %q), want operational",
			kind, input)
	}
}

func TestPreserveCrashInputRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	pkg := filepath.Join(dir, "pkg")
	if err := os.Mkdir(pkg, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret-input")
	if err := os.WriteFile(secret, []byte("must not copy"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := preserveCrashInput("example.test/m", fuzzTarget{
		ImportPath: "example.test/m/pkg",
		Dir:        pkg,
		Name:       "FuzzEscape",
	}, "../secret-input")
	if err == nil || !strings.Contains(err.Error(), "escapes package directory") {
		t.Fatalf("want package-boundary error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(pkg, "testdata")); !os.IsNotExist(err) {
		t.Fatalf("escape attempt created corpus directory: %v", err)
	}
}

func TestRunFuzzBuildFailure(t *testing.T) {
	dir := chdirModule(t)
	// A declaration-level syntax error in a NON-test file: go list does not
	// parse it (verified empirically: `go list -e` keeps the package with
	// Err=<nil>), so discovery succeeds and the failure surfaces only when
	// `go test` compiles the package. The output then carries neither crash
	// marker (verified: `# example.test/m/p` + `[build failed]`, exit 1), so
	// it must classify as an exit-2 tool error naming the import path.
	writeModuleFile(t, dir, "p/p.go", "package p\n\nfunc Broken {\n")
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport \"testing\"\n\nfunc FuzzBroken(f *testing.F) {}\n")
	_, _, err := runFuzz(fuzzGuards("1s"))
	if err == nil {
		t.Fatal("want tool error for an uncompilable package")
	}
	if !strings.Contains(err.Error(), "example.test/m/p") ||
		!strings.Contains(err.Error(), "build failed") {
		t.Fatalf("error = %v, want import path and [build failed] tail", err)
	}
}

func TestRunFuzzBuildFailureEchoingSeedMarkerIsNotCrash(t *testing.T) {
	dir := chdirModule(t)
	// The compiler echoes the offending source line, so the seed marker
	// appears in the build output as a quoted string literal, not as go's
	// marker line (verified verbatim: `./fuzz_p_test.go:6:2: "failure while
	// testing seed corpus entry: ..." (untyped string constant) is not
	// used`). The package never ran an input, so this must stay an exit-2
	// operational error (AC-10), never a fabricated guard.fuzz.crash finding.
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport \"testing\"\n\nfunc FuzzBroken(f *testing.F) {\n"+
			"\t\"failure while testing seed corpus entry: FuzzBroken/seed#0\"\n}\n")
	_, _, err := runFuzz(fuzzGuards("1s"))
	if err == nil {
		t.Fatal("build failure must be an operational error, not a crash finding")
	}
	if !strings.Contains(err.Error(), "example.test/m/p") {
		t.Fatalf("error = %v, want the import path", err)
	}
}

func TestRunFuzzWorkerStartFailureIsNotCrash(t *testing.T) {
	dir := chdirModule(t)
	// Deterministic stand-in for the fd-pressure reproduction (go1.26.3,
	// ulimit -n 64): the testing framework's `--- FAIL: <Name>` plus a
	// worker-start diagnostic and neither corpus-attribution marker. TestMain
	// emits those exact bytes and exits non-zero without exhausting file
	// descriptors in the test process, so the whole runFuzz -> runOneFuzz ->
	// classifyFuzzFailure path is exercised. Before FJ-060 this shape produced
	// a fabricated `guard.fuzz.crash` at p/fuzz_p_test.go:5; a bare
	// `--- FAIL: <Name>` must now be an exit-2 operational error.
	writeModuleFile(t, dir, "p/p.go", "package p\n")
	writeModuleFile(t, dir, "p/fuzz_p_test.go",
		"package p\n\nimport (\n\t\"fmt\"\n\t\"os\"\n\t\"testing\"\n)\n\n"+
			"func TestMain(m *testing.M) {\n"+
			"\tfmt.Println(\"--- FAIL: FuzzSafe (0.01s)\")\n"+
			"\tfmt.Fprintln(os.Stderr, \"    fork/exec /x/fz.test: too many open files\")\n"+
			"\tos.Exit(1)\n}\n\n"+
			"func FuzzSafe(f *testing.F) {\n\tf.Add(\"x\")\n\tf.Fuzz(func(t *testing.T, s string) {})\n}\n")
	_, _, err := runFuzz(fuzzGuards("1s"))
	if err == nil {
		t.Fatal("worker-start failure must be an operational error, not a clean run")
	}
	if !strings.Contains(err.Error(), "example.test/m/p") {
		t.Fatalf("error = %v, want the import path", err)
	}
}
