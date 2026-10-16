// SPDX-License-Identifier: AGPL-3.0
package app

import "testing"

func TestWebhookWants(t *testing.T) {
	cases := []struct {
		events []string
		action string
		want   bool
	}{
		{nil, "issues", true},
		{[]string{}, "issues", true},
		{[]string{"*"}, "push", true},
		{[]string{"issues", "push"}, "issues", true},
		{[]string{"issues"}, "pull_request", false},
	}
	for _, c := range cases {
		if got := webhookWants(c.events, c.action); got != c.want {
			t.Fatalf("webhookWants(%v,%q)=%v want %v", c.events, c.action, got, c.want)
		}
	}
}

func TestSplitN(t *testing.T) {
	got := splitN("1|issues|{\"a\":1}", "|", 3)
	if len(got) != 3 || got[0] != "1" || got[1] != "issues" || got[2] != `{"a":1}` {
		t.Fatalf("splitN = %#v", got)
	}
}
