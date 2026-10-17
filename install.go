// SPDX-License-Identifier: AGPL-3.0
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const systemdUnitTemplate = `[Unit]
Description=gitli - lightweight self-hosted Git service
Documentation=https://github.com/LiStudioorg/gitli
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%[1]s
Group=%[1]s
WorkingDirectory=%[2]s
ExecStart=%[3]s serve --config %[4]s
Restart=on-failure
RestartSec=3
LimitNOFILE=65536
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=false

[Install]
WantedBy=multi-user.target
`

// runInstall registers gitli as a system service. It only generates and enables
// a systemd unit; it never overwrites an existing installation silently.
func runInstall(args []string) error {
	unitPath := "/etc/systemd/system/gitli.service"
	user := "gitli"
	configPath := "/etc/gitli/config.toml"
	workDir := "/var/lib/gitli"
	enable := true
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--user":
			if i+1 < len(args) {
				user = args[i+1]
				i++
			}
		case "--config":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		case "--dir":
			if i+1 < len(args) {
				workDir = args[i+1]
				i++
			}
		case "--no-enable":
			enable = false
		case "--print":
			fmt.Print(renderUnit(user, workDir, execPath(), configPath))
			return nil
		}
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("安装为系统服务需要 root 权限，请使用 sudo 运行；或先用 `gitli install --print` 输出 unit 文件")
	}
	if _, err := os.Stat(unitPath); err == nil {
		return fmt.Errorf("%s 已存在，为避免覆盖已安装的服务，请先手动移除", unitPath)
	}
	if err := os.MkdirAll(workDir, 0o750); err != nil {
		return fmt.Errorf("创建数据目录: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		return fmt.Errorf("创建配置目录: %w", err)
	}
	if err := os.WriteFile(unitPath, []byte(renderUnit(user, workDir, execPath(), configPath)), 0o644); err != nil {
		return fmt.Errorf("写入 unit 文件: %w", err)
	}
	for _, cmd := range [][]string{{"systemctl", "daemon-reload"}} {
		if out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %w: %s", strings.Join(cmd, " "), err, strings.TrimSpace(string(out)))
		}
	}
	if enable {
		out, err := exec.Command("systemctl", "enable", "--now", "gitli.service").CombinedOutput()
		if err != nil {
			return fmt.Errorf("启用服务失败: %w: %s", err, strings.TrimSpace(string(out)))
		}
	}
	fmt.Printf("已安装并启动 gitli.service\n  配置文件：%s\n  数据目录：%s\n  查看状态：systemctl status gitli\n", configPath, workDir)
	return nil
}

func renderUnit(user, workDir, bin, configPath string) string {
	return fmt.Sprintf(systemdUnitTemplate, user, workDir, bin, configPath)
}

func execPath() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return "/usr/local/bin/gitli"
}
