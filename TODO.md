# TODO — 未完成事项与实现备忘

> 状态基线：M1 进行到一半（骨架 + config + db/sqlc + auth + users 已完成并编译通过）。
> 本文件记录剩余工作、已完成部分的注意事项、以及后续实现"该怎么写"的要点。
> 每完成一项请勾选并从 git 提交（提交信息：祈使句、小写、简短）。

## 里程碑进度

- [x] 技术选型定稿（见 AGENTS.md，勿偏离）
- [x] AGENTS.md / README.md / sqlc.yaml / go.mod
- [x] internal/config（TOML + 环境变量覆盖，GITLI_* 前缀）
- [x] internal/db 迁移 0001（users/sessions）+ sqlc 生成（db/models/querier/*.sql.go）
- [x] internal/auth（bcrypt、session 增删查、token SHA-256 哈希存储）
- [x] internal/users（Register/Get/Search，首个注册用户自动 is_admin=1）
- [x] **M1 完成**：web render（internal/assets 内嵌模板）、middleware（session/CSRF/secure headers）、
      chi 路由、注册/登录/退出、cmd/gitli serve 入口、冒烟测试通过
- [x] **M2 完成**：repos/pats/collaborators 表 + sqlc、internal/git（InitRepo/路径校验/SmartHTTPHandler
      桥接 git http-backend）、internal/auth/pat.go（PAT 认证）、internal/auth/access.go（CanAccess 唯一权限入口）、
      仓库创建页、Basic Auth 推拉；冒烟测试通过：匿名 clone public ✓、PAT/密码 push ✓、
      匿名读 private 401 ✓、非协作者 push 拒绝 ✓、管理员读全部 ✓（is_admin 绕过，设计行为）
- [ ] **M3 Web 代码浏览（当前任务）**
- [ ] M4 SSH server（golang.org/x/crypto/ssh，公钥查库 → spawn `git upload-pack/receive-pack`）
      + git:// 9418 匿名 daemon（仅 public，`git daemon` spawn 或自实现）
- [ ] M5 Issue + 组织（orgs 表、org_members 角色、可见性三级权限收敛到一个 `CanAccess`）
- [ ] M6 PR：fork、diff、评论、merge/squash/rebase（全部 shell git）
- [ ] M7 Wiki（`<repo>.wiki.git`）+ OAuth2/OIDC（golang.org/x/oauth2）+ 管理后台
- [ ] M8 Dockerfile（多阶段，golang 构建 → 运行时只需 git + 二进制）、docker-compose、发布

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
