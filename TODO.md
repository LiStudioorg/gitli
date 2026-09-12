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
- [ ] **M1 收尾（当前任务，从这里继续）**
  - [ ] `internal/web/render.go` 已写一半——注意：`NewRenderer` 里 pages 列表需要与
        `web/templates/pages/*.html` 实际文件对齐；`pageData` 结构里 gofmt 对齐问题顺手修
  - [ ] `web/templates/layouts/layout.html` 已写；缺 `partials/flash.html`（渲染 Flash）
  - [ ] 缺页面模板：`pages/home.html`、`pages/login.html`、`pages/register.html`、`pages/error.html`
        （每个模板第一行 `{{define "content"}} ... {{end}}`，layout 里 `{{template "content" .}}`）
  - [ ] `internal/web/middleware.go`：LoadSession（读 cookie `gitli_session` → auth.SessionUserWithCSRF
        → 存 *db.User + CSRF 进 context）、RequireLogin、CSRF 校验（POST 表单字段 `csrf_token`，
        与 session 里比对，constant-time）
  - [ ] `internal/web/handlers_auth.go`：GET/POST /register、/login、/logout；
        注册/登录失败把 Flash 传回模板（M1 直接内联渲染，不做 cookie flash）
  - [ ] `internal/web/handlers_home.go`：GET / 首页
  - [ ] `internal/web/server.go`：chi 路由组装 + `/static/*`（http.FileServer，embed static）
  - [ ] `cmd/gitli/main.go`：子命令 serve（flag 解析 -config，默认 `config.toml`，不存在也能跑）；
        mkdir data 目录；db.Open → users.Service → web.New → http.ListenAndServe；
        预留 SSH/Git server 启动位置（M4 再填）
  - [ ] `web/static/style.css`：极简即可（topbar/container/footer）
  - [ ] `config.example.toml`：与 internal/config 字段一一对应
  - [ ] `.gitignore`：`data/`、`bin/`、`config.toml`
  - [ ] `Makefile`：build / dev / test / sqlc（sqlc 用 `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0`）
  - [ ] 冒烟测试：注册→登录→退出；`go build ./... && go vet ./...`
- [ ] M2 仓库：repos 表迁移 + sqlc、创建仓库（public/org/private）、协作者表、
      smart HTTP（`git http-backend` 或自实现 GET /info/refs + POST upload-pack/receive-pack，
      PAT Basic Auth；**建议直接 spawn `git http-backend`，env 传 GIT_PROJECT_ROOT/GIT_HTTP_EXPORT_ALL**）
- [ ] M3 Web 代码浏览：文件树（`git ls-tree`）、README（bluemonday 消毒后 template.HTML，唯一例外）、
      提交历史（`git log --format`）、diff（`git show`）、blame（`git blame --porcelain`）、分支/tag
- [ ] M4 SSH server（golang.org/x/crypto/ssh，公钥查库 → spawn `git upload-pack/receive-pack`）
      + git:// 9418 匿名 daemon（仅 public，`git daemon` spawn 或自实现）
- [ ] M5 Issue + 组织（orgs 表、org_members 角色、可见性三级权限收敛到一个 `CanAccess`）
- [ ] M6 PR：fork、diff、评论、merge/squash/rebase（全部 shell git）
- [ ] M7 Wiki（`<repo>.wiki.git`）+ OAuth2/OIDC（golang.org/x/oauth2）+ 管理后台
- [ ] M8 Dockerfile（多阶段，golang 构建 → 运行时只需 git + 二进制）、docker-compose、发布

## 已定决策与注意事项（写代码前先读）

1. **环境坑（本机）**：根分区只剩 ~300MB，构建必须设置
   `export GOCACHE=/data/home/admin1/gocache GOTMPDIR=/data/home/admin1/gotmp`；
   网络走 `GOPROXY=https://goproxy.cn,direct`。建议把这两个变量写进 Makefile。
2. **Go 版本**：本地 go 1.22.2。golang.org/x/crypto 新版本会强制 toolchain 升到 1.26 并下载，
   因此已锁 x/crypto v0.31.0；`golang-migrate` v4.20.1 会把 go.mod 的 go 指令改成 1.25.11，
   已用 `go mod edit -go=1.22` 改回但依赖树仍可编译。加新依赖时注意别引入需要 go>=1.23 的版本，
   每次 `go mod tidy` 后检查 `grep "^go " go.mod`。
