package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type listError struct {
	Import string `json:"Import"`
	Pos    string `json:"Pos"`
	Err    string `json:"Err"`
}

// listedPackage is the subset of `go list -json` the subcommands consume.
type listedPackage struct {
	ImportPath   string       `json:"ImportPath"`
	Dir          string       `json:"Dir"`
	GoFiles      []string     `json:"GoFiles"`
	TestGoFiles  []string     `json:"TestGoFiles"`
	XTestGoFiles []string     `json:"XTestGoFiles"`
	Imports      []string     `json:"Imports"`
	TestImports  []string     `json:"TestImports"`
	XTestImports []string     `json:"XTestImports"`
	Error        *listError   `json:"Error"`
	DepsErrors   []*listError `json:"DepsErrors"`
}

// allImports unions the three import lists (aligned with the three file
// kinds) so a package's test-only imports are guarded exactly like its
// production imports.
func (p listedPackage) allImports() []string {
	seen := map[string]bool{}
	var out []string
	for _, imp := range append(append(append([]string{}, p.Imports...),
		p.TestImports...), p.XTestImports...) {
		if !seen[imp] {
			seen[imp] = true
			out = append(out, imp)
		}
	}
	return out
}

// fileCount is GoFiles + TestGoFiles + XTestGoFiles — the scan scope of
// interpretation note 8.
func (p listedPackage) fileCount() int {
	return len(p.GoFiles) + len(p.TestGoFiles) + len(p.XTestGoFiles)
}

// pathSkipped reports whether the FIRST segment of the module-relative path
// names a gitignored tool directory — verify's root-anchored NON_REPO_DIRS
// semantics (note 8). A nested directory that merely shares a tool-dir name
// (internal/agent) is repository content and stays in the scan.
func pathSkipped(modPath, importPath string, skipDirs []string) bool {
	head, _, _ := strings.Cut(moduleRel(modPath, importPath), "/")
	for _, s := range skipDirs {
		if head == s {
			return true
		}
	}
	return false
}

// readModulePath parses the module directive of ./go.mod. Subcommands run
// from the module root (verify cds there; the tests t.Chdir there).
func readModulePath() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			m := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			if m != "" {
				return m, nil
			}
		}
	}
	return "", errors.New("go.mod: no module directive")
}

// moduleRel maps an import path to its module-relative directory
// ("fake-jev/internal/cli" -> "internal/cli", the root package -> ".").
// Import-path arithmetic instead of filesystem paths keeps this immune to
// symlinked temp directories.
func moduleRel(modPath, importPath string) string {
	if importPath == modPath {
		return "."
	}
	return strings.TrimPrefix(importPath, modPath+"/")
}

// listPackages runs `go list -e -json ./...`, drops skipped tool directories
// first, and fails on any kept package's load error or its dependency load
// errors: absence of violations cannot be proven from a package that failed
// to load (fail-closed).
func listPackages(modPath string, skipDirs []string) ([]listedPackage, error) {
	cmd := exec.Command("go", "list", "-e", "-json", "./...")
	// Deterministic and offline by construction (design: stdlib-only, no
	// network): a missing external module becomes a load error at exit 2
	// instead of a proxy lookup, and an ambient go.work cannot change the
	// package set.
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w: %s", err,
			strings.TrimSpace(stderr.String()))
	}
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var pkgs []listedPackage
	for {
		var p listedPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("go list: decode: %w", err)
		}
		if pathSkipped(modPath, p.ImportPath, skipDirs) {
			continue
		}
		if p.Error != nil {
			return nil, fmt.Errorf("go list: package %s: %s",
				p.ImportPath, p.Error.Err)
		}
		// go list -e exits 0 with Error null when a dependency cannot load;
		// DepsErrors carries those. A kept package whose imports do not
		// resolve is a load failure too (B1), never a silent pass.
		if len(p.DepsErrors) > 0 {
			return nil, fmt.Errorf("go list: package %s: %s: %s",
				p.ImportPath, p.DepsErrors[0].Pos, p.DepsErrors[0].Err)
		}
		pkgs = append(pkgs, p)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return pkgs[i].ImportPath < pkgs[j].ImportPath
	})
	return pkgs, nil
}
