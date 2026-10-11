# 安全模型（SECURITY）

本文档描述 gitli 的安全设计、实现要点与对应配置项。gitli 遵循**安全默认、按需加固、最小权限**的原则：所有敏感能力默认可关闭，关闭时不初始化、不分配资源、不启动 goroutine。

---

## 1. 密码哈希（Argon2id）

用户密码一律以 **Argon2id** 哈希后存储，永不落盘明文，也永不写入日志。

- **算法**：Argon2id（`golang.org/x/crypto/argon2`）。
- **参数**：memory `64 MiB`（`m=65536`）、iterations `3`（`t=3`）、parallelism `2`（`p=2`）、salt 16 字节、输出密钥 32 字节。
- **存储格式**：PHC 字符串，例如
  `$argon2id$v=19$m=65536,t=3,p=2$<base64-salt>$<base64-hash>`。
- **Pepper**：全局密钥盐 `pepper` 由配置注入，在哈希前与密码拼接，使数据库泄露后离线爆破的成本进一步升高。比较使用 `subtle.ConstantTimeCompare`，避免时序侧信道。
- **弱密码**：可选内置弱口令库检测（`top10000`），并可选用 HaveIBeenPwned k-匿名检测。
- **最小长度**：低于 `min_length` 的密码拒绝注册/改密。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `security.password.algorithm` | 算法（`argon2id`） | `argon2id` |
| `security.password.memory_mb` | 内存开销 MiB | `64` |
| `security.password.iterations` | 迭代次数 | `3` |
| `security.password.parallelism` | 并行度 | `2` |
| `security.password.salt_length` | 盐长度（字节） | `16` |
| `security.password.key_length` | 输出长度（字节） | `32` |
| `security.password.min_length` | 密码最小长度 | `12` |
| `security.password.hibp` | HIBP 弱密码检测 | `false` |
| `security.password.pepper` | 全局 pepper（仅环境变量 `GITLI_PEPPER`） | `""` |

> 修改 `pepper` 会导致所有既有密码无法验证，务必谨慎并做好迁移预案。

---

## 2. Token 哈希与 PAT 类型

所有长期凭证（PAT / Session / API Token）在数据库中**只存 SHA-256 哈希**，明文仅在创建时返回一次。校验时对提交值做 SHA-256 后与库中哈希比对。

- **Token 生成**：加密随机数 + 前缀，便于识别用途与轮换。
- **PAT 类型**：
  - **经典 PAT**：全局作用域（scope）集合，按需勾选权限；默认不勾选任何权限（最小权限）。
  - **细粒度 PAT**：绑定到指定仓库，并限定 `read` / `write` 能力。
- **有效期**：PAT 可设置过期时间，`max_expiry_days` 限制最长有效期；`0` 表示不限。
- **泄露处置**：可即时吊销，吊销后哈希立即失效。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `security.pat.allow_classic` | 允许经典 PAT | `true` |
| `security.pat.allow_fine_grained` | 允许细粒度 PAT | `true` |
| `security.pat.default_scopes_empty` | 经典 PAT 默认不勾选权限 | `true` |
| `security.pat.max_expiry_days` | 最长有效期（天），`0` 不限 | `365` |

---

## 3. 对称加密（AES-256-GCM）

需要回读的机密（TOTP secret、SMTP 密码、OAuth/OIDC client secret、Git 镜像凭证、CI Secrets 等）使用 **AES-256-GCM** 加密后存储。

- **密钥**：32 字节（64 个 hex 字符），仅从环境变量注入（`GITLI_CRYPTO_KEY` / `GITLI_SECURITY_CRYPTO_KEY`），不写入配置文件、不提交仓库。
- **密文标识**：加密字段带 `enc:` 前缀，读取时据此判断是否为密文。
- **随机 nonce**：每次加密使用独立随机 nonce；GCM 自带完整性认证，篡改密文将解密失败。
- **未配置密钥**：`key_hex` 为空时不启用加密存储；**生产环境务必配置**，否则相关机密以明文形式存储。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `security.crypto.key_hex` | AES-256-GCM 密钥（仅环境变量） | `""` |

---

## 4. TOTP 双因素与恢复码

双因素认证基于 TOTP（`pquerna/otp`）。

