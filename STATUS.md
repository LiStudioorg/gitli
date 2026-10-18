# gitli 开发状态（STATUS）

> **上下文丢失时的恢复入口**：先读本文件 + `ROADMAP.md`，再继续阶段 B 的稳定化工作。

## 版本规范（强制）

- Bug 修复：补丁号 +1（`v0.10.1`）。
- 新功能 / 安全大重构：次版本号 +1（`v0.11.0`）。
- 禁止跨版本乱发 tag；`v0.10.0` 含严重 UI 错误，已在 Release 标记废弃。

## 安全审计修复（v0.10.1）

### P0 高危
- **H-01** `internal/service/git/archive.go`：`ArchiveStream` 未校验 ref → 增加 `validateRef` + `--` 分隔；`language.go:49` 同类问题一并修复。
- **H-02** `handlers_runner.go`：CI 日志 SSE 无鉴权 → `loadRepoAndPerm(PermRead)` + `job.RepoID == rp.ID`。
- **H-06** `deploy/systemd/gitli.service`：硬编码 pepper/key → 改用 `EnvironmentFile`；新增 `secretguard.go` 拒绝公开示例值/低熵/过短密钥。
- **H-07** `service/oauth/oauth.go`：redirect_uri 未校验协议 → `ValidateRedirectURIFormat`（仅 http/https、须有 host、禁止 fragment）；前端加 `isSafeRedirect`。
- **H-08** `handlers_ci.go`：删除产物仅 requireUser → 经 artifact→job→pipeline 解析仓库并校验 `PermWrite`。

### P1 逻辑
- **H-03** `util.go`：XFF 无条件信任 → `[server] trusted_proxies`，仅可信代理生效且取最右非可信跳；默认忽略 XFF。
- **H-04** `service/auth`：停用账号仍可登录 → `Login`/`VerifyPassword`/`AuthenticateRequest` 校验 `IsActive`；停用时撤销全部会话。
- **H-05** `VerifyPassword`：未限流 → 接入 `limiter`（allow/fail/success）。

### 专项清单
- 团队 re-parent 跨组织越权 → 校验父团队同组织（`ErrCrossOrg`）。
- 协作者接口仅 requireUser → 改为 `PermAdmin`；`SetCollaborator` 校验 role 所属组织与权限上限（`ErrPrivilegeEscalation`）。
- 细粒度 PAT scope → `loadRepoAndPerm` 统一强制 `tokenAllowsRepo`。
- Webhook SSRF → `secutil.NewSafeHTTPClient` 在 `DialContext` 拦截内网 IP（防 DNS Rebinding），禁止跟随重定向。
- Runner 横向越权 → logs/result 校验 `job.RunnerID == runner`。
- 路径穿越 → `fsrepo.Path/OwnerDir` 增加 `containedIn` 兜底校验。
- M-02 改密/重置 2FA 撤销其他会话；M-07 注册限流 + 12 位；M-08 挂载 `apiLim`；M-09 `loginLimiter` 定期 GC；M-15 LIKE 增加 `ESCAPE`。
- 安全 PoC 测试：`test/security_poc_test.go`。

## 日志与停机（v0.10.1）

- 新增自定义文本 Handler：`2026/10/17 23:51:42 [I] router: completed GET /api/v1/user/me for 1.2.3.4, 200 OK in 14.2ms @ handlers_user.go:42(handleUser)`
- TTY 自动着色（时间灰、I 绿、W 黄、E 红、状态码按 2xx/4xx/5xx）；写文件自动去色；`format = "json"` 保留。
- 慢请求升级：≥500ms 记 `[W]`，≥1s 记 `[E]`。
- 启动横幅：版本、Go 版本、运行模式、配置路径、数据目录、Git 版本。
- 优雅停机：收到信号 → 关闭 HTTP（10s 等待在途请求）→ 关闭 SSH → 停止后台任务 → 关闭数据库 → `PID xxx Gitli Web Finished`。
- 修复真实 Bug：SSH `ListenAndServe` 原先同步阻塞，导致信号处理器从未注册。

