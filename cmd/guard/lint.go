package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var errorIface = func() *types.Interface {
	iface, ok := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	if !ok {
		panic("guard: builtin error is not an interface")
	}
	return iface
}()

// implementsError reports whether t, or any member of the result tuple t,
// implements error. nil (void results) never does.
func implementsError(t types.Type) bool {
	if t == nil {
		return false
	}
	if tuple, ok := t.(*types.Tuple); ok {
		for i := 0; i < tuple.Len(); i++ {
			if implementsError(tuple.At(i).Type()) {
				return true
			}
		}
		return false
	}
	return types.Implements(t, errorIface)
}

func unwrapParens(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}

// fileRel maps a package dir + file name to the module-relative file path.
func fileRel(relPkg, name string) string {
	if relPkg == "." {
		return name
	}
	return relPkg + "/" + name
}

// calleeDisplay names the callee in finding messages: the bare function
// name for a plain identifier, the full expression otherwise. Purely
// syntactic — the zero-tolerance rule needs no exemption key (note 7).
func calleeDisplay(fun ast.Expr) string {
	fun = unwrapParens(fun)
	switch f := fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return types.ExprString(f)
	default:
		return types.ExprString(fun)
	}
}

// alignedRhsType picks the result type aligned with Lhs[i]: a single tuple
// RHS indexes the tuple; a same-length multi-RHS assign takes RHS[i].
func alignedRhsType(as *ast.AssignStmt, i int, info *types.Info) types.Type {
	if len(as.Rhs) == 1 {
		tv := info.Types[as.Rhs[0]]
		if tuple, ok := tv.Type.(*types.Tuple); ok {
			if i < tuple.Len() {
				return tuple.At(i).Type()
			}
			return nil
		}
		if i == 0 {
			return tv.Type
		}
		return nil
	}
	if i < len(as.Rhs) {
		return info.Types[as.Rhs[i]].Type
	}
	return nil
}

// exportImporter builds a go/types importer over `go list -e -deps -export`
// data: deterministic (fixed package set), offline (GOPROXY=off), and
// fail-closed — a dependency without export data surfaces as a type error.
// Each kept package's TestImports are roots too: -deps walks only the
// regular import graph, but runLint type-checks TestGoFiles alongside
// GoFiles, so the test-only imports need export data as well.
func exportImporter(fset *token.FileSet, pkgs []listedPackage) (types.Importer, error) {
	seen := map[string]bool{}
	var paths []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		paths = append(paths, p)
	}
	for _, p := range pkgs {
		add(p.ImportPath)
		for _, imp := range p.TestImports {
			add(imp)
		}
	}
	args := []string{"list", "-e", "-deps", "-export", "-json"}
	args = append(args, paths...)
	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list -export: %w: %s", err,
			strings.TrimSpace(stderr.String()))
	}
	exports := map[string]string{}
	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	for {
		var e struct {
			ImportPath string `json:"ImportPath"`
			Export     string `json:"Export"`
		}
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("go list -export: decode: %w", err)
		}
		if e.Export != "" {
			exports[e.ImportPath] = e.Export
		}
	}
	return importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		f, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("no export data for %q", path)
		}
		return os.Open(f)
	}), nil
}

