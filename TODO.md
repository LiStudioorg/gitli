# gitli 开发任务清单

> 外部记忆文件。每次完成小模块就更新。上下文被压缩后先读本文件 + `ROADMAP.md` + `STATUS.md`。

## 正在进行
- [x] SSH key 双用途（认证 + 签名）+ GPG 签名密钥

## 企业级审计系统（用户单独要求，已完成）
- [x] audit_logs 表扩展（actor/event/resource/target/old/new/result/request_id/metadata + 索引）
- [x] audit_logs_archive 归档表 + audit_alert_rules 告警表
- [x] 事件标准化（resource.action）+ 异步写入
- [x] 查询/详情/流式导出(CSV/JSON)/统计 API
- [x] 保留策略（retention_days）+ 每日归档任务
- [x] 告警规则（邮件/Webhook）
- [x] 前端管理面板审计 Tab（筛选/图表/导出/告警规则）
- [x] 用户「安全活动」页（我的活动 + 会话管理）
- [x] 文档 docs/AUDIT.md + README + AGENTS 约束
- [x] git push 事件审计

## 功能批次

### 批次 1：基础协作 + 归档（进行中）
- [ ] 1.1 部署密钥（Deploy Key）：`deploy_keys` 表、只读/读写、API、前端、SSH 认证
- [ ] 1.2 CODEOWNERS：解析根/`.github/`/`docs/`、路径模式、PR 自动请求 review、分支保护 `require_codeowners_review`
- [ ] 1.3 仓库重命名 + 重定向：`repo_redirects` 表、Web 301、Git HTTP 301、SSH 重定向
- [ ] 1.4 仓库软删除 + 恢复：`repos.deleted_at`、30 天清理任务、管理后台恢复、用户自查
- [ ] 1.5 许可证自动识别（SPDX）：解析 LICENSE、写 `repos.license`、首页徽章
- [ ] 1.6 Issue/PR 模板：`.gitli`/`.github` 模板、YAML front matter、创建时选择
- [ ] 1.7 归档功能：`repos.archived`/`archived_at`、禁止 push/Issue/PR/Wiki/Release、仅允许 clone/浏览/Star/Fork、组织级归档、审计

### 批次 2：用户与社区
- [ ] 2.1 保存的回复（Saved Replies）：`saved_replies` 表 + API + 前端
- [ ] 2.2 全局搜索 + 高级搜索：跨仓库/Issue/PR/用户/组织、`language:`/`stars:`/`is:`/`author:`/`org:`
- [ ] 2.3 用户主页 + 关注/粉丝：`/username`、`follows` 表、关注按钮、粉丝列表
- [ ] 2.4 探索页 + 趋势 + 话题：`/explore`、`/topics/{topic}`、`trending_cache`、每日聚合

### 批次 3：企业安全
- [ ] 3.1 OAuth2 应用（Provider）：`oauth_apps`/`oauth_authorizations`、授权/Token/用户信息端点、管理页
- [ ] 3.2 会话管理 UI：列表/单个注销/全部注销，显示 IP/UA/时间
- [ ] 3.3 IP 允许列表：`ip_allowlist` 表、中间件、组织设置
- [ ] 3.4 审计日志导出：CSV/JSON 流式导出
- [ ] 3.5 API 文档（OpenAPI）：swaggo 生成 + Swagger UI `/api/docs`

### 批次 4：企业集成
- [ ] 4.1 LDAP 登录：`[ldap]` 配置、认证后关联本地用户、组映射
- [ ] 4.2 SAML 登录：crewjam/saml、SP/IdP-initiated
- [ ] 4.3 依赖图：解析 go.mod/package.json/pom.xml/requirements.txt、`repo_dependencies`、依赖 Tab
- [ ] 4.4 入站邮件（Reply by Email）：IMAP、reply+{token}、评论写入、附件

### 批次 5：体验提升
- [ ] 5.1 讨论（Discussions）：`discussions`/`discussion_comments`/`discussion_categories`、分类、标记答案
- [ ] 5.2 项目看板（Projects）：`projects`/`project_columns`/`project_cards`、拖拽、筛选
- [ ] 5.3 命令面板 + 快捷键：Ctrl+K、`g i`/`g p`/`c`/`/`
- [ ] 5.4 多语言（i18n）：Vue i18n 中英文、后端 Accept-Language
- [ ] 5.5 邮件摘要：每日/每周聚合、通知频率设置

