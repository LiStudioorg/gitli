// SPDX-License-Identifier: AGPL-3.0
package inboundmail

import (
	"strings"
	"testing"

	"github.com/LiStudioorg/gitli/internal/config"
)

func TestTokenRoundTrip(t *testing.T) {
	tk := NewToken([]byte("0123456789abcdef0123456789abcdef"))
	tok, err := tk.Sign("issue", 42)
	if err != nil {
		t.Fatal(err)
	}
	kind, id, err := tk.Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "issue" || id != 42 {
		t.Fatalf("parsed %s:%d", kind, id)
	}
	// Tampered token rejected.
	if _, _, err := tk.Parse(tok + "x"); err == nil {
		t.Fatal("tampered token should fail")
	}
	// Wrong key rejected.
	other := NewToken([]byte("ffffffffffffffffffffffffffffffff"))
	if _, _, err := other.Parse(tok); err == nil {
		t.Fatal("wrong key should fail")
	}
}

func TestExtractToken(t *testing.T) {
	if tok, ok := ExtractToken("reply+abc123@example.com"); !ok || tok != "abc123" {
		t.Fatalf("extract = %q %v", tok, ok)
	}
	if _, ok := ExtractToken("someone@example.com"); ok {
		t.Fatal("should not extract without reply+ prefix")
	}
}

func TestParseMessage(t *testing.T) {
	tk := NewToken([]byte("0123456789abcdef0123456789abcdef"))
	tok, _ := tk.Sign("pull", 7)
	raw := "From: alice@example.com\r\n" +
		"To: " + ReplyAddress(tok, "example.com") + "\r\n" +
		"Subject: Re: PR #7\r\n" +
		"\r\n" +
		"This is my reply.\r\n"
	reply, err := ParseMessage(strings.NewReader(raw), tk)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Kind != "pull" || reply.TargetID != 7 {
		t.Fatalf("reply = %+v", reply)
	}
	if !strings.Contains(reply.Body, "my reply") {
		t.Fatalf("body = %q", reply.Body)
	}
}

func TestParseMessageNoToken(t *testing.T) {
	tk := NewToken([]byte("0123456789abcdef0123456789abcdef"))
	raw := "From: a@b.com\r\nTo: other@example.com\r\nSubject: x\r\n\r\nbody"
	if _, err := ParseMessage(strings.NewReader(raw), tk); err != ErrInvalidToken {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestFetcherDisabled(t *testing.T) {
	if NewFetcher(config.InboundMailConfig{}) != nil {
		t.Fatal("disabled fetcher should be nil")
	}
}
