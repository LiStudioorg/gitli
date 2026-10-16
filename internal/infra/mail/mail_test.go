// SPDX-License-Identifier: AGPL-3.0
package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSendDisabled(t *testing.T) {
	s := New(Config{Enabled: false})
	if s.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
	err := s.Send(context.Background(), "a@example.com", "hi", "body")
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("Send() = %v, want ErrDisabled", err)
	}
}

func TestBuildMessageHeaders(t *testing.T) {
	msg, err := buildMessage("from@example.com", "to@example.com", "Hello", "line1\nline2")
	if err != nil {
		t.Fatalf("buildMessage error: %v", err)
	}
	got := string(msg)

	for _, want := range []string{
		"From: from@example.com\r\n",
		"To: to@example.com\r\n",
		"Subject: Hello\r\n",
		"Date: ",
		"MIME-Version: 1.0\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message missing %q\ngot:\n%s", want, got)
		}
	}

	if !strings.Contains(got, "\r\n\r\n") {
		t.Errorf("message missing header/body separator")
	}
	if strings.Contains(strings.ReplaceAll(got, "\r\n", ""), "\n") {
		t.Errorf("message contains bare LF")
	}
	if !strings.HasSuffix(got, "\r\n") {
		t.Errorf("message does not end with CRLF")
	}
	if !strings.Contains(got, "line1\r\nline2\r\n") {
		t.Errorf("body not CRLF-normalized, got:\n%s", got)
	}
}

func TestBuildMessageHeaderInjection(t *testing.T) {
	if _, err := buildMessage("from@example.com", "to@example.com", "subj\r\nBcc: evil@example.com", "body"); err == nil {
		t.Fatal("subject injection not rejected")
	}
	if _, err := buildMessage("from@example.com", "to@example.com\r\nBcc: evil@example.com", "subj", "body"); err == nil {
		t.Fatal("recipient injection not rejected")
	}
	if _, err := buildMessage("from@example.com", "", "subj", "body"); err == nil {
		t.Fatal("empty recipient not rejected")
	}
}