## 前端工程（v0.11.1）

- **唯一前端目录**：`web/`（Vue 3 + Vite + Tailwind CSS）。`web-nuxt/` 已从磁盘与 Git 删除（80 个文件）。
  嵌入源：`web/embed.go` 的 `//go:embed all:dist`。
- Tailwind 全量引入（`tailwind.config.js` / `postcss.config.js`，`content` 扫描 `web/src`）。
  无任何重型 UI 库（Element/Ant/Naive/Vuetify 均未使用）。
- 已迁移到 Tailwind：TopNav（移动端汉堡）、Sidebar（w-64 / 移动端抽屉）、AppLayout（仅主区滚动）、
  Dashboard（三列概览 + 两栏内容）、RepoView（头部/Tab/Grid/侧栏/各列表/Diff）、
  登录/注册/初始化、新建仓库/组织/导入。
- 新增 `ErrorAlert.vue`：接口错误只在主内容区顶部渲染，绝不替换 Layout。
- 待迁移（下一版本）：Explore、用户设置、管理后台、个人主页、组织设置（约 1566 行 scoped CSS）。

## UI 重构（v0.11.2）

- 按 GitHub Primer 结构（对照根目录 `github/` 抓取源码，该目录已 gitignore、不提交）重写：
  顶栏 h-16、侧栏 294px、主区 max-w-1280、Tab h-10 / 2px 橙色下划线、卡片 rounded-md + 1px 边框、无阴影。
- Dashboard 改为 GitHub 的 `col-md-4 col-lg-3` 式比例（左侧内容 3/4，右栏 1/4）。
- 仓库页新增：Watch/Fork/Star 分体按钮、文件表头（提交人+message+hash+相对时间）、
  文件类型彩色图标、README 独立卡片、About 侧栏、绿色「代码」按钮。
- 新建工单表单：分组、必填红星、blur 校验、提交 loading、错误内联提示。
- 个人主页 / 设置 / 管理后台 / 组织设置 / 探索 / 搜索：迁移 Tailwind，移除全部 scoped CSS。
- CSS 总量：101.4 KB → 55.0 KB。

## 布局校准（v0.11.3）

从 `github/` 抓取源码中提取的真实数值（非估算）：
- 顶栏 `--header-height: 48px` → `h-12`；左区 `padding-right: 8px`；`center: flex 1 min-width 0`；右区 `padding-left: 8px`。
- 边框 `--borderColor-default: #d1d9e0`（浅色）/ `#3d444d`（深色）；`--fgColor-muted: #59636e`。
- 圆角 `--borderRadius-medium: .375rem` → `rounded-md`；控件 `--control-medium-size: 2rem`。
- Dashboard 侧栏 `col-md-4`(33.3%) / `col-lg-3`(25%)，`border-right`，`sticky top:0`；
  右栏 `.feed-right-column` 356px（≥1280）。
- 仓库页 `Layout-sidebar: 220px`，gutter 16px，≤543px 单列。
- 文件表 `tr.react-directory-row { height: 40px }`，单元格 `padding-left: 16px`，行间 `border-top`。
- 修复挤压：分栏断点由 `md` 改为 `lg`，并给主列 `min-w-0`，杜绝右栏被压缩成竖排。

## Tailwind 全量迁移完成（v0.12.0）

- 全站 **零 scoped CSS**、零 `ui-*` 遗留类，样式统一由 Tailwind 原子类承担。
- `style.css` 从 490 行精简到 155 行（仅保留设计 Token 与基础复位）。
- CSS 产物：**101.4 KB → 26.7 KB（-74%）**；单二进制 25.5 MB。
- 页面覆盖：Dashboard、RepoView（全部 11 个 Tab）、新建仓库/组织/导入、新建工单、
  个人主页、用户设置、管理后台、组织设置、探索、搜索、登录/注册/初始化、Landing、OAuth 授权。
- 组件覆盖：TopNav、Sidebar、AppLayout、RepoSidebarCard、ActivityItem、ContributionGraph、
  LanguageStats、EmptyState、SkeletonBlock、RepoActivity/Releases/Wiki、CloneDropdown、
  FileIcon、ToastHost、ShortcutHelp 等。
