package git

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"gitli/internal/db"
)

// GitProtocolServer git:// 9418 匿名协议 server：仅 public 仓库、只读（upload-pack）。
type GitProtocolServer struct {
	addr     string
	reposDir string
	q        db.Querier
}

func NewGitProtocolServer(addr, reposDir string, q db.Querier) *GitProtocolServer {
	return &GitProtocolServer{addr: addr, reposDir: reposDir, q: q}
}

func (s *GitProtocolServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("git protocol listen %s: %w", s.addr, err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return fmt.Errorf("git protocol accept: %w", err)
		}
		go s.handleConn(conn)
	}
}

func (s *GitProtocolServer) handleConn(conn net.Conn) {
	defer conn.Close()
	service, owner, name, err := s.readRequest(conn)
	if err != nil {
		slog.Debug("git protocol bad request", "err", err)
		return
	}
	if service != "git-upload-pack" {
		return
	}
	ctx := context.Background()
	repo, err := s.q.GetRepoByOwnerAndName(ctx, db.GetRepoByOwnerAndNameParams{Username: owner, Name: name})
	if err != nil {
		slog.Debug("git protocol repo not found", "owner", owner, "name", name)
		return
	}
	if repo.Visibility != "public" {
		slog.Debug("git protocol repo not public", "owner", owner, "name", name)
		return
	}

	path := RepoPath(s.reposDir, owner, name)
	cmd := exec.CommandContext(ctx, "git", "upload-pack", path)
	cmd.Stdin = conn
	cmd.Stdout = conn
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		slog.Debug("git upload-pack exited", "err", err)
	}
}

// readRequest 读取 pktline 请求行 "git-upload-pack /owner/name.git\0host=..."。
func (s *GitProtocolServer) readRequest(conn net.Conn) (service, owner, name string, err error) {
	payload, err := readPktLine(conn)
	if err != nil {
		return "", "", "", err
	}
	// host 与 path 以 NUL 分隔（v0 协议）
	head := payload
	if idx := strings.IndexByte(head, 0); idx >= 0 {
		head = head[:idx]
	}
	parts := strings.SplitN(head, " ", 2)
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("malformed request")
	}
	service = parts[0]
	repoPath := strings.TrimPrefix(parts[1], "/")
	if owner, name, err = splitRepoPath(repoPath); err != nil {
		return "", "", "", err
	}
	return service, owner, name, nil
}

// readPktLine 读一帧 pkt-line（4 字节 hex 长度 + payload，长度含自身 4 字节）。
func readPktLine(r io.Reader) (string, error) {
	hdr := make([]byte, 4)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return "", fmt.Errorf("read pktline header: %w", err)
	}
	n, err := strconv.ParseUint(string(hdr), 16, 16)
	if err != nil || n < 4 || n > 65516 {
		return "", fmt.Errorf("bad pktline length %q", string(hdr))
	}
	payload := make([]byte, n-4)
	if _, err := io.ReadFull(r, payload); err != nil {
		return "", fmt.Errorf("read pktline payload: %w", err)
	}
	return string(payload), nil
}

var _ = hex.EncodedLen
