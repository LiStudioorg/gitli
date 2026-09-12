# TODO — 未完成事项与实现备忘

> 状态：**M1–M8 全部完成**（2026-09-12），最终回归 24 项冒烟全 PASS（PR 表单字段为 base/head，
> 最终验证脚本 /tmp/opencode/final_smoke.sh 可复跑）。
> 后续方向见文末「后续方向」。每完成一项请勾选并提交（祈使句、小写、简短）。

## 全量回归清单（已验证）

- M1：注册/登录/退出/CSRF/首个用户自动 admin
- M2：仓库创建（public/org/private）、git smart HTTP 推拉（PAT/密码 Basic Auth）、匿名读 public、
  private 拒绝（401）、协作者
- M3：仓库首页（树+README markdown+最近提交）、tree/blob/raw/commits 分页/commit diff/blame/branches/tags、
  XSS 消毒（bluemonday）
- M4：内置 SSH server（公钥指纹认证、host key 持久化）、git:// 9418 匿名协议（仅 public），
  private 两协议均拒绝，owner key 可推拉
- M5：Issue（创建/评论/关闭/标签/指派）、组织（owner/member 角色、组织仓库、非成员不可见）
- M6：PR（diff、评论复用 issue_comments、merge/squash/rebase 三种合并、merge-tree 无工作区合并）
- M7：Wiki（裸仓库 plumbing 写入、markdown 渲染）、OAuth2/OIDC（未配置 404 不崩）、管理后台
- M8：Makefile build/release（5 平台静态编译）、Dockerfile 多阶段（golang:1.25-alpine 基础镜像，
  go.mod 要求 ≥1.25.11）、docker-compose、config.example.toml 校对

## 后续方向（按优先级）

- [ ] Web UI 按用户指定的组件规范重构（见下「Web UI 设计参考」），当前是功能优先的原生表单
- [ ] PAT 管理页面（pats 表已就绪，缺 UI 生成/列出/吊销）
- [ ] SSH key 管理页面（ssh_keys 表已就绪，缺 UI，目前测试直接插库）
- [ ] 协作者管理页面（collaborators 表已就绪，缺 UI）
- [ ] Issue/PR 编号全局唯一性检查（当前 number 按 repo 递增，正常）
- [ ] PR rebase 大量场景测试（冲突、非 fast-forward）
- [ ] go test 单元测试覆盖（当前零测试文件，go test ./... 输出 no test files）
- [ ] 日志级别配置实际生效检查（newLogger 读 env 不读 toml 的 log.level）
- [ ] git:// 与 SSH 的连接数/速率限制
- [ ] 镜像仓库（mirror pull）、Webhook

## 已定决策与注意事项（写代码前先读）

1. **环境坑（本机）**：构建缓存已永久迁移：GOMODCACHE=/data/home/admin1/gomod/mod（go env -w 已写入）、
   GOCACHE=/data/home/admin1/gocache、GOTMPDIR=/data/home/admin1/gotmp（Makefile 已 export）。
   网络 GOPROXY=https://goproxy.cn,direct。**/tmp 后台进程会随 shell 会话结束被杀，测试脚本须起-测-停一体。**
2. **Go 版本**：go.mod 实际为 go 1.25.11（依赖拉高），本地 toolchain 自动下载 1.25.11。
   加依赖时注意 go 指令变化；每次 go mod tidy 后检查 `grep "^go " go.mod`。
