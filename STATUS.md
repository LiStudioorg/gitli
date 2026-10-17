# gitli 开发状态（STATUS）

> 每次完成一个小模块就更新本文件。上下文被压缩后先读本文件恢复记忆。

## 当前进度

**阶段**：批次 4 完成（LDAP / SAML / 依赖图 / 入站邮件）；批次 1-3 已完成；批次 1 完成（deploy keys / CODEOWNERS / 重命名重定向 / 软删除恢复 / SPDX 许可证 / Issue-PR 模板 / 归档）。已完成：SSH key 双用途 + GPG 签名密钥；企业级审计系统（表扩展/异步写入/查询导出统计/归档/告警/前端）。功能批次 1-5 与阶段 A-C 见 TODO.md。
**HEAD**：`0bd3ca4`（main）
**发布**：`v0.1.2`（29 附件 + ghcr.io 多架构镜像），`v0.1.1`（29 附件）。v0.1.0 首次 CD 因需 svft/syft 失败（未创建 Release），已用后续版本替代。
**最近更新**：修复 CD（去掉 sboms、Dockerfile 基础镜像升级）；发布流水线全绿。

## 关键 commit

- `ea0a789` feat: ci artifacts/cache/runners, lfs locks, review comments, user webhooks
- `f40896c` feat: repo topics, CI secrets, user blocks, notification prefs, org 2FA policies
- `61c8840` feat: reactions, subscriptions, path permissions, commit signature verification
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
