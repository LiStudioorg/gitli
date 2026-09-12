// Package assets 内嵌模板与静态资源，供 internal/web 渲染层引用。
package assets

import (
	"embed"
	"io/fs"
)

//go:embed templates static
var files embed.FS

// Templates 模板文件系统子树（templates/...）。
func Templates() embed.FS { return files }

// StaticFS 静态资源子树（static/...）。
func StaticFS() fs.FS { return files }
