// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/infra/crypto"
	"github.com/LiStudioorg/gitli/internal/infra/password"
	"github.com/LiStudioorg/gitli/internal/service/audit"
	"github.com/LiStudioorg/gitli/internal/service/user"
)

// cliAudit records a system-originated admin action.
func (a *App) cliAudit(ctx context.Context, event, detail string) {
	if a.Deps.Audit == nil {
		return
	}
	a.Deps.Audit.Log(audit.Entry{
		ActorType: "system",
		Event:     event,
		Resource:  strings.SplitN(event, ".", 2)[0],
		Action:    event,
		Result:    audit.ResultSuccess,
		IP:        "127.0.0.1",
		Detail:    detail,
	})
}

// --- user management ---

// CLIUserCreate creates a user. Password may be empty to generate one.
func (a *App) CLIUserCreate(ctx context.Context, username, email, pwd string, admin bool) (*domain.User, string, error) {
	generated := ""
	if pwd == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, "", err
		}
		pwd = hexString(key[:12])
		generated = pwd
	}
	hasher := password.NewHasher(password.DefaultParams(), a.Cfg.Security.Password.Pepper)
	hash, err := hasher.Hash(pwd)
	if err != nil {
		return nil, "", err
	}
	u, err := a.Deps.Users.Create(ctx, userCreateParams(username, email, hash, admin))
	if err != nil {
		return nil, "", err
	}
	a.cliAudit(ctx, "user.create", fmt.Sprintf("username=%s admin=%v", username, admin))
	return u, generated, nil
}

// CLIUserList returns all users.
func (a *App) CLIUserList(ctx context.Context) ([]*domain.User, error) {
	return a.Deps.Users.ListAll(ctx)
}

// CLIUserSetAdmin toggles admin rights with last-admin protection.
func (a *App) CLIUserSetAdmin(ctx context.Context, username string, admin bool) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if !admin && u.IsAdmin {
		if err := a.ensureAnotherAdmin(ctx, u.ID); err != nil {
			return err
		}
	}
	if err := a.Deps.Users.SetAdmin(ctx, u.ID, admin); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.set_admin", fmt.Sprintf("username=%s admin=%v", username, admin))
	return nil
}

// CLIUserDisable disables a user with last-admin protection.
func (a *App) CLIUserDisable(ctx context.Context, username string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if u.IsAdmin {
		if err := a.ensureAnotherAdmin(ctx, u.ID); err != nil {
			return err
		}
	}
	if err := a.Deps.Users.SetActive(ctx, u.ID, false); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.disable", "username="+username)
	return nil
}

// CLIUserEnable enables a user.
func (a *App) CLIUserEnable(ctx context.Context, username string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := a.Deps.Users.SetActive(ctx, u.ID, true); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.enable", "username="+username)
	return nil
}

// CLIUserChangePassword sets a new password.
func (a *App) CLIUserChangePassword(ctx context.Context, username, pwd string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	hasher := password.NewHasher(password.DefaultParams(), a.Cfg.Security.Password.Pepper)
	hash, err := hasher.Hash(pwd)
	if err != nil {
		return err
	}
	if err := a.Deps.Users.UpdatePassword(ctx, u.ID, hash); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.change_password", "username="+username)
	return nil
}

// CLIUserChangeEmail updates a user's email.
func (a *App) CLIUserChangeEmail(ctx context.Context, username, email string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := a.Deps.Users.SetEmail(ctx, u.ID, email); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.change_email", fmt.Sprintf("username=%s email=%s", username, email))
	return nil
}

// CLIUserResetTOTP clears 2FA.
func (a *App) CLIUserResetTOTP(ctx context.Context, username string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := a.Deps.Users.ResetTOTP(ctx, u.ID); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.reset_2fa", "username="+username)
	return nil
}

// CLIUserPurge permanently deletes a user and all owned repositories.
func (a *App) CLIUserPurge(ctx context.Context, username string) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if u.IsAdmin {
		if err := a.ensureAnotherAdmin(ctx, u.ID); err != nil {
			return err
		}
	}
	repos, err := a.Deps.Repos.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, r := range repos {
		if r.OwnerType == domain.OwnerUser && r.OwnerID == u.ID {
			owner := a.RepoOwnerSlug(ctx, r)
			path := a.Deps.FS.Path(owner, r.Name)
			_ = a.Deps.Repos.Purge(ctx, r.ID)
			_ = os.RemoveAll(path)
		}
	}
	if err := a.Deps.Users.Delete(ctx, u.ID); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.purge", "username="+username)
	return nil
}

// ensureAnotherAdmin verifies at least one other active admin remains.
func (a *App) ensureAnotherAdmin(ctx context.Context, exceptID int64) error {
	admins, err := a.Deps.Users.ListAll(ctx)
	if err != nil {
		return err
	}
	for _, u := range admins {
		if u.ID != exceptID && u.IsAdmin && u.IsActive {
			return nil
		}
	}
	return ErrLastAdmin
}

// ErrLastAdmin mirrors user.ErrLastAdmin for CLI callers.
var ErrLastAdmin = errorString("refusing to remove the last active administrator")