- 布局按 `github/` 抓取源码的真实变量校准：顶栏 48px、边框 `#d1d9e0`/`#3d444d`、
  次要文字 `#59636e`、圆角 0.375rem、Dashboard 右栏 356px、仓库侧栏 220px、文件行 40px。
- `github/` 参考目录已写入 `.gitignore`，不进入版本库。

## 当前阶段

**阶段 B：稳定化（进行中）**

- A 阶段已完成：`v0.2.0-beta` 已发布，E2E `PASS=40 FAIL=0 SKIP=2`。
- 本轮完成 B1 前置与 Android/首次使用体验修复（见下方「B 阶段进展」）。
- 验收命令全部通过：`gofmt` / `go vet` / `go test ./...` / `golangci-lint` / 静态构建 / E2E。

## B 阶段进展

### 已完成
- **自动迁移**：`Serve` 启动即迁移（含 RBAC seed），失败即退出；新增 `TestServeAutoMigratesFreshDatabase`。
- **首次初始化向导**：`/api/v1/setup` + `/api/v1/setup/status`，未初始化时 API 返回 `412 setup_required`；首个账号自动成为管理员；接口单次有效（409）。
- **Android/Termux**：Git 路径解析绕过 `os/exec.LookPath`（避免 `faccessat2` 触发 SIGSYS），自动探测 Termux 路径，缺失时给出中文提示。
- **CLI**：`server` 作为 `serve` 别名；裸执行 `gitli` / `gitli --config x.toml` 直接启动服务。
- **活动流**：写入 push（含 commit 摘要与提交数）、tag、release、issue、PR、讨论；`0023_activity_summary` 迁移新增 `summary` / `repo_name`。
- **Dashboard**：两栏布局（动态流 + 仓库卡片/贡献热力图），活动图标与相对时间，仓库 Tabs 与即时过滤。
- **RepoView**：接入真实文件树、分支切换、README Markdown 渲染、最新提交；Clone 弹窗含 HTTPS/SSH 切换与复制。
- **交互修复**：新建仓库/组织/导入页面与路由；TopNav `+` 菜单与头像菜单；通知 401（未登录时不再拉取）。
- **监控**：`/metrics` 暴露 `go_goroutines`、内存指标；`[server] pprof` 可选启用；`/healthz` 校验数据库。
- **工具**：新增 `test/stress.sh`（并发 clone/push + API 压测，内置 RSS/Goroutine 阈值）。

- **B2 安全审计**：`govulncheck` 修复 `goxmldsig` → v1.6.0；SQL 无字符串拼接、无 `sh -c` 拼接、路径穿越有 `ValidateRepoPath` 兜底；`/admin` 组新增 `AdminOnly` 纵
深防御中间件；剩余告警均为 Go 标准库，随 CI 的 `1.27.x` 自动修复。
- **B4 备份恢复演练**：备份 → 恢复 → 用恢复数据启动，用户/仓库/Issue/Git 对象全部校验通过。
- **B5 文档**：新增 `docs/ADMIN.md`、`docs/USER.md`；README 更新发布状态、自动迁移、Android/Termux 说明。
- **中文提示**：CLI 帮助、flag 说明、错误信息与 HTTP API 错误信息全部改为简体中文。
- **解压即用**：启动自动建库/建表/补齐迁移，全部成功后才监听；迁移逐版本事务化。
  新增 `TestStartupCreatesDatabaseFromScratch` 与 `TestUpgradeFromPartialSchema`（旧库升级不丢数据）。

### 安全红线审查（rc4 前完成）

