package web

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"gitli/internal/auth"
	"gitli/internal/db"
	"gitli/internal/git"
	"gitli/internal/repos"
)

const commitsPerPage = 50

// browseCtx 每个代码浏览 handler 的公共前置数据。
type browseCtx struct {
	repo    db.Repo
	owner   string
	repoCtx *repoCtx
}

// repoCtx 模板公共数据（repo 头部 partial 用）。
type repoCtx struct {
	OwnerName   string
	RepoName    string
	Description string
	Visibility  string
	CloneURL    string
	IsOwner     bool
	ActiveTab   string
	Ref         string // 当前分支/引用（顶部 ref 选择器用）
}

// loadBrowseCtx 公共前置：取仓库 → CanAccess → 404 兜底。
func (s *Server) loadBrowseCtx(w http.ResponseWriter, r *http.Request) (*browseCtx, bool) {
	ownerName := r.PathValue("owner")
	repoName := r.PathValue("repo")
	oi, err := s.repos.GetWithKind(r.Context(), ownerName, repoName)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return nil, false
	}
	repo := oi.Repo
	user := UserFromContext(r.Context())
	if !auth.CanAccess(r.Context(), s.q, user, repo, false) {
		s.renderError(w, http.StatusNotFound, "仓库不存在")
		return nil, false
	}
	return &browseCtx{
		repo:  repo,
		owner: oi.OwnerName,
		repoCtx: &repoCtx{
			OwnerName:   oi.OwnerName,
			RepoName:    repo.Name,
			Description: repo.Description,
			Visibility:  repo.Visibility,
			CloneURL:    s.cloneURL(oi.OwnerName, repo.Name),
			IsOwner:     user != nil && (user.ID == repo.OwnerID || (oi.Kind == repos.OwnerOrg && s.orgs.UserRole(r.Context(), repo.OwnerID, user.ID) == "owner")),
		},
	}, true
}

func (bc *browseCtx) data(activeTab, ref string) browseData {
	bc.repoCtx.ActiveTab = activeTab
	bc.repoCtx.Ref = ref
	return browseData{Repo: *bc.repoCtx}
}

// browseData 代码浏览页面公共数据壳。
type browseData struct {
	Repo          repoCtx
	Entries       []git.TreeEntry
	ParentDir     string
	CommitSummary string
	CommitShort   string
	CommitTime    int64
	Branches      []git.BranchRef
	Tags          []git.BranchRef
	READMEHTML    template.HTML
	HasREADME     bool
	Commits       []git.Commit
	PrevPage      int
	NextPage      int
	HasPrev       bool
	HasNext       bool
	EmptyRepo     bool
	Commit        git.Commit
	Diff          []git.DiffLine
	BlobName      string
	BlobPath      string
	BlobLines     []string
	BlobTooLarge  bool
	BlobBinary    bool
	BlobSize      int64
	Blame         []git.BlameLine
	RenderErr     string
}

// refOr404 校验并解析 ref，失败渲染 404。
func (s *Server) refOr404(w http.ResponseWriter, r *http.Request, repoPath, ref string) (string, bool) {
	if err := git.ValidateRef(ref); err != nil {
		s.renderError(w, http.StatusNotFound, "无效的引用")
		return "", false
	}
	if _, err := git.ResolveRef(r.Context(), repoPath, ref); err != nil {
		s.renderError(w, http.StatusNotFound, "引用不存在")
		return "", false
	}
	return ref, true
}

func (s *Server) repoBasePath(owner, name string) string {
	return "/" + owner + "/" + name
}

// treeURL 拼接 tree/blob/raw/blame 链接。
func treeURL(kind, owner, name, ref, p string) string {
	if p == "" {
		return fmt.Sprintf("/%s/%s/%s/%s", owner, name, kind, ref)
	}
	return fmt.Sprintf("/%s/%s/%s/%s/%s", owner, name, kind, ref, p)
}

