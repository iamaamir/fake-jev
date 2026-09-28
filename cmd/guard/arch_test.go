package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func archTestGuards(rule ArchRule) *Guards {
	return &Guards{
		SkipDirs: []string{"agent", ".agents", ".claude", ".pi", ".scratch"},
		Arch:     ArchGuards{Forbidden: []ArchRule{rule}},
	}
}

func TestRunArchViolation(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "engine/engine.go",
		"package engine\n\nimport _ \"net/http\"\n")
	g := archTestGuards(ArchRule{
		Pkg:         "engine",
		ForbidExact: []string{"net/http", "os/exec"},
	})
	findings, stats, err := runArch(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want exactly 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.arch.forbidden_import" || f.Path != "engine" ||
		f.Line != 0 || !strings.Contains(f.Message, "net/http") {
		t.Fatalf("unexpected finding: %+v", f)
	}
	if stats["packages"] != 1 || stats["files"] != 1 {
		t.Fatalf("stats = %v, want packages=1 files=1", stats)
	}
}

func TestRunArchPrefixUsesRawPrefixSemantics(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go",
		"package m\n\nimport _ \"example.test/m/secret\"\nimport _ \"example.test/m/secret/v1\"\nimport _ \"example.test/m/secretsafe\"\n")
	// The imported packages must exist on disk: a kept package whose
	// imports do not resolve is a load error (B1), so phantom imports
	// would exit 2 before the prefix semantics under test were observable.
	writeModuleFile(t, dir, "secret/secret.go", "package secret\n")
	writeModuleFile(t, dir, "secret/v1/v1.go", "package v1\n")
	writeModuleFile(t, dir, "secretsafe/secretsafe.go", "package secretsafe\n")
	g := archTestGuards(ArchRule{
		Pkg:          ".",
		ForbidPrefix: []string{"example.test/m/secret/"},
	})
	findings, _, err := runArch(g)
	if err != nil {
		t.Fatal(err)
	}
	// trailing-slash family matches .../secret/v1 only: the bare package and
	// the sibling secretsafe are legal (interpretation note 8).
	if len(findings) != 1 ||
		!strings.Contains(findings[0].Message, "example.test/m/secret/v1") {
		t.Fatalf("findings = %+v, want only .../secret/v1", findings)
	}
}

func TestRunArchExactCoversWholeStringOnly(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go",
		"package m\n\nimport _ \"net/http\"\nimport _ \"net/http/httptest\"\n")
	g := archTestGuards(ArchRule{
		Pkg:         ".",
		ForbidExact: []string{"net/http"},
	})
	findings, _, err := runArch(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 ||
		!strings.Contains(findings[0].Message, `"net/http"`) {
		t.Fatalf("findings = %+v, want only the exact net/http import", findings)
	}
}

func TestRunArchSkippedDirNeverMatchesRule(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	writeModuleFile(t, dir, "agent/agent.go",
		"package agent\n\nimport _ \"net/http\"\n")
	g := archTestGuards(ArchRule{Pkg: "agent", ForbidExact: []string{"net/http"}})
	findings, stats, err := runArch(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["packages"] != 1 {
		t.Fatalf("findings=%+v stats=%v, want skipped dir invisible", findings, stats)
	}
}

func TestRunArchThroughCLI(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "engine/engine.go",
		"package engine\n\nimport _ \"os/exec\"\n")
	full := &Guards{
		SkipDirs: []string{"agent", ".agents", ".claude", ".pi", ".scratch"},
		Arch: ArchGuards{Forbidden: []ArchRule{{
			Pkg: "engine", ForbidExact: []string{"net/http", "os/exec"},
			ForbidPrefix: []string{"fake-jev/internal/compat/jev",
				"fake-jev/internal/cli", "fake-jev/internal/control"},
		}}},
		Lint: LintGuards{
			Rules: []string{"err_discarded", "unsafe_type_assert"},
		},
		Complexity: ComplexityGuards{CyclomaticMax: 15, CrapMax: 50},
		Mutation: MutationGuards{OverallMinPct: 75, PerPackageMinPct: 60,
			MutantTimeoutSeconds: 60},
		Trace: TraceGuards{Active: []string{}, Catalog: "docs/catalog.md"},
		Fuzz:  FuzzGuards{Fuzztime: "30s"},
	}
	raw, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	writeGuardsFile(t, dir, string(raw))
	var out, errb strings.Builder
	if code := run([]string{"arch"}, &out, &errb); code != 1 {
		t.Fatalf("exit %d, want 1 (findings); stderr %q", code, errb.String())
	}
	var rep Report
	if err := json.Unmarshal([]byte(out.String()), &rep); err != nil {
		t.Fatalf("stdout is not a report: %v\n%s", err, out.String())
	}
	if rep.Check != "arch" || len(rep.Findings) != 1 ||
		rep.Findings[0].Code != "guard.arch.forbidden_import" {
		t.Fatalf("report = %+v", rep)
	}
}

func TestRunArchCleanFixtureExitsZero(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n")
	full := &Guards{
		SkipDirs: []string{"agent"},
		Arch: ArchGuards{Forbidden: []ArchRule{{
			Pkg: "internal/engine", ForbidExact: []string{"net/http"},
		}}},
		Lint: LintGuards{
			Rules: []string{"err_discarded", "unsafe_type_assert"},
		},
		Complexity: ComplexityGuards{CyclomaticMax: 15, CrapMax: 50},
		Mutation: MutationGuards{OverallMinPct: 75, PerPackageMinPct: 60,
			MutantTimeoutSeconds: 60},
		Trace: TraceGuards{Active: []string{}, Catalog: "docs/catalog.md"},
		Fuzz:  FuzzGuards{Fuzztime: "30s"},
	}
	raw, err := json.Marshal(full)
	if err != nil {
		t.Fatal(err)
	}
	writeGuardsFile(t, dir, string(raw))
	var out, errb strings.Builder
	if code := run([]string{"arch"}, &out, &errb); code != 0 {
		t.Fatalf("exit %d, want 0; stderr %q", code, errb.String())
	}
	if !strings.Contains(out.String(), `"check": "arch"`) ||
		!strings.Contains(out.String(), `"findings": []`) {
		t.Fatalf("stdout = %s", out.String())
	}
}

func TestRunArchDependencyErrorExitsTwo(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "pkg.go", "package m\n\nimport _ \"example.test/nope\"\n")
	writeGuardsFile(t, dir, validGuardsJSON)
	var out, errb strings.Builder
	if code := run([]string{"arch"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2 (kept package with dependency load error)", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty on exit 2, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "example.test/nope") {
		t.Fatalf("stderr %q must name the unresolvable import", errb.String())
	}
}

func TestRunArchOutsideModuleFailsClosed(t *testing.T) {
	// A trailing comment on the go.mod module directive parses fine for go
	// but makes readModulePath return "example.test/m // hijack", which is
	// no import path's prefix: TrimPrefix would no-op, the root rule "."
	// would silently miss, and the scan would exit 0 (I2 fail-open).
	dir := t.TempDir()
	t.Chdir(dir)
	writeModuleFile(t, dir, "go.mod", "module example.test/m // hijack\n\ngo 1.26\n")
	writeModuleFile(t, dir, "pkg.go", "package m\n\nimport _ \"net/http\"\n")
	g := archTestGuards(ArchRule{Pkg: ".", ForbidExact: []string{"net/http"}})
	findings, _, err := runArch(g)
	if err == nil || !strings.Contains(err.Error(), "outside module") {
		t.Fatalf("want outside-module error, got findings=%+v err=%v", findings, err)
	}
}