type errorString string

func (e errorString) Error() string { return string(e) }

// CLIUserSSHKeys lists a user's SSH keys.
func (a *App) CLIUserSSHKeys(ctx context.Context, username string) ([]*domain.SSHKey, error) {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return a.Deps.Users.ListSSHKeys(ctx, u.ID)
}

// CLIUserRevokeSSHKey deletes one SSH key.
func (a *App) CLIUserRevokeSSHKey(ctx context.Context, username string, keyID int64) error {
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := a.Deps.Users.DeleteSSHKey(ctx, u.ID, keyID); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.user.revoke_ssh_key", fmt.Sprintf("username=%s key_id=%d", username, keyID))
	return nil
}

// --- helpers ---

func userCreateParams(username, email, hash string, admin bool) user.CreateParams {
	return user.CreateParams{Username: username, Email: email, PasswordHash: hash, IsAdmin: admin}
}

func hexString(b []byte) string {
	const hexdigits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, hexdigits[c>>4], hexdigits[c&0xf])
	}
	return string(out)
}

// --- repo management ---

// CLIRepoList lists all repos (optionally scoped to an owner).
func (a *App) CLIRepoList(ctx context.Context, owner string) ([]*domain.Repo, error) {
	all, err := a.Deps.Repos.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if owner == "" {
		return all, nil
	}
	target, err := a.Deps.Users.ByUsername(ctx, owner)
	if err != nil {
		return nil, err
	}
	var out []*domain.Repo
	for _, r := range all {
		if r.OwnerType == domain.OwnerUser && r.OwnerID == target.ID {
			out = append(out, r)
		}
	}
	return out, nil
}

// CLIRepoResolve finds a repo by "owner/repo".
func (a *App) CLIRepoResolve(ctx context.Context, ref string) (*domain.Repo, error) {
	owner, name, ok := strings.Cut(ref, "/")
	if !ok || owner == "" || name == "" {
		return nil, fmt.Errorf("invalid repository reference %q (want owner/repo)", ref)
	}
	ownerUser, err := a.Deps.Users.ByUsername(ctx, owner)
	if err != nil {
		return nil, err
	}
	return a.Deps.Repos.ByOwnerName(ctx, ownerUser.ID, name)
}

// CLIRepoPurge physically removes a repository.
func (a *App) CLIRepoPurge(ctx context.Context, ref string) error {
	r, err := a.CLIRepoResolve(ctx, ref)
	if err != nil {
		return err
	}
	owner := a.RepoOwnerSlug(ctx, r)
	path := a.Deps.FS.Path(owner, r.Name)
	if err := a.Deps.Repos.Purge(ctx, r.ID); err != nil {
		return err
	}
	_ = os.RemoveAll(path)
	a.cliAudit(ctx, "admin.repo.purge", "repo="+ref)
	return nil
}

// CLIRepoArchive archives/unarchives a repository.
func (a *App) CLIRepoArchive(ctx context.Context, ref string, archived bool) error {
	r, err := a.CLIRepoResolve(ctx, ref)
	if err != nil {
		return err
	}
	if err := a.Deps.Repos.Archive(ctx, r.ID, archived); err != nil {
		return err
	}
	action := "admin.repo.archive"
	if !archived {
		action = "admin.repo.unarchive"
	}
	a.cliAudit(ctx, action, "repo="+ref)
	return nil
}

// CLIRepoTransfer moves a repo to another user.
func (a *App) CLIRepoTransfer(ctx context.Context, ref, newOwner string) error {
	r, err := a.CLIRepoResolve(ctx, ref)
	if err != nil {
		return err
	}
	target, err := a.Deps.Users.ByUsername(ctx, newOwner)
	if err != nil {
		return err
	}
	if err := a.Deps.Repos.Transfer(ctx, r.ID, target.ID, domain.OwnerUser, nil); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.repo.transfer", fmt.Sprintf("repo=%s to=%s", ref, newOwner))
	return nil
}

// CLIRepoRecalcSize recomputes a repo's size.
func (a *App) CLIRepoRecalcSize(ctx context.Context, ref string) (int64, error) {
	r, err := a.CLIRepoResolve(ctx, ref)
	if err != nil {
		return 0, err
	}
	kb, err := a.Deps.Repos.RecalcSize(ctx, r.ID)
	if err == nil {
		a.cliAudit(ctx, "admin.repo.recalc_size", fmt.Sprintf("repo=%s size_kb=%d", ref, kb))
	}
	return kb, err
}

// --- org management ---

// CLIOrgCreate creates an org owned by adminUsername.
func (a *App) CLIOrgCreate(ctx context.Context, name, adminUsername string) (*domain.Org, error) {
	u, err := a.Deps.Users.ByUsername(ctx, adminUsername)
	if err != nil {
		return nil, err
	}
	o, err := a.Deps.Orgs.Create(ctx, name, name, u.ID)
	if err != nil {
		return nil, err
	}
	a.cliAudit(ctx, "admin.org.create", fmt.Sprintf("org=%s owner=%s", name, adminUsername))
	return o, nil
}

