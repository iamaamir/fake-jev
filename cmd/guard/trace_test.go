package main

import (
	"fmt"
	"strings"
	"testing"
)

const traceTestCatalog = "| ID | Scope |\n| --- | --- |\n" +
	"| C-GOLD-001 | first |\n| C-GOLD-002 | second |\n"

// unknownTraceID is assembled from two literals on purpose: this file is a
// `_test.go` that the repository-level `guard trace` scan walks, and a
// literal unknown ID here would make the tool fail its own unknown_id floor.
var unknownTraceID = "C-" + "ZZZZ-999"

func writeTraceFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	writeModuleFile(t, dir, "docs/development/acceptance-catalog.md",
		traceTestCatalog)
	for rel, body := range files {
		writeModuleFile(t, dir, rel, body)
	}
	return dir
}

func traceGuards(active ...string) *Guards {
	return &Guards{Trace: TraceGuards{
		Active:  active,
		Catalog: "docs/development/acceptance-catalog.md",
	}}
}

// markerTestFile builds a _test.go whose line 6 carries the marker.
func markerTestFile(marker string) string {
	return fmt.Sprintf("package p\n\nimport \"testing\"\n\n"+
		"func TestX(t *testing.T) {\n\tt.Run(%q, func(t *testing.T) {})\n}\n",
		marker)
}

func TestRunTraceEmptyActivePass(t *testing.T) {
	t.Chdir(writeTraceFixture(t, nil))
	findings, stats, err := runTrace(traceGuards())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["active"] != 0 ||
		stats["test_files"] != 0 || stats["covered"] != 0 {
		t.Fatalf("findings=%v stats=%v, want empty pass", findings, stats)
	}
}

func TestRunTraceUntracedActive(t *testing.T) {
	t.Chdir(writeTraceFixture(t, nil))
	findings, _, err := runTrace(traceGuards("C-GOLD-001"))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.trace.untraced" || f.Path != ".agent/guards.json" ||
		f.Line != 0 || !strings.Contains(f.Message, "C-GOLD-001") {
		t.Fatalf("finding = %+v", f)
	}
}

func TestRunTraceKnownMarkerCovers(t *testing.T) {
	dir := writeTraceFixture(t, map[string]string{
		"p/x_test.go": markerTestFile("C-GOLD-001/choice"),
	})
	t.Chdir(dir)
	findings, stats, err := runTrace(traceGuards("C-GOLD-001"))
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["covered"] != 1 || stats["test_files"] != 1 {
		t.Fatalf("findings=%v stats=%v, want covered pass", findings, stats)
	}
}

func TestRunTraceUnknownMarker(t *testing.T) {
	dir := writeTraceFixture(t, map[string]string{
		"p/x_test.go": markerTestFile(unknownTraceID + "/never"),
	})
	t.Chdir(dir)
	findings, _, err := runTrace(traceGuards())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.trace.unknown_id" || f.Path != "p/x_test.go" ||
		f.Line != 6 || !strings.Contains(f.Message, unknownTraceID) {
		t.Fatalf("finding = %+v", f)
	}
}

func TestRunTraceRawStringMarkerLineExact(t *testing.T) {
	// Physical layout: 1 package p | 2 blank | 3 var s = ` | 4 padding |
	// 5 marker | 6 closing backtick.
	body := "package p\n\nvar s = `\npadding\n" +
		unknownTraceID + "/raw\n`\n"
	dir := writeTraceFixture(t, map[string]string{"p/raw_test.go": body})
	t.Chdir(dir)
	findings, _, err := runTrace(traceGuards())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.trace.unknown_id" || f.Path != "p/raw_test.go" ||
		f.Line != 5 || !strings.Contains(f.Message, unknownTraceID) {
		t.Fatalf("finding = %+v, want raw-string marker at line 5", f)
	}
}

func TestRunTraceEscapedNewlineMarkerUsesPhysicalLine(t *testing.T) {
	// The literal starts and ends on physical line 3; `\n` is an escape
	// sequence, not a source newline, so the marker must report line 3.
	body := "package p\n\nvar s = \"pre\\npost " +
		unknownTraceID + "/esc\"\n"
	dir := writeTraceFixture(t, map[string]string{"p/esc_test.go": body})
	t.Chdir(dir)
	findings, _, err := runTrace(traceGuards())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.trace.unknown_id" || f.Path != "p/esc_test.go" ||
		f.Line != 3 || !strings.Contains(f.Message, unknownTraceID) {
		t.Fatalf("finding = %+v, want escaped-newline marker at line 3", f)
	}
}

