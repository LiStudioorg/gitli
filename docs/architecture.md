# 架构概览

gitli 采用严格分层的单二进制架构。

```
main.go
  └── internal/app          应用装配与生命周期
        ├── internal/transport/{http,ssh,sse}   协议接入
        ├── internal/service                    业务逻辑
        └── internal/infra                      基础设施
```

## 分层职责

| 层 | 职责 | 依赖 |
| --- | --- | --- |
| `domain` | 纯领域结构体 | 仅标准库 |
| `infra` | git 子进程、文件系统、队列、加密、密码、Token | domain |
| `service` | 业务用例 | domain、infra、config |
| `transport` | HTTP / SSH / SSE 协议 | service、domain |
| `app` | 装配、启动、优雅退出 | 全部 |

依赖方向严格单向，禁止反向或循环依赖。详见 [`AGENTS.md`](../AGENTS.md)。

## 单二进制

- 前端构建产物位于 `web/dist`，通过 `//go:embed` 嵌入。
- 数据库迁移位于 `internal/migrations/{sqlite,mysql,postgres}`，同样 embed。
- `go build -o gitli .` 产出唯一可执行文件。

## 数据库抽象

- 业务层只依赖 `database/sql`。
- 方言差异通过 `internal/db` 的 `Dialect` 与占位符重写（`?` ↔ `$1`）统一。
- 迁移脚本按方言分目录维护。

## 低内存策略

- Git / 打包 / CI 等重任务按需启动子进程，并以并发信号量限流。
- 大文件上传下载全程流式（`io.Copy` / `io.Pipe`）。
- 空闲内存目标 < 150 MB。
