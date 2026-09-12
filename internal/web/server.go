package web

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitli/internal/assets"
	"gitli/internal/db"
	"gitli/internal/users"
)

// Server 装配路由与依赖。
type Server struct {
	q     db.Querier
	users *users.Service
	r     *Renderer
	mux   *chi.Mux
}

func NewServer(q db.Querier) (*Server, error) {
	r, err := NewRenderer()
	if err != nil {
		return nil, err
	}
	s := &Server{
		q:     q,
		users: users.NewService(q),
		r:     r,
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	m := s.mux
	m.Use(middleware.RealIP)
	m.Use(middleware.RequestID)
	m.Use(middleware.Recoverer)
	m.Use(SecureHeaders)
	m.Use(LoadSession(s.q))
	m.Use(VerifyCSRF)

	staticFS, _ := fs.Sub(assets.StaticFS(), "static")
	m.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	m.Get("/", s.handleHome)
	m.Get("/login", s.showLogin)
	m.Post("/login", s.doLogin)
	m.Get("/register", s.showRegister)
	m.Post("/register", s.doRegister)
	m.Post("/logout", s.doLogout)

	m.NotFound(func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, http.StatusNotFound, "页面不存在")
	})
	m.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		s.renderError(w, http.StatusMethodNotAllowed, "不支持的请求方法")
	})
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
