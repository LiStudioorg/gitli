# gitli 用户手册

本手册介绍如何使用 gitli：克隆仓库、提交代码、配置 SSH/GPG，以及使用工单、合并请求等功能。

---

## 1. 首次使用

1. 打开实例地址（如 `https://git.example.com`）。
2. 首次部署时会出现**初始化向导**，由管理员创建第一个账号。
3. 之后可在 `/register` 注册新账号，或用管理员在后台/CLI 创建。

密码规则：**至少 8 位**。

---

## 2. 克隆与推送

仓库页右上角 **代码** 按钮提供 HTTPS、SSH 与下载地址。

### 2.1 HTTPS

```bash
# 首次会提示输入用户名与密码/令牌
git clone https://git.example.com/<owner>/<repo>.git

# 推荐把令牌写进凭证助手，避免反复输入
git config --global credential.helper store
```

> 密码栏应填写**个人访问令牌（PAT）**，而不是登录密码（启用两步验证后尤其如此）。
> 令牌在「设置 → 访问令牌」创建，明文只显示一次。

快速推送时也可临时内联：

```bash
git push https://<user>:<token>@git.example.com/<owner>/<repo>.git main
```

### 2.2 SSH

```bash
ssh-keygen -t ed25519 -C "you@example.com"
cat ~/.ssh/id_ed25519.pub
```

把公钥粘贴到「设置 → SSH 密钥」，并勾选用途：

- **认证**（`can_authenticate`）：用于 `git clone/push` 的 SSH 认证。
- **签名**（`can_sign`）：用于提交签名验证。

至少勾选一项。仅勾选「签名」的密钥不能用于 Git 认证。

```bash
git clone ssh://git@git.example.com:2222/<owner>/<repo>.git
```

若 SSH 使用非 22 端口，可在 `~/.ssh/config` 中配置：

```sshconfig
Host git.example.com
    HostName git.example.com
    Port 2222
    User git
    IdentityFile ~/.ssh/id_ed25519
```

### 2.3 下载源码

```bash
curl -LO "https://git.example.com/<owner>/<repo>/archive/main.tar.gz"
curl -LO "https://git.example.com/<owner>/<repo>/archive/main.zip"
```

---

## 3. 提交签名（GPG / SSH）

### 3.1 GPG

```bash
gpg --full-generate-key
gpg --list-secret-keys --keyid-format=long
gpg --armor --export <KEY_ID>
```

把导出的公钥粘贴到「设置 → GPG 密钥」。然后在仓库中：

```bash
git config user.signingkey <KEY_ID>
git config commit.gpgsign true
git commit -S -m "feat: signed commit"
```

### 3.2 SSH 签名

```bash
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519.pub
git config commit.gpgsign true
```

用于签名的 SSH 密钥需在 gitli 中勾选「签名」用途。

> 分支保护可要求签名提交；未签名的推送会被拒绝并提示原因。

---

## 4. 令牌与权限

| 类型 | 适用 | 说明 |
| --- | --- | --- |
| 经典 PAT | 全局 | 拥有所选作用域（如 `repo`）的全局访问权 |
| 细粒度 PAT | 指定仓库 | 仅对指定仓库授予 `read`/`write` |

使用示例：

```bash
export GIT_TOKEN=gitli_xxxxxxxx
git clone https://<user>:$GIT_TOKEN@git.example.com/<owner>/<repo>.git
```

令牌可随时在「设置 → 访问令牌」吊销。

---

## 5. 工单与合并请求

### 5.1 工单

仓库页「工单」标签：

- 新建工单，支持 Markdown、模板、标签、里程碑。
- 可评论、关闭、重新打开。

### 5.2 合并请求

1. 在分支上提交并推送：

   ```bash
   git switch -c feat/my-change
   git commit -am "feat: my change"
   git push origin feat/my-change
   ```

2. 仓库页「合并请求」→ 新建，选择 base 与 head 分支。
3. 评审者可在行级评论、批准或请求修改。
4. 合并后源分支可按配置自动删除。

> 受保护分支可能要求评审通过、签名提交或指定 CODEOWNERS 批准。

---

## 6. 两步验证（TOTP）

「设置 → 安全」启用两步验证：

1. 用任意 TOTP 应用（如 Aegis、Google Authenticator）扫描二维码。
2. 输入 6 位验证码完成启用。
3. **妥善保存 10 个恢复码**——它们是丢失设备时唯一的登录方式。

启用后，`git push` 仍使用 PAT（不需要 TOTP 码）。

---

## 7. 常见问题

**Q：`git clone` 提示 401 / 认证失败？**
- HTTPS 的密码栏要填 **PAT**，不是登录密码。
- 确认令牌未过期、未被吊销、作用域足够。
- 启用两步验证后，必须使用 PAT。

**Q：`Permission denied (publickey)`？**
- 确认公钥已上传且勾选了「认证」用途。
- 用 `ssh -T -p 2222 git@git.example.com` 测试。
- 确认 `~/.ssh/config` 中的端口正确。

**Q：推送被拒绝，提示分支受保护？**
- 使用功能分支发起合并请求，而不是直接推送受保护分支。
- 若要求签名，请按第 3 节配置签名。

**Q：忘记密码？**
- 联系管理员重置：`gitli admin user change-password <username>`。

**Q：设备丢失且没有恢复码？**
- 联系管理员执行 `gitli admin user reset-2fa <username>`。

**Q：Android / Termux 上如何使用？**
- 使用 `pkg install git` 安装 Git，克隆/推送方式与桌面端一致；
  服务器端需正确配置 `[git] bin`（见管理员手册）。
