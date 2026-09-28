package main

import (
	"fmt"
	"strings"
)

// runArch proves spec §17.3 + guards.json forbidden imports absent
// (design §3.2). Findings are package-level: Path is the module-relative
// directory, Line 0, deduplicated per (package, import) (note 8).
func runArch(g *Guards) ([]Finding, map[string]int, error) {
	modPath, err := readModulePath()
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := listPackages(modPath, g.SkipDirs)
	if err != nil {
		return nil, nil, err
	}
	stats := map[string]int{"packages": len(pkgs)}
	files := 0
	var findings []Finding
	seen := map[string]bool{}
	for _, p := range pkgs {
		files += p.fileCount()
		// An import path outside the module means readModulePath misparsed
		// go.mod: moduleRel would no-op, no rule could match, and the scan
		// would exit 0 with empty findings (I2 fail-open). Reject instead.
		if p.ImportPath != modPath && !strings.HasPrefix(p.ImportPath, modPath+"/") {
			return nil, nil, fmt.Errorf("go list: package %q outside module %q",
				p.ImportPath, modPath)
		}
		rel := moduleRel(modPath, p.ImportPath)
		for _, rule := range g.Arch.Forbidden {
			if rule.Pkg != rel {
				continue
			}
			exact := map[string]bool{}
			for _, e := range rule.ForbidExact {
				exact[e] = true
			}
			for _, imp := range p.allImports() {
				forbidden := exact[imp]
				if !forbidden {
					for _, pre := range rule.ForbidPrefix {
						if pre != "" && strings.HasPrefix(imp, pre) {
							forbidden = true
							break
						}
					}
				}
				if !forbidden {
					continue
				}
				key := rel + "\x00" + imp
				if seen[key] {
					continue
				}
				seen[key] = true
				findings = append(findings, Finding{
					Code:    "guard.arch.forbidden_import",
					Path:    rel,
					Line:    0,
					Message: fmt.Sprintf("package %s imports %q", p.ImportPath, imp),
				})
			}
		}
	}
	stats["files"] = files
	return findings, stats, nil
}
