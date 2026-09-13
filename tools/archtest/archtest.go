// Package archtest enforces aa's module layering rules (architecture decision D5).
//
// It inspects the transitive Go dependencies of each product module and fails
// when a module imports another module that is not permitted by the layering.
package archtest

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ImportPrefix is the shared prefix of every aa Go module.
const ImportPrefix = "github.com/greadee/aa/"

// Module describes a product module.
type Module struct {
	Name string
	Dir  string
}

// Violation is an illegal dependency edge.
type Violation struct {
	Module     string
	Dependency string
	Import     string
}

// String renders a violation for reports.
func (v Violation) String() string {
	return fmt.Sprintf("%s must not import %s (via %s)", v.Module, v.Dependency, v.Import)
}

// ModuleFromImport returns the product module name for an aa import path.
// The second result is false for imports outside the aa import prefix.
func ModuleFromImport(importPath string) (string, bool) {
	if !strings.HasPrefix(importPath, ImportPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(importPath, ImportPrefix)
	if rest == "" {
		return "", false
	}
	name := rest
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		name = rest[:i]
	}
	return name, true
}

// Check reports dependency edges from self that are not in allowed.
// Self-references and non-aa imports are ignored.
func Check(self string, allowed map[string]bool, deps []string) []Violation {
	var violations []Violation
	for _, dep := range deps {
		name, ok := ModuleFromImport(dep)
		if !ok || name == self {
			continue
		}
		if !allowed[name] {
			violations = append(violations, Violation{Module: self, Dependency: name, Import: dep})
		}
	}
	return violations
}

// ListDeps runs `go list -deps` for the module rooted at dir and returns the
// sorted, unique import paths.
func ListDeps(dir string) ([]string, error) {
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.ImportPath}}", "./...")
	cmd.Dir = dir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list in %s: %v: %s", dir, err, strings.TrimSpace(errBuf.String()))
	}
	seen := make(map[string]bool)
	var deps []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		deps = append(deps, line)
	}
	return deps, nil
}
