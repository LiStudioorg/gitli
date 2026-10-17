# AGENTS.md — gitli AI Agent 工作手册

> ⚠️ **如果上下文丢失，请立刻读取 `ROADMAP.md` 和 `STATUS.md` 继续 E2E 测试。**

本文件面向所有参与开发 gitli 的 AI Agent 与人类工程师，是最高优先级的工作准则。
任何改动都必须遵守本文件。若与本文件冲突的需求出现，先更新本文件再写代码。

提交前必须全部通过：

```bash
gofmt -l .        # 无输出
go vet ./...      # 无错误
go test ./...     # 通过
go build -o gitli # 成功
```

---

## 1. 项目定位

gitli 是一个**轻量级、自研的 Git 服务平台**，目标是在单台低配服务器上提供 自托管 Git 服务平台的核心能力：

- Git 托管（HTTP + SSH，clone/push/pull）
- 用户 / 组织 / 团队 / 权限
- Issue / Pull Request / Review / 评论
- Wiki / Gist / Release
- Git LFS、提交签名验证、仓库镜像与导入
- 包管理（Generic / NPM / Docker(OCI) / Go modules / Maven）
- 轻量 CI（宿主机执行、限并发、Runner 轮询）
- 搜索、徽章、RSS、活动流

设计哲学：**单二进制、低内存、无外部强依赖（无 Redis）、按需加载、流式处理。**

---

## 2. 核心约束（不可违背）

1. **单二进制**：`go build -o gitli .` 只产出一个可执行文件。前端通过 `//go:embed` 嵌入，数据库迁移文件同样 embed。
2. **低内存**：主进程空闲内存 < 150 MB。重功能必须按需启动或流式处理，禁止一次性把大文件读进内存。
3. **多数据库**：SQLite（`modernc.org/sqlite`，纯 Go 无 CGO）、MySQL、PostgreSQL 三选一，业务层只依赖标准库 `database/sql`。
4. `main.go` 必须在项目根目录。
5. **不使用 Redis**：进程内队列 + SQLite 持久化。
6. Git 操作统一调用系统 `git` 子进程（`exec.CommandContext`），不引入 libgit2。Git 二进制路径优先取配置 `[git] bin`，为空时从 `PATH` 查找；**严禁 shell 字符串拼接（禁止 `sh -c`）**；必须带超时（默认 300s）；必须白名单化环境变量（`PATH`/`HOME`/`LANG=C.UTF-8`/`GIT_TERMINAL_PROMPT=0`），禁止传递 `GIT_SSH_COMMAND`、`GIT_EXTERNAL_DIFF` 等危险变量；所有仓库路径必须经 `ValidateRepoPath` 校验并限制在 data/repos 前缀内。
7. SSH 使用系统 OpenSSH + `git-shell`。
8. **无 AI、无 Pages、无安全扫描引擎**。
9. 包管理只做：Generic、NPM、Docker/OCI、Go modules、Maven。
10. **配置格式为 TOML**，解析库用 `github.com/BurntSushi/toml`。
11. **所有源文件头部加 SPDX 标识**：`// SPDX-License-Identifier: AGPL-3.0`。

---

## 3. 技术栈

### 后端
- Go 1.26+（依赖 modernc.org/sqlite 要求；当前开发环境 go1.27）
- HTTP 路由：`github.com/go-chi/chi/v5`
- 数据库驱动：`modernc.org/sqlite`、`github.com/go-sql-driver/mysql`、`github.com/jackc/pgx/v5/stdlib`
- 迁移：`github.com/golang-migrate/migrate/v4`
- 配置：`github.com/BurntSushi/toml`
- 日志：标准库 `log/slog`
- Markdown：`github.com/yuin/goldmark` + `github.com/alecthomas/chroma/v2`
- 密码：`golang.org/x/crypto/argon2`
- TOTP：`github.com/pquerna/otp`
- Session：`github.com/gorilla/sessions`
- JWT：`github.com/golang-jwt/jwt/v5`

### 前端
- Vue 3 + TypeScript + Vite
- vue-router、pinia、axios、markdown-it、highlight.js

---

## 4. 目录结构

```
gitli/
├── main.go                       # 入口，子命令分发
├── go.mod / go.sum
├── Makefile
├── AGENTS.md / README.md / LICENSE / config.example.toml
├── internal/
│   ├── app/                      # 应用装配、生命周期
│   ├── config/                   # config.go/load.go/defaults.go
│   ├── db/                       # 三驱动 + 迁移执行
│   ├── migrations/{sqlite,mysql,postgres}/
│   ├── transport/{http,ssh,sse}/ # 协议接入层
│   ├── domain/                   # 领域模型（纯结构体）
│   ├── service/                  # 业务逻辑（见下）
│   ├── infra/                    # gitcmd/fsrepo/queue/mail/oidc
│   │   ├── crypto/ password/ token/ backup/
│   └── version/                  # 版本信息
├── web/                          # 前端（embed.go + dist + src）
├── api/                          # API 契约/OpenAPI
├── scripts/                      # 构建脚本
├── deploy/{systemd,docker,caddy}/
├── docs/
└── test/
```

