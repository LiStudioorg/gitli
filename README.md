# gitli

**gitli** 是一个轻量级、自研的 Git 服务平台，目标是在单台低配服务器上提供 GitHub / GitLab / Gitea / Forgejo 级别的核心能力。

- **单二进制**：`go build -o gitli .` 只产出一个可执行文件，前端资源与数据库迁移均通过 `//go:embed` 内嵌。
- **低内存**：主进程空闲内存 < 150 MB，重任务按需启动子进程并限流。
- **无外部强依赖**：不使用 Redis、消息队列；进程内队列 + 数据库持久化。
- **多数据库**：SQLite / MySQL / PostgreSQL 三选一。

许可证：**AGPL-3.0**

---

## 发布状态

**最新发布：[`v0.1.2`](https://github.com/LiStudioorg/gitli/releases)**（早期 Alpha）

> ⚠️ v0.1.x 为**早期 Alpha 版本**：后端核心功能已就绪，前端页面仍在完善中，请勿用于生产环境。

- 二进制：Linux（amd64/arm64/armv7/386）、macOS（amd64/arm64）、Windows（amd64/arm64/386）、FreeBSD（amd64/arm64/386/arm）、OpenBSD（amd64/arm64）、NetBSD（amd64）的 `tar.gz`/`zip`。
- 系统包：Linux `deb` / `rpm` / `apk`（amd64/arm64/armv7/386）。
- 校验：每个 Release 附带 `SHA256SUMS`。
- 容器镜像：`ghcr.io/listudioorg/gitli`（多架构 `linux/amd64`、`linux/arm64`、`linux/arm/v7`）。

```bash
docker pull ghcr.io/listudioorg/gitli:latest
docker run --rm ghcr.io/listudioorg/gitli:latest version
```

---

## CI/CD 状态

仓库包含两个**触发条件互斥**的 GitHub Actions 工作流（在仓库 Actions 页面按名称查看运行状态）：

| 工作流 | 文件 | 触发 | 职责 |
| --- | --- | --- | --- |
| **CI** | [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) | 分支 push（`main`）/ PR / 手动，显式排除 tag | gofmt、go vet、go test(-race)、前端构建、静态编译与交叉编译验证、golangci-lint |
| **Release** | [`.github/workflows/cd.yml`](./.github/workflows/cd.yml) | 仅 `v*` tag / 手动 | GoReleaser 原生编译 16 个平台目标 + 打包（tar.gz/zip、deb/rpm/apk、SHA256SUMS）+ 推送多架构镜像 `ghcr.io/LiStudioorg/gitli` + 创建 GitHub Release |

> 在 GitHub 页面点击 **Actions** 标签页可查看两者的实时状态徽章；本文档不内联易失效的徽章 URL。
> 工作流细节见 [`docs/CI.md`](./docs/CI.md)，发布步骤见 [`docs/RELEASE.md`](./docs/RELEASE.md)。

---

## 功能特性

下表对应**实际已实现**的服务模块（`internal/service/`）与 HTTP 路由（`internal/transport/http/server.go`）。

| 模块 | 能力 |
| --- | --- |
| 用户与认证 | 注册登录、Session、SSH Key、PAT（经典 / 细粒度）、TOTP 双因素、密码修改、用户名改名（旧名 301 重定向） |
| 组织与 RBAC | 组织、成员、**多级嵌套团队**（可 reparent）、自定义角色、协作者、团队仓库授权 |
| 仓库托管 | 仓库创建 / 删除、Fork、Transfer、分支 / 标签保护、归档下载、语言统计 |
| Git 协议 | HTTP Smart 协议（`info/refs`、`git-upload-pack`、`git-receive-pack`）、**内置 SSH 服务器**、pre-receive 钩子 |
| 协作 | Issue、Pull Request、Review、评论、标签、里程碑、状态流转、合并 |
| 代码浏览 | 分支列表、目录树、文件 Blob、提交历史、**提交图（graph）**、Diff、Blame、代码搜索 |
| Web IDE | 目录树、文件读取、声明 / 符号（symbols）搜索 |
| 内容 | Wiki、Gist、Release + 附件（含发布 / 详情 / 资源列表） |
| Git 扩展 | Git LFS（batch / 上传 / 下载）、仓库镜像（mirror）、外部仓库导入（import） |
| 包管理 | Generic、NPM、Docker/OCI、Go modules、Maven；**内置 Docker Registry V2**（`/v2/` blobs / manifests / tags） |
| CI | 宿主机执行的轻量调度器、限并发、Runner 轮询、Pipeline 与 Job 列表 |
| 通知与事件 | 站内通知、**SSE 事件流**（`/api/v1/events`）、活动流、审计日志 |
| Webhook | HMAC-SHA256 签名、SSRF 防护、重试与死信 |
| 其他 | 代码搜索、徽章（badge SVG）、RSS / Atom、提交签名验证（GPG / SSH）、Prometheus 指标、管理端统计与审计 |

---

## 命令

### 默认启动

**直接运行 `gitli` 即可启动主服务**（等价于 `serve`），无需子命令：

```bash
./gitli                          # 使用当前目录 config.toml 启动
./gitli --config /etc/gitli/config.toml   # 指定配置并启动
```

### 子命令

`gitli` 为单二进制，管理命令通过 `gitli <command> [--config path/to/config.toml]` 调用：

| 子命令 | 说明 |
| --- | --- |
| `serve` | 启动 HTTP + SSH 主服务 |
| `migrate` | 按 `database.driver` 执行数据库迁移 |
| `hook` | 执行服务端 git 钩子（`gitli hook <name> --owner <owner> --repo <repo>`） |
| `backup` | 流式备份（`--out <file> [--source <dir>] [--encrypt]`，加密密钥取自 `GITLI_BACKUP_KEY`） |
| `restore` | 从备份归档恢复（`--in <file> [--dest <dir>]`） |
| `version` | 打印版本、commit 与构建时间 |

> **CLI 是最高管理权限，Web 后台仅为便捷视图（其子集）。**
> 毁灭性操作（彻底删除用户/仓库、垃圾回收、恢复备份）只能通过 CLI 执行。

## 管理命令（`gitli admin`）

```bash
# 用户管理
gitli admin user create <username> <email> [--password <pwd>] [--admin]
gitli admin user list [--format table|json]
gitli admin user info <username>
gitli admin user set-admin <username> [--revoke]
gitli admin user disable <username>
gitli admin user enable <username>
gitli admin user change-password <username>      # 交互式输入，不回显
gitli admin user change-email <username> <new_email>
gitli admin user reset-2fa <username>
gitli admin user ssh-keys <username>
gitli admin user revoke-ssh-key <username> <key_id>
gitli admin user delete <username> --purge       # 二次确认，物理删除及其名下仓库

# 仓库管理
gitli admin repo list [--owner <username>]
gitli admin repo info <owner>/<repo>
gitli admin repo delete <owner>/<repo> --purge   # 二次确认，物理删除
gitli admin repo transfer <owner>/<repo> <new_owner>
gitli admin repo archive <owner>/<repo>
gitli admin repo unarchive <owner>/<repo>
gitli admin repo recalc-size <owner>/<repo>

# 组织管理
gitli admin org list
gitli admin org create <org_name> <admin_username>
gitli admin org delete <org_name>
gitli admin org add-member <org_name> <username> [--role <role_name>]
gitli admin org remove-member <org_name> <username>

# Runner 管理
gitli admin runner list
gitli admin runner create <name> --labels "linux,amd64"
gitli admin runner delete <runner_id>
gitli admin runner reset-token <runner_id>

# 系统维护
gitli admin doctor            # 数据库/目录权限/git 版本/管理员数量自检
gitli admin generate-secret   # 生成随机密钥
gitli admin gc                # 清理过期 session 与上传临时文件
gitli admin reindex           # 搜索索引（当前按需 grep）
gitli admin audit list        # 终端查看审计日志
gitli admin backup            # 提示使用 `gitli backup`
```

> 安全：交互式密码使用 `term.ReadPassword`；删除/降级管理员前会校验至少保留一个活跃管理员；毁灭性命令需二次输入名称确认；所有 CLI 操作写入 `audit_logs`（`actor_type=system`，IP `127.0.0.1`）。
| `help` | 显示帮助 |

> **保留子命令（尚未实现）**：`runner`、`mirror`、`import`、`search-index`、`admin`。当前调用会返回 `subcommand "..." is not implemented yet`；对应能力已通过 `serve` 进程或 HTTP API 提供（如镜像 / 导入 / 搜索 / 管理端）。

---

## 支持的平台与架构

gitli 为纯 Go 实现（SQLite 使用 `modernc.org/sqlite`，无 CGO），支持主流平台与架构。

| 平台 | 支持的架构 |
| --- | --- |
| Linux | amd64, arm64, armv6, armv7, 386, riscv64, loong64, ppc64le, s390x |
| macOS (darwin) | amd64, arm64 |
| Windows | amd64, arm64, 386 |
| FreeBSD | amd64, arm64, 386, arm |
| OpenBSD | amd64, arm64 |
| NetBSD | amd64 |
| Android (Termux) | arm64（见 `deploy/termux/README.md`） |

> 由于 `modernc.org/sqlite` 为纯 Go 移植实现，**不支持** `js/wasm`、`wasip1/wasm`、`plan9`、`aix`、`illumos`、`solaris`、`dragonfly`、`linux/mips*`、`linux/ppc64` 等平台（这些平台无法编译该依赖）。发布流水线按上述支持矩阵分片构建。

交叉编译示例：

```bash
GOOS=linux   GOARCH=amd64 go build -o gitli-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o gitli-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o gitli.exe .
```

发布版本由 `.github/workflows/cd.yml` 的构建矩阵产出各平台二进制及 SHA-256 校验和，详见 [`docs/CI.md`](./docs/CI.md)。

---

## SSH 服务

SSH Git 传输可按配置开启或关闭：

```toml
[ssh]
enabled = true             # false 时完全关闭，零开销
mode = "builtin"           # builtin | system
address = ":2222"
host_key_path = "./data/ssh_host_ed25519_key"
handshake_timeout_seconds = 10
idle_timeout_seconds = 300
max_connections = 32
```

- **enabled = false**：完全跳过 SSH 模块初始化——不监听端口、不加载 Host Key、不创建 goroutine，日志记录 `SSH server disabled by config`。此时只能通过 HTTP(S) 访问 Git。
- **mode = "builtin"**：使用 gitli 内置 SSH 服务器，自动生成/加载 Host Key，公钥认证对接数据库中的 SSH Key。
- **mode = "system"**：由系统 `sshd` + `git-shell` 提供服务，gitli 仅同步用户公钥到 `authorized_keys`。

---

## Runner（独立执行器）

gitli 采用**主服务与执行器分离**的架构。**主服务永不执行用户脚本**，用户提交的 Pipeline / Job 脚本由**独立二进制 Runner** 在隔离环境中执行。

- **独立二进制**：[`github.com/LiStudioorg/gitli-runner`](https://github.com/LiStudioorg/gitli-runner) 是独立仓库、独立发布的程序，不随主服务打包。
- **职责分离**：主服务只负责调度、下发任务 payload 与存储日志；Runner 负责实际拉起隔离环境运行脚本。
- **隔离执行**：Runner 可运行在另一台机器、另一个容器甚至另一片网络中，与主服务进程隔离。

### 添加一个 Runner

1. 以管理员身份登录，进入 Web 管理面板 → **Runner** 标签页。
2. 点击 **新建 Runner**。
3. 复制生成的 **runner_id** 与 **token**（token **仅显示这一次**，关闭后无法再次查看）。

### 配置 Runner 侧

Runner 侧通过其自身的 TOML 配置连接主服务并选择执行方式，完整示例见 [`runner.example.toml`](./runner.example.toml)：

```toml
[runner]
server = "https://git.example.com"   # 主服务对外地址
id     = "r_xxxxxxxxxxxxxxxx"        # 管理面板生成的 runner_id
token  = "ghr_yyyyyyyyyyyyyyyy"      # 管理面板生成的 token（仅显示一次）
labels = ["linux", "docker"]         # 标签，用于主服务按标签调度

[executor]
type = "docker"                      # docker | podman | shell
                                     # shell 默认禁用，需显式开启（隔离性最弱）
```

### HTTP 协议

Runner 与主服务之间通过 HTTP + Bearer token 通信，请求需携带两个头：

```
Authorization: Bearer <token>
X-Runner-ID: <runner_id>
```

| 端点 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/runner/heartbeat` | POST | Runner 周期心跳，上报在线状态与标签 |
| `/api/v1/runner/register` | POST | 注册 Runner 及其能力（标签 / executor 类型） |
| `/api/v1/runner/jobs/next` | POST | 长轮询领取下一个待执行任务 |
| `/api/v1/runner/jobs/{id}/logs` | POST | 流式上报任务日志（分块追加） |
| `/api/v1/runner/jobs/{id}/result` | POST | 上报任务最终结果（成功 / 失败 / 退出码） |

### 安全说明

- **Token 只存哈希**：主服务仅以 **SHA-256 哈希**形式存储 Runner token，明文在生成时仅返回一次。
- **认证凭据**：Runner 通过 **ID + Token** 认证，二者缺一不可；token 泄露可立即在管理面板吊销。
- **Job payload 不含密钥**：下发给 Runner 的任务 payload **不包含任何 secret**；主服务只做调度与日志落库，日志在入库前做脱敏处理。

---

## 监听与对外地址

gitli 将**实际监听地址**（进程绑定的网卡/端口）与**对外展示地址**（用户和第三方看到的地址）严格分离。在反向代理、端口映射、NAT 环境下二者必然不同，混用会导致克隆 URL、Webhook 回调、OAuth 重定向指向错误地址。

配置项位于 `[server]`：

- `http_listen`：HTTP 实际监听地址数组（支持 IPv4/IPv6，可不同端口）。
- `http_external_url`：对外展示的 Web 地址，用于克隆 URL、Webhook 回调、OAuth 重定向等。
- `ssh_enabled`：是否启用 SSH。
- `ssh_listen`：SSH 实际监听地址数组。
- `ssh_external`：SSH 对外地址，格式 `host` 或 `host:port`。

### 场景示例

**1. 无反向代理，直接监听**

```toml
[server]
http_listen = ["0.0.0.0:8080"]
http_external_url = ""            # 自动推断为 http://localhost:8080
ssh_listen = ["0.0.0.0:2222"]
ssh_external = ""                 # 自动推断为 localhost:2222
```

**2. Nginx 反代 + HTTPS**

```toml
[server]
http_listen = ["127.0.0.1:8080"]  # 只监听本机回环
http_external_url = "https://git.example.com"
ssh_listen = ["0.0.0.0:2222"]
ssh_external = "git.example.com:2222"
```

**3. 端口映射（内网 8080 → 公网 80）**

```toml
[server]
http_listen = ["0.0.0.0:8080"]
http_external_url = "http://example.com"   # 用户访问的是 80 端口
```

**4. NAT 环境（内网 IP，公网域名）**

```toml
[server]
http_listen = ["10.0.0.5:8080"]
http_external_url = "https://git.example.com"
ssh_listen = ["10.0.0.5:2222"]
ssh_external = "git.example.com:2222"
```

**5. IPv4/IPv6 双栈，不同端口**

```toml
[server]
http_listen = ["0.0.0.0:8080", "[::]:8081"]
http_external_url = "https://git.example.com"
ssh_listen = ["0.0.0.0:2222", "[::]:2223"]
```

**6. SSH 独立域名和端口**

```toml
[server]
http_external_url = "https://git.example.com"
ssh_external = "ssh.example.com:2222"
```

环境变量同样支持，数组用逗号分隔：

```bash
export GITLI_SERVER_HTTP_LISTEN="0.0.0.0:8080,[::]:8080"
export GITLI_SERVER_HTTP_EXTERNAL_URL="https://git.example.com"
export GITLI_SERVER_SSH_ENABLED=true
export GITLI_SERVER_SSH_LISTEN="0.0.0.0:2222,[::]:2222"
export GITLI_SERVER_SSH_EXTERNAL="git.example.com:2222"
```

> 旧版 `http` / `ssh` / `external_url` 单地址配置仍可被识别，会自动转换为新的监听数组并打印告警，请尽快迁移。

---

## 快速开始

### 1. 构建

```bash
# 后端（前端资源已内嵌时可直接构建）
go build -o gitli .

# 前端资源（首次或前端改动后）
cd web && npm install && npm run build && cd ..
go build -o gitli .
```

### 2. 初始化配置

```bash
cp config.example.toml config.toml
# 按需编辑 config.toml
```

### 3. 迁移数据库并启动

```bash
./gitli migrate
./gitli serve
```

默认 HTTP 监听 `:8080`，SSH 监听 `:2222`。

> **安全基线：禁止以 root 运行。** gitli 在 Linux/Unix 下检测到 `os.Getuid() == 0` 时默认拒绝启动并退出（退出码 1）。请创建专用用户运行；确需强制时在 `[server]` 设置 `allow_root = true`（强烈不建议）。

```bash
# 创建 gitli 用户（无登录权限）
sudo useradd -r -s /bin/false -d /var/lib/gitli gitli
sudo mkdir -p /var/lib/gitli
sudo chown -R gitli:gitli /var/lib/gitli

# 以 gitli 用户运行
sudo -u gitli ./gitli serve
```

### 5. 验证

```bash
./gitli version
curl http://127.0.0.1:8080/healthz
```

---

## TOML 配置说明

配置文件默认为当前目录的 `config.toml`，可通过 `--config /path/to/config.toml` 指定。
完整示例见 [`config.example.toml`](./config.example.toml)。

主要段落：

- `[server]`：HTTP / SSH 监听地址与外部 URL。
- `[database]`：驱动（`sqlite` / `mysql` / `postgres`）、DSN、连接池。
- `[storage]`：仓库、LFS、包、头像、附件存储路径。
- `[security]`：密码哈希参数、加密密钥、登录限流。
- `[modules]`：功能模块开关与包管理子模块。
- `[limits]`：并发与包体大小限制。
- `[log]`：日志级别与格式。
- `[mail]`：SMTP 配置。
- `[git]`：系统 git 二进制与调用约束。

### [git]

| 键 | 说明 | 默认 |
| --- | --- | --- |
| `bin` | git 可执行文件路径，留空则从 `PATH` 查找 | 空 |
| `timeout_seconds` | 单次 git 调用超时（秒） | `300` |
| `check_version` | 版本低于 `min_version` 时是否拒绝启动 | `true` |
| `min_version` | 最低 git 版本 | `2.30` |

生产环境**推荐显式指定** `bin = "/usr/bin/git"`，避免受 `PATH` 影响导致加载到非预期的二进制。所有 git 调用均带超时、白名单环境变量，且禁止 shell 拼接。

### 配置优先级

**环境变量 > 配置文件 > 默认值。**

环境变量前缀为 `GITLI_`，嵌套字段用下划线连接，例如：

| 环境变量 | 对应字段 |
| --- | --- |
| `GITLI_DATABASE_DSN` | `database.dsn` |
| `GITLI_DATABASE_DRIVER` | `database.driver` |
| `GITLI_SECURITY_PASSWORD_PEPPER` | `security.password.pepper` |
| `GITLI_SECURITY_CRYPTO_KEY` | `security.crypto.key_hex` |
| `GITLI_MAIL_PASSWORD` | `mail.password` |

> **敏感字段（pepper、crypto key、mail password）只从环境变量读取，不写入配置文件。**

---

## 数据库选择

### SQLite（默认，零依赖）

```toml
[database]
driver = "sqlite"
dsn = "./data/gitli.db"
```

使用 `modernc.org/sqlite`，纯 Go 实现，无 CGO，无需额外安装。

### MySQL

```toml
[database]
driver = "mysql"
dsn = "user:pass@tcp(127.0.0.1:3306)/gitli?charset=utf8mb4&parseTime=true&loc=Local"
```

### PostgreSQL

```toml
[database]
driver = "postgres"
dsn = "postgres://user:pass@127.0.0.1:5432/gitli?sslmode=disable"
```

迁移文件按方言分别存放于 `internal/migrations/{sqlite,mysql,postgres}/`，`gitli migrate` 会根据 `database.driver` 自动选择。

---

## 反向代理

生产环境建议在 gitli 前放置反向代理并启用 HTTPS。以下以 Caddy 为例：

```caddy
git.example.com {
    encode gzip
    reverse_proxy 127.0.0.1:8080
}
```

Nginx 示例：

```nginx
server {
    listen 443 ssl http2;
    server_name git.example.com;

    ssl_certificate     /etc/ssl/git.example.com.crt;
    ssl_certificate_key /etc/ssl/git.example.com.key;

    client_max_body_size 512m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

记得将 `server.http_external_url`（及 `server.ssh_external`）设置为对外访问地址（例如 `https://git.example.com`）。

---

## systemd 部署

将可执行文件安装到 `/usr/local/bin/gitli`，配置放在 `/etc/gitli/config.toml`：

```ini
[Unit]
Description=gitli Git service
After=network.target

[Service]
Type=simple
User=gitli
Group=gitli
WorkingDirectory=/var/lib/gitli
ExecStart=/usr/local/bin/gitli serve --config /etc/gitli/config.toml
Environment=GITLI_PEPPER=change-me
Environment=GITLI_CRYPTO_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gitli
sudo systemctl status gitli
```

也可参考仓库内 [`deploy/systemd/`](./deploy/systemd) 提供的单元文件模板。

---

## 审计日志

gitli 内置企业级审计系统，记录所有敏感操作，用于安全合规与事后追溯。

- **标准化事件**：事件命名遵循 `resource.action`（如 `repo.delete`、`auth.login.failure`、`git.push`、`ssh_key.create`）。
- **完整上下文**：记录操作者、类型（user/admin/system/runner）、IP、User-Agent、目标、结果（success/failure/denied）、请求 ID、旧值/新值（JSON）。
- **异步写入**：通过进程内队列异步落库，不阻塞业务主流程。
- **不可篡改**：应用层禁止修改/删除审计记录，超期仅归档到 `audit_logs_archive`。
- **查询与导出**：管理面板支持按事件、用户、结果、时间范围筛选；支持流式导出 CSV / JSON。
- **统计**：按天、按事件类型、按用户聚合。
- **保留策略**：`[audit] retention_days`（默认 365）天后每日自动归档；`archive_retention_days`（默认 1095）控制归档保留。
- **告警规则**：当某事件在时间窗口内超过阈值时触发（邮件 / Webhook 渠道）。

管理 API（需管理员）：`GET /api/v1/admin/audit`、`/audit/{id}`、`/audit/stats`、`/audit/export`、`POST /audit/archive`、`/audit/alert-rules`。

用户可在「设置 → 安全活动」查看自己的操作记录与登录设备，并可远程注销会话。

详见 [`docs/AUDIT.md`](./docs/AUDIT.md)。

---

## 开发指南

### 环境要求

- Go 1.26+（依赖 modernc.org/sqlite 要求；CI 使用 go 1.27.x）
- Node.js 18+ 与 npm

### 常用命令

```bash
make build        # 构建后端（纯静态、去符号，含前端内嵌）
make frontend     # 构建前端
make test         # 运行测试
make lint         # gofmt + go vet
make run          # 本地运行
```

### 静态编译

gitli 始终以**纯静态**方式编译，零运行时依赖（除系统 `git` 外）：

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gitli .
```

- `CGO_ENABLED=0`：不依赖 glibc / libc，可运行于 Alpine、scratch 及任意 Linux 发行版。
- `-trimpath`：去掉本地构建路径。
- `-ldflags="-s -w"`：去掉符号表与 DWARF 调试信息，体积可减小 30–40%。

验证：

```bash
file gitli     # 应显示 "statically linked"
ldd gitli      # 应显示 "not a dynamic executable"
```

SQLite 使用纯 Go 的 `modernc.org/sqlite`，全仓库不含任何 CGO 依赖。

### 验收命令

```bash
gofmt -l .                 # 应无输出
go vet ./...               # 应无错误
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gitli .   # 成功，只产出一个二进制
./gitli version            # 打印版本
cd web && npm run build    # 前端构建成功
```

### 代码约定

详见 [`AGENTS.md`](./AGENTS.md)。核心要点：所有 `.go` 文件首行加 `// SPDX-License-Identifier: AGPL-3.0`；严格单向依赖；业务层只依赖 `database/sql`。

### 更多文档

- [架构（ARCHITECTURE）](./docs/ARCHITECTURE.md)
- [配置参考](./docs/configuration.md)
- [安全模型（SECURITY）](./docs/SECURITY.md)
- [CI/CD 流水线](./docs/CI.md)
- [发布流程（RELEASE）](./docs/RELEASE.md)

---

## CI 与发布

仓库使用两个**完全独立、触发条件互斥**的工作流：

| 工作流 | 触发 | 用途 |
| --- | --- | --- |
| `ci.yml`（CI） | 分支 push（`main`）/ PR，显式排除 tag | gofmt、go vet、go test、前端构建、静态编译验证、交叉编译验证、golangci-lint |
| `cd.yml`（Release） | 仅 `v*` tag | 前端构建 + GoReleaser 全量编译 16 个平台目标 + 打包（tar.gz/zip、deb/rpm/apk、SHA256SUMS）+ 多架构 Docker 镜像推送 `ghcr.io/LiStudioorg/gitli` + 创建 GitHub Release |

**提交代码或开 PR** 时 CI 自动跑测试与验证（约 2 分钟内）；**打 tag（如 `v1.0.0`）** 时 Release 自动全量构建并发布。

两个工作流使用不同的 `concurrency` group，不共享 job、不互相依赖。两者均可通过 `workflow_dispatch` 手动触发。

### 构建时间优化

为把单次 Release 控制在 8 分钟以内：

- **分层触发**：只在 tag 时全量构建；分支/PR 仅构建 `linux/amd64`。
- **缓存优先**：`~/.cache/go-build`、`~/go/pkg/mod`、`web/node_modules`、Docker layer（`type=gha`）。
- **分片并行**：22 个目标拆到 3 个 matrix 分片，2 核 runner 并行编译。
- **避免重复编译**：分片产出二进制上传 artifact，最终 job 用 `goreleaser release --skip=build` 只做打包与发布。

---

## 常见问题

**Q：为什么不用 Redis？**
A：降低部署复杂度与内存占用，使用进程内队列 + 数据库持久化即可满足单机场景。

**Q：SQLite 需要安装 CGO 工具链吗？**
A：不需要。使用 `modernc.org/sqlite`，纯 Go 实现。

**Q：如何切换数据库？**
A：修改 `config.toml` 的 `[database]` 段，重新执行 `gitli migrate` 即可（数据不自动迁移）。

**Q：支持 HTTPS 吗？**
A：gitli 建议部署在反向代理（Caddy / Nginx）之后，由代理负责 TLS 终止；应用层强制 HSTS 与安全头。

**Q：忘记了管理员密码怎么办？**
A：`admin` 子命令为保留项（尚未实现）。当前可通过直接更新数据库中的密码哈希，或删除用户后重新注册首个管理员账号来恢复访问。

---

## 许可证

本项目采用 [AGPL-3.0](./LICENSE) 许可证。

---

## 贡献指南

1. Fork 仓库并创建特性分支：`git checkout -b feat/my-feature`。
2. 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 提交规范。
3. 提交前确保通过 `gofmt -l .`、`go vet ./...`、`go build -o gitli .` 与前端构建。
4. 发起 Pull Request，描述改动动机、方案与验证方式。

> 提交代码即表示同意以 AGPL-3.0 许可证发布。
