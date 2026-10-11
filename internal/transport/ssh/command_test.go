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
		// quoted paths with a leading slash are normalized.
		{"git-upload-pack \"/owner/repo.git\"", "git-upload-pack owner/repo.git", true},
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

func TestParseCommandRejectsDangerous(t *testing.T) {
	rejected := []string{
		"shell",
		"scp -t /",
		"git-upload-pack; rm -rf /",
		"git-upload-pack /abs/../etc",
		"git-upload-pack owner/repo\x00--upload-pack=sh",
		"\x00",
		"git-upload-pack a && rm -rf /",
		"git-upload-pack a | cat /etc/passwd",
		"git-upload-pack a > /tmp/x",
		"git-upload-pack `id`",
		"git-upload-pack $(id)",
		"subsystem sftp",
		"pty-req",
	}
	for _, in := range rejected {
		if got, ok := ParseCommand(in); ok {
			t.Fatalf("ParseCommand(%q) accepted as %q; want rejection", in, got)
		}
	}
}

func TestParseCommandAcceptsValid(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"git-upload-pack 'team/project'", "git-upload-pack team/project"},
		{"git-receive-pack \"/team/project.git\"", "git-receive-pack team/project.git"},
		{"git-upload-pack team/project-2.git", "git-upload-pack team/project-2.git"},
	}
	for _, c := range cases {
		got, ok := ParseCommand(c.in)
		if !ok || got != c.want {
			t.Fatalf("ParseCommand(%q)=(%q,%v) want (%q,true)", c.in, got, ok, c.want)
		}
	}
}

func TestIsAllowedVerb(t *testing.T) {
	allowed := []string{"git-upload-pack", "git-receive-pack", "git-upload-archive"}
	for _, v := range allowed {
		if !IsAllowedVerb(v) {
			t.Fatalf("IsAllowedVerb(%q)=false want true", v)
		}
	}
	denied := []string{"", "shell", "scp", "git", "git-upload-pack ", "GIT-UPLOAD-PACK", "rm"}
	for _, v := range denied {
		if IsAllowedVerb(v) {
			t.Fatalf("IsAllowedVerb(%q)=true want false", v)
		}
	}
}
