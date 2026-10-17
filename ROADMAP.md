# gitli 项目路线图（ROADMAP）

> 本文件是项目"外部记忆"。上下文被压缩或新 AI 接手时，先读本文件 + `STATUS.md` +
> `TODO.md` + `AGENTS.md` + `config.example.toml` 即可恢复认知。

## 1. 项目定位

gitli 是**轻量级、自研的 Git 服务平台**，在单台低配服务器上提供 自托管 Git 服务平台的核心能力：

- Git 托管（HTTP + SSH，clone/push/pull）
- 用户 / 组织 / 团队 / RBAC 权限
- Issue / Pull Request / Review / 评论
- Wiki / Gist / Release
- Git LFS、提交签名验证、仓库镜像与导入
- 包管理（Generic / NPM / Docker(OCI) / Go modules / Maven）
- 轻量 CI + 独立 Runner（gitli-runner）
- 搜索、徽章、RSS、活动流、语言统计

设计哲学：**单二进制、低内存（空闲 < 150 MB）、无外部强依赖（无 Redis）、按需加载、流式处理。**

- 仓库：https://github.com/LiStudioorg/gitli
- Runner 仓库：https://github.com/LiStudioorg/gitli-runner
- 许可证：AGPL-3.0；默认分支：main；Go 1.26+（依赖 modernc.org/sqlite）。

## 2. 核心约束（不可违背）

1. **单二进制**：`go build -o gitli .`；前端与迁移 embed。
2. **低内存**：空闲 < 150 MB；重任务流式/按需。
3. **多数据库**：SQLite（modernc，纯 Go）/ MySQL / PostgreSQL，业务层只用 `database/sql`。
4. `main.go` 在根目录。
5. **无 Redis**：进程内队列 + DB 持久化。
6. Git 调用系统 `git` 子进程，禁止 `sh -c`，带超时 + 环境变量白名单 + 路径校验。
7. SSH：系统 OpenSSH 或内置服务器（x/crypto/ssh，命令白名单）。
8. **无 AI、无 Pages、无安全扫描引擎。**
9. 包管理：Generic、NPM、Docker/OCI、Go modules、Maven。
10. 配置 TOML（BurntSushi/toml）。
11. 所有 `.go` 文件首行 `// SPDX-License-Identifier: AGPL-3.0`。
12. 静态编译：`CGO_ENABLED=0 -trimpath -ldflags="-s -w ..."`。
13. **主服务永不执行用户脚本**；执行由独立 Runner（gitli-runner）在隔离环境完成。
14. 监听地址与对外地址分离，统一走 `config.ResolveExternalURLs`。

## 3. 技术栈

后端：Go 1.26+、chi/v5、modernc.org/sqlite、go-sql-driver/mysql、pgx/v5、golang-migrate、BurntSushi/toml、log/slog、goldmark、chroma、argon2、pquerna/otp、golang-jwt、x/crypto/ssh。

前端：Vue3 + TS + Vite + vue-router + pinia + axios + markdown-it + highlight.js + sass。

Runner：Go 1.23+、BurntSushi/toml、yaml.v3、stdlib net/http + os/exec。

## 4. 目录结构

```
gitli/
├── main.go / go.mod / Makefile
├── ROADMAP.md / STATUS.md / TODO.md / AGENTS.md / README.md / LICENSE / config.example.toml
├── .github/workflows/{ci.yml,cd.yml} / .goreleaser.yml / .golangci.yml
├── internal/
│   ├── app/       装配、生命周期、hook、runner、events、commands
│   ├── config/    config/load/defaults/external/template
│   ├── db/        三驱动 + 迁移 + Rebind
│   ├── migrations/{sqlite,mysql,postgres}/
│   ├── transport/{http,ssh}/
│   ├── domain/    纯结构体
│   ├── service/   auth user org rbac repo git issue pull comment wiki gist release
│   │               lfs webhook notification mirror import search package pkgsvc
│   │               registry ci ide badge activity audit metrics pat runner store
│   └── infra/     gitcmd fsrepo queue mail oidc crypto password token backup secutil netutil weakpass
├── web/           前端（embed.go + dist + src）
├── deploy/{systemd,docker,launchd,windows,termux}/
├── docs/ scripts/ test/ api/
```

## 5. 数据库表（核心）

用户/认证：users、user_redirects、ssh_keys、access_tokens、token_classic_scopes、token_fine_grained_scopes、token_repositories、sessions、totp_used_codes、user_blocks、user_secrets、user_webhooks。

RBAC：permissions、roles、role_permissions、org_members、teams(parent_team_id)、team_members、team_repos、repo_collaborators、branch_protection_rules、path_permissions、permission_audit。

仓库：repos、collaborators、branch_protections、tag_protections、repo_mirrors、repo_imports、repo_topics、repo_languages。

协作：issues、pulls、comments、labels、issue_labels、milestones、reviews、review_comments、reactions、subscriptions、wiki_pages、gists、releases、release_assets。

Git 扩展：lfs_objects、lfs_locks、commit_signatures。
Webhook/通知：webhooks、webhook_deliveries、notifications、notification_prefs。
活动/运维：activities、audit_logs、jobs。
包管理：package_blobs、package_files、package_versions、docker_manifests、docker_tags、docker_uploads。
CI/Runner：ci_pipelines、ci_jobs、ci_artifacts、ci_cache、ci_secrets、ci_runners、runners、runner_jobs、runner_job_logs。