func TestRunTraceTestdataCoversWithoutUnknownCheck(t *testing.T) {
	dir := writeTraceFixture(t, map[string]string{
		"testdata/contracts/v.json": fmt.Sprintf(`{"tags": ["C-GOLD-002", "%s"]}`, unknownTraceID),
	})
	t.Chdir(dir)
	findings, stats, err := runTrace(traceGuards("C-GOLD-002"))
	if err != nil {
		t.Fatal(err)
	}
	// Testdata tags are coverage only: the unknown tag must not be flagged
	// (note 5 — five real vector tags have no catalog row).
	if len(findings) != 0 || stats["covered"] != 2 {
		t.Fatalf("findings=%v stats=%v, want testdata coverage, no findings",
			findings, stats)
	}
}

func TestRunTraceMissingCatalog(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, _, err := runTrace(traceGuards()); err == nil ||
		!strings.Contains(err.Error(), "trace: read catalog") {
		t.Fatalf("want catalog read error, got %v", err)
	}
}

func TestRunTraceRowlessCatalog(t *testing.T) {
	dir := writeTraceFixture(t, nil)
	writeModuleFile(t, dir, "docs/development/acceptance-catalog.md",
		"| ID | Scope |\n| --- | --- |\n")
	t.Chdir(dir)
	if _, _, err := runTrace(traceGuards()); err == nil ||
		!strings.Contains(err.Error(), "has no C-ID rows") {
		t.Fatalf("want rowless catalog error, got %v", err)
	}
}

func TestRunTraceIgnoresDotAndSkipDirs(t *testing.T) {
	body := markerTestFile(unknownTraceID + "/never")
	dir := writeTraceFixture(t, map[string]string{
		".git/x_test.go":  body,
		"agent/y_test.go": body,
	})
	t.Chdir(dir)
	g := traceGuards()
	g.SkipDirs = []string{"agent"}
	findings, stats, err := runTrace(g)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["test_files"] != 0 {
		t.Fatalf("findings=%v stats=%v, want dot/skip dirs ignored",
			findings, stats)
	}
}

func TestRunTraceTestdataFilesNotMarkerSources(t *testing.T) {
	body := fmt.Sprintf("package p\n\nimport \"testing\"\n\n"+
		"func TestX(t *testing.T) {\n\tt.Run(%q, func(t *testing.T) {})\n"+
		"\tt.Run(%q, func(t *testing.T) {})\n}\n",
		"C-GOLD-001/choice", unknownTraceID+"/never")
	dir := writeTraceFixture(t, map[string]string{
		"testdata/sample_test.go": body,
	})
	t.Chdir(dir)
	findings, stats, err := runTrace(traceGuards("C-GOLD-001"))
	if err != nil {
		t.Fatal(err)
	}
	// Files under testdata/ (contracts excepted) are never executed by
	// `go test`, so they neither cover an active ID nor raise unknown_id.
	if len(findings) != 1 {
		t.Fatalf("findings = %v, want only untraced C-GOLD-001", findings)
	}
	f := findings[0]
	if f.Code != "guard.trace.untraced" || f.Path != ".agent/guards.json" ||
		f.Line != 0 || !strings.Contains(f.Message, "C-GOLD-001") {
		t.Fatalf("finding = %+v", f)
	}
	if stats["active"] != 1 || stats["test_files"] != 0 ||
		stats["covered"] != 0 {
		t.Fatalf("stats = %v, want testdata excluded (test_files=0 covered=0)",
			stats)
	}
}

func TestRunTraceSkipsUnderscorePaths(t *testing.T) {
	body := markerTestFile(unknownTraceID + "/never")
	dir := writeTraceFixture(t, map[string]string{
		"_ignored/sample_test.go": body,
		"_legacy_test.go":         body,
	})
	t.Chdir(dir)
	findings, stats, err := runTrace(traceGuards())
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 || stats["test_files"] != 0 || stats["covered"] != 0 {
		t.Fatalf("findings=%v stats=%v, want underscore paths skipped",
			findings, stats)
	}
}

func TestRunTraceStats(t *testing.T) {
	dir := writeTraceFixture(t, map[string]string{
		"p/x_test.go": markerTestFile("C-GOLD-001/choice"),
	})
	t.Chdir(dir)
	_, stats, err := runTrace(traceGuards("C-GOLD-001"))
	if err != nil {
		t.Fatal(err)
	}
	if stats["active"] != 1 || stats["covered"] != 1 ||
		stats["test_files"] != 1 {
		t.Fatalf("stats = %v, want active=1 covered=1 test_files=1", stats)
	}
}
