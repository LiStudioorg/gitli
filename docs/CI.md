# CI 与发布流水线（CI/CD）

gitli 使用两个**完全独立、触发条件互斥**的 GitHub Actions 工作流：

| 文件 | 名称 | 触发 | 职责 |
| --- | --- | --- | --- |
| `.github/workflows/ci.yml` | CI | 分支 push（`main`）/ PR / 手动，显式排除 tag | 只做测试与验证 |
| `.github/workflows/cd.yml` | Release | 仅 `v*` tag / 手动 | 只做全量构建、打包、发布 |

两者**不共享 job、不互相依赖、使用不同的 `concurrency` group**：CI 为 `ci-${{ github.ref }}`，Release 为 `release-${{ github.ref }}`，互不阻塞。CI 只读（`permissions: contents: read`），Release 需要写权限（`contents: write`、`packages: write`）。

---

## 1. `ci.yml` — 测试与验证

**触发条件**

```yaml
on:
  workflow_dispatch:
  push:
    branches: [main]
    tags-ignore: ['**']      # 显式排除 tag，与 Release 互斥
  pull_request:
    branches: [main]
```

`tags-ignore: ['**']` 保证打 tag 时只跑 `cd.yml`，不会重复触发 CI。

**执行步骤（单 job `test`，ubuntu-latest）**

1. Checkout（`fetch-depth: 1`）。
2. Setup Go `1.22.x`，开启 module 缓存。
3. 缓存 `~/.cache/go-build`（key 含 `**/*.go` 与 `go.sum` 哈希）。
4. `gofmt -l .` 检查（有输出即失败）。
5. `go vet ./...`。
6. `go test ./... -race -coverprofile=coverage.out`。
7. 上传覆盖率 artifact（保留 7 天）。
8. Setup Node `20`（缓存 `web/package-lock.json`）。
9. 前端构建：`npm ci && npm run build`。
10. 静态编译验证（`CGO_ENABLED=0 -trimpath -ldflags="-s -w"`），验证 embed 正常。
11. 交叉编译检查（`linux/arm64`）。
12. 静态性校验（`file` / `ldd` 判定静态链接）。
13. `golangci-lint`（存在 `.golangci.yml` 时执行）。

CI **不发布、不推镜像、不上传构建产物**（仅覆盖率 artifact）。提交代码或开 PR 后约 2 分钟内得出结果。

---

## 2. `cd.yml` — 全量构建与发布

**触发条件**

```yaml
on:
  push:
    tags: ['v*']
  workflow_dispatch:
    inputs:
      tag: { description: 'Tag to release (e.g. v1.0.0)', required: true }
```

Release **不跑单元测试**（测试由 CI 负责）。

### 2.1 阶段一：`frontend`

构建前端产物 `web/dist`，作为 artifact `web-dist`（保留 1 天）供后续分片下载。

### 2.2 阶段二：`build`（5 分片矩阵）

`needs: frontend`，`fail-fast: false`。把 **46 个平台/架构目标**拆到 **5 个 matrix 分片**并行编译，2 核 runner 也能高效利用：

| 分片 | 目标 |
| --- | --- |
| 1 | `linux/amd64`、`linux/arm64`、`linux/armv6`、`linux/armv7` |
| 2 | `linux/386`、`riscv64`、`loong64`、`ppc64le`、`s390x`、`mips`、`mipsle`、`mips64`、`mips64le`、`ppc64` |
| 3 | `darwin/amd64`、`darwin/arm64`、`windows/amd64`、`windows/arm64`、`windows/386` |
| 4 | `freebsd`（386/amd64/arm/arm64）、`openbsd`（386/amd64/arm/arm64/ppc64/riscv64）、`netbsd`（386/amd64/arm/arm64） |
| 5 | `android`（386/amd64/arm/arm64）、`aix/ppc64`、`dragonfly/amd64`、`illumos/amd64`、`solaris/amd64`、`plan9`（386/amd64/arm）、`js/wasm`、`wasip1/wasm` |

每个分片：