service 子包：auth、user、org、repo、git、issue、pull、comment、wiki、gist、release、lfs、webhook、notification、mirror、import、search、package、ci、ide、badge、activity、audit、metrics。

---

## 5. 依赖方向（严格单向）

```
main.go
  └── internal/app
        ├── internal/transport  (http, ssh, sse)
        ├── internal/service     (业务)
        └── internal/infra       (基础设施)

internal/service  →  internal/domain, internal/infra, internal/config
internal/transport →  internal/service, internal/domain
internal/infra     →  internal/domain
internal/domain    →  仅标准库（禁止 import 其它 internal 包）
```

**禁止**：
- domain 依赖 service / transport / infra
- service 直接 import transport
- 任何业务包直接 import 具体数据库驱动（只能通过 `database/sql`）
- 循环依赖

---

## 6. 数据库约定

- 三方言目录：`internal/migrations/{sqlite,mysql,postgres}/`
- 迁移文件命名：`0001_init.sql`、`0002_xxx.sql`，支持 up/down。
- 主键：
  - SQLite：`INTEGER PRIMARY KEY AUTOINCREMENT`
  - MySQL：`BIGINT AUTO_INCREMENT PRIMARY KEY`
  - PostgreSQL：`BIGSERIAL PRIMARY KEY`
- 时间：
  - SQLite / MySQL：`DATETIME`
  - PostgreSQL：`TIMESTAMPTZ`
- 布尔：
  - SQLite：`INTEGER`
  - MySQL：`TINYINT(1)`
  - PostgreSQL：`BOOLEAN`
- 字符串默认 `VARCHAR(255)`，大文本 `TEXT`。
- 所有表带 `created_at`、`updated_at`。
- 业务 SQL 必须参数化，禁止字符串拼接值。
- 查询方言差异通过 `internal/db` 中的抽象或占位符重写（`?` ↔ `$1`）统一。

---

## 7. 安全约定

- **密码**：Argon2id，memory 64 MiB，iterations 3，parallelism 2，salt 16 字节，输出 PHC 格式 `$argon2id$v=19$m=65536,t=3,p=2$...`；pepper 从 `GITLI_PEPPER` 注入；比较使用 `subtle.ConstantTimeCompare`。
- **Token**：PAT / Session / API Token 存储 SHA-256 哈希，明文只在创建时返回一次。
- **PAT**：经典 PAT（全局作用域）与细粒度 PAT（指定仓库 + read/write）。
- **TOTP**：`pquerna/otp`，secret 用 AES-256-GCM 加密存储；10 个恢复码存 SHA-256 哈希；校验防重放（`totp_used_codes`）。
- **传输**：强制 HTTPS、HSTS、TLS 1.2+、SSH 现代算法。
- **Web**：CSRF、CSP、参数化 SQL、Markdown 安全模式、登录限流（IP+用户名，5 次失败锁 15 分钟）。
- **Git**：`exec.Command` 分开传参（禁止 shell 拼接），防路径穿越，pre-receive 钩子，并发限制。
- **Webhook**：HMAC-SHA256 签名、SSRF 防护（禁止内网/元数据地址）、重试 + 死信。
- **用户名可改**：旧名写入 `user_redirects`，HTTP 301 重定向，SSH 同样支持别名。
- **审计**：所有敏感操作必须记审计；事件命名遵循 `resource.action`；写入异步、不可篡改、仅归档；仅管理员可查询/导出。事件清单见 `docs/AUDIT.md`。
- **CLI 权限最高**：`gitli admin` 拥有最高、最全权限；Web 管理后台仅是 CLI 的子集与便捷视图，禁止在 Web 暴露彻底删除用户/仓库、垃圾回收、恢复备份等毁灭性操作。
- **CLI 安全防呆**：交互式密码必须用 `golang.org/x/term.ReadPassword`；删除或降级管理员前必须校验至少保留一个活跃管理员；`--purge` 需二次确认；所有 CLI 操作写审计（`actor_type=system`，IP `127.0.0.1`）。
- **默认启动**：裸执行 `gitli` 或 `gitli --config x.toml` 直接启动服务，不得强制 `serve`。
- **SSH 密钥用途**：必须区分「认证」与「签名」用途，至少勾选一个；仅 `can_authenticate` 的密钥可用于 Git 认证，仅 `can_sign` 的密钥可用于提交签名验证。GPG 密钥默认用于签名。
- **禁止以 root 运行**：Linux/Unix 下检测到 `os.Getuid() == 0` 时默认拒绝启动并退出（退出码 1），除非显式配置 `[server] allow_root = true`（强烈不建议）。Windows 跳过该检查。
- **Runner 隔离执行**：gitli 主服务永不执行用户脚本；执行由独立 Runner 二进制（gitli-runner）在隔离环境中完成。
- **Runner 认证**：Runner 通过 ID + Token 认证；Token 只在主服务存 SHA-256 哈希，明文仅生成时显示一次。
- **Runner 数据最小化**：Runner 任务 payload 不含任何 secret；日志脱敏；主服务只做调度与日志存储。

### 跨平台约束

