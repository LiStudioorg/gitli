# gitli 管理员手册

本手册面向部署与运维人员，覆盖部署、配置、CLI 命令、备份恢复与故障排查。

> CLI（`gitli admin`）拥有最高权限；Web 管理后台只是它的子集与便捷视图。
> 彻底删除用户/仓库、垃圾回收、恢复备份等毁灭性操作**只提供 CLI**。

---

## 1. 部署

### 1.1 二进制部署

```bash
# 从 Release 下载对应平台的压缩包并解压
tar xzf gitli_0.2.0-beta_linux_amd64.tar.gz
sudo install -m 0755 gitli /usr/local/bin/gitli

# 创建专用用户（禁止以 root 运行）
sudo useradd --system --home /var/lib/gitli --shell /usr/sbin/nologin gitli
sudo mkdir -p /var/lib/gitli && sudo chown gitli:gitli /var/lib/gitli

sudo -u gitli cp /etc/gitli/config.example.toml /etc/gitli/config.toml
sudo -u gitli /usr/local/bin/gitli serve --config /etc/gitli/config.toml
```

首次启动会自动执行数据库迁移并进入初始化向导（`/setup`），
创建第一个管理员后接口自动锁定。

### 1.2 必须设置的环境变量

| 变量 | 说明 |
| --- | --- |
| `GITLI_PEPPER` | 密码 pepper，变更会导致所有密码失效 |
| `GITLI_CRYPTO_KEY` | 32 字节 hex，用于 TOTP/CI Secret 的 AES-256-GCM 加密 |
| `GITLI_BACKUP_KEY` | 备份加密密钥（`--encrypt` 时必需） |

```bash
gitli admin generate-secret    # 生成随机密钥
```

### 1.3 Docker

```bash
docker run -d --name gitli \
  -p 8080:8080 -p 2222:2222 \
  -v /srv/gitli:/data \
  -e GITLI_PEPPER=<random> \
  -e GITLI_CRYPTO_KEY=<64-hex> \
  ghcr.io/listudioorg/gitli:v0.2.0-beta
```

### 1.4 关键配置项

```toml
[server]
http_listen = ["0.0.0.0:8080"]
http_external_url = "https://git.example.com"   # 对外地址，务必正确
ssh_listen = ["0.0.0.0:2222"]
ssh_external = "git.example.com:2222"
pprof = false                                    # 排障时临时开启

[database]
driver = "sqlite"                                # sqlite | mysql | postgres
dsn = "/var/lib/gitli/gitli.db"

[git]
bin = ""                                         # 空则从 PATH 查找
check_version = true
min_version = "2.30"
timeout_seconds = 300
```

> **监听地址与对外地址必须分离**：克隆地址、Webhook 回调、OAuth 重定向、邮件链接
> 全部由 `http_external_url` / `ssh_external` 生成。

> **Android / Termux**：必须显式设置 `[git] bin = "/data/data/com.termux/files/usr/bin/git"`，
> 因为 Android 的 seccomp 禁止 `faccessat2`，gitli 不会调用 `os.PathLookPath`。

---

## 2. 子命令

```bash
gitli serve            # 启动服务（别名：server；裸执行 gitli 等同）
gitli migrate          # 手动强制同步数据库结构
gitli runner           # 内置 CI Runner（独立 runner 二进制更推荐）
gitli mirror           # 镜像同步调度
gitli import           # 仓库导入工具
gitli search-index     # 无操作（搜索按需 git grep）
gitli admin            # 管理命令（最高权限）
gitli backup           # 创建备份
gitli restore          # 恢复备份
gitli hook             # 供 git 钩子内部调用
gitli version          # 版本信息
```

所有子命令支持 `--config /path/to/config.toml`。

---

## 3. `gitli admin` 命令大全

```bash
gitli admin <group> <subcommand> [args] [--config path]
```

### 3.1 用户

```bash
gitli admin user create <username> <email> [--password <pwd>] [--admin]
gitli admin user list [--format json]
gitli admin user info <username>
gitli admin user set-admin <username> [--revoke]
gitli admin user disable <username>
gitli admin user enable <username>
gitli admin user change-password <username>
gitli admin user change-email <username> <email>
gitli admin user reset-2fa <username>
gitli admin user ssh-keys <username>
gitli admin user revoke-ssh-key <username> <key-id>
gitli admin user delete <username> --purge
```

