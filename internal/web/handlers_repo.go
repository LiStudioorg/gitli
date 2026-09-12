package web

import (
	"errors"
	"fmt"
	"net/http"

	"gitli/internal/auth"
	"gitli/internal/git"
	"gitli/internal/repos"
	"gitli/internal/users"
)

type userData struct {
	OwnerName string
	Repos     []repoView
	Repo      repoView
	Flash     Flash
}

type repoView struct {
	ID          int64
	Name        string
	Description string
	Visibility  string
	CloneURL    string
	IsOwner     bool
}

func (s *Server) showUser(w http.ResponseWriter, r *http.Request) {
	ownerName := r.PathValue("owner")
	flash := Flash{}
	owner, err := s.users.GetByUsername(r.Context(), ownerName)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			s.renderError(w, http.StatusNotFound, "用户不存在")
			return
		}
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}

	user := UserFromContext(r.Context())
	isSelf := user != nil && user.ID == owner.ID

	dbRepos, err := s.repos.ListByOwner(r.Context(), owner.Username)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	repoViews := make([]repoView, 0, len(dbRepos))
	for _, rp := range dbRepos {
		// 仅展示可见仓库
		if !auth.CanAccess(r.Context(), s.q, user, rp, false) {
			continue
		}
		repoViews = append(repoViews, repoView{
			ID:          rp.ID,
			Name:        rp.Name,
			Description: rp.Description,
			Visibility:  rp.Visibility,
			CloneURL:    s.cloneURL(owner.Username, rp.Name),
		})
	}

	s.render(w, http.StatusOK, "user", pageData{
		Title: owner.Username,
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: userData{
			OwnerName: owner.Username,
			Repos:     repoViews,
			Flash:     flash,
		},
	})
	_ = isSelf
}

func (s *Server) doCreateRepo(w http.ResponseWriter, r *http.Request) {
	ownerName := r.PathValue("owner")
	user := UserFromContext(r.Context())
	if user == nil || auth.NormalizeUsername(ownerName) != user.Username {
		s.renderError(w, http.StatusForbidden, "只能在自己的主页创建仓库")
		return
	}
	name := r.PostFormValue("name")
	desc := r.PostFormValue("description")
	vis := r.PostFormValue("visibility")

	_, err := s.repos.Create(r.Context(), *user, name, desc, vis)
	if err != nil {
		msg := "创建失败"
		switch {
		case errors.Is(err, repos.ErrNameTaken):
			msg = "仓库名已存在"
		case errors.Is(err, repos.ErrInvalidInput):
			msg = err.Error()
		default:
			s.renderError(w, http.StatusInternalServerError, "internal error")
			return
		}
		owner, uerr := s.users.GetByUsername(r.Context(), ownerName)
		if uerr != nil {
			s.renderError(w, http.StatusInternalServerError, "internal error")
			return
		}
		list, _ := s.repos.ListByOwner(r.Context(), owner.Username)
		views := make([]repoView, 0, len(list))
		for _, rp := range list {
			views = append(views, repoView{Name: rp.Name, Visibility: rp.Visibility, Description: rp.Description})
		}
		s.render(w, http.StatusBadRequest, "user", pageData{
			Title: owner.Username,
			User:  user,
			CSRF:  CSRFFromContext(r.Context()),
			Data:  userData{OwnerName: owner.Username, Repos: views, Flash: Flash{Kind: "error", Message: msg}},
		})
		return
	}
	http.Redirect(w, r, "/"+ownerName+"/"+name, http.StatusSeeOther)
}

func (s *Server) showRepo(w http.ResponseWriter, r *http.Request) {
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	repo, owner, err := s.repos.Get(r.Context(), ownerName, repoName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	user := UserFromContext(r.Context())
	if !auth.CanAccess(r.Context(), s.q, user, repo, false) {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return
	}
	s.render(w, http.StatusOK, "repo", pageData{
		Title: ownerName + "/" + repoName,
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: userData{
			OwnerName: ownerName,
			Repo: repoView{
				ID:          repo.ID,
				Name:        repo.Name,
				Description: repo.Description,
				Visibility:  repo.Visibility,
				CloneURL:    s.cloneURL(owner.Username, repo.Name),
				IsOwner:     user != nil && user.ID == repo.OwnerID,
			},
		},
	})
}

func (s *Server) cloneURL(owner, repo string) string {
	return fmt.Sprintf("%s/%s/%s.git", s.rootURL, owner, repo)
}

var _ = git.ValidateRepoName