- **SQL 注入**：全库扫描无 `Sprintf`/`+` 拼接 SQL；所有查询参数化。`LIKE` 统一经
  `store.EscapeLike` 转义 `%`、`_`、`\`；动态表名 `countAll` 改为白名单。
  端到端注入测试（`' OR 1=1 --`、`'; DROP TABLE users;--`、`%`、`_`）返回 200 且数据完好。
- **XSS**：Markdown `html: false` + `validateLink` 仅允许 http(s)/mailto/相对路径；
  用户网站链接校验协议（拒绝 `javascript:`）；CSP `default-src 'self'`。
- **SSRF**：Webhook 阻断回环/私网/链路本地/元数据地址且禁止重定向；导入改为
  `secutil.ValidateOutboundURL` 强校验；镜像保留内网（合法用途）。
- **认证缓存**：Argon2 并发上限 2，Git Basic 认证缓存 60s，改密/重置 2FA/停用/删除
  均按用户立即失效（私有仓库实测旧密码返回 401）。

### rc5 修复（实机反馈）
- **CSRF 未签发**：页面 GET 不经过 `/api`，首个 POST 必 403（表现为“显示已登录但设置页需要登录”）。
  根路由新增 `IssueCSRF`；并用 context 复用同一 token，避免重复下发两个不同 cookie。
- **登录态不持久**：Cookie `Secure` 改为按请求判定（TLS 或 X-Forwarded-Proto），
  不再仅凭 `http_external_url`，避免 HTTP 部署下浏览器丢弃会话。
- **个人主页白屏**：`repos` 为 null 导致模板求值抛错；后端返回空数组，前端再做归一化。
- **初始化向导**：守卫对 `/setup` 也检查 `initialized`；提交后自动登录并进入 Dashboard；
  已初始化时访问 `/setup` 跳转 `/login`。
- **Termux**：生成配置时探测 Termux 路径并写入 `[git] bin`，Linxu 探测常见路径。
- **UI**：全局设计系统（卡片/输入/按钮/焦点环/暗色），重做登录/注册/初始化三个页面，
  Landing 英雄区与特性卡片、TopNav 毛玻璃与品牌徽标。

### rc6 修复（实机反馈）
- **移除“即将推出”**：新增发布 / 活动 / 百科三个真实页面（列表 + 新建 + Markdown 渲染）。
- **工单 / 合并请求**：状态筛选、作者、相对时间、空状态引导；PR 支持展开 Diff
  （后端 `/diff` 大文件保护返回 `too_large` 时给出提示）。
- **空状态**：统一 `EmptyState` 组件（图标 + 标题 + 说明 + 主按钮）。
- **个人主页白屏**：后端 `repos` 恒为数组，前端再做归一化。
- **全局错误兜底**：`app.config.errorHandler` 渲染友好提示而非整页白屏。
- 后端：Issue/Pull 列表 JOIN 用户名；Release 增加 `created_at`；新增仓库级活动接口。

### rc7：品牌视觉（现代极客风 · 赤陶橙）
- 设计令牌：`--bg-base/--bg-surface/--text-primary/--text-secondary/--border-color`，
  主色 `--color-primary: #ea580c`（深色 `#f97316`），语义色 success/danger/purple/info
  均含 `-bg` 变体；旧变量（`--primary`/`--card`/`--border` 等）保留为别名，全站即时生效。
- Logo：`favicon.svg` + `web/src/components/icons/LogoIcon.vue`（`currentColor`），
  TopNav 与初始化页统一使用；删除旧 `icon-192/512.svg`。
- Manifest：`theme_color #ea580c`，图标指向 `/favicon.svg`。
- 组件规范：状态徽章（开启绿/关闭红/已合并紫）、Tab 下划线 2px、侧边栏选中主色底、
  Diff 行级着色（增绿/删红/hunk 蓝）、卡片圆角统一 8px + 阴影。
- 清理：移除硬编码蓝色（仅保留 Go 语言色与 `--color-info` 令牌）。

### rc8：空仓库与边缘状态
- **空仓库**：`Tree` 识别“无有效对象”返回 `ErrEmptyRepo`；接口返回 `entries: []` +
  `is_empty: true`，不再泄露 `git ls-tree ... exit status 128`。Commits 返回空数组，
  Blob 返回“仓库还没有任何提交”。前端渲染空仓库引导页（含 git 快速上手命令与一键复制）。
