package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeModuleFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func chdirModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	writeModuleFile(t, dir, "go.mod", "module example.test/m\n\ngo 1.26\n")
	return dir
}

func TestListPackagesSkipsToolDirsBeforeErrors(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	// agent/ matches the skip list; its load error (mixed package names)
	// must never surface, proving skip happens before the Error check.
	writeModuleFile(t, dir, "agent/a.go", "package agent\n")
	writeModuleFile(t, dir, "agent/b.go", "package agentwrong\n")
	pkgs, err := listPackages("example.test/m", []string{"agent", ".claude"})
	if err != nil {
		t.Fatalf("skipped dirs must not surface load errors: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].ImportPath != "example.test/m" {
		t.Fatalf("pkgs = %+v, want only the kept root package", pkgs)
	}
}

func TestListPackagesKeptLoadErrorFails(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	// go list does not parse function bodies, so a body syntax error is
	// invisible to it; mixed package names in one directory are a real load
	// error it must report (verified empirically: exit 0 + Error field).
	writeModuleFile(t, dir, "bad/a.go", "package ok\n")
	writeModuleFile(t, dir, "bad/b.go", "package wrong\n")
	if _, err := listPackages("example.test/m", []string{"agent"}); err == nil ||
		!strings.Contains(err.Error(), "found packages") {
		t.Fatalf("want kept-package load error, got %v", err)
	}
}

func TestListPackagesSortedWithFields(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	writeModuleFile(t, dir, "b/b.go", "package b\nimport _ \"strings\"\n")
	writeModuleFile(t, dir, "a/a.go", "package a\nimport _ \"net/http\"\n")
	pkgs, err := listPackages("example.test/m", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 3 || pkgs[0].ImportPath != "example.test/m" ||
		pkgs[1].ImportPath != "example.test/m/a" || pkgs[2].ImportPath != "example.test/m/b" {
		t.Fatalf("pkgs = %+v, want root/a/b sorted", pkgs)
	}
	if got := pkgs[1].allImports(); len(got) != 1 || got[0] != "net/http" {
		t.Fatalf("allImports = %v, want [net/http]", got)
	}
	if pkgs[1].fileCount() != 1 {
		t.Fatalf("fileCount = %d, want 1", pkgs[1].fileCount())
	}
	if rel := moduleRel("example.test/m", "example.test/m/a"); rel != "a" {
		t.Fatalf("moduleRel = %q, want a", rel)
	}
	if rel := moduleRel("example.test/m", "example.test/m"); rel != "." {
		t.Fatalf("moduleRel root = %q, want .", rel)
	}
}

func TestReadModulePathMissingFails(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, err := readModulePath(); err == nil {
		t.Fatal("want error without go.mod")
	}
}

func TestListPackagesDependencyErrorFails(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n\nimport _ \"example.test/nope\"\n")
	// go list -e exits 0 and leaves Error null for an unresolvable import;
	// DepsErrors carries it instead (go1.26). A kept package whose
	// dependency cannot load must fail closed, never scan as violation-free.
	if _, err := listPackages("example.test/m", nil); err == nil ||
		!strings.Contains(err.Error(), "example.test/nope") {
		t.Fatalf("want dependency load error naming the import, got %v", err)
	}
}

func TestListPackagesSkipsOnlyRootSegment(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	writeModuleFile(t, dir, "agent/a.go", "package agent\n")
	writeModuleFile(t, dir, "internal/agent/a.go", "package agent\n")
	// verify's NON_REPO_DIRS is root-anchored: a nested directory that
	// merely shares a tool-dir name stays in the scan (I1 fail-open).
	pkgs, err := listPackages("example.test/m", []string{"agent"})
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, p := range pkgs {
		paths = append(paths, p.ImportPath)
	}
	if len(paths) != 2 ||
		paths[0] != "example.test/m" || paths[1] != "example.test/m/internal/agent" {
		t.Fatalf("pkgs = %v, want root + internal/agent kept, agent skipped", paths)
	}
}
