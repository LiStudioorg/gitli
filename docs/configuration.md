# 配置参考

gitli 使用 TOML 配置。默认读取当前目录的 `config.toml`，可用 `--config` 指定路径。

## 优先级

**环境变量 > 配置文件 > 内置默认值。**

环境变量前缀 `GITLI_`，嵌套字段以 `_` 连接。

## 段落

### [server]

| 键 | 说明 | 默认 |
| --- | --- | --- |
| `http` | HTTP 监听地址 | `:8080` |
| `ssh` | SSH 监听地址 | `:2222` |
| `external_url` | 对外访问地址 | `http://localhost:8080` |

### [database]

| 键 | 说明 | 默认 |
| --- | --- | --- |
| `driver` | `sqlite` / `mysql` / `postgres` | `sqlite` |
| `dsn` | 连接串 | `./data/gitli.db` |
| `max_open_conns` | 最大连接数 | `25` |
| `max_idle_conns` | 空闲连接数 | `5` |

### [storage]

`repos`、`lfs`、`packages`、`avatars`、`attachments` 各自指定存储根目录。

### [security.password]

Argon2id 参数：`memory_mb`（64）、`iterations`（3）、`parallelism`（2）、`salt_length`（16）、`key_length`（32）、`pepper`（仅环境变量）。

### [security.crypto]

`key_hex`：32 字节 hex 密钥，仅从 `GITLI_CRYPTO_KEY` / `GITLI_SECURITY_CRYPTO_KEY` 读取。

### [security.login]

登录限流：`max_attempts`（5）、`lockout_minutes`（15）、`window_minutes`（15）。

### [modules]

功能开关：`core`、`collaboration`、`git_ext`、`registry`、`ci`、`ide`、`ops`、`search`。
`[modules.packages].enabled` 列出启用的包格式。

### [limits] / [limits.packages]

并发与包体大小上限。

### [log]

`level`（`debug`/`info`/`warn`/`error`）、`format`（`text`/`json`）。

### [mail]

SMTP 配置，`password` 仅从 `GITLI_MAIL_PASSWORD` 读取。

## 环境变量示例

```bash
export GITLI_DATABASE_DRIVER=postgres
export GITLI_DATABASE_DSN="postgres://user:pass@127.0.0.1:5432/gitli?sslmode=disable"
export GITLI_PEPPER="a-long-random-pepper"
export GITLI_CRYPTO_KEY="$(openssl rand -hex 32)"
```
