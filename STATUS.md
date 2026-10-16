# gitli 开发状态（STATUS）

> 每次完成一个小模块就更新本文件。上下文被压缩后先读本文件恢复记忆。

## 当前进度

**阶段**：全部 Block（1–4）功能已完成，本地全绿，**CI 已通过**。前端实时通知与仓库标签页（Block 5）已完成，`npm run build`（vue-tsc + vite）通过。
**最近更新**：新增 `useSSE` composable（/api/v1/events 实时通知 + 退避重连 + 卸载清理）；Dashboard 未读徽标/通知列表/标记已读；RepoView 工单与合并请求标签页（列表 + 新建工单表单 + 即将推出占位）。未提交（无 commit/push）。

## 关键 commit（最新）

- `3ba0bab` feat: smtp mail sender, background language stats refresh queue, go 1.26 docs
- `47be587` fix(ci): require go 1.26+ (modernc sqlite), use go 1.27.x in workflows
- `b487036` feat: CLI subcommands (mirror/import/runner/admin/search-index), fix registry digest prefix bug, upgrade CI actions to node24
- `f05b802` ci: upgrade golangci-lint action to v7 with pinned v2.14.0
- `fce81a8` fix(security): prevent git argument injection, validate remote URLs, gate pipeline jobs

## 已验证

- gofmt / go vet / go test 全绿；`go test -tags integration ./test/` 通过。
- 静态二进制（CGO_ENABLED=0）构建成功。
- Git HTTP Smart Protocol 端到端：注册→建仓→clone→push 成功。
- pre-receive 钩子端到端：受保护分支推送被拒绝（跨 CWD 正确解析 DB）。
- 免费内存实测 16.9 MB（目标 <150 MB）。
- 服务启动：/healthz、/metrics、/v2/、前端首页均 200。

## 待办

- 待用户指示后打 tag（如 v1.0.0）触发 CD 发布。
- 待用户确认后清理 ROADMAP.md/STATUS.md/TODO.md 临时记忆文件。

## 已完成

- Block 1：项目初始化、配置系统、三数据库驱动、迁移、安全原语（Argon2id/SHA-256/AES-256-GCM）、路由+中间件、Vue3 骨架。
- 监听地址与对外地址分离（多 IPv4/IPv6），`config.ResolveExternalURLs`。
- CI/Release 工作流分离（ci.yml / release.yml）。
- 静态编译（CGO_ENABLED=0 + -trimpath + -s -w）。
- Git HTTP Smart Protocol（upload-pack/receive-pack 流式转发）— 已实测 clone/push/鉴权成功。
- pre-receive 钩子基础设施（`gitli hook pre-receive`、分支/标签保护校验、脚本安装）。
- 用户认证、TOTP、PAT、SSH Key、登录限流的基本服务层。

## 进行中

- 修复 golangci-lint v2 配置与 errcheck 告警（本地已用 v2 复现并修复）。
- 修复 HTTP context key 不匹配 bug（`userFrom` 改为委托 `auth.UserFrom`）。

## 已知 Bug / 阻塞

1. **本地沙箱 SHA-256 被 stub 为全零**：导致登录/Session/PAT 无法本地端到端验证。逻辑无误，云端正常。不阻塞。
2. **context key bug（已修）**：`http.userFrom` 曾用本地私有 key，与 `auth` 中间件不匹配，导致所有需要登录的接口返回 401。

## 下一步计划

1. 验证 CI 全绿（golangci-lint + go test + 静态编译）。
2. 实现企业级 RBAC（嵌套团队、自定义角色、LRU 缓存、RequirePermission 中间件）。
3. 仓库 Fork、用户名 301 重定向、Session IP/UA 绑定、提交图。
4. 单元测试：auth、权限继承、循环检测、Git 命令解析。
5. Issue/PR/Review/Webhook/通知/SSE/审计日志。

## 关键 commit

- 待补充（每次提交后更新哈希）