## 生产就绪阶段

### 阶段 A：功能冻结
- [ ] 完成批次 1-5
- [ ] 完成 E2E 测试，修复所有 Bug
- [ ] 冻结新功能
- [ ] 发布 v0.2.0（Beta）

### 阶段 B：稳定化
- [ ] B1 长期运行测试（7 天：内存/CPU/错误率）
- [ ] B2 压力测试（100 用户 / 1000 仓库 / 50 并发 Git）
- [ ] B3 安全审计（govulncheck + SQL 注入/XSS/CSRF/SSRF/路径穿越/命令注入/越权）
- [ ] B4 监控（/metrics、/healthz、JSON 日志、告警规则文档）
- [ ] B5 备份恢复（加密、校验、恢复演练）
- [ ] B6 文档（ADMIN.md / USER.md / API.md / TROUBLESHOOTING.md / SECURITY.md）
- [ ] B7 发布 v0.9.0（RC）

### 阶段 C：生产就绪
- [ ] staging 环境完整验证
- [ ] 小范围内部试用（10-20 人）
- [ ] 收集反馈，修复关键问题
- [ ] 发布 v1.0.0，打 tag 触发 CD

## 已完成

### 基础设施
- [x] 项目初始化、TOML 配置、三数据库、迁移
- [x] 安全原语（Argon2id / SHA-256 / AES-256-GCM）
- [x] 多地址监听 + 对外地址分离
- [x] CI/CD 工作流分离 + 静态编译 + 跨平台
- [x] 发布流水线打通（v0.1.1 / v0.1.2，29 附件 + ghcr.io 多架构镜像）

### 核心
- [x] 用户认证（注册/登录/Session/TOTP/PAT/SSH Key/限流）
- [x] 用户名 301 重定向 + Session IP/UA 绑定
- [x] 企业级 RBAC（嵌套团队、自定义角色、LRU 缓存、RequirePermission）
- [x] 组织 / 团队、仓库（含 Fork/转移）、分支/标签保护
- [x] HTTP Smart Protocol + 内置 SSH + pre/post-receive 钩子
- [x] 文件树/Blob/Diff/Blame/提交历史/提交图
- [x] Issue/PR/Review/评论/标签/里程碑
- [x] Webhook（HMAC+SSRF）、通知+SSE、审计、OIDC

### 扩展
- [x] Wiki/Gist/Release+附件、LFS、镜像/导入
- [x] 包管理（Generic/NPM/Docker/Go/Maven）、Docker Registry V2
- [x] CI 调度器、搜索、IDE/Symbols、徽章/RSS/Atom、活动/语言统计、SMTP
- [x] reactions/subscriptions/path_permissions/commit_signatures/repo_topics
- [x] ci_secrets/user_blocks/notification_prefs/org_2fa/ci_artifacts/cache/runners/lfs_locks/review_comments/user_webhooks

### 前端与运维
- [x] Landing/Dashboard/Repo/Admin/Settings + 工作流 Tab + Runner 页
- [x] 暗色模式、命令面板、响应式、SSE 通知、Explore 页
- [x] /metrics + /healthz、备份/恢复 CLI、CLI 子命令、多平台构建、部署模板

### Runner
- [x] gitli 主服务 Runner 后端 + 前端
- [x] 独立二进制 gitli-runner（docker/podman/shell、workflow、流式日志）
- [x] 端到端联调通过

### E2E 验收（已通过）
- [x] 认证/改名301/TOTP/PAT/组织/HTTP/SSH clone-push/Issue（17/17）
- [x] Issue评论/关闭/PR/Review/合并/Webhook/Registry/包管理/Runner/公开接口（16/16）
- [x] 前端 SPA/资源/安全头（13/13）
- [x] 内存空闲 17.4 MB
- [x] 修复：SSH git 子命令、包管理 CSRF/Basic 认证