1. 下载 `web-dist` 到 `web/dist`。
2. 设置 `VERSION=${{ github.ref_name }}`、`COMMIT=${{ github.sha }}`、`DATE`，调用 `scripts/build-targets.sh` 构建该分片目标，`armvN` 自动映射为 `GOARCH=arm GOARM=N`，Windows 产物加 `.exe`。
3. 每个二进制生成 `.sha256` 校验文件。
4. 上传 artifact `binaries-<shard>`。

### 2.3 阶段三：`publish`

`needs: build`。合并所有分片 artifact 后打包发布：

1. `fetch-depth: 0`（GoReleaser 需要完整历史生成 changelog）。
2. 下载并合并 `binaries-*` 到 `dist/`。
3. 把二进制重命名为 `<name>_<version>` 形式放入 `release/`。
4. `docker/setup-qemu-action` + `buildx`，登录 `ghcr.io`。
5. 运行 `goreleaser release --clean --skip=build`。

**关键：`--skip=build`。** 二进制已由分片构建，GoReleaser 只做打包与发布，避免重复编译。`.goreleaser.yml` 中 `builds[].skip: true` 印证此模式，`ldflags` 仍保留以便本地 `goreleaser build --snapshot` 校验。

GoReleaser 产出：

- **归档**：默认 `tar.gz`；`windows` 覆盖为 `zip`；命名 `gitli_<Version>_<Os>_<Arch>[vN]`，内含 `LICENSE`、`README.md`、`config.example.toml`。
- **校验和**：`SHA256SUMS`（sha256）。
- **SBOM**：基于 archive 生成。
- **系统包（nfpms）**：`deb`、`rpm`、`apk`，安装到 `/usr/local/bin`，附带 `/etc/gitli/config.example.toml` 与 systemd 单元。
- **多架构镜像**：推送到 `ghcr.io/LiStudioorg/gitli`，构建 `amd64`、`arm64`、`armv7` 分别打 tag，再合并为多架构 manifest `<version>` 与 `latest`。
- **GitHub Release**：`draft: false`，`prerelease: auto`（含预发布标识的 tag 自动标记为 prerelease）。

**changelog**：基于 GitHub，按提交类型生成，自动过滤 `docs:`、`test:`、`chore:`、`ci:` 前缀。

---

## 3. 缓存策略

两个工作流统一使用以下缓存，缩短构建时间：

| 缓存 | 路径 | 用途 |
| --- | --- | --- |
| Go module | `~/go/pkg/mod`（`setup-go` 内建 `cache: true`） | 依赖下载 |
| Go build | `~/.cache/go-build` | 增量编译 |
| 前端 | `web/node_modules`（`setup-node`，`cache: npm`） | 前端依赖 |
| Docker layer | `type=gha`（buildx） | 镜像层复用 |

缓存 key 以 `**/*.go`、`go.sum`（或 `web/package-lock.json`）内容哈希计算，配合 `restore-keys` 兜底。

---

## 4. 为什么这样拆分

- **触发分层**：分支/PR 只做验证，tag 才全量构建——单次 Release 目标控制在 8 分钟以内。
- **分片并行**：46 个目标均摊到 5 个 runner，避免单机串行编译。
- **避免重复编译**：分片出二进制，最终 job 只打包（`--skip=build`）。
- **职责隔离**：CI 与 Release 互斥触发、不同并发组，任一失败不影响另一条流水线。

---

## 5. 本地复现

```bash
# CI 等价检查
gofmt -l .
go vet ./...
go test ./... -race
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o gitli .
cd web && npm ci && npm run build && cd ..

# 单目标构建（等价于分片脚本的一部分）
VERSION=v0.1.0 COMMIT=$(git rev-parse --short HEAD) DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  bash scripts/build-targets.sh "linux/amd64,linux/arm64" dist

# 打包（需安装 goreleaser，且 dist/ 已有匹配二进制）
goreleaser release --clean --skip=build
```

> 本地沙箱无 `goreleaser` / `docker` 时无法完整复现 Release 的打包与镜像推送；完整链路依赖 GitHub Actions 云端环境。
