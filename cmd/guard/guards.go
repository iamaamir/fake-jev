package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var activeIDPattern = regexp.MustCompile(`^C-[A-Z0-9]+-[0-9]+$`)

// fuzztimePattern is the single source of truth for the fuzztime shape: the
// unit alternation below is shared by validation (MatchString) and the
// nanosecond conversion (FindStringSubmatch).
var fuzztimePattern = regexp.MustCompile(`^([0-9]+)(ns|us|µs|ms|s|m|h)$`)

type ArchRule struct {
	Pkg          string   `json:"pkg"`
	ForbidExact  []string `json:"forbidExact"`
	ForbidPrefix []string `json:"forbidPrefix"`
}

type ArchGuards struct {
	Forbidden []ArchRule `json:"forbidden"`
}

type LintGuards struct {
	Rules []string `json:"rules"`
}

type ComplexityGuards struct {
	CyclomaticMax int `json:"cyclomaticMax"`
	CrapMax       int `json:"crapMax"`
}

type MutationGuards struct {
	OverallMinPct        int `json:"overallMinPct"`
	PerPackageMinPct     int `json:"perPackageMinPct"`
	MutantTimeoutSeconds int `json:"mutantTimeoutSeconds"`
}

type TraceGuards struct {
	Active  []string `json:"active"`
	Catalog string   `json:"catalog"`
}

type FuzzGuards struct {
	Fuzztime string `json:"fuzztime"`
}

// Guards is the whole of .agent/guards.json (design §3.5). Every section is
// validated on every invocation: thresholds are floors, not hints.
type Guards struct {
	SkipDirs   []string         `json:"skipDirs"`
	Arch       ArchGuards       `json:"arch"`
	Lint       LintGuards       `json:"lint"`
	Complexity ComplexityGuards `json:"complexity"`
	Mutation   MutationGuards   `json:"mutation"`
	Trace      TraceGuards      `json:"trace"`
	Fuzz       FuzzGuards       `json:"fuzz"`

	srcPath string
}

// loadGuards reads and strictly validates the thresholds file. Missing file,
// unknown fields, trailing data, or any violated invariant is an error: the
// caller exits 2 and verify records guard.tool.failed (design §4).
func loadGuards(path string) (*Guards, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("guards.json: %w", err)
	}
	var g Guards
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("guards.json: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("guards.json: trailing data after the JSON object")
	}
	if err := g.validate(); err != nil {
		return nil, fmt.Errorf("guards.json: %w", err)
	}
	g.srcPath = path
	return &g, nil
}

func (g *Guards) validate() error {
	if len(g.SkipDirs) == 0 {
		return errors.New("skipDirs must list at least one directory")
	}
	for _, d := range g.SkipDirs {
		if d == "" || d == "." || d == ".." || strings.ContainsAny(d, `/\`) {
			return fmt.Errorf("skipDirs entry %q must be a bare directory name", d)
		}
	}
	if len(g.Arch.Forbidden) == 0 {
		return errors.New("arch.forbidden must contain at least one rule")
	}
	for i, rule := range g.Arch.Forbidden {
		if rule.Pkg == "" || strings.HasPrefix(rule.Pkg, "/") ||
			strings.HasSuffix(rule.Pkg, "/") {
			return fmt.Errorf("arch.forbidden[%d].pkg %q must be a module-relative package path",
				i, rule.Pkg)
		}
		if len(rule.ForbidExact) == 0 && len(rule.ForbidPrefix) == 0 {
			return fmt.Errorf("arch.forbidden[%d] needs forbidExact or forbidPrefix entries", i)
		}
		for _, e := range append(append([]string{}, rule.ForbidExact...), rule.ForbidPrefix...) {
			if e == "" {
				return fmt.Errorf("arch.forbidden[%d] has an empty import pattern", i)
			}
		}
	}
	if len(g.Lint.Rules) != 2 {
		return errors.New("lint.rules must enable exactly the closed set (err_discarded, unsafe_type_assert)")
	}
	enabled := map[string]bool{}
	for _, r := range g.Lint.Rules {
		enabled[r] = true
	}
	if !enabled["err_discarded"] || !enabled["unsafe_type_assert"] {
		return errors.New("lint.rules must enable exactly the closed set (err_discarded, unsafe_type_assert)")
	}
	if g.Complexity.CyclomaticMax < 1 || g.Complexity.CrapMax < 1 {
		return errors.New("complexity thresholds must be >= 1")
	}
	if g.Mutation.OverallMinPct < 0 || g.Mutation.OverallMinPct > 100 ||
		g.Mutation.PerPackageMinPct < 0 || g.Mutation.PerPackageMinPct > 100 {
		return errors.New("mutation percentages must be within 0..100")
	}
	if g.Mutation.MutantTimeoutSeconds < 1 {
		return errors.New("mutation.mutantTimeoutSeconds must be >= 1")
	}
	if g.Trace.Catalog == "" {
		return errors.New("trace.catalog must be a non-empty path")
	}
	for i, id := range g.Trace.Active {
		if !activeIDPattern.MatchString(id) {
			return fmt.Errorf("trace.active[%d] %q is not a C-ID", i, id)
		}
	}
	if !fuzztimePattern.MatchString(g.Fuzz.Fuzztime) {
		return fmt.Errorf("fuzz.fuzztime %q must match <digits><ns|us|µs|ms|s|m|h>", g.Fuzz.Fuzztime)
	}
	return nil
}

// fuzztimeNanos converts the guarded fuzztime string for reporting. A value
// whose nanosecond expansion does not fit in int64 is an error, never a
// wrapped number: Task 6 fails closed on it (design §4).
func fuzztimeNanos(s string) (int64, error) {
	m := fuzztimePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("bad fuzztime %q", s)
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0, err
	}
	var unitNanos int64
	switch m[2] {
	case "ns":
		unitNanos = 1
	case "us", "µs":
		unitNanos = 1000
	case "ms":
		unitNanos = 1000000
	case "s":
		unitNanos = 1000000000
	case "m":
		unitNanos = 60000000000
	default: // h
		unitNanos = 3600000000000
	}
	if n > math.MaxInt64/unitNanos {
		return 0, fmt.Errorf("fuzztime %q overflows int64 nanoseconds", s)
	}
	return n * unitNanos, nil
}
