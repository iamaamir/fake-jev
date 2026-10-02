package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// fuzzTarget is one fuzz entry point discovered without executing anything.
type fuzzTarget struct {
	ImportPath string
	Dir        string // absolute directory - the exec working directory
	File       string // module-relative declaring file
	Line       int    // 1-based line of the FuncDecl
	Name       string // function name
}

// writtenInputPattern matches go1.26's crash-input line. go may indent the
// line inside the FAIL block, so the match is unanchored (verified verbatim:
// `Failing input written to testdata/fuzz/<Name>/<hash>`, no colon, path
// relative to the package directory - interpretation note 9).
var writtenInputPattern = regexp.MustCompile(`Failing input written to ([^\r\n]+)`)

const maxFuzzOutput = 1 << 20

// fuzzOutput bounds memory used for subprocess diagnostics. A fuzz target can
// write arbitrary output before crashing; retaining an unbounded bytes.Buffer
// would turn that untrusted output into a resource-exhaustion vector. Truncation
// causes unknown failures to fail closed, while preserving normal Go diagnostics.
type fuzzOutput struct {
	buf       bytes.Buffer
	truncated bool
}

func (o *fuzzOutput) Write(p []byte) (int, error) {
	remaining := maxFuzzOutput - o.buf.Len()
	if remaining > 0 {
		if len(p) > remaining {
			n, err := o.buf.Write(p[:remaining])
			if err != nil {
				return n, err
			}
			o.truncated = true
		} else {
			n, err := o.buf.Write(p)
			if err != nil {
				return n, err
			}
		}
	} else if len(p) > 0 {
		o.truncated = true
	}
	return len(p), nil
}

func (o *fuzzOutput) String() string { return o.buf.String() }

// seedFailurePattern matches go's seed-corpus failure line. The marker must
// start a line (leading indentation tolerated): go1.26.3 emits
// `failure while testing seed corpus entry: <Target>/<entry>` verbatim and
// unindented. A bare substring search is not safe - a build failure's compiler
// diagnostic echoes the offending source line
// (`./f.go:6:2: "failure while testing ..." is not used`), which would
// fabricate a guard.fuzz.crash finding for a package that never ran an input.
// The diagnostic's `file:line:col:` prefix keeps that echo from matching, so
// an uncompilable package stays an operational error (AC-10).
var seedFailurePattern = regexp.MustCompile(
	`(?m)^[ \t]*failure while testing seed corpus entry: `)

// fuzzFailureKind is the disposition of a non-zero `go test -fuzz` exit.
type fuzzFailureKind int

const (
	// fuzzFailureWrittenInput: go attributed the failure to a written input
	// (`Failing input written to <path>`) - the input is the regression and
	// is preserved under testdata/fuzz/<Name>/.
	fuzzFailureWrittenInput fuzzFailureKind = iota
	// fuzzFailureSeedCorpus: a committed seed entry failed during warmup
	// (`failure while testing seed corpus entry: <Target>/<entry>`); no new
	// input is written because the seed is already in the tree.
	fuzzFailureSeedCorpus
	// fuzzFailureOperational: go reported no corpus-attributable failure - a
	// fuzzing worker that could not start, communicate, or terminate, a build
	// failure, or any other unrecognized output.
	fuzzFailureOperational
)

// classifyFuzzFailure maps a non-zero `go test -fuzz` exit's combined output
// to a disposition. Only go's two corpus-attribution markers are target-level:
// `Failing input written to <path>` (go attributed the failure to a concrete
// input) and `failure while testing seed corpus entry` (a committed seed
// failed). A bare `--- FAIL: <Name>` carries no attribution: go1.26.3 emits it
// identically when a fuzzing worker fails to start, communicate, or terminate
// (verified verbatim under fd pressure: `--- FAIL: FuzzSafe` followed by
// `fork/exec ...: too many open files`, with neither marker present). Such
// worker/infrastructure failures are operational errors, never
// guard.fuzz.crash findings. The written-input marker is evaluated first, the
// line-anchored seed marker second; any other output fails closed to an
// operational error.
func classifyFuzzFailure(text string) (fuzzFailureKind, string) {
	if m := writtenInputPattern.FindStringSubmatch(text); m != nil {
		return fuzzFailureWrittenInput, m[1]
	}
	if seedFailurePattern.MatchString(text) {
		return fuzzFailureSeedCorpus, ""
	}
	return fuzzFailureOperational, ""
}

func isFuzzName(name string) bool {
	return len(name) > len("Fuzz") && strings.HasPrefix(name, "Fuzz")
}

func isTestingFParam(e ast.Expr) bool {
	star, ok := e.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && sel.Sel.Name == "F"
}

