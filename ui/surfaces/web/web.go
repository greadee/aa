// Package web is the console's local web surface: a read-only inspector over
// the control plane. It serves semantic HTML (and a plain-text alternative) and
// takes no commands in v1.
package web

import (
	"html/template"
	"net/http"

	"github.com/greadee/aa/ui/controlplane"
	"github.com/greadee/aa/ui/state"
	"github.com/greadee/aa/ui/views"
)

// Server is the read-only web surface.
type Server struct {
	client    controlplane.Client
	projectID string
	tmpl      *template.Template
}

// New returns a web surface over a client.
func New(client controlplane.Client, projectID string) *Server {
	return &Server{client: client, projectID: projectID, tmpl: template.Must(template.New("page").Parse(pageHTML))}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.page)
	mux.HandleFunc("GET /plain", s.plain)
	return mux
}

func (s *Server) load(r *http.Request) (state.Model, error) {
	return state.Load(r.Context(), s.client, s.projectID)
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	m, err := s.load(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) plain(w http.ResponseWriter, r *http.Request) {
	m, err := s.load(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(views.RenderPlain(m)))
}

const pageHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>aa console</title></head>
<body>
<h1>aa console</h1>
<nav><a href="/plain">plain text</a></nav>
<p>projects={{.Status.Projects}} nodes={{.Status.Nodes}} active_assignments={{.Status.ActiveAssignments}}</p>

<section><h2>Projects</h2>
{{if .Projects}}<ul>{{range .Projects}}<li>{{.ID}} — {{.Name}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Work ({{.ProjectID}})</h2>
{{if .Work}}<table><tr><th>id</th><th>state</th><th>title</th></tr>{{range .Work}}<tr><td>{{.ID}}</td><td>{{.State}}</td><td>{{.Title}}</td></tr>{{end}}</table>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Graph ({{.ProjectID}})</h2>
{{if .Graph.Nodes}}<ul>{{range .Graph.Nodes}}<li>{{.WorkPackageID}} — {{.Readiness}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Memory</h2>
<h3>Issues</h3>
{{if .Issues}}<ul>{{range .Issues}}<li>{{.ID}} — {{.Status}} {{.Title}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
<h3>Strategies</h3>
{{if .Strategies}}<ul>{{range .Strategies}}<li>{{.ID}} — {{.Lifecycle}} {{.Title}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Reports</h2>
{{if .Sessions}}<ul>{{range .Sessions}}<li>{{.ID}} — events={{.EventCount}} project={{.ProjectID}}</li>{{end}}</ul>{{else}}<p>(no sessions)</p>{{end}}
{{if .Report.SessionID}}<p>report {{.Report.SessionID}}: events={{.Report.Events}} sequences={{.Report.FirstSequence}}..{{.Report.LastSequence}}</p>{{end}}
</section>

<section><h2>Approvals</h2>
{{if .Assignments}}<ul>{{range .Assignments}}<li>{{.ID}} — {{.State}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Nodes</h2>
{{if .Nodes}}<ul>{{range .Nodes}}<li>{{.ID}} — {{.State}}</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>

<section><h2>Tools</h2>
{{if .Tools}}<ul>{{range .Tools}}<li>{{.ID}} ({{.Kind}})</li>{{end}}</ul>{{else}}<p>(none)</p>{{end}}
</section>
</body></html>`
