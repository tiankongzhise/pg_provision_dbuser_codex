package app

import (
	"html/template"
	"io/fs"
	"net/http"

	webassets "pg_provision_dbuser_codex/web"
)

type Server struct {
	mux       *http.ServeMux
	templates *template.Template
}

func New() (*Server, error) {
	templates, err := template.ParseFS(webassets.Files, "templates/*.html")
	if err != nil {
		return nil, err
	}

	server := &Server{
		mux:       http.NewServeMux(),
		templates: templates,
	}
	server.routes()
	return server, nil
}

func (s *Server) Routes() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	staticRoot, err := fs.Sub(webassets.Files, "static")
	if err == nil {
		s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticRoot))))
	}

	s.mux.HandleFunc("GET /", s.handleHome)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	_ = s.templates.ExecuteTemplate(w, "home.html", map[string]any{
		"Title": "PostgreSQL 用户开通工具",
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}
