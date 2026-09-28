package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validGuardsJSON = `{
  "skipDirs": ["agent", ".agents", ".claude", ".pi", ".scratch"],
  "arch": {
    "forbidden": [
      {"pkg": "internal/engine",
       "forbidExact": ["net/http", "os/exec"],
       "forbidPrefix": ["net/http/", "os/exec/", "fake-jev/internal/compat/jev",
                        "fake-jev/internal/cli", "fake-jev/internal/control"]}
    ]
  },
  "lint": {
    "rules": ["err_discarded", "unsafe_type_assert"]
  },
  "complexity": {"cyclomaticMax": 15, "crapMax": 50},
  "mutation": {"overallMinPct": 75, "perPackageMinPct": 60,
               "mutantTimeoutSeconds": 60},
  "trace": {"active": [], "catalog": "docs/development/acceptance-catalog.md"},
  "fuzz": {"fuzztime": "30s"}
}
`

func writeGuardsFile(t *testing.T, dir, body string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ".agent", "guards.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunUsageErrors(t *testing.T) {
	for name, args := range map[string][]string{
		"no args":  nil,
		"two args": {"arch", "lint"},
	} {
		var out, errb strings.Builder
		if code := run(args, &out, &errb); code != 2 {
			t.Errorf("%s: exit %d, want 2", name, code)
		}
		if !strings.Contains(errb.String(), "usage: guard") {
			t.Errorf("%s: stderr %q lacks usage", name, errb.String())
		}
		if out.Len() != 0 {
			t.Errorf("%s: stdout must stay empty on exit 2, got %q", name, out.String())
		}
	}
}

func TestRunUnknownCheck(t *testing.T) {
	var out, errb strings.Builder
	if code := run([]string{"bogus"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), `unknown check "bogus"`) {
		t.Fatalf("stderr %q", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty, got %q", out.String())
	}
}

func TestRunMissingThresholds(t *testing.T) {
	t.Chdir(t.TempDir())
	var out, errb strings.Builder
	if code := run([]string{"arch"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "guards.json") {
		t.Fatalf("stderr %q must name guards.json", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty on exit 2, got %q", out.String())
	}
}

func TestRunMalformedThresholds(t *testing.T) {
	dir := t.TempDir()
	writeGuardsFile(t, dir, "{")
	t.Chdir(dir)
	var out, errb strings.Builder
	if code := run([]string{"arch"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), "guards.json") {
		t.Fatalf("stderr %q must name guards.json", errb.String())
	}
	if !strings.Contains(errb.String(), "invalid character") &&
		!strings.Contains(errb.String(), "unexpected end") &&
		!strings.Contains(errb.String(), "unexpected EOF") {
		t.Fatalf("stderr %q must carry the decoder reason", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty on exit 2, got %q", out.String())
	}
}

func TestRunKnownButNotImplemented(t *testing.T) {
	dir := t.TempDir()
	writeGuardsFile(t, dir, validGuardsJSON)
	t.Chdir(dir)
	var out, errb strings.Builder
	if code := run([]string{"complexity"}, &out, &errb); code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errb.String(), `check "complexity" not implemented yet`) {
		t.Fatalf("stderr %q", errb.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty until a handler runs, got %q", out.String())
	}
}

func TestLoadGuardsValid(t *testing.T) {
	dir := t.TempDir()
	path := writeGuardsFile(t, dir, validGuardsJSON)
	g, err := loadGuards(path)
	if err != nil {
		t.Fatalf("loadGuards: %v", err)
	}
	if len(g.SkipDirs) != 5 || g.Fuzz.Fuzztime != "30s" ||
		g.Trace.Catalog != "docs/development/acceptance-catalog.md" {
		t.Fatalf("unexpected parse: %+v", g)
	}
}

func TestLoadGuardsRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	path := writeGuardsFile(t, dir,
		strings.Replace(validGuardsJSON, "{", `{"bogus": 1,`, 1))
	if _, err := loadGuards(path); err == nil ||
		!strings.Contains(err.Error(), "guards.json") {
		t.Fatalf("want guards.json error for unknown field, got %v", err)
	}
}

func TestLoadGuardsRejectsBadFuzztime(t *testing.T) {
	dir := t.TempDir()
	path := writeGuardsFile(t, dir,
		strings.Replace(validGuardsJSON, `"30s"`, `"soon"`, 1))
	if _, err := loadGuards(path); err == nil ||
		!strings.Contains(err.Error(), "fuzz.fuzztime") {
		t.Fatalf("want fuzz.fuzztime error, got %v", err)
	}
}

func TestLoadGuardsRejectsBadActiveID(t *testing.T) {
	dir := t.TempDir()
	path := writeGuardsFile(t, dir,
		strings.Replace(validGuardsJSON, `"active": []`, `"active": ["not-an-id"]`, 1))
	if _, err := loadGuards(path); err == nil ||
		!strings.Contains(err.Error(), "trace.active") {
		t.Fatalf("want trace.active error, got %v", err)
	}
}

func TestEmitReportSortedAndNonNull(t *testing.T) {
	var b strings.Builder
	r := &Report{
		Check: "arch",
		Findings: []Finding{
			{Code: "b", Path: "z.go", Line: 9, Message: "second"},
			{Code: "a", Path: "a.go", Line: 1, Message: "first"},
		},
	}
	if err := emitReport(&b, r); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if strings.Index(s, "a.go") > strings.Index(s, "z.go") {
		t.Fatalf("findings not sorted by path: %s", s)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, s)
	}
	if string(doc["findings"][:1]) != "[" {
		t.Fatalf("findings must be an array: %s", s)
	}
}

func TestEmitReportEmptyFindingsNotNull(t *testing.T) {
	var b strings.Builder
	if err := emitReport(&b, &Report{Check: "arch"}); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	if !strings.Contains(s, `"findings": []`) || !strings.Contains(s, `"stats": {}`) {
		t.Fatalf("empty arrays must serialize as [], got %s", s)
	}
	if !strings.Contains(s, `"check": "arch"`) {
		t.Fatalf("check field missing: %s", s)
	}
}

func TestLoadGuardsTrailingData(t *testing.T) {
	for name, tc := range map[string]struct {
		suffix  string
		wantErr bool
	}{
		"valid alone":     {"", false},
		"trailing brace":  {"}", true},
		"trailing number": {"2", true},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeGuardsFile(t, t.TempDir(), validGuardsJSON+tc.suffix)
			_, err := loadGuards(path)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("loadGuards: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "trailing data") {
				t.Fatalf("want trailing data error, got %v", err)
			}
		})
	}
}