// discoverFuzzTargets finds every fuzz entry point with the syntactic
// signature go requires: FuncDecl named Fuzz*, no receiver, no results, and
// exactly one parameter that parses as *testing.F (a field binding several
// names, (f, g *testing.F), does not count). Files come from go list's
// TestGoFiles/XTestGoFiles; a test file that does not parse is a tool error -
// absence of targets cannot be proven from an unreadable file (fail-closed).
// The result is sorted by (ImportPath, Name) for deterministic execution.
func discoverFuzzTargets(modPath string, skipDirs []string) ([]fuzzTarget, error) {
	pkgs, err := listPackages(modPath, skipDirs)
	if err != nil {
		return nil, err
	}
	var targets []fuzzTarget
	for _, p := range pkgs {
		// Same fail-closed guard as runArch/runLint: a misparsed go.mod
		// module directive would make moduleRel no-op and every declaring
		// path absolute (I2 fail-open). Reject instead.
		if p.ImportPath != modPath && !strings.HasPrefix(p.ImportPath, modPath+"/") {
			return nil, fmt.Errorf("go list: package %q outside module %q",
				p.ImportPath, modPath)
		}
		relDir := moduleRel(modPath, p.ImportPath)
		fset := token.NewFileSet()
		testFiles := append(append([]string{}, p.TestGoFiles...), p.XTestGoFiles...)
		for _, name := range testFiles {
			path := filepath.Join(p.Dir, name)
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return nil, fmt.Errorf("fuzz: parse %s: %w", name, err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil ||
					(fn.Type.Results != nil && len(fn.Type.Results.List) > 0) ||
					!isFuzzName(fn.Name.Name) || len(fn.Type.Params.List) != 1 {
					continue
				}
				if len(fn.Type.Params.List[0].Names) != 1 ||
					!isTestingFParam(fn.Type.Params.List[0].Type) {
					continue
				}
				targets = append(targets, fuzzTarget{
					ImportPath: p.ImportPath,
					Dir:        p.Dir,
					File:       filepath.Join(relDir, name),
					Line:       fset.Position(fn.Pos()).Line,
					Name:       fn.Name.Name,
				})
			}
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].ImportPath != targets[j].ImportPath {
			return targets[i].ImportPath < targets[j].ImportPath
		}
		return targets[i].Name < targets[j].Name
	})
	return targets, nil
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// rejectSymlinkParents prevents a repository-controlled testdata symlink from
// redirecting a preserved crash input outside its package directory.
func rejectSymlinkParents(root, path string) error {
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("path root %s is a symlink", root)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}
	cur := root
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		cur = filepath.Join(cur, part)
		info, err := os.Lstat(cur)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %s is a symlink", cur)
		}
	}
	return nil
}

// preserveCrashInput validates go's reported path before reading or writing.
// The subprocess output is not trusted: without the package-root check a
// crashing target could make the guard copy an arbitrary file into its corpus.
func preserveCrashInput(modPath string, tgt fuzzTarget, reported string) (string, error) {
	pkgDir, err := filepath.Abs(tgt.Dir)
	if err != nil {
		return "", fmt.Errorf("resolve package directory: %w", err)
	}
	src := strings.TrimSpace(reported)
	if src == "" {
		return "", errors.New("crash input path is empty")
	}
	if !filepath.IsAbs(src) {
		src = filepath.Join(pkgDir, src)
	}
	src, err = filepath.Abs(src)
	if err != nil {
		return "", fmt.Errorf("resolve crash input: %w", err)
	}
	if !pathWithin(pkgDir, src) {
		return "", fmt.Errorf("crash input path %q escapes package directory", reported)
	}
	name := filepath.Base(src)
	if name == "." || name == string(filepath.Separator) || name == ".." {
		return "", fmt.Errorf("crash input path %q is not a file", reported)
	}
	dest := filepath.Join(pkgDir, "testdata", "fuzz", tgt.Name, name)
	if !pathWithin(pkgDir, dest) {
		return "", fmt.Errorf("crash input destination escapes package directory")
	}
	if err := rejectSymlinkParents(pkgDir, filepath.Dir(dest)); err != nil {
		return "", fmt.Errorf("validate crash input destination: %w", err)
	}
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return "", fmt.Errorf("crash input vanished: %w", err)
	}
	if !srcInfo.Mode().IsRegular() {
		return "", fmt.Errorf("crash input %s is not a regular file", src)
	}
	destInfo, destErr := os.Lstat(dest)
	if destErr == nil {
		if !destInfo.Mode().IsRegular() {
			return "", fmt.Errorf("crash input destination %s is not a regular file", dest)
		}
	} else if !errors.Is(destErr, os.ErrNotExist) {
		return "", fmt.Errorf("inspect crash input destination: %w", destErr)
	}
	if src != dest && errors.Is(destErr, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", fmt.Errorf("preserve crash input: %w", err)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return "", fmt.Errorf("read crash input: %w", err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return "", fmt.Errorf("preserve crash input: %w", err)
		}
	}
	return filepath.Join(moduleRel(modPath, tgt.ImportPath),
		"testdata", "fuzz", tgt.Name, name), nil
}

