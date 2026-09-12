package web

import (
	"errors"
	"net/http"

	"gitli/internal/auth"
	"gitli/internal/db"
	"gitli/internal/git"
	"gitli/internal/orgs"
	"gitli/internal/repos"
)

type orgData struct {
	Org         db.Org
	Members     []db.ListOrgMembersWithUsersRow
	Repos       []repoView
	IsMember    bool
	Role        string
	CurrentRole string
	Flash       Flash
	AddUser     string
}

func (s *Server) showOrgNew(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	s.render(w, http.StatusOK, "org_new", pageData{
		Title: "新建组织",
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data:  Flash{},
	})
}

func (s *Server) doOrgCreate(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	name := r.PostFormValue("name")
	desc := r.PostFormValue("description")
	org, err := s.orgs.CreateOrg(r.Context(), name, desc)
	if err != nil {
		msg := "创建失败"
		switch {
		case errors.Is(err, orgs.ErrNameTaken):
			msg = "组织名已被占用"
		case errors.Is(err, orgs.ErrInvalidInput):
			msg = err.Error()
		default:
			s.renderError(w, http.StatusInternalServerError, "internal error")
			return
		}
		s.render(w, http.StatusBadRequest, "org_new", pageData{
			Title: "新建组织",
			User:  user,
			CSRF:  CSRFFromContext(r.Context()),
			Data:  Flash{Kind: "error", Message: msg},
		})
		return
	}
	// 创建者自动成为第一个 owner 成员
	if err := s.orgs.AddMember(r.Context(), org.ID, user.ID, "owner"); err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.Redirect(w, r, "/orgs/"+org.Name, http.StatusSeeOther)
}

func (s *Server) showOrg(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	org, err := s.orgs.GetByName(r.Context(), name)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "组织不存在")
		return
	}
	user := UserFromContext(r.Context())
	members, err := s.orgs.ListMembers(r.Context(), org.ID)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	isMember := user != nil && s.orgs.IsMember(r.Context(), org.ID, user.ID)
	role := ""
	if user != nil {
		role = s.orgs.UserRole(r.Context(), org.ID, user.ID)
	}

	dbRepos, err := s.orgs.ListRepos(r.Context(), org.Name)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	views := make([]repoView, 0, len(dbRepos))
	for _, rp := range dbRepos {
		// org/private 仓库仅成员可见（CanAccess 统一判定）
		if !auth.CanAccess(r.Context(), s.q, user, rp, false) {
			continue
		}
		views = append(views, repoView{
			ID:          rp.ID,
			Name:        rp.Name,
			Description: rp.Description,
			Visibility:  rp.Visibility,
			CloneURL:    s.cloneURL(org.Name, rp.Name),
		})
	}

	s.render(w, http.StatusOK, "org", pageData{
		Title: org.Name,
		User:  user,
		CSRF:  CSRFFromContext(r.Context()),
		Data: orgData{
			Org:         org,
			Members:     members,
			Repos:       views,
			IsMember:    isMember,
			CurrentRole: role,
			Flash:       Flash{},
		},
	})
}

func (s *Server) doOrgAddMember(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	orgName := r.PathValue("name")
	org, err := s.orgs.GetByName(r.Context(), orgName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "组织不存在")
		return
	}
	if user == nil || s.orgs.UserRole(r.Context(), org.ID, user.ID) != "owner" {
		s.renderError(w, http.StatusForbidden, "仅组织 owner 可添加成员")
		return
	}
	username := auth.NormalizeUsername(r.PostFormValue("username"))
	target, err := s.users.GetByUsername(r.Context(), username)
	if err != nil {
		http.Redirect(w, r, "/orgs/"+orgName, http.StatusSeeOther)
		return
	}
	if err := s.orgs.AddMember(r.Context(), org.ID, target.ID, "member"); err != nil {
		s.renderError(w, http.StatusInternalServerError, "internal error")
		return
	}
	http.Redirect(w, r, "/orgs/"+orgName, http.StatusSeeOther)
}

func (s *Server) doOrgCreateRepo(w http.ResponseWriter, r *http.Request) {
	user := UserFromContext(r.Context())
	orgName := r.PathValue("name")
	org, err := s.orgs.GetByName(r.Context(), orgName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "组织不存在")
		return
	}
	if user == nil || s.orgs.UserRole(r.Context(), org.ID, user.ID) != "owner" {
		s.renderError(w, http.StatusForbidden, "仅组织 owner 可创建仓库")
		return
	}
	name := r.PostFormValue("name")
	desc := r.PostFormValue("description")
	vis := r.PostFormValue("visibility")

	_, err = s.repos.CreateInOrg(r.Context(), org, name, desc, vis)
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
		// 重新渲染组织页并带错误信息
		members, _ := s.orgs.ListMembers(r.Context(), org.ID)
		dbRepos, _ := s.orgs.ListRepos(r.Context(), org.Name)
		views := make([]repoView, 0, len(dbRepos))
		for _, rp := range dbRepos {
			if !auth.CanAccess(r.Context(), s.q, user, rp, false) {
				continue
			}
			views = append(views, repoView{Name: rp.Name, Visibility: rp.Visibility, Description: rp.Description})
		}
		s.render(w, http.StatusBadRequest, "org", pageData{
			Title: org.Name,
			User:  user,
			CSRF:  CSRFFromContext(r.Context()),
			Data: orgData{
				Org: org, Members: members, Repos: views,
				IsMember: true, CurrentRole: "owner",
				Flash: Flash{Kind: "error", Message: msg},
			},
		})
		return
	}
	http.Redirect(w, r, "/"+orgName+"/"+name, http.StatusSeeOther)
}

var _ = git.ValidateRepoName
var _ = db.Org{}
