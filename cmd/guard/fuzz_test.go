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
