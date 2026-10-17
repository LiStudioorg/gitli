// SPDX-License-Identifier: AGPL-3.0
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/LiStudioorg/gitli/internal/app"
	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/domain"
	"github.com/LiStudioorg/gitli/internal/service/audit"
)

func runAdmin(args []string) error {
	if len(args) == 0 {
		adminUsage()
		return nil
	}
	group := args[0]
	rest := args[1:]
	if group == "-h" || group == "--help" || group == "help" {
		adminUsage()
		return nil
	}
	fs := flag.NewFlagSet("admin "+group, flag.ContinueOnError)
	configPath := fs.String("config", "config.toml", "配置文件路径")
	// group-specific flags are parsed by subhandlers; parse global flags here.
	if err := fs.Parse(rest); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a, err := app.New(context.Background(), cfg, *configPath)
	if err != nil {
		return err
	}
	defer func() { _ = a.Close() }()
	ctx := context.Background()
	sub := fs.Args()

	switch group {
	case "user":
		return adminUser(ctx, a, sub)
	case "repo":
		return adminRepo(ctx, a, sub)
	case "org":
		return adminOrg(ctx, a, sub)
	case "runner":
		return adminRunner(ctx, a, sub)
	case "generate-secret":
		secret, err := a.CLIGenerateSecret(32)
		if err != nil {
			return err
		}
		fmt.Println(secret)
		return nil
	case "doctor":
		rep := a.CLIDoctor(ctx)
		for k, v := range rep {
			fmt.Printf("%-22s %s\n", k+":", v)
		}
		return nil
	case "gc":
		res, err := a.CLIGC(ctx)
		if err != nil {
			return err
		}
		for k, v := range res {
			fmt.Printf("%s cleaned: %d\n", k, v)
		}
		return nil
	case "reindex":
		fmt.Println("reindex: gitli uses on-demand git grep; no index to rebuild")
		return nil
	case "audit":
		return adminAudit(ctx, a, sub)
	case "backup":
		fmt.Println("use `gitli backup --out <file>` to create a backup")
		return nil
	default:
		adminUsage()
		return fmt.Errorf("未知的管理分组 %q", group)
	}
}

func adminUser(ctx context.Context, a *app.App, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin user：缺少子命令")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "create":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin user create <username> <email> [--password <pwd>] [--admin]")
		}
		fs := flag.NewFlagSet("user create", flag.ContinueOnError)
		pwd := fs.String("password", "", "密码（留空则自动生成）")
		admin := fs.Bool("admin", false, "授予管理员权限")
		if err := fs.Parse(rest[2:]); err != nil {
			return err
		}
		u, pw, err := a.CLIUserCreate(ctx, rest[0], rest[1], *pwd, *admin)
		if err != nil {
			return err
		}
		fmt.Printf("created user %s (id=%d)", u.Username, u.ID)
		if *pwd == "" {
			fmt.Printf(" password=%s", pw)
		}
		fmt.Println()
		return nil
	case "list":
		users, err := a.CLIUserList(ctx)
		if err != nil {
			return err
		}
		format := "table"
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--format" && i+1 < len(rest) {
				format = rest[i+1]
			}
		}
		if format == "json" {
			return json.NewEncoder(os.Stdout).Encode(users)
		}
		for _, u := range users {
			fmt.Printf("% -6d %-24s %-32s admin=%-5v active=%v\n", u.ID, u.Username, u.Email, u.IsAdmin, u.IsActive)
		}
		return nil
	case "info":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user info <username>")
		}
		u, err := a.Deps.Users.ByUsername(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("id:       %d\nusername: %s\nemail:    %s\nadmin:    %v\nactive:   %v\ntotp:     %v\ncreated:  %s\n",
			u.ID, u.Username, u.Email, u.IsAdmin, u.IsActive, u.TOTPEnabled, u.CreatedAt.Format("2006-01-02 15:04:05"))
		return nil
	case "set-admin":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user set-admin <username> [--revoke]")
		}
		revoke := false
		for _, r := range rest[1:] {
			if r == "--revoke" {
				revoke = true
			}
		}
		return a.CLIUserSetAdmin(ctx, rest[0], !revoke)
	case "disable":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user disable <username>")
		}
		return a.CLIUserDisable(ctx, rest[0])
	case "enable":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user enable <username>")
		}
		return a.CLIUserEnable(ctx, rest[0])
	case "change-password":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user change-password <username>")
		}
		pw, err := promptPassword("New password: ")
		if err != nil {
			return err
		}
		if err := a.CLIUserChangePassword(ctx, rest[0], pw); err != nil {
			return err
		}
		fmt.Println("password updated")
		return nil
	case "change-email":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin user change-email <username> <new_email>")
		}
		return a.CLIUserChangeEmail(ctx, rest[0], rest[1])
	case "reset-2fa":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user reset-2fa <username>")
		}
		if err := a.CLIUserResetTOTP(ctx, rest[0]); err != nil {
			return err
		}
		fmt.Println("2FA reset")
		return nil
	case "ssh-keys":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user ssh-keys <username>")
		}
		keys, err := a.CLIUserSSHKeys(ctx, rest[0])
		if err != nil {
			return err
		}
		for _, k := range keys {
			fmt.Printf("%-4d %-24s %s\n", k.ID, k.Title, k.Fingerprint)
		}
		return nil
	case "revoke-ssh-key":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin user revoke-ssh-key <username> <key_id>")
		}
		var id int64
		if _, err := fmt.Sscanf(rest[1], "%d", &id); err != nil {
			return fmt.Errorf("无效的密钥 ID")
		}
		return a.CLIUserRevokeSSHKey(ctx, rest[0], id)
	case "delete":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin user delete <username> --purge")
		}
		purge := false
		for _, r := range rest[1:] {
			if r == "--purge" {
				purge = true
			}
		}
		if !purge {
			return fmt.Errorf("已拒绝：彻底删除用户需显式传入 --purge")
		}
		if !confirm("Type the username to confirm permanent deletion: ", rest[0]) {
			return fmt.Errorf("已取消")
		}
		return a.CLIUserPurge(ctx, rest[0])
	default:
		return fmt.Errorf("未知的 user 子命令 %q", sub)
	}
}