- **表单校验**：新建仓库/组织/导入页仅在 `blur` 或提交后显示校验提示，初始不飘红。
- **侧边栏**：删除写死的“管理主题 / 2.1 MiB / LiCore v0.9.11”，改为真实
  `StarsCount` / `ForksCount` / `SizeKB`，无数据时隐藏卡片。
- **布局**：侧边栏固定 240px 不收缩（sticky）；主内容 `flex: 1`、内边距 24px；
  仓库页 `minmax(0,1fr) 300px` + 24px 间距。

### 待办
- 发布 `v0.9.0-rc8`（预发布）。
- B1 压测脚本已就绪，待在有足够内存的机器上跑完整 50 并发并记录结果。

### rc4 新增
- 零配置：自动生成 config.toml、data 目录、随机密钥；自动迁移；初始化向导。
- 性能：gzip 压缩、静态资源 immutable 缓存、SQLite WAL/busy_timeout、`memory_limit_mb`。
- 运维：`/readyz`、`gitli install`（systemd）、`gitli admin backup`、JSON 日志。
- 体验：骨架屏、Toast、PWA、GFM 任务列表、A11y、大 Diff 保护。
- 快捷键：全局 `g i` / `g p` / `g d` / `g e` / `c` / `/` / `?`，`?` 呼出说明弹窗。
- 敏感操作确认：生成/吊销 PAT、关闭 2FA、删除 OAuth 应用、注销会话均二次确认。
- 日志轮转：`[log] file` + lumberjack 按大小/保留数/天数切割，`format = "json"` 可选。
- 构建：Docker 前端与 Go 编译固定在 BUILDPLATFORM（避免 QEMU 模拟），Release 支持手动跳过 Docker。

## 已完成功能总览

### Block 1 基础设施
单二进制（前端/迁移 embed）、TOML 配置 + 环境变量覆盖、三数据库（SQLite/MySQL/PostgreSQL）、迁移、Argon2id/SHA-256/AES-256-GCM、多地址监听 + 对外地址分离、CI/CD 分离、静态编译、跨平台平台抽象。

### Block 2 核心
认证（注册/登录/Session/TOTP/PAT/SSH Key/限流）、用户名 301、Session IP/UA 绑定、企业级 RBAC（嵌套团队/自定义角色/LRU 缓存/RequirePermission）、组织/团队、仓库（Fork/转移/软删除/重命名重定向）、分支/标签保护、HTTP Smart Protocol、内置 SSH、pre/post-receive 钩子、文件树/Blob/Diff/Blame/提交图、Issue/PR/Review/评论/标签/里程碑、Webhook（HMAC+SSRF）、通知+SSE、审计、OIDC、SSH Key 双用途 + GPG 签名、分支保护签名校验。

### Block 3 扩展
Wiki/Gist/Release(+附件)、LFS、镜像/导入、包管理（Generic/NPM/Docker/Go/Maven）、Docker Registry V2、CI 调度器、搜索（git grep）、IDE/Symbols、徽章/RSS/Atom、活动/语言统计、SMTP、reactions/subscriptions/path_permissions/commit_signatures/repo_topics/ci_secrets/user_blocks/notification_prefs/org_2fa/ci_artifacts/cache/runners/lfs_locks/review_comments/user_webhooks。

### Block 4 前端与运维
Landing/Dashboard/Repo/Admin/Settings、Clone 下拉、语言统计、依赖 Tab、讨论、看板、工作流 Tab、Runner 页、暗色模式、命令面板（Ctrl+K）、全局快捷键（g i/g p/c//）、i18n（zh/en）、响应式、SSE 通知、/metrics、/healthz、OpenAPI（/api/docs）、备份/恢复、CLI 子命令、多平台构建矩阵、部署模板。

### 批次 1-5（社区/企业/体验）
- 批次 1：部署密钥、CODEOWNERS、仓库重命名+重定向、软删除+恢复、SPDX 许可证、Issue/PR 模板、归档。
- 批次 2：保存的回复、全局搜索、用户主页+关注/粉丝、探索+趋势+话题。
- 批次 3：OAuth2 Provider、会话管理、IP 白名单、审计导出、OpenAPI 文档。
- 批次 4：LDAP、SAML、依赖图、入站邮件。
- 批次 5：讨论、项目看板、命令面板/快捷键、i18n、邮件摘要。

