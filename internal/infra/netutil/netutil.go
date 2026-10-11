// SPDX-License-Identifier: AGPL-3.0
// Package netutil provides address helpers shared across transports.
package netutil

import (
	"net"
	"strings"
)

// NetworkFor returns the tcp network selector ("tcp4", "tcp6" or "tcp")
// appropriate for a listen address. Using a specific family avoids the
// platform binding an IPv4 wildcard as a dual-stack IPv6 socket.
func NetworkFor(addr string) string {
	host := hostOf(addr)
	switch {
	case host == "":
		return "tcp"
	case strings.Contains(host, ":"):
		return "tcp6"
	default:
		return "tcp4"
	}
}

func hostOf(addr string) string {
	if strings.HasPrefix(addr, "[") {
		if i := strings.Index(addr, "]"); i >= 0 {
			return addr[1:i]
		}
	}
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		return addr[:i]
	}
	return addr
}

// Family classifies a listener address as IPv4 or IPv6.
func Family(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "unknown"
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return "unknown"
	}
	if ip.To4() != nil {
		return "IPv4"
	}
	return "IPv6"
}
