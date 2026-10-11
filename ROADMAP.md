# gitli 项目路线图（ROADMAP）

> 本文件是项目的"外部记忆"。上下文被压缩或新 AI 接手时，先读本文件 +
> `STATUS.md` + `TODO.md` + `AGENTS.md` + `config.example.toml` 即可恢复全部认知。

## 1. 项目定位

gitli 是一个**轻量级、自研的 Git 服务平台**，在单台低配服务器上提供
GitHub / GitLab / Gitea / Forgejo 级别的核心能力：

- Git 托管（HTTP + SSH，clone/push/pull）
- 用户 / 组织 / 团队 / RBAC 权限
- Issue / Pull Request / Review / 评论
- Wiki / Gist / Release
- Git LFS、提交签名验证、仓库镜像与导入
- 包管理（Generic / NPM / Docker(OCI) / Go modules / Maven）
- 轻量 CI（宿主机执行、限并发、Runner 轮询）
- 搜索、徽章、RSS、活动流、语言统计

设计哲学：**单二进制、低内存（空闲 < 150 MB）、无外部强依赖（无 Redis）、按需加载、流式处理。**

- 仓库：https://github.com/LiStudioorg/gitli
- 许可证：AGPL-3.0
- 默认分支：main

## 2. 核心约束（不可违背）

1. **单二进制**：`go build -o gitli .` 只产出一个可执行文件；前端 `//go:embed` 嵌入，迁移文件 embed。
2. **低内存**：主进程空闲内存 < 150 MB；重功能按需启动或流式处理。
3. **多数据库**：SQLite（`modernc.org/sqlite`，纯 Go 无 CGO）、MySQL、PostgreSQL 三选一，业务层只用 `database/sql`。
4. `main.go` 必须在项目根目录。
5. **不使用 Redis**：进程内队列 + 数据库持久化。
6. Git 调用系统 `git` 子进程（`exec.CommandContext`），不引入 libgit2。禁止 `sh -c` 拼接。
7. SSH 使用系统 OpenSSH + git-shell，或内置 SSH 服务器（`golang.org/x/crypto/ssh`）。
8. **无 AI、无 Pages、无安全扫描引擎**。
9. 包管理只做：Generic、NPM、Docker/OCI、Go modules、Maven。
10. **配置格式 TOML**，库 `github.com/BurntSushi/toml`。
11. 所有源文件头部加 `// SPDX-License-Identifier: AGPL-3.0`。
12. **静态编译**：所有构建场景 `CGO_ENABLED=0 -trimpath -ldflags="-s -w ..."`。

## 3. 技术栈

后端：Go 1.22+（开发用 go1.27）、chi/v5、modernc.org/sqlite、go-sql-driver/mysql、pgx/v5、golang-migrate、BurntSushi/toml、log/slog、goldmark、chroma、argon2、pquerna/otp、gorilla/sessions、golang-jwt、golang.org/x/crypto/ssh。

前端：Vue3 + TypeScript + Vite + vue-router + pinia + axios + markdown-it + highlight.js + sass。

## 4. 目录结构

```
gitli/
├── main.go                       # 入口，子命令分发
├── go.mod / go.sum / Makefile
├── ROADMAP.md / STATUS.md / TODO.md / AGENTS.md / README.md / LICENSE / config.example.toml
├── .github/workflows/{ci.yml,release.yml}
├── .goreleaser.yml / .golangci.yml
├── internal/
│   ├── app/                      # 装配、生命周期、hook、ssh 适配
│   ├── config/                   # config.go/load.go/defaults.go/external.go/template.go
│   ├── db/                       # 三驱动 + 迁移执行 + Rebind
│   ├── migrations/{sqlite,mysql,postgres}/
│   ├── transport/{http,ssh,sse}/
│   ├── domain/                   # 纯结构体（仅标准库）
│   ├── service/                  # 业务逻辑
│   ├── infra/                    # gitcmd/fsrepo/queue/mail/oidc/crypto/password/token/backup/secutil/netutil/weakpass
│   └── version/
├── web/                          # 前端（embed.go + dist + src）
├── api / scripts / deploy / docs / test
```

service 子包：auth、user、org、repo、git、issue、pull、comment、wiki、gist、release、lfs、webhook、notification、mirror、import、search、package、ci、ide、badge、activity、audit、metrics、pat、store。

## 5. 依赖方向（严格单向）

```
main.go → internal/app → {transport, service, infra}
service → domain, infra, config, store
transport → service, domain
infra → domain
domain → 仅标准库
```

禁止：domain 依赖其他 internal；service import transport；业务包直接 import 具体数据库驱动；循环依赖。

## 6. 数据库约定

- 三方言目录：`internal/migrations/{sqlite,mysql,postgres}/`，文件名 `NNNN_name.sql`。
- 主键：SQLite `INTEGER PRIMARY KEY AUTOINCREMENT`；MySQL `BIGINT AUTO_INCREMENT PRIMARY KEY`；PostgreSQL `BIGSERIAL PRIMARY KEY`。
- 时间：SQLite/MySQL `DATETIME`；PostgreSQL `TIMESTAMPTZ`。
- 布尔：SQLite `INTEGER`；MySQL `TINYINT(1)`；PostgreSQL `BOOLEAN`。
- 字符串 `VARCHAR(255)`，大文本 `TEXT`。所有表带 `created_at`、`updated_at`。
- 业务 SQL 必须参数化。占位符统一用 `?`，由 `db.Rebind` 在 PG 下改写为 `$1`。
- **注意**：迁移执行器的 `splitStatements` 会剥离 `--` 整行注释，但绝不能让注释吞掉语句（历史 bug，已修）。

