// SPDX-License-Identifier: AGPL-3.0
// Package inboundmail ingests replies to notification emails as comments.
package inboundmail

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"

	"github.com/LiStudioorg/gitli/internal/config"
)

// ErrInvalidToken indicates a forged or malformed reply token.
var ErrInvalidToken = errors.New("inboundmail: invalid reply token")

// Token signs a (kind, id) pair into an opaque reply address token.
type Token struct {
	key []byte
}

// NewToken builds a signer from a 32-byte key.
func NewToken(key []byte) *Token {
	return &Token{key: key}
}

// Sign returns a URL-safe token encoding kind:id with an HMAC signature.
func (t *Token) Sign(kind string, id int64) (string, error) {
	if t == nil || len(t.key) == 0 {
		return "", fmt.Errorf("inboundmail: missing signing key")
	}
	payload := kind + ":" + strconv.FormatInt(id, 10)
	sig := hmac.New(sha256.New, t.key)
	sig.Write([]byte(payload))
	raw := payload + "." + hex.EncodeToString(sig.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(raw)), nil
}

// Parse verifies and decodes a token into (kind, id).
func (t *Token) Parse(tok string) (string, int64, error) {
	if t == nil || len(t.key) == 0 {
		return "", 0, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		return "", 0, ErrInvalidToken
	}
	parts := strings.SplitN(string(raw), ".", 2)
	if len(parts) != 2 {
		return "", 0, ErrInvalidToken
	}
	payload, gotSig := parts[0], parts[1]
	sig := hmac.New(sha256.New, t.key)
	sig.Write([]byte(payload))
	if !hmac.Equal([]byte(hex.EncodeToString(sig.Sum(nil))), []byte(gotSig)) {
		return "", 0, ErrInvalidToken
	}
	kv := strings.SplitN(payload, ":", 2)
	if len(kv) != 2 {
		return "", 0, ErrInvalidToken
	}
	id, err := strconv.ParseInt(kv[1], 10, 64)
	if err != nil {
		return "", 0, ErrInvalidToken
	}
	return kv[0], id, nil
}

// ReplyAddress builds reply+<token>@domain.
func ReplyAddress(token, domain string) string {
	return "reply+" + token + "@" + domain
}

// ExtractToken pulls the token out of a reply+<token>@domain address.
func ExtractToken(addr string) (string, bool) {
	at := strings.LastIndexByte(addr, '@')
	if at < 0 {
		return "", false
	}
	local := addr[:at]
	if !strings.HasPrefix(local, "reply+") {
		return "", false
	}
	return strings.TrimPrefix(local, "reply+"), true
}

// Reply is a parsed inbound reply.
type Reply struct {
	From     string
	Subject  string
	Body     string
	Kind     string
	TargetID int64
	Token    string
}

// ParseMessage parses a raw RFC822 message and resolves the reply token.
func ParseMessage(raw io.Reader, tk *Token) (*Reply, error) {
	msg, err := mail.ReadMessage(raw)
	if err != nil {
		return nil, fmt.Errorf("inboundmail: read message: %w", err)
	}
	from := msg.Header.Get("From")
	if addr, err := mail.ParseAddress(from); err == nil {
		from = addr.Address
	}
	subject := msg.Header.Get("Subject")
	body, err := io.ReadAll(io.LimitReader(msg.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	// The token is embedded in the To address of the reply.
	to := msg.Header.Get("To")
	token := ""
	for _, addr := range strings.Split(to, ",") {
		addr = strings.TrimSpace(addr)
		if a, err := mail.ParseAddress(addr); err == nil {
			addr = a.Address
		}
		if t, ok := ExtractToken(addr); ok {
			token = t
			break
		}
	}
	if token == "" {
		return nil, ErrInvalidToken
	}
	kind, id, err := tk.Parse(token)
	if err != nil {
		return nil, err
	}
	return &Reply{From: from, Subject: subject, Body: string(body), Kind: kind, TargetID: id, Token: token}, nil
}

// Fetcher polls an IMAP mailbox for new messages.
type Fetcher struct {
	cfg config.InboundMailConfig
}

// NewFetcher builds an IMAP fetcher when enabled, else nil.
func NewFetcher(cfg config.InboundMailConfig) *Fetcher {
	if !cfg.Enabled || cfg.Host == "" {
		return nil
	}
	return &Fetcher{cfg: cfg}
}

// Fetch retrieves unread messages and invokes handle for each raw message.
func (f *Fetcher) Fetch(ctx context.Context, handle func(raw []byte) error) (int, error) {
	if f == nil {
		return 0, nil
	}
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", f.cfg.Host, f.cfg.Port), nil)
	if err != nil {
		return 0, fmt.Errorf("inboundmail: dial: %w", err)
	}
	defer func() { _ = c.Logout() }()
	if err := c.Login(f.cfg.Username, f.cfg.Password); err != nil {
		return 0, fmt.Errorf("inboundmail: login: %w", err)
	}
	mbox := f.cfg.Mailbox
	if mbox == "" {
		mbox = "INBOX"
	}
	if _, err := c.Select(mbox, false); err != nil {
		return 0, fmt.Errorf("inboundmail: select: %w", err)
	}
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}
	ids, err := c.Search(criteria)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)
	section := &imap.BodySectionName{}
	items := []imap.FetchItem{section.FetchItem()}
	msgs := make(chan *imap.Message, len(ids))
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, items, msgs)
	}()
	count := 0
	for msg := range msgs {
		if err := ctx.Err(); err != nil {
			return count, err
		}
		r := msg.GetBody(section)
		if r == nil {
			continue
		}
		raw, err := io.ReadAll(io.LimitReader(r, 4<<20))
		if err != nil {
			continue
		}
		if err := handle(raw); err == nil {
			count++
		}
	}
	if err := <-done; err != nil {
		return count, err
	}
	return count, nil
}
