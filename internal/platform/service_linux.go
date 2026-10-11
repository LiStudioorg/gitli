// SPDX-License-Identifier: AGPL-3.0
//go:build linux

package platform

// ServiceTemplate returns the systemd unit template for Linux.
func ServiceTemplate() string {
	return `[Unit]
Description=gitli Git service
After=network.target

[Service]
Type=simple
User=gitli
Group=gitli
WorkingDirectory=/var/lib/gitli
ExecStart=/usr/local/bin/gitli serve --config /etc/gitli/config.toml
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
`
}
