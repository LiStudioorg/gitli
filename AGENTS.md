# gitli — 项目方案与开发规范

gitli 是一个轻量级自托管 Git 托管平台，使用 Go 编写，
单二进制 + SQLite 部署，面向中小规模公开服务。

## 技术栈

| 层 | 选型 |
|---|---|
| 语言 | Go 1.22+ |
| HTTP 路由 | `go-chi/chi` + 中间件 |
| 数据库 | SQLite，驱动用 `modernc.org/sqlite`（纯 Go，免 CGO，交叉编译友好） |
| 数据访问 | `sqlc` 生成类型安全代码（SQL 写在 `internal/db/queries/*.sql`） |
| 迁移 | `golang-migrate`（SQL 迁移文件内嵌，启动时自动执行） |
| 模板 | `html/template` 服务端渲染，模板与静态资源用 `embed` 内嵌 |
| Git | shell 调用 git 命令（`git upload-pack / receive-pack`），不自研 Git 实现 |
| SSH | Go 内置 SSH server（`golang.org/x/crypto/ssh`），不依赖系统 sshd |
| 会话 | 数据库 session（HttpOnly cookie）+ CSRF token；Git 推拉用 Personal Access Token 或 SSH key |
| 第三方登录 | OAuth2 / 通用 OIDC（golang.org/x/oauth2） |

## 架构决策（重要，勿偏离）

1. **单二进制**：所有模板、静态资源、迁移 SQL 通过 `embed` 打进二进制；`go run ./cmd/gitli serve` 即可跑起来。
2. **Git 操作一律走 shell 调用 git 命令**，用 `internal/git` 包统一封装：
   - 读（clone/fetch/浏览）→ `git upload-pack`、`git log`、`git show` 等
   - 写（push）→ `git receive-pack`
   - 禁止引入 go-git 做核心存储操作（可用的例外：仅解析 diff 输出等辅助场景，且需先讨论）
3. **可见性三级模型**：public（匿名可读）/ org（仅组织成员可见）/ private（仅所有者与协作者可见）。
   权限判断收敛到单一函数（`auth.CanAccess` 类似物），所有路由/Handler 复用，禁止散落判断。
4. **认证**：
   - Web：session cookie + CSRF
   - Git over HTTP：Basic Auth（用户名 + PAT）
   - Git over SSH：公钥认证
   - OAuth2/OIDC 登录后落地为本地用户
5. **数据层**：所有 SQL 必须写在 queries 文件里由 sqlc 生成，Handler 里禁止裸写 SQL。
6. **模板**：按页面组织在 `web/templates/pages/`，公共部分 `web/templates/partials/`、`layout`；
   每个 handler 负责渲染自己的模板；模板函数集中在 `web/render`。
7. **配置**：TOML（`config.example.toml`），环境变量可覆盖，路径默认相对工作目录。

## 项目结构

```
gitli/
├── cmd/gitli/            # 入口。子命令: serve（默认）、admin
├── internal/
│   ├── config/           # 配置加载
│   ├── db/               # migrations/（SQL 迁移）、queries/（sqlc 输入）、sqlc 生成代码
│   ├── git/              # git 命令封装、仓库初始化、smart HTTP、SSH server
│   ├── auth/             # session、password、PAT、OAuth2/OIDC、权限判断
│   ├── users/            # 用户领域逻辑
│   ├── orgs/             # 组织领域逻辑
│   ├── repos/            # 仓库领域逻辑
│   ├── issues/           # Issue 逻辑
│   ├── pulls/            # PR 逻辑
│   ├── wiki/             # Wiki 逻辑
│   └── web/              # chi 路由、handler、middleware、render
├── web/
│   ├── templates/        # html 模板（embed）
│   └── static/           # css/js（embed）
├── config.example.toml
├── Makefile
├── Dockerfile
└── docker-compose.yml
```

## 里程碑

- **M1** 项目骨架：config + 日志 + SQLite/sqlc + 迁移 + 用户注册/登录/session + 基础布局模板
- **M2** 仓库：创建（public/org/private）、协作者、Git smart HTTP 推拉（PAT 认证）、匿名读公开仓库
- **M3** Web 代码浏览：文件树、README 渲染、提交历史、commit diff、blame、分支/tag 列表
- **M4** 内置 SSH server + Git 匿名协议（9418，仅 public 仓库）
- **M5** Issue（标签/指派/评论/关闭）+ 组织与成员角色（owner/member）+ 三级可见性权限落地
- **M6** PR：fork/分支对比、diff 视图、评论、合并（merge/squash/rebase）
- **M7** Wiki（`<repo>.wiki.git` 裸仓库 + Web 编辑）+ OAuth2/OIDC 登录 + 管理后台
- **M8** Dockerfile / docker-compose / Makefile 发布（单二进制 + 镜像）

## 开发规范

- 每完成一个里程碑：`go build ./... && go vet ./...` 通过，并实际冒烟测试对应功能。
- 错误处理：handler 层统一 404/500 页面；底层返回 wrapped error。
- 安全红线：
  - 所有用户输入渲染必须走 `html/template` 自动转义，禁止 `template.HTML` 包裹用户内容
    （唯一例外：README/Markdown 渲染须经 sanitization，如 bluemonday）
  - 仓库路径必须校验：名称只允许 `[A-Za-z0-9_.-]`，且不允许 `..`、不允许以 `.git` 结尾以外的保留名
  - shell 调用 git 一律用参数数组（`exec.Command`），禁止拼接字符串进 shell
  - session/PAT 存储一律哈希（SHA-256 足够，因 token 本身高熵）；密码用 bcrypt
- 数据库时间统一存 UTC（RFC3339 或 integer unix）。
- 提交信息风格：祈使句、小写开头、简短（如 `add repo create handler`）。
- 不要引入重型依赖（ORM、前端框架、消息队列等）；新依赖先说明理由再加。

## 常用命令

```sh
make build          # 编译单二进制 bin/gitli
make dev            # go run ./cmd/gitli serve（开发）
make sqlc           # 重新生成 sqlc 代码（需要 sqlc，可用 go run 安装）
make test           # go test ./...
```

运行时数据目录：`./data/`（SQLite 文件 + git 裸仓库），默认不入库，已在 .gitignore 中排除。