## 7. 安全要求

- 密码：Argon2id，memory 64 MiB，iterations 3，parallelism 2，salt 16，PHC 格式，pepper 从 `GITLI_PEPPER` 注入，`subtle.ConstantTimeCompare`。
- Token：PAT/Session 存 SHA-256 哈希，明文只返回一次。
- PAT：经典（全局 scope）+ 细粒度（指定仓库 read/write）。
- TOTP：pquerna/otp，secret 用 AES-256-GCM 加密（`enc:` 前缀），10 个恢复码存 SHA-256，防重放（`totp_used_codes`）。
- 传输：强制 HTTPS、HSTS、TLS 1.2+、SSH 现代算法。
- Web：CSRF（双提交 Cookie）、CSP、参数化 SQL、Markdown 安全模式、登录限流（IP+用户名，5 次锁 15 分钟）。
- Git：`exec.CommandContext` 分开传参，环境变量白名单，超时 300s，`ValidateRepoPath` 限制在 data/repos 前缀内。
- Webhook：HMAC-SHA256 签名、SSRF 防护、重试 + 死信。
- 用户名可改：旧名写 `user_redirects`，HTTP 301 重定向，SSH 别名。
- 禁止以 root 运行：`os.Getuid()==0` 默认拒绝启动（退出码 1），除非 `[server] allow_root=true`。
- 跨平台：禁止硬编码路径分隔符与平台 syscall，用 `internal/platform` 的 `SignalSet()`；平台代码用 build tag 隔离。

## 8. 数据库表（核心）

用户与认证：`users`、`user_redirects`、`ssh_keys`、`access_tokens`、`token_classic_scopes`、`token_fine_grained_scopes`、`token_repositories`、`sessions`、`totp_used_codes`、`user_blocks`、`user_secrets`、`user_webhooks`。

组织与权限（RBAC）：`orgs`、`teams`、`team_members`、`org_2fa_policies`、`permissions`、`roles`、`role_permissions`、`org_members`、`team_repos`、`repo_collaborators`、`branch_protection_rules`、`path_permissions`、`permission_audit`。

仓库：`repos`、`collaborators`、`branch_protections`、`tag_protections`、`repo_mirrors`、`repo_imports`、`repo_topics`、`repo_languages`。

协作：`issues`、`pulls`、`comments`、`labels`、`issue_labels`、`milestones`、`reviews`、`review_comments`、`reactions`、`subscriptions`、`wiki_pages`、`gists`、`releases`、`release_assets`。

Git 扩展：`lfs_objects`、`lfs_locks`、`commit_signatures`。

Webhook/通知：`webhooks`、`webhook_deliveries`、`notifications`、`notification_prefs`。

活动：`activities`。运维：`audit_logs`、`jobs`。

包管理：`package_blobs`、`package_files`、`package_versions`、`docker_manifests`、`docker_tags`、`docker_uploads`。

CI：`ci_pipelines`、`ci_jobs`、`ci_artifacts`、`ci_cache`、`ci_secrets`、`ci_runners`。

## 9. 子命令

```
gitli serve / migrate / runner / mirror / import / search-index / admin / backup / restore / version / hook
```
所有子命令支持 `--config`。`gitli hook <name> --owner X --repo Y` 供 git 服务端钩子调用。

## 10. 分块执行计划

### Block 1：初始化与基础设施（已完成）
AGENTS/README/LICENSE/.gitignore/config.example.toml、目录结构、main.go、config、db（三驱动）、migrate、password/token/crypto、router+中间件、三套迁移、Vue3 骨架。

### Block 2：核心 Git 平台
用户认证（注册/登录/Session/改名+301/TOTP/PAT/SSH Key/限流）、企业级 RBAC（嵌套团队、自定义角色、细粒度权限、LRU 缓存）、组织与仓库、HTTP + SSH Git 协议、pre-receive 钩子、文件树/Blob/Diff/Blame/历史/提交图、Issue/PR/Review/评论/标签/里程碑、Webhook/通知/SSE/审计日志。

### Block 3：扩展功能
Wiki/Gist/Release+附件、Git LFS、提交签名验证、仓库镜像/导入、包管理（Generic/NPM/Docker/Go/Maven）、Docker Registry V2 认证+分块上传、简化 CI、搜索（grep/FTS/zoekt）、徽章/RSS/OpenGraph/活动时间线/语言统计。

### Block 4：前端、运维、测试、文档
前端完善（Landing/Dashboard/Repo、暗色模式、响应式、无障碍、命令面板）、Web IDE、指标/健康检查、管理面板、备份/恢复 CLI、测试、CI、文档、最终验收。

## 11. CI / CD

- `ci.yml`：push main / PR（`tags-ignore: ['**']`）→ gofmt、vet、test(-race)、golangci-lint、前端构建、静态编译验证、交叉编译验证。
- `release.yml`：仅 `v*` tag → 5 分片 matrix 构建 46 个平台目标，`goreleaser release --skip=build` 打包 + 多架构镜像推 ghcr.io + 创建 Release。
- 两者触发互斥、不共享 job、不互相依赖、不同 concurrency group。

## 12. 已知环境限制（本地沙箱）

- 本地沙箱的 SHA-256 / SHA-1 被 stub 为全零 → 依赖哈希的登录/会话/PAT 在本地无法端到端验证，但逻辑正确，云端真实环境正常。
- 本地无 `goreleaser` / `docker`，无法完整跑 release；GitHub Actions 云端具备。
- 远端 CI 失败时用 GitHub REST API 查（无 `gh`）：
  `curl -s https://api.github.com/repos/LiStudioorg/gitli/actions/runs`
