# gitli

**gitli** 是一个轻量级、自研的 Git 服务平台，目标是在单台低配服务器上提供 GitHub / GitLab / Gitea / Forgejo 级别的核心能力。

- **单二进制**：`go build -o gitli .` 只产出一个可执行文件，前端资源与数据库迁移均通过 `//go:embed` 内嵌。
- **低内存**：主进程空闲内存 < 150 MB，重任务按需启动子进程并限流。
- **无外部强依赖**：不使用 Redis、消息队列；进程内队列 + 数据库持久化。
- **多数据库**：SQLite / MySQL / PostgreSQL 三选一。

许可证：**AGPL-3.0**

---

## 功能特性

| 模块 | 能力 |
| --- | --- |
| Git 托管 | HTTP + SSH 协议，clone / push / pull，pre-receive 钩子 |
| 用户与认证 | 注册登录、Session、SSH Key、PAT（经典 / 细粒度）、TOTP 双因素 |
| 组织与权限 | 组织、团队、协作者、分支 / 标签保护 |
| 协作 | Issue、Pull Request、Review、评论、标签、里程碑 |
| 内容 | Wiki、Gist、Release + 附件 |
| Git 扩展 | Git LFS、提交签名验证（GPG / SSH）、仓库镜像与导入 |
| 包管理 | Generic、NPM、Docker/OCI、Go modules、Maven |
| CI | 宿主机执行、限并发、Runner 轮询、缓存、Artifacts、Secrets |
| 其他 | 搜索、徽章、RSS、活动流、审计日志、Webhook、通知 |

---

## 支持的平台与架构

gitli 为纯 Go 实现（SQLite 使用 `modernc.org/sqlite`，无 CGO），支持广泛的平台与架构。

| 平台 | 架构 | 说明 |
| --- | --- | --- |
| Linux | amd64 / arm64 / armv7 | 主推部署平台 |
| macOS | amd64 / arm64 | 开发与自托管 |
| Windows | amd64 / arm64 | 作为 Windows Service 运行 |
| Android (Termux) | arm64 / armv7 | 见 `deploy/termux/README.md` |

交叉编译示例：

```bash
GOOS=linux   GOARCH=amd64 go build -o gitli-linux-amd64 .
GOOS=darwin  GOARCH=arm64 go build -o gitli-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o gitli.exe .
```

发布版本由 CI 构建矩阵产出各平台二进制及 SHA-256 校验和，见 `.github/workflows/build.yml`。

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

记得将 `server.external_url` 设置为对外访问地址（例如 `https://git.example.com`）。

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

## 开发指南

### 环境要求

- Go 1.22+（开发使用 go1.27）
- Node.js 18+ 与 npm

### 常用命令

```bash
make build        # 构建后端（含前端内嵌）
make frontend     # 构建前端
make test         # 运行测试
make lint         # gofmt + go vet
make run          # 本地运行
```

### 验收命令

```bash
gofmt -l .                 # 应无输出
go vet ./...               # 应无错误
go build -o gitli .        # 成功，只产出一个二进制
./gitli version            # 打印版本
cd web && npm run build    # 前端构建成功
```

### 代码约定

详见 [`AGENTS.md`](./AGENTS.md)。核心要点：所有 `.go` 文件首行加 `// SPDX-License-Identifier: AGPL-3.0`；严格单向依赖；业务层只依赖 `database/sql`。

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
A：使用 `gitli admin` 子命令重置（详见 `docs/`）。

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
