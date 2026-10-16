# gitli 开发任务清单

## 正在进行
- [ ] 最终验收（等待用户指示打 tag v1.0.0）

## 已完成

### 基础设施
- [x] 项目初始化（AGENTS.md, README.md, LICENSE, config.example.toml）
- [x] TOML 配置 + 环境变量覆盖 + 路径绝对化
- [x] 数据库驱动（SQLite / MySQL / PostgreSQL）+ 迁移
- [x] 安全原语（Argon2id, SHA-256, AES-256-GCM）
- [x] 多地址监听 + 对外地址分离（IPv4/IPv6）
- [x] CI/CD 工作流分离（ci.yml / cd.yml）
- [x] 静态编译（CGO_ENABLED=0 + -trimpath + -s -w）
- [x] 跨平台（build tag、信号、权限）

### Block 2 核心
- [x] 用户认证（注册/登录/Session/TOTP/PAT/SSH Key/登录限流）
- [x] 用户名 301 重定向 + Session IP/UA 绑定
- [x] 企业级 RBAC（嵌套团队、自定义角色、LRU 缓存、RequirePermission 中间件）
- [x] 仓库创建/删除/转移/Fork、协作者、分支/标签保护
- [x] HTTP Smart Protocol、内置 SSH、pre-receive 钩子
- [x] 文件树/Blob/Diff/Blame/提交历史/提交图
- [x] Issue/PR/Review/评论/标签/里程碑
- [x] Webhook/站内通知/SSE/审计日志

### Block 3 扩展
- [x] Wiki/Gist/Release、Git LFS、镜像/导入
- [x] 包管理（Generic/NPM/Docker/Go/Maven）+ Docker Registry V2
- [x] CI 调度器（限并发、ClaimJob、宿主机执行）
- [x] 搜索（git grep）、符号搜索（IDE）
- [x] 徽章/RSS/Atom、语言统计（后台队列刷新）
- [x] SMTP 邮件发送

### Block 4 前端与运维
- [x] Landing/Dashboard/Repo/Admin/Settings 页面
- [x] Clone 下拉菜单（HTTPS/SSH/CLI 占位）、语言统计进度条
- [x] 暗色模式/命令面板/响应式/无障碍
- [x] 指标 /metrics、健康检查 /healthz
- [x] 管理面板 API、备份/恢复 CLI
- [x] CLI 子命令（serve/migrate/hook/backup/restore/mirror/import/runner/admin/search-index）

### 质量与安全
- [x] 单元测试 + 集成测试（build tag）
- [x] 安全审计与加固（git 参数注入、URL 校验、权限越权）
- [x] CI 全绿（gofmt/vet/test/lint/静态编译/交叉编译）
- [x] 文档（README/docs: SECURITY/CI/RELEASE/ARCHITECTURE/SECURITY_AUDIT）

## 待办（等你指示）
- [ ] 打 tag v1.0.0 触发 CD 发布（全平台 + Docker + deb/rpm/apk）
- [ ] 清理 ROADMAP.md/STATUS.md/TODO.md 临时记忆文件
