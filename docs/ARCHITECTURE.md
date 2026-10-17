# 架构（ARCHITECTURE）

本文件是 gitli 架构的**权威文档**（原 `docs/architecture.md` 精简副本已合并进本文件，
避免在大小写不敏感的文件系统上互相覆盖）。

gitli 是**单二进制、严格分层、依赖单向**的 Git 服务平台。

---

## 1. 分层总览

```
main.go
  └── internal/app                      应用装配与生命周期
        ├── internal/transport          协议接入层
        │     ├── http                  HTTP / REST / Git Smart Protocol
        │     ├── ssh                   内置 SSH 服务器 / git-shell 适配
        │     └── sse                   Server-Sent Events
        ├── internal/service            业务逻辑层
        └── internal/infra              基础设施层
```

| 层 | 职责 | 允许依赖 |
| --- | --- | --- |
| `domain` | 纯领域结构体 | 仅标准库 |
| `infra` | git 子进程、文件系统、队列、加密、密码、Token、备份 | `domain` |
| `service` | 业务用例（用户、仓库、RBAC、Issue/PR…） | `domain`、`infra`、`config`、`store` |
| `transport` | HTTP / SSH / SSE 协议接入 | `service`、`domain` |
| `app` | 装配、启动、优雅退出、钩子、SSH 适配 | 全部 |

---

## 2. 依赖方向（严格单向）

```
main.go → internal/app → {transport, service, infra}

service   → domain, infra, config, store
transport → service, domain
infra     → domain
domain    → 仅标准库
```

**禁止**：

- `domain` 依赖任何其它 internal 包。
- `service` 直接 import `transport`。
- 业务包直接 import 具体数据库驱动（只能用 `database/sql`）。
- 任何形式的循环依赖。

违反依赖方向会导致编译失败，这是有意设计的护栏。

---

## 3. 包地图

```
gitli/
├── main.go                       入口，子命令分发
├── internal/
│   ├── app/                      装配、生命周期、hook、ssh 适配
│   ├── config/                   config.go / load.go / defaults.go / external.go / template.go
│   ├── db/                       三驱动 + 迁移执行 + Rebind
│   ├── migrations/{sqlite,mysql,postgres}/
│   ├── transport/{http,ssh,sse}/ 协议接入
│   ├── domain/                   纯结构体（仅标准库）
│   ├── service/                  业务逻辑（见下）
│   ├── infra/                    gitcmd / fsrepo / queue / mail / oidc / crypto / password / token / backup / secutil / netutil / weakpass
│   ├── platform/                 跨平台抽象（SignalSet、build tag 隔离）
│   └── version/                  版本信息
├── web/                          前端（embed.go + dist + src）
├── api / scripts / deploy / docs / test
```

**service 子包**：`auth`、`user`、`org`、`repo`、`git`、`issue`、`pull`、`comment`、`wiki`、`gist`、`release`、`lfs`、`webhook`、`notification`、`mirror`、`import`、`search`、`package`、`pkgsvc`、`registry`、`ci`、`ide`、`badge`、`activity`、`audit`、`metrics`、`pat`、`rbac`、`store`。

---

## 4. 数据库抽象

业务层只依赖标准库 `database/sql`，三类驱动（SQLite `modernc.org/sqlite`、MySQL、PostgreSQL `pgx/v5/stdlib`）在 `internal/db` 内注册与选择。

- **占位符统一**：业务 SQL 一律写 `?`；PostgreSQL 由 `db.Rebind` 改写为 `$1`、`$2`…，业务代码无需感知方言。
- **方言差异**：主键、时间、布尔类型按方言分别建模，见下表。
- **迁移**：文件按方言分目录（`internal/migrations/{sqlite,mysql,postgres}`），命名 `NNNN_name.sql`，支持 up/down；`gitli migrate` 依据 `database.driver` 选择目录并执行。

| 概念 | SQLite | MySQL | PostgreSQL |
| --- | --- | --- | --- |
| 主键 | `INTEGER PRIMARY KEY AUTOINCREMENT` | `BIGINT AUTO_INCREMENT PRIMARY KEY` | `BIGSERIAL PRIMARY KEY` |
| 时间 | `DATETIME` | `DATETIME` | `TIMESTAMPTZ` |
| 布尔 | `INTEGER` | `TINYINT(1)` | `BOOLEAN` |
| 占位符 | `?` | `?` | `$1`（由 `Rebind` 改写） |

> 迁移执行器的 `splitStatements` 会剥离 `--` 整行注释，但绝不能让注释吞掉语句（历史 bug，已修）。

---

## 5. 单二进制与 embed

- 前端构建产物 `web/dist` 通过 `//go:embed` 内嵌。
- 数据库迁移文件同样 embed。
- `go build -o gitli .` 产出**唯一**可执行文件；所有构建场景统一 `CGO_ENABLED=0 -trimpath -ldflags="-s -w ..."`，纯静态、可运行于 Alpine / scratch。

---

## 6. 低内存策略

- Git / 打包 / CI 等重任务**按需启动子进程**，并以并发信号量限流（`limits.max_git_concurrency` 等）。
- 大文件上传下载全程**流式**（`io.Copy` / `io.Pipe`），备份加密分块流式处理，不整包读入内存。
- 可选模块可通过配置完全关闭；关闭时不初始化、不分配资源、不启动 goroutine。
- 空闲内存目标 **< 150 MB**。

---

## 7. 跨平台

- 禁止硬编码路径分隔符与平台 syscall；平台相关代码用 build tag 隔离，集中于 `internal/platform`。
- 信号由 `platform.SignalSet()` 提供：Unix 监听 `SIGINT`/`SIGTERM`，Windows 仅监听 `os.Interrupt`。
- 监听地址与对外地址严格分离，对外 URL 统一由 `config.ResolveExternalURLs` 生成。

---

## 8. 相关文档

- [安全模型](./SECURITY.md)
- [配置参考](./configuration.md)
- [CI/CD](./CI.md)
- [发布流程](./RELEASE.md)