- **Secret 存储**：TOTP secret 经 **AES-256-GCM** 加密（`enc:` 前缀）后入库。
- **恢复码**：一次性生成 10 个恢复码，每个以 **SHA-256 哈希**存储；使用后即失效。
- **防重放**：已成功的 TOTP 时间步记录于 `totp_used_codes`，防止同一动态码被重复使用。
- **组织策略**：组织可通过 `org_2fa_policies` 强制成员启用 2FA。

---

## 5. RBAC（基于角色的访问控制）

gitli 使用企业级 RBAC：**资源 × 动作** 的细粒度权限模型。

- **权限矩阵**：权限定义为 `资源（resource）× 动作（action）`，通过 `permissions`、`roles`、`role_permissions` 三张表解耦。
- **嵌套团队**：团队成员可包含子团队，形成树形继承；权限沿继承链向下传递。
- **自定义角色**：组织可定义角色并组合任意权限，授予团队或仓库协作者。
- **仓库级约束**：`team_repos`、`repo_collaborators` 决定团队/个人对具体仓库的可见与操作范围。
- **缓存**：权限判定结果进入 **LRU 缓存**，角色/成员变更时失效，兼顾安全与低内存。
- **审计**：权限相关变更写入 `permission_audit` 审计表。
- **执行点**：HTTP 中间件 `RequirePermission` 在进入业务逻辑前完成校验；Git 服务端钩子同样校验分支/标签保护规则。

---

## 6. 会话安全

Session 使用 `gorilla/sessions` 管理，支持绑定客户端特征以降低会话劫持风险。

- **IP 绑定**：`bind_ip = true` 时，会话与客户端 IP 绑定，IP 变化要求重新登录。
- **UA 绑定**：`bind_agent = true` 时，会话与 User-Agent 绑定。
- **有效期**：`ttl_hours` 控制会话存活时长。
- **防护**：配合 CSRF 双提交 Cookie、HttpOnly、SameSite 等 Cookie 属性。

> IP 绑定在移动网络、多出口 NAT 场景下可能造成频繁掉线，请按部署环境权衡。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `security.session.bind_ip` | 会话绑定 IP | `false` |
| `security.session.bind_agent` | 会话绑定 User-Agent | `false` |
| `security.session.ttl_hours` | 会话有效期（小时） | `720` |

---

## 7. Web 安全头与 CSRF/CSP

HTTP 传输层统一施加安全基线：

- **CSRF**：双提交 Cookie（double-submit cookie）模式校验状态变更请求。
- **CSP**：Content-Security-Policy 限制脚本/样式/资源来源，降低 XSS 影响面。
- **强制 HTTPS / HSTS**：建议部署在反向代理后由代理终止 TLS；应用层启用 HSTS，要求 TLS 1.2+。
- **其它安全头**：`X-Content-Type-Options: nosniff`、`X-Frame-Options` / `frame-ancestors`、`Referrer-Policy` 等。
- **Markdown**：渲染启用安全模式，过滤危险 HTML 与脚本。
- **SQL**：全部参数化，禁止字符串拼接值；占位符统一由 `db.Rebind` 按方言改写。

> 严禁通过配置关闭 CSRF/CSP 等安全中间件。

---

## 8. Git 子进程安全

Git 操作统一调用系统 `git` 子进程（`exec.CommandContext`），不引入 libgit2，并施加多重约束：

- **禁止 shell 拼接**：分开传参，**严禁** `exec.Command("sh", "-c", ...)`。
- **环境变量白名单**：仅传递 `PATH`、`HOME`、`LANG=C.UTF-8`、`GIT_TERMINAL_PROMPT=0`；禁止传递 `GIT_SSH_COMMAND`、`GIT_EXTERNAL_DIFF` 等危险变量。
- **超时**：单次调用默认 `300s`，超时即终止。
- **路径校验**：所有仓库路径经 `ValidateRepoPath` 校验，限制在 `data/repos` 前缀内，防止路径穿越。
- **并发限制**：通过信号量限流，避免资源耗尽。
- **二进制路径**：生产环境推荐显式设置 `git.bin`（如 `/usr/bin/git`），避免 `PATH` 被篡改；启动时可校验版本。
- **服务端钩子**：`pre-receive` 钩子执行分支/标签保护与签名校验。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `git.bin` | git 可执行文件路径（空则查 `PATH`） | `""` |
| `git.timeout_seconds` | 单次调用超时（秒） | `300` |
| `git.check_version` | 启动时校验版本 | `true` |
| `git.min_version` | 最低版本 | `2.30` |
| `limits.max_git_concurrency` | Git 操作最大并发 | `4` |