- 交互式密码使用 `term.ReadPassword`，不会回显。
- 降级/删除管理员前会校验至少保留一个活跃管理员。
- `--purge` 需要二次确认。

### 3.2 仓库

```bash
gitli admin repo list [--format json]
gitli admin repo info <owner>/<repo>
gitli admin repo delete <owner>/<repo> --purge
gitli admin repo transfer <owner>/<repo> <new-owner>
gitli admin repo archive <owner>/<repo>
gitli admin repo unarchive <owner>/<repo>
gitli admin repo recalc-size
```

### 3.3 组织

```bash
gitli admin org list
gitli admin org create <name> <admin_username>
gitli admin org delete <name>
gitli admin org add-member <org> <username> [--role <role>]
gitli admin org remove-member <org> <username>
```

### 3.4 Runner

```bash
gitli admin runner list
gitli admin runner create <name> [--labels linux,amd64]
gitli admin runner delete <id>
gitli admin runner reset-token <id>
```

Token 明文仅在创建/重置时显示一次，服务端只保存 SHA-256 哈希。

### 3.5 运维

```bash
gitli admin doctor          # 环境自检（Git、数据库、目录、版本）
gitli admin gc              # 清理过期 Session / 已删除仓库等
gitli admin reindex         # 无操作（按需 grep）
gitli admin audit list      # 查看审计日志
gitli admin generate-secret # 生成随机密钥
```

所有 CLI 操作都会写审计（`actor_type=system`，IP `127.0.0.1`）。

---

## 4. 备份与恢复

### 4.1 创建备份

```bash
# 明文 tar.gz
gitli backup --config /etc/gitli/config.toml --out /backup/gitli-$(date +%F).tar.gz

# 加密（AES-256-GCM + HMAC-SHA256）
export GITLI_BACKUP_KEY=<hex>
gitli backup --config /etc/gitli/config.toml --out /backup/gitli.tar.gz.enc --encrypt
```

备份内容：数据库、`repos/`、`lfs/`、`packages/`、`avatars/`、`attachments/`。

### 4.2 恢复

```bash
# 先停止服务
systemctl stop gitli

gitli restore --in /backup/gitli.tar.gz --dest /var/lib/gitli

systemctl start gitli
```

加密备份恢复时同样需要 `GITLI_BACKUP_KEY`。

### 4.3 演练建议

1. 在测试实例创建用户、仓库、Issue、推送代码。
2. `gitli backup` 生成归档。
3. 恢复到全新目录，用新配置启动。
4. 校验：能登录、仓库文件树与 README 正确、Issue 存在、`git clone` 成功。

> 本项目的备份恢复演练已通过：用户、仓库、Issue、Git 对象全部完整恢复。

---

## 5. 监控与健康检查

| 端点 | 说明 |
| --- | --- |
| `GET /healthz` | 数据库连通性；异常返回 503 `degraded` |
| `GET /metrics` | Prometheus 文本格式：计数器、`go_goroutines`、内存指标、`gitli_uptime_seconds` |
| `GET /debug/pprof/*` | 仅 `[server] pprof = true` 时启用 |

推荐告警：

- `/healthz` 非 200
- `go_goroutines` 持续单调增长
- 进程 RSS 持续增长（参考 `test/stress.sh` 的阈值）

---

## 6. 日志

- 格式：`[log] format = "json"`（生产建议）或 `text`
- 级别：`debug` / `info` / `warn` / `error`
- 敏感信息（密码、Token、Secret）在日志中一律脱敏为 `***`

---

## 7. 安全基线

- 禁止以 root 运行（`allow_root = true` 可强制，强烈不建议）。
- 生产必须启用 HTTPS，并正确设置 `http_external_url`（Cookie `Secure` 依赖它）。
- 定期执行 `gitli admin gc`、检查审计日志、轮换 Runner Token。
- 升级后如遇结构异常，先 `gitli migrate`；正常升级无需手动迁移。
