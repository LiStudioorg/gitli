// SPDX-License-Identifier: AGPL-3.0
package webhook

import "testing"

func TestValidateURLBlocksInternalTargets(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/hook",
		"http://localhost/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://10.1.2.3/hook",
		"http://192.168.0.10/hook",
		"http://172.20.0.1/hook",
		"http://[::1]/hook",
		"file:///etc/passwd",
		"javascript:alert(1)",
	}
	for _, u := range blocked {
		if err := validateURL(u); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}
	if err := validateURL("https://example.com/hook"); err != nil {
		t.Errorf("public URL rejected: %v", err)
	}
}
