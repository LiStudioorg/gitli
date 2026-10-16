# gitli 开发状态（STATUS）

> 每次完成一个小模块就更新本文件。上下文被压缩后先读本文件恢复记忆。

## 当前进度

**阶段**：Block 1–4 全部功能已实现并推送；Runner 独立二进制已完成联调；主/子仓库 CI 均全绿。
**HEAD**：`9e2a9bf`（main）
**最近更新**：Runner 后端 + 前端 + 文档；公开仓库匿名 clone 修复；CD 流水线修复。

## 关键 commit

- `9e2a9bf` feat: runner admin frontend + workflow tab, docs, config
- `0f37a57` fix: allow anonymous read of public repos over git http, add runner register endpoint
- `7d627c6` feat: runner support backend (id+token auth, admin api, jobs, logs, sse, workflow trigger)
- `3607954` ci(cd): decouple docker image build from goreleaser (use build-push-action)
- `4b392d5` ci(cd): use goreleaser native build (free edition), explicit target matrix
- `456ccc3` chore: remove temporary development memory files, simplify AGENTS.md

## 已验证

- 主 gitli：gofmt / vet / test（30 包）/ golangci-lint 0 issue / 静态二进制 / 交叉编译；CI success。
- gitli-runner：gofmt / vet / test / 静态二进制；CI success。
- **Runner 端到端联调通过**：register → heartbeat → jobs/next → git clone → 执行 workflow → 流式日志 → 报告 success（日志含 `workflow executed OK`）。
- 空闲内存 16.9 MB（目标 < 150 MB）。

## 已知情况（非阻塞）

- 本地沙箱 SHA-256 在部分场景被 stub；依赖哈希的会话本地端到端不稳，云端正常。
- 本地无 docker/goreleaser；完整 CD 在云端验证。
- 部分边缘功能（reactions、subscriptions、path_permissions 的 API）尚未接线，服务/表已就绪。

## 下一步

- 触发 v0.1.0 正式 Release，验证 CD（用户指示后执行）。
- 补边缘 API（reactions/subscriptions/path_permissions）与更多测试（可选）。