## 6. 分块计划

### Block 1：初始化与基础设施 —— 已完成
AGENTS/README/LICENSE/config.example.toml、目录、main.go、config、db 三驱动、migrate、password/token/crypto、router+中间件、三套迁移、Vue3 骨架。

### Block 2：核心 Git 平台 —— 已完成
认证（注册/登录/Session/TOTP/PAT/SSH Key/限流）、用户名 301、Session IP/UA 绑定、企业级 RBAC（嵌套团队、自定义角色、细粒度权限、LRU 缓存、RequirePermission 中间件）、组织/团队、仓库 CRUD/转移/Fork、分支/标签保护、HTTP Smart Protocol、内置 SSH、pre-receive/post-receive 钩子、文件树/Blob/Diff/Blame/提交历史/提交图、Issue/PR/Review/评论/标签/里程碑、Webhook（HMAC+SSRF）、通知+SSE、审计日志、OIDC 登录。

### Block 3：扩展功能 —— 已完成
Wiki、Gist、Release+附件上传下载、Git LFS、仓库镜像/导入、包管理（Generic/NPM/Docker/Go/Maven）、Docker Registry V2（blob 上传会话、manifest/tag）、CI 调度器、搜索（git grep）、IDE/Symbols、徽章/RSS/Atom、活动时间线、语言统计（后台队列刷新）、SMTP 邮件。

### Block 4：前端、运维、测试、文档 —— 已完成

> **当前：功能冻结，阶段 A（E2E + v0.2.0-beta）。详见 STATUS.md。**
Landing/Dashboard/Repo/Admin/Settings 页面、Clone 下拉、语言统计进度条、暗色模式、命令面板、响应式、无障碍、SSE 通知、工作流 Tab、Runner 管理页、指标 /metrics、健康检查 /healthz、管理面板 API、备份/恢复 CLI（加密+HMAC）、CLI 子命令（serve/migrate/hook/backup/restore/mirror/import/runner/admin/search-index）、多平台构建矩阵、静态编译、部署模板（systemd/docker/launchd/windows/termux）、文档（SECURITY/CI/RELEASE/ARCHITECTURE/SECURITY_AUDIT）。

### 独立 Runner（gitli-runner）—— 已完成
ID+Token 认证（Token 仅存 SHA-256）、心跳、原子 ClaimJob、日志限流、结果回传、docker/podman/shell 执行器、workflow YAML 解析、流式日志、注册/连通性校验。仓库：github.com/LiStudioorg/gitli-runner。

## 7. CI / CD

- `ci.yml`：分支/PR（`tags-ignore: ['**']`）→ gofmt/vet/test(-race)/golangci-lint/前端构建/静态编译验证/交叉编译。
- `cd.yml`：仅 `v*` tag → 前端构建 + goreleaser 原生编译（16 平台目标）+ 打包（tar.gz/zip、deb/rpm/apk、SHA256SUMS、SBOM）+ Release + Docker 多架构镜像（build-push-action → ghcr.io）。
- gitli-runner 有独立 `ci.yml`。

## 8. 已知环境限制（本地沙箱）

- 本地沙箱 SHA-256/SHA-1 在部分场景被 stub；依赖哈希的会话/PAT 本地端到端不稳，云端正常。
- 本地无 goreleaser/docker 运行完整 CD；已用配置文件校验 + 交叉编译 + 云端验证替代。
- 无 gh CLI；查 Actions 用 GitHub REST API：`curl -s https://api.github.com/repos/LiStudioorg/gitli/actions/runs`


## 9. 阶段 A：功能冻结与 E2E（进行中）

- 目标版本：`v0.2.0-beta`
- 所有功能批次（1-5）、Runner、CLI、审计**已完成并冻结**。
- 任务：编写并跑通 `test/e2e.sh` → 打 tag → 验证 CD。
- 恢复提示：本文件 + `STATUS.md` 是唯一权威进度来源。

---

## 10. v1.x 未来高级规划（仅记录，暂不实现）

以下为 v1.x 阶段的候选方向，**不属于当前 RC 范围**，待 rc 在真实环境充分验证后再逐项评估。

1. **存储分离**：将 LFS、附件、Docker 镜像层、Release 产物迁移到 S3 / MinIO 等对象存储，支持本地与对象存储双后端。
2. **Web Terminal**：基于 xterm.js + WebSocket 的网页终端，允许在浏览器内执行只读排查命令。
3. **极速克隆**：支持 Partial Clone 与 Sparse Checkout，优化超大型仓库的克隆与检出体验。
4. **依赖安全扫描**：定时扫描依赖清单，根据 CVE 数据自动发起升级 PR 并附带影响面说明。
5. **阅后即焚分享链接**：为私有代码/文件生成带有效期和访问密码的临时链接，支持随时吊销。
6. **富文本 Diff**：支持图片、PDF、Word 等非代码文件的对比与预览。
7. **深度邮件交互**：支持在邮件中直接回复指令（如 `/assign me`、`/close`）来自动操作 Issue。

> 约束：以上功能必须保持单二进制、低内存、无外部强依赖（无 Redis）的设计前提。
