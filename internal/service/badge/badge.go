// SPDX-License-Identifier: AGPL-3.0
// Package badge renders shields.io-style SVG status badges.
package badge

import (
	"fmt"
	"html"
)

const (
	charWidth    = 6.5
	padding      = 5.0
	labelXOffset = 10.0
	svgHeight    = 20
)

// ColorForStatus maps a build or check status to a flat badge color.
func ColorForStatus(status string) string {
	switch status {
	case "success", "passed":
		return "#10b981"
	case "failure", "failed":
		return "#ef4444"
	case "pending", "running":
		return "#f59e0b"
	default:
		return "#3b82f6"
	}
}

// SVG renders a flat SVG badge with the given label, message and color.
func SVG(label, message, color string) string {
	labelW := textWidth(label) + padding*2
	messageW := textWidth(message) + padding*2
	total := labelW + messageW
	labelMid := labelW / 2
	messageMid := total - messageW/2

	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%d" viewBox="0 0 %.0f %d" role="img" aria-label="%s: %s">
  <title>%s: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r">
    <rect width="%.0f" height="%d" rx="3" fill="#fff"/>
  </clipPath>
  <g clip-path="url(#r)">
    <rect width="%.0f" height="%d" fill="#555"/>
    <rect x="%.0f" width="%.0f" height="%d" fill="%s"/>
    <rect width="%.0f" height="%d" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="%.0f" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%.0f" y="14">%s</text>
    <text x="%.0f" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%.0f" y="14">%s</text>
  </g>
</svg>`,
		total, svgHeight, total, svgHeight,
		html.EscapeString(label), html.EscapeString(message),
		html.EscapeString(label), html.EscapeString(message),
		total, svgHeight,
		labelW, svgHeight,
		labelW, messageW, svgHeight, color,
		total, svgHeight,
		labelMid, html.EscapeString(label),
		labelMid, html.EscapeString(label),
		messageMid, html.EscapeString(message),
		messageMid, html.EscapeString(message),
	)
}

// ShieldJSON returns the SVG badge as a byte slice.
func ShieldJSON(label, message, color string) []byte {
	return []byte(SVG(label, message, color))
}

func textWidth(s string) float64 {
	return float64(len(s)) * charWidth
}
