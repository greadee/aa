package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
)

// BoundaryViolation is a visualizer single-observation-boundary violation.
type BoundaryViolation struct {
	Rule   string
	File   string
	Detail string
}

// String renders a violation for reports.
func (v BoundaryViolation) String() string {
	return fmt.Sprintf("visualizer boundary [%s] %s: %s", v.Rule, v.File, v.Detail)
}

// Visualizer boundary rules: obsv is the only observation vocabulary, and the
// visualizer defines no second protocol, transport, or event-store.
const (
	// RuleSingleAdapter: only visualizer/adapter maps the obsv observation
	// protocol into the contracts taxonomy.
	RuleSingleAdapter = "single-obsv-adapter"
	// RuleSingleTransport: only visualizer/source consumes the obsv transport
	// and journal.
	RuleSingleTransport = "single-obsv-transport"
	// RuleNoTransport: no package declares a second observation transport or
	// event-store surface.
	RuleNoTransport = "no-second-transport"
	// RuleNoProtocol: no package declares a second observation event type.
	RuleNoProtocol = "no-second-protocol"
)

const (
	obsvProtocolImport  = "github.com/greadee/aa/obsv/protocol"
	obsvTransportImport = "github.com/greadee/aa/obsv/transport"
	obsvJournalImport   = "github.com/greadee/aa/obsv/journal"
)

type visualizerType struct {
	file     string
	isStruct bool
	fields   map[string]bool
	methods  map[string]bool
}

// CheckVisualizerBoundaries enforces the single-observation guard on the
// visualizer module rooted at root/visualizer. It inspects production Go files
// (test files are exempt) and reports a violation when a package maps the obsv
// protocol outside the one adapter, consumes the obsv transport outside the one
// source, or declares a second observation protocol, transport, or event-store.
func CheckVisualizerBoundaries(root string) ([]BoundaryViolation, error) {
	dir := filepath.Join(root, "visualizer")
	var violations []BoundaryViolation
	packages := map[string]map[string]*visualizerType{}
	fset := token.NewFileSet()

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return fmt.Errorf("parse %s: %w", path, perr)
		}
		rel, _ := filepath.Rel(root, path)
		pkg := file.Name.Name

		for _, imp := range file.Imports {
			target, _ := strconv.Unquote(imp.Path.Value)
			switch target {
			case obsvProtocolImport:
				if pkg != "adapter" {
					violations = append(violations, BoundaryViolation{
						Rule:   RuleSingleAdapter,
						File:   rel,
						Detail: fmt.Sprintf("package %q imports obsv/protocol; only visualizer/adapter may map the observation protocol", pkg),
					})
				}
			case obsvTransportImport, obsvJournalImport:
				if pkg != "source" {
					violations = append(violations, BoundaryViolation{
						Rule:   RuleSingleTransport,
						File:   rel,
						Detail: fmt.Sprintf("package %q imports %s; only visualizer/source may consume the obsv transport", pkg, target),
					})
				}
			}
		}

		pkgDir := filepath.Dir(path)
		types := packages[pkgDir]
		if types == nil {
			types = map[string]*visualizerType{}
			packages[pkgDir] = types
		}
		collectTypes(file, rel, types)
		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, types := range packages {
		for name, t := range types {
			if t.isStruct && t.fields["SessionID"] && t.fields["Confidence"] {
				violations = append(violations, BoundaryViolation{
					Rule:   RuleNoProtocol,
					File:   t.file,
					Detail: fmt.Sprintf("type %q declares an observation event shape (SessionID and Confidence); obsv owns the observation protocol", name),
				})
			}
			if transportLike(t.methods) {
				violations = append(violations, BoundaryViolation{
					Rule:   RuleNoTransport,
					File:   t.file,
					Detail: fmt.Sprintf("type %q declares an observation transport/event-store surface; obsv owns the transport and event-store", name),
				})
			}
		}
	}

	return violations, nil
}

// collectTypes records the type and method declarations of one file keyed by
// type name within its package.
func collectTypes(file *ast.File, rel string, types map[string]*visualizerType) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				t := types[ts.Name.Name]
				if t == nil {
					t = &visualizerType{file: rel, methods: map[string]bool{}}
					types[ts.Name.Name] = t
				}
				switch st := ts.Type.(type) {
				case *ast.StructType:
					t.isStruct = true
					if t.fields == nil {
						t.fields = map[string]bool{}
					}
					for name := range structFieldNames(st) {
						t.fields[name] = true
					}
				case *ast.InterfaceType:
					for _, m := range st.Methods.List {
						for _, n := range m.Names {
							t.methods[n.Name] = true
						}
					}
				}
			}
		case *ast.FuncDecl:
			if d.Recv == nil {
				continue
			}
			name := receiverName(d.Recv)
			if name == "" {
				continue
			}
			t := types[name]
			if t == nil {
				t = &visualizerType{file: rel, methods: map[string]bool{}}
				types[name] = t
			}
			t.methods[d.Name.Name] = true
		}
	}
}

// transportLike reports whether a method set is an observation transport or
// event-store surface: an Append plus at least two lifecycle verbs. The
// visualizer's own seam (Replay/Subscribe, Events/Close) is not a transport.
func transportLike(methods map[string]bool) bool {
	if !methods["Append"] {
		return false
	}
	verbs := 0
	for _, m := range []string{"Hello", "Replay", "Subscribe", "Trim", "Len"} {
		if methods[m] {
			verbs++
		}
	}
	return verbs >= 2
}

func structFieldNames(st *ast.StructType) map[string]bool {
	out := map[string]bool{}
	for _, f := range st.Fields.List {
		for _, n := range f.Names {
			out[n.Name] = true
		}
	}
	return out
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	switch t := recv.List[0].Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.IndexExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.IndexListExpr:
		if id, ok := t.X.(*ast.Ident); ok {
			return id.Name
		}
	}
	return ""
}