// runOneFuzz executes one bounded invocation and classifies a non-zero exit
// via classifyFuzzFailure: a written crash input is the primary finding
// (preserved under testdata/fuzz/<Name>/ - go has normally written it there
// already, making the copy a no-op by construction); a seed-corpus failure
// points at the declaring source; anything else - a bare `--- FAIL: <Name>`,
// a fuzzing-worker start/communication/termination failure, [build failed],
// vet, toolchain trouble - is an operational error, never a silent pass and
// never a fabricated crash finding (FJ-060).
func runOneFuzz(g *Guards, modPath string, tgt fuzzTarget) ([]Finding, error) {
	cmd := exec.Command("go", "test", "-count=1", "-run", "^$",
		"-fuzz", "^"+regexp.QuoteMeta(tgt.Name)+"$",
		"-fuzztime", g.Fuzz.Fuzztime, tgt.ImportPath)
	cmd.Dir = tgt.Dir
	// Offline and workspace-free by construction, mirroring listPackages: a
	// fuzz run may never reach a proxy or be reshaped by an ambient go.work.
	cmd.Env = append(os.Environ(), "GOPROXY=off", "GOWORK=off")
	var out fuzzOutput
	cmd.Stdout = &out // one shared writer: os/exec serializes concurrent
	cmd.Stderr = &out // writes when Stdout == Stderr (comparable writers)
	err := cmd.Run()
	if err == nil {
		return nil, nil
	}
	// errors.As instead of `if _, ok := err.(*exec.ExitError)`: the blank
	// slot of that assertion aligns with *exec.ExitError, which implements
	// error, so lint L1 (zero tolerance, note 7) counts it as a discarded
	// error result. No suppression list exists; the form must not violate.
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return nil, fmt.Errorf("fuzz: %s: %w", tgt.ImportPath, err)
	}
	text := out.String()
	switch kind, input := classifyFuzzFailure(text); kind {
	case fuzzFailureWrittenInput:
		rel, preserveErr := preserveCrashInput(modPath, tgt, input)
		if preserveErr != nil {
			return nil, fmt.Errorf("fuzz: %s: %w", tgt.Name, preserveErr)
		}
		return []Finding{{
			Code: "guard.fuzz.crash",
			Path: rel,
			Line: 0,
			Message: fmt.Sprintf("fuzz target %s crashed; failing input preserved",
				tgt.Name),
		}}, nil
	case fuzzFailureSeedCorpus:
		return []Finding{{
			Code:    "guard.fuzz.crash",
			Path:    tgt.File,
			Line:    tgt.Line,
			Message: fmt.Sprintf("fuzz target %s failed", tgt.Name),
		}}, nil
	}
	trimmed := strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	if runes := []rune(trimmed); len(runes) > 400 {
		trimmed = string(runes[:400]) + "..."
	}
	return nil, fmt.Errorf("fuzz: %s: unexpected failure: %s", tgt.ImportPath,
		trimmed)
}

// runFuzz is the bounded live-fuzz runner (design 3.3): syntactic discovery,
// one `go test -fuzz` invocation per target under guards.json fuzztime, and
// zero targets as the vacuous-pass convention (design 4 - no invocation at
// all, which is what keeps task-less runs instant). The positivity check runs
// before any filesystem or process work: guards.json validation accepts "0s"
// as a well-formed pattern, so an unlimited window must be impossible here.
func runFuzz(g *Guards) ([]Finding, map[string]int, error) {
	ns, err := fuzztimeNanos(g.Fuzz.Fuzztime)
	if err != nil || ns <= 0 {
		return nil, nil, fmt.Errorf("fuzz.fuzztime %q must be a positive duration",
			g.Fuzz.Fuzztime)
	}
	modPath, err := readModulePath()
	if err != nil {
		return nil, nil, err
	}
	targets, err := discoverFuzzTargets(modPath, g.SkipDirs)
	if err != nil {
		return nil, nil, err
	}
	var findings []Finding
	for _, tgt := range targets {
		got, err := runOneFuzz(g, modPath, tgt)
		if err != nil {
			return nil, nil, err
		}
		findings = append(findings, got...)
	}
	return findings, map[string]int{"targets": len(targets)}, nil
}
