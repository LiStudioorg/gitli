// SPDX-License-Identifier: AGPL-3.0
package ssh

import "testing"

func TestParseCommand(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"git-upload-pack 'owner/repo.git'", "git-upload-pack owner/repo.git", true},
		{"git-receive-pack '/owner/repo.git'", "git-receive-pack owner/repo.git", true},
		{"git-upload-archive owner/repo", "git-upload-archive owner/repo", true},
		{"rm -rf /", "", false},
		{"git-upload-pack", "", false},
		{"bash -c 'x'", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := ParseCommand(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("ParseCommand(%q)=(%q,%v) want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
