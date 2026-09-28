package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func lintFullGuards() *Guards {
	return &Guards{
		SkipDirs: []string{"agent", ".agents", ".claude", ".pi", ".scratch"},
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
}

func runLintFixture(t *testing.T, files map[string]string) ([]Finding, error) {
	t.Helper()
	dir := chdirModule(t)
	for rel, body := range files {
		writeModuleFile(t, dir, rel, body)
	}
	findings, _, err := runLint(lintFullGuards())
	return findings, err
}

func TestLintDiscardedStatement(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func f() error { return nil }

func g() {
	f()
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want 1", findings)
	}
	f := findings[0]
	if f.Code != "guard.lint.err_discarded" || f.Path != "m.go" ||
		f.Line != 6 || f.Message != "f: error result discarded as statement" {
		t.Fatalf("unexpected finding: %+v", f)
	}
}

func TestLintBlankAssignDiscard(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func f() (int, error) { return 0, nil }

func g() {
	_, _ = f()
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 ||
		findings[0].Code != "guard.lint.err_discarded" ||
		findings[0].Line != 6 {
		t.Fatalf("findings = %+v, want the error slot of line 6", findings)
	}
}

func TestLintMethodDiscardFlagged(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

type T struct{}

func (T) Do() error { return nil }

func g(t T) {
	t.Do()
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 ||
		findings[0].Code != "guard.lint.err_discarded" {
		t.Fatalf("findings = %+v, want the method discard", findings)
	}
}

func TestLintFmtFamilyFlagged(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

import (
	"fmt"
	"io"
)

func g(w io.Writer) {
	fmt.Fprint(w, "x")
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Line != 9 ||
		findings[0].Message != "fmt.Fprint: error result discarded as statement" {
		t.Fatalf("L1 has no exemptions: fmt.Fprint must be flagged, got %+v", findings)
	}
}

func TestLintNonErrorResultIgnored(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func f() int { return 0 }

func g() {
	f()
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("non-error results are out of scope, got %+v", findings)
	}
}

func TestLintCommaOKAssertionExempt(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func g(x any) {
	_, ok := x.(string)
	_ = ok
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("comma-ok is exempt, got %+v", findings)
	}
}

func TestLintUnsafeAssertionFlagged(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func g(x any) {
	s := x.(string)
	_ = s
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 ||
		findings[0].Code != "guard.lint.unsafe_type_assert" ||
		findings[0].Line != 4 {
		t.Fatalf("findings = %+v, want the line-4 assertion", findings)
	}
}

func TestLintTypeSwitchNeverFlagged(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func g(x any) {
	switch x.(type) {
	case string:
	}
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf(".(type) has a nil type and must never flag, got %+v", findings)
	}
}

func TestLintTypeErrorIsToolError(t *testing.T) {
	if _, err := runLintFixture(t, map[string]string{"m.go": `package m

var x int = "s"
`}); err == nil {
		t.Fatal("type errors must fail closed (exit 2), want error")
	}
}

func TestLintChannelReceiveNotFlagged(t *testing.T) {
	// Design §3.2 L1 is call-based (errcheck-class): a receive operation
	// is not a call, so draining an error channel must stay exempt — L1
	// has no suppression list, so flagging it would be an un-exemptionable
	// red gate on an idiomatic pattern.
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func recv(ch <-chan error) {
	<-ch
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("non-call expression statements are out of L1 scope, got %+v", findings)
	}
}

func TestLintDeferAndGoNeverFlagged(t *testing.T) {
	findings, err := runLintFixture(t, map[string]string{"m.go": `package m

func discardErr() error { return nil }

func g() {
	defer discardErr()
	go discardErr()
}
`})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("defer/go statements are not expression statements, got %+v", findings)
	}
}

func TestRunLintParseErrorExitsTwo(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "m.go", "package m\n\nfunc f( {\n")
	writeGuardsFile(t, dir, validGuardsJSON)
	var out, errb strings.Builder
	if code := run([]string{"lint"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2 (parse errors are tool errors, never empty findings)", code)
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty on exit 2, got %q", out.String())
	}
	if !strings.Contains(errb.String(), "parse") {
		t.Fatalf("stderr %q must name the parse failure", errb.String())
	}
}

func TestRunLintThroughCLI(t *testing.T) {
	dir := chdirModule(t)
	writeModuleFile(t, dir, "m.go", `package m

func f() error { return nil }

func g() {
	f()
}
`)
	raw, err := json.Marshal(lintFullGuards())
	if err != nil {
		t.Fatal(err)
	}
	writeGuardsFile(t, dir, string(raw))
	var out, errb strings.Builder
	if code := run([]string{"lint"}, &out, &errb); code != 1 {
		t.Fatalf("exit %d, want 1; stderr %q", code, errb.String())
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr must stay empty on exit 1, got %q", errb.String())
	}
	var rep Report
	if err := json.Unmarshal([]byte(out.String()), &rep); err != nil {
		t.Fatalf("stdout is not a report: %v\n%s", err, out.String())
	}
	if rep.Check != "lint" || len(rep.Findings) != 1 ||
		rep.Findings[0].Code != "guard.lint.err_discarded" {
		t.Fatalf("report = %+v", rep)
	}
}
