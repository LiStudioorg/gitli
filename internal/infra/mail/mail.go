// SPDX-License-Identifier: AGPL-3.0
package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

var ErrDisabled = errors.New("mail: disabled")

const (
	dialTimeout = 10 * time.Second
	sendTimeout = 60 * time.Second
)

type Config struct {
	Enabled  bool
	Host     string
	Port     int
	Username string
	Password string
	From     string
	TLS      bool
}

type Sender struct {
	cfg Config
}

func New(cfg Config) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) Enabled() bool {
	return s.cfg.Enabled
}

func (s *Sender) Send(ctx context.Context, to, subject, body string) error {
	if !s.cfg.Enabled {
		return ErrDisabled
	}
	msg, err := buildMessage(s.cfg.From, to, subject, body)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(s.cfg.Host, fmt.Sprintf("%d", s.cfg.Port))

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	d := &net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mail: dial: %w", err)
	}

	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mail: new client: %w", err)
	}
	defer func() { _ = c.Close() }()

	if s.cfg.TLS {
		if err := c.StartTLS(nil); err != nil {
			return fmt.Errorf("mail: starttls: %w", err)
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("mail: auth: %w", err)
			}
		}
	}
	if err := c.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("mail: mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mail: rcpt to: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mail: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mail: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mail: close: %w", err)
	}
	return c.Quit()
}

// buildMessage renders a minimal RFC5322 text/plain message with CRLF line
// endings. A \r or \n in to or subject is rejected to prevent header injection.
func buildMessage(from, to, subject, body string) ([]byte, error) {
	if strings.ContainsAny(to, "\r\n") {
		return nil, errors.New("mail: invalid recipient: header injection")
	}
	if strings.ContainsAny(subject, "\r\n") {
		return nil, errors.New("mail: invalid subject: header injection")
	}
	if strings.ContainsAny(from, "\r\n") {
		return nil, errors.New("mail: invalid from: header injection")
	}
	if to == "" {
		return nil, errors.New("mail: empty recipient")
	}

	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\n", "\r\n")

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\r\n") {
		b.WriteString("\r\n")
	}
	return b.Bytes(), nil
}
