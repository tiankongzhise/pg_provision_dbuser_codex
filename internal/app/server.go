package app

import (
	"html/template"
	"io/fs"
	"net/http"

	"pg_provision_dbuser_codex/internal/config"
	"pg_provision_dbuser_codex/internal/session"
	webassets "pg_provision_dbuser_codex/web"
)

type Server struct {
	cfg       config.Config
	mux       *http.ServeMux
	sessions  *session.Manager
	templates *template.Template
}

func New(cfg config.Config) (*Server, error) {
	templates, err := template.ParseFS(webassets.Files, "templates/*.html")
	if err != nil {
		return nil, err
	}

	server := &Server{
		cfg:       cfg,
		mux:       http.NewServeMux(),
		sessions:  session.NewManager(cfg.AppSessionSecret),
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

	s.mux.Handle("GET /", s.requireAuth(http.HandlerFunc(s.handleHome)))
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLogin)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	_ = s.templates.ExecuteTemplate(w, "home.html", map[string]any{
		"Title":         "PostgreSQL 用户开通工具",
		"TargetSummary": s.cfg.TargetSummary(),
	})
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.sessions.Username(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	_ = s.templates.ExecuteTemplate(w, "login.html", map[string]any{
		"Title": "管理员登录",
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	loginKey := r.FormValue("login_key")
	if username != s.cfg.AppLoginUser || loginKey != s.cfg.AppLoginKey {
		w.WriteHeader(http.StatusUnauthorized)
		_ = s.templates.ExecuteTemplate(w, "login.html", map[string]any{
			"Title": "管理员登录",
			"Error": "账号或密钥错误",
		})
		return
	}

	s.sessions.Issue(w, username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.sessions.Clear(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.sessions.Username(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}
