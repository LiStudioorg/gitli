# gitli 开发任务清单

## 正在进行
- [ ] 前端完善（暗色模式/响应式/命令面板/无障碍）
- [ ] Web IDE 占位

## 已完成
- [x] 项目初始化（AGENTS.md, README.md, LICENSE）
- [x] 配置系统（TOML + 环境变量覆盖）
- [x] 数据库驱动（SQLite / MySQL / PostgreSQL）
- [x] 安全原语（Argon2id, SHA-256, AES-256-GCM）
- [x] 多地址监听 + 对外地址分离
- [x] CI/Release 工作流分离（ci.yml / cd.yml）
- [x] 静态编译（CGO_ENABLED=0 + -s -w）
- [x] HTTP Smart Protocol + pre-receive 钩子
- [x] RBAC（嵌套团队、自定义角色、LRU 缓存、API、中间件）
- [x] 用户名 301 重定向、仓库 Fork、提交图、Session IP/UA 绑定
- [x] 搜索（git grep）、徽章/RSS/Atom、LFS、包 Blob Store
- [x] Wiki / Gist / Release、镜像 / 导入
- [x] CI 调度器（限并发、ClaimJob）
- [x] Docker Registry V2、包格式（Generic/NPM/Go/Maven）
- [x] 指标 /metrics、管理面板 API、备份/恢复 CLI

## 待办（Block 2）
- [x] 用户认证（注册、登录、Session）
- [ ] 用户名更改 + 301 重定向
- [ ] TOTP 2FA（启用、验证、禁用、恢复码）
- [ ] 经典 PAT + 细粒度 PAT
- [ ] 企业级 RBAC（嵌套团队、自定义角色、细粒度权限）
- [ ] 仓库管理（创建、删除、转移、Fork）
- [x] 分支保护 + 标签保护
- [x] HTTP Smart Protocol
- [ ] 内置 SSH 服务器（命令白名单）
- [ ] Issue / PR / Review / 评论 / 标签 / 里程碑
- [ ] Webhook 投递（HMAC-SHA256 + SSRF 防护）
- [ ] 站内通知 + SSE
- [ ] 审计日志
- [ ] 提交图 / 文件树 / Blob / Diff / Blame / 历史

## 待办（Block 3）
- [ ] Wiki / Gist / Release + 附件
- [ ] Git LFS / 提交签名验证
- [ ] 仓库镜像 / 导入工具
- [ ] 包管理（Generic, NPM, Docker/OCI, Go, Maven）
- [ ] Docker Registry V2 认证 + 分块上传
- [ ] 简化 CI（宿主机执行）
- [ ] 搜索（git grep / SQLite FTS）
- [ ] 状态徽章 / RSS / 活动时间线 / 语言统计

## 待办（Block 4）
- [ ] 前端 Landing / Dashboard / Repo 页面
- [ ] Clone 下拉菜单 + 语言统计
- [ ] 暗色模式 / 响应式 / 无障碍
- [ ] 指标 /metrics + 健康检查 /healthz
- [ ] 管理面板
- [ ] 备份/恢复 CLI
- [ ] 多平台构建 + Docker 镜像 + deb/rpm/apk
- [ ] 最终验收