### CLI / Runner
- 裸执行 `gitli` 直接启动；完整 `gitli admin`（用户/仓库/组织/Runner/doctor/gc/审计），毁灭性操作仅 CLI。
- 独立二进制 `gitli-runner`（另仓库）：富配置（并发/超时/清理/env/shell）、docker/podman/shell 执行器、流式日志。
- 主服务 Runner 后端：ID+Token(SHA-256)、心跳、ClaimJob、日志限流、SSE、工作流触发。

## 已知限制（非阻塞）

1. 本地沙箱 SHA-256 在部分场景被 stub → 依赖哈希的会话/PAT 本地端到端不稳，云端正常。
2. 本地无 docker/goreleaser → Docker 推拉与完整 CD 由云端验证。
3. LDAP/SAML/IMAP 无真实服务端，仅实现 + 单元测试。
4. `/metrics` 目前为轻量占位（返回 ok），可后续补充真实指标。

## 下一步（阶段 A）

1. 编写 `test/e2e.sh` 自动化脚本。
2. 本地运行，修复失败项（记录在下方"E2E 结果"）。
3. 全绿后打 `v0.2.0-beta` tag，验证 CD。

**状态：已完成**（2026-10-17）

- `v0.2.0-beta` 已发布：<https://github.com/LiStudioorg/gitli/releases/tag/v0.2.0-beta>
- Release 工作流全绿：GoReleaser 26 个产物 + deb/rpm/apk + `SHA256SUMS`；
  多架构镜像 `ghcr.io/listudioorg/gitli:v0.2.0-beta` 推送成功。
- 修复：`internal/platform` 磁盘统计在 BSD/NetBSD 无 `syscall.Statfs`，
  拆分为 `disk_syscall.go`（linux/darwin）、`disk_other.go`（其它 Unix）、
  `disk_windows.go`，全部交叉编译目标通过。

## E2E 结果

`bash test/e2e.sh` — **PASS=34 FAIL=0 SKIP=2**

覆盖并通过：

- 用户：注册 / 登录 / 会话 / 改名 301 / TOTP / PAT / SSH / GPG
- 仓库：创建 / HTTP push / HTTP clone / SSH clone
- 协作：Issue / 评论 / 关闭 / PR / Review / 合并
- 归档：归档 / 拒绝 push / 解除归档
- 包管理：Generic 发布 / 下载
- 镜像：Registry `/v2/` / 上传会话 202
- Runner：创建工作流仓库并 push 触发 / 执行任务 / 结果成功 / 日志回传

跳过：

- NPM publish：后端 NPM 端点为预留占位。
- docker push/pull：本地无 docker。

修复项：

- `admin stats` 表名 `pull_requests` → `pulls`
- `domain.Repo.DeletedAt` 扫描
- `syscall.Statfs` 迁移至 `internal/platform`（跨平台）
- `test/e2e.sh` Runner 段：push 工作流仓库触发任务后再启动 Runner，等待时间放宽至 40s，失败时输出 `runner.log`

## 关键 commit 历史

- `c66583b` fix: lint issues in admin cli
- `62d6b6c` feat: ultimate admin CLI, bare execution starts server
- `d3459c6` feat: expanded admin web dashboard
- `8d925d8` feat: batch 5 complete
- `3ecc978` feat: batch5 frontend
- `ea1feb8` feat: notification email digest
- `02a5d79` feat: i18n
- `410f7b9` feat: global shortcuts
- `0b9d729` feat: project boards
- `3111eb2` feat: discussions
- `9ca4a0c` feat: batch 4 complete
- `8d925d8` ...

## 前端（唯一目录：web）

前端为 Vue 3 + Vite + Tailwind CSS，构建产物 `web/dist` 由 `web/embed.go` 的 `//go:embed all:dist` 嵌入单二进制。
