// SPDX-License-Identifier: AGPL-3.0
package secutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// NewSafeHTTPClient returns an HTTP client that refuses to connect to private,
// loopback, link-local or unspecified addresses at dial time. Validating only
// the URL lets an attacker pass the initial DNS check and then rebind the name
// to 127.0.0.1 before the actual connection (DNS rebinding); checking the
// resolved IP inside DialContext closes that window.
func NewSafeHTTPClient(timeout time.Duration, maxRedirects int) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	transport := &http.Transport{
		Proxy: nil, // never follow environment proxies to internal networks
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ip := range ips {
				if isBlockedIP(ip.IP) {
					lastErr = fmt.Errorf("secutil: refusing to connect to private address %s", ip.IP)
					continue
				}
				conn, derr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
				if derr == nil {
					return conn, nil
				}
				lastErr = derr
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("secutil: no usable address for %s", host)
			}
			return nil, lastErr
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	redirects := 0
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects++
			if maxRedirects > 0 && redirects > maxRedirects {
				return fmt.Errorf("secutil: too many redirects")
			}
			// Re-validate the redirect target scheme and host.
			if err := ValidateOutboundURL(req.URL.String(), true); err != nil {
				return err
			}
			return nil
		},
	}
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() || ip.IsUnspecified() || ip.IsMulticast()
}