3. **sqlc**：配置在 `sqlc.yaml`；改了 `internal/db/queries/*.sql` 或迁移后必须 `make sqlc` 重新生成。
   生成的 `internal/db/db.go` 是 sqlc 的文件——**不要把自己的代码写进 internal/db/db.go**（我已踩过：
   自建 open 逻辑请放 `internal/db/open.go`，现在就是这个布局）。
4. **sqlc 细节**：LIKE 查询参数会生成 `sql.NullString`（见 users.sql.go 的 SearchUsers）；
   nullable integer 默认映射 `sql.NullInt64`，如需裸 int64 在 sqlc.yaml overrides 里加（上次因
   `go_nullable_type` 字段名不对已删掉，正确写法是 `go_type: ["database/sql", "NullInt64"]` 或按需定制）。
5. **迁移**：用 golang-migrate + `database/sqlite` driver（modernc）。文件命名
   `NNNN_name.up.sql` / `.down.sql`，内嵌 `//go:embed migrations/*.sql`。新增迁移时注意
   sqlc 的 schema 校验读的是 migrations 目录，两边必须一致。
6. **SQLite 连接**：`SetMaxOpenConns(1)`（WAL 下写互斥，避免 SQLITE_BUSY）；
   DSN 带 `_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)`。
7. **安全红线（AGENTS.md 有全文，最易犯的三条）**：
   - 用户内容禁止 `template.HTML`，唯一例外 README/Markdown 且必须过 bluemonday（M3 引入）。
   - 一切 git 子进程用 `exec.Command(name, args...)` 数组形式，禁止 shell 字符串拼接；
     仓库名/路径先正则校验 `^[A-Za-z0-9_.-]+$` 且禁止 `..`。
   - session/PAT 库里只存 SHA-256 hex（auth.go 里 hashToken 已实现），密码 bcrypt。
8. **权限收敛**：M5 的三级可见性判断必须写成唯一入口（如 `auth.CanAccess(user, repo)`），
   任何 handler 不得自行 `if repo.IsPrivate` 散落判断。
9. **模板约定**：`NewRenderer` 的 pages 列表是手工登记的，新增页面三步：
   pages 列表加名字 → `web/templates/pages/<name>.html` 写 `{{define "content"}}` → handler 调 Render。
   模板内数据统一从 `pageData{Title, User, CSRF, Data}` 走，Data 是页面私有结构。
10. **错误处理**：底层 wrapped error（`fmt.Errorf("...: %w", err)`），handler 层统一
    `RenderError(w, 404/500, msg)`；`users` 包的哨兵错误（ErrUsernameTaken 等）在 handler
    里 `errors.Is` 判断后转成友好提示。
11. **时间**：所有 unix 秒 UTC（`time.Now().UTC().Unix()`），模板里用 `timefmt` 函数。
12. **提交风格**：祈使句、小写开头、简短（`add repo create handler`）。每完成一个小块就 commit。

## M2 smart HTTP 实现要点（提前记录，防止返工）

- 路由：`/{owner}/{repo}.git/info/refs?service=git-upload-pack|git-receive-pack`
  和 `/{owner}/{repo}.git/git-upload-pack|git-receive-pack`（POST）。
- 鉴权：GET info/refs 无 service 参数（ dumb 探测）直接 403；push 必须 Basic Auth 用户名+PAT，
  fetch 对 public 仓库匿名放行，org/private 走 CanAccess。
- Content-Type 必须 `application/x-git-<service>-advertisement` / `-result`；
  首包 pktline `# service=git-upload-pack\n` + `0000` flush。
- 建议实现：直接 `exec.Command("git", "http-backend")`，CGI env（GIT_PROJECT_ROOT、
  PATH_INFO、QUERY_STRING、REQUEST_METHOD、CONTENT_TYPE、HTTP_CONTENT_ENCODING、REMOTE_USER）
  并用 `cmd.StdinPipe/StdoutPipe` 桥接 HTTP body——比手写协议包解析更稳。
- push 权限校验点：receive-pack 前置（不依赖 post-receive hook）。

## Git 协议常量备忘

- smart HTTP service 名：`git-upload-pack`（读）、`git-receive-pack`（写）。
- pktline flush 帧是 `0000`。
- git daemon（M4）如 spawn 系统进程记得 `--export-all=false` + 按 public 仓库目录组织，
  或干脆自实现 9418（pktline 与 SSH 版共用 internal/git 代码）。