// handleTree GET /{owner}/{repo}/tree/{ref} 与 /tree/{ref}/{path...}
func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	ref := r.PathValue("ref")
	if _, ok := s.refOr404(w, r, repoPath, ref); !ok {
		return
	}
	dirPath := pathValueRest(r, "path")

	// 若指向 blob，跳转到 blob 视图
	entries, err := git.ParseLsTree(r.Context(), repoPath, ref, dirPath)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "路径不存在")
		return
	}
	if len(entries) == 1 && entries[0].Type == "blob" && dirPath != "" {
		http.Redirect(w, r, treeURL("blob", bc.owner, bc.repo.Name, ref, dirPath), http.StatusSeeOther)
		return
	}

	data := bc.data("files", ref)
	data.Entries = sortTreeEntries(entries)
	data.ParentDir = path.Dir(dirPath)
	if data.ParentDir == "." {
		data.ParentDir = ""
	}

	// 目录最近提交摘要
	if log, err := git.LogCommits(r.Context(), repoPath, ref, 1, 0); err == nil && len(log) == 1 {
		data.CommitShort = log[0].Short
		data.CommitSummary = log[0].Subject
		data.CommitTime = log[0].Time
	}

	s.render(w, http.StatusOK, "repo_tree", pageData{
		Title: bc.owner + "/" + bc.repo.Name,
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

func sortTreeEntries(entries []git.TreeEntry) []git.TreeEntry {
	sorted := make([]git.TreeEntry, len(entries))
	copy(sorted, entries)
	// 目录在前，名称升序
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			ti, tj := sorted[i].Type == "tree", sorted[j].Type == "tree"
			if tj && !ti || (ti == tj && sorted[j].Name < sorted[i].Name) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

// handleBlob GET /{owner}/{repo}/blob/{ref}/{path...}
func (s *Server) handleBlob(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	ref := r.PathValue("ref")
	if _, ok := s.refOr404(w, r, repoPath, ref); !ok {
		return
	}
	filePath := pathValueRest(r, "path")
	if err := git.ValidatePath(filePath); err != nil || filePath == "" {
		s.renderError(w, http.StatusNotFound, "无效的文件路径")
		return
	}

	content, err := git.GetBlob(r.Context(), repoPath, ref, filePath)
	if err != nil {
		// 若是目录则跳转 tree 视图
		entries, terr := git.ParseLsTree(r.Context(), repoPath, ref, filePath)
		if terr == nil && len(entries) > 0 {
			http.Redirect(w, r, treeURL("tree", bc.owner, bc.repo.Name, ref, filePath), http.StatusSeeOther)
			return
		}
		s.renderError(w, http.StatusNotFound, "文件不存在")
		return
	}

	data := bc.data("files", ref)
	data.BlobName = path.Base(filePath)
	data.BlobPath = filePath
	data.BlobSize = int64(len(content))
	data.ParentDir = path.Dir(filePath)
	if data.ParentDir == "." {
		data.ParentDir = ""
	}

	switch {
	case git.IsBinary(content):
		data.BlobBinary = true
	case len(content) > 1<<20:
		data.BlobTooLarge = true
	default:
		text := strings.ReplaceAll(string(content), "\r\n", "\n")
		data.BlobLines = strings.Split(text, "\n")
	}

	s.render(w, http.StatusOK, "repo_blob", pageData{
		Title: fmt.Sprintf("%s: %s", bc.repo.Name, filePath),
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

// handleRaw GET /{owner}/{repo}/raw/{ref}/{path...}
func (s *Server) handleRaw(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	ref := r.PathValue("ref")
	if _, ok := s.refOr404(w, r, repoPath, ref); !ok {
		return
	}
	filePath := pathValueRest(r, "path")
	if err := git.ValidatePath(filePath); err != nil || filePath == "" {
		s.renderError(w, http.StatusNotFound, "无效的文件路径")
		return
	}

	content, err := git.GetBlob(r.Context(), repoPath, ref, filePath)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "文件不存在")
		return
	}
	if git.IsBinary(content) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Write(content)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(content)
}

// handleCommits GET /{owner}/{repo}/commits/{ref}?page=N
func (s *Server) handleCommits(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	ref := r.PathValue("ref")
	if _, ok := s.refOr404(w, r, repoPath, ref); !ok {
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	// 多取 1 条判断是否有下一页
	commits, err := git.LogCommits(r.Context(), repoPath, ref, commitsPerPage+1, (page-1)*commitsPerPage)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无法读取提交历史")
		return
	}
	hasNext := len(commits) > commitsPerPage
	if hasNext {
		commits = commits[:commitsPerPage]
	}

	data := bc.data("commits", ref)
	data.Commits = commits
	data.HasPrev = page > 1
	data.HasNext = hasNext
	data.PrevPage = page - 1
	data.NextPage = page + 1

	s.render(w, http.StatusOK, "repo_commits", pageData{
		Title: fmt.Sprintf("%s: 提交历史", bc.repo.Name),
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

// handleCommit GET /{owner}/{repo}/commit/{sha}
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	sha := r.PathValue("sha")
	if err := git.ValidateRef(sha); err != nil {
		s.renderError(w, http.StatusNotFound, "无效的提交标识")
		return
	}

	commit, diff, err := git.GetCommit(r.Context(), repoPath, sha)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "提交不存在")
		return
	}

	data := bc.data("commits", commit.Short)
	data.Commit = commit
	data.Diff = diff

	s.render(w, http.StatusOK, "repo_commit", pageData{
		Title: fmt.Sprintf("%s: %s", bc.repo.Name, commit.Subject),
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

// handleBlame GET /{owner}/{repo}/blame/{ref}/{path...}
func (s *Server) handleBlame(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)
	ref := r.PathValue("ref")
	if _, ok := s.refOr404(w, r, repoPath, ref); !ok {
		return
	}
	filePath := pathValueRest(r, "path")
	if err := git.ValidatePath(filePath); err != nil || filePath == "" {
		s.renderError(w, http.StatusNotFound, "无效的文件路径")
		return
	}

	lines, err := git.Blame(r.Context(), repoPath, ref, filePath)
	if err != nil {
		s.renderError(w, http.StatusNotFound, "无法读取 blame")
		return
	}

	data := bc.data("files", ref)
	data.BlobName = path.Base(filePath)
	data.BlobPath = filePath
	data.Blame = lines

	s.render(w, http.StatusOK, "repo_blame", pageData{
		Title: fmt.Sprintf("%s: blame %s", bc.repo.Name, filePath),
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

// handleRefs GET /{owner}/{repo}/branches 与 /tags（同一模板）
func (s *Server) handleRefs(w http.ResponseWriter, r *http.Request) {
	bc, ok := s.loadBrowseCtx(w, r)
	if !ok {
		return
	}
	repoPath := s.repos.DiskPath(bc.owner, bc.repo.Name)

	branches, err := git.ListBranches(r.Context(), repoPath)
	if err != nil {
		s.renderError(w, http.StatusInternalServerError, "无法读取分支")
		return
	}
	tags, _ := git.ListTags(r.Context(), repoPath)

	activeTab := "branches"
	if strings.HasPrefix(r.URL.Path, "/tags") || strings.HasSuffix(r.URL.Path, "/tags") {
		activeTab = "tags"
	}

	data := bc.data(activeTab, "")
	data.Branches = branches
	data.Tags = tags

	s.render(w, http.StatusOK, "repo_refs", pageData{
		Title: fmt.Sprintf("%s: 分支与标签", bc.repo.Name),
		User:  UserFromContext(r.Context()),
		CSRF:  CSRFFromContext(r.Context()),
		Data:  data,
	})
}

func pathValueRest(r *http.Request, _ string) string {
	v := chi.URLParam(r, "*")
	if v == "" {
		return ""
	}
	return strings.TrimPrefix(v, "/")
}

// findREADME 在目录树中查找 README 文件（优先 README.md）。
func findREADME(entries []git.TreeEntry) (git.TreeEntry, bool) {
	var fallback *git.TreeEntry
	for _, e := range entries {
		if e.Type != "blob" {
			continue
		}
		lower := strings.ToLower(e.Name)
		if lower == "readme.md" {
			return e, true
		}
		if lower == "readme" || lower == "readme.txt" {
			fallback = &e
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return git.TreeEntry{}, false
}