func TestLoadGuardsRejectsInvalidConfig(t *testing.T) {
	cases := []struct {
		name string
		old  string
		new  string
		want string
	}{
		{"absent skipDirs",
			`"skipDirs": ["agent", ".agents", ".claude", ".pi", ".scratch"],
  `, ``,
			"skipDirs must list at least one directory"},
		{"empty skipDirs entry",
			`"skipDirs": ["agent",`, `"skipDirs": ["",`,
			`skipDirs entry`},
		{"skipDirs entry with separator",
			`"skipDirs": ["agent",`, `"skipDirs": ["a/b",`,
			"bare directory name"},
		{"empty arch.forbidden",
			`"forbidden": [
      {"pkg": "internal/engine",
       "forbidExact": ["net/http", "os/exec"],
       "forbidPrefix": ["net/http/", "os/exec/", "fake-jev/internal/compat/jev",
                        "fake-jev/internal/cli", "fake-jev/internal/control"]}
    ]`,
			`"forbidden": []`,
			"arch.forbidden must contain at least one rule"},
		{"empty rule pkg",
			`"pkg": "internal/engine"`, `"pkg": ""`,
			"module-relative package path"},
		{"rule with empty patterns",
			`"forbidExact": ["net/http", "os/exec"],
       "forbidPrefix": ["net/http/", "os/exec/", "fake-jev/internal/compat/jev",
                        "fake-jev/internal/cli", "fake-jev/internal/control"]`,
			`"forbidExact": [],
       "forbidPrefix": []`,
			"forbidExact or forbidPrefix entries"},
		{"empty import pattern",
			`"forbidExact": ["net/http", "os/exec"]`, `"forbidExact": [""]`,
			"empty import pattern"},
		{"unknown lint rule",
			`"rules": ["err_discarded", "unsafe_type_assert"]`,
			`"rules": ["err_discarded", "bogus"]`,
			"lint.rules"},
		{"empty lint rules",
			`"rules": ["err_discarded", "unsafe_type_assert"]`, `"rules": []`,
			"lint.rules"},
		{"complexity below 1",
			`"cyclomaticMax": 15`, `"cyclomaticMax": 0`,
			"complexity thresholds"},
		{"mutation pct above 100",
			`"overallMinPct": 75`, `"overallMinPct": 101`,
			"mutation percentages"},
		{"mutation pct below 0",
			`"perPackageMinPct": 60`, `"perPackageMinPct": -1`,
			"mutation percentages"},
		{"mutation timeout below 1",
			`"mutantTimeoutSeconds": 60`, `"mutantTimeoutSeconds": 0`,
			"mutantTimeoutSeconds"},
		{"empty trace.catalog",
			`"catalog": "docs/development/acceptance-catalog.md"`, `"catalog": ""`,
			"trace.catalog"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := strings.Replace(validGuardsJSON, tc.old, tc.new, 1)
			if body == validGuardsJSON {
				t.Fatalf("replacement %q did not apply", tc.old)
			}
			path := writeGuardsFile(t, t.TempDir(), body)
			_, err := loadGuards(path)
			if err == nil {
				t.Fatal("want error, got nil")
			}
			if !strings.Contains(err.Error(), "guards.json") {
				t.Errorf("error %q lacks guards.json", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q lacks %q", err, tc.want)
			}
		})
	}
}

func TestFuzztimeNanos(t *testing.T) {
	if got, err := fuzztimeNanos("30s"); err != nil || got != 30000000000 {
		t.Fatalf("30s = %d, %v; want 30000000000, nil", got, err)
	}
	if _, err := fuzztimeNanos("9223372036854775807h"); err == nil {
		t.Fatal("want overflow error for 9223372036854775807h, got nil")
	} else if !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("want overflow error, got %v", err)
	}
	if _, err := fuzztimeNanos("soon"); err == nil {
		t.Fatal("want error for bad unit, got nil")
	}
}