func adminRepo(ctx context.Context, a *app.App, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin repo：缺少子命令")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		owner := ""
		for i := 0; i < len(rest); i++ {
			if rest[i] == "--owner" && i+1 < len(rest) {
				owner = rest[i+1]
			}
		}
		repos, err := a.CLIRepoList(ctx, owner)
		if err != nil {
			return err
		}
		for _, r := range repos {
			fmt.Printf("%-32s size=%dKB archived=%v deleted=%v\n", r.Name, r.SizeKB, r.Archived, r.DeletedAt != nil)
		}
		return nil
	case "info":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin repo info <owner>/<repo>")
		}
		r, err := a.CLIRepoResolve(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("id:      %d\nname:    %s\ndefault: %s\nprivate: %v\narchived:%v\nsize:    %d KB\n",
			r.ID, r.Name, r.DefaultBranch, r.IsPrivate, r.Archived, r.SizeKB)
		return nil
	case "delete":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin repo delete <owner>/<repo> --purge")
		}
		purge := false
		for _, r := range rest[1:] {
			if r == "--purge" {
				purge = true
			}
		}
		if !purge {
			return fmt.Errorf("已拒绝：彻底删除仓库需显式传入 --purge")
		}
		if !confirm("Type the repository (owner/repo) to confirm deletion: ", rest[0]) {
			return fmt.Errorf("已取消")
		}
		return a.CLIRepoPurge(ctx, rest[0])
	case "transfer":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin repo transfer <owner>/<repo> <new_owner>")
		}
		return a.CLIRepoTransfer(ctx, rest[0], rest[1])
	case "archive":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin repo archive <owner>/<repo>")
		}
		return a.CLIRepoArchive(ctx, rest[0], true)
	case "unarchive":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin repo unarchive <owner>/<repo>")
		}
		return a.CLIRepoArchive(ctx, rest[0], false)
	case "recalc-size":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin repo recalc-size <owner>/<repo>")
		}
		kb, err := a.CLIRepoRecalcSize(ctx, rest[0])
		if err != nil {
			return err
		}
		fmt.Printf("size: %d KB\n", kb)
		return nil
	default:
		return fmt.Errorf("未知的 repo 子命令 %q", sub)
	}
}

func adminOrg(ctx context.Context, a *app.App, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin org：缺少子命令")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		orgs, err := a.Deps.Orgs.ListAll(ctx)
		if err != nil {
			return err
		}
		for _, o := range orgs {
			fmt.Printf("%-24s owner_id=%d\n", o.Name, o.OwnerID)
		}
		return nil
	case "create":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin org create <org_name> <admin_username>")
		}
		o, err := a.CLIOrgCreate(ctx, rest[0], rest[1])
		if err != nil {
			return err
		}
		fmt.Printf("created org %s (id=%d)\n", o.Name, o.ID)
		return nil
	case "delete":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin org delete <org_name>")
		}
		if !confirm("Type the org name to confirm deletion: ", rest[0]) {
			return fmt.Errorf("已取消")
		}
		return a.CLIOrgDelete(ctx, rest[0])
	case "add-member":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin org add-member <org_name> <username> [--role <role_name>]")
		}
		role := ""
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--role" && i+1 < len(rest) {
				role = rest[i+1]
			}
		}
		return a.CLIOrgAddMember(ctx, rest[0], rest[1], role)
	case "remove-member":
		if len(rest) < 2 {
			return fmt.Errorf("用法：gitli admin org remove-member <org_name> <username>")
		}
		return a.CLIOrgRemoveMember(ctx, rest[0], rest[1])
	default:
		return fmt.Errorf("未知的 org 子命令 %q", sub)
	}
}

