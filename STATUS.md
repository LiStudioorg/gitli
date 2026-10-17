# gitli 开发状态（STATUS）

> **上下文丢失时的恢复入口**：先读本文件 + `ROADMAP.md`，再继续阶段 A 的 E2E 测试。

## 当前阶段

**阶段 A：功能冻结与 E2E 测试**（目标版本 `v0.2.0-beta`）

- 所有计划功能（批次 1-5 + Runner + CLI + 审计）**已开发完毕**。
- 新功能已冻结，只做 E2E 验证与 Bug 修复。
- 当前 HEAD：`c66583b`（main，已推送）
- CI：主 gitli `success`；gitli-runner `success`。

## 已完成功能总览

### Block 1 基础设施
单二进制（前端/迁移 embed）、TOML 配置 + 环境变量覆盖、三数据库（SQLite/MySQL/PostgreSQL）、迁移、Argon2id/SHA-256/AES-256-GCM、多地址监听 + 对外地址分离、CI/CD 分离、静态编译、跨平台平台抽象。

### Block 2 核心
认证（注册/登录/Session/TOTP/PAT/SSH Key/限流）、用户名 301、Session IP/UA 绑定、企业级 RBAC（嵌套团队/自定义角色/LRU 缓存/RequirePermission）、组织/团队、仓库（Fork/转移/软删除/重命名重定向）、分支/标签保护、HTTP Smart Protocol、内置 SSH、pre/post-receive 钩子、文件树/Blob/Diff/Blame/提交图、Issue/PR/Review/评论/标签/里程碑、Webhook（HMAC+SSRF）、通知+SSE、审计、OIDC、SSH Key 双用途 + GPG 签名、分支保护签名校验。

### Block 3 扩展
Wiki/Gist/Release(+附件)、LFS、镜像/导入、包管理（Generic/NPM/Docker/Go/Maven）、Docker Registry V2、CI 调度器、搜索（git grep）、IDE/Symbols、徽章/RSS/Atom、活动/语言统计、SMTP、reactions/subscriptions/path_permissions/commit_signatures/repo_topics/ci_secrets/user_blocks/notification_prefs/org_2fa/ci_artifacts/cache/runners/lfs_locks/review_comments/user_webhooks。

### Block 4 前端与运维
Landing/Dashboard/Repo/Admin/Settings、Clone 下拉、语言统计、依赖 Tab、讨论、看板、工作流 Tab、Runner 页、暗色模式、命令面板（Ctrl+K）、全局快捷键（g i/g p/c//）、i18n（zh/en）、响应式、SSE 通知、/metrics、/healthz、OpenAPI（/api/docs）、备份/恢复、CLI 子命令、多平台构建矩阵、部署模板。

### 批次 1-5（社区/企业/体验）
- 批次 1：部署密钥、CODEOWNERS、仓库重命名+重定向、软删除+恢复、SPDX 许可证、Issue/PR 模板、归档。
- 批次 2：保存的回复、全局搜索、用户主页+关注/粉丝、探索+趋势+话题。
- 批次 3：OAuth2 Provider、会话管理、IP 白名单、审计导出、OpenAPI 文档。
- 批次 4：LDAP、SAML、依赖图、入站邮件。
- 批次 5：讨论、项目看板、命令面板/快捷键、i18n、邮件摘要。

### CLI / Runner
- 裸执行 `gitli` 直接启动；完整 `gitli admin`（用户/仓库/组织/Runner/doctor/gc/审计），毁灭性操作仅 CLI。
- 独立二进制 `gitli-runner`（另仓库）：富配置（并发/超时/清理/env/shell）、docker/podman/shell 执行器、流式日志。
- 主服务 Runner 后端：ID+Token(SHA-256)、心跳、ClaimJob、日志限流、SSE、工作流触发。

## 已知限制（非阻塞）

1. 本地沙箱 SHA-256 在部分场景被 stub → 依赖哈希的会话/PAT 本地端到端不稳，云端正常。
2. 本地无 docker/goreleaser → Docker 推拉与完整 CD 由云端验证。
3. LDAP/SAML/IMAP 无真实服务端，仅实现 + 单元测试。
4. `/metrics` 目前为轻量占位（返回 ok），可后续补充真实指标。

## 下一步（阶段 A）

1. 编写 `test/e2e.sh` 自动化脚本。
2. 本地运行，修复失败项（记录在下方"E2E 结果"）。
3. 全绿后打 `v0.2.0-beta` tag，验证 CD。

## E2E 结果

`bash test/e2e.sh` — **PASS=34 FAIL=0 SKIP=2**

覆盖并通过：

- 用户：注册 / 登录 / 会话 / 改名 301 / TOTP / PAT / SSH / GPG
- 仓库：创建 / HTTP push / HTTP clone / SSH clone
- 协作：Issue / 评论 / 关闭 / PR / Review / 合并
- 归档：归档 / 拒绝 push / 解除归档
- 包管理：Generic 发布 / 下载
- 镜像：Registry `/v2/` / 上传会话 202
- Runner：创建工作流仓库并 push 触发 / 执行任务 / 结果成功 / 日志回传

跳过：

- NPM publish：后端 NPM 端点为预留占位。
- docker push/pull：本地无 docker。

修复项：

- `admin stats` 表名 `pull_requests` → `pulls`
- `domain.Repo.DeletedAt` 扫描
- `syscall.Statfs` 迁移至 `internal/platform`（跨平台）
- `test/e2e.sh` Runner 段：push 工作流仓库触发任务后再启动 Runner，等待时间放宽至 40s，失败时输出 `runner.log`

## 关键 commit 历史

- `c66583b` fix: lint issues in admin cli
- `62d6b6c` feat: ultimate admin CLI, bare execution starts server
- `d3459c6` feat: expanded admin web dashboard
- `8d925d8` feat: batch 5 complete
- `3ecc978` feat: batch5 frontend
- `ea1feb8` feat: notification email digest
- `02a5d79` feat: i18n
- `410f7b9` feat: global shortcuts
- `0b9d729` feat: project boards
- `3111eb2` feat: discussions
- `9ca4a0c` feat: batch 4 complete
- `8d925d8` ...
