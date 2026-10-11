// SPDX-License-Identifier: AGPL-3.0
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/LiStudioorg/gitli/internal/config"
	"github.com/LiStudioorg/gitli/internal/platform"
)

// errRefusedRoot signals the process must exit with code 1.
var errRefusedRoot = errors.New("refused to run as root")

// refuseRoot blocks startup when running as root unless explicitly allowed.
func refuseRoot(cfg *config.Config) error {
	if !platform.IsRoot() {
		return nil
	}
	if cfg.Server.AllowRoot {
		return nil
	}
	fmt.Fprintln(os.Stderr, "gitli 拒绝以 root 用户运行，请创建专用用户（如 gitli）并使用普通用户启动。如需强制运行，请设置 allow_root = true。")
	return errRefusedRoot
}
