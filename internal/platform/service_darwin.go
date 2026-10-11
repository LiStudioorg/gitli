// SPDX-License-Identifier: AGPL-3.0
//go:build darwin

package platform

// ServiceTemplate returns the launchd plist template for macOS.
func ServiceTemplate() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gitli</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/gitli</string>
        <string>serve</string>
        <string>--config</string>
        <string>/usr/local/etc/gitli/config.toml</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/usr/local/var/gitli</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
`
}
