package app

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"pg_provision_dbuser_codex/internal/config"
	"pg_provision_dbuser_codex/internal/password"
	"pg_provision_dbuser_codex/internal/provision"
	"pg_provision_dbuser_codex/internal/session"
	webassets "pg_provision_dbuser_codex/web"
)

type Server struct {
	cfg       config.Config
	drafts    *provision.DraftStore
	executor  *provision.Executor
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
		drafts:    provision.NewDraftStore(15 * time.Minute),
		executor:  provision.NewExecutor(cfg.Postgres, provision.NewPGXRunner),
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
	s.mux.Handle("GET /api/password", s.requireAuth(http.HandlerFunc(s.handlePassword)))
	s.mux.Handle("POST /preview", s.requireAuth(http.HandlerFunc(s.handlePreview)))
	s.mux.Handle("POST /execute", s.requireAuth(http.HandlerFunc(s.handleExecute)))
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLogin)
	s.mux.HandleFunc("POST /logout", s.handleLogout)
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	s.render(w, "home.html", s.homeData(provision.Request{}, nil, ""))
}

func (s *Server) homeData(values provision.Request, errors []provision.FieldError, message string) map[string]any {
	return map[string]any{
		"Title":         "PostgreSQL 用户开通工具",
		"TargetSummary": s.cfg.TargetSummary(),
		"Values":        values.Normalized(),
		"Errors":        errors,
		"Message":       message,
	}
}

func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request) {
	generated, err := password.Generate(password.DefaultLength)
	if err != nil {
		http.Error(w, "生成密码失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"password": generated,
	})
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	username, ok := s.sessions.Username(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}

	req := provision.Request{
		RoleName:     r.FormValue("role_name"),
		DatabaseName: r.FormValue("database_name"),
		RolePassword: r.FormValue("role_password"),
	}.Normalized()
	if errors := provision.ValidateRequest(req); len(errors) > 0 {
		req.RolePassword = ""
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, "home.html", s.homeData(req, errors, "请修正表单后再预览 SQL"))
		return
	}

	draft, err := s.drafts.Save(username, req)
	if err != nil {
		http.Error(w, "创建预览草稿失败", http.StatusInternalServerError)
		return
	}

	s.render(w, "preview.html", map[string]any{
		"Title":         "SQL 预览",
		"TargetSummary": s.cfg.TargetSummary(),
		"Draft":         draft,
	})
}

func (s *Server) handleExecute(w http.ResponseWriter, r *http.Request) {
	username, ok := s.sessions.Username(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "表单解析失败", http.StatusBadRequest)
		return
	}
	draft, ok := s.drafts.Get(r.FormValue("draft_id"), username)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		s.render(w, "result.html", map[string]any{
			"Title": "执行结果",
			"Error": "预览草稿不存在或已过期，请重新提交预览",
		})
		return
	}

	result := s.executor.Execute(r.Context(), draft.Request)
	s.drafts.Delete(draft.ID)
	s.render(w, "result.html", map[string]any{
		"Title":  "执行结果",
		"Draft":  draft,
		"Result": result,
	})
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.sessions.Username(r); ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	s.render(w, "login.html", map[string]any{
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
		s.render(w, "login.html", map[string]any{
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

func (s *Server) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "页面渲染失败", http.StatusInternalServerError)
	}
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
