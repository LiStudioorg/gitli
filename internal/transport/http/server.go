// SPDX-License-Identifier: AGPL-3.0
// Package http assembles the HTTP transport: router, middleware and handlers.
package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/LiStudioorg/gitli/internal/transport/http/middleware"
	"github.com/LiStudioorg/gitli/internal/version"
	"github.com/LiStudioorg/gitli/web"
)

// Server holds dependencies for the HTTP transport.
type Server struct {
	deps   *Deps
	apiLim *middleware.RateLimiter
}

// NewServer constructs the HTTP transport.
func NewServer(deps *Deps) *Server {
	return &Server{
		deps:   deps,
		apiLim: middleware.NewRateLimiter(600, time.Minute),
	}
}

// Router builds and returns the fully wired chi router.
func (s *Server) Router() http.Handler {
	d := s.deps
	r := chi.NewRouter()
	r.Use(middleware.Logger(d.Log))
	r.Use(middleware.Recover(d.Log))
	r.Use(middleware.SecurityHeaders(d.HTTPSecure()))
	r.Use(middleware.CORS(d.ExternalBase()))
	r.Use(s.usernameRedirect)

	r.Get("/healthz", s.handleHealth)
	r.Get("/metrics", s.handleMetrics)
	r.Get("/api/openapi.json", s.handleOpenAPI)
	r.Get("/api/docs", s.handleAPIDocs)

	// OCI Distribution API (/v2). No CSRF; auth via session, PAT or Basic.
	r.Group(func(vr chi.Router) {
		vr.Use(d.Auth.AuthenticateMiddleware)
		vr.Get("/v2/", s.handleV2Check)
		vr.Post("/v2/{owner}/{repo}/blobs/uploads/", s.handleBlobUploadStart)
		vr.Put("/v2/{owner}/{repo}/blobs/uploads/{uuid}", s.handleBlobUploadFinish)
		vr.Get("/v2/{owner}/{repo}/blobs/{digest}", s.handleGetBlob)
		vr.Put("/v2/{owner}/{repo}/manifests/{reference}", s.handlePutManifest)
		vr.Get("/v2/{owner}/{repo}/manifests/{reference}", s.handleGetManifest)
		vr.Get("/v2/{owner}/{repo}/tags/list", s.handleListTags)
	})

	// Package registry (npm/go/maven/generic). No CSRF; package clients
	// authenticate with HTTP Basic (password or PAT).
	r.Group(func(pg chi.Router) {
		pg.Use(d.Auth.AuthenticateMiddleware)
		pg.Use(middleware.MaxBody(middleware.BodyLimitHuge))
		pg.Get("/api/v1/repos/{owner}/{repo}/packages/{type}", s.handleListPackageVersions)
		pg.Post("/api/v1/repos/{owner}/{repo}/packages/{type}/{name}/{version}", s.handlePublishPackageVersion)
		pg.Get("/api/v1/repos/{owner}/{repo}/packages/{type}/{name}/{version}/files/{filename}", s.handleDownloadPackageFile)
	})

	// Git smart HTTP protocol. No CSRF (git clients), Basic/PAT auth.
	r.Group(func(gr chi.Router) {
		gr.Use(d.Auth.AuthenticateMiddleware)
		gr.Get("/{owner}/{repo}/info/refs", s.handleGitInfoRefs)
		gr.Post("/{owner}/{repo}/git-upload-pack", func(w http.ResponseWriter, r *http.Request) {
			s.handleGitService(w, r, "git-upload-pack")
		})
		gr.Post("/{owner}/{repo}/git-receive-pack", func(w http.ResponseWriter, r *http.Request) {
			s.handleGitService(w, r, "git-receive-pack")
		})
	})

	// Runner protocol (/api/v1/runner/*). Auth via Bearer token + X-Runner-ID;
	// no CSRF and no session. Large body for log streaming.
	r.Group(func(rn chi.Router) {
		rn.Use(middleware.MaxBody(middleware.BodyLimitHuge))
		rn.Post("/api/v1/runner/heartbeat", s.handleRunnerHeartbeat)
		rn.Post("/api/v1/runner/register", s.handleRunnerRegister)
		rn.Get("/api/v1/runner/jobs/next", s.handleRunnerNextJob)
		rn.Post("/api/v1/runner/jobs/{id}/logs", s.handleRunnerJobLogs)
		rn.Post("/api/v1/runner/jobs/{id}/result", s.handleRunnerJobResult)
	})

	api := chi.NewRouter()
	api.Use(middleware.MaxBody(middleware.BodyLimitLarge))
	api.Use(d.Auth.AuthenticateMiddleware)
	api.Use(middleware.CSRF())

	api.Get("/version", s.handleVersion)
	api.Get("/search", s.handleGlobalSearch)
	api.Get("/oauth/authorize-info", s.handleOAuthAuthorizeInfo)
	api.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/oauth/approve", s.handleOAuthApprove)

	api.Route("/admin", func(ar chi.Router) {
		ar.Get("/stats", s.handleAdminStats)
		ar.Get("/audit", s.handleAuditQuery)
		ar.Get("/audit/stats", s.handleAuditStats)
		ar.Get("/audit/export", s.handleAuditExport)
		ar.Post("/audit/archive", s.handleAuditArchive)
		ar.Get("/audit/alert-rules", s.handleListAlertRules)
		ar.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/audit/alert-rules", s.handleCreateAlertRule)
		ar.Delete("/audit/alert-rules/{id}", s.handleDeleteAlertRule)
		ar.Get("/audit/{id}", s.handleAuditGet)
		ar.Get("/runners", s.handleListRunners)
		ar.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/runners", s.handleCreateRunner)
		ar.Patch("/runners/{id}", s.handleUpdateRunner)
		ar.Delete("/runners/{id}", s.handleDeleteRunner)
		ar.Post("/runners/{id}/reset-token", s.handleResetRunnerToken)
		ar.Get("/runners/{id}/jobs", s.handleListRunnerJobsAdmin)
		ar.Get("/ci/runners", s.handleListCIRunners)
		ar.Delete("/repos/{id}/purge", s.handlePurgeDeletedRepo)
		ar.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/ci/runners", s.handleCreateCIRunner)
		ar.Patch("/ci/runners/{id}", s.handleUpdateCIRunner)
		ar.Delete("/ci/runners/{id}", s.handleDeleteCIRunner)
	})

	api.Route("/auth", func(ar chi.Router) {
		ar.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/register", s.handleRegister)
		ar.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/login", s.handleLogin)
		ar.Post("/logout", s.handleLogout)
		ar.Get("/oidc/login", s.handleOIDCLogin)
		ar.Get("/oidc/callback", s.handleOIDCCallback)
		ar.Get("/saml/login", s.handleSAMLLogin)
		ar.Post("/saml/acs", s.handleSAMLACS)
	})

	api.Route("/user", func(ur chi.Router) {
		ur.Get("/me", s.handleMe)
		ur.Get("/repos", s.handleMyRepos)
		ur.Patch("/me", s.handleUpdateProfile)
		ur.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/me/password", s.handleChangePassword)
		ur.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/me/rename", s.handleRename)
		ur.Post("/me/totp/begin", s.handleTOTPBegin)
		ur.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/me/totp/enable", s.handleTOTPEnable)
		ur.With(middleware.MaxBody(middleware.BodyLimitTiny)).Post("/me/totp/disable", s.handleTOTPDisable)
		ur.Get("/me/keys", s.handleListSSHKeys)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/me/keys", s.handleAddSSHKey)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Patch("/me/keys/{id}", s.handleUpdateSSHKey)
		ur.Delete("/me/keys/{id}", s.handleDeleteSSHKey)
		ur.Get("/me/gpg-keys", s.handleListSigningKeys)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/me/gpg-keys", s.handleAddSigningKey)
		ur.Delete("/me/gpg-keys/{id}", s.handleDeleteSigningKey)
		ur.Get("/me/tokens", s.handleListTokens)
		ur.Get("/activity", s.handleMyActivity)
		ur.Get("/deleted-repos", s.handleListDeletedRepos)
		ur.Get("/oauth-apps", s.handleListOAuthApps)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/oauth-apps", s.handleCreateOAuthApp)
		ur.Delete("/oauth-apps/{id}", s.handleDeleteOAuthApp)
		ur.Get("/oauth-authorizations", s.handleListOAuthAuthorizations)
		ur.Delete("/oauth-authorizations/{appId}", s.handleRevokeOAuthAuthorization)
		ur.Get("/saved-replies", s.handleListSavedReplies)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/saved-replies", s.handleCreateSavedReply)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Patch("/saved-replies/{id}", s.handleUpdateSavedReply)
		ur.Delete("/saved-replies/{id}", s.handleDeleteSavedReply)
		ur.Post("/deleted-repos/{id}/restore", s.handleRestoreRepo)
		ur.Get("/me/sessions", s.handleListSessions)
		ur.Delete("/me/sessions", s.handleRevokeOtherSessions)
		ur.Delete("/me/sessions/{id}", s.handleRevokeSession)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/me/tokens", s.handleCreateToken)
		ur.Delete("/me/tokens/{id}", s.handleRevokeToken)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/blocks", s.handleBlockUser)
		ur.Delete("/blocks/{username}", s.handleUnblockUser)
		ur.Get("/blocks", s.handleListBlockedUsers)
		ur.Get("/notification-prefs", s.handleListNotifPrefs)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Put("/notification-prefs", s.handleSetNotifPref)
		ur.Get("/webhooks", s.handleListUserWebhooks)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/webhooks", s.handleCreateUserWebhook)
		ur.Delete("/webhooks/{id}", s.handleDeleteUserWebhook)
		ur.With(middleware.MaxBody(middleware.BodyLimitSmall)).Patch("/webhooks/{id}", s.handleUpdateUserWebhook)
	})

	api.Route("/orgs", func(or chi.Router) {
		or.Post("/", s.handleCreateOrg)
		or.Get("/{org}/2fa-policy", s.handleGetOrg2FAPolicy)
		or.With(middleware.MaxBody(middleware.BodyLimitSmall)).Put("/{org}/2fa-policy", s.handleSetOrg2FAPolicy)
		or.Group(func(gr chi.Router) {
			gr.Use(s.RequireOrgPermission("org", "admin"))
			gr.Post("/{org}/members", s.handleAddOrgMember)
			gr.Delete("/{org}/members/{username}", s.handleRemoveOrgMember)
			gr.Post("/{org}/teams", s.handleCreateTeam)
			gr.Post("/{org}/teams/{team}/reparent", s.handleReparentTeam)
			gr.Post("/{org}/teams/{team}/members", s.handleAddTeamMember)
			gr.Delete("/{org}/teams/{team}/members/{username}", s.handleRemoveTeamMember)
			gr.Post("/{org}/teams/{team}/repos", s.handleGrantTeamRepo)
			gr.Get("/{org}/ip-allowlist", s.handleListIPAllow)
			gr.Post("/{org}/ip-allowlist", s.handleAddIPAllow)
			gr.Delete("/{org}/ip-allowlist/{id}", s.handleDeleteIPAllow)
			gr.Put("/{org}/ip-allowlist/toggle", s.handleToggleIPAllow)
			gr.Post("/{org}/roles", s.handleCreateRole)
			gr.Delete("/{org}/roles/{role}", s.handleDeleteRole)
		})
	})

	api.Route("/repos", func(rr chi.Router) {
		rr.Post("/", s.handleCreateRepo)
		rr.Get("/{owner}/{repo}", s.handleGetRepo)
		rr.Get("/{owner}/{repo}/meta", s.handleRepoMeta)
		rr.Get("/{owner}/{repo}/clone-info", s.handleCloneInfo)
		rr.Get("/{owner}/{repo}/archive/{format}", s.handleArchive)
		rr.Delete("/{owner}/{repo}", s.handleDeleteRepo)
		rr.Post("/{owner}/{repo}/fork", s.handleForkRepo)
		rr.Post("/{owner}/{repo}/transfer", s.handleTransferRepo)
		rr.Post("/{owner}/{repo}/rename", s.handleRenameRepo)
		rr.Post("/{owner}/{repo}/archive", s.handleArchiveRepo)
		rr.Get("/{owner}/{repo}/branches", s.handleBranches)
		rr.Get("/{owner}/{repo}/tree/{ref}", s.handleTree)
		rr.Get("/{owner}/{repo}/tree/{ref}/*", s.handleTree)
		rr.Get("/{owner}/{repo}/blob/{ref}/*", s.handleBlob)
		rr.Get("/{owner}/{repo}/commits/{ref}", s.handleCommits)
		rr.Get("/{owner}/{repo}/commits/{sha}/signature", s.handleCommitSignature)
		rr.Get("/{owner}/{repo}/path-permissions", s.handleListPathPermissions)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/path-permissions", s.handleCreatePathPermission)
		rr.Delete("/{owner}/{repo}/path-permissions/{id}", s.handleDeletePathPermission)
		rr.Get("/{owner}/{repo}/graph", s.handleGraph)
		rr.Get("/{owner}/{repo}/pipelines", s.handleListPipelines)
		rr.Get("/{owner}/{repo}/workflows", s.handleListRepoWorkflowJobs)
		rr.Post("/{owner}/{repo}/workflows/{id}/cancel", s.handleCancelWorkflowJob)
		rr.Get("/{owner}/{repo}/workflows/{id}/logs", s.handleRunnerJobLogStream)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/pipelines", s.handleCreatePipeline)
		rr.Get("/{owner}/{repo}/ide/tree", s.handleIDETree)
		rr.Get("/{owner}/{repo}/ide/file", s.handleIDEFile)
		rr.Get("/{owner}/{repo}/ide/symbols", s.handleIDESymbols)
		rr.Get("/{owner}/{repo}/diff", s.handleDiff)
		rr.Get("/{owner}/{repo}/blame/{ref}/*", s.handleBlame)
		rr.Get("/{owner}/{repo}/protections", s.handleListProtections)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/protections/branches", s.handleAddBranchProtection)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/protections/tags", s.handleAddTagProtection)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/collaborators", s.handleAddCollaborator)
		rr.Get("/{owner}/{repo}/keys", s.handleListDeployKeys)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/keys", s.handleAddDeployKey)
		rr.Patch("/{owner}/{repo}/keys/{id}", s.handleUpdateDeployKey)
		rr.Delete("/{owner}/{repo}/keys/{id}", s.handleDeleteDeployKey)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/webhooks", s.handleCreateWebhook)
		rr.Get("/{owner}/{repo}/webhooks", s.handleListWebhooks)
		rr.Get("/{owner}/{repo}/topics", s.handleListTopics)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Put("/{owner}/{repo}/topics", s.handleSetTopics)
		rr.Get("/{owner}/{repo}/ci/secrets", s.handleListCISecrets)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Put("/{owner}/{repo}/ci/secrets", s.handleSetCISecret)
		rr.Delete("/{owner}/{repo}/ci/secrets/{name}", s.handleDeleteCISecret)
		rr.Get("/{owner}/{repo}/ci/cache", s.handleListCache)
		rr.Delete("/{owner}/{repo}/ci/cache/{key}", s.handleDeleteCache)
		rr.Group(func(gr chi.Router) {
			gr.Use(s.RequirePermission("settings", "write"))
			gr.Post("/{owner}/{repo}/rbac/collaborators", s.handleSetCollaborator)
			gr.Delete("/{owner}/{repo}/rbac/collaborators/{username}", s.handleRemoveCollaborator)
			gr.Post("/{owner}/{repo}/rbac/branch-rules", s.handleAddBranchRule)
		})
		rr.Get("/{owner}/{repo}/rbac/branch-rules", s.handleListBranchRules)
		rr.Get("/{owner}/{repo}/wiki", s.handleListWikiPages)
		rr.Get("/{owner}/{repo}/wiki/{slug}", s.handleGetWikiPage)
		rr.With(middleware.MaxBody(middleware.BodyLimitLarge)).Post("/{owner}/{repo}/wiki", s.handleCreateWikiPage)
		rr.With(middleware.MaxBody(middleware.BodyLimitLarge)).Put("/{owner}/{repo}/wiki/{slug}", s.handleUpdateWikiPage)
		rr.Delete("/{owner}/{repo}/wiki/{slug}", s.handleDeleteWikiPage)
		rr.Get("/{owner}/{repo}/releases", s.handleListReleases)
		rr.Get("/{owner}/{repo}/releases/{tag}", s.handleGetRelease)
		rr.Post("/{owner}/{repo}/releases", s.handleCreateRelease)
		rr.Post("/{owner}/{repo}/releases/{tag}/publish", s.handlePublishRelease)
		rr.Get("/{owner}/{repo}/releases/{tag}/assets", s.handleListReleaseAssets)
		rr.Get("/{owner}/{repo}/issue-templates", s.handleListIssueTemplates)
		rr.Get("/{owner}/{repo}/dependencies", s.handleListDependencies)
		rr.Post("/{owner}/{repo}/dependencies/refresh", s.handleRefreshDependencies)
		rr.Get("/{owner}/{repo}/pr-template", s.handlePullRequestTemplate)
		rr.With(middleware.MaxBody(middleware.BodyLimitHuge)).Post("/{owner}/{repo}/releases/{tag}/assets", s.handleUploadReleaseAsset)
		rr.Get("/{owner}/{repo}/releases/{tag}/assets/{name}", s.handleDownloadReleaseAsset)
		rr.Get("/{owner}/{repo}/search", s.handleSearch)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/lfs/objects/batch", s.handleLFSBatch)
		rr.Put("/{owner}/{repo}/lfs/objects/{oid}", s.handleLFSUpload)
		rr.Get("/{owner}/{repo}/lfs/objects/{oid}", s.handleLFSDownload)
		rr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/{owner}/{repo}/lfs/locks", s.handleCreateLFSLock)
		rr.Post("/{owner}/{repo}/lfs/locks/{id}/unlock", s.handleUnlockLFSLock)
		rr.Get("/{owner}/{repo}/lfs/locks", s.handleListLFSLocks)
		rr.Post("/{owner}/{repo}/mirrors", s.handleCreateMirror)
		rr.Get("/{owner}/{repo}/mirrors", s.handleListMirrors)
		rr.Post("/{owner}/{repo}/imports", s.handleCreateImport)
		rr.Get("/{owner}/{repo}/imports", s.handleListImports)
	})

	api.Route("/users", func(pr chi.Router) {
		pr.Get("/{username}", s.handleUserProfile)
		pr.Get("/{username}/followers", s.handleListFollowers)
		pr.Get("/{username}/following", s.handleListFollowing)
		pr.Post("/{username}/follow", s.handleFollowUser)
		pr.Delete("/{username}/follow", s.handleUnfollowUser)
	})

	api.Route("/gists", func(gr chi.Router) {
		gr.Get("/", s.handleListGists)
		gr.Post("/", s.handleCreateGist)
		gr.Get("/{id}", s.handleGetGist)
		gr.Delete("/{id}", s.handleDeleteGist)
	})

	api.Route("/issues", func(ir chi.Router) {
		ir.With(middleware.MaxBody(middleware.BodyLimitMedium)).Post("/", s.handleCreateIssue)
		ir.Get("/", s.handleListIssues)
		ir.Get("/{number}", s.handleGetIssue)
		ir.Post("/{number}/state", s.handleSetIssueState)
		ir.With(middleware.MaxBody(middleware.BodyLimitMedium)).Patch("/{number}", s.handleUpdateIssue)
		ir.Get("/{number}/comments", s.handleListIssueComments)
		ir.With(middleware.MaxBody(middleware.BodyLimitMedium)).Post("/{number}/comments", s.handleCreateIssueComment)
	})

	api.Route("/labels", func(lr chi.Router) {
		lr.Get("/", s.handleListLabels)
		lr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/", s.handleCreateLabel)
		lr.Delete("/{id}", s.handleDeleteLabel)
	})

	api.Route("/milestones", func(mr chi.Router) {
		mr.Get("/", s.handleListMilestones)
		mr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/", s.handleCreateMilestone)
	})

	api.Route("/pulls", func(pr chi.Router) {
		pr.With(middleware.MaxBody(middleware.BodyLimitMedium)).Post("/", s.handleCreatePull)
		pr.Get("/", s.handleListPulls)
		pr.Get("/{number}", s.handleGetPull)
		pr.Post("/{number}/merge", s.handleMergePull)
		pr.Get("/{number}/reviews", s.handleListReviews)
		pr.With(middleware.MaxBody(middleware.BodyLimitMedium)).Post("/{number}/reviews", s.handleCreateReview)
		pr.Get("/{number}/reviews/{reviewID}/comments", s.handleListReviewComments)
		pr.With(middleware.MaxBody(middleware.BodyLimitMedium)).Post("/{number}/reviews/{reviewID}/comments", s.handleCreateReviewComment)
	})

	api.Route("/reactions", func(rx chi.Router) {
		rx.Get("/", s.handleListReactions)
		rx.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/", s.handleAddReaction)
		rx.With(middleware.MaxBody(middleware.BodyLimitSmall)).Delete("/", s.handleRemoveReaction)
	})

	api.Route("/subscriptions", func(sr chi.Router) {
		sr.Get("/", s.handleGetSubscription)
		sr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Post("/", s.handleSubscribe)
		sr.With(middleware.MaxBody(middleware.BodyLimitSmall)).Delete("/", s.handleUnsubscribe)
	})

	api.Get("/notifications", s.handleListNotifications)
	api.Post("/notifications/read-all", s.handleMarkAllNotificationsRead)
	api.Get("/pipelines/{id}/jobs", s.handleListPipelineJobs)
	api.Get("/pipelines/{id}/artifacts", s.handleListPipelineArtifacts)
	api.Delete("/artifacts/{id}", s.handleDeleteArtifact)
	api.Post("/notifications/{id}/read", s.handleMarkNotificationRead)
	api.Get("/events", s.handleSSE)
	api.Get("/activities", s.handleActivities)

	r.Mount("/api/v1", api)

	// Public badge and feed endpoints (no authentication).
	r.Get("/badges/{owner}/{repo}/status.svg", s.handleBadge)
	r.Get("/feed/{owner}/{repo}.xml", s.handleRSS)
	r.Get("/feed/{owner}/{repo}.atom", s.handleAtom)
	r.Post("/oauth/token", s.handleOAuthToken)
	r.Get("/oauth/userinfo", s.handleOAuthUserInfo)
	r.Get("/api/v1/explore/repos", s.handleExploreRepos)
	r.Get("/api/v1/explore/trending", s.handleTrending)
	r.Get("/api/v1/explore/topics", s.handleTopTopics)
	r.Get("/api/v1/explore/topics/{topic}", s.handleTopic)
	r.Get("/api/v1/explore/activity", s.handleExploreActivity)

	web.Mount(r, d.Log)
	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.DB.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(s.deps.Metrics.Prometheus()))
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	info := version.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"version": info.Version,
		"commit":  info.Commit,
		"date":    info.Date,
	})
}
