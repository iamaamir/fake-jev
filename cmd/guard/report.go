package main

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
)

// Finding is one machine-coded violation. Code values are the design §2
// registry (guard.arch.forbidden_import, guard.lint.err_discarded, ...);
// Path is repository-relative, Line is 1-based (0 = package-level finding).
type Finding struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// Report is the stable JSON contract of every subcommand (design §2):
// {check, findings[], stats{}}. Findings and stats are never null.
type Report struct {
	Check    string         `json:"check"`
	Findings []Finding      `json:"findings"`
	Stats    map[string]int `json:"stats"`
}

// emitReport sorts findings deterministically by (path, line, code) — stable,
// so same-key findings keep AST walk order — then encodes into a buffer and
// writes it once, so an encode failure can never leave partial JSON on stdout
// (the exit-2 contract of main.go).
func emitReport(w io.Writer, r *Report) error {
	if r.Findings == nil {
		r.Findings = []Finding{}
	}
	if r.Stats == nil {
		r.Stats = map[string]int{}
	}
	sort.SliceStable(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Code < b.Code
	})
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return err
	}
	_, err := w.Write(buf.Bytes())
	return err
}