func adminRunner(ctx context.Context, a *app.App, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("admin runner：缺少子命令")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		runners, err := a.Deps.Runners.List(ctx)
		if err != nil {
			return err
		}
		for _, r := range runners {
			fmt.Printf("%-16s %-24s status=%-8s labels=%s\n", r.RunnerID, r.Name, r.Status, r.Labels)
		}
		return nil
	case "create":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin runner create <name> [--labels \"linux,amd64\"]")
		}
		labels := "[]"
		for i := 1; i < len(rest); i++ {
			if rest[i] == "--labels" && i+1 < len(rest) {
				parts := strings.Split(rest[i+1], ",")
				b, _ := json.Marshal(parts)
				labels = string(b)
			}
		}
		res, err := a.Deps.Runners.Create(ctx, rest[0], labels)
		if err != nil {
			return err
		}
		fmt.Printf("runner_id: %s\ntoken:     %s\n", res.Runner.RunnerID, res.Token)
		fmt.Println("(store the token now; it is shown only once)")
		return nil
	case "delete":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin runner delete <runner_id>")
		}
		r, err := a.Deps.Runners.ByRunnerID(ctx, rest[0])
		if err != nil {
			return err
		}
		return a.Deps.Runners.Delete(ctx, r.ID)
	case "reset-token":
		if len(rest) < 1 {
			return fmt.Errorf("用法：gitli admin runner reset-token <runner_id>")
		}
		r, err := a.Deps.Runners.ByRunnerID(ctx, rest[0])
		if err != nil {
			return err
		}
		token, err := a.Deps.Runners.ResetToken(ctx, r.ID)
		if err != nil {
			return err
		}
		fmt.Println(token)
		return nil
	default:
		return fmt.Errorf("未知的 runner 子命令 %q", sub)
	}
}

func adminAudit(ctx context.Context, a *app.App, args []string) error {
	if len(args) > 0 && args[0] != "list" {
		return fmt.Errorf("用法：gitli admin audit list")
	}
	logs, err := a.Deps.Audit.Query(ctx, auditQuery())
	if err != nil {
		return err
	}
	for _, l := range logs {
		fmt.Printf("%s %-28s actor=%-6s ip=%-15s %s\n", l.CreatedAt.Format("2006-01-02 15:04:05"),
			l.Event, actorLabel(l.ActorID), l.ActorIP, l.Result)
	}
	return nil
}

func auditQuery() audit.Query {
	q := audit.Query{Limit: 50}
	return q
}

func actorLabel(id *int64) string {
	if id == nil {
		return "system"
	}
	return fmt.Sprintf("%d", *id)
}

func promptPassword(label string) (string, error) {
	fmt.Fprint(os.Stderr, label)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func confirm(prompt, expected string) bool {
	fmt.Fprint(os.Stderr, prompt)
	var got string
	if _, err := fmt.Fscanln(os.Stdin, &got); err != nil {
		return false
	}
	return got == expected
}

func adminUsage() {
	fmt.Fprint(os.Stderr, `gitli admin — 最高权限管理命令（Web 后台仅为其子集）

用法：
  gitli admin <分组> <子命令> [参数] [--config 配置文件]

分组：
  user     create|list|info|set-admin|disable|enable|change-password|change-email|reset-2fa|ssh-keys|revoke-ssh-key|delete --purge
  repo     list|info|delete --purge|transfer|archive|unarchive|recalc-size
  org      list|create|delete|add-member|remove-member
  runner   list|create|delete|reset-token
  generate-secret   生成随机密钥
  doctor            环境自检
  gc                清理过期数据
  reindex           重建搜索索引（当前为按需 grep）
  audit list        查看审计日志

示例：
  gitli admin user create alice alice@example.com --admin
  gitli admin repo list
  gitli admin doctor

提示：
  毁灭性操作（delete --purge、gc）仅 CLI 提供，且需要二次确认。
`)
}

var _ = domain.PermAdmin
