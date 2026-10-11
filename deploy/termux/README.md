# Android (Termux) 安装说明

gitli 使用纯 Go SQLite 驱动（`modernc.org/sqlite`），无需 CGO，可直接在 Termux 中编译运行。

## 安装

```bash
pkg update && pkg upgrade
pkg install golang git nodejs-lts openssh
```

## 构建

```bash
git clone https://github.com/LiStudioorg/gitli.git
cd gitli
# 构建前端（可选，发布版已内嵌 dist）
cd web && npm install && npm run build && cd ..
# 构建 ARM64 二进制
go build -o gitli .
```

## 配置与运行

```bash
cp config.example.toml config.toml
# 将 SSH 端口改为非特权端口，例如 :2222
# Android 无 root 时无法绑定 1024 以下端口
./gitli migrate
./gitli serve
```

## 说明

- Android 无 POSIX root 概念；gitli 在非 root 环境正常启动。
- 系统可能没有 `sshd`，可将 `[ssh] enabled = false` 关闭 SSH，仅使用 HTTP Git。
- 数据目录默认在 `./data`，确保 Termux 存储权限可用。
- 内存受限，建议调低 `[limits]` 相关并发值。
