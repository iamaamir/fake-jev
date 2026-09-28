package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// catalogRowPattern extracts the known C-ID set from the acceptance catalog
// table (FJ-049's file, read-only for guards); (?m) anchors ^ per line.
// Empirically: 108 rows on the repository catalog.
var catalogRowPattern = regexp.MustCompile(`(?m)^\|\s*(C-[A-Z0-9]+-[0-9]+)\s*\|`)

// markerPattern finds C-ID occurrences in string literals and testdata tags.
// The leading boundary refuses matches inside longer tokens (XC-GOLD-001 is
// not a marker); group 2 is the ID.
var markerPattern = regexp.MustCompile(`(^|[^A-Za-z0-9-])(C-[A-Z0-9]+-[0-9]+)`)

// loadCatalogIDs reads the known-ID set. An absent or rowless catalog cannot
// prove anything, so it is a tool error (exit 2), never a vacuous pass.
func loadCatalogIDs(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("trace: read catalog %s: %w", path, err)
	}
	known := map[string]bool{}
	for _, m := range catalogRowPattern.FindAllStringSubmatch(string(data), -1) {
		known[m[1]] = true
	}
	if len(known) == 0 {
		return nil, fmt.Errorf("trace: catalog %s has no C-ID rows", path)
	}
	return known, nil
}

// runTrace proves design §3.1 (note 5): every active C-ID has a marker
// (zero tolerance, path .agent/guards.json) and every `_test.go` marker names
// a catalog ID. testdata/contracts tags count as coverage but are never
// catalog-validated. The scan is syntactic — no go.mod, no build — and the
// walk skips dot-dirs plus guards.json skipDirs like every other subcommand.
func runTrace(g *Guards) ([]Finding, map[string]int, error) {
	known, err := loadCatalogIDs(g.Trace.Catalog)
	if err != nil {
		return nil, nil, err
	}
	covered := map[string]bool{}
	seenUnknown := map[string]bool{}
	var findings []Finding
	testFiles := 0
	fset := token.NewFileSet()
	walkErr := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(path)
		if d.IsDir() {
			if path == "." {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			if strings.HasPrefix(d.Name(), "_") {
				return fs.SkipDir
			}
			// Root-anchored skipDirs (note 8): only the FIRST segment of the
			// walk path (module-relative, cwd is the module root) can name a
			// skipped tool directory; a nested dir merely sharing the name
			// (internal/agent) is repository content and stays in the scan.
			head, _, _ := strings.Cut(rel, "/")
			for _, s := range g.SkipDirs {
				if head == s {
					return fs.SkipDir
				}
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if strings.HasPrefix(d.Name(), "_") {
			return nil
		}
		if strings.HasPrefix(rel, "testdata/contracts/") {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("trace: read %s: %w", rel, err)
			}
			for _, m := range markerPattern.FindAllStringSubmatch(string(data), -1) {
				covered[m[2]] = true
			}
			return nil
		}
		if strings.HasPrefix(rel, "testdata/") {
			return nil
		}
		if !strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		testFiles++
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return fmt.Errorf("trace: parse %s: %w", rel, err)
		}
		var unqErr error
		ast.Inspect(f, func(n ast.Node) bool {
			if unqErr != nil {
				return false
			}
			bl, ok := n.(*ast.BasicLit)
			if !ok || bl.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(bl.Value)
			if err != nil {
				unqErr = fmt.Errorf("trace: unquote %s: %w", rel, err)
				return false
			}
			base := fset.Position(bl.Pos()).Line
			for _, m := range markerPattern.FindAllStringSubmatchIndex(s, -1) {
				id := s[m[4]:m[5]]
				covered[id] = true
				if known[id] {
					continue
				}
				line := base
				if bl.Value[0] == '`' {
					line = base + strings.Count(s[:m[4]], "\n")
				}
				key := rel + "\x00" + strconv.Itoa(line) + "\x00" + id
				if seenUnknown[key] {
					continue
				}
				seenUnknown[key] = true
				findings = append(findings, Finding{
					Code: "guard.trace.unknown_id",
					Path: rel,
					Line: line,
					Message: fmt.Sprintf("unknown acceptance ID %q (not in %s)",
						id, g.Trace.Catalog),
				})
			}
			return true
		})
		if unqErr != nil {
			return unqErr
		}
		return nil
	})
	if walkErr != nil {
		return nil, nil, walkErr
	}
	for _, id := range g.Trace.Active {
		if !covered[id] {
			findings = append(findings, Finding{
				Code: "guard.trace.untraced",
				Path: ".agent/guards.json",
				Line: 0,
				Message: fmt.Sprintf("active acceptance ID %q has no test marker",
					id),
			})
		}
	}
	stats := map[string]int{
		"active":     len(g.Trace.Active),
		"test_files": testFiles,
		"covered":    len(covered),
	}
	return findings, stats, nil
}
