// Package template renders aa's versioned process templates.
//
// Templates are data: the engine loads Markdown and PlantUML files from
// docs/github-os/templates and renders them with Go text/template. Rendering is
// deterministic, and no template executes outside the renderer.
package template

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// DefaultEnv names the environment variable that overrides the template root.
const DefaultEnv = "AA_TEMPLATES_DIR"

// RelativeRoot is the in-repository template root.
const RelativeRoot = "docs/github-os/templates"

// Engine loads and renders process templates from a root directory.
type Engine struct {
	root string
	tpl  *template.Template
}

// NewEngine loads every Markdown and PlantUML template under root.
func NewEngine(root string) (*Engine, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("template: root %q: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("template: root %q is not a directory", root)
	}
	base := template.New("forge").Funcs(funcs())
	engine := &Engine{root: root, tpl: base}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".uml":
		default:
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, err := base.New(name).Parse(string(data)); err != nil {
			return fmt.Errorf("template: parse %s: %w", name, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return engine, nil
}

// Root returns the directory the engine loaded from.
func (e *Engine) Root() string { return e.root }

// Names returns the loaded template names in sorted order.
func (e *Engine) Names() []string {
	var names []string
	for _, t := range e.tpl.Templates() {
		if t.Name() == "forge" {
			continue
		}
		names = append(names, t.Name())
	}
	sort.Strings(names)
	return names
}

// Render renders a loaded template by name.
func (e *Engine) Render(name string, data any) (string, error) {
	var builder strings.Builder
	if err := e.tpl.ExecuteTemplate(&builder, name, data); err != nil {
		return "", fmt.Errorf("template: render %s: %w", name, err)
	}
	return builder.String(), nil
}

// RenderString renders an inline template (usable without a root).
func RenderString(text string, data any) (string, error) {
	tpl, err := template.New("inline").Funcs(funcs()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("template: parse inline: %w", err)
	}
	var builder strings.Builder
	if err := tpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("template: render inline: %w", err)
	}
	return builder.String(), nil
}

// DefaultRoot returns the template root from the environment, or the nearest
// docs/github-os/templates directory at or above start.
func DefaultRoot(start string) (string, bool) {
	if env := strings.TrimSpace(os.Getenv(DefaultEnv)); env != "" {
		return env, true
	}
	if start == "" {
		if cwd, err := os.Getwd(); err == nil {
			start = cwd
		}
	}
	return FindRoot(start)
}

// FindRoot walks up from start looking for RelativeRoot.
func FindRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		candidate := filepath.Join(dir, filepath.FromSlash(RelativeRoot))
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func funcs() template.FuncMap {
	return template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"trim":  strings.TrimSpace,
		"default": func(fallback, value string) string {
			if strings.TrimSpace(value) == "" {
				return fallback
			}
			return value
		},
	}
}