// runLint proves the closed L1/L2 set absent over GoFiles+TestGoFiles of
// every kept package (note 7). Parse or type errors abort with exit 2.
func runLint(g *Guards) ([]Finding, map[string]int, error) {
	modPath, err := readModulePath()
	if err != nil {
		return nil, nil, err
	}
	pkgs, err := listPackages(modPath, g.SkipDirs)
	if err != nil {
		return nil, nil, err
	}
	stats := map[string]int{"packages": len(pkgs), "files": 0}
	if len(pkgs) == 0 {
		return nil, stats, nil
	}
	fset := token.NewFileSet()
	imp, err := exportImporter(fset, pkgs)
	if err != nil {
		return nil, nil, err
	}
	var findings []Finding
	for _, p := range pkgs {
		// Same fail-closed guard as runArch: a misparsed go.mod module
		// directive would make moduleRel no-op and every finding Path
		// absolute (I2 fail-open). Reject instead.
		if p.ImportPath != modPath && !strings.HasPrefix(p.ImportPath, modPath+"/") {
			return nil, nil, fmt.Errorf("go list: package %q outside module %q",
				p.ImportPath, modPath)
		}
		relPkg := moduleRel(modPath, p.ImportPath)
		names := append(append([]string{}, p.GoFiles...), p.TestGoFiles...)
		var files []*ast.File
		var fileNames []string
		for _, name := range names {
			f, err := parser.ParseFile(fset, filepath.Join(p.Dir, name), nil,
				parser.AllErrors)
			if err != nil {
				return nil, nil, fmt.Errorf("parse %s/%s: %w",
					p.ImportPath, name, err)
			}
			files = append(files, f)
			fileNames = append(fileNames, name)
		}
		stats["files"] += len(files)
		if len(files) == 0 {
			continue
		}
		info := &types.Info{
			Types: map[ast.Expr]types.TypeAndValue{},
		}
		conf := types.Config{Importer: imp}
		if _, err := conf.Check(p.ImportPath, fset, files, info); err != nil {
			return nil, nil, fmt.Errorf("type check %s: %w", p.ImportPath, err)
		}
		for idx, file := range files {
			relFile := fileRel(relPkg, fileNames[idx])
			// Pass 1: mark comma-ok assertions (2 LHS, 1 RHS) exempt.
			exempt := map[ast.Expr]bool{}
			ast.Inspect(file, func(n ast.Node) bool {
				as, ok := n.(*ast.AssignStmt)
				if !ok || len(as.Lhs) != 2 || len(as.Rhs) != 1 {
					return true
				}
				if ta, ok := unwrapParens(as.Rhs[0]).(*ast.TypeAssertExpr); ok {
					exempt[ta] = true
				}
				return true
			})
			// Pass 2: L1 statement/blank discards, L2 unsafe assertions.
			ast.Inspect(file, func(n ast.Node) bool {
				switch st := n.(type) {
				case *ast.ExprStmt:
					// Design §3.2 L1 is call-based (errcheck-class): a
					// non-call expression statement — a receive operation
					// like `<-errCh` — is not a discarded error result.
					if _, ok := unwrapParens(st.X).(*ast.CallExpr); !ok {
						return true
					}
					tv, ok := info.Types[st.X]
					if !ok || !implementsError(tv.Type) {
						return true
					}
					findings = append(findings, Finding{
						Code: "guard.lint.err_discarded",
						Path: relFile,
						Line: fset.Position(st.Pos()).Line,
						Message: calleeDisplay(callFun(st.X)) +
							": error result discarded as statement",
					})
				case *ast.AssignStmt:
					for i, lhs := range st.Lhs {
						blank, ok := lhs.(*ast.Ident)
						if !ok || blank.Name != "_" {
							continue
						}
						t := alignedRhsType(st, i, info)
						if !implementsError(t) {
							continue
						}
						msg := "error result assigned to _"
						if call, ok := unwrapParens(st.Rhs[min(len(st.Rhs)-1, i)]).(*ast.CallExpr); ok {
							msg = calleeDisplay(call.Fun) + ": " + msg
						}
						findings = append(findings, Finding{
							Code:    "guard.lint.err_discarded",
							Path:    relFile,
							Line:    fset.Position(lhs.Pos()).Line,
							Message: msg,
						})
					}
				case *ast.TypeAssertExpr:
					if st.Type == nil || exempt[st] {
						return true
					}
					findings = append(findings, Finding{
						Code:    "guard.lint.unsafe_type_assert",
						Path:    relFile,
						Line:    fset.Position(st.Pos()).Line,
						Message: "unchecked type assertion panics on mismatch",
					})
				}
				return true
			})
		}
	}
	return findings, stats, nil
}

// callFun extracts the callee of a call expression for the finding message;
// a non-call expression is passed through unchanged.
func callFun(e ast.Expr) ast.Expr {
	if call, ok := e.(*ast.CallExpr); ok {
		return call.Fun
	}
	return e
}