---

## 9. Webhook 安全（HMAC + SSRF 防护）

出站 Webhook 投递同时防篡改与防内网探测：

- **HMAC-SHA256 签名**：每个请求体以共享密钥计算 HMAC-SHA256，置于签名头，接收方据此验证来源与完整性。
- **SSRF 防护**：默认禁止投递到内网、回环、链路本地与云元数据地址（如 `169.254.169.254`）。
- **重试与死信**：失败按退避重试，超过上限进入死信，投递记录留痕。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `webhook.timeout_seconds` | 单次投递超时（秒） | `15` |
| `webhook.max_retries` | 最大重试次数 | `3` |
| `webhook.retry_interval_seconds` | 重试间隔（秒） | `60` |
| `webhook.block_private_addresses` | 禁止内网/元数据地址（SSRF 防护） | `true` |

---

## 10. 登录限流

防止暴力破解与会话枚举：

- **维度**：同时按 **IP + 用户名** 计数。
- **策略**：窗口期内失败达 `max_attempts` 后锁定 `lockout_minutes`。
- **行为**：锁定期内继续尝试均被拒绝，并记录审计日志。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `security.login.max_attempts` | 窗口内最大失败次数 | `5` |
| `security.login.lockout_minutes` | 锁定时长（分钟） | `15` |
| `security.login.window_minutes` | 统计窗口（分钟） | `15` |

---

## 11. 拒绝以 root 运行

- **检测**：Linux/Unix 下检测到 `os.Getuid() == 0` 时，默认**拒绝启动并退出（退出码 1）**。
- **建议**：创建专用无登录权限用户（如 `gitli`）运行服务。
- **强制选项**：确需以 root 运行时，可在 `[server]` 设置 `allow_root = true`（**强烈不建议**）。Windows 跳过该检查。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `server.allow_root` | 允许以 root 运行 | `false` |

---

## 12. 备份加密与 HMAC

`gitli backup` 支持加密备份，恢复时校验完整性：

- **加密**：启用后输出 `.tar.gz.enc`，采用 **AES-256-GCM** 分块加密（固定大小帧 + 随机 nonce），全程**流式**处理，不整包读入内存。
- **HMAC 校验**：在全部密文帧之后追加 **HMAC-SHA256 尾迹**；恢复时流式重算并比对，任何篡改都会导致校验失败而拒绝恢复。
- **密钥来源**：密钥从指定环境变量读取（默认 `GITLI_BACKUP_KEY`），仅内存持有，不落盘。
- **Magic 头**：加密归档带固定 Magic，便于快速识别与校验格式。

| 配置键 | 说明 | 默认 |
| --- | --- | --- |
| `backup.encrypt` | 备份是否加密 | `false` |
| `backup.encryption_key_env` | 密钥环境变量名 | `GITLI_BACKUP_KEY` |
| `backup.backup_dir` | 备份输出目录 | `./data/backups` |

---

## 13. 其它约定

- **用户名可改**：旧名写入 `user_redirects`，HTTP 返回 301，SSH 支持别名，避免链接失效与钓鱼混淆。
- **跨平台**：禁止硬编码路径分隔符与平台 syscall；平台相关代码用 build tag 隔离，集中于 `internal/platform`。
- **可关闭模块**：所有可选模块可通过配置完全关闭；关闭时不初始化、不分配资源、不启动 goroutine。
- **敏感配置**：pepper、crypto key、mail/SMTP 密码、OIDC client secret 等仅从环境变量注入，绝不提交仓库。

---

## 14. 报告漏洞

如发现安全问题，请通过仓库 Issue 或维护者邮箱**私下**报告，避免公开披露细节；我们会在修复后致谢。请不要在公开 Issue 中粘贴可利用的 PoC。
