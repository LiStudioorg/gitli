package web

import (
	"html/template"

	"github.com/gomarkdown/markdown"
	"github.com/microcosm-cc/bluemonday"
)

// mdPolicy UGC 级消毒策略：全项目唯一允许 template.HTML 的豁免点（README/Markdown）。
var mdPolicy = bluemonday.UGCPolicy()

// RenderMarkdown 渲染 Markdown 并消毒，结果可安全作为 template.HTML 输出。
func RenderMarkdown(content []byte) template.HTML {
	rendered := markdown.ToHTML(content, nil, nil)
	return template.HTML(mdPolicy.SanitizeBytes(rendered))
}
