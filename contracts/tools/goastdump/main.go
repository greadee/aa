// Command goastdump emits the contract struct shapes used by the binding
// parity gate. It exists so field names, Go types, pointers/omitempty, embedded
// structs, and grouped type declarations are read with go/ast instead of
// regular expressions.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"strings"
	"unicode"
)

type field struct {
	JSONName string `json:"jsonName"`
	GoName   string `json:"goName"`
	GoType   string `json:"goType"`
	Optional bool   `json:"optional"`
	Pointer  bool   `json:"pointer"`
}

type typeShape struct {
	Kind     string   `json:"kind"`
	Alias    string   `json:"alias,omitempty"`
	Fields   []field  `json:"fields,omitempty"`
	Embedded []string `json:"embedded,omitempty"`
}

type output struct {
	Types map[string]typeShape `json:"types"`
}

func main() {
	dir := flag.String("dir", "go/v2", "directory containing contract Go files")
	flag.Parse()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, *dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		fail(err)
	}
	out := output{Types: map[string]typeShape{}}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			collect(file, out.Types)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fail(err)
	}
}

func collect(file *ast.File, typesOut map[string]typeShape) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			switch t := ts.Type.(type) {
			case *ast.StructType:
				typesOut[ts.Name.Name] = typeShape{
					Kind:     "struct",
					Fields:   structFields(t),
					Embedded: embeddedTypes(t),
				}
			default:
				typesOut[ts.Name.Name] = typeShape{
					Kind:  "alias",
					Alias: types.ExprString(t),
				}
			}
		}
	}
}

func structFields(st *ast.StructType) []field {
	var fields []field
	for _, f := range st.Fields.List {
		if len(f.Names) == 0 {
			continue
		}
		tag := ""
		if f.Tag != nil {
			tag = strings.Trim(f.Tag.Value, "`")
		}
		jsonTag, hasJSON := parseJSONTag(tag)
		_, isPointer := f.Type.(*ast.StarExpr)
		// omitempty controls JSON key presence. A pointer without omitempty is
		// still a required key (it may encode as null); it is not optional.
		optional := strings.Contains(tag, "omitempty")
		for _, name := range f.Names {
			jsonName := jsonTag
			if !hasJSON {
				jsonName = lowerFirst(name.Name)
			}
			if jsonName == "-" {
				continue
			}
			fields = append(fields, field{
				JSONName: jsonName,
				GoName:   name.Name,
				GoType:   types.ExprString(f.Type),
				Optional: optional,
				Pointer:  isPointer,
			})
		}
	}
	return fields
}

func embeddedTypes(st *ast.StructType) []string {
	var embedded []string
	for _, f := range st.Fields.List {
		if len(f.Names) != 0 {
			continue
		}
		embedded = append(embedded, baseTypeName(f.Type))
	}
	return embedded
}

func baseTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return baseTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	default:
		return types.ExprString(expr)
	}
}

func parseJSONTag(tag string) (string, bool) {
	for _, part := range strings.Split(tag, " ") {
		if !strings.HasPrefix(part, `json:"`) {
			continue
		}
		value := strings.TrimPrefix(part, `json:"`)
		value = strings.TrimSuffix(value, `"`)
		name := strings.Split(value, ",")[0]
		return name, name != ""
	}
	return "", false
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "goastdump: %v\n", err)
	os.Exit(1)
}