// CLIOrgDelete removes an org.
func (a *App) CLIOrgDelete(ctx context.Context, name string) error {
	o, err := a.Deps.Orgs.ByName(ctx, name)
	if err != nil {
		return err
	}
	if _, err := a.Deps.DB.ExecContext(ctx, a.Deps.DB.Rebind(`DELETE FROM orgs WHERE id = ?`), o.ID); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.org.delete", "org="+name)
	return nil
}

// CLIOrgAddMember adds a member to an org with an optional role name.
func (a *App) CLIOrgAddMember(ctx context.Context, orgName, username, role string) error {
	o, err := a.Deps.Orgs.ByName(ctx, orgName)
	if err != nil {
		return err
	}
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	var roleID *int64
	if role != "" {
		if n, err := a.roleIDByName(ctx, role); err == nil {
			roleID = &n
		}
	}
	if err := a.Deps.RBAC.AddOrgMember(ctx, 0, o.ID, u.ID, roleID, "member"); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.org.add_member", fmt.Sprintf("org=%s user=%s role=%s", orgName, username, role))
	return nil
}

// CLIOrgRemoveMember removes an org member.
func (a *App) CLIOrgRemoveMember(ctx context.Context, orgName, username string) error {
	o, err := a.Deps.Orgs.ByName(ctx, orgName)
	if err != nil {
		return err
	}
	u, err := a.Deps.Users.ByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := a.Deps.RBAC.RemoveOrgMember(ctx, 0, o.ID, u.ID); err != nil {
		return err
	}
	a.cliAudit(ctx, "admin.org.remove_member", fmt.Sprintf("org=%s user=%s", orgName, username))
	return nil
}

// --- system maintenance ---

// CLIGenerateSecret generates a random hex secret and writes it to stdout.
func (a *App) CLIGenerateSecret(nbytes int) (string, error) {
	if nbytes <= 0 {
		nbytes = 32
	}
	key, err := crypto.GenerateKey()
	if err != nil {
		return "", err
	}
	if nbytes <= len(key) {
		return hexString(key[:nbytes]), nil
	}
	return hexString(key), nil
}

// CLIGC performs light garbage collection of stale sessions and upload temp files.
func (a *App) CLIGC(ctx context.Context) (map[string]int64, error) {
	out := map[string]int64{}
	res, err := a.Deps.DB.ExecContext(ctx, a.Deps.DB.Rebind(`DELETE FROM sessions WHERE expires_at IS NOT NULL AND expires_at < ?`), time.Now().UTC())
	if err == nil {
		n, _ := res.RowsAffected()
		out["sessions"] = n
	}
	// Remove stale docker upload staging files older than 1 day.
	uploads := filepath.Join(a.Cfg.Storage.Packages, "uploads")
	if entries, e := os.ReadDir(uploads); e == nil {
		var removed int64
		cutoff := time.Now().Add(-24 * time.Hour)
		for _, e := range entries {
			info, ierr := e.Info()
			if ierr == nil && info.ModTime().Before(cutoff) {
				_ = os.Remove(filepath.Join(uploads, e.Name()))
				removed++
			}
		}
		out["uploads"] = removed
	}
	a.cliAudit(ctx, "admin.gc", fmt.Sprintf("sessions=%d uploads=%d", out["sessions"], out["uploads"]))
	return out, nil
}

// CLIDoctor reports environment facts for `admin doctor`.
func (a *App) CLIDoctor(ctx context.Context) map[string]string {
	res := map[string]string{}
	res["database_driver"] = a.Cfg.Database.Driver
	if err := a.Deps.DB.PingContext(ctx); err != nil {
		res["database"] = "ERROR: " + err.Error()
	} else {
		res["database"] = "ok"
	}
	for name, dir := range map[string]string{
		"storage.repos": a.Cfg.Storage.Repos, "storage.lfs": a.Cfg.Storage.LFS,
		"storage.packages": a.Cfg.Storage.Packages, "storage.avatars": a.Cfg.Storage.Avatars,
		"storage.attachments": a.Cfg.Storage.Attachments,
	} {
		if info, err := os.Stat(dir); err != nil {
			res[name] = "MISSING"
		} else if !info.IsDir() {
			res[name] = "NOT A DIRECTORY"
		} else {
			res[name] = fmt.Sprintf("ok (%04o)", info.Mode().Perm())
		}
	}
	gitPath := a.Deps.GitCmd.Path()
	if out, err := exec.Command(gitPath, "--version").Output(); err == nil {
		res["git"] = strings.TrimSpace(string(out))
	} else {
		res["git"] = "ERROR: " + err.Error()
	}
	if u, err := a.Deps.Users.CountAdmins(ctx); err == nil {
		res["active_admins"] = fmt.Sprintf("%d", u)
	}
	return res
}

func (a *App) roleIDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := a.Deps.DB.QueryRowContext(ctx, a.Deps.DB.Rebind(`SELECT id FROM roles WHERE name = ? AND org_id IS NULL`), name).Scan(&id)
	return id, err
}
