# gitli

轻量级自托管 Git 托管平台。Go 编写，**单二进制 + SQLite**，
`go run` 一条命令即可跑起来，适合中小团队与个人自部署。

## 特性

- 🗄 **仓库托管**：Git smart HTTP 推拉、内置 SSH server、Git 匿名协议（9418）
- 🔒 **三级可见性**：public（匿名可读）/ org（组织成员可见）/ private（私有）
- 👥 **组织**：组织、成员角色（owner/member）、组织仓库
- 🐛 **Issue**：标签、指派、评论、开关状态
- 🔀 **Pull Request**：分支对比、diff 视图、评论、merge/squash/rebase 合并
- 📖 **Wiki**：每仓库一个 wiki 裸仓库，Web 端编辑
- 🔑 **认证**：session 登录、Personal Access Token、SSH 公钥、OAuth2/OIDC 第三方登录
- 🔍 **搜索**：仓库 / Issue / 用户模糊搜索
- ⚙️ **管理后台**：用户与仓库管理
- 🐳 **部署**：单二进制（模板/静态资源全部内嵌）或 Docker

## 快速开始

### 源码运行

依赖：Go 1.22+、git。

```sh
git clone <repo-url> gitli && cd gitli
make build          # 产出 bin/gitli
./bin/gitli serve   # 默认监听 http://localhost:3000
```

开发模式：

```sh
make dev            # go run ./cmd/gitli serve
```

### Docker

```sh
docker compose up -d          # 构建 + 启动，数据落在 ./docker-data/
```

端口映射：`3000`（HTTP + Git smart HTTP）、`2222`（SSH）、`9418`（Git 匿名协议）；
数据目录挂载在容器 `/data`。也可手动构建：

```sh
docker build -t gitli:latest .
docker run -d -p 3000:3000 -p 2222:2222 -p 9418:9418 -v gitli-data:/data gitli:latest
```

### 交叉编译发布

```sh
make release   # 产出 bin/gitli-{linux,darwin}-{amd64,arm64} 与 bin/gitli-windows-amd64.exe
```

## 配置

复制 `config.example.toml` 为 `config.toml` 按需修改，环境变量可覆盖。
默认数据落在工作目录 `./data/`（SQLite 文件 + git 裸仓库）。

主要配置项：

| 项 | 默认 | 说明 |
|---|---|---|
| `server.http_addr` | `:3000` | HTTP 监听地址 |
| `server.ssh_addr` | `:2222` | 内置 SSH server 监听地址 |
| `server.git_addr` | `:9418` | Git 匿名协议监听地址 |
| `server.root_url` | `http://localhost:3000` | 对外访问地址（clone URL 生成用） |
| `app.data_dir` | `./data` | SQLite 与仓库存储目录 |
| `oauth2.*` | — | 可选，OIDC provider 配置 |

## Git 使用

HTTP（密码处填 Personal Access Token）：

```sh
git clone http://localhost:3000/alice/demo.git
git clone ssh://git@localhost:2222/alice/demo.git
git clone git://localhost:9418/alice/demo.git   # 仅 public 仓库
```

## 开发

```sh
make build   # 编译 bin/gitli
make test    # go test ./...
make vet     # go vet ./...
make sqlc    # 修改 internal/db/queries/*.sql 后重新生成代码
make dev     # 开发运行
make clean   # 清理 bin/
```

架构与开发规范见 [AGENTS.md](AGENTS.md)。

## License

见 [LICENSE](LICENSE)。
