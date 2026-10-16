# gitli 开发任务清单

## 正在进行
- [ ] 触发 v0.1.0 正式 Release，验证 CD 流水线（用户指示后执行）

## 已完成

### 边缘 API 补全（全 72 张表均已接线）
- [x] reactions（表情回应）
- [x] subscriptions（订阅）
- [x] path_permissions（路径级权限）
- [x] commit_signatures（提交签名验证）
- [x] repo_topics（仓库主题）
- [x] ci_secrets（CI 密钥，AES-256-GCM 加密）
- [x] user_blocks（用户屏蔽）
- [x] notification_prefs（通知偏好）
- [x] org_2fa_policies（组织 2FA 策略）
- [x] ci_artifacts / ci_cache / ci_runners（CI 制品/缓存/Runner）
- [x] lfs_locks（LFS 文件锁）
- [x] review_comments（行级评论）
- [x] user_webhooks（用户级 Webhook）

## 已完成

### Block 1 基础设施
- [x] 项目初始化（AGENTS/README/LICENSE/config.example.toml）
- [x] TOML 配置 + 环境变量覆盖 + 路径绝对化
- [x] 数据库驱动（SQLite/MySQL/PostgreSQL）+ 迁移
- [x] 安全原语（Argon2id / SHA-256 / AES-256-GCM）
- [x] 多地址监听 + 对外地址分离
- [x] CI/CD 工作流分离 + 静态编译 + 跨平台

### Block 2 核心
- [x] 用户认证（注册/登录/Session/TOTP/PAT/SSH Key/登录限流）
- [x] 用户名 301 重定向 + Session IP/UA 绑定
- [x] 企业级 RBAC（嵌套团队、自定义角色、细粒度权限、LRU 缓存、RequirePermission 中间件）
- [x] 组织 / 团队管理
- [x] 仓库创建/删除/转移/Fork、协作者、分支/标签保护
- [x] HTTP Smart Protocol + 内置 SSH + pre/post-receive 钩子
- [x] 文件树/Blob/Diff/Blame/提交历史/提交图
- [x] Issue/PR/Review/评论/标签/里程碑（含编辑、状态）
- [x] Webhook 投递（HMAC-SHA256 + SSRF 防护）
- [x] 站内通知 + SSE + 审计日志
- [x] OIDC / OAuth2 登录（PKCE）

### Block 3 扩展
- [x] Wiki / Gist / Release + 附件上传下载
- [x] Git LFS
- [x] 仓库镜像 / 导入工具
- [x] 包管理（Generic / NPM / Docker / Go / Maven）
- [x] Docker Registry V2（blob 上传会话、manifest/tag、/v2/ 路由）
- [x] CI 调度器（限并发、ClaimJob）
- [x] 搜索（git grep）
- [x] 状态徽章 / RSS / Atom / 活动时间线 / 语言统计（后台刷新）
- [x] SMTP 邮件发送

### Block 4 前端与运维
- [x] Landing / Dashboard / Repo / Admin / Settings 页面
- [x] Clone 下拉菜单 + 语言统计进度条 + 工作流 Tab
- [x] 暗色模式 / 命令面板 / 响应式 / 无障碍 / SSE 通知
- [x] 指标 /metrics + 健康检查 /healthz
- [x] 管理面板（用户/仓库/任务/审计/Runner）
- [x] 备份/恢复 CLI（加密 + HMAC）
- [x] CLI 子命令（serve/migrate/hook/backup/restore/mirror/import/runner/admin/search-index）
- [x] 多平台构建 + Docker 多架构镜像 + deb/rpm/apk
- [x] 文档（README + docs: SECURITY/CI/RELEASE/ARCHITECTURE/SECURITY_AUDIT）

### Runner
- [x] gitli 主服务 Runner 后端（id+token、admin api、jobs、logs、sse、workflow 触发）
- [x] 前端 Runner 管理页 + 仓库工作流 Tab
- [x] 独立二进制 gitli-runner（docker/podman/shell、workflow 解析、流式日志）
- [x] 端到端联调通过

## 待办（可选 / 后续）
- [ ] reactions / subscriptions / path_permissions 的 HTTP API 接线
- [ ] 更全面的单元测试覆盖
- [ ] v0.1.0 正式 Release 验证 CD（用户指示后）
