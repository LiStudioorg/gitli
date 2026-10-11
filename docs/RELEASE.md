# 发布流程（RELEASE）

本文档描述 gitli 的正式发布步骤、产物校验与回滚办法。发布由 `.github/workflows/cd.yml`（Release）自动完成，本文是配套的操作手册。

---

## 0. 前置约定

- 版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)：`vMAJOR.MINOR.PATCH`。
- 发布 tag 必须以 `v` 开头（如 `v1.0.0`），否则不触发 Release 工作流。
- 分支上的代码必须先经 CI 全绿后，再打 tag。
- 发布**仅由 tag 触发**，请勿手动改产物或绕过工作流。

---

## 1. 发布前检查

在 `main` 上进行，全部通过才可发布：

```bash
# 1. 确认工作区干净、已同步远端
git status
git pull --ff-only

# 2. 验收命令
gofmt -l .                     # 应无输出
go vet ./...                   # 应无错误
go test ./... -race            # 全部通过
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gitli .
./gitli version                # 打印版本

# 3. 前端构建
cd web && npm ci && npm run build && cd ..
```

同时确认远端 **CI 工作流（`ci.yml`）对当前 `main` 的运行是绿色**。

> 提示：可先在功能分支/PR 上让 CI 跑一遍，确认无误再合并到 `main`。

---

## 2. 打 tag 并推送

```bash
# 确保位于要发布的提交
git checkout main
git pull --ff-only

# 创建附注 tag（推荐，可写发布说明）
git tag -a v1.0.0 -m "gitli v1.0.0"

# 推送 tag（这会触发 Release 工作流）
git push origin v1.0.0
```

> 推送 tag 即触发 `cd.yml`；请勿在打 tag 后再修改同一提交的源码。

---

## 3. 观察 Actions 运行

进入仓库 **Actions → Release** 页面，依次观察三个阶段：

1. **Build frontend** — 前端产物 `web-dist` 生成。
2. **Build shard 1..5** — 5 个分片并行编译 46 个平台目标（`fail-fast: false`，单片失败不影响其它片）。
3. **Package and release** — 合并分片产物，运行 `goreleaser release --clean --skip=build`，打系统包、推镜像、创建 Release。

全部为绿色后，进入下一步校验。若某个分片失败，只需修复代码并**重新打一个新的 patch tag**（如 `v1.0.1`）重跑——不要删除已推送的 tag 复用同一版本号。

可选：手动触发（`workflow_dispatch`）时需填写 `tag` 输入（如 `v1.0.0`）。

---

## 4. 校验发布产物

在仓库 **Releases** 页面打开对应版本，逐项确认：

- [ ] **归档包**：各平台 `gitli_<Version>_<Os>_<Arch>[vN].tar.gz`（Windows 为 `.zip`）。
- [ ] **校验和**：`SHA256SUMS` 存在，且与下载包一致。
- [ ] **SBOM**：随归档生成的软件物料清单。
- [ ] **系统包**：`deb` / `rpm` / `apk` 均存在。
- [ ] **Release 说明**：changelog 已按提交类型生成（自动过滤 `docs:`/`test:`/`chore:`/`ci:`）。
- [ ] **预发布标记**：含预发布标识的 tag 自动为 prerelease。

校验校验和示例：

```bash
sha256sum -c SHA256SUMS
# 或手动比对某文件
sha256sum gitli_1.0.0_linux_amd64.tar.gz
```

校验容器镜像：

```bash
docker pull ghcr.io/LiStudioorg/gitli:1.0.0
docker manifest inspect ghcr.io/LiStudioorg/gitli:1.0.0   # 应含 amd64/arm64/armv7
```

---

## 5. 回滚

### 5.1 发布有误（二进制/说明问题）

- **不要**删除或强制移动已发布的 tag。
- 修复问题后，发布一个更高的 **patch** 版本（如 `v1.0.1`）覆盖修复。
- 可在 GitHub Release 页面把有问题的版本标记为 **prerelease** 或编辑说明引导用户升级。

### 5.2 容器镜像回滚

- `latest` 始终指向最近一次成功发布；如需回到旧版，直接使用**固定版本 tag**：

  ```bash
  docker pull ghcr.io/LiStudioorg/gitli:0.9.0
  ```

- 生产环境建议**锁定精确版本 tag**，而非依赖 `latest`。

### 5.3 服务回滚

1. 停止服务（如 `systemctl stop gitli`）。
2. 换回上一版本二进制（保留旧版本产物以便快速切换）。
3. **数据库迁移不可自动回退**：若新版本包含迁移，回滚前请评估兼容性，必要时从发布前备份恢复（见 `docs/SECURITY.md` 备份加密与 `gitli restore`）。

---

## 6. 发布清单（Checklist）

- [ ] `main` 上 CI（`ci.yml`）绿色。
- [ ] 本地验收命令全部通过。
- [ ] 附注 tag `vX.Y.Z` 已创建并推送。
- [ ] Release 工作流三阶段全绿。
- [ ] Release 资产、`SHA256SUMS`、系统包、镜像完整。
- [ ] 在发行说明中标注升级注意事项（尤其含数据库迁移时）。
- [ ] 发布后再次确认下载包校验和匹配。