3. **sqlc**：改 queries/*.sql 或迁移后必须 generate（Makefile: make sqlc）。
   自建代码在 internal/db/open.go，**不要动 sqlc 生成的 db.go**。LIKE 参数生成 sql.NullString。
4. **迁移**：golang-migrate + database/sqlite driver。**大坑：m.Close() 会关闭底层 *sql.DB**，
   open.go 里绝不能 defer m.Close()。
5. **SQLite**：DSN `_pragma=journal_mode(WAL)&busy_timeout(5000)&foreign_keys(1)`，SetMaxOpenConns(1)。
6. **Git smart HTTP（踩坑记录）**：
   - PATH_INFO 必须以 / 开头且相对 GIT_PROJECT_ROOT（/alice/demo.git/info/refs），否则 `aliased` 错误
   - cgi.Env 必须含 os.Environ()（PATH），否则 http-backend 找不到 git 子命令
   - GIT_PROJECT_ROOT 必须绝对路径（config.validate 已转 abs）
   - CGI 输出无 HTTP 状态行，**不能 http.ReadResponse**，手动逐行读头（smarthttp.go）
   - M3 教训：chi v5 的 `*` 通配参数用 `chi.URLParam(r, "*")` 而非 r.PathValue
   - M4 教训：ssh channel 不要在 spawn 后立刻 CloseWrite()（发 EOF 导致客户端 128）；
     exec-reply 必须在 spawn 前回；host key 持久化到 dataDir
7. **权限**：auth.CanAccess 唯一入口（user/org repo、三级可见性、管理员绕过）。首个注册用户 is_admin=1。
8. **安全红线**：
   - 用户内容禁止 template.HTML，唯一豁免 internal/web/markdown.go（gomarkdown + bluemonday.UGCPolicy）
   - git 子进程全部 exec.Command 参数数组；仓库名/路径 git.ValidateRepoName/ValidateUsername 校验
   - session/PAT 只存 SHA-256 hex；密码 bcrypt
9. **模板**：internal/assets/templates/（embed）。新页面三步：render.go pages 列表 → pages/<name>.html
   （{{define "content"}}）→ handler s.render。数据走 pageData{Title,User,CSRF,Data}。
10. **PR 表单字段名**：base/head（不是 base_branch/head_branch）。
11. **提交风格**：祈使句、小写开头、简短。

## Web UI 设计参考（用户指定）

前端组件规范参考 https://spicytater.cn/terms/catalog/frontend-interaction
（subcategory: buttons-links、upload、time-picker、switch、select、radio、input-number、checkbox、button、slider）。
做 Web UI 打磨时按该 catalog 的交互规范：button 状态层级、select/radio/checkbox/switch 表单语义、
slider/input-number 数值场景、time-picker/时间统一格式、upload 交互模式（Wiki 附件等）。

## M3 实现要点（已实现，留作参考）

- internal/git/browse.go：ls-tree/for-each-ref/log/show/blame --porcelain，全部 cmd.Dir=repo
- README 渲染：gomarkdown + bluemonday.UGCPolicy()（template.HTML 唯一豁免）
- 路由：/{owner}/{repo}、tree/blob/raw/commits/commit/blame/branches/tags
- ref 校验：git rev-parse --verify <ref>^{commit}；路径穿越已校验（Clean + 前缀 + ".." 拒绝）

## M4 实现要点（已实现，留作参考）

- SSH：golang.org/x/crypto/ssh，FingerprintSHA256 查 ssh_keys 表，host key 存 <dataDir>/ssh_host_ed25519_key
- git://：自实现最小 pktline 解析（4字节hex长度），仅 upload-pack + public 仓库
- 权限统一走注入的 AuthorizeFunc（auth.CanAccess），git 包不能 import repos（循环依赖）

## Git 协议常量备忘

- smart HTTP service：git-upload-pack（读）、git-receive-pack（写）；pktline flush 帧 `0000`

## 已定决策与注意事项（写代码前先读）

1. **环境坑（本机）**：根分区曾 100% 满。构建缓存已永久迁移：
   `GOMODCACHE=/data/home/admin1/gomod/mod`（go env -w 已写入）、
   `GOCACHE=/data/home/admin1/gocache`、`GOTMPDIR=/data/home/admin1/gotmp`（Makefile 已 export）。
   网络走 `GOPROXY=https://goproxy.cn,direct`。
   **注意 /tmp 会话间不稳定，长跑服务测试用前台 `timeout N ./gitli-bin serve` 或单脚本内起停。**
2. **Go 版本**：本地 go 1.22.2，toolchain 会自动下载 1.25.11 到 GOMODCACHE。
   加依赖时注意别引入需要 go>=1.23 的版本；每次 `go mod tidy` 后检查 `grep "^go " go.mod`。
3. **sqlc**：改了 `internal/db/queries/*.sql` 或迁移后必须重新 generate：
   `GOCACHE=... GOTMPDIR=... go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate`。
   自建代码放 `internal/db/open.go`，**不要动 sqlc 生成的 db.go**。
   LIKE 参数生成 `sql.NullString`；nullable 列生成 `sql.NullInt64`。
4. **迁移**：golang-migrate + `database/sqlite` driver（modernc）。
   **大坑已修**：`m.Close()` 会关闭底层 `*sql.DB`（Migrate.Close → database.Close），
   open.go 里绝对不能 defer m.Close()。
5. **SQLite**：DSN `_pragma=journal_mode(WAL)&busy_timeout(5000)&foreign_keys(1)`，
   `SetMaxOpenConns(1)`。
6. **Git smart HTTP（M2 踩坑记录，均已修）**：
   - `git http-backend` 的 PATH_INFO **必须以 / 开头**且相对 GIT_PROJECT_ROOT（如
     `/alice/demo.git/info/refs`），否则报 `fatal: '...': aliased`。
   - cgi.Env 必须含完整 os.Environ()（PATH），否则 http-backend 找不到 git 子命令。
   - GIT_PROJECT_ROOT 必须是**绝对路径**（config.validate 已把 DataDir 转 abs）。
   - CGI 输出是裸 header + 空行 + body（无 HTTP/1.1 状态行），**不能用 http.ReadResponse**，
     要手动 textproto 式逐行读头（Status: 头需特殊处理），代码见 smarthttp.go。
   - 验证脚本模式：/tmp/opencode/smoke.sh 起-测-停一体，`bash /tmp/opencode/smoke.sh` 可复跑。
7. **权限**：`auth.CanAccess` 是唯一入口。首个注册用户 is_admin=1，管理员绕过所有可见性检查（设计行为）。
8. **安全红线（AGENTS.md 有全文，最易犯的三条）**：
   - 用户内容禁止 `template.HTML`，唯一例外 README/Markdown 且必须过 bluemonday（M3 引入）。
   - 一切 git 子进程用 `exec.Command(name, args...)` 数组形式，禁止 shell 字符串拼接；
     仓库名/路径先正则校验 `^[A-Za-z0-9_.-]+$` 且禁止 `..`（git.ValidateRepoName/ValidateUsername）。
   - session/PAT 库里只存 SHA-256 hex（auth.hashToken），密码 bcrypt。
9. **模板**：internal/assets/templates/（embed）。新增页面三步：render.go pages 列表加名 →
   pages/<name>.html 写 `{{define "content"}}` → handler 调 s.render。数据走 pageData{Title,User,CSRF,Data}。
10. **错误处理**：底层 wrapped error，handler 层 `errors.Is` 转友好 Flash 或 renderError。
11. **时间**：全部 unix 秒 UTC。
12. **提交风格**：祈使句、小写开头、简短。每完成一个小块就 commit。

## Web UI 设计参考（用户指定）

前端组件规范参考 https://spicytater.cn/terms/catalog/frontend-interaction
（subcategory: buttons-links、upload、time-picker、switch、select、radio、input-number、checkbox、button、slider 等）。
做 Web UI（M3 起的页面打磨或 v2 UI 重构）时，按该 catalog 的交互规范实现：
- button / buttons-links：层级与状态（hover/active/disabled/loading）
- select / radio / checkbox / switch：表单控件语义与可见状态
- slider / input-number：数值输入场景（如每页条数、文件大小显示配置）
- time-picker / 时间展示：提交历史、Issue 时间线用统一时间格式
- upload：M7 Wiki 附件、仓库头像上传参考其交互模式
当前 M1/M2 用的是极简原生表单，功能优先；UI 规范化放到后续里程碑统一做。

## M3 实现要点（当前任务，从这里继续）

- `internal/git/log.go` 等薄封装：
  - 文件树：`git ls-tree <ref> [-- <path>]`，解析 mode/type/sha/name
  - 提交历史：`git log --format=%H%x00%h%x00%an%x00%at%x00%s%x00%P -n 50 <ref>`
  - 单文件内容：`git show <ref>:<path>`（二进制检测：读前 8KB 查 NUL）
  - diff：`git show --format=fuller <sha>` 或 `git diff A...B`
  - blame：`git blame --porcelain <ref> -- <path>`
  - 分支：`git for-each-ref refs/heads refs/tags --format=...`
  - 裸仓库无工作区，所有命令加 `--git-dir=<path>` 或 cmd.Dir=path（现在统一 cmd.Dir=repo）
- README 渲染：引入 gomarkdown/markdown 或 goldmark + **bluemonday.UGCPolicy()** 消毒，
  这是 template.HTML 的唯一豁免点。
- 路由：GET /{owner}/{repo}（默认分支树）、/tree/{ref}/{path...}、/blob/{ref}/{path...}、
  /commits/{ref}、/commit/{sha}、/blame/{ref}/{path...}、/branches、/tags
- ref 解析 helper：分支/tag/commit 均可作 ref，先 `git rev-parse --verify <ref>^{commit}` 校验。
- 权限：每个 handler 先 Get repo → CanAccess(读) → 再动 git，404 兜底不暴露存在性。

## M4 smart HTTP 之后的要点（提前记录，防止返工）

- SSH server：golang.org/x/crypto/ssh（锁 v0.31.0，新版要求 go1.26）。
  publickey 认证 → 查 ssh_keys 表（M4 加迁移）→ `git upload-pack/receive-pack <repo>`
  PATH_INFO 风格：repo 路径为绝对路径直接传 spawn 参数（无 CGI）。
- git:// 9418：spawn `git daemon --base-path=<repos> --export-all --user-path`？不行——
  daemon 不做可见性过滤，仅 public 仓库方案：自实现 pktline 太重，可对 daemon 用
  `--interpolated-path` 或干脆只允许 public：逐仓库 `git daemon --base-path` 不支持；
  实用做法：一个 git daemon 实例 + 只 export public 仓库目录（repos/public/ 布局），
  或在 daemon 前面用 xinetd 做过滤都不优雅——**决定：M4 用 `git upload-pack --timeout`
  直连模式自实现最小 git protocol v0（pktline ref 协商），只挂 public 仓库**。

## Git 协议常量备忘

- smart HTTP service 名：`git-upload-pack`（读）、`git-receive-pack`（写）。
- pktline flush 帧是 `0000`。
- git protocol v0：客户端 GET info/refs?service=... → 服务端回
  `# service=git-upload-pack\n` pktline + flush + ref-advertisement。