- **禁止硬编码路径分隔符**：一律使用 `filepath.Join`，禁止直接拼接 `/` 或 `\`。
- **禁止直接引用平台特定 syscall**：不直接引用 `syscall.SIGTERM` 等；信号由 `internal/platform` 的 `SignalSet()` 提供，Unix 监听 `SIGINT`/`SIGTERM`，Windows 仅监听 `os.Interrupt`。
- **所有平台相关代码必须用 build tag 隔离**：例如 `//go:build !windows` 与 `//go:build windows`，集中放在 `internal/platform`。
- **文件权限检查仅在 Unix 生效**；Windows 跳过权限位校验，仅校验目录存在。
- **所有可选模块必须支持通过配置完全关闭**：关闭时不初始化、不分配资源、不启动 goroutine。
- **监听地址与对外地址必须分离**：禁止把监听地址直接当作对外地址使用。所有对外 URL（克隆地址、Webhook 回调、OAuth 重定向、邮件链接、RSS、OpenGraph）必须从 `http_external_url` / `ssh_external` 解析生成，统一走 `config.ResolveExternalURLs`。

---

## 8. 编码规范

- 每个 `.go` 文件首行：`// SPDX-License-Identifier: AGPL-3.0`
- 包名小写单词；导出标识符有 doc comment。
- **不写无意义注释**；注释解释「为什么」而非「是什么」。
- 错误必须处理；使用 `fmt.Errorf("...: %w", err)` 包装。
- 外部输入一律校验；`context.Context` 作为首个参数贯穿 IO。
- 日志用 `slog`，禁止 `fmt.Println` 作为日志。
- 前端 TS 严格模式，组件用 `<script setup lang="ts">`。
- Go 代码必须通过 `go vet ./...` 与 `gofmt`。

---

## 9. Git 提交规范

Conventional Commits：

```
feat: ...
fix: ...
docs: ...
refactor: ...
test: ...
chore: ...
ci: ...
```

- 每个功能块完成后提交并推送 `origin main`。
- 不提交密钥、`.env`、`config.toml`（只提交 `config.example.toml`）。
- 提交信息用中文或英文均可，保持简洁明确。

---

## 10. 子命令列表

```
gitli serve          # 主服务（HTTP + SSH）
gitli migrate        # 数据库迁移
gitli runner         # CI Runner
gitli mirror         # 镜像同步调度
gitli import         # 导入工具
gitli search-index   # 可选索引
gitli admin          # 管理命令
gitli backup         # 备份
gitli restore        # 恢复
gitli version        # 版本
```

所有子命令支持 `--config /path/to/config.toml`，默认读取当前目录 `config.toml`。

---

## 11. 禁止事项

- ❌ 引入 CGO 依赖（SQLite 必须用 modernc.org/sqlite）。
- ❌ 引入 Redis / 消息队列等外部服务。
- ❌ 使用 libgit2 或其它 Git 原生绑定。
- ❌ 单文件读入内存处理大对象（必须流式）。
- ❌ 业务包直接依赖具体数据库驱动。
- ❌ 把敏感配置写入仓库。
- ❌ 关闭 CSRF/CSP 等安全中间件。
- ❌ 使用 `exec.Command("sh", "-c", ...)` 拼接 Git 命令。

---

## 12. 内存目标

- 主进程**空闲内存 < 150 MB**。
- Git / 打包 / CI 等重任务按需启动子进程，用并发信号量限流。
- 大文件上传下载必须流式（`io.Copy`、`io.Pipe`）。

---

## 13. 验收命令

每个块完成后必须全部通过：

```bash
gofmt -l .                 # 应无输出
go vet ./...               # 应无错误
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gitli .   # 成功，只产出一个静态二进制
./gitli version            # 打印版本
cd web && npm run build    # 前端构建成功
```

---

## 14. 静态编译原则

- 所有构建场景（`go build`、`go test`、goreleaser、CI、Makefile）必须统一使用：
  `CGO_ENABLED=0`、`-trimpath`、`-ldflags="-s -w -X ...version -X ...commit -X ...date"`。
- 纯静态二进制：不依赖 glibc/libc，可运行于 Alpine、scratch 与任意 Linux 发行版。
- **禁止引入 CGO 依赖**；SQLite 必须使用 `modernc.org/sqlite`，禁止 `mattn/go-sqlite3`；禁止 libgit2、openssl 等 C 绑定。

---

## 15. CI 优化原则

- **只在 tag 时全量构建**：分支/PR 仅做测试与 `linux/amd64` 验证；`v*` tag 才全量构建、打包、发布。
- **工作流分离**：CI 与 Release 必须独立，触发条件互斥；禁止在 CI 中发布产物，禁止在 Release 中跑测试；两者不共享 job、不互相依赖、使用不同 concurrency group。
- **缓存优先**：`~/.cache/go-build`、`~/go/pkg/mod`、`web/node_modules`、Docker layer（`type=gha`）。
- **分片并行**：多个 matrix job 分摊平台矩阵。
- **避免重复编译**：分片产出二进制，最终 job 只打包（`goreleaser release --skip=build`）。
